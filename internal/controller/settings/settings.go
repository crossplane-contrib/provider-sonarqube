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
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Settings controller"))
		}
	}, v1alpha1.SettingsGroupVersionKind)
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SettingsGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewSettingsClient,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.SettingsList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.SettingsList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.SettingsGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Settings{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
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
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Settings)
	if !ok {
		return nil, errors.New(errNotSettings)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	m := mg.(resource.ModernManaged)

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
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSettings)
	}

	// Deleting: SonarQube settings cannot be deleted; mark the
	// external resource as non-existent so the managed reconciler can
	// remove the finalizer and allow the CR to be deleted.
	if !cr.DeletionTimestamp.IsZero() {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	settings, resp, err := c.settingsClient.Values(instance.GenerateSettingsValuesOptions(&cr.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get settings values")
	}
	observation := instance.GenerateSettingsObservation(settings)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.AreSettingsUpToDate(cr.Spec.ForProvider, observation),
	}, nil
}

// Create sets the SonarQube Settings based on the desired state in the managed resource. It should return an error if the creation failed, or nil if it succeeded.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSettings)
	}

	cr.SetConditions(xpv1.Creating())

	errs := make([]error, 0, len(cr.Spec.ForProvider.Settings))

	// Iterate over the settings in the CR and create them in SonarQube using the settingsClient.
	for key, params := range cr.Spec.ForProvider.Settings {
		settingSetOptions := instance.GenerateSettingSetOptions(params, cr.Spec.ForProvider.Component)
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
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSettings)
	}

	var updateErrors []error
	// Update out of date settings
	updateErrors = append(updateErrors, c.updateOutOfDateSettings(cr)...)
	// Reset obsolete settings
	updateErrors = append(updateErrors, c.resetObsoleteSettings(cr)...)

	if len(updateErrors) == 0 {
		return managed.ExternalUpdate{}, nil
	}
	if len(updateErrors) == 1 {
		return managed.ExternalUpdate{}, updateErrors[0]
	}
	return managed.ExternalUpdate{}, stderrors.Join(updateErrors...)
}

// updateOutOfDateSettings updates settings that are out of date by comparing the desired settings in the CR with the observed settings in SonarQube.
func (c *external) updateOutOfDateSettings(cr *v1alpha1.Settings) []error {
	var updateErrors []error
	for key, params := range cr.Spec.ForProvider.Settings {
		if !instance.IsSettingUpToDate(params, cr.Status.AtProvider.Settings[key]) {
			settingSetOptions := instance.GenerateSettingSetOptions(params, cr.Spec.ForProvider.Component)
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
func (c *external) resetObsoleteSettings(cr *v1alpha1.Settings) []error {
	var resetErrors []error
	toResetList := make([]string, 0)
	for key := range cr.Status.AtProvider.Settings {
		if _, exists := cr.Spec.ForProvider.Settings[key]; !exists {
			toResetList = append(toResetList, key)
		}
	}

	if len(toResetList) > 0 {
		settingsResetOptions := instance.GenerateSettingsResetOptionsFromList(toResetList, cr.Spec.ForProvider.Component)
		resp, err := c.settingsClient.Reset(settingsResetOptions) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)
		if err != nil {
			resetErrors = append(resetErrors, errors.Errorf("failed to reset settings that are not in the desired state: %s", err.Error()))
		}
	}
	return resetErrors
}

// Delete resets the settings in SonarQube based on the given managed resource. It returns any error that occurred during deletion.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Settings)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotSettings)
	}

	cr.SetConditions(xpv1.Deleting())

	// Reset the settings and then delete the resource. This ensures that we don't leave any orphaned settings in SonarQube after the resource is deleted.
	settingsResetOptions := instance.GenerateSettingsResetOptions(cr.Spec.ForProvider)
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
