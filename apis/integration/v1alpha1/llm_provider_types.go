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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// LLMProviderHTTPHeader is a custom HTTP header sent with every request
// SonarQube makes to the LLM provider. Only provider types whose definition
// has an HTTP headers field (e.g. CUSTOM_PROXY) accept headers.
//
// +kubebuilder:validation:XValidation:rule="has(self.value) != has(self.valueSecretRef)",message="exactly one of value or valueSecretRef must be set"
type LLMProviderHTTPHeader struct {
	// Name is the HTTP header name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Value is the plain-text header value. Exactly one of Value or
	// ValueSecretRef must be set.
	// +optional
	Value *string `json:"value,omitempty"`

	// ValueSecretRef references the secret key holding the header value. The
	// header is sent to SonarQube as a secret header, stored encrypted and
	// masked in API responses. The secret is read from the LLMProvider's
	// namespace. Exactly one of Value or ValueSecretRef must be set.
	// +optional
	ValueSecretRef *xpv1.LocalSecretKeySelector `json:"valueSecretRef,omitempty"`
}

// LLMProviderModelMapping selects the provider and models SonarQube uses for
// one AI capability.
//
// +kubebuilder:validation:XValidation:rule="self.aiCapability != 'HUNTER_AGENT' || has(self.secondaryModelIdentifier)",message="secondaryModelIdentifier is required for HUNTER_AGENT"
type LLMProviderModelMapping struct {
	// AICapability is the AI capability served by this provider.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=AI_CODEFIX;HUNTER_AGENT;REMEDIATION_AGENT
	AICapability string `json:"aiCapability"`

	// ModelIdentifier is the identifier of the model used for the capability,
	// as known by the provider (e.g. a deployment name for Azure OpenAI). The
	// supported models are listed by api/v2/llm-connectivity/supported-models.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	ModelIdentifier string `json:"modelIdentifier"`

	// SecondaryModelIdentifier is the identifier of the secondary model used
	// for the capability. Required for HUNTER_AGENT, which needs both models.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	SecondaryModelIdentifier *string `json:"secondaryModelIdentifier,omitempty"`
}

// LLMProviderParameters are the configurable fields of a LLMProvider.
//
// The configuration keys depend on the provider type; SonarQube lists them,
// and which ones are secret or required, in
// api/v2/llm-connectivity/llm-provider-definitions. The provider checks the
// spec against that definition before every create or update.
//
// +kubebuilder:validation:XValidation:rule="!has(self.configuration) || !has(self.configurationSecretRefs) || self.configuration.all(k, !(k in self.configurationSecretRefs))",message="a key cannot be set in both configuration and configurationSecretRefs"
type LLMProviderParameters struct {
	// Provider is the LLM provider type. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=AZURE_OPENAI;AWS_BEDROCK;AWS_BEDROCK_MANTLE;VERTEX_AI;CUSTOM_PROXY;OPENAI;ANTHROPIC
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="provider is immutable"
	Provider string `json:"provider"`

	// Label is the display name of the provider in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Label string `json:"label"`

	// Configuration holds the non-secret configuration fields of the
	// provider, keyed by field key (e.g. an endpoint URL or a region).
	// Secret fields must be set in ConfigurationSecretRefs instead.
	// Removing a key does not unset it in SonarQube.
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	Configuration map[string]string `json:"configuration,omitempty"`

	// ConfigurationSecretRefs holds the configuration fields read from
	// Kubernetes Secrets, keyed by field key (e.g. an API key). Every field
	// the provider definition marks as secret must be set here. The secrets
	// are read from the LLMProvider's namespace.
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	ConfigurationSecretRefs map[string]xpv1.LocalSecretKeySelector `json:"configurationSecretRefs,omitempty"`

	// HTTPHeaders are custom HTTP headers sent with every request to the
	// provider. Only provider types with an HTTP headers configuration field
	// (e.g. CUSTOM_PROXY) accept them.
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=64
	HTTPHeaders []LLMProviderHTTPHeader `json:"httpHeaders,omitempty"`

	// ModelMappings selects this provider, and the models to use, for AI
	// capabilities. SonarQube maps each capability to at most one provider,
	// so a capability must not be listed by several LLMProviders.
	// SonarQube cannot unmap a capability: removing it from this list leaves
	// it mapped to this provider, and the provider cannot be deleted while a
	// capability is mapped to it. Map the capability to another LLMProvider
	// first.
	// +optional
	// +listType=map
	// +listMapKey=aiCapability
	// +kubebuilder:validation:MaxItems=3
	ModelMappings []LLMProviderModelMapping `json:"modelMappings,omitempty"`
}

// LLMProviderHTTPHeaderObservation is an HTTP header as observed from the
// SonarQube API.
type LLMProviderHTTPHeaderObservation struct {
	// Name is the HTTP header name.
	Name string `json:"name,omitempty"`
	// Value is the header value. Empty for secret headers.
	Value string `json:"value,omitempty"`
	// Secret indicates whether SonarQube stores the value encrypted.
	Secret bool `json:"secret,omitempty"`
}

// LLMProviderModelMappingObservation is a model mapping of the provider as
// observed from the SonarQube API.
type LLMProviderModelMappingObservation struct {
	// AICapability is the AI capability mapped to this provider.
	AICapability string `json:"aiCapability,omitempty"`
	// ModelIdentifier is the identifier of the model used for the capability.
	ModelIdentifier string `json:"modelIdentifier,omitempty"`
	// SecondaryModelIdentifier is the identifier of the secondary model.
	SecondaryModelIdentifier string `json:"secondaryModelIdentifier,omitempty"`
}

// LLMProviderObservation are the observable fields of a LLMProvider.
type LLMProviderObservation struct {
	// ID is the identifier SonarQube assigned to the provider (used as
	// external-name).
	ID string `json:"id,omitempty"`
	// Provider is the observed provider type.
	Provider string `json:"provider,omitempty"`
	// Label is the observed display name.
	Label string `json:"label,omitempty"`
	// Configuration holds the observed non-secret configuration fields.
	// Secret fields are never reported.
	Configuration map[string]string `json:"configuration,omitempty"`
	// HTTPHeaders are the observed custom HTTP headers.
	HTTPHeaders []LLMProviderHTTPHeaderObservation `json:"httpHeaders,omitempty"`
	// ModelMappings are the AI capabilities currently mapped to this
	// provider.
	ModelMappings []LLMProviderModelMappingObservation `json:"modelMappings,omitempty"`
}

// A LLMProviderSpec defines the desired state of a LLMProvider.
//
// writeConnectionSecretToRef is required: SonarQube never returns secret
// configuration values, so the provider compares the referenced secrets with
// the values it last applied, saved in the connection secret, to detect
// rotations.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type LLMProviderSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider LLMProviderParameters `json:"forProvider"`
}

// A LLMProviderStatus represents the observed state of a LLMProvider.
type LLMProviderStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider LLMProviderObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A LLMProvider is a managed resource that represents a SonarQube LLM
// provider connection, used by AI features such as AI CodeFix, and the AI
// capabilities it serves. Requires SonarQube Enterprise Edition or above;
// the underlying API is marked internal by SonarQube.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PROVIDER",type="string",JSONPath=".spec.forProvider.provider"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type LLMProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMProviderSpec   `json:"spec"`
	Status LLMProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMProviderList contains a list of LLMProvider.
type LLMProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LLMProvider `json:"items"`
}

// LLMProvider type metadata.
var (
	LLMProviderKind             = reflect.TypeFor[LLMProvider]().Name()
	LLMProviderGroupKind        = schema.GroupKind{Group: Group, Kind: LLMProviderKind}.String()
	LLMProviderKindAPIVersion   = LLMProviderKind + "." + SchemeGroupVersion.String()
	LLMProviderGroupVersionKind = SchemeGroupVersion.WithKind(LLMProviderKind)
)

// init registers the LLMProvider resource with the Scheme.
func init() {
	register(&LLMProvider{}, &LLMProviderList{})
}
