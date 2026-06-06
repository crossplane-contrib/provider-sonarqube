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

// PermissionsTemplateParameters are the configurable fields of a
// PermissionsTemplate.
type PermissionsTemplateParameters struct {
	// Name of the PermissionsTemplate. This is a required field and must be unique within the SonarQube instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Description of the PermissionsTemplate. This is an optional field that provides additional information about the PermissionsTemplate.
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`
	// ProjectKeyPattern is a regex pattern that defines which projects the PermissionsTemplate applies to. This is an optional field that allows you to specify a pattern for matching project keys.
	// +kubebuilder:validation:Optional
	ProjectKeyPattern *string `json:"projectKeyPattern,omitempty"`
	// Default is a boolean field that indicates whether this PermissionsTemplate is the default template for new projects. This is an optional field that can be set to true if you want this template to be used as the default for new projects.
	// +kubebuilder:validation:Optional
	Default *bool `json:"default,omitempty"`
	// CreatorPermissions is a list of permissions that apply to the creator of the project. This is an optional field that allows you to specify permissions for the project creator.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:items:Enum=admin;codeviewer;issueadmin;securityhotspotadmin;scan;user
	CreatorPermissions *[]string `json:"creatorPermissions,omitempty"`
	// UserPermissions is a list of permissions assigned to specific users in the context of the PermissionsTemplate. This is an optional field that allows you to specify user-specific permissions for the PermissionsTemplate.
	// +kubebuilder:validation:Optional
	UserPermissions *[]PermissionsTemplateUserParameters `json:"userPermissions,omitempty"`
	// GroupPermissions is a list of permissions assigned to specific groups in the context of the PermissionsTemplate. This is an optional field that allows you to specify group-specific permissions for the PermissionsTemplate.
	// +kubebuilder:validation:Optional
	GroupPermissions *[]PermissionsTemplateGroupParameters `json:"groupPermissions,omitempty"`
}

// PermissionsTemplateUserParameters is a struct that represents the permissions
// assigned to a user in a SonarQube Permission Template.
type PermissionsTemplateUserParameters struct {
	// Login is the unique identifier of the user in SonarQube. This field is required to associate the permissions with a specific user.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Login string `json:"login"`
	// Permissions is a list of permissions assigned to the user. This field is required and specifies the permissions the user has in the context of the Permission Template.
	// Allowed values are "admin", "codeviewer", "issueadmin", "securityhotspotadmin", "scan", "user"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:items:Enum=admin;codeviewer;issueadmin;securityhotspotadmin;scan;user
	Permissions *[]string `json:"permissions"`
}

// PermissionsTemplateGroupParameters is a struct that represents the
// permissions assigned to a group in a SonarQube Permission Template.
type PermissionsTemplateGroupParameters struct {
	// Name is the name of the group in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Permissions is a list of permissions assigned to the group. An empty list means that the group will be associated with the PermissionsTemplate but will not have any permissions.
	// Allowed values are "admin", "codeviewer", "issueadmin", "securityhotspotadmin", "scan", "user"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:items:Enum=admin;codeviewer;issueadmin;securityhotspotadmin;scan;user
	Permissions *[]string `json:"permissions,omitempty"`
}

// PermissionsTemplateObservation are the observable fields of a
// PermissionsTemplate.
type PermissionsTemplateObservation struct {
	// ID is the unique identifier of the PermissionsTemplate in SonarQube.  This field is created by the SonarQube API when the PermissionsTemplate is created.
	ID string `json:"id"`
	// Name is the name of the PermissionsTemplate. This reflects the actual name of the PermissionsTemplate in SonarQube.
	Name string `json:"name"`
	// Description is the description of the PermissionsTemplate. This reflects the actual description of the PermissionsTemplate in SonarQube.
	Description string `json:"description"`
	// ProjectKeyPattern is a regex pattern that defines which projects the PermissionsTemplate applies to. This reflects the actual pattern of the PermissionsTemplate in SonarQube.
	ProjectKeyPattern string `json:"projectKeyPattern"`
	// Default is a boolean field that indicates whether this PermissionsTemplate is the default template for new projects. This reflects the actual default status of the PermissionsTemplate in SonarQube.
	Default bool `json:"default"`
	// CreatorPermissions is a list of permissions that apply to the creator of the project. This reflects the actual permissions for the project creator in SonarQube.
	// +kubebuilder:validation:Optional
	CreatorPermissions []string `json:"creatorPermissions"`
	// UserPermissions is a list of permissions assigned to specific users in the context of the PermissionsTemplate. This reflects the actual user-specific permissions for the PermissionsTemplate in SonarQube.
	// +kubebuilder:validation:Optional
	UserPermissions []PermissionsTemplateUserObservation `json:"userPermissions"`
	// GroupPermissions is a list of permissions assigned to specific groups in the context of the PermissionsTemplate. This reflects the actual group-specific permissions for the PermissionsTemplate in SonarQube.
	// +kubebuilder:validation:Optional
	GroupPermissions []PermissionsTemplateGroupObservation `json:"groupPermissions"`
	// CreatedAt is the timestamp when the PermissionsTemplate was created in SonarQube. This field is set by the SonarQube API and reflects the actual creation time of the PermissionsTemplate.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
	// UpdatedAt is the timestamp when the PermissionsTemplate was last updated in SonarQube. This field is set by the SonarQube API and reflects the actual last update time of the PermissionsTemplate.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

// PermissionsTemplateUserObservation is a struct that represents
// the observed permissions for a user in a SonarQube Permission Template.
type PermissionsTemplateUserObservation struct {
	// Login is the unique identifier of the user in SonarQube. This field is observed from the SonarQube API and reflects the actual login of the user associated with the PermissionsTemplate.
	Login string `json:"login"`
	// Permissions is a list of permissions assigned to the user in the context of the PermissionsTemplate. This field is observed from the SonarQube API and reflects the actual permissions assigned to the user for the PermissionsTemplate.
	Permissions []string `json:"permissions"`
}

// PermissionsTemplateGroupObservation is a struct that represents the observed
// state of a group in a SonarQube Permission Template.
// It contains the actual name of the group and the permissions assigned to it
// as observed from the SonarQube API response.
type PermissionsTemplateGroupObservation struct {
	// Name is the name of the group in SonarQube. This field is observed from the SonarQube API and reflects the actual name of the group associated with the PermissionsTemplate.
	Name string `json:"name"`
	// Permissions represents the list of permissions assigned to the group in the context of the PermissionsTemplate. This field is observed from the SonarQube API and reflects the actual permissions assigned to the group for the PermissionsTemplate.
	Permissions []string `json:"permissions"`
}

// A PermissionsTemplateSpec defines the desired state of a PermissionsTemplate.
type PermissionsTemplateSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider PermissionsTemplateParameters `json:"forProvider"`
}

// A PermissionsTemplateStatus represents the observed state of a
// PermissionsTemplate.
type PermissionsTemplateStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider PermissionsTemplateObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A PermissionsTemplate is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type PermissionsTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermissionsTemplateSpec   `json:"spec"`
	Status PermissionsTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PermissionsTemplateList contains a list of PermissionsTemplate.
type PermissionsTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PermissionsTemplate `json:"items"`
}

// PermissionsTemplate type metadata.
var (
	PermissionsTemplateKind             = reflect.TypeFor[PermissionsTemplate]().Name()
	PermissionsTemplateGroupKind        = schema.GroupKind{Group: APIGroup, Kind: PermissionsTemplateKind}.String()
	PermissionsTemplateKindAPIVersion   = PermissionsTemplateKind + "." + SchemeGroupVersion.String()
	PermissionsTemplateGroupVersionKind = SchemeGroupVersion.WithKind(PermissionsTemplateKind)
)

// init registers the PermissionsTemplate resource with the Scheme.
func init() {
	SchemeBuilder.Register(&PermissionsTemplate{}, &PermissionsTemplateList{})
}
