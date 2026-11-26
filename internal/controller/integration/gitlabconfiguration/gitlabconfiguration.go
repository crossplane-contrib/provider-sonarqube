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

// Package gitlabconfiguration provides a controller for GitLabConfiguration
// resources.
package gitlabconfiguration

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
	// errGetConfiguration indicates the GitLab configuration get call failed.
	errGetConfiguration = "cannot get GitLab configuration"
	// errSearchConfigurations indicates the GitLab configuration search call
	// failed.
	errSearchConfigurations = "cannot search GitLab configurations"
	// errCreateConfiguration indicates the GitLab configuration create call
	// failed.
	errCreateConfiguration = "cannot create GitLab configuration"
	// errUpdateConfiguration indicates the GitLab configuration update call
	// failed.
	errUpdateConfiguration = "cannot update GitLab configuration"
	// errDeleteConfiguration indicates the GitLab configuration delete call
	// failed.
	errDeleteConfiguration = "cannot delete GitLab configuration"
	// errNoConfigurationID indicates the create API response has no ID.
	errNoConfigurationID = "API response missing GitLab configuration id"
	// errGetApplicationSecret indicates the application secret could not be
	// read.
	errGetApplicationSecret = "cannot get GitLab application secret"
	// errEmptyApplicationSecret indicates the application secret is empty.
	errEmptyApplicationSecret = "GitLab application secret is empty"
	// errGetProvisioningToken indicates the provisioning token could not be
	// read.
	errGetProvisioningToken = "cannot get GitLab provisioning token"
	// errGetSavedCredentials indicates the last applied credentials could not
	// be read.
	errGetSavedCredentials = "cannot get saved GitLab configuration credentials"

	// errConfigurationExists indicates SonarQube already has a GitLab
	// configuration, which it supports only one of.
	errConfigurationExists = "SonarQube already has a GitLab configuration (id %q) and supports only one: " +
		"set the crossplane.io/external-name annotation to this id to manage it"
	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for GitLabConfiguration resource %s"

	// connectionDetailApplicationSecretKey is the connection secret key
	// under which the application secret last applied is stored.
	connectionDetailApplicationSecretKey = "applicationSecret"
	// connectionDetailProvisioningTokenKey is the connection secret key
	// under which the provisioning token last applied is stored.
	connectionDetailProvisioningTokenKey = "provisioningToken"
)

// SetupGated adds a controller that reconciles GitLabConfiguration managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup GitLabConfiguration controller"))
		}
	}, v1alpha1.GitLabConfigurationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles GitLabConfiguration managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.GitLabConfigurationGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like the configuration ID would
		// take over an existing configuration.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.GitLabConfiguration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewGitLabConfigurationsClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.GitLabConfigurationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.GitLabConfigurationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.GitLabConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.GitLabConfigurationsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (managed.TypedExternalClient[*v1alpha1.GitLabConfiguration], error) {
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

// external implements the ExternalClient interface for GitLabConfiguration
// resources.
type external struct {
	kubeClient client.Client
	client     integration.GitLabConfigurationsClient
}

// Observe checks whether the GitLab configuration exists in SonarQube and
// whether it is up to date.
//
// The external name is the configuration ID assigned by SonarQube. It is
// empty until the resource is created or imported.
func (c *external) Observe(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, resp, err := c.client.GetGitlabConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		// Only a 404 means the configuration is missing. Reporting any other
		// failure as missing would release the finalizer of a resource being
		// deleted, leaving the configuration behind in SonarQube.
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetConfiguration)
	}

	configuration.Status.AtProvider = integration.GenerateGitLabConfigurationObservation(observed)

	// The credentials are not needed to delete the configuration, and their
	// Secret may already be gone, which must not block the deletion.
	if meta.WasDeleted(configuration) {
		return managed.ExternalObservation{ResourceExists: true}, nil
	}

	current, err := c.readCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, err
	}

	saved, err := c.getSavedCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, err
	}

	former := configuration.Spec.ForProvider.DeepCopy()
	integration.LateInitializeGitLabConfiguration(&configuration.Spec.ForProvider, &configuration.Status.AtProvider)

	// SonarQube validates an enabled configuration against GitLab when it is
	// read, and reports why it cannot be used.
	if message := configuration.Status.AtProvider.ErrorMessage; message != "" {
		configuration.SetConditions(xpv1.Unavailable().WithMessage(message))
	} else {
		configuration.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsGitLabConfigurationUpToDate(&configuration.Spec.ForProvider, &configuration.Status.AtProvider, current, saved),
		ResourceLateInitialized: integration.IsGitLabConfigurationLateInitialized(former, &configuration.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the GitLab configuration in SonarQube and sets the external
// name to the ID assigned by SonarQube.
//
// SonarQube supports a single GitLab configuration. If one already exists,
// Create fails and asks to set the external name to its ID instead of taking
// it over, so that deleting the resource cannot delete a configuration that
// Crossplane did not create.
func (c *external) Create(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (managed.ExternalCreation, error) {
	configuration.SetConditions(xpv1.Creating())

	credentials, err := c.readCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	existing, resp, err := c.client.SearchGitlabConfigurations(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSearchConfigurations)
	}

	if existing != nil && len(existing.GitlabConfigurations) > 0 {
		return managed.ExternalCreation{}, errors.Errorf(errConfigurationExists, existing.GitlabConfigurations[0].Id)
	}

	createOptions := integration.GenerateGitLabConfigurationCreateOptions(&configuration.Spec.ForProvider, credentials)

	created, resp, err := c.client.CreateGitlabConfiguration(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(helpers.RedactURLError(err), errCreateConfiguration)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoConfigurationID), errCreateConfiguration)
	}

	meta.SetExternalName(configuration, created.Id)

	return managed.ExternalCreation{ConnectionDetails: connectionDetails(credentials)}, nil
}

// Update updates the GitLab configuration in SonarQube to match the desired
// state, including the application secret and provisioning token.
func (c *external) Update(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, configuration.Name)
	}

	credentials, err := c.readCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateGitLabConfigurationUpdateOptions(&configuration.Spec.ForProvider, credentials)

	_, resp, err := c.client.UpdateGitlabConfiguration(ctx, externalName, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(helpers.RedactURLError(err), errUpdateConfiguration)
	}

	return managed.ExternalUpdate{ConnectionDetails: connectionDetails(credentials)}, nil
}

// Delete removes the GitLab configuration from SonarQube.
func (c *external) Delete(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (managed.ExternalDelete, error) {
	configuration.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteGitlabConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
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

// readCredentials reads the application secret and, when referenced, the
// provisioning token from the spec's secret references.
func (c *external) readCredentials(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (integration.GitLabConfigurationCredentials, error) {
	forProvider := &configuration.Spec.ForProvider

	applicationSecret, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, configuration, forProvider.ApplicationSecretRef)
	if err != nil {
		return integration.GitLabConfigurationCredentials{}, errors.Wrap(err, errGetApplicationSecret)
	}

	credentials := integration.GitLabConfigurationCredentials{ApplicationSecret: ptr.Deref(applicationSecret, "")}
	if credentials.ApplicationSecret == "" {
		return integration.GitLabConfigurationCredentials{}, errors.New(errEmptyApplicationSecret)
	}

	if forProvider.ProvisioningTokenSecretRef != nil {
		provisioningToken, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, configuration, forProvider.ProvisioningTokenSecretRef)
		if err != nil {
			return integration.GitLabConfigurationCredentials{}, errors.Wrap(err, errGetProvisioningToken)
		}

		credentials.ProvisioningToken = ptr.Deref(provisioningToken, "")
	}

	return credentials, nil
}

// getSavedCredentials reads the credentials last applied to SonarQube from
// the connection secret. A missing connection secret or key reads as an
// empty value, not an error.
func (c *external) getSavedCredentials(ctx context.Context, configuration *v1alpha1.GitLabConfiguration) (integration.GitLabConfigurationCredentials, error) {
	applicationSecret, err := c.getSavedValue(ctx, configuration, connectionDetailApplicationSecretKey)
	if err != nil {
		return integration.GitLabConfigurationCredentials{}, err
	}

	provisioningToken, err := c.getSavedValue(ctx, configuration, connectionDetailProvisioningTokenKey)
	if err != nil {
		return integration.GitLabConfigurationCredentials{}, err
	}

	return integration.GitLabConfigurationCredentials{ApplicationSecret: applicationSecret, ProvisioningToken: provisioningToken}, nil
}

// getSavedValue reads one key of the connection secret referenced in
// writeConnectionSecretToRef.
func (c *external) getSavedValue(ctx context.Context, configuration *v1alpha1.GitLabConfiguration, key string) (string, error) {
	ref := configuration.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	value, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, configuration, ref, key)
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
func connectionDetails(credentials integration.GitLabConfigurationCredentials) managed.ConnectionDetails {
	details := managed.ConnectionDetails{
		connectionDetailApplicationSecretKey: []byte(credentials.ApplicationSecret),
	}

	if credentials.ProvisioningToken != "" {
		details[connectionDetailProvisioningTokenKey] = []byte(credentials.ProvisioningToken)
	}

	return details
}
