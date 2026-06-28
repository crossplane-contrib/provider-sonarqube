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

// PortfolioParameters are the configurable fields of a Portfolio.
type PortfolioParameters struct {
	// Key is the portfolio key. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable"
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// Name is the portfolio display name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is the optional portfolio description.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Visibility is the portfolio visibility. Immutable after creation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=public
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Visibility is immutable"
	Visibility string `json:"visibility,omitempty"`

	// SelectionMode describes how projects are selected for this portfolio.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=MANUAL
	// +kubebuilder:validation:Enum=MANUAL;NONE;REGEXP;REMAINING;TAGS
	SelectionMode string `json:"selectionMode,omitempty"`

	// Regexp is the regular expression used for project key matching when SelectionMode is REGEXP.
	// +kubebuilder:validation:Optional
	Regexp string `json:"regexp,omitempty"`

	// Tags is a comma-separated list of project tags when SelectionMode is TAGS.
	// +kubebuilder:validation:Optional
	Tags string `json:"tags,omitempty"`

	// Branch is the branch used in matched projects when SelectionMode is REGEXP, REMAINING, or TAGS.
	// +kubebuilder:validation:Optional
	Branch string `json:"branch,omitempty"`
}

// PortfolioObservation are the observable fields of a Portfolio.
type PortfolioObservation struct {
	// Key is the portfolio key as stored in SonarQube.
	Key string `json:"key,omitempty"`
	// Name is the portfolio name.
	Name string `json:"name,omitempty"`
	// Description is the portfolio description.
	Description string `json:"description,omitempty"`
	// Qualifier is the component qualifier (VW for root portfolio, SVW for sub-portfolio).
	Qualifier string `json:"qualifier,omitempty"`
	// Visibility is the portfolio visibility.
	Visibility string `json:"visibility,omitempty"`
	// SelectionMode is the current project selection mode.
	SelectionMode string `json:"selectionMode,omitempty"`
}

// A PortfolioSpec defines the desired state of a Portfolio.
type PortfolioSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider PortfolioParameters `json:"forProvider"`
}

// A PortfolioStatus represents the observed state of a Portfolio.
type PortfolioStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider PortfolioObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Portfolio manages a SonarQube portfolio (Enterprise Edition only).
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Portfolio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PortfolioSpec   `json:"spec"`
	Status PortfolioStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PortfolioList contains a list of Portfolio.
type PortfolioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Portfolio `json:"items"`
}

// Portfolio type metadata.
var (
	PortfolioKind             = reflect.TypeFor[Portfolio]().Name()
	PortfolioGroupKind        = schema.GroupKind{Group: Group, Kind: PortfolioKind}.String()
	PortfolioKindAPIVersion   = PortfolioKind + "." + SchemeGroupVersion.String()
	PortfolioGroupVersionKind = SchemeGroupVersion.WithKind(PortfolioKind)
)

// init registers the Portfolio resource with the Scheme.
func init() {
	SchemeBuilder.Register(&Portfolio{}, &PortfolioList{})
}
