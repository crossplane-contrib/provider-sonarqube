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

// ALMAzureParameters are the configurable fields of a ALMAzure.
type ALMAzureParameters struct {
	ALMCommonParameters `json:",inline"`
}

// ALMAzureObservation are the observable fields of a ALMAzure.
type ALMAzureObservation struct {
	ALMCommonObservation `json:",inline"`
}

// A ALMAzureSpec defines the desired state of a ALMAzure.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type ALMAzureSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ALMAzureParameters `json:"forProvider"`
}

// A ALMAzureStatus represents the observed state of a ALMAzure.
type ALMAzureStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider ALMAzureObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An ALMAzure is a managed resource that represents a
// SonarQube ALM integration for Azure DevOps.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type ALMAzure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ALMAzureSpec   `json:"spec"`
	Status ALMAzureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ALMAzureList contains a list of ALMAzure.
type ALMAzureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ALMAzure `json:"items"`
}

// ALMAzure type metadata.
var (
	ALMAzureKind             = reflect.TypeFor[ALMAzure]().Name()
	ALMAzureGroupKind        = schema.GroupKind{Group: Group, Kind: ALMAzureKind}.String()
	ALMAzureKindAPIVersion   = ALMAzureKind + "." + SchemeGroupVersion.String()
	ALMAzureGroupVersionKind = SchemeGroupVersion.WithKind(ALMAzureKind)
)

// init registers the ALMAzure resource with the Scheme.
func init() {
	SchemeBuilder.Register(&ALMAzure{}, &ALMAzureList{})
}
