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

// QualityProfileUsergroupAssociationParameters are the configurable fields
// of a QualityProfileUsergroupAssociation resource.
// Exactly one of GroupName or Login must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.groupName) || has(self.login)) && !(has(self.groupName) && has(self.login))",message="exactly one of groupName or login must be set"
type QualityProfileUsergroupAssociationParameters struct {
	// QualityProfile is the display name of the Quality Profile the group
	// or user is associated with.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="QualityProfile is immutable."
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityProfile
	// +crossplane:generate:reference:extractor=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityProfileName()
	QualityProfile string `json:"qualityProfile"`

	// QualityProfileRef references a QualityProfile resource to populate
	// QualityProfile.
	// +kubebuilder:validation:Optional
	QualityProfileRef *xpv1.NamespacedReference `json:"qualityProfileRef,omitempty"`

	// QualityProfileSelector selects a QualityProfile resource to populate
	// QualityProfile.
	// +kubebuilder:validation:Optional
	QualityProfileSelector *xpv1.NamespacedSelector `json:"qualityProfileSelector,omitempty"`

	// Language is the programming language of the Quality Profile.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Language is immutable."
	// +kubebuilder:validation:MinLength=1
	Language string `json:"language"`

	// GroupName is the name of the group granted edit rights on the Quality
	// Profile. Mutually exclusive with login. Immutable once set.
	// Principal names are referenced directly: this resource lives in the
	// instance API group, and instance types cannot import the iam types
	// without creating a cycle in the generated reference resolvers.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GroupName is immutable."
	GroupName *string `json:"groupName,omitempty"`

	// Login is the login of the user granted edit rights on the Quality
	// Profile. Mutually exclusive with groupName. Immutable once set.
	// Principal names are referenced directly: this resource lives in the
	// instance API group, and instance types cannot import the iam types
	// without creating a cycle in the generated reference resolvers.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Login is immutable."
	Login *string `json:"login,omitempty"`
}

// QualityProfileUsergroupAssociationObservation are the observable fields
// of a QualityProfileUsergroupAssociation resource.
type QualityProfileUsergroupAssociationObservation struct {
	// QualityProfile is the display name of the associated Quality Profile.
	QualityProfile string `json:"qualityProfile,omitempty"`

	// Language is the programming language of the associated Quality
	// Profile.
	Language string `json:"language,omitempty"`

	// GroupName is the associated group name, if the principal is a group.
	GroupName string `json:"groupName,omitempty"`

	// Login is the associated user login, if the principal is a user.
	Login string `json:"login,omitempty"`
}

// A QualityProfileUsergroupAssociationSpec defines the desired state of a
// QualityProfileUsergroupAssociation.
type QualityProfileUsergroupAssociationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	// ForProvider represents the desired state of the association.
	ForProvider QualityProfileUsergroupAssociationParameters `json:"forProvider"`
}

// A QualityProfileUsergroupAssociationStatus represents the observed state
// of a QualityProfileUsergroupAssociation.
type QualityProfileUsergroupAssociationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	// AtProvider represents the observed state of the association.
	AtProvider QualityProfileUsergroupAssociationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A QualityProfileUsergroupAssociation associates a SonarQube group or
// user with edit rights on a specific Quality Profile.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type QualityProfileUsergroupAssociation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QualityProfileUsergroupAssociationSpec   `json:"spec"`
	Status QualityProfileUsergroupAssociationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QualityProfileUsergroupAssociationList contains a list of
// QualityProfileUsergroupAssociation.
type QualityProfileUsergroupAssociationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []QualityProfileUsergroupAssociation `json:"items"`
}

// QualityProfileUsergroupAssociation type metadata.
var (
	QualityProfileUsergroupAssociationKind             = reflect.TypeFor[QualityProfileUsergroupAssociation]().Name()
	QualityProfileUsergroupAssociationGroupKind        = schema.GroupKind{Group: Group, Kind: QualityProfileUsergroupAssociationKind}.String()
	QualityProfileUsergroupAssociationKindAPIVersion   = QualityProfileUsergroupAssociationKind + "." + SchemeGroupVersion.String()
	QualityProfileUsergroupAssociationGroupVersionKind = SchemeGroupVersion.WithKind(QualityProfileUsergroupAssociationKind)
)

// init registers the QualityProfileUsergroupAssociation resource with the
// Scheme.
func init() {
	SchemeBuilder.Register(&QualityProfileUsergroupAssociation{}, &QualityProfileUsergroupAssociationList{})
}
