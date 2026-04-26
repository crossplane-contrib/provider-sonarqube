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

// GroupParameters are the configurable fields of a Group.
type GroupParameters struct {
	// Name of the Group. This is a required field and must be unique across all Groups in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	Name string `json:"name"`
	// Description of the Group. This is an optional field that provides additional information about the Group.
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`
	// Permissions is a list of permissions to be assigned to the Group. This is an optional field that specifies the permissions the Group should have in SonarQube.
	// Allowed values are "admin", "gateadmin", "profileadmin", "provisioning", "scan", "applicationcreator", "portfoliocreator"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:items:Enum=admin;gateadmin;profileadmin;provisioning;scan;applicationcreator;portfoliocreator
	Permissions *[]string `json:"permissions,omitempty"`
}

// GroupObservation are the observable fields of a Group.
type GroupObservation struct {
	// ID is the unique identifier of the Group in SonarQube. This field is set by the SonarQube API.
	ID string `json:"id"`
	// Name is the name of the Group as observed from SonarQube. Reflects the actual name of the Group in SonarQube.
	Name string `json:"name"`
	// Description is the description of the Group as observed from SonarQube. This field the actual description of the Group in SonarQube.
	Description string `json:"description"`
	// Managed indicates if the group is created and managed by SonarQube itself. Managed groups cannot be modified or deleted by users, and are typically used for internal purposes within SonarQube.
	Managed bool `json:"managed"`
	// Default indicates whether the Group is a default group in SonarQube.
	Default bool `json:"default"`
	// Permissions is a list of permissions assigned to the Group as observed from SonarQube. This field reflects the actual permissions the Group has in SonarQube.
	Permissions []string `json:"permissions"`
}

// A GroupSpec defines the desired state of a Group.
type GroupSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider GroupParameters `json:"forProvider"`
}

// A GroupStatus represents the observed state of a Group.
type GroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider GroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Group is the Schema for the Groups API. It represents a
// Group resource in SonarQube, which is a collection of users that can be
// assigned permissions to access and manage resources in SonarQube.
// The Group resource allows you to define and manage groups of users in
// SonarQube, including their names, descriptions, and permissions.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSpec   `json:"spec"`
	Status GroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Group.
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Group `json:"items"`
}

// Group type metadata.
var (
	GroupKind             = reflect.TypeFor[Group]().Name()
	GroupGroupKind        = schema.GroupKind{Group: APIGroup, Kind: GroupKind}.String()
	GroupKindAPIVersion   = GroupKind + "." + SchemeGroupVersion.String()
	GroupGroupVersionKind = SchemeGroupVersion.WithKind(GroupKind)
)

// init registers the Group resource with the Scheme.
func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
