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

// NewCodePeriodParameters are the configurable fields of a NewCodePeriod.
// +kubebuilder:validation:XValidation:rule="self.type == 'PREVIOUS_VERSION' ? !has(self.value) : true",message="value must not be set when type is PREVIOUS_VERSION"
// +kubebuilder:validation:XValidation:rule="self.type != 'NUMBER_OF_DAYS' || (has(self.value) && size(self.value) > 0)",message="value must be set when type is NUMBER_OF_DAYS"
type NewCodePeriodParameters struct {
	// Type is the type of the instance-wide default new code definition.
	// Only PREVIOUS_VERSION and NUMBER_OF_DAYS can be set at instance level.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=PREVIOUS_VERSION;NUMBER_OF_DAYS
	Type string `json:"type"`
	// Value is the value of the new code definition. It must be omitted when the
	// type is PREVIOUS_VERSION, and a number between 1 and 90 when the type is
	// NUMBER_OF_DAYS.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern="^([1-9]|[1-8][0-9]|90)?$"
	Value *string `json:"value,omitempty"`
}

// NewCodePeriodObservation are the observable fields of a NewCodePeriod.
type NewCodePeriodObservation struct {
	// Type is the observed type of the instance-wide default new code definition.
	Type string `json:"type,omitempty"`
	// Value is the observed value of the new code definition.
	Value string `json:"value,omitempty"`
	// Inherited indicates whether the value is inherited from a parent.
	Inherited bool `json:"inherited,omitempty"`
	// UpdatedAt is the timestamp (Unix epoch milliseconds) of the last update.
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

// A NewCodePeriodSpec defines the desired state of a NewCodePeriod.
type NewCodePeriodSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider NewCodePeriodParameters `json:"forProvider"`
}

// A NewCodePeriodStatus represents the observed state of a NewCodePeriod.
type NewCodePeriodStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider NewCodePeriodObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// NewCodePeriod manages the instance-wide default new code definition of a
// SonarQube instance.
// WARNING: only one NewCodePeriod resource should target a given SonarQube
// instance, as multiple resources would conflict with each other.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type NewCodePeriod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NewCodePeriodSpec   `json:"spec"`
	Status NewCodePeriodStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NewCodePeriodList contains a list of NewCodePeriod.
type NewCodePeriodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NewCodePeriod `json:"items"`
}

// NewCodePeriod type metadata.
var (
	NewCodePeriodKind             = reflect.TypeFor[NewCodePeriod]().Name()
	NewCodePeriodGroupKind        = schema.GroupKind{Group: Group, Kind: NewCodePeriodKind}.String()
	NewCodePeriodKindAPIVersion   = NewCodePeriodKind + "." + SchemeGroupVersion.String()
	NewCodePeriodGroupVersionKind = SchemeGroupVersion.WithKind(NewCodePeriodKind)
)

// init registers the NewCodePeriod resource with the Scheme.
func init() {
	SchemeBuilder.Register(&NewCodePeriod{}, &NewCodePeriodList{})
}
