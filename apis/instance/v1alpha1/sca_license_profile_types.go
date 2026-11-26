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

// SCALicenseProfileParameters are the configurable fields of a
// SCALicenseProfile.
type SCALicenseProfileParameters struct {
	// Name is the license profile display name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Organization is the key of the organization the license profile
	// belongs to. SonarQube Server has a single organization, whose key is
	// default-organization. Immutable after creation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=default-organization
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Organization is immutable"
	Organization string `json:"organization,omitempty"`

	// Default sets whether this license profile is the default one, used to
	// analyze the projects that are not assigned to any license profile.
	// When unset, it is late-initialized from SonarQube.
	// +kubebuilder:validation:Optional
	Default *bool `json:"default,omitempty"`

	// Categories are the policies of license categories. Only the listed
	// categories are managed: the other categories keep their policy.
	// Categories are applied before Licenses, so the listed license
	// policies take precedence over the policy of their category.
	// +kubebuilder:validation:Optional
	// +listType=map
	// +listMapKey=category
	Categories []SCALicenseProfileCategoryPolicy `json:"categories,omitempty"`

	// Licenses are the policies of single licenses. Only the listed licenses
	// are managed: the other licenses keep their policy.
	// +kubebuilder:validation:Optional
	// +listType=map
	// +listMapKey=spdxLicenseId
	Licenses []SCALicenseProfileLicensePolicy `json:"licenses,omitempty"`

	// Projects is the exhaustive list of keys of the projects assigned to the
	// license profile. Any project assigned to this license profile that is
	// not in this list is unassigned from it, and then falls back to the
	// default license profile.
	// +kubebuilder:validation:Optional
	// +listType=set
	Projects []string `json:"projects,omitempty"`

	// ProjectRefs are references to Projects used to populate Projects.
	// +kubebuilder:validation:Optional
	ProjectRefs []xpv1.Reference `json:"projectRefs,omitempty"`

	// ProjectSelector selects references to Projects used to populate Projects.
	// +kubebuilder:validation:Optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`
}

// SCALicenseProfileCategoryPolicy is the policy of a license category
// within a SCALicenseProfile.
type SCALicenseProfileCategoryPolicy struct {
	// Category is the license category key.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=UNKNOWN;COPYLEFT_WEAK;COPYLEFT_STRONG;COPYLEFT_NETWORK;COPYLEFT_MAXIMAL;PERMISSIVE_STANDARD;PERMISSIVE_AMATEUR
	Category string `json:"category"`

	// Policy is the policy applied to the licenses of the category.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ALLOW;DENY
	Policy string `json:"policy"`
}

// SCALicenseProfileLicensePolicy is the policy of a single license within a
// SCALicenseProfile.
type SCALicenseProfileLicensePolicy struct {
	// SPDXLicenseID is the SPDX identifier of the license, as reported by
	// SonarQube (for example MIT or Apache-2.0).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	SPDXLicenseID string `json:"spdxLicenseId"`

	// Policy is the policy applied to the license.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ALLOW;DENY
	Policy string `json:"policy"`
}

// SCALicenseProfileObservation are the observable fields of a
// SCALicenseProfile.
type SCALicenseProfileObservation struct {
	// ID is the license profile identifier.
	ID string `json:"id,omitempty"`
	// Name is the license profile name.
	Name string `json:"name,omitempty"`
	// Default indicates whether this is the default license profile.
	Default bool `json:"default,omitempty"`
	// UpdatedAt is the time of the last update of the license profile.
	UpdatedAt string `json:"updatedAt,omitempty"`
	// Categories are the policies of every license category.
	Categories []SCALicenseProfileCategoryObservation `json:"categories,omitempty"`
	// Licenses are the policies of the licenses listed in the spec. The
	// other licenses are not reported, to keep the status small.
	Licenses []SCALicenseProfileLicenseObservation `json:"licenses,omitempty"`
	// Projects are the keys of the projects assigned to the license profile.
	Projects []string `json:"projects,omitempty"`
}

// SCALicenseProfileCategoryObservation is the observed policy of a license
// category within a SCALicenseProfile.
type SCALicenseProfileCategoryObservation struct {
	// Category is the license category key.
	Category string `json:"category"`
	// Policy is the policy applied to the licenses of the category.
	Policy string `json:"policy,omitempty"`
}

// SCALicenseProfileLicenseObservation is the observed policy of a single
// license within a SCALicenseProfile.
type SCALicenseProfileLicenseObservation struct {
	// ID is the identifier of the license policy in the license profile.
	ID string `json:"id,omitempty"`
	// SPDXLicenseID is the SPDX identifier of the license.
	SPDXLicenseID string `json:"spdxLicenseId"`
	// Name is the license display name.
	Name string `json:"name,omitempty"`
	// Category is the license category key.
	Category string `json:"category,omitempty"`
	// Policy is the policy applied to the license.
	Policy string `json:"policy,omitempty"`
}

// A SCALicenseProfileSpec defines the desired state of a SCALicenseProfile.
type SCALicenseProfileSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider SCALicenseProfileParameters `json:"forProvider"`
}

// A SCALicenseProfileStatus represents the observed state of a
// SCALicenseProfile.
type SCALicenseProfileStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider SCALicenseProfileObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SCALicenseProfile manages a SonarQube Software Composition Analysis (SCA)
// license profile (Enterprise Edition only, with SCA enabled): the policies
// allowing or denying license categories and single licenses, and the
// projects analyzed with it. The external name is the license profile
// identifier assigned by SonarQube.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type SCALicenseProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SCALicenseProfileSpec   `json:"spec"`
	Status SCALicenseProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SCALicenseProfileList contains a list of SCALicenseProfile.
type SCALicenseProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SCALicenseProfile `json:"items"`
}

// SCALicenseProfile type metadata.
var (
	SCALicenseProfileKind             = reflect.TypeFor[SCALicenseProfile]().Name()
	SCALicenseProfileGroupKind        = schema.GroupKind{Group: Group, Kind: SCALicenseProfileKind}.String()
	SCALicenseProfileKindAPIVersion   = SCALicenseProfileKind + "." + SchemeGroupVersion.String()
	SCALicenseProfileGroupVersionKind = SchemeGroupVersion.WithKind(SCALicenseProfileKind)
)

// init registers the SCALicenseProfile resource with the Scheme.
func init() {
	register(&SCALicenseProfile{}, &SCALicenseProfileList{})
}
