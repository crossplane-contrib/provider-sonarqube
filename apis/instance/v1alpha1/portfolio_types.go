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
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.parent) || has(self.parent)",message="parent cannot be unset once set"
// +kubebuilder:validation:XValidation:rule="!has(self.projects) || size(self.projects) == 0 || (has(self.selectionMode) && self.selectionMode == 'MANUAL')",message="projects can only be set when selectionMode is MANUAL"
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
	// Only applies to root portfolios: sub-portfolios inherit the visibility of their root portfolio.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=public
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Visibility is immutable"
	Visibility string `json:"visibility,omitempty"`

	// SelectionMode describes how projects are selected for this portfolio.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=NONE
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

	// Parent is the key of the parent portfolio, making this portfolio a sub-portfolio of it.
	// The parent can be changed to move the sub-portfolio to another portfolio, but it cannot be
	// unset once set: a sub-portfolio cannot be turned into a root portfolio.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Parent *string `json:"parent,omitempty"`

	// ParentRef is a reference to a Portfolio used to set Parent.
	// +kubebuilder:validation:Optional
	ParentRef *xpv1.Reference `json:"parentRef,omitempty"`

	// ParentSelector selects a reference to a Portfolio used to set Parent.
	// +kubebuilder:validation:Optional
	ParentSelector *xpv1.Selector `json:"parentSelector,omitempty"`

	// Projects is the exhaustive list of projects manually selected in the portfolio.
	// Any selected project that is not in this list is removed from the portfolio.
	// When unset, the selected projects are not managed.
	// Only allowed when SelectionMode is MANUAL.
	// +kubebuilder:validation:Optional
	Projects *[]PortfolioProjectParameters `json:"projects,omitempty"`

	// Applications is the exhaustive list of applications added to the portfolio.
	// Any application of the portfolio that is not in this list is removed from it.
	// When unset, the applications of the portfolio are not managed.
	// +kubebuilder:validation:Optional
	Applications *[]PortfolioApplicationParameters `json:"applications,omitempty"`

	// SubPortfolios is the exhaustive list of keys of the existing portfolios referenced in this
	// portfolio. Any referenced portfolio that is not in this list is removed from the portfolio.
	// When unset, and no SubPortfolioRefs or SubPortfolioSelector is set, the referenced
	// portfolios are not managed.
	// Sub-portfolios created under this portfolio are managed through the parent of their own
	// Portfolio resource and are not affected by this list.
	// +kubebuilder:validation:Optional
	// +listType=set
	SubPortfolios *[]string `json:"subPortfolios,omitempty"`

	// SubPortfolioRefs are references to Portfolios used to populate SubPortfolios.
	// +kubebuilder:validation:Optional
	SubPortfolioRefs []xpv1.Reference `json:"subPortfolioRefs,omitempty"`

	// SubPortfolioSelector selects references to Portfolios used to populate SubPortfolios.
	// +kubebuilder:validation:Optional
	SubPortfolioSelector *xpv1.Selector `json:"subPortfolioSelector,omitempty"`
}

// PortfolioProjectParameters select a project, and optionally its branches,
// in a portfolio.
// +kubebuilder:validation:XValidation:rule="has(self.key) || has(self.keyRef) || has(self.keySelector)",message="one of key, keyRef or keySelector must be set"
type PortfolioProjectParameters struct {
	// Key is the project key.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Key *string `json:"key,omitempty"`

	// KeyRef is a reference to a Project used to set Key.
	// +kubebuilder:validation:Optional
	KeyRef *xpv1.Reference `json:"keyRef,omitempty"`

	// KeySelector selects a reference to a Project used to set Key.
	// +kubebuilder:validation:Optional
	KeySelector *xpv1.Selector `json:"keySelector,omitempty"`

	// Branches is the exhaustive list of the project branches selected in the portfolio.
	// When empty, the selected branches are not managed: SonarQube selects the project main
	// branch when the project is added. List the main branch explicitly to select it alongside
	// other branches.
	// +kubebuilder:validation:Optional
	// +listType=set
	Branches []string `json:"branches,omitempty"`
}

// PortfolioApplicationParameters select an application, and optionally its
// branches, in a portfolio.
// +kubebuilder:validation:XValidation:rule="has(self.key) || has(self.keyRef) || has(self.keySelector)",message="one of key, keyRef or keySelector must be set"
type PortfolioApplicationParameters struct {
	// Key is the application key.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Key *string `json:"key,omitempty"`

	// KeyRef is a reference to an Application used to set Key.
	// +kubebuilder:validation:Optional
	KeyRef *xpv1.Reference `json:"keyRef,omitempty"`

	// KeySelector selects a reference to an Application used to set Key.
	// +kubebuilder:validation:Optional
	KeySelector *xpv1.Selector `json:"keySelector,omitempty"`

	// Branches is the exhaustive list of the application branches selected in the portfolio.
	// When empty, the selected branches are not managed: SonarQube selects the application main
	// branch when the application is added. List the main branch explicitly to select it
	// alongside other branches.
	// +kubebuilder:validation:Optional
	// +listType=set
	Branches []string `json:"branches,omitempty"`
}

// PortfolioMemberObservation is the observed state of a project or an
// application selected in a portfolio.
type PortfolioMemberObservation struct {
	// Key is the project or application key.
	Key string `json:"key"`
	// Branches are the explicitly selected branches, if any.
	Branches []string `json:"branches,omitempty"`
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
	// Parent is the key of the parent portfolio, when the portfolio is a sub-portfolio of the
	// desired parent. SonarQube does not return the parent of a portfolio, so it is observed by
	// looking up the sub-portfolios of the desired parent only.
	Parent string `json:"parent,omitempty"`
	// Projects are the projects manually selected in the portfolio.
	Projects []PortfolioMemberObservation `json:"projects,omitempty"`
	// Applications are the applications added to the portfolio.
	Applications []PortfolioMemberObservation `json:"applications,omitempty"`
	// SubPortfolios are the keys of the portfolios referenced in the portfolio.
	SubPortfolios []string `json:"subPortfolios,omitempty"`
	// LocalSubPortfolios are the keys of the sub-portfolios created under the portfolio.
	LocalSubPortfolios []string `json:"localSubPortfolios,omitempty"`
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
	register(&Portfolio{}, &PortfolioList{})
}
