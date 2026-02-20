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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// AlmGithubParameters represent the desired state of an ALM GitHub integration.
type AlmGithubParameters struct {
	// Key is the unique key for the GitHub instance setting.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable."
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Key string `json:"key"`
	// AppID is the GitHub App ID.
	// +kubebuilder:validation:MaxLength=80
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	AppID string `json:"appId"`
	// ClientID is the GitHub App Client ID.
	// +kubebuilder:validation:MaxLength=80
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`
	// ClientSecretSecretRef is a reference to a Secret key containing the GitHub App Client Secret.
	// +kubebuilder:validation:Required
	ClientSecretSecretRef xpv1.SecretKeySelector `json:"clientSecretSecretRef"`
	// PrivateKeySecretRef is a reference to a Secret key containing the GitHub App private key.
	// +kubebuilder:validation:Required
	PrivateKeySecretRef xpv1.SecretKeySelector `json:"privateKeySecretRef"`
	// URL is the GitHub API URL.
	// +kubebuilder:validation:MaxLength=2000
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// WebhookSecretSecretRef is a reference to a Secret key containing the GitHub App Webhook Secret.
	// +kubebuilder:validation:Optional
	WebhookSecretSecretRef *xpv1.SecretKeySelector `json:"webhookSecretSecretRef,omitempty"`
}

// AlmGithubObservation are the observable fields of an ALM GitHub integration.
type AlmGithubObservation struct {
	// Key is the unique key of the GitHub instance setting.
	Key string `json:"key"`
	// AppID is the GitHub App ID.
	AppID string `json:"appId"`
	// ClientID is the GitHub App Client ID.
	ClientID string `json:"clientId"`
	// URL is the GitHub API URL.
	URL string `json:"url"`
}

// An AlmGithubSpec defines the desired state of an ALM GitHub integration.
type AlmGithubSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider represents the desired state of the ALM GitHub integration.
	ForProvider AlmGithubParameters `json:"forProvider"`
}

// An AlmGithubStatus represents the observed state of an ALM GitHub integration.
type AlmGithubStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider represents the observed state of the ALM GitHub integration.
	AtProvider AlmGithubObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An AlmGithub manages a SonarQube ALM GitHub integration.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type AlmGithub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlmGithubSpec   `json:"spec"`
	Status AlmGithubStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlmGithubList contains a list of AlmGithub.
type AlmGithubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AlmGithub `json:"items"`
}

// AlmGithub type metadata.
var (
	AlmGithubKind             = reflect.TypeFor[AlmGithub]().Name()
	AlmGithubGroupKind        = schema.GroupKind{Group: Group, Kind: AlmGithubKind}.String()
	AlmGithubKindAPIVersion   = AlmGithubKind + "." + SchemeGroupVersion.String()
	AlmGithubGroupVersionKind = SchemeGroupVersion.WithKind(AlmGithubKind)
)

func init() {
	SchemeBuilder.Register(&AlmGithub{}, &AlmGithubList{})
}
