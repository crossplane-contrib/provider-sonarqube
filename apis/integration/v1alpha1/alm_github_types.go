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

// ALMGitHubParameters are the configurable fields of a ALMGitHub.
type ALMGitHubParameters struct {
	// URL is the GitHub API URL.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`
	// Key is the unique identifier for the ALM integration in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// AppID is the GitHub App ID.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	AppID string `json:"appId"`
	// ClientID is the GitHub App Client ID.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	ClientID string `json:"clientId"`
	// ClientSecretRef references a Secret containing the GitHub App Client Secret.
	// +kubebuilder:validation:Required
	ClientSecretRef *xpv1.LocalSecretKeySelector `json:"clientSecretRef"`
	// PrivateKeyRef references a Secret containing the GitHub App Private Key.
	// +kubebuilder:validation:Required
	PrivateKeyRef *xpv1.LocalSecretKeySelector `json:"privateKeyRef"`
	// WebhookSecretRef references a Secret containing the GitHub App Webhook Secret.
	// +kubebuilder:validation:Optional
	WebhookSecretRef *xpv1.LocalSecretKeySelector `json:"webhookSecretRef,omitempty"`
}

// ALMGitHubObservation are the observable fields of a ALMGitHub.
type ALMGitHubObservation struct {
	ALMCommonObservation `json:",inline"`

	// AppID is the GitHub App ID as observed from the SonarQube API.
	AppID string `json:"appId,omitempty"`
	// ClientID is the GitHub App Client ID as observed from the SonarQube API.
	ClientID string `json:"clientId,omitempty"`
}

// A ALMGitHubSpec defines the desired state of a ALMGitHub.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type ALMGitHubSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ALMGitHubParameters `json:"forProvider"`
}

// A ALMGitHubStatus represents the observed state of a ALMGitHub.
type ALMGitHubStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider ALMGitHubObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ALMGitHub is a managed resource that represents a SonarQube ALM integration
// for GitHub.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type ALMGitHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ALMGitHubSpec   `json:"spec"`
	Status ALMGitHubStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ALMGitHubList contains a list of ALMGitHub.
type ALMGitHubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ALMGitHub `json:"items"`
}

// ALMGitHub type metadata.
var (
	ALMGitHubKind             = reflect.TypeFor[ALMGitHub]().Name()
	ALMGitHubGroupKind        = schema.GroupKind{Group: Group, Kind: ALMGitHubKind}.String()
	ALMGitHubKindAPIVersion   = ALMGitHubKind + "." + SchemeGroupVersion.String()
	ALMGitHubGroupVersionKind = SchemeGroupVersion.WithKind(ALMGitHubKind)
)

// init registers the ALMGitHub resource with the Scheme.
func init() {
	SchemeBuilder.Register(&ALMGitHub{}, &ALMGitHubList{})
}
