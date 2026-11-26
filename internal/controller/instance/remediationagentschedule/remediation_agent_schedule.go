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

// Package remediationagentschedule provides a controller for
// RemediationAgentSchedule resources.
package remediationagentschedule

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errGetSchedule indicates the schedule could not be retrieved.
	errGetSchedule = "cannot get SonarQube remediation agent schedule"
	// errCreateSchedule indicates the schedule could not be created.
	errCreateSchedule = "cannot create SonarQube remediation agent schedule"
	// errUpdateSchedule indicates the schedule could not be updated.
	errUpdateSchedule = "cannot update SonarQube remediation agent schedule"
	// errDeleteSchedule indicates the schedule could not be deleted.
	errDeleteSchedule = "cannot delete SonarQube remediation agent schedule"
	// errUnresolvedProject indicates the project of the schedule is unknown.
	errUnresolvedProject = "the project reference of the remediation agent schedule is not resolved"

	// selectionModeAll applies the instance-wide schedule to every project.
	selectionModeAll = "ALL"
	// selectionModeSelected restricts it to a list of projects.
	selectionModeSelected = "SELECTED"
)

// SetupGated adds a controller that reconciles RemediationAgentSchedule
// resources, registered behind a gate so it only runs when the gate is opened.
func SetupGated(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(v1alpha1.RemediationAgentScheduleGroupKind)

	opts.Gate.Register(func() {
		err := Setup(mgr, opts, name)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup RemediationAgentSchedule controller"))
		}
	}, v1alpha1.RemediationAgentScheduleGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles RemediationAgentSchedule resources.
func Setup(mgr ctrl.Manager, opts helpers.Options, name string) error {
	options := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*v1alpha1.RemediationAgentSchedule](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewRemediationAgentSchedulesClient,
		}),
		managed.WithDeterministicExternalName(true),
		// The external name derives from the spec, possibly through a reference.
		// The default initializer would set it to the resource name instead.
		managed.WithInitializers(),
	)

	err := helpers.AddMRStateRecorder(mgr, opts, &v1alpha1.RemediationAgentScheduleList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.RemediationAgentScheduleGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.RemediationAgentSchedule{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.RemediationAgentSchedulesClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube client.
func (c *connector) Connect(ctx context.Context, schedule *v1alpha1.RemediationAgentSchedule) (managed.TypedExternalClient[*v1alpha1.RemediationAgentSchedule], error) {
	err := c.usage.Track(ctx, schedule)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, schedule)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for
// RemediationAgentSchedule resources.
type external struct {
	client instance.RemediationAgentSchedulesClient
}

// Observe checks whether SonarQube holds a schedule for the scope of the
// resource and whether it matches the desired state. A schedule that already
// exists is adopted, as SonarQube holds at most one per scope.
func (c *external) Observe(ctx context.Context, schedule *v1alpha1.RemediationAgentSchedule) (managed.ExternalObservation, error) {
	scheduleID := instance.RemediationAgentScheduleID(&schedule.Spec.ForProvider)
	if scheduleID == "" {
		// References are not resolved during deletion: a schedule whose project
		// was never resolved was never created either.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, resp, err := c.client.GetScheduledAgentConfig(ctx, scheduleID) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetSchedule)
	}

	schedule.Status.AtProvider = instance.GenerateRemediationAgentScheduleObservation(config)

	// The instance-wide schedule cannot be deleted: disabled is as good as gone.
	if meta.WasDeleted(schedule) && scheduleID == instance.GlobalRemediationAgentScheduleID && !schedule.Status.AtProvider.Enabled {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	schedule.SetConditions(xpv1.Available())

	former := schedule.Spec.ForProvider.DeepCopy()
	instance.LateInitializeRemediationAgentSchedule(&schedule.Spec.ForProvider, &schedule.Status.AtProvider)
	lateInitialized := instance.IsRemediationAgentScheduleLateInitialized(former, &schedule.Spec.ForProvider)

	// Persisting the external name goes through late initialization.
	if meta.GetExternalName(schedule) != scheduleID {
		meta.SetExternalName(schedule, scheduleID)

		lateInitialized = true
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsRemediationAgentScheduleUpToDate(&schedule.Spec.ForProvider, &schedule.Status.AtProvider),
		ResourceLateInitialized: lateInitialized,
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the schedule in SonarQube.
func (c *external) Create(ctx context.Context, schedule *v1alpha1.RemediationAgentSchedule) (managed.ExternalCreation, error) {
	schedule.SetConditions(xpv1.Creating())

	scheduleID := instance.RemediationAgentScheduleID(&schedule.Spec.ForProvider)
	if scheduleID == "" {
		return managed.ExternalCreation{}, errors.New(errUnresolvedProject)
	}

	_, resp, err := c.client.CreateScheduledAgentConfig(ctx, instance.GenerateRemediationAgentScheduleCreateOptions(&schedule.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateSchedule)
	}

	meta.SetExternalName(schedule, scheduleID)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update replaces the schedule in SonarQube with the desired one.
func (c *external) Update(ctx context.Context, schedule *v1alpha1.RemediationAgentSchedule) (managed.ExternalUpdate, error) {
	scheduleID := instance.RemediationAgentScheduleID(&schedule.Spec.ForProvider)
	if scheduleID == "" {
		return managed.ExternalUpdate{}, errors.New(errUnresolvedProject)
	}

	// SonarQube replaces the whole configuration. Unset fields are completed
	// from a fresh read, as persisting late initialized fields resets the status.
	config, getResp, err := c.client.GetScheduledAgentConfig(ctx, scheduleID) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(getResp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetSchedule)
	}

	observation := instance.GenerateRemediationAgentScheduleObservation(config)
	options := instance.GenerateRemediationAgentScheduleUpdateOptions(&schedule.Spec.ForProvider, &observation)

	_, resp, err := c.client.UpdateScheduledAgentConfig(ctx, scheduleID, options) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateSchedule)
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete deletes a project schedule. The instance-wide schedule cannot be
// deleted and is disabled instead, keeping its other settings.
func (c *external) Delete(ctx context.Context, schedule *v1alpha1.RemediationAgentSchedule) (managed.ExternalDelete, error) {
	schedule.SetConditions(xpv1.Deleting())

	scheduleID := instance.RemediationAgentScheduleID(&schedule.Spec.ForProvider)

	switch scheduleID {
	case "":
		return managed.ExternalDelete{}, nil
	case instance.GlobalRemediationAgentScheduleID:
		return managed.ExternalDelete{}, c.disableGlobalSchedule(ctx)
	}

	resp, err := c.client.DeleteScheduledAgentConfig(ctx, scheduleID) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteSchedule)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// disableGlobalSchedule disables the instance-wide schedule. It resends the
// configuration as SonarQube currently holds it, so that deleting the resource
// does not apply pending spec changes.
func (c *external) disableGlobalSchedule(ctx context.Context) error {
	config, getResp, err := c.client.GetScheduledAgentConfig(ctx, instance.GlobalRemediationAgentScheduleID) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(getResp)

	if err != nil {
		return errors.Wrap(err, errGetSchedule)
	}

	// SonarQube prunes deleted projects from the selection, but rejects an empty
	// one on write, which would block the deletion.
	selectionMode := config.ProjectSelectionMode
	if selectionMode == selectionModeSelected && len(config.SelectedProjectKeys) == 0 {
		selectionMode = selectionModeAll
	}

	_, resp, err := c.client.UpdateScheduledAgentConfig(ctx, instance.GlobalRemediationAgentScheduleID, &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{ //nolint:bodyclose // closed via helpers.CloseBody
		Enabled:              new(false),
		Frequency:            config.Frequency,
		Hour:                 &config.Hour,
		DayOfWeek:            config.DayOfWeek,
		Timezone:             config.Timezone,
		MaxIssuesPerRun:      &config.MaxIssuesPerRun,
		MaxOpenPRs:           &config.MaxOpenPRs,
		ProjectSelectionMode: selectionMode,
		SelectedProjectKeys:  config.SelectedProjectKeys,
	})
	defer helpers.CloseBody(resp)

	return errors.Wrap(err, errDeleteSchedule)
}
