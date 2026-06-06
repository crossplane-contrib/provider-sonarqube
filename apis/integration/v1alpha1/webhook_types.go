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

// WebhookParameters are the configurable fields of a Webhook.
type WebhookParameters struct {
	// Name is the display name of the webhook.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// URL is the target endpoint for webhook payloads.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	URL string `json:"url"`

	// ProjectKey scopes the webhook to a specific SonarQube project.
	// If omitted the webhook is global. Immutable after creation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=400
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ProjectKey is immutable."
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.Project
	ProjectKey *string `json:"projectKey,omitempty"`

	// ProjectKeyRef resolves projectKey from a Project managed resource.
	// +kubebuilder:validation:Optional
	ProjectKeyRef *xpv1.NamespacedReference `json:"projectKeyRef,omitempty"`

	// ProjectKeySelector selects a Project managed resource to resolve projectKey.
	// +kubebuilder:validation:Optional
	ProjectKeySelector *xpv1.NamespacedSelector `json:"projectKeySelector,omitempty"`

	// SecretRef references a Kubernetes Secret containing the HMAC signing secret.
	// The secret value must be 16-200 characters (SonarQube requirement).
	// WARNING: This must be used together with writeConnectionSecretToRef to
	// avoid permanent reconcile state due to missing secret.
	// +kubebuilder:validation:Optional
	SecretRef *xpv1.LocalSecretKeySelector `json:"secretRef,omitempty"`
}

// WebhookObservation are the observable fields of a Webhook.
type WebhookObservation struct {
	// Key is the unique identifier assigned by SonarQube (used as external-name).
	Key string `json:"key,omitempty"`
	// Name is the observed display name.
	Name string `json:"name,omitempty"`
	// URL is the observed target endpoint.
	URL string `json:"url,omitempty"`
	// HasSecret indicates whether an HMAC secret is configured.
	HasSecret bool `json:"hasSecret,omitempty"`
}

// A WebhookSpec defines the desired state of a Webhook.
type WebhookSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider WebhookParameters `json:"forProvider"`
}

// A WebhookStatus represents the observed state of a Webhook.
type WebhookStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider WebhookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Webhook is a managed resource that represents a SonarQube Webhook.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Webhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookSpec   `json:"spec"`
	Status WebhookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebhookList contains a list of Webhook.
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Webhook `json:"items"`
}

// Webhook type metadata.
var (
	WebhookKind             = reflect.TypeFor[Webhook]().Name()
	WebhookGroupKind        = schema.GroupKind{Group: Group, Kind: WebhookKind}.String()
	WebhookKindAPIVersion   = WebhookKind + "." + SchemeGroupVersion.String()
	WebhookGroupVersionKind = SchemeGroupVersion.WithKind(WebhookKind)
)

// init registers the Webhook resource with the Scheme.
func init() {
	SchemeBuilder.Register(&Webhook{}, &WebhookList{})
}
