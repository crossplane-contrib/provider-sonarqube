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

package qualitygate

import (
	"context"
	"fmt"

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

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
	errNotQualityGate = "managed resource is not a QualityGate custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"

	errCreateQualityGate  = "cannot create SonarQube Quality Gate"
	errDefaultQualityGate = "cannot set SonarQube Quality Gate as default"
	errUpdateQualityGate  = "cannot update SonarQube Quality Gate"
	errDeleteQualityGate  = "cannot delete SonarQube Quality Gate"
	errShowQualityGate    = "cannot get SonarQube Quality Gate"
)

// SetupGated adds a controller that reconciles QualityGate managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup QualityGate controller"))
		}
	}, v1alpha1.QualityGateGroupVersionKind)
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.QualityGateGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewQualityGatesClient}),
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.QualityGateList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.QualityGateList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.QualityGateGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.QualityGate{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.QualityGatesClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.QualityGate)
	if !ok {
		return nil, errors.New(errNotQualityGate)
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

	return &external{qualityGatesClient: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// qualityGatesClient is used to interact with SonarQube Quality Gates API
	qualityGatesClient instance.QualityGatesClient
}

// Observe checks if the external resource exists and if it matches the
// desired state of the managed resource.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.QualityGate)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQualityGate)
	}

	// Use external name as the identifier to check if the resource exists
	// This allows returning early when the external name is not set
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Retrieve the Quality Gate from SonarQube
	qualityGate, resp, err := c.qualityGatesClient.Show(&sonargo.QualitygatesShowOption{ //nolint:bodyclose // closed via helpers.CloseBody
		Name: externalName,
	})
	defer helpers.CloseBody(resp)
	if err != nil {
		// If the quality gate is not found, treat as resource doesn't exist
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update status with observed state
	cr.Status.AtProvider = instance.GenerateQualityGateObservation(qualityGate)
	cr.Status.SetConditions(xpv1.Available())

	current := cr.Spec.ForProvider.DeepCopy()
	// Late initialize the spec with observed state (includes conditions)
	instance.LateInitializeQualityGate(&cr.Spec.ForProvider, &cr.Status.AtProvider)

	// Generate associations between QualityGateConditions spec and observation
	associations := instance.GenerateQualityGateConditionsAssociation(cr.Spec.ForProvider.Conditions, cr.Status.AtProvider.Conditions)

	// Check if conditions were late-initialized
	conditionsLateInitialized := instance.WereQualityGateConditionsLateInitialized(current.Conditions, cr.Spec.ForProvider.Conditions)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.IsQualityGateUpToDate(&cr.Spec.ForProvider, &cr.Status.AtProvider, associations),
		// Check both regular fields and conditions for late-initialization
		ResourceLateInitialized: !cmp.Equal(
			current,
			&cr.Spec.ForProvider,
			cmpopts.IgnoreFields(v1alpha1.QualityGateParameters{}, "Conditions"),
		) || conditionsLateInitialized,
	}, nil
}

// syncQualityGateConditions synchronizes the Quality Gate Conditions in SonarQube
// It deletes unwanted conditions, creates missing conditions, and updates out-of-date conditions
func (c *external) syncQualityGateConditions(qualityGate *v1alpha1.QualityGate, qualityGateConditionAssociations map[string]instance.QualityGateConditionAssociation) error {
	if len(qualityGateConditionAssociations) == 0 {
		return nil
	}

	externalName := meta.GetExternalName(qualityGate)
	if externalName == "" {
		return fmt.Errorf("external name is not set for Quality Gate %s", qualityGate.Name)
	}

	if err := c.deleteUnwantedQualityGateConditions(qualityGateConditionAssociations); err != nil {
		return err
	}

	if err := c.createMissingQualityGateConditions(externalName, qualityGateConditionAssociations); err != nil {
		return err
	}

	if err := c.updateOutdatedQualityGateConditions(qualityGateConditionAssociations); err != nil {
		return err
	}

	return nil
}

// deleteUnwantedQualityGateConditions deletes Quality Gate Conditions that are no longer needed
func (c *external) deleteUnwantedQualityGateConditions(qualityGateConditionAssociations map[string]instance.QualityGateConditionAssociation) error {
	missingQualityGateConditions := instance.FindMissingQualityGateConditions(qualityGateConditionAssociations)
	for _, conditionObservation := range missingQualityGateConditions {
		if conditionObservation == nil {
			continue
		}
		deleteResponse, err := c.qualityGatesClient.DeleteCondition(instance.GenerateDeleteQualityGateConditionOption(conditionObservation.ID)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(deleteResponse)
		if err != nil {
			return errors.Wrapf(err, "cannot delete SonarQube Quality Gate Condition with ID %s", conditionObservation.ID)
		}
		// Delete the condition from the associations map after successful deletion
		delete(qualityGateConditionAssociations, conditionObservation.ID)
	}
	return nil
}

// createMissingQualityGateConditions creates Quality Gate Conditions that are specified but do not exist
func (c *external) createMissingQualityGateConditions(externalName string, qualityGateConditionAssociations map[string]instance.QualityGateConditionAssociation) error {
	nonExistingQualityGateConditions := instance.FindNonExistingQualityGateConditions(qualityGateConditionAssociations)
	for _, conditionSpec := range nonExistingQualityGateConditions {
		if conditionSpec == nil {
			continue
		}

		// Find the old key for this spec (likely "new:{metric}")
		var oldKey string
		for key, assoc := range qualityGateConditionAssociations {
			if assoc.Spec == conditionSpec {
				oldKey = key
				break
			}
		}

		qualityGateCondition, createResponse, err := c.qualityGatesClient.CreateCondition(instance.GenerateCreateQualityGateConditionOption(externalName, *conditionSpec)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(createResponse)
		if err != nil {
			return errors.Wrapf(err, "cannot create SonarQube Quality Gate Condition for Quality Gate %s", externalName)
		}
		conditionObservation := instance.GenerateQualityGateConditionObservationFromCreate(qualityGateCondition)

		// Remove the old "new:{metric}" key from the associations map
		if oldKey != "" {
			delete(qualityGateConditionAssociations, oldKey)
		}

		// Add the new condition to the associations map with the real ID as the key
		// Note: Late initialization of the spec happens in Observe, not here
		qualityGateConditionAssociations[qualityGateCondition.ID] = instance.QualityGateConditionAssociation{
			Spec:        conditionSpec,
			Observation: conditionObservation,
			UpToDate:    true,
		}
	}
	return nil
}

// updateOutdatedQualityGateConditions updates Quality Gate Conditions that are out of date
func (c *external) updateOutdatedQualityGateConditions(qualityGateConditionAssociations map[string]instance.QualityGateConditionAssociation) error {
	outdatedQualityGateConditions := instance.FindNotUpToDateQualityGateConditions(qualityGateConditionAssociations)
	for _, association := range outdatedQualityGateConditions {
		if association.Spec == nil || association.Observation == nil {
			continue
		}
		updateResponse, err := c.qualityGatesClient.UpdateCondition(instance.GenerateUpdateQualityGateConditionOption(association.Observation.ID, *association.Spec)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(updateResponse)
		if err != nil {
			return errors.Wrapf(err, "cannot update SonarQube Quality Gate Condition with ID %s", *association.Spec.Id)
		}
	}
	return nil
}

// Create creates the external resource and sets the external name
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.QualityGate)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotQualityGate)
	}

	cr.Status.SetConditions(xpv1.Creating())

	qualityGateCreateOptions := instance.GenerateQualityGateCreateOptions(cr.Spec.ForProvider)

	qualityGate, resp, err := c.qualityGatesClient.Create(qualityGateCreateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateQualityGate)
	}

	// Set the external name to the Name of the created Quality Gate
	meta.SetExternalName(cr, qualityGate.Name)

	// Set Quality Gate as default if specified in the spec
	if cr.Spec.ForProvider.Default != nil && *cr.Spec.ForProvider.Default {
		setDefaultResp, err := c.qualityGatesClient.SetAsDefault(&sonargo.QualitygatesSetAsDefaultOption{ //nolint:bodyclose // closed via helpers.CloseBody
			Name: qualityGate.Name,
		})
		defer helpers.CloseBody(setDefaultResp)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errDefaultQualityGate)
		}
	}

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state of the managed resource
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.QualityGate)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotQualityGate)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf("external name is not set for Quality Gate %s", cr.Name)
	}

	// Set Quality Gate as default if specified in the spec (idempotent)
	if cr.Spec.ForProvider.Default != nil && *cr.Spec.ForProvider.Default {
		updateSetDefaultResp, err := c.qualityGatesClient.SetAsDefault(&sonargo.QualitygatesSetAsDefaultOption{ //nolint:bodyclose // closed via helpers.CloseBody
			Name: cr.Spec.ForProvider.Name,
		})
		defer helpers.CloseBody(updateSetDefaultResp)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errDefaultQualityGate)
		}
	}

	associations := instance.GenerateQualityGateConditionsAssociation(cr.Spec.ForProvider.Conditions, cr.Status.AtProvider.Conditions)

	// Sync Quality Gate Conditions
	if err := c.syncQualityGateConditions(cr, associations); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot sync Quality Gate Conditions")
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.QualityGate)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotQualityGate)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	destroyResp, err := c.qualityGatesClient.Destroy(&sonargo.QualitygatesDestroyOption{ //nolint:bodyclose // closed via helpers.CloseBody
		Name: externalName,
	})
	defer helpers.CloseBody(destroyResp)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityGate)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
