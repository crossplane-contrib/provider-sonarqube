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

// QualityGateUsergroupAssociationParameters are the configurable fields of a
// QualityGateUsergroupAssociation resource.
// GateName must be set, directly or through a reference or selector.
// Exactly one of GroupName or Login must be set, directly or through a
// reference or selector.
// +kubebuilder:validation:XValidation:rule="has(self.gateName) || has(self.gateNameRef) || has(self.gateNameSelector)",message="one of gateName, gateNameRef or gateNameSelector must be set"
// +kubebuilder:validation:XValidation:rule="(has(self.groupName) || has(self.groupNameRef) || has(self.groupNameSelector)) != (has(self.login) || has(self.loginRef) || has(self.loginSelector))",message="exactly one of groupName (or groupNameRef/groupNameSelector) or login (or loginRef/loginSelector) must be set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.gateName) || has(self.gateName)",message="gateName cannot be unset once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.groupName) || has(self.groupName)",message="groupName cannot be unset once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.login) || has(self.login)",message="login cannot be unset once set"
type QualityGateUsergroupAssociationParameters struct {
	// GateName is the name of the Quality Gate the group or user is associated
	// with.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GateName is immutable."
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Optional
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityGate
	GateName string `json:"gateName,omitempty"`

	// GateNameRef references a QualityGate resource to populate GateName.
	// +kubebuilder:validation:Optional
	GateNameRef *xpv1.NamespacedReference `json:"gateNameRef,omitempty"`

	// GateNameSelector selects a QualityGate resource to populate GateName.
	// +kubebuilder:validation:Optional
	GateNameSelector *xpv1.NamespacedSelector `json:"gateNameSelector,omitempty"`

	// GroupName is the name of the group granted edit rights on the Quality
	// Gate. Mutually exclusive with login. Immutable once set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GroupName is immutable."
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.Group
	// +crossplane:generate:reference:extractor=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.GroupName()
	GroupName *string `json:"groupName,omitempty"`

	// GroupNameRef references a Group resource to populate GroupName.
	// +kubebuilder:validation:Optional
	GroupNameRef *xpv1.NamespacedReference `json:"groupNameRef,omitempty"`

	// GroupNameSelector selects a Group resource to populate GroupName.
	// +kubebuilder:validation:Optional
	GroupNameSelector *xpv1.NamespacedSelector `json:"groupNameSelector,omitempty"`

	// Login is the login of the user granted edit rights on the Quality Gate.
	// Mutually exclusive with groupName. Immutable once set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Login is immutable."
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.User
	// +crossplane:generate:reference:extractor=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.UserLogin()
	Login *string `json:"login,omitempty"`

	// LoginRef references a User resource to populate Login.
	// +kubebuilder:validation:Optional
	LoginRef *xpv1.NamespacedReference `json:"loginRef,omitempty"`

	// LoginSelector selects a User resource to populate Login.
	// +kubebuilder:validation:Optional
	LoginSelector *xpv1.NamespacedSelector `json:"loginSelector,omitempty"`
}

// QualityGateUsergroupAssociationObservation are the observable fields of a
// QualityGateUsergroupAssociation resource.
type QualityGateUsergroupAssociationObservation struct {
	// GateName is the name of the associated Quality Gate.
	GateName string `json:"gateName,omitempty"`

	// GroupName is the associated group name, if the principal is a group.
	GroupName string `json:"groupName,omitempty"`

	// Login is the associated user login, if the principal is a user.
	Login string `json:"login,omitempty"`
}

// A QualityGateUsergroupAssociationSpec defines the desired state of a
// QualityGateUsergroupAssociation.
type QualityGateUsergroupAssociationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	// ForProvider represents the desired state of the association.
	ForProvider QualityGateUsergroupAssociationParameters `json:"forProvider"`
}

// A QualityGateUsergroupAssociationStatus represents the observed state of a
// QualityGateUsergroupAssociation.
type QualityGateUsergroupAssociationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	// AtProvider represents the observed state of the association.
	AtProvider QualityGateUsergroupAssociationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A QualityGateUsergroupAssociation associates a SonarQube group or user
// with edit rights on a specific Quality Gate.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type QualityGateUsergroupAssociation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QualityGateUsergroupAssociationSpec   `json:"spec"`
	Status QualityGateUsergroupAssociationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QualityGateUsergroupAssociationList contains a list of
// QualityGateUsergroupAssociation.
type QualityGateUsergroupAssociationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []QualityGateUsergroupAssociation `json:"items"`
}

// QualityGateUsergroupAssociation type metadata.
var (
	QualityGateUsergroupAssociationKind             = reflect.TypeFor[QualityGateUsergroupAssociation]().Name()
	QualityGateUsergroupAssociationGroupKind        = schema.GroupKind{Group: APIGroup, Kind: QualityGateUsergroupAssociationKind}.String()
	QualityGateUsergroupAssociationKindAPIVersion   = QualityGateUsergroupAssociationKind + "." + SchemeGroupVersion.String()
	QualityGateUsergroupAssociationGroupVersionKind = SchemeGroupVersion.WithKind(QualityGateUsergroupAssociationKind)
)

// init registers the QualityGateUsergroupAssociation resource with the
// Scheme.
func init() {
	SchemeBuilder.Register(&QualityGateUsergroupAssociation{}, &QualityGateUsergroupAssociationList{})
}
