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

//nolint:dupl // Provider-specific ALM type files intentionally share this structure.
package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// ALMBitbucketParameters are the configurable fields of a ALMBitbucket.
type ALMBitbucketParameters struct {
	ALMCommonParameters `json:",inline"`
}

// ALMBitbucketObservation are the observable fields of a ALMBitbucket.
type ALMBitbucketObservation struct {
	ALMCommonObservation `json:",inline"`
}

// A ALMBitbucketSpec defines the desired state of a ALMBitbucket.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type ALMBitbucketSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ALMBitbucketParameters `json:"forProvider"`
}

// A ALMBitbucketStatus represents the observed state of a ALMBitbucket.
type ALMBitbucketStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider ALMBitbucketObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ALMBitbucket is a managed resource that represents a SonarQube ALM
// integration for Bitbucket Server.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type ALMBitbucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ALMBitbucketSpec   `json:"spec"`
	Status ALMBitbucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ALMBitbucketList contains a list of ALMBitbucket.
type ALMBitbucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ALMBitbucket `json:"items"`
}

// ALMBitbucket type metadata.
var (
	ALMBitbucketKind             = reflect.TypeFor[ALMBitbucket]().Name()
	ALMBitbucketGroupKind        = schema.GroupKind{Group: Group, Kind: ALMBitbucketKind}.String()
	ALMBitbucketKindAPIVersion   = ALMBitbucketKind + "." + SchemeGroupVersion.String()
	ALMBitbucketGroupVersionKind = SchemeGroupVersion.WithKind(ALMBitbucketKind)
)

// init registers the ALMBitbucket resource with the Scheme.
func init() {
	SchemeBuilder.Register(&ALMBitbucket{}, &ALMBitbucketList{})
}
