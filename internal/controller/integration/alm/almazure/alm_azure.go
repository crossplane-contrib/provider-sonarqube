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

// Package almazure provides a controller for ALMAzure resources.
package almazure

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotALMAzure indicates managed resource is not ALMAzure.
	errNotALMAzure = "managed resource is not a ALMAzure custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for ALMAzure resource %s"

	// connectionDetailTokenKey is the token key.
	connectionDetailTokenKey = "token"
)

// SetupGated adds a controller that reconciles ALMAzure managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup ALMAzure controller"))
		}
	}, v1alpha1.ALMAzureGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles ALMAzure managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.ALMAzureGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(options.Logger.WithValues("controller", name)),
		managed.WithPollInterval(options.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor is marked as deprecated but is not yet replaced with an alternative in controller-runtime, and the APIRecorder is still required for recording events.*/)),
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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.ALMAzureList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ALMAzureList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ALMAzureGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ALMAzure{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
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
	almAzure, isValid := managedResource.(*v1alpha1.ALMAzure)
	if !isValid {
		return nil, errors.New(errNotALMAzure)
	}

	err := c.usage.Track(ctx, almAzure)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	if config == nil {
		return nil, errors.New(errGetPC)
	}

	return &external{
		kubeClient:         c.kube,
		integrationsClient: integration.NewALMIntegrationsAzureClient(*config),
		settingsClient:     integration.NewALMSettingsAzureClient(*config),
	}, nil
}

// external is used to observe and manipulate the external resource (ALMAzure)
// in SonarQube API. It implements the managed.ExternalClient interface.
type external struct {
	// kubeClient is the Kubernetes API client that can be used to get and update the managed resource. This is expected to be initialized by the connector when the external client is created.
	kubeClient client.Client
	// integration client that can be used to observe and manipulate ALMAzure resources in SonarQube API. This is expected to be initialized by the connector when the external client is created.
	integrationsClient integration.ALMIntegrationsAzureClient
	// settingsClient can be used to observe and manipulate SonarQube settings related to ALMAzure integration. This is expected to be initialized by the connector when the external client is created.
	settingsClient integration.ALMSettingsAzureClient
}

// Observe observes the ALMAzure corresponding external resource using the
// SonarQube API, and returns an ExternalObservation.
// It is used to determine if the resource exists,
// is up to date, and late initialize the spec if needed.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	almAzure, ok := mg.(*v1alpha1.ALMAzure)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotALMAzure)
	}

	// Use external name as the identifier to get the resource
	externalName := meta.GetExternalName(almAzure)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almAzure, almAzure.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case ptr.Deref(pat, "") == "":
		return managed.ExternalObservation{}, errors.New("personal access token is empty or not provided")
	}

	savedAPIToken, err := c.getSavedAPIToken(ctx, almAzure)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	almDefinitions, resp, err := c.settingsClient.ListDefinitions(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot list ALM settings definitions from SonarQube API")
	}

	definition := integration.FindAzureALMDefinitionByKey(&almDefinitions.Azure, externalName)

	if definition == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	almAzure.Status.AtProvider = integration.GenerateALMAzureObservation(definition)

	// Late initialize the ALMAzure spec with the API response to determine if there are any fields that can be defaulted.
	former := almAzure.Spec.ForProvider.DeepCopy()
	integration.LateInitializeALMAzure(&almAzure.Spec.ForProvider, &almAzure.Status.AtProvider)

	almAzure.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsALMAzureUpToDate(&almAzure.Spec.ForProvider, *pat, &almAzure.Status.AtProvider, savedAPIToken),
		ResourceLateInitialized: integration.IsALMAzureLateInitialized(former, &almAzure.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the ALMAzure external resource using the SonarQube API,
// based on the desired state of the managed resource.
// It returns an ExternalCreation which may include connection details to be
// stored in a secret.
// It also sets the external name of the managed resource to the identifier
// of the created external resource if it is not already set.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	almAzure, ok := mg.(*v1alpha1.ALMAzure)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotALMAzure)
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almAzure, almAzure.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case ptr.Deref(pat, "") == "":
		return managed.ExternalCreation{}, errors.New("personal access token is empty or not provided")
	}

	createOptions := integration.GenerateALMAzureCreateOptions(&almAzure.Spec.ForProvider, *pat)

	resp, err := c.settingsClient.CreateAzure(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create ALMAzure resource in SonarQube API")
	}

	meta.SetExternalName(almAzure, createOptions.Key)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailTokenKey: []byte(*pat),
		},
	}, nil
}

// Update updates the ALMAzure corresponding external resource to reflect
// the desired state of the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	almAzure, ok := mg.(*v1alpha1.ALMAzure)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotALMAzure)
	}

	externalName := meta.GetExternalName(almAzure)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, almAzure.Name)
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almAzure, almAzure.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case ptr.Deref(pat, "") == "":
		return managed.ExternalUpdate{}, errors.New("personal access token is empty or not provided")
	}

	updateOptions := integration.GenerateALMAzureUpdateOptions(externalName, &almAzure.Spec.ForProvider, *pat)

	resp, err := c.settingsClient.UpdateAzure(ctx, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update ALMAzure resource in SonarQube API")
	}

	if updateOptions.NewKey != "" {
		// If the Key was updated, also update the external name to the new Key to keep it in sync with the identifier of the ALMAzure resource in SonarQube API.
		meta.SetExternalName(almAzure, updateOptions.NewKey)

		kubeErr := c.kubeClient.Update(ctx, almAzure)
		if kubeErr != nil {
			return managed.ExternalUpdate{}, errors.Wrap(kubeErr, "cannot update external name annotation after key change")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailTokenKey: []byte(*pat),
		},
	}, nil
}

// Delete deletes the ALMAzure corresponding external resource using the
// SonarQube API. It returns an ExternalDelete which may include
// connection details to be stored in a secret.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	almAzure, ok := mg.(*v1alpha1.ALMAzure)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotALMAzure)
	}

	almAzure.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(almAzure)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	deleteOptions := integration.GenerateALMDeleteOptions(externalName)

	resp, err := c.settingsClient.Delete(ctx, deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete ALMAzure resource in SonarQube API")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect closes the external client connection.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// getSavedAPIToken retrieves the saved API token from the connection secret.
func (c *external) getSavedAPIToken(ctx context.Context, almAzure *v1alpha1.ALMAzure) (string, error) {
	ref := almAzure.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	pat, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, almAzure, ref, connectionDetailTokenKey)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, "cannot get saved API token")
	}

	return ptr.Deref(pat, ""), nil
}
