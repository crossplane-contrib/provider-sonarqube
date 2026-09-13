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
// Exactly one of GroupName or Login must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.groupName) || has(self.login)) && !(has(self.groupName) && has(self.login))",message="exactly one of groupName or login must be set"
type QualityGateUsergroupAssociationParameters struct {
	// GateName is the name of the Quality Gate the group or user is associated
	// with.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GateName is immutable."
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityGate
	// +crossplane:generate:reference:extractor=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityGateName()
	GateName string `json:"gateName"`

	// GateNameRef references a QualityGate resource to populate GateName.
	// +kubebuilder:validation:Optional
	GateNameRef *xpv1.NamespacedReference `json:"gateNameRef,omitempty"`

	// GateNameSelector selects a QualityGate resource to populate GateName.
	// +kubebuilder:validation:Optional
	GateNameSelector *xpv1.NamespacedSelector `json:"gateNameSelector,omitempty"`

	// GroupName is the name of the group granted edit rights on the Quality
	// Gate. Mutually exclusive with login. Immutable once set.
	// Principal names are referenced directly: this resource lives in the
	// instance API group, and instance types cannot import the iam types
	// without creating a cycle in the generated reference resolvers.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GroupName is immutable."
	GroupName *string `json:"groupName,omitempty"`

	// Login is the login of the user granted edit rights on the Quality Gate.
	// Mutually exclusive with groupName. Immutable once set.
	// Principal names are referenced directly: this resource lives in the
	// instance API group, and instance types cannot import the iam types
	// without creating a cycle in the generated reference resolvers.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Login is immutable."
	Login *string `json:"login,omitempty"`
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
	QualityGateUsergroupAssociationGroupKind        = schema.GroupKind{Group: Group, Kind: QualityGateUsergroupAssociationKind}.String()
	QualityGateUsergroupAssociationKindAPIVersion   = QualityGateUsergroupAssociationKind + "." + SchemeGroupVersion.String()
	QualityGateUsergroupAssociationGroupVersionKind = SchemeGroupVersion.WithKind(QualityGateUsergroupAssociationKind)
)

// init registers the QualityGateUsergroupAssociation resource with the
// Scheme.
func init() {
	SchemeBuilder.Register(&QualityGateUsergroupAssociation{}, &QualityGateUsergroupAssociationList{})
}
