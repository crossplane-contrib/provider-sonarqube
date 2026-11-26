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

// Package emailconfiguration provides a controller for EmailConfiguration
// resources.
package emailconfiguration

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errGetConfiguration indicates the email configuration get call failed.
	errGetConfiguration = "cannot get email configuration"
	// errSearchConfigurations indicates the email configuration search call
	// failed.
	errSearchConfigurations = "cannot search email configurations"
	// errCreateConfiguration indicates the email configuration create call
	// failed.
	errCreateConfiguration = "cannot create email configuration"
	// errUpdateConfiguration indicates the email configuration update call
	// failed.
	errUpdateConfiguration = "cannot update email configuration"
	// errDeleteConfiguration indicates the email configuration delete call
	// failed.
	errDeleteConfiguration = "cannot delete email configuration"
	// errNoConfigurationID indicates the create API response has no ID.
	errNoConfigurationID = "API response missing email configuration id"
	// errGetBasicPassword indicates the SMTP password could not be read.
	errGetBasicPassword = "cannot get SMTP password"
	// errGetOAuthClientID indicates the OAuth client ID could not be read.
	errGetOAuthClientID = "cannot get OAuth client ID"
	// errGetOAuthClientSecret indicates the OAuth client secret could not be
	// read.
	errGetOAuthClientSecret = "cannot get OAuth client secret"
	// errEmptyCredential indicates a referenced credential is empty.
	errEmptyCredential = "referenced secret value is empty"
	// errGetSavedCredentials indicates the last applied credentials could not
	// be read.
	errGetSavedCredentials = "cannot get saved email configuration credentials"

	// errConfigurationExists indicates SonarQube already has an email
	// configuration, which it supports only one of.
	errConfigurationExists = "SonarQube already has an email configuration (id %q) and supports only one: " +
		"set the crossplane.io/external-name annotation to this id to manage it"
	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for EmailConfiguration resource %s"

	// connectionDetailBasicPasswordKey is the connection secret key under
	// which the SMTP password last applied is stored.
	connectionDetailBasicPasswordKey = "basicPassword"
	// connectionDetailOAuthClientIDKey is the connection secret key under
	// which the OAuth client ID last applied is stored.
	connectionDetailOAuthClientIDKey = "oauthClientId"
	// connectionDetailOAuthClientSecretKey is the connection secret key under
	// which the OAuth client secret last applied is stored.
	connectionDetailOAuthClientSecretKey = "oauthClientSecret"
)

// SetupGated adds a controller that reconciles EmailConfiguration managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup EmailConfiguration controller"))
		}
	}, v1alpha1.EmailConfigurationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles EmailConfiguration managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.EmailConfigurationGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like the configuration ID would
		// take over an existing configuration.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.EmailConfiguration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewEmailConfigurationsClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.EmailConfigurationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.EmailConfigurationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.EmailConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.EmailConfigurationsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (managed.TypedExternalClient[*v1alpha1.EmailConfiguration], error) {
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

// external implements the ExternalClient interface for EmailConfiguration
// resources.
type external struct {
	kubeClient client.Client
	client     instance.EmailConfigurationsClient
}

// Observe checks whether the email configuration exists in SonarQube and
// whether it is up to date.
//
// The external name is the configuration ID assigned by SonarQube. It is
// empty until the resource is created, unless set to import a configuration.
func (c *external) Observe(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, resp, err := c.client.GetEmailConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
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

	configuration.Status.AtProvider = instance.GenerateEmailConfigurationObservation(observed)

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
	instance.LateInitializeEmailConfiguration(&configuration.Spec.ForProvider, &configuration.Status.AtProvider)

	configuration.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsEmailConfigurationUpToDate(&configuration.Spec.ForProvider, &configuration.Status.AtProvider, current, saved),
		ResourceLateInitialized: instance.IsEmailConfigurationLateInitialized(former, &configuration.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the email configuration in SonarQube and sets the external
// name to the ID assigned by SonarQube.
//
// SonarQube supports a single email configuration. If one already exists,
// Create fails and asks to set the external name to its ID instead of taking
// it over, so that deleting the resource cannot delete a configuration that
// Crossplane did not create.
func (c *external) Create(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (managed.ExternalCreation, error) {
	configuration.SetConditions(xpv1.Creating())

	credentials, err := c.readCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	existing, resp, err := c.client.SearchEmailConfigurations(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSearchConfigurations)
	}

	if existing != nil && len(existing.EmailConfigurations) > 0 {
		return managed.ExternalCreation{}, errors.Errorf(errConfigurationExists, existing.EmailConfigurations[0].Id)
	}

	createOptions := instance.GenerateEmailConfigurationCreateOptions(&configuration.Spec.ForProvider, credentials)

	created, resp, err := c.client.CreateEmailConfiguration(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
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

// Update updates the email configuration in SonarQube to match the desired
// state, including the credentials.
func (c *external) Update(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, configuration.Name)
	}

	credentials, err := c.readCredentials(ctx, configuration)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := instance.GenerateEmailConfigurationUpdateOptions(&configuration.Spec.ForProvider, credentials)

	_, resp, err := c.client.UpdateEmailConfiguration(ctx, externalName, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(helpers.RedactURLError(err), errUpdateConfiguration)
	}

	return managed.ExternalUpdate{ConnectionDetails: connectionDetails(credentials)}, nil
}

// Delete removes the email configuration from SonarQube.
func (c *external) Delete(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (managed.ExternalDelete, error) {
	configuration.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteEmailConfiguration(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
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

// readCredentials reads the credentials of the selected authentication method
// from the spec's secret references.
func (c *external) readCredentials(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (instance.EmailConfigurationCredentials, error) {
	forProvider := &configuration.Spec.ForProvider
	credentials := instance.EmailConfigurationCredentials{}

	if forProvider.AuthMethod != v1alpha1.EmailAuthMethodOAuth {
		password, err := c.readCredential(ctx, configuration, forProvider.BasicPasswordSecretRef)
		if err != nil {
			return instance.EmailConfigurationCredentials{}, errors.Wrap(err, errGetBasicPassword)
		}

		credentials.BasicPassword = password

		return credentials, nil
	}

	clientID, err := c.readCredential(ctx, configuration, forProvider.OAuthClientIDSecretRef)
	if err != nil {
		return instance.EmailConfigurationCredentials{}, errors.Wrap(err, errGetOAuthClientID)
	}

	clientSecret, err := c.readCredential(ctx, configuration, forProvider.OAuthClientSecretSecretRef)
	if err != nil {
		return instance.EmailConfigurationCredentials{}, errors.Wrap(err, errGetOAuthClientSecret)
	}

	credentials.OAuthClientID = clientID
	credentials.OAuthClientSecret = clientSecret

	return credentials, nil
}

// readCredential reads one non-empty value from a secret in the resource's
// namespace.
func (c *external) readCredential(ctx context.Context, configuration *v1alpha1.EmailConfiguration, selector *xpv1.LocalSecretKeySelector) (string, error) {
	value, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, configuration, selector)
	if err != nil {
		return "", err
	}

	if ptr.Deref(value, "") == "" {
		return "", errors.New(errEmptyCredential)
	}

	return *value, nil
}

// getSavedCredentials reads the credentials of the selected authentication
// method last applied to SonarQube from the connection secret. A missing
// connection secret or key reads as an empty value, not an error.
func (c *external) getSavedCredentials(ctx context.Context, configuration *v1alpha1.EmailConfiguration) (instance.EmailConfigurationCredentials, error) {
	credentials := instance.EmailConfigurationCredentials{}

	targets := map[string]*string{connectionDetailBasicPasswordKey: &credentials.BasicPassword}
	if configuration.Spec.ForProvider.AuthMethod == v1alpha1.EmailAuthMethodOAuth {
		targets = map[string]*string{
			connectionDetailOAuthClientIDKey:     &credentials.OAuthClientID,
			connectionDetailOAuthClientSecretKey: &credentials.OAuthClientSecret,
		}
	}

	for key, target := range targets {
		value, err := c.getSavedValue(ctx, configuration, key)
		if err != nil {
			return instance.EmailConfigurationCredentials{}, err
		}

		*target = value
	}

	return credentials, nil
}

// getSavedValue reads one key of the connection secret referenced in
// writeConnectionSecretToRef.
func (c *external) getSavedValue(ctx context.Context, configuration *v1alpha1.EmailConfiguration, key string) (string, error) {
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
func connectionDetails(credentials instance.EmailConfigurationCredentials) managed.ConnectionDetails {
	details := managed.ConnectionDetails{}

	for key, value := range map[string]string{
		connectionDetailBasicPasswordKey:     credentials.BasicPassword,
		connectionDetailOAuthClientIDKey:     credentials.OAuthClientID,
		connectionDetailOAuthClientSecretKey: credentials.OAuthClientSecret,
	} {
		if value != "" {
			details[key] = []byte(value)
		}
	}

	return details
}
