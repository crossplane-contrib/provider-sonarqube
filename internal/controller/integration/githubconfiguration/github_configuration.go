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

// Package githubconfiguration provides a controller for GitHubConfiguration
// resources.
package githubconfiguration

import (
	"context"
	"fmt"
	"strings"

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
	// errSearchConfigurations indicates the GitHub configurations search
	// failed.
	errSearchConfigurations = "cannot search GitHub configurations"
	// errCreateConfiguration indicates the GitHub configuration creation
	// failed.
	errCreateConfiguration = "cannot create GitHub configuration"
	// errUpdateConfiguration indicates the GitHub configuration update
	// failed.
	errUpdateConfiguration = "cannot update GitHub configuration"
	// errDeleteConfiguration indicates the GitHub configuration deletion
	// failed.
	errDeleteConfiguration = "cannot delete GitHub configuration"
	// errNoConfigurationID indicates SonarQube did not return the ID of the
	// created configuration.
	errNoConfigurationID = "API response missing GitHub configuration id"
	// errGetSavedCredentials indicates the credentials last applied could
	// not be read from the connection secret.
	errGetSavedCredentials = "cannot get saved GitHub App credentials"

	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for GitHubConfiguration resource %s"

	// connectionDetailClientIDKey is the connection secret key of the client
	// ID last applied.
	connectionDetailClientIDKey = "clientId"
	// connectionDetailClientSecretKey is the connection secret key of the
	// client secret last applied.
	connectionDetailClientSecretKey = "clientSecret"
	// connectionDetailPrivateKeyKey is the connection secret key of the
	// private key last applied.
	connectionDetailPrivateKeyKey = "privateKey"
)

// SetupGated adds a controller that reconciles GitHubConfiguration managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup GitHubConfiguration controller"))
		}
	}, v1alpha1.GitHubConfigurationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles GitHubConfiguration managed
// resources.
//
// The external name is the ID SonarQube assigns to the configuration, so
// the default initializer, which would set it to the resource name, is
// disabled. As SonarQube supports a single GitHub configuration per
// instance, a resource without an external name adopts the existing one:
// a creation whose result was lost is found again, which makes the external
// name deterministic.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.GitHubConfigurationGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.GitHubConfiguration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewGitHubConfigurationsClient,
		}),
		managed.WithInitializers(),
		managed.WithDeterministicExternalName(true),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.GitHubConfigurationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.GitHubConfigurationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.GitHubConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.GitHubConfigurationsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (managed.TypedExternalClient[*v1alpha1.GitHubConfiguration], error) {
	err := c.usage.Track(ctx, configuration)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, configuration)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{kubeClient: c.kube, client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for GitHubConfiguration
// resources.
type external struct {
	kubeClient client.Client
	client     integration.GitHubConfigurationsClient
}

// Observe observes the GitHub configuration of the SonarQube instance. A
// resource without an external name adopts the existing configuration.
func (c *external) Observe(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(configuration)

	search, resp, err := c.client.SearchGithubConfigurations(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errSearchConfigurations)
	}

	if search == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	found := integration.FindGitHubConfiguration(search.GithubConfigurations, externalName)
	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	adopted := externalName == ""
	if adopted {
		meta.SetExternalName(configuration, found.Id)
	}

	configuration.Status.AtProvider = integration.GenerateGitHubConfigurationObservation(found)

	// The credentials are not needed to delete the configuration, and their
	// Secret may already be gone, which must not block the deletion.
	if meta.WasDeleted(configuration) {
		return managed.ExternalObservation{ResourceExists: true}, nil
	}

	desired, err := c.getCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	saved, err := c.getSavedCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	former := configuration.Spec.ForProvider.DeepCopy()
	integration.LateInitializeGitHubConfiguration(&configuration.Spec.ForProvider, &configuration.Status.AtProvider)

	configuration.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsGitHubConfigurationUpToDate(&configuration.Spec.ForProvider, &configuration.Status.AtProvider, desired, saved),
		ResourceLateInitialized: adopted || integration.IsGitHubConfigurationLateInitialized(former, &configuration.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the GitHub configuration in SonarQube and sets the external
// name to the ID SonarQube assigned to it.
func (c *external) Create(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (managed.ExternalCreation, error) {
	configuration.SetConditions(xpv1.Creating())

	credentials, err := c.getCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	createOptions := integration.GenerateGitHubConfigurationCreateOptions(&configuration.Spec.ForProvider, credentials)

	created, resp, err := c.client.CreateGithubConfiguration(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(helpers.RedactURLError(err), errCreateConfiguration)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoConfigurationID), errCreateConfiguration)
	}

	meta.SetExternalName(configuration, created.Id)

	return managed.ExternalCreation{ConnectionDetails: credentialsConnectionDetails(credentials)}, nil
}

// Update updates the GitHub configuration in SonarQube to match the desired
// state.
func (c *external) Update(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, configuration.Name)
	}

	credentials, err := c.getCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateGitHubConfigurationUpdateOptions(&configuration.Spec.ForProvider, credentials)

	_, resp, err := c.client.UpdateGithubConfiguration(ctx, externalName, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(helpers.RedactURLError(err), errUpdateConfiguration)
	}

	return managed.ExternalUpdate{ConnectionDetails: credentialsConnectionDetails(credentials)}, nil
}

// Delete deletes the GitHub configuration from SonarQube.
func (c *external) Delete(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (managed.ExternalDelete, error) {
	configuration.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteGithubConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
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

// credentialsConnectionDetails returns the connection details saving the
// given credentials, so that later observations can detect their changes.
func credentialsConnectionDetails(credentials integration.GitHubConfigurationCredentials) managed.ConnectionDetails {
	return managed.ConnectionDetails{
		connectionDetailClientIDKey:     []byte(credentials.ClientID),
		connectionDetailClientSecretKey: []byte(credentials.ClientSecret),
		connectionDetailPrivateKeyKey:   []byte(credentials.PrivateKey),
	}
}

// getCredentials resolves the desired GitHub App credentials from the spec
// and its secret references.
func (c *external) getCredentials(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (integration.GitHubConfigurationCredentials, error) {
	clientSecret, err := c.getRequiredSecret(ctx, configuration, configuration.Spec.ForProvider.ClientSecretRef, "client secret")
	if err != nil {
		return integration.GitHubConfigurationCredentials{}, err
	}

	privateKey, err := c.getRequiredSecret(ctx, configuration, configuration.Spec.ForProvider.PrivateKeyRef, "private key")
	if err != nil {
		return integration.GitHubConfigurationCredentials{}, err
	}

	return integration.GitHubConfigurationCredentials{
		ClientID:     configuration.Spec.ForProvider.ClientID,
		ClientSecret: clientSecret,
		PrivateKey:   privateKey,
	}, nil
}

// getRequiredSecret reads a non-empty value from the given secret reference.
func (c *external) getRequiredSecret(ctx context.Context, configuration *v1alpha1.GitHubConfiguration, ref *xpv1.LocalSecretKeySelector, name string) (string, error) {
	value, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, configuration, ref)

	switch {
	case err != nil:
		return "", errors.Wrapf(err, "cannot get %s from secret reference", name)
	case value == nil || *value == "":
		return "", errors.Errorf("%s is empty", name)
	}

	return *value, nil
}

// getSavedCredentials reads the GitHub App credentials last applied from the
// connection secret. A missing connection secret or key reads as empty.
func (c *external) getSavedCredentials(ctx context.Context, configuration *v1alpha1.GitHubConfiguration) (integration.GitHubConfigurationCredentials, error) {
	saved := integration.GitHubConfigurationCredentials{}

	ref := configuration.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return saved, nil
	}

	for _, pair := range []struct {
		key    string
		target *string
	}{
		{connectionDetailClientIDKey, &saved.ClientID},
		{connectionDetailClientSecretKey, &saved.ClientSecret},
		{connectionDetailPrivateKeyKey, &saved.PrivateKey},
	} {
		value, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, configuration, ref, pair.key)
		if err != nil {
			if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
				continue
			}

			return integration.GitHubConfigurationCredentials{}, errors.Wrap(err, errGetSavedCredentials)
		}

		*pair.target = ptr.Deref(value, "")
	}

	return saved, nil
}
