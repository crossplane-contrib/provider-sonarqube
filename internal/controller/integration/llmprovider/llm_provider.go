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

// Package llmprovider provides a controller for LLMProvider resources.
package llmprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
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
	// errListProviders indicates the LLM providers list API call failed.
	errListProviders = "cannot list LLM providers"
	// errGetDefinitions indicates the LLM provider definitions API call
	// failed.
	errGetDefinitions = "cannot get LLM provider definitions"
	// errGetMappings indicates the LLM provider mappings API call failed.
	errGetMappings = "cannot get LLM provider mappings"
	// errCreateProvider indicates the LLM provider create API call failed.
	errCreateProvider = "cannot create LLM provider"
	// errUpdateProvider indicates the LLM provider update API call failed.
	errUpdateProvider = "cannot update LLM provider"
	// errUpsertMapping indicates the LLM provider mapping API call failed.
	errUpsertMapping = "cannot map AI capability %s to LLM provider"
	// errDeleteProvider indicates the LLM provider delete API call failed.
	errDeleteProvider = "cannot delete LLM provider"
	// errGetSavedSecrets indicates the connection secret could not be read.
	errGetSavedSecrets = "cannot get saved secrets"

	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for LLMProvider resource %s"
	// errProviderTypeChanged indicates the observed provider type differs
	// from the spec, which SonarQube cannot update.
	errProviderTypeChanged = "LLM provider %s has type %s in SonarQube, not %s: the type cannot be changed, recreate the LLMProvider"
)

// SetupGated adds a controller that reconciles LLMProvider managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup LLMProvider controller"))
		}
	}, v1alpha1.LLMProviderGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles LLMProvider managed resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.LLMProviderGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.LLMProvider](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewLLMConnectivityClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.LLMProviderList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.LLMProviderGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.LLMProvider{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.LLMConnectivityClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (managed.TypedExternalClient[*v1alpha1.LLMProvider], error) {
	err := c.usage.Track(ctx, llmProvider)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, llmProvider)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{kubeClient: c.kube, client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for LLMProvider
// resources.
type external struct {
	kubeClient client.Client
	client     integration.LLMConnectivityClient
}

// Observe checks whether the LLMProvider exists in SonarQube and whether it,
// and the AI capabilities it should serve, are up to date.
func (c *external) Observe(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(llmProvider)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	providers, resp, err := c.client.ListProviders(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errListProviders)
	}

	found := integration.FindLLMProviderByID(providers, externalName)
	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Deletion only needs to know the provider still exists: skip the
	// secrets, which may be deleted along with the LLMProvider.
	if meta.WasDeleted(llmProvider) {
		return managed.ExternalObservation{ResourceExists: true}, nil
	}

	current, err := c.readSecrets(ctx, llmProvider)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	saved, err := c.getSavedSecrets(ctx, llmProvider)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	definition, err := c.getDefinition(ctx, found.Provider)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	mappings, err := c.getMappings(ctx)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	llmProvider.Status.AtProvider = integration.GenerateLLMProviderObservation(found, definition, integration.FindLLMProviderMappings(mappings, externalName))
	llmProvider.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  integration.IsLLMProviderUpToDate(&llmProvider.Spec.ForProvider, &llmProvider.Status.AtProvider, current, saved),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create creates the LLMProvider in SonarQube, which tests the connection
// first, and sets the external name to the ID assigned by the API. The AI
// capabilities are mapped by the Update following the next Observe.
func (c *external) Create(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (managed.ExternalCreation, error) {
	llmProvider.SetConditions(xpv1.Creating())

	secrets, err := c.readSecrets(ctx, llmProvider)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	spec := &llmProvider.Spec.ForProvider

	definition, err := c.getDefinition(ctx, spec.Provider)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = integration.ValidateLLMProviderSpec(spec, definition)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	created, resp, err := c.client.CreateProvider(ctx, integration.GenerateLLMProviderCreateOptions(spec, definition, secrets)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(helpers.RedactURLError(err), errCreateProvider)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.New("API response missing LLM provider id")
	}

	meta.SetExternalName(llmProvider, created.Id)

	return managed.ExternalCreation{ConnectionDetails: secrets.ConnectionDetails()}, nil
}

// Update updates the LLMProvider in SonarQube when its label or
// configuration drifted, then maps the AI capabilities that are not mapped
// to it as desired.
func (c *external) Update(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(llmProvider)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, llmProvider.Name)
	}

	spec := &llmProvider.Spec.ForProvider
	observation := &llmProvider.Status.AtProvider

	if observation.Provider != "" && observation.Provider != spec.Provider {
		return managed.ExternalUpdate{}, fmt.Errorf(errProviderTypeChanged, externalName, observation.Provider, spec.Provider)
	}

	secrets, err := c.readSecrets(ctx, llmProvider)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	saved, err := c.getSavedSecrets(ctx, llmProvider)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if !integration.IsLLMProviderConfigurationUpToDate(spec, observation, secrets, saved) {
		err = c.updateProvider(ctx, externalName, spec, secrets)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	for _, mapping := range integration.OutdatedLLMProviderModelMappings(spec, observation) {
		_, resp, err := c.client.UpsertProviderMapping(ctx, integration.GenerateLLMProviderMappingUpsertOptions(externalName, &mapping)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrapf(err, errUpsertMapping, mapping.AICapability)
		}
	}

	return managed.ExternalUpdate{ConnectionDetails: secrets.ConnectionDetails()}, nil
}

// Delete removes the LLMProvider from SonarQube. SonarQube refuses to
// delete a provider mapped to an AI capability and has no way to unmap one,
// so Delete fails, naming the capabilities, until they are mapped to
// another provider.
func (c *external) Delete(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (managed.ExternalDelete, error) {
	llmProvider.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(llmProvider)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	mappings, err := c.getMappings(ctx)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	err = integration.CheckLLMProviderDeletable(mappings, externalName)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	resp, err := c.client.DeleteProvider(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteProvider)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the LLMProvider controller.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// updateProvider validates the spec, then updates the provider's label and
// configuration.
func (c *external) updateProvider(ctx context.Context, providerID string, spec *v1alpha1.LLMProviderParameters, secrets integration.LLMProviderSecrets) error {
	definition, err := c.getDefinition(ctx, spec.Provider)
	if err != nil {
		return err
	}

	err = integration.ValidateLLMProviderSpec(spec, definition)
	if err != nil {
		return err
	}

	_, resp, err := c.client.UpdateProvider(ctx, providerID, integration.GenerateLLMProviderUpdateOptions(spec, definition, secrets)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return errors.Wrap(helpers.RedactURLError(err), errUpdateProvider)
	}

	return nil
}

// getDefinition returns the definition of the given provider type, or nil
// when SonarQube does not know the type.
func (c *external) getDefinition(ctx context.Context, provider string) (*sonar.LlmConnectivityLlmProviderDefinitionResponse, error) {
	definitions, resp, err := c.client.GetProviderDefinitions(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return nil, errors.Wrap(err, errGetDefinitions)
	}

	return integration.FindLLMProviderDefinition(definitions, provider), nil
}

// getMappings returns the mappings of every AI capability.
func (c *external) getMappings(ctx context.Context) (*sonar.LlmConnectivityProviderMappingsResponse, error) {
	mappings, resp, err := c.client.GetProviderMappings(ctx, &sonar.LlmConnectivityGetProviderMappingsOptions{}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return nil, errors.Wrap(err, errGetMappings)
	}

	return mappings, nil
}

// readSecrets reads the secret configuration fields and HTTP header values
// from their Secret references. Empty values are rejected.
func (c *external) readSecrets(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (integration.LLMProviderSecrets, error) {
	spec := &llmProvider.Spec.ForProvider
	secrets := integration.LLMProviderSecrets{Configuration: map[string]string{}, HTTPHeaders: map[string]string{}}

	for key, ref := range spec.ConfigurationSecretRefs {
		value, err := c.readSecret(ctx, llmProvider, &ref)
		if err != nil {
			return integration.LLMProviderSecrets{}, errors.Wrapf(err, "cannot get configuration %q from secret reference", key)
		}

		secrets.Configuration[key] = value
	}

	for _, header := range spec.HTTPHeaders {
		if header.ValueSecretRef == nil {
			continue
		}

		value, err := c.readSecret(ctx, llmProvider, header.ValueSecretRef)
		if err != nil {
			return integration.LLMProviderSecrets{}, errors.Wrapf(err, "cannot get HTTP header %q from secret reference", header.Name)
		}

		secrets.HTTPHeaders[header.Name] = value
	}

	return secrets, nil
}

// readSecret reads one non-empty value from a Secret reference.
func (c *external) readSecret(ctx context.Context, llmProvider *v1alpha1.LLMProvider, ref *xpv1.LocalSecretKeySelector) (string, error) {
	value, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, llmProvider, ref)
	if err != nil {
		return "", err
	}

	if ptr.Deref(value, "") == "" {
		return "", errors.New("secret value is empty")
	}

	return *value, nil
}

// getSavedSecrets reads the secret values last applied to SonarQube from
// the connection secret referenced in writeConnectionSecretToRef. Missing
// values (or a missing connection secret) are returned empty, not as
// errors, so that they are reported as drift and applied again.
func (c *external) getSavedSecrets(ctx context.Context, llmProvider *v1alpha1.LLMProvider) (integration.LLMProviderSecrets, error) {
	saved := integration.LLMProviderSecrets{Configuration: map[string]string{}, HTTPHeaders: map[string]string{}}

	ref := llmProvider.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return saved, nil
	}

	spec := &llmProvider.Spec.ForProvider

	for key := range spec.ConfigurationSecretRefs {
		value, err := c.getSavedSecret(ctx, llmProvider, ref, integration.LLMProviderConnectionDetailConfigurationPrefix+key)
		if err != nil {
			return integration.LLMProviderSecrets{}, err
		}

		saved.Configuration[key] = value
	}

	for _, header := range spec.HTTPHeaders {
		if header.ValueSecretRef == nil {
			continue
		}

		value, err := c.getSavedSecret(ctx, llmProvider, ref, integration.LLMProviderConnectionDetailHTTPHeaderPrefix+header.Name)
		if err != nil {
			return integration.LLMProviderSecrets{}, err
		}

		saved.HTTPHeaders[header.Name] = value
	}

	return saved, nil
}

// getSavedSecret reads one key of the connection secret, returning an empty
// string when the secret or the key is missing.
func (c *external) getSavedSecret(ctx context.Context, llmProvider *v1alpha1.LLMProvider, ref *xpv1.LocalSecretReference, key string) (string, error) {
	value, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, llmProvider, ref, key)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, errGetSavedSecrets)
	}

	return ptr.Deref(value, ""), nil
}
