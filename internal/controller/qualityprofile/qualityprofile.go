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

package qualityprofile

import (
	"context"
	"fmt"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/utils/ptr"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	errNotQualityProfile = "managed resource is not a QualityProfile custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"

	errCreateQualityProfile  = "cannot create SonarQube Quality Profile"
	errDefaultQualityProfile = "cannot set SonarQube Quality Profile as default"
	errUpdateQualityProfile  = "cannot update SonarQube Quality Profile"
	errDeleteQualityProfile  = "cannot delete SonarQube Quality Profile"
	errShowQualityProfile    = "cannot get SonarQube Quality Profile"
)

// SetupGated adds a controller that reconciles QualityProfile managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup QualityProfile controller"))
		}
	}, v1alpha1.QualityProfileGroupVersionKind)

	return nil
}

func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.QualityProfileGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		options = append(options, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		options = append(options, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		options = append(options, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.QualityProfileList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.QualityProfileList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.QualityProfileGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.QualityProfile{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	profile, ok := managedResource.(*v1alpha1.QualityProfile)
	if !ok {
		return nil, errors.New(errNotQualityProfile)
	}

	err := c.usage.Track(ctx, profile)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{
		qualityProfilesClient: instance.NewQualityProfilesClient(*config),
		rulesClient:           instance.NewRulesClient(*config),
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// qualityProfilesClient is used to interact with SonarQube Quality Profiles API
	qualityProfilesClient instance.QualityProfilesClient
	// rulesClient is used to interact with SonarQube Rules API
	rulesClient instance.RulesClient
}

// Observe checks if the external resource exists and if it matches the
// desired state of the managed resource.
func (c *external) Observe(ctx context.Context, managedResource resource.Managed) (managed.ExternalObservation, error) {
	profile, ok := managedResource.(*v1alpha1.QualityProfile)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQualityProfile)
	}

	// Use external name as the identifier to check if the resource exists
	// This allows returning early when the external name is not set
	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Retrieve the Quality Profile from SonarQube
	qualityProfile, resp, err := c.qualityProfilesClient.Show(&sonar.QualityprofilesShowOption{ //nolint:bodyclose // closed via helpers.CloseBody
		Key: externalName,
	})
	defer helpers.CloseBody(resp)

	if err != nil {
		// If the quality profile is not found, treat as resource doesn't exist
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Retrieve Quality Profile Rules (paginated)
	rules, err := instance.FetchAllQualityProfileRules(c.rulesClient, externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errShowQualityProfile)
	}

	// Update status with observed state
	profile.Status.AtProvider = instance.GenerateQualityProfileObservation(qualityProfile, rules)
	profile.Status.SetConditions(xpv1.Available())
	current := profile.Spec.ForProvider.DeepCopy()
	// Late initialize the spec with observed state (includes conditions)
	instance.LateInitializeQualityProfile(&profile.Spec.ForProvider, &profile.Status.AtProvider)

	// Generate associations between QualityProfileRules spec and observation
	associations := instance.GenerateQualityProfileRulesAssociation(profile.Spec.ForProvider.Rules, profile.Status.AtProvider.Rules)

	// Check if rules were late-initialized
	rulesLateInitialized := instance.WereQualityProfileRulesLateInitialized(current.Rules, profile.Spec.ForProvider.Rules)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.IsQualityProfileUpToDate(&profile.Spec.ForProvider, &profile.Status.AtProvider, associations),
		// Check both regular fields and conditions for late-initialization
		ResourceLateInitialized: !cmp.Equal(
			current,
			&profile.Spec.ForProvider,
			cmpopts.IgnoreFields(v1alpha1.QualityProfileParameters{}, "Rules"),
		) || rulesLateInitialized,
	}, nil
}

// Create creates the external resource and sets the external name.
func (c *external) Create(ctx context.Context, managedResource resource.Managed) (managed.ExternalCreation, error) {
	profile, ok := managedResource.(*v1alpha1.QualityProfile)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotQualityProfile)
	}

	profile.Status.SetConditions(xpv1.Creating())

	qualityProfile, resp, err := c.qualityProfilesClient.Create(instance.GenerateCreateQualityProfileOption(profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateQualityProfile)
	}

	// Set the external name to the Key of the created Quality Profile
	meta.SetExternalName(profile, qualityProfile.Profile.Key)

	// Set Quality Profile as default if specified in the spec
	if ptr.Deref(profile.Spec.ForProvider.Default, false) {
		setDefaultResp, err := c.qualityProfilesClient.SetDefault(instance.GenerateQualityprofilesSetDefaultOption(profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(setDefaultResp)

		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errDefaultQualityProfile)
		}
	}

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state of the managed resource.
func (c *external) Update(ctx context.Context, managedResource resource.Managed) (managed.ExternalUpdate, error) {
	profile, ok := managedResource.(*v1alpha1.QualityProfile)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotQualityProfile)
	}

	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf("external name is not set for Quality Profile %s", profile.Name)
	}

	// Set Quality Profile as default if specified in the spec (idempotent)
	if ptr.Deref(profile.Spec.ForProvider.Default, false) {
		updateSetDefaultResp, err := c.qualityProfilesClient.SetDefault(instance.GenerateQualityprofilesSetDefaultOption(profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(updateSetDefaultResp)

		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errDefaultQualityProfile)
		}
	}

	// Set Quality Profile name if it has changed
	if profile.Spec.ForProvider.Name != profile.Status.AtProvider.Name {
		updateResp, err := c.qualityProfilesClient.Rename(instance.GenerateRenameQualityProfileOption(externalName, profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(updateResp)

		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateQualityProfile)
		}
	}

	associations := instance.GenerateQualityProfileRulesAssociation(profile.Spec.ForProvider.Rules, profile.Status.AtProvider.Rules)

	// Sync Quality Profile Rules
	err := c.syncQualityProfileRules(profile, associations)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot sync Quality Profile Rules")
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (c *external) Delete(ctx context.Context, managedResource resource.Managed) (managed.ExternalDelete, error) {
	profile, ok := managedResource.(*v1alpha1.QualityProfile)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotQualityProfile)
	}

	profile.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	destroyResp, err := c.qualityProfilesClient.Delete(instance.GenerateDeleteQualityProfileOption(profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(destroyResp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityProfile)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (c *external) syncQualityProfileRules(profile *v1alpha1.QualityProfile, associations map[string]instance.QualityProfileRuleAssociation) error {
	if len(associations) == 0 {
		return nil
	}

	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return fmt.Errorf("external name is not set for Quality Profile %s", profile.Name)
	}

	var aggregatedErrors []error

	// Phase 1: Deactivate rules that should not be active (in observation but not in spec)
	deactivateErrors := c.deactivateUnwantedQualityProfileRules(externalName, associations)
	aggregatedErrors = append(aggregatedErrors, deactivateErrors...)

	// Phase 2: Activate rules that should be active (in spec but not in observation)
	activateErrors := c.activateMissingQualityProfileRules(externalName, associations)
	aggregatedErrors = append(aggregatedErrors, activateErrors...)

	// Phase 3: Update rules that are out of date (in both but with different parameters)
	updateErrors := c.updateOutdatedQualityProfileRules(externalName, associations)
	aggregatedErrors = append(aggregatedErrors, updateErrors...)

	if len(aggregatedErrors) > 0 {
		return errors.Errorf("encountered %d error(s) during Quality Profile rules sync: %v", len(aggregatedErrors), aggregatedErrors)
	}

	return nil
}

// deactivateUnwantedQualityProfileRules deactivates rules that are in the observation but not in the spec.
// Returns a slice of errors encountered during deactivation.
func (c *external) deactivateUnwantedQualityProfileRules(externalName string, associations map[string]instance.QualityProfileRuleAssociation) []error {
	var errs []error

	missingRules := instance.FindMissingQualityProfileRules(associations)

	for _, ruleObservation := range missingRules {
		if ruleObservation == nil {
			continue
		}

		deactivateResp, err := c.qualityProfilesClient.DeactivateRule(instance.GenerateQualityProfileDeactivateRuleOption(externalName, ruleObservation.Key)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(deactivateResp)

		if err != nil {
			errs = append(errs, errors.Wrapf(err, "cannot deactivate rule %s", ruleObservation.Key))

			continue
		}
		// Remove from associations after successful deactivation
		delete(associations, ruleObservation.Key)
	}

	return errs
}

// activateMissingQualityProfileRules activates rules that are in the spec but not in the observation.
// Returns a slice of errors encountered during activation.
func (c *external) activateMissingQualityProfileRules(externalName string, associations map[string]instance.QualityProfileRuleAssociation) []error {
	var errs []error

	nonExistingRules := instance.FindNonExistingQualityProfileRules(associations)

	for _, ruleSpec := range nonExistingRules {
		if ruleSpec == nil {
			continue
		}

		activateResp, err := c.qualityProfilesClient.ActivateRule(instance.GenerateQualityProfileActivateRuleOption(externalName, *ruleSpec)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(activateResp)

		if err != nil {
			errs = append(errs, errors.Wrapf(err, "cannot activate rule %s", ruleSpec.Rule))

			continue
		}
		// Update association to reflect the activation (mark as up to date)
		associations[ruleSpec.Rule] = instance.QualityProfileRuleAssociation{
			Spec:        ruleSpec,
			Observation: nil, // Will be populated on next Observe
			UpToDate:    true,
		}
	}

	return errs
}

// updateOutdatedQualityProfileRules updates rules that have different parameters between spec and observation.
// For SonarQube, updating a rule means re-activating it with the new parameters.
// Returns a slice of errors encountered during update.
func (c *external) updateOutdatedQualityProfileRules(externalName string, associations map[string]instance.QualityProfileRuleAssociation) []error {
	var errs []error

	outdatedRules := instance.FindNotUpToDateQualityProfileRules(associations)

	for _, assoc := range outdatedRules {
		if assoc.Spec == nil {
			continue
		}
		// Re-activate rule with new parameters to update it
		activateResp, err := c.qualityProfilesClient.ActivateRule(instance.GenerateQualityProfileActivateRuleOption(externalName, *assoc.Spec)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(activateResp)

		if err != nil {
			errs = append(errs, errors.Wrapf(err, "cannot update rule %s", assoc.Spec.Rule))

			continue
		}
		// Update association to reflect the update
		associations[assoc.Spec.Rule] = instance.QualityProfileRuleAssociation{
			Spec:        assoc.Spec,
			Observation: assoc.Observation,
			UpToDate:    true,
		}
	}

	return errs
}
