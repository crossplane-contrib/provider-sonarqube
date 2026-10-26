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

// PermissionsParameters are the configurable fields of a Permissions resource.
// Exactly one of GroupName or Login must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.groupName) || has(self.login)) && !(has(self.groupName) && has(self.login))",message="exactly one of groupName or login must be set"
type PermissionsParameters struct {
	// GroupName is the name of the group to assign permissions to.
	// Mutually exclusive with Login. Immutable once set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GroupName is immutable."
	GroupName *string `json:"groupName,omitempty"`

	// GroupNameRef references a Group resource to populate GroupName.
	// +kubebuilder:validation:Optional
	GroupNameRef *xpv1.Reference `json:"groupNameRef,omitempty"`

	// GroupNameSelector selects a Group resource to populate GroupName.
	// +kubebuilder:validation:Optional
	GroupNameSelector *xpv1.Selector `json:"groupNameSelector,omitempty"`

	// Login is the login of the user to assign permissions to.
	// Mutually exclusive with GroupName. Immutable once set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Login is immutable."
	Login *string `json:"login,omitempty"`

	// LoginRef references a User resource to populate Login.
	// +kubebuilder:validation:Optional
	LoginRef *xpv1.Reference `json:"loginRef,omitempty"`

	// LoginSelector selects a User resource to populate Login.
	// +kubebuilder:validation:Optional
	LoginSelector *xpv1.Selector `json:"loginSelector,omitempty"`

	// ProjectKey scopes permissions to a specific project. If omitted, global permissions are managed.
	// Immutable once set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ProjectKey is immutable."
	ProjectKey *string `json:"projectKey,omitempty"`

	// ProjectKeyRef references a Project resource to populate ProjectKey.
	// +kubebuilder:validation:Optional
	ProjectKeyRef *xpv1.Reference `json:"projectKeyRef,omitempty"`

	// ProjectKeySelector selects a Project resource to populate ProjectKey.
	// +kubebuilder:validation:Optional
	ProjectKeySelector *xpv1.Selector `json:"projectKeySelector,omitempty"`

	// Permissions is the desired set of permissions to assign. An empty set revokes all permissions in this scope.
	// Global permissions: admin, gateadmin, profileadmin, provisioning, scan, applicationcreator, portfoliocreator.
	// Project permissions: admin, codeviewer, issueadmin, securityhotspotadmin, scan, user.
	// On public projects, codeviewer and user are implicit: SonarQube never reports them, so listing them never converges.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=0
	Permissions []string `json:"permissions"`
}

// PermissionsObservation are the observable fields of a Permissions resource.
type PermissionsObservation struct {
	// Permissions is the currently observed set of permissions in SonarQube.
	// +kubebuilder:validation:Optional
	Permissions []string `json:"permissions,omitempty"`
}

// A PermissionsSpec defines the desired state of a Permissions resource.
type PermissionsSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider PermissionsParameters `json:"forProvider"`
}

// A PermissionsStatus represents the observed state of a Permissions resource.
type PermissionsStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider PermissionsObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Permissions resource manages a set of SonarQube permissions for a group
// or user, optionally scoped to a project.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Permissions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermissionsSpec   `json:"spec"`
	Status PermissionsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PermissionsList contains a list of Permissions.
type PermissionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Permissions `json:"items"`
}

// Permissions type metadata.
var (
	PermissionsKind             = reflect.TypeFor[Permissions]().Name()
	PermissionsGroupKind        = schema.GroupKind{Group: APIGroup, Kind: PermissionsKind}.String()
	PermissionsKindAPIVersion   = PermissionsKind + "." + SchemeGroupVersion.String()
	PermissionsGroupVersionKind = SchemeGroupVersion.WithKind(PermissionsKind)
)

// init registers the Permissions resource with the Scheme.
func init() {
	register(&Permissions{}, &PermissionsList{})
}
