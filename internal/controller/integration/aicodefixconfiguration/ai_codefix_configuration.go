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

// Package aicodefixconfiguration provides a controller for
// AICodeFixConfiguration resources.
package aicodefixconfiguration

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

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
	// errGetEnablement indicates the AI CodeFix enablement could not be
	// retrieved.
	errGetEnablement = "cannot get AI CodeFix enablement"
	// errSetEnablement indicates the AI CodeFix enablement could not be set.
	errSetEnablement = "cannot set AI CodeFix enablement"
	// errDisable indicates AI CodeFix could not be disabled on deletion.
	errDisable = "cannot disable AI CodeFix"
)

// SetupGated adds a controller that reconciles AICodeFixConfiguration
// managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup AICodeFixConfiguration controller"))
		}
	}, v1alpha1.AICodeFixConfigurationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles AICodeFixConfiguration managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.AICodeFixConfigurationGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.AICodeFixConfiguration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewFixSuggestionsClient,
		}),
		managed.WithDeterministicExternalName(true),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.AICodeFixConfigurationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.AICodeFixConfigurationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.AICodeFixConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.FixSuggestionsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, configuration *v1alpha1.AICodeFixConfiguration) (managed.TypedExternalClient[*v1alpha1.AICodeFixConfiguration], error) {
	err := c.usage.Track(ctx, configuration)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, configuration)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for
// AICodeFixConfiguration resources.
type external struct {
	client integration.FixSuggestionsClient
}

// Observe observes the instance-wide AI CodeFix enablement.
//
// The enablement always exists, so the resource is reported as existing,
// and drift is fixed by Update. While the resource is being deleted, it is
// reported as existing until AI CodeFix is disabled, so that Delete runs
// once and the finalizer is then removed.
func (c *external) Observe(ctx context.Context, configuration *v1alpha1.AICodeFixConfiguration) (managed.ExternalObservation, error) {
	enablement, resp, err := c.client.GetEnablement(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetEnablement)
	}

	configuration.Status.AtProvider = integration.GenerateAICodeFixConfigurationObservation(enablement)

	if meta.WasDeleted(configuration) {
		return managed.ExternalObservation{
			ResourceExists: configuration.Status.AtProvider.Enablement != v1alpha1.AICodeFixDisabled,
		}, nil
	}

	configuration.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: integration.IsAICodeFixConfigurationUpToDate(&configuration.Spec.ForProvider, &configuration.Status.AtProvider),
	}, nil
}

// Create sets the AI CodeFix enablement. Observe always reports the
// enablement as existing, so Create only runs if that ever changes.
func (c *external) Create(ctx context.Context, configuration *v1alpha1.AICodeFixConfiguration) (managed.ExternalCreation, error) {
	configuration.SetConditions(xpv1.Creating())

	err := c.setEnablement(ctx, integration.GenerateAICodeFixEnablementOptions(&configuration.Spec.ForProvider), errSetEnablement)

	return managed.ExternalCreation{}, err
}

// Update sets the AI CodeFix enablement to match the spec.
func (c *external) Update(ctx context.Context, configuration *v1alpha1.AICodeFixConfiguration) (managed.ExternalUpdate, error) {
	err := c.setEnablement(ctx, integration.GenerateAICodeFixEnablementOptions(&configuration.Spec.ForProvider), errSetEnablement)

	return managed.ExternalUpdate{}, err
}

// Delete disables AI CodeFix, its default state.
func (c *external) Delete(ctx context.Context, configuration *v1alpha1.AICodeFixConfiguration) (managed.ExternalDelete, error) {
	configuration.SetConditions(xpv1.Deleting())

	err := c.setEnablement(ctx, &sonar.FixSuggestionsSetEnablementOptions{Enablement: v1alpha1.AICodeFixDisabled}, errDisable)

	return managed.ExternalDelete{}, err
}

// Disconnect is a no-op for the AICodeFixConfiguration controller.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// setEnablement sends the enablement, wrapping a failure with errMessage.
func (c *external) setEnablement(ctx context.Context, opts *sonar.FixSuggestionsSetEnablementOptions, errMessage string) error {
	resp, err := c.client.SetEnablement(ctx, opts) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return errors.Wrap(err, errMessage)
	}

	return nil
}
