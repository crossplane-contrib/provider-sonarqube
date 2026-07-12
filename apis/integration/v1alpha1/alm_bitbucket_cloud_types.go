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

// ALMBitbucketCloudParameters are the configurable fields of a
// ALMBitbucketCloud.
type ALMBitbucketCloudParameters struct {
	// Key is the unique identifier for the ALM integration in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// ClientID is the Bitbucket Cloud OAuth consumer Client ID.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	ClientID string `json:"clientId"`
	// ClientSecretRef references a Secret containing the Bitbucket Cloud
	// OAuth consumer Client Secret.
	// +kubebuilder:validation:Required
	ClientSecretRef *xpv1.LocalSecretKeySelector `json:"clientSecretRef"`
	// Workspace is the Bitbucket Cloud workspace ID.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	Workspace string `json:"workspace"`
}

// ALMBitbucketCloudObservation are the observable fields of a
// ALMBitbucketCloud.
type ALMBitbucketCloudObservation struct {
	// Key is the unique identifier for the ALM integration in SonarQube.
	Key string `json:"key,omitempty"`
	// ClientID is the Bitbucket Cloud OAuth consumer Client ID as observed
	// from the SonarQube API.
	ClientID string `json:"clientId,omitempty"`
	// Workspace is the Bitbucket Cloud workspace ID as observed from the
	// SonarQube API.
	Workspace string `json:"workspace,omitempty"`
}

// A ALMBitbucketCloudSpec defines the desired state of a ALMBitbucketCloud.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type ALMBitbucketCloudSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ALMBitbucketCloudParameters `json:"forProvider"`
}

// A ALMBitbucketCloudStatus represents the observed state of a
// ALMBitbucketCloud.
type ALMBitbucketCloudStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider ALMBitbucketCloudObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ALMBitbucketCloud is a managed resource that represents a SonarQube
// ALM integration for Bitbucket Cloud.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type ALMBitbucketCloud struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ALMBitbucketCloudSpec   `json:"spec"`
	Status ALMBitbucketCloudStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ALMBitbucketCloudList contains a list of ALMBitbucketCloud.
type ALMBitbucketCloudList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ALMBitbucketCloud `json:"items"`
}

// ALMBitbucketCloud type metadata.
var (
	ALMBitbucketCloudKind             = reflect.TypeFor[ALMBitbucketCloud]().Name()
	ALMBitbucketCloudGroupKind        = schema.GroupKind{Group: Group, Kind: ALMBitbucketCloudKind}.String()
	ALMBitbucketCloudKindAPIVersion   = ALMBitbucketCloudKind + "." + SchemeGroupVersion.String()
	ALMBitbucketCloudGroupVersionKind = SchemeGroupVersion.WithKind(ALMBitbucketCloudKind)
)

// init registers the ALMBitbucketCloud resource with the Scheme.
func init() {
	SchemeBuilder.Register(&ALMBitbucketCloud{}, &ALMBitbucketCloudList{})
}
