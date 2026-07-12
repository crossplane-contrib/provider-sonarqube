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

// Package almgithub provides a controller for ALMGitHub resources.
package almgithub

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
	// errNotALMGitHub indicates managed resource is not ALMGitHub.
	errNotALMGitHub = "managed resource is not a ALMGitHub custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for ALMGitHub resource %s"

	// connectionDetailClientSecretKey is the client secret key.
	connectionDetailClientSecretKey = "clientSecret"
	// connectionDetailPrivateKeyKey is the private key key.
	connectionDetailPrivateKeyKey = "privateKey"
	// connectionDetailWebhookSecretKey is the webhook secret key.
	connectionDetailWebhookSecretKey = "webhookSecret"
)

// SetupGated adds a controller that reconciles ALMGitHub managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup ALMGitHub controller"))
		}
	}, v1alpha1.ALMGitHubGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles ALMGitHub managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.ALMGitHubGroupKind)

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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.ALMGitHubList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ALMGitHubList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ALMGitHubGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ALMGitHub{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
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
	almgithub, isValid := managedResource.(*v1alpha1.ALMGitHub)
	if !isValid {
		return nil, errors.New(errNotALMGitHub)
	}

	err := c.usage.Track(ctx, almgithub)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

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
		kubeClient:     c.kube,
		settingsClient: integration.NewALMSettingsGitHubClient(*config),
	}, nil
}

// external implements the ExternalClient interface for ALMGitHub resources.
type external struct {
	kubeClient     client.Client
	settingsClient integration.ALMSettingsGitHubClient
}

// Observe observes the ALMGitHub corresponding external resource using the
// SonarQube API, and returns an ExternalObservation.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	almGitHub, ok := mg.(*v1alpha1.ALMGitHub)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotALMGitHub)
	}

	externalName := meta.GetExternalName(almGitHub)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	clientSecret, privateKey, err := c.getRequiredSecrets(ctx, almGitHub)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	webhookSecret, err := c.getWebhookSecret(ctx, almGitHub)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	savedClientSecret, savedPrivateKey, savedWebhookSecret, err := c.getSavedSecrets(ctx, almGitHub)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	almDefinitions, resp, err := c.settingsClient.ListDefinitions(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot list ALM settings definitions from SonarQube API")
	}

	definition := integration.FindGitHubALMDefinitionByKey(&almDefinitions.Github, externalName)

	if definition == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	almGitHub.Status.AtProvider = integration.GenerateALMGitHubObservation(definition)

	former := almGitHub.Spec.ForProvider.DeepCopy()
	integration.LateInitializeALMGitHub(&almGitHub.Spec.ForProvider, &almGitHub.Status.AtProvider)

	almGitHub.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsALMGitHubUpToDate(&almGitHub.Spec.ForProvider, &almGitHub.Status.AtProvider, clientSecret, savedClientSecret, privateKey, savedPrivateKey, webhookSecret, savedWebhookSecret),
		ResourceLateInitialized: integration.IsALMGitHubLateInitialized(former, &almGitHub.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the ALMGitHub external resource using the SonarQube API.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	almGitHub, ok := mg.(*v1alpha1.ALMGitHub)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotALMGitHub)
	}

	clientSecret, privateKey, err := c.getRequiredSecrets(ctx, almGitHub)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	webhookSecret, err := c.getWebhookSecret(ctx, almGitHub)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	createOptions := integration.GenerateALMGitHubCreateOptions(&almGitHub.Spec.ForProvider, clientSecret, privateKey, webhookSecret)

	resp, err := c.settingsClient.CreateGithub(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create ALMGitHub resource in SonarQube API")
	}

	meta.SetExternalName(almGitHub, createOptions.Key)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailClientSecretKey:  []byte(clientSecret),
			connectionDetailPrivateKeyKey:    []byte(privateKey),
			connectionDetailWebhookSecretKey: []byte(webhookSecret),
		},
	}, nil
}

// Update updates the ALMGitHub corresponding external resource to reflect the
// desired state of the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	almGitHub, ok := mg.(*v1alpha1.ALMGitHub)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotALMGitHub)
	}

	externalName := meta.GetExternalName(almGitHub)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, almGitHub.Name)
	}

	clientSecret, privateKey, err := c.getRequiredSecrets(ctx, almGitHub)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	webhookSecret, err := c.getWebhookSecret(ctx, almGitHub)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateALMGitHubUpdateOptions(externalName, &almGitHub.Spec.ForProvider, clientSecret, privateKey, webhookSecret)

	resp, err := c.settingsClient.UpdateGithub(ctx, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update ALMGitHub resource in SonarQube API")
	}

	if updateOptions.NewKey != "" {
		meta.SetExternalName(almGitHub, updateOptions.NewKey)

		kubeErr := c.kubeClient.Update(ctx, almGitHub)
		if kubeErr != nil {
			return managed.ExternalUpdate{}, errors.Wrap(kubeErr, "cannot update external name annotation after key change")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailClientSecretKey:  []byte(clientSecret),
			connectionDetailPrivateKeyKey:    []byte(privateKey),
			connectionDetailWebhookSecretKey: []byte(webhookSecret),
		},
	}, nil
}

// Delete deletes the ALMGitHub corresponding external resource using the
// SonarQube API.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	almGitHub, ok := mg.(*v1alpha1.ALMGitHub)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotALMGitHub)
	}

	almGitHub.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(almGitHub)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	deleteOptions := integration.GenerateALMDeleteOptions(externalName)

	resp, err := c.settingsClient.Delete(ctx, deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete ALMGitHub resource in SonarQube API")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect closes the external client connection.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// getSavedSecrets retrieves the saved GitHub App secrets from the connection
// secret referenced in the ALMGitHub spec.
// Returns empty strings (not errors) if the connection secret
// or any key is missing.
func (c *external) getSavedSecrets(ctx context.Context, almGitHub *v1alpha1.ALMGitHub) (clientSecret, privateKey, webhookSecret string, err error) {
	ref := almGitHub.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", "", "", nil
	}

	for _, pair := range []struct {
		key    string
		target *string
	}{
		{connectionDetailClientSecretKey, &clientSecret},
		{connectionDetailPrivateKeyKey, &privateKey},
		{connectionDetailWebhookSecretKey, &webhookSecret},
	} {
		val, readErr := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, almGitHub, ref, pair.key)
		if readErr != nil {
			if strings.Contains(readErr.Error(), common.ErrSecretNotFound) || strings.Contains(readErr.Error(), common.ErrSecretKeyNotFound) {
				continue
			}

			return "", "", "", errors.Wrap(readErr, "cannot get saved secrets")
		}

		*pair.target = ptr.Deref(val, "")
	}

	return clientSecret, privateKey, webhookSecret, nil
}

// getRequiredSecrets retrieves GitHub App secrets from secret references.
func (c *external) getRequiredSecrets(ctx context.Context, almGitHub *v1alpha1.ALMGitHub) (clientSecret, privateKey string, err error) {
	clientSecretVal, clientSecretErr := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitHub, almGitHub.Spec.ForProvider.ClientSecretRef)

	switch {
	case clientSecretErr != nil:
		return "", "", errors.Wrap(clientSecretErr, "cannot get client secret from secret reference")
	case clientSecretVal == nil:
		return "", "", errors.New("client secret is nil")
	case *clientSecretVal == "":
		return "", "", errors.New("client secret is empty")
	}

	privateKeyVal, privateKeyErr := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitHub, almGitHub.Spec.ForProvider.PrivateKeyRef)

	switch {
	case privateKeyErr != nil:
		return "", "", errors.Wrap(privateKeyErr, "cannot get private key from secret reference")
	case privateKeyVal == nil:
		return "", "", errors.New("private key is nil")
	case *privateKeyVal == "":
		return "", "", errors.New("private key is empty")
	}

	return *clientSecretVal, *privateKeyVal, nil
}

// getWebhookSecret retrieves the webhook secret from secret references.
func (c *external) getWebhookSecret(ctx context.Context, almGitHub *v1alpha1.ALMGitHub) (string, error) {
	if almGitHub.Spec.ForProvider.WebhookSecretRef == nil {
		return "", nil
	}

	ws, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitHub, almGitHub.Spec.ForProvider.WebhookSecretRef)
	if err != nil {
		return "", errors.Wrap(err, "cannot get webhook secret from secret reference")
	}

	return ptr.Deref(ws, ""), nil
}
