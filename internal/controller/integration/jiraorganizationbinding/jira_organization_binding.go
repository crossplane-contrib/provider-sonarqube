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

// Package jiraorganizationbinding provides a controller for
// JiraOrganizationBinding resources.
package jiraorganizationbinding

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errGetBinding indicates the organization binding retrieval failed.
	errGetBinding = "cannot get Jira organization binding"
	// errUpdateBinding indicates the organization binding update failed.
	errUpdateBinding = "cannot update Jira organization binding"
	// errDeleteBinding indicates the organization binding deletion failed.
	errDeleteBinding = "cannot delete Jira organization binding"

	// errBindingExists indicates SonarQube already has a binding, which must
	// be explicitly imported.
	errBindingExists = "SonarQube already has a Jira organization binding (id %q): " +
		"set the crossplane.io/external-name annotation to this id to manage it"
	// errCannotCreate indicates the binding does not exist and cannot be
	// created by the provider.
	errCannotCreate = "a Jira organization binding cannot be created through the API, as it requires an " +
		"Atlassian OAuth consent in a browser: connect SonarQube to Jira from its user interface, then set " +
		"the crossplane.io/external-name annotation to the id of the binding"
)

// SetupGated adds a controller that reconciles JiraOrganizationBinding managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup JiraOrganizationBinding controller"))
		}
	}, v1alpha1.JiraOrganizationBindingGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles JiraOrganizationBinding managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.JiraOrganizationBindingGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like the binding ID would take
		// over an existing binding.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.JiraOrganizationBinding](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewJiraClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.JiraOrganizationBindingList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.JiraOrganizationBindingGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.JiraOrganizationBinding{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.JiraClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (managed.TypedExternalClient[*v1alpha1.JiraOrganizationBinding], error) {
	err := c.usage.Track(ctx, binding)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, binding)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for
// JiraOrganizationBinding resources.
type external struct {
	client integration.JiraClient
}

// Observe checks whether the imported organization binding exists in
// SonarQube and whether it is up to date.
//
// The external name is the binding ID assigned by SonarQube. A binding with
// another ID is reported as missing, so it is never updated nor deleted.
func (c *external) Observe(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(binding)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, err := c.getBinding(ctx, binding)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if observed == nil || observed.ID != externalName {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	binding.Status.AtProvider = integration.GenerateJiraOrganizationBindingObservation(observed)
	lateInitialized := integration.LateInitializeJiraOrganizationBinding(&binding.Spec.ForProvider, &binding.Status.AtProvider)

	binding.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsJiraOrganizationBindingUpToDate(&binding.Spec.ForProvider, &binding.Status.AtProvider),
		ResourceLateInitialized: lateInitialized,
	}, nil
}

// Create always fails: SonarQube only establishes a binding from an Atlassian
// OAuth consent given in a browser. The error tells how to import the binding.
func (c *external) Create(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (managed.ExternalCreation, error) {
	binding.SetConditions(xpv1.Creating())

	existing, err := c.getBinding(ctx, binding)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if existing != nil {
		return managed.ExternalCreation{}, errors.Errorf(errBindingExists, existing.ID)
	}

	return managed.ExternalCreation{}, errors.New(errCannotCreate)
}

// Update updates the editable properties of the organization binding.
func (c *external) Update(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (managed.ExternalUpdate, error) {
	if binding.Spec.ForProvider.IsTokenShared == nil {
		return managed.ExternalUpdate{}, nil
	}

	_, resp, err := c.client.UpdateOrganizationBinding(ctx, integration.GenerateJiraOrganizationBindingUpdate(&binding.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBinding)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the organization binding, and every related project binding,
// from SonarQube. It only deletes the binding identified by the external name.
func (c *external) Delete(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (managed.ExternalDelete, error) {
	binding.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(binding)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	// The deletion endpoint is keyed by organization, so make sure the binding
	// it holds is still the managed one.
	existing, err := c.getBinding(ctx, binding)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	if existing == nil || existing.ID != externalName {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteOrganizationBinding(ctx, integration.GenerateJiraOrganizationBindingOptions(&binding.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
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

// getBinding returns the organization binding of SonarQube, or nil if there is
// none.
func (c *external) getBinding(ctx context.Context, binding *v1alpha1.JiraOrganizationBinding) (*integration.JiraOrganizationBinding, error) {
	raw, resp, err := c.client.GetOrganizationBinding(ctx, integration.GenerateJiraOrganizationBindingOptions(&binding.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return nil, nil //nolint:nilnil // no binding is not an error
		}

		return nil, errors.Wrap(err, errGetBinding)
	}

	observed, err := integration.ParseJiraOrganizationBinding(raw)
	if err != nil {
		return nil, errors.Wrap(err, errGetBinding)
	}

	return observed, nil
}
