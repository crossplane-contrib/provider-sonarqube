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

// DevOpsPlatform is a DevOps platform whose roles can be mapped to SonarQube
// permissions.
// +kubebuilder:validation:Enum=github;gitlab
type DevOpsPlatform string

const (
	// DevOpsPlatformGitHub selects the GitHub role mappings.
	DevOpsPlatformGitHub DevOpsPlatform = "github"
	// DevOpsPlatformGitLab selects the GitLab role mappings.
	DevOpsPlatformGitLab DevOpsPlatform = "gitlab"
)

// DevOpsPermissionMappingParameters are the configurable fields of a
// DevOpsPermissionMapping resource.
type DevOpsPermissionMappingParameters struct {
	// Platform is the DevOps platform the role belongs to. Immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Platform is immutable."
	Platform DevOpsPlatform `json:"platform"`

	// Role is the name of the role on the DevOps platform, as SonarQube lists
	// it (case-sensitive). A built-in role of the platform (for example
	// maintain or developer) always has a mapping, which this resource takes
	// over. Any other name is a custom role: its mapping is created if missing
	// and adopted if it already exists. In both cases the resource owns the
	// mapping, so declare each role of a platform in only one resource per
	// SonarQube instance. Immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[^/]+$`
	// +kubebuilder:validation:XValidation:rule="self != '.' && self != '..'",message="Role cannot be a dot path segment."
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Role is immutable."
	Role string `json:"role"`

	// Permissions is the exhaustive set of SonarQube project permissions
	// granted to the role: any permission not listed is revoked. An empty set
	// revokes all of them.
	// Valid values: user (Browse), codeviewer (See Source Code), issueadmin
	// (Administer Issues), securityhotspotadmin (Administer Security
	// Hotspots), admin (Administer), scan (Execute Analysis).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxItems=6
	// +kubebuilder:validation:items:Enum=user;codeviewer;issueadmin;securityhotspotadmin;admin;scan
	// +listType=set
	Permissions []string `json:"permissions"`
}

// DevOpsPermissionMappingObservation are the observable fields of a
// DevOpsPermissionMapping resource.
type DevOpsPermissionMappingObservation struct {
	// ID is the SonarQube identifier of the mapping.
	ID string `json:"id,omitempty"`
	// Role is the DevOps platform role of the mapping.
	Role string `json:"role,omitempty"`
	// BaseRole indicates whether the role is a built-in role of the DevOps
	// platform. The mapping of a built-in role cannot be deleted.
	BaseRole bool `json:"baseRole,omitempty"`
	// Permissions is the set of permissions currently granted to the role.
	Permissions []string `json:"permissions,omitempty"`
}

// A DevOpsPermissionMappingSpec defines the desired state of a
// DevOpsPermissionMapping resource.
type DevOpsPermissionMappingSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider DevOpsPermissionMappingParameters `json:"forProvider"`
}

// A DevOpsPermissionMappingStatus represents the observed state of a
// DevOpsPermissionMapping resource.
type DevOpsPermissionMappingStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider DevOpsPermissionMappingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DevOpsPermissionMapping resource manages the SonarQube project permissions
// granted to one GitHub or GitLab role when users are provisioned from that
// platform. It requires a commercial edition of SonarQube.
//
// Deleting the resource deletes the mapping of a custom role. The mapping of a
// built-in role cannot be deleted: it keeps its current permissions in
// SonarQube, and only the resource is removed.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PLATFORM",type="string",JSONPath=".spec.forProvider.platform"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type DevOpsPermissionMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevOpsPermissionMappingSpec   `json:"spec"`
	Status DevOpsPermissionMappingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DevOpsPermissionMappingList contains a list of DevOpsPermissionMapping.
type DevOpsPermissionMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DevOpsPermissionMapping `json:"items"`
}

// DevOpsPermissionMapping type metadata.
var (
	DevOpsPermissionMappingKind             = reflect.TypeFor[DevOpsPermissionMapping]().Name()
	DevOpsPermissionMappingGroupKind        = schema.GroupKind{Group: APIGroup, Kind: DevOpsPermissionMappingKind}.String()
	DevOpsPermissionMappingKindAPIVersion   = DevOpsPermissionMappingKind + "." + SchemeGroupVersion.String()
	DevOpsPermissionMappingGroupVersionKind = SchemeGroupVersion.WithKind(DevOpsPermissionMappingKind)
)

// init registers the DevOpsPermissionMapping resource with the Scheme.
func init() {
	register(&DevOpsPermissionMapping{}, &DevOpsPermissionMappingList{})
}
