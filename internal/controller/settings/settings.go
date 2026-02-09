/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package settings

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"

	stderrors "errors"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	errNotSettings  = "managed resource is not a Settings custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
)

// SetupGated adds a controller that reconciles Settings managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Settings controller"))
		}
	}, v1alpha1.SettingsGroupVersionKind)

	return nil
}

func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.SettingsGroupKind)

	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewSettingsClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		reconcilerOpts = append(reconcilerOpts, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		reconcilerOpts = append(reconcilerOpts, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		reconcilerOpts = append(reconcilerOpts, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.SettingsList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.SettingsList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.SettingsGroupVersionKind), reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Settings{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.SettingsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	settings, ok := managedResource.(*v1alpha1.Settings)
	if !ok {
		return nil, errors.New(errNotSettings)
	}

	err := c.usage.Track(ctx, settings)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	m, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, m)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{settingsClient: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// settingsClient is used to interact with SonarQube Settings API
	settingsClient instance.SettingsClient
}

// Observe checks if the external resource exists and if it matches the desired state specified by the managed resource. It returns an ExternalObservation indicating whether the resource exists, whether it is up to date, and any connection details or errors.
func (c *external) Observe(ctx context.Context, managedResource resource.Managed) (managed.ExternalObservation, error) {
	settings, ok := managedResource.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSettings)
	}

	// Deleting: SonarQube settings cannot be deleted; mark the
	// external resource as non-existent so the managed reconciler can
	// remove the finalizer and allow the CR to be deleted.
	if !settings.DeletionTimestamp.IsZero() {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	sonarSettings, resp, err := c.settingsClient.Values(instance.GenerateSettingsValuesOptions(&settings.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get settings values")
	}

	observation := instance.GenerateSettingsObservation(sonarSettings)
	settings.Status.AtProvider = observation

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.AreSettingsUpToDate(settings.Spec.ForProvider, observation),
	}, nil
}

// Create sets the SonarQube Settings based on the desired state in the managed resource. It should return an error if the creation failed, or nil if it succeeded.
func (c *external) Create(ctx context.Context, managedResource resource.Managed) (managed.ExternalCreation, error) {
	settings, ok := managedResource.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSettings)
	}

	settings.SetConditions(xpv1.Creating())

	errs := make([]error, 0, len(settings.Spec.ForProvider.Settings))

	// Iterate over the settings in the CR and create them in SonarQube using the settingsClient.
	for key, params := range settings.Spec.ForProvider.Settings {
		settingSetOptions := instance.GenerateSettingSetOptions(key, params, settings.Spec.ForProvider.Component)

		resp, err := c.settingsClient.Set(settingSetOptions) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		if err != nil {
			errs = append(errs, errors.Errorf("failed to set setting %s: %s", key, err.Error()))
		}
	}

	// Since there is no external name to return or connection details to provide, we can return an empty ExternalCreation and no error.
	if len(errs) == 0 {
		return managed.ExternalCreation{}, nil
	}

	if len(errs) == 1 {
		return managed.ExternalCreation{}, errs[0]
	}

	return managed.ExternalCreation{}, stderrors.Join(errs...)
}

// Update checks for any differences between the desired state in the managed resource and the observed state in SonarQube. If there are differences, it updates the SonarQube settings to match the desired state. It returns an error if the update failed, or nil if it succeeded.
func (c *external) Update(ctx context.Context, managedResource resource.Managed) (managed.ExternalUpdate, error) {
	settings, ok := managedResource.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSettings)
	}

	var updateErrors []error
	// Update out of date settings
	updateErrors = append(updateErrors, c.updateOutOfDateSettings(settings)...)
	// Reset obsolete settings
	updateErrors = append(updateErrors, c.resetObsoleteSettings(settings)...)

	if len(updateErrors) == 0 {
		return managed.ExternalUpdate{}, nil
	}

	if len(updateErrors) == 1 {
		return managed.ExternalUpdate{}, updateErrors[0]
	}

	return managed.ExternalUpdate{}, stderrors.Join(updateErrors...)
}

// Delete deletes the external resource.
// updateOutOfDateSettings updates settings that are out of date by comparing the desired settings in the CR with the observed settings in SonarQube.
func (c *external) Delete(ctx context.Context, managedResource resource.Managed) (managed.ExternalDelete, error) {
	settings, ok := managedResource.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotSettings)
	}

	settings.SetConditions(xpv1.Deleting())

	// Reset the settings and then delete the resource. This ensures that we don't leave any orphaned settings in SonarQube after the resource is deleted.
	settingsResetOptions := instance.GenerateSettingsResetOptions(settings.Spec.ForProvider)

	resp, err := c.settingsClient.Reset(settingsResetOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to reset settings during deletion")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
func (c *external) updateOutOfDateSettings(settings *v1alpha1.Settings) []error {
	var updateErrors []error

	for key, params := range settings.Spec.ForProvider.Settings {
		if !instance.IsSettingUpToDate(params, settings.Status.AtProvider.Settings[key]) {
			settingSetOptions := instance.GenerateSettingSetOptions(key, params, settings.Spec.ForProvider.Component)

			resp, err := c.settingsClient.Set(settingSetOptions) //nolint:bodyclose // closed via helpers.CloseBody
			defer helpers.CloseBody(resp)

			if err != nil {
				updateErrors = append(updateErrors, errors.Errorf("failed to update setting %s: %s", key, err.Error()))
			}
		}
	}

	return updateErrors
}

// resetObsoleteSettings resets any settings that are not in the desired settings in the CR. This ensures that any settings that were manually changed in SonarQube or removed from the CR are reset to their default values.
func (c *external) resetObsoleteSettings(settings *v1alpha1.Settings) []error {
	var resetErrors []error

	toResetList := make([]string, 0)

	for key := range settings.Status.AtProvider.Settings {
		if _, exists := settings.Spec.ForProvider.Settings[key]; !exists {
			toResetList = append(toResetList, key)
		}
	}

	if len(toResetList) > 0 {
		settingsResetOptions := instance.GenerateSettingsResetOptionsFromList(toResetList, settings.Spec.ForProvider.Component)

		resp, err := c.settingsClient.Reset(settingsResetOptions) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		if err != nil {
			resetErrors = append(resetErrors, errors.Errorf("failed to reset settings that are not in the desired state: %s", err.Error()))
		}
	}

	return resetErrors
}

// Delete resets the settings in SonarQube based on the given managed resource. It returns any error that occurred during deletion.
