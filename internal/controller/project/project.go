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

package project

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	stderrors "errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
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
	errNotProject   = "managed resource is not a Project custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
)

// SetupGated adds a controller that reconciles Project managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Project controller"))
		}
	}, v1alpha1.ProjectGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error { //nolint:varnamelen // consistent with other controllers
	name := managed.ControllerName(v1alpha1.ProjectGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.ProjectList{}, o.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ProjectList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ProjectGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Project{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, o.GlobalRateLimiter))
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
	project, isValid := managedResource.(*v1alpha1.Project)
	if !isValid {
		return nil, errors.New(errNotProject)
	}

	err := c.usage.Track(ctx, project)
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
		projectsClient:              instance.NewProjectsClient(*config),
		projectLinksClient:          instance.NewProjectLinksClient(*config),
		projectBranchesClient:       instance.NewProjectBranchesClient(*config),
		projectNewCodePeriodsClient: instance.NewNewCodePeriodsClient(*config),
		qualityGatesClient:          instance.NewQualityGatesClient(*config),
		qualityProfilesClient:       instance.NewQualityProfilesClient(*config),
		projectTagsClient:           instance.NewProjectTagsClient(*config),
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// projectsClient is used to interact with SonarQube Projects API
	projectsClient instance.ProjectsClient
	// projectLinksClient is used to interact with SonarQube Project Links API
	projectLinksClient instance.ProjectLinksClient
	// projectBranchesClient is used to interact with SonarQube Project Branches API
	projectBranchesClient instance.ProjectBranchesClient
	// projectNewCodePeriodsClient is used to interact with SonarQube Project New Code Periods API
	projectNewCodePeriodsClient instance.NewCodePeriodsClient
	// qualityGatesClient is used to interact with SonarQube Quality Gates API
	qualityGatesClient instance.QualityGatesClient
	// qualityProfilesClient is used to interact with SonarQube Quality Profiles API
	qualityProfilesClient instance.QualityProfilesClient
	// projectTagsClient is used to interact with SonarQube Project Tags API
	projectTagsClient instance.ProjectTagsClient
}

// observeResult holds the results of all concurrent observation API calls.
type observeResult struct {
	branches             map[string]v1alpha1.ProjectBranchObservation
	mainBranch           string
	links                map[string]v1alpha1.ProjectLinkObservation
	projectNewCodePeriod v1alpha1.ProjectNewCodePeriodObservation
	branchNewCodePeriods map[string]v1alpha1.ProjectNewCodePeriodObservation
	qualityGateName      string
	qualityProfiles      map[string]v1alpha1.ProjectQualityProfileObservation
	errors               []error
}

// Observe observes the external resource and updates the managed resource's status with the observed state of the external resource.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	externalName := meta.GetExternalName(project)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Check if the Project exists in SonarQube and update the observation with the project details if it exists.
	exists, err := c.observeProjectExistence(externalName, &project.Status.AtProvider)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot observe Project existence")
	}

	if !exists {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Make all observation API calls concurrently, collecting results safely.
	result := c.observeProjectDetails(externalName, project.Status.AtProvider.Uuid)

	// Safely populate observation from collected results (single-threaded)
	c.applyObserveResult(&project.Status.AtProvider, &result)

	if len(result.errors) > 0 {
		return managed.ExternalObservation{
			ResourceExists: true,
		}, stderrors.Join(result.errors...)
	}

	// Mark the managed resource as available now that the external resource exists and has been observed.
	project.Status.SetConditions(xpv1.Available())

	former := project.Spec.ForProvider.DeepCopy()

	instance.LateInitializeProject(&project.Spec.ForProvider, &project.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsProjectUpToDate(&project.Spec.ForProvider, &project.Status.AtProvider),
		ResourceLateInitialized: instance.IsProjectLateInitialized(former, &project.Spec.ForProvider),
	}, nil
}

// Create creates the project in SonarQube using the SonarQube API client.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	project.SetConditions(xpv1.Creating())

	// Create the SonarQube Project using the SonarQube API client
	createdProject, resp, err := c.projectsClient.Create(instance.GenerateProjectsCreateOptions(project.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create Project")
	}

	meta.SetExternalName(project, createdProject.Project.Key)

	// After creating the project, we need to update the links, branches and new code periods of the project to match the desired state in the spec.
	// We will perform these updates in the Update method since the Create method is only responsible for creating the project and not updating it.

	return managed.ExternalCreation{}, nil
}

// Update will update the external resource if it exists and is not up to date.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	projectKey := meta.GetExternalName(project)
	if projectKey == "" {
		return managed.ExternalUpdate{}, errors.New("external name is not set for the Project")
	}

	allErrors := c.updateProjectConcurrently(project, projectKey, project.Status.AtProvider.Uuid)

	if len(allErrors) > 0 {
		return managed.ExternalUpdate{}, stderrors.Join(allErrors...)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Delete will delete the external resource if it exists.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProject)
	}

	externalName := meta.GetExternalName(project)
	if externalName == "" {
		// External name is not set, so we cannot delete the resource in SonarQube.
		// However, we can consider the resource deleted since it does not exist in SonarQube.
		return managed.ExternalDelete{}, nil
	}

	project.SetConditions(xpv1.Deleting())

	// Delete the SonarQube Project using the SonarQube API client
	resp, err := c.projectsClient.Delete(instance.GenerateProjectDeleteOptions(externalName)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete Project")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Project external client.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
