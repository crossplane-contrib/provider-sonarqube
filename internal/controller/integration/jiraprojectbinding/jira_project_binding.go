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

// Package jiraprojectbinding provides a controller for JiraProjectBinding
// resources.
package jiraprojectbinding

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errSearchProject indicates the project lookup failed.
	errSearchProject = "cannot search SonarQube project"
	// errGetBinding indicates the project binding retrieval failed.
	errGetBinding = "cannot get Jira project binding"
	// errCreateBinding indicates the project binding creation failed.
	errCreateBinding = "cannot create Jira project binding"
	// errUpdateBinding indicates the project binding update failed.
	errUpdateBinding = "cannot update Jira project binding"
	// errDeleteBinding indicates the project binding deletion failed.
	errDeleteBinding = "cannot delete Jira project binding"
	// errNoBindingID indicates the create API response has no ID.
	errNoBindingID = "API response missing Jira project binding id"
	// errEmptySearch indicates the project search returned no response.
	errEmptySearch = "empty API response"

	// errProjectNotFound indicates the project to bind does not exist.
	errProjectNotFound = "SonarQube project %q does not exist"
	// errBindingExists indicates the project already has a binding, which
	// must be explicitly imported.
	errBindingExists = "SonarQube project %q already has a Jira binding (id %q): " +
		"set the crossplane.io/external-name annotation to this id to manage it"
	// errBindingChanged indicates the project no longer holds the managed
	// binding.
	errBindingChanged = "the project no longer has the managed Jira binding"
	// errNoProjectID indicates SonarQube did not report the project ID.
	errNoProjectID = "SonarQube did not report the id of project %q: SonarQube 2025.6 or later is required"
)

// SetupGated adds a controller that reconciles JiraProjectBinding managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup JiraProjectBinding controller"))
		}
	}, v1alpha1.JiraProjectBindingGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles JiraProjectBinding managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.JiraProjectBindingGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like the binding ID would take
		// over an existing binding.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.JiraProjectBinding](&connector{
			kube:          mgr.GetClient(),
			usage:         resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn:  integration.NewJiraClient,
			newProjectsFn: instance.NewProjectsClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.JiraProjectBindingList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.JiraProjectBindingGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.JiraProjectBinding{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube          client.Client
	usage         *resource.ProviderConfigUsageTracker
	newServiceFn  func(config common.Config) integration.JiraClient
	newProjectsFn func(config common.Config) instance.ProjectsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (managed.TypedExternalClient[*v1alpha1.JiraProjectBinding], error) {
	err := c.usage.Track(ctx, binding)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, binding)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config), projects: c.newProjectsFn(*config)}, nil
}

// external implements the ExternalClient interface for JiraProjectBinding
// resources.
type external struct {
	client   integration.JiraClient
	projects instance.ProjectsClient
}

// Observe checks whether the project binding exists in SonarQube and whether
// it is up to date.
//
// The external name is the binding ID assigned by SonarQube. A binding with
// another ID is reported as missing, so it is never updated nor deleted. A
// binding disappears with its project.
func (c *external) Observe(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(binding)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	_, observed, err := c.findBinding(ctx, binding)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if observed == nil || observed.ID != externalName {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	binding.Status.AtProvider = integration.GenerateJiraProjectBindingObservation(observed)

	binding.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: integration.IsJiraProjectBindingUpToDate(&binding.Spec.ForProvider, &binding.Status.AtProvider),
	}, nil
}

// Create binds the project to the Jira project and sets the external name to
// the ID assigned by SonarQube.
//
// The creation endpoint also updates, so Create fails when the project already
// has a binding instead of taking it over.
func (c *external) Create(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (managed.ExternalCreation, error) {
	binding.SetConditions(xpv1.Creating())

	projectKey := ptr.Deref(binding.Spec.ForProvider.ProjectKey, "")

	projectID, existing, err := c.findBinding(ctx, binding)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if projectID == "" {
		return managed.ExternalCreation{}, errors.Errorf(errProjectNotFound, projectKey)
	}

	if existing != nil {
		return managed.ExternalCreation{}, errors.Errorf(errBindingExists, projectKey, existing.ID)
	}

	created, err := c.putBinding(ctx, projectID, binding)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBinding)
	}

	if created == nil {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoBindingID), errCreateBinding)
	}

	meta.SetExternalName(binding, created.ID)

	return managed.ExternalCreation{}, nil
}

// Update updates the project binding to match the desired state.
func (c *external) Update(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (managed.ExternalUpdate, error) {
	projectID, existing, err := c.findBinding(ctx, binding)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	// The update endpoint also creates, and is keyed by project: only call it
	// while the project holds the managed binding.
	if existing == nil || existing.ID != meta.GetExternalName(binding) {
		return managed.ExternalUpdate{}, errors.New(errBindingChanged)
	}

	_, err = c.putBinding(ctx, projectID, binding)

	return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBinding)
}

// Delete removes the project binding from SonarQube. It only deletes the
// binding identified by the external name.
func (c *external) Delete(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (managed.ExternalDelete, error) {
	binding.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(binding)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	// The deletion endpoint is keyed by project, so make sure the binding it
	// holds is still the managed one.
	projectID, existing, err := c.findBinding(ctx, binding)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	if existing == nil || existing.ID != externalName {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteProjectBinding(ctx, integration.GenerateJiraProjectBindingOptions(projectID)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	// The client resends a DELETE after a transient failure, so a 404 can mean
	// an earlier attempt succeeded.
	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBinding)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// findBinding returns the ID of the project of the resource, empty if the
// project does not exist, and the Jira binding of that project, nil if it has
// none.
func (c *external) findBinding(ctx context.Context, binding *v1alpha1.JiraProjectBinding) (string, *integration.JiraProjectBinding, error) {
	projectID, err := c.findProjectID(ctx, ptr.Deref(binding.Spec.ForProvider.ProjectKey, ""))
	if err != nil || projectID == "" {
		return "", nil, err
	}

	raw, resp, err := c.client.GetProjectBinding(ctx, integration.GenerateJiraProjectBindingOptions(projectID)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return projectID, nil, nil
		}

		return "", nil, errors.Wrap(err, errGetBinding)
	}

	observed, err := integration.ParseJiraProjectBinding(raw)
	if err != nil {
		return "", nil, errors.Wrap(err, errGetBinding)
	}

	return projectID, observed, nil
}

// findProjectID returns the ID of the project with the given key, which the
// Jira endpoints identify projects with. It is empty if no project has this
// key.
func (c *external) findProjectID(ctx context.Context, projectKey string) (string, error) {
	// A resource without project key cannot have bound anything. Searching
	// without a key would list every project.
	if projectKey == "" {
		return "", nil
	}

	projects, resp, err := c.projects.Search(ctx, instance.GenerateProjectSearchOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return "", errors.Wrap(err, errSearchProject)
	}

	if projects == nil {
		return "", errors.Wrap(errors.New(errEmptySearch), errSearchProject)
	}

	for _, project := range projects.Components {
		if project.Key != projectKey {
			continue
		}

		// SonarQube reports the project ID since 2025.6. Without it, the
		// binding must not be reported as missing.
		if project.ProjectUuid == "" {
			return "", errors.Errorf(errNoProjectID, projectKey)
		}

		return project.ProjectUuid, nil
	}

	return "", nil
}

// putBinding creates or updates the binding of the project with the given ID.
// It returns nil if SonarQube returned no binding.
func (c *external) putBinding(ctx context.Context, projectID string, binding *v1alpha1.JiraProjectBinding) (*integration.JiraProjectBinding, error) {
	raw, resp, err := c.client.CreateProjectBinding(ctx, integration.GenerateJiraProjectBindingBody(projectID, &binding.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return nil, err
	}

	return integration.ParseJiraProjectBinding(raw)
}
