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

// DashboardParameters are the configurable fields of a Dashboard. A
// dashboard targets exactly one project or one portfolio, which cannot be
// changed after creation.
// +kubebuilder:validation:XValidation:rule="(has(self.projectKey) || has(self.projectKeyRef) || has(self.projectKeySelector)) != (has(self.portfolioKey) || has(self.portfolioKeyRef) || has(self.portfolioKeySelector))",message="exactly one of a project (projectKey, projectKeyRef or projectKeySelector) or a portfolio (portfolioKey, portfolioKeyRef or portfolioKeySelector) must be set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.projectKey) || has(self.projectKey)",message="projectKey cannot be unset once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.portfolioKey) || has(self.portfolioKey)",message="portfolioKey cannot be unset once set"
type DashboardParameters struct {
	// Name is the dashboard name. It identifies the dashboard among those of
	// its project or portfolio: a dashboard is not created if one with the
	// same name already exists there.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=300
	Name string `json:"name"`

	// Description is the dashboard description.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=500
	Description *string `json:"description,omitempty"`

	// Layout is the dashboard layout, in the format SonarQube stores it
	// (a JSON document). It is sent as written. JSON layouts are compared by
	// content, so formatting and key order do not matter.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Layout string `json:"layout"`

	// ProjectKey is the key of the project the dashboard belongs to.
	// Immutable.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=400
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ProjectKey is immutable."
	ProjectKey *string `json:"projectKey,omitempty"`

	// ProjectKeyRef resolves projectKey from a Project managed resource.
	// +kubebuilder:validation:Optional
	ProjectKeyRef *xpv1.Reference `json:"projectKeyRef,omitempty"`

	// ProjectKeySelector selects a Project managed resource to resolve
	// projectKey.
	// +kubebuilder:validation:Optional
	ProjectKeySelector *xpv1.Selector `json:"projectKeySelector,omitempty"`

	// PortfolioKey is the key of the portfolio the dashboard belongs to.
	// Immutable.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=400
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="PortfolioKey is immutable."
	PortfolioKey *string `json:"portfolioKey,omitempty"`

	// PortfolioKeyRef resolves portfolioKey from a Portfolio managed resource.
	// +kubebuilder:validation:Optional
	PortfolioKeyRef *xpv1.Reference `json:"portfolioKeyRef,omitempty"`

	// PortfolioKeySelector selects a Portfolio managed resource to resolve
	// portfolioKey.
	// +kubebuilder:validation:Optional
	PortfolioKeySelector *xpv1.Selector `json:"portfolioKeySelector,omitempty"`
}

// DashboardObservation are the observable fields of a Dashboard.
type DashboardObservation struct {
	// ID is the dashboard identifier assigned by SonarQube (used as
	// external-name).
	ID string `json:"id,omitempty"`
	// Name is the dashboard name.
	Name string `json:"name,omitempty"`
	// Description is the dashboard description.
	Description string `json:"description,omitempty"`
	// Layout is the dashboard layout as stored in SonarQube.
	Layout string `json:"layout,omitempty"`
	// ResourceType is the type of the target: project or portfolio.
	ResourceType string `json:"resourceType,omitempty"`
	// ResourceID is the SonarQube identifier of the target project or
	// portfolio.
	ResourceID string `json:"resourceId,omitempty"`
	// CreatedAt is the creation timestamp reported by SonarQube.
	CreatedAt int64 `json:"createdAt,omitempty"`
	// UpdatedAt is the last update timestamp reported by SonarQube.
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

// A DashboardSpec defines the desired state of a Dashboard.
type DashboardSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider DashboardParameters `json:"forProvider"`
}

// A DashboardStatus represents the observed state of a Dashboard.
type DashboardStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider DashboardObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Dashboard manages a SonarQube custom dashboard of a project or a
// portfolio (Enterprise Edition only).
//
// An existing dashboard is never taken over: to manage it, set the
// crossplane.io/external-name annotation to its ID.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Dashboard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardSpec   `json:"spec"`
	Status DashboardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DashboardList contains a list of Dashboard.
type DashboardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Dashboard `json:"items"`
}

// Dashboard type metadata.
var (
	DashboardKind             = reflect.TypeFor[Dashboard]().Name()
	DashboardGroupKind        = schema.GroupKind{Group: Group, Kind: DashboardKind}.String()
	DashboardKindAPIVersion   = DashboardKind + "." + SchemeGroupVersion.String()
	DashboardGroupVersionKind = SchemeGroupVersion.WithKind(DashboardKind)
)

// init registers the Dashboard resource with the Scheme.
func init() {
	register(&Dashboard{}, &DashboardList{})
}
