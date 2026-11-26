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

// InstanceConfigurationParameters are the configurable fields of an
// InstanceConfiguration.
//
// Every field is optional and maps to one instance-wide option that SonarQube
// sets through a dedicated endpoint rather than api/settings. Only the fields
// that are set are observed and reconciled: the options left out keep the
// value they have in SonarQube, so Community Edition users can simply omit the
// Enterprise Edition fields.
type InstanceConfigurationParameters struct {
	// CleanCodePolicyMode is the mode of the instance: MQR (Multi-Quality
	// Rules) or STANDARD_EXPERIENCE. Changing it makes SonarQube re-index
	// issues and rules.
	// +optional
	// +kubebuilder:validation:Enum=MQR;STANDARD_EXPERIENCE
	CleanCodePolicyMode *string `json:"cleanCodePolicyMode,omitempty"`

	// DefaultProjectVisibility is the visibility given to new projects that
	// are created without an explicit one. It does not change the visibility
	// of existing projects.
	// +optional
	// +kubebuilder:validation:Enum=public;private
	DefaultProjectVisibility *string `json:"defaultProjectVisibility,omitempty"`

	// ComputeEngineWorkerCount is the number of Compute Engine workers that
	// process analysis reports in parallel.
	// Requires SonarQube Enterprise Edition or above. SonarQube marks this
	// endpoint as internal.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	ComputeEngineWorkerCount *int64 `json:"computeEngineWorkerCount,omitempty"`

	// SCIMEnabled enables or disables SCIM user and group provisioning.
	// Enabling it makes users and groups read-only in SonarQube; disabling it
	// removes all provisioning data, turning provisioned users and groups
	// into local ones.
	// Requires SonarQube Enterprise Edition or above. SonarQube marks this
	// endpoint as internal.
	// +optional
	SCIMEnabled *bool `json:"scimEnabled,omitempty"`

	// SCAEnabled enables or disables Software Composition Analysis (SonarQube
	// Advanced Security).
	// Requires SonarQube Enterprise Edition or above with a license that
	// includes SCA. SonarQube marks this endpoint as internal.
	// +optional
	SCAEnabled *bool `json:"scaEnabled,omitempty"`
}

// InstanceConfigurationObservation are the observable fields of an
// InstanceConfiguration. Only the options set in the spec are observed, the
// others are left empty.
type InstanceConfigurationObservation struct {
	// CleanCodePolicyMode is the observed mode of the instance.
	CleanCodePolicyMode *string `json:"cleanCodePolicyMode,omitempty"`

	// DefaultProjectVisibility is the observed default visibility of new
	// projects.
	DefaultProjectVisibility *string `json:"defaultProjectVisibility,omitempty"`

	// ComputeEngineWorkerCount is the observed number of Compute Engine
	// workers.
	ComputeEngineWorkerCount *int64 `json:"computeEngineWorkerCount,omitempty"`

	// SCIMEnabled is whether SCIM provisioning is observed as enabled.
	SCIMEnabled *bool `json:"scimEnabled,omitempty"`

	// SCAEnabled is whether Software Composition Analysis is observed as
	// enabled.
	SCAEnabled *bool `json:"scaEnabled,omitempty"`
}

// An InstanceConfigurationSpec defines the desired state of an
// InstanceConfiguration.
type InstanceConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider InstanceConfigurationParameters `json:"forProvider"`
}

// An InstanceConfigurationStatus represents the observed state of an
// InstanceConfiguration.
type InstanceConfigurationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider InstanceConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An InstanceConfiguration manages instance-wide options of a SonarQube
// instance that are set through dedicated endpoints rather than api/settings.
// Only the options set in spec.forProvider are reconciled.
// WARNING: only one InstanceConfiguration resource should target a given
// SonarQube instance, as multiple resources would conflict with each other.
// Deleting this resource leaves the options at their current value in
// SonarQube.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type InstanceConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceConfigurationSpec   `json:"spec"`
	Status InstanceConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceConfigurationList contains a list of InstanceConfiguration.
type InstanceConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InstanceConfiguration `json:"items"`
}

// InstanceConfiguration type metadata.
var (
	InstanceConfigurationKind             = reflect.TypeFor[InstanceConfiguration]().Name()
	InstanceConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceConfigurationKind}.String()
	InstanceConfigurationKindAPIVersion   = InstanceConfigurationKind + "." + SchemeGroupVersion.String()
	InstanceConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(InstanceConfigurationKind)
)

// init registers the InstanceConfiguration resource with the Scheme.
func init() {
	register(&InstanceConfiguration{}, &InstanceConfigurationList{})
}
