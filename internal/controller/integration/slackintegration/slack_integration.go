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

// Package slackintegration provides a controller for SlackIntegration
// resources.
package slackintegration

import (
	"context"
	"fmt"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
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
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errListConfigurations indicates the Slack configuration list call
	// failed.
	errListConfigurations = "cannot list Slack configurations"
	// errCreateConfiguration indicates the Slack configuration create call
	// failed.
	errCreateConfiguration = "cannot create Slack configuration"
	// errUpdateConfiguration indicates the Slack configuration update call
	// failed.
	errUpdateConfiguration = "cannot update Slack configuration"
	// errDeleteConfiguration indicates the Slack configuration delete call
	// failed.
	errDeleteConfiguration = "cannot delete Slack configuration"
	// errNoConfigurationID indicates the create API response has no ID.
	errNoConfigurationID = "API response missing Slack configuration id"
	// errGetClientSecret indicates the client secret could not be read.
	errGetClientSecret = "cannot get Slack client secret"
	// errEmptyClientSecret indicates the client secret is empty.
	errEmptyClientSecret = "Slack client secret is empty"
	// errGetSigningSecret indicates the signing secret could not be read.
	errGetSigningSecret = "cannot get Slack signing secret"
	// errEmptySigningSecret indicates the signing secret is empty.
	errEmptySigningSecret = "Slack signing secret is empty"
	// errGetSavedCredentials indicates the last applied credentials could not
	// be read.
	errGetSavedCredentials = "cannot get saved Slack configuration credentials"

	// errConfigurationExists indicates SonarQube already has a Slack
	// configuration, which it supports only one of.
	errConfigurationExists = "SonarQube already has a Slack configuration (id %q) and supports only one: " +
		"set the crossplane.io/external-name annotation to this id to manage it"
	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for SlackIntegration resource %s"

	// connectionDetailClientSecretKey is the connection secret key under
	// which the client secret last applied is stored.
	connectionDetailClientSecretKey = "clientSecret"
	// connectionDetailSigningSecretKey is the connection secret key under
	// which the signing secret last applied is stored.
	connectionDetailSigningSecretKey = "signingSecret"
)

// SetupGated adds a controller that reconciles SlackIntegration managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup SlackIntegration controller"))
		}
	}, v1alpha1.SlackIntegrationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles SlackIntegration managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.SlackIntegrationGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like the configuration ID would
		// take over an existing configuration.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.SlackIntegration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewSlackIntegrationsClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.SlackIntegrationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.SlackIntegrationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.SlackIntegration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.SlackIntegrationsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, slack *v1alpha1.SlackIntegration) (managed.TypedExternalClient[*v1alpha1.SlackIntegration], error) {
	err := c.usage.Track(ctx, slack)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, slack)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{kubeClient: c.kube, client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for SlackIntegration
// resources.
type external struct {
	kubeClient client.Client
	client     integration.SlackIntegrationsClient
}

// Observe checks whether the Slack configuration exists in SonarQube and
// whether it is up to date.
//
// The external name is the configuration ID assigned by SonarQube. It is
// empty until the resource is created or imported.
func (c *external) Observe(ctx context.Context, slack *v1alpha1.SlackIntegration) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(slack)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Any failure is an error, including a 404, which an instance without the
	// endpoint would answer: only a successful listing proves the
	// configuration is missing. Reporting it as missing otherwise would
	// release the finalizer of a resource being deleted, leaving the
	// configuration behind in SonarQube.
	configurations, err := c.listConfigurations(ctx)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	observed := integration.FindSlackIntegration(configurations, externalName)
	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	slack.Status.AtProvider = integration.GenerateSlackIntegrationObservation(observed)

	// The credentials are not needed to delete the configuration, and their
	// Secret may already be gone, which must not block the deletion.
	if meta.WasDeleted(slack) {
		return managed.ExternalObservation{ResourceExists: true}, nil
	}

	current, err := c.readCredentials(ctx, slack)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, err
	}

	saved, err := c.getSavedCredentials(ctx, slack)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, err
	}

	slack.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  integration.IsSlackIntegrationUpToDate(&slack.Spec.ForProvider, &slack.Status.AtProvider, current, saved),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create creates the Slack configuration in SonarQube and sets the external
// name to the ID assigned by SonarQube.
//
// SonarQube supports a single Slack configuration. If one already exists,
// Create fails and asks to set the external name to its ID instead of taking
// it over, so that deleting the resource cannot delete a configuration that
// Crossplane did not create.
func (c *external) Create(ctx context.Context, slack *v1alpha1.SlackIntegration) (managed.ExternalCreation, error) {
	slack.SetConditions(xpv1.Creating())

	credentials, err := c.readCredentials(ctx, slack)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	existing, err := c.listConfigurations(ctx)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if existing != nil && len(existing.IntegrationConfigurations) > 0 {
		return managed.ExternalCreation{}, errors.Errorf(errConfigurationExists, existing.IntegrationConfigurations[0].Id)
	}

	createOptions := integration.GenerateSlackIntegrationCreateOptions(&slack.Spec.ForProvider, credentials)

	created, resp, err := c.client.CreateIntegrationConfiguration(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(helpers.RedactURLError(err), errCreateConfiguration)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoConfigurationID), errCreateConfiguration)
	}

	meta.SetExternalName(slack, created.Id)

	return managed.ExternalCreation{ConnectionDetails: connectionDetails(credentials)}, nil
}

// Update updates the Slack configuration in SonarQube to match the desired
// state, including the client secret and signing secret.
func (c *external) Update(ctx context.Context, slack *v1alpha1.SlackIntegration) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(slack)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, slack.Name)
	}

	credentials, err := c.readCredentials(ctx, slack)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateSlackIntegrationUpdateOptions(&slack.Spec.ForProvider, credentials)

	_, resp, err := c.client.UpdateIntegrationConfiguration(ctx, externalName, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(helpers.RedactURLError(err), errUpdateConfiguration)
	}

	return managed.ExternalUpdate{ConnectionDetails: connectionDetails(credentials)}, nil
}

// Delete removes the Slack configuration, and its related data, from
// SonarQube.
func (c *external) Delete(ctx context.Context, slack *v1alpha1.SlackIntegration) (managed.ExternalDelete, error) {
	slack.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(slack)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteIntegrationConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		// The client resends a DELETE after a transient failure, so a 404
		// can mean an earlier attempt succeeded.
		if common.IsResponseNotFound(resp) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteConfiguration)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// listConfigurations lists the Slack configurations of SonarQube.
func (c *external) listConfigurations(ctx context.Context) (*sonar.IntegrationsIntegrationConfigurationSearch, error) {
	configurations, resp, err := c.client.ListIntegrationConfigurations(ctx, integration.GenerateSlackIntegrationListOptions()) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return nil, errors.Wrap(err, errListConfigurations)
	}

	return configurations, nil
}

// readCredentials reads the client secret and signing secret from the spec's
// secret references.
func (c *external) readCredentials(ctx context.Context, slack *v1alpha1.SlackIntegration) (integration.SlackIntegrationCredentials, error) {
	forProvider := &slack.Spec.ForProvider

	clientSecret, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, slack, forProvider.ClientSecretRef)
	if err != nil {
		return integration.SlackIntegrationCredentials{}, errors.Wrap(err, errGetClientSecret)
	}

	signingSecret, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, slack, forProvider.SigningSecretRef)
	if err != nil {
		return integration.SlackIntegrationCredentials{}, errors.Wrap(err, errGetSigningSecret)
	}

	credentials := integration.SlackIntegrationCredentials{
		ClientSecret:  ptr.Deref(clientSecret, ""),
		SigningSecret: ptr.Deref(signingSecret, ""),
	}

	if credentials.ClientSecret == "" {
		return integration.SlackIntegrationCredentials{}, errors.New(errEmptyClientSecret)
	}

	if credentials.SigningSecret == "" {
		return integration.SlackIntegrationCredentials{}, errors.New(errEmptySigningSecret)
	}

	return credentials, nil
}

// getSavedCredentials reads the credentials last applied to SonarQube from
// the connection secret. A missing connection secret or key reads as an
// empty value, not an error.
func (c *external) getSavedCredentials(ctx context.Context, slack *v1alpha1.SlackIntegration) (integration.SlackIntegrationCredentials, error) {
	clientSecret, err := c.getSavedValue(ctx, slack, connectionDetailClientSecretKey)
	if err != nil {
		return integration.SlackIntegrationCredentials{}, err
	}

	signingSecret, err := c.getSavedValue(ctx, slack, connectionDetailSigningSecretKey)
	if err != nil {
		return integration.SlackIntegrationCredentials{}, err
	}

	return integration.SlackIntegrationCredentials{ClientSecret: clientSecret, SigningSecret: signingSecret}, nil
}

// getSavedValue reads one key of the connection secret referenced in
// writeConnectionSecretToRef.
func (c *external) getSavedValue(ctx context.Context, slack *v1alpha1.SlackIntegration, key string) (string, error) {
	ref := slack.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	value, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, slack, ref, key)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, errGetSavedCredentials)
	}

	return ptr.Deref(value, ""), nil
}

// connectionDetails returns the credentials applied to SonarQube, stored in
// the connection secret to detect their rotation.
func connectionDetails(credentials integration.SlackIntegrationCredentials) managed.ConnectionDetails {
	return managed.ConnectionDetails{
		connectionDetailClientSecretKey:  []byte(credentials.ClientSecret),
		connectionDetailSigningSecretKey: []byte(credentials.SigningSecret),
	}
}
