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

package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

const (
	// llmProviderFieldTypeHTTPHeaders is the type of the provider definition
	// field holding the custom HTTP headers. Every other field type holds a
	// string.
	llmProviderFieldTypeHTTPHeaders = "HTTP_HEADERS"

	// LLMProviderConnectionDetailConfigurationPrefix prefixes the connection
	// detail keys of the secret configuration fields last applied.
	LLMProviderConnectionDetailConfigurationPrefix = "configuration."
	// LLMProviderConnectionDetailHTTPHeaderPrefix prefixes the connection
	// detail keys of the secret HTTP header values last applied.
	LLMProviderConnectionDetailHTTPHeaderPrefix = "httpHeader."
)

// LLMConnectivityClient is the interface for interacting with the SonarQube
// V2 LLM connectivity API (api/v2/llm-connectivity).
type LLMConnectivityClient interface {
	GetProviderDefinitions(ctx context.Context) (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error)
	ListProviders(ctx context.Context) (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error)
	GetProviderMappings(ctx context.Context, opt *sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error)
	CreateProvider(ctx context.Context, opt *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error)
	UpdateProvider(ctx context.Context, providerID string, opt *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error)
	DeleteProvider(ctx context.Context, providerID string) (*http.Response, error)
	UpsertProviderMapping(ctx context.Context, opt *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error)
}

// NewLLMConnectivityClient creates a new LLMConnectivityClient using the
// provided SonarQube client configuration. When the observe cache is
// enabled, the client caches the provider definitions, providers and
// mappings in cache.Default().
func NewLLMConnectivityClient(clientConfig common.Config) LLMConnectivityClient {
	return NewCachedLLMConnectivityClient(common.NewClient(clientConfig).V2.LlmConnectivity, cache.ForConfig(clientConfig))
}

// LLMProviderSecrets holds secret values of a LLMProvider: either the values
// read from the referenced Secrets, or the values last applied to SonarQube,
// as saved in the connection secret.
type LLMProviderSecrets struct {
	// Configuration holds the secret configuration fields, by field key.
	Configuration map[string]string
	// HTTPHeaders holds the secret HTTP header values, by header name.
	HTTPHeaders map[string]string
}

// ConnectionDetails returns the connection details saving secrets, so that
// the next Observe can detect a rotation of the referenced Secrets.
func (s LLMProviderSecrets) ConnectionDetails() managed.ConnectionDetails {
	details := managed.ConnectionDetails{}

	for key, value := range s.Configuration {
		details[LLMProviderConnectionDetailConfigurationPrefix+key] = []byte(value)
	}

	for name, value := range s.HTTPHeaders {
		details[LLMProviderConnectionDetailHTTPHeaderPrefix+name] = []byte(value)
	}

	return details
}

// FindLLMProviderByID returns the provider with the given ID, or nil.
func FindLLMProviderByID(providers *sonar.LlmConnectivityLlmProvidersResponse, providerID string) *sonar.LlmConnectivityLlmProviderResponse {
	if providers == nil {
		return nil
	}

	for i := range providers.Providers {
		if providers.Providers[i].Id == providerID {
			return &providers.Providers[i]
		}
	}

	return nil
}

// FindLLMProviderDefinition returns the definition of the given provider
// type, or nil.
func FindLLMProviderDefinition(definitions *sonar.LlmConnectivityLlmProviderDefinitionsResponse, provider string) *sonar.LlmConnectivityLlmProviderDefinitionResponse {
	if definitions == nil {
		return nil
	}

	for i := range definitions.ProviderDefinitions {
		if definitions.ProviderDefinitions[i].Provider == provider {
			return &definitions.ProviderDefinitions[i]
		}
	}

	return nil
}

// FindLLMProviderMappings returns the mappings selecting the given provider,
// sorted by AI capability.
func FindLLMProviderMappings(mappings *sonar.LlmConnectivityProviderMappingsResponse, providerID string) []sonar.LlmConnectivityProviderMappingResponse {
	if mappings == nil {
		return nil
	}

	var found []sonar.LlmConnectivityProviderMappingResponse

	for _, mapping := range mappings.ProviderMappings {
		if mapping.LlmProviderId == providerID {
			found = append(found, mapping)
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].AiCapability < found[j].AiCapability })

	return found
}

// llmProviderHTTPHeadersField returns the key of the definition's HTTP
// headers field, or an empty string when the provider type takes no
// headers.
func llmProviderHTTPHeadersField(definition *sonar.LlmConnectivityLlmProviderDefinitionResponse) string {
	if definition == nil {
		return ""
	}

	for _, field := range definition.Fields {
		if field.Type == llmProviderFieldTypeHTTPHeaders {
			return field.Key
		}
	}

	return ""
}

// ValidateLLMProviderSpec checks the spec against the definition of its
// provider type, so that misconfigurations surface with the supported keys
// instead of SonarQube's connection test error. It reports every unknown
// configuration key, every secret field given in plain text, every missing
// required field, and HTTP headers given to a provider type without them.
func ValidateLLMProviderSpec(spec *v1alpha1.LLMProviderParameters, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse) error {
	if definition == nil {
		return fmt.Errorf("SonarQube does not support LLM provider type %q", spec.Provider)
	}

	stringFields := map[string]sonar.LlmConnectivityLlmProviderFieldResponse{}

	for _, field := range definition.Fields {
		if field.Type != llmProviderFieldTypeHTTPHeaders {
			stringFields[field.Key] = field
		}
	}

	problems := unknownLLMProviderConfigurationKeys(spec, stringFields)
	problems = append(problems, missingLLMProviderFields(spec, definition)...)

	if len(spec.HTTPHeaders) > 0 && llmProviderHTTPHeadersField(definition) == "" {
		problems = append(problems, fmt.Sprintf("LLM provider type %q does not accept httpHeaders", spec.Provider))
	}

	if len(problems) == 0 {
		return nil
	}

	sort.Strings(problems)

	keys := make([]string, 0, len(stringFields))
	for key := range stringFields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return fmt.Errorf("invalid configuration for LLM provider type %q (supported keys: %s): %s",
		spec.Provider, strings.Join(keys, ", "), strings.Join(problems, "; "))
}

// unknownLLMProviderConfigurationKeys reports the configuration keys that
// are not string fields of the definition, and the secret fields given in
// plain text.
func unknownLLMProviderConfigurationKeys(spec *v1alpha1.LLMProviderParameters, stringFields map[string]sonar.LlmConnectivityLlmProviderFieldResponse) []string {
	var problems []string

	for key := range spec.Configuration {
		field, known := stringFields[key]

		switch {
		case !known:
			problems = append(problems, fmt.Sprintf("unknown configuration key %q", key))
		case field.Secret:
			problems = append(problems, fmt.Sprintf("configuration key %q is secret and must be set in configurationSecretRefs", key))
		}
	}

	for key := range spec.ConfigurationSecretRefs {
		if _, known := stringFields[key]; !known {
			problems = append(problems, fmt.Sprintf("unknown configurationSecretRefs key %q", key))
		}
	}

	return problems
}

// missingLLMProviderFields reports the required fields of the definition
// that the spec does not set.
func missingLLMProviderFields(spec *v1alpha1.LLMProviderParameters, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse) []string {
	var problems []string

	for _, field := range definition.Fields {
		if !field.Required {
			continue
		}

		if field.Type == llmProviderFieldTypeHTTPHeaders {
			if len(spec.HTTPHeaders) == 0 {
				problems = append(problems, fmt.Sprintf("required field %q is missing: set httpHeaders", field.Key))
			}

			continue
		}

		_, inConfiguration := spec.Configuration[field.Key]
		_, inSecretRefs := spec.ConfigurationSecretRefs[field.Key]

		if !inConfiguration && !inSecretRefs {
			problems = append(problems, fmt.Sprintf("required configuration key %q is missing", field.Key))
		}
	}

	return problems
}

// generateLLMProviderConfiguration builds the configuration sent to
// SonarQube: the plain configuration, the secret configuration, and the
// HTTP headers under the definition's headers field. The headers are sent
// even when empty if withEmptyHeaders is set, so that an update removes the
// headers dropped from the spec.
func generateLLMProviderConfiguration(spec *v1alpha1.LLMProviderParameters, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse, secrets LLMProviderSecrets, withEmptyHeaders bool) map[string]any {
	configuration := map[string]any{}

	for key, value := range spec.Configuration {
		configuration[key] = value
	}

	for key := range spec.ConfigurationSecretRefs {
		configuration[key] = secrets.Configuration[key]
	}

	headersField := llmProviderHTTPHeadersField(definition)
	if headersField == "" || (len(spec.HTTPHeaders) == 0 && !withEmptyHeaders) {
		return configuration
	}

	headers := make([]map[string]any, 0, len(spec.HTTPHeaders))

	for _, header := range spec.HTTPHeaders {
		secret := header.ValueSecretRef != nil

		value := ptr.Deref(header.Value, "")
		if secret {
			value = secrets.HTTPHeaders[header.Name]
		}

		headers = append(headers, map[string]any{"name": header.Name, "value": value, "secret": secret})
	}

	configuration[headersField] = headers

	return configuration
}

// GenerateLLMProviderCreateOptions generates the options for creating a
// LLMProvider in SonarQube.
func GenerateLLMProviderCreateOptions(spec *v1alpha1.LLMProviderParameters, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse, secrets LLMProviderSecrets) *sonar.LlmConnectivityCreateProviderOptions {
	return &sonar.LlmConnectivityCreateProviderOptions{
		Provider:      spec.Provider,
		Label:         spec.Label,
		Configuration: generateLLMProviderConfiguration(spec, definition, secrets, false),
	}
}

// GenerateLLMProviderUpdateOptions generates the options for updating a
// LLMProvider in SonarQube. The whole configuration is sent, secrets
// included, as SonarQube re-tests the connection on every update.
func GenerateLLMProviderUpdateOptions(spec *v1alpha1.LLMProviderParameters, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse, secrets LLMProviderSecrets) *sonar.LlmConnectivityUpdateProviderOptions {
	return &sonar.LlmConnectivityUpdateProviderOptions{
		Label:         new(spec.Label),
		Configuration: generateLLMProviderConfiguration(spec, definition, secrets, true),
	}
}

// GenerateLLMProviderMappingUpsertOptions generates the options mapping an
// AI capability to the given provider.
func GenerateLLMProviderMappingUpsertOptions(providerID string, mapping *v1alpha1.LLMProviderModelMapping) *sonar.LlmConnectivityUpsertProviderMappingOptions {
	return &sonar.LlmConnectivityUpsertProviderMappingOptions{
		AiCapability:             mapping.AICapability,
		LlmProviderId:            providerID,
		ModelIdentifier:          mapping.ModelIdentifier,
		SecondaryModelIdentifier: ptr.Deref(mapping.SecondaryModelIdentifier, ""),
	}
}

// GenerateLLMProviderObservation generates the observation of a provider and
// of the mappings selecting it. Configuration fields are only reported when
// the definition marks them as non-secret, and secret header values are
// never reported, so that no credential reaches the status even if
// SonarQube returned it unmasked.
func GenerateLLMProviderObservation(provider *sonar.LlmConnectivityLlmProviderResponse, definition *sonar.LlmConnectivityLlmProviderDefinitionResponse, mappings []sonar.LlmConnectivityProviderMappingResponse) v1alpha1.LLMProviderObservation {
	if provider == nil {
		return v1alpha1.LLMProviderObservation{}
	}

	observation := v1alpha1.LLMProviderObservation{
		ID:       provider.Id,
		Provider: provider.Provider,
		Label:    provider.Label,
	}

	if definition != nil {
		for _, field := range definition.Fields {
			value, set := provider.Configuration[field.Key]
			if !set || value == nil {
				continue
			}

			switch {
			case field.Type == llmProviderFieldTypeHTTPHeaders:
				observation.HTTPHeaders = generateLLMProviderHTTPHeadersObservation(value)
			case !field.Secret:
				if observation.Configuration == nil {
					observation.Configuration = map[string]string{}
				}

				observation.Configuration[field.Key] = stringifyLLMProviderValue(value)
			}
		}
	}

	for _, mapping := range mappings {
		observation.ModelMappings = append(observation.ModelMappings, v1alpha1.LLMProviderModelMappingObservation{
			AICapability:             mapping.AiCapability,
			ModelIdentifier:          mapping.ModelIdentifier,
			SecondaryModelIdentifier: mapping.SecondaryModelIdentifier,
		})
	}

	return observation
}

// generateLLMProviderHTTPHeadersObservation parses the observed value of
// the HTTP headers field, a list of {name, value, secret} objects.
func generateLLMProviderHTTPHeadersObservation(value any) []v1alpha1.LLMProviderHTTPHeaderObservation {
	items, isList := value.([]any)
	if !isList {
		return nil
	}

	headers := make([]v1alpha1.LLMProviderHTTPHeaderObservation, 0, len(items))

	for _, item := range items {
		fields, isObject := item.(map[string]any)
		if !isObject {
			continue
		}

		header := v1alpha1.LLMProviderHTTPHeaderObservation{}
		header.Name, _ = fields["name"].(string)
		header.Secret, _ = fields["secret"].(bool)

		if !header.Secret {
			header.Value = stringifyLLMProviderValue(fields["value"])
		}

		headers = append(headers, header)
	}

	return headers
}

// stringifyLLMProviderValue returns the string form of an observed
// configuration value.
func stringifyLLMProviderValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

// IsLLMProviderConfigurationUpToDate reports whether the provider itself
// (label and configuration, not its mappings) matches the spec. current
// holds the secret values read from the referenced Secrets and saved the
// values last applied: SonarQube does not return secrets, so a difference
// between them is the only way to detect a rotation. Configuration keys set
// in SonarQube but absent from the spec are ignored.
func IsLLMProviderConfigurationUpToDate(spec *v1alpha1.LLMProviderParameters, observation *v1alpha1.LLMProviderObservation, current, saved LLMProviderSecrets) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Label != observation.Label {
		return false
	}

	for key, value := range spec.Configuration {
		observed, set := observation.Configuration[key]
		if !set || observed != value {
			return false
		}
	}

	for key := range spec.ConfigurationSecretRefs {
		if current.Configuration[key] != saved.Configuration[key] {
			return false
		}
	}

	return areLLMProviderHTTPHeadersUpToDate(spec.HTTPHeaders, observation.HTTPHeaders, current, saved)
}

// areLLMProviderHTTPHeadersUpToDate reports whether the observed headers
// are exactly the desired ones.
func areLLMProviderHTTPHeadersUpToDate(desired []v1alpha1.LLMProviderHTTPHeader, observed []v1alpha1.LLMProviderHTTPHeaderObservation, current, saved LLMProviderSecrets) bool {
	if len(desired) != len(observed) {
		return false
	}

	observedByName := make(map[string]v1alpha1.LLMProviderHTTPHeaderObservation, len(observed))
	for _, header := range observed {
		observedByName[header.Name] = header
	}

	for _, header := range desired {
		got, found := observedByName[header.Name]
		if !found {
			return false
		}

		if header.ValueSecretRef == nil {
			if got.Secret || got.Value != ptr.Deref(header.Value, "") {
				return false
			}

			continue
		}

		if !got.Secret || current.HTTPHeaders[header.Name] != saved.HTTPHeaders[header.Name] {
			return false
		}
	}

	return true
}

// OutdatedLLMProviderModelMappings returns the desired mappings that are not
// observed on the provider: missing, mapped to another provider, or using
// other models.
func OutdatedLLMProviderModelMappings(spec *v1alpha1.LLMProviderParameters, observation *v1alpha1.LLMProviderObservation) []v1alpha1.LLMProviderModelMapping {
	if spec == nil {
		return nil
	}

	var outdated []v1alpha1.LLMProviderModelMapping

	for _, mapping := range spec.ModelMappings {
		upToDate := observation != nil && slices.ContainsFunc(observation.ModelMappings, func(observed v1alpha1.LLMProviderModelMappingObservation) bool {
			return observed.AICapability == mapping.AICapability &&
				observed.ModelIdentifier == mapping.ModelIdentifier &&
				observed.SecondaryModelIdentifier == ptr.Deref(mapping.SecondaryModelIdentifier, "")
		})

		if !upToDate {
			outdated = append(outdated, mapping)
		}
	}

	return outdated
}

// IsLLMProviderUpToDate reports whether the provider, its type and its
// mappings match the spec. See IsLLMProviderConfigurationUpToDate for
// current and saved.
func IsLLMProviderUpToDate(spec *v1alpha1.LLMProviderParameters, observation *v1alpha1.LLMProviderObservation, current, saved LLMProviderSecrets) bool {
	if spec == nil {
		return true
	}

	if observation == nil || spec.Provider != observation.Provider {
		return false
	}

	return IsLLMProviderConfigurationUpToDate(spec, observation, current, saved) &&
		len(OutdatedLLMProviderModelMappings(spec, observation)) == 0
}

// ErrLLMProviderMapped is returned when deleting a provider still selected
// by an AI capability, which SonarQube refuses.
var ErrLLMProviderMapped = errors.New("the LLM provider is still mapped to AI capabilities; map them to another LLMProvider before deleting it")

// CheckLLMProviderDeletable returns ErrLLMProviderMapped, naming the
// capabilities, when some mapping still selects the provider.
func CheckLLMProviderDeletable(mappings *sonar.LlmConnectivityProviderMappingsResponse, providerID string) error {
	found := FindLLMProviderMappings(mappings, providerID)
	if len(found) == 0 {
		return nil
	}

	capabilities := make([]string, 0, len(found))
	for _, mapping := range found {
		capabilities = append(capabilities, mapping.AiCapability)
	}

	return fmt.Errorf("%w: %s", ErrLLMProviderMapped, strings.Join(capabilities, ", "))
}
