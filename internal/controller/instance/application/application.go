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

// Package application provides a controller for Application resources.
package application

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotApplication indicates the managed resource is not an Application.
	errNotApplication = "managed resource is not an Application custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errExternalNameNotSet indicates the external name annotation is missing.
	errExternalNameNotSet = "external name is not set for Application resource %s"

	// errObserveApplication indicates application observation failed.
	errObserveApplication = "cannot observe SonarQube Application"
	// errObserveApplicationBranch indicates branch observation failed.
	errObserveApplicationBranch = "cannot observe SonarQube Application branch %s"
	// errCreateApplication indicates application creation in SonarQube failed.
	errCreateApplication = "cannot create SonarQube Application"
	// errUpdateApplication indicates application update in SonarQube failed.
	errUpdateApplication = "cannot update SonarQube Application"
	// errDeleteApplication indicates application deletion in SonarQube failed.
	errDeleteApplication = "cannot delete SonarQube Application"
	// errAddProject indicates adding a project to the application failed.
	errAddProject = "cannot add project %s to SonarQube Application"
	// errRemoveProject indicates removing a project from the application failed.
	errRemoveProject = "cannot remove project %s from SonarQube Application"
	// errCreateBranch indicates application branch creation failed.
	errCreateBranch = "cannot create SonarQube Application branch %s"
	// errUpdateBranch indicates application branch update failed.
	errUpdateBranch = "cannot update SonarQube Application branch %s"
	// errDeleteBranch indicates application branch deletion failed.
	errDeleteBranch = "cannot delete SonarQube Application branch %s"
)

// SetupGated adds a controller that reconciles Application managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Application controller"))
		}
	}, v1alpha1.ApplicationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Application managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.ApplicationGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewApplicationsClient,
		}),
		managed.WithLogger(options.Logger.WithValues("controller", name)),
		managed.WithPollInterval(options.PollInterval),
		managed.WithRecorder(helpers.NewEventRecorder(mgr, name)),
	}

	if options.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if options.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(options.ChangeLogOptions.ChangeLogger))
	}

	if options.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(options.MetricOptions.MRMetrics))
	}

	if options.MetricOptions != nil && options.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.ApplicationList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ApplicationList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ApplicationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Application{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its
// Connect method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.ApplicationsClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube applications client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	application, isValid := managedResource.(*v1alpha1.Application)
	if !isValid {
		return nil, errors.New(errNotApplication)
	}

	err := c.usage.Track(ctx, application)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, application)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// external implements the ExternalClient interface for Application resources.
type external struct {
	client instance.ApplicationsClient
}

// Observe checks if the Application exists in SonarQube and whether its
// name, description, member projects and branches match the desired state.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	application, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplication)
	}

	externalName := meta.GetExternalName(application)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	result, resp, err := c.client.Show(ctx, &sonar.ApplicationsShowOptions{Application: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObserveApplication)
	}

	if result == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	branchProjects, err := c.observeBranchProjects(ctx, externalName, result.Application.Branches)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	application.Status.AtProvider = instance.GenerateApplicationObservation(&result.Application, branchProjects)
	application.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  instance.IsApplicationUpToDate(&application.Spec.ForProvider, &application.Status.AtProvider),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create creates the Application in SonarQube and sets the external name
// to the application key. Member projects and branches are reconciled by
// the subsequent Update.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	application, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplication)
	}

	application.Status.SetConditions(xpv1.Creating())

	_, resp, err := c.client.Create(ctx, instance.GenerateApplicationCreateOptions(&application.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateApplication)
	}

	meta.SetExternalName(application, application.Spec.ForProvider.Key)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update reconciles the Application name, description, member projects and
// branches in SonarQube.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	application, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplication)
	}

	externalName := meta.GetExternalName(application)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.Errorf(errExternalNameNotSet, application.Name)
	}

	spec := &application.Spec.ForProvider
	observation := &application.Status.AtProvider

	if spec.Name != observation.Name || !helpers.IsComparablePtrEqualComparable(spec.Description, observation.Description) {
		resp, err := c.client.Update(ctx, instance.GenerateApplicationUpdateOptions(externalName, spec, observation)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateApplication)
		}
	}

	// Projects are added before branches are reconciled, since branches can
	// only reference member projects, and removed afterwards so that no
	// desired branch still references a removed project.
	err := c.addProjects(ctx, externalName, instance.ApplicationProjectsToAdd(spec.Projects, observation.Projects))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.updateBranches(ctx, externalName, spec.Branches, observation.Branches)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.removeProjects(ctx, externalName, instance.ApplicationProjectsToRemove(spec.Projects, observation.Projects))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete deletes the Application from SonarQube.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	application, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotApplication)
	}

	application.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(application)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(ctx, &sonar.ApplicationsDeleteOptions{Application: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteApplication)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// observeBranchProjects fetches the project branches used by each non-main
// branch of the application.
func (c *external) observeBranchProjects(ctx context.Context, key string, branches []sonar.ApplicationBranch) (map[string][]sonar.ApplicationProject, error) {
	branchProjects := make(map[string][]sonar.ApplicationProject, len(branches))

	for _, branch := range branches {
		if branch.IsMain {
			continue
		}

		result, resp, err := c.client.Show(ctx, &sonar.ApplicationsShowOptions{Application: key, Branch: branch.Name}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, errors.Wrapf(err, errObserveApplicationBranch, branch.Name)
		}

		if result != nil {
			branchProjects[branch.Name] = result.Application.Projects
		}
	}

	return branchProjects, nil
}

// addProjects adds the given projects to the application.
func (c *external) addProjects(ctx context.Context, key string, projects []string) error {
	for _, project := range projects {
		resp, err := c.client.AddProject(ctx, &sonar.ApplicationsAddProjectOptions{Application: key, Project: project}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errAddProject, project)
		}
	}

	return nil
}

// removeProjects removes the given projects from the application.
func (c *external) removeProjects(ctx context.Context, key string, projects []string) error {
	for _, project := range projects {
		resp, err := c.client.RemoveProject(ctx, &sonar.ApplicationsRemoveProjectOptions{Application: key, Project: project}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errRemoveProject, project)
		}
	}

	return nil
}

// updateBranches deletes the unwanted non-main application branches, then
// creates the missing branches and updates the drifted ones.
func (c *external) updateBranches(ctx context.Context, key string, spec []v1alpha1.ApplicationBranchParameters, observation []v1alpha1.ApplicationBranchObservation) error {
	for _, branch := range instance.ApplicationBranchesToDelete(spec, observation) {
		resp, err := c.client.DeleteBranch(ctx, &sonar.ApplicationsDeleteBranchOptions{Application: key, Branch: branch}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errDeleteBranch, branch)
		}
	}

	for i := range spec {
		branch := &spec[i]

		observed, found := instance.FindApplicationBranchObservation(observation, branch.Name)
		if !found {
			resp, err := c.client.CreateBranch(ctx, instance.GenerateApplicationCreateBranchOptions(key, branch)) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return errors.Wrapf(err, errCreateBranch, branch.Name)
			}

			continue
		}

		if instance.IsApplicationBranchUpToDate(branch, observed) {
			continue
		}

		resp, err := c.client.UpdateBranch(ctx, instance.GenerateApplicationUpdateBranchOptions(key, branch)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errUpdateBranch, branch.Name)
		}
	}

	return nil
}
