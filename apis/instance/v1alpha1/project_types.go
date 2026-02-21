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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// ProjectParameters are the configurable fields of a Project.
type ProjectParameters struct {
	// Key is the unique key of the project.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=400
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable."
	Key string `json:"key"`
	// Name is the display name of the project.
	// WARNING: This field is immutable because it cannot be changed using the SonarQube API
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=500
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`
	// Description is the description of the project.
	// WARNING: This field is currently not implemented in the SonarQube API, and any value set in this field will be ignored. This field is reserved for future use when the SonarQube API supports it.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description *string `json:"description,omitempty"`
	// Visibility is the visibility of the project. Can be either "public" or "private".
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=public;private
	Visibility *string `json:"visibility,omitempty"`
	// Tags are the list of tags associated with the project.
	// +kubebuilder:validation:Optional
	Tags *[]string `json:"tags,omitempty"`
	// Links are the list of links associated with the project.
	// +kubebuilder:validation:Optional
	Links []ProjectLinkParameters `json:"links,omitempty"`
	// DefaultBranch is the default branch of the project.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	DefaultBranch *string `json:"defaultBranch,omitempty"`
	// NewCodePeriod is the new code definition of the project.
	// +kubebuilder:validation:Optional
	NewCodePeriod *ProjectNewCodePeriodParameters `json:"newCodePeriod,omitempty"`
	// Branches is the map of authorized branches of the project.
	// Any branch that is not in this list will be considered unauthorized and will be deleted by SonarQube.
	// +kubebuilder:validation:Optional
	Branches map[string]*ProjectNewCodePeriodParameters `json:"branches,omitempty"`
	// QualityGateName is the name of the quality gate to be associated with the project.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	QualityGateName *string `json:"qualityGateName,omitempty"`
	// QualityGateNameRef is a reference to a QualityGate to retrieve its name and associate it with the project.
	// +kubebuilder:validation:Optional
	QualityGateNameRef *xpv1.NamespacedReference `json:"qualityGateNameRef,omitempty"`
	// QualityGateNameSelector selects reference to a QualityGate to retrieve its name and associate it with the project.
	// +kubebuilder:validation:Optional
	QualityGateNameSelector *xpv1.NamespacedSelector `json:"qualityGateNameSelector,omitempty"`
	// QualityProfiles is a map of language:quality profile to be associated with the project.
	// The key of the map is the language of the quality profile to be associated with the project, and the value is the reference to the quality profile.
	// +kubebuilder:validation:Optional
	QualityProfiles map[string]ProjectQualityProfileReference `json:"qualityProfiles,omitempty"`
}

// ProjectLinkParameters represent the parameters of a link associated with a project.
type ProjectLinkParameters struct {
	// ID is the unique identifier of the link. This field is used for updating and deleting links, and is not required when creating a new link.
	// +kubebuilder:validation:Optional
	ID *string `json:"id,omitempty"`
	// Name is the name of the link.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	Name string `json:"name"`
	// URL is the URL of the link.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	URL string `json:"url"`
}

// ProjectNewCodePeriodParameters represent the parameters of the new code definition of a project.
type ProjectNewCodePeriodParameters struct {
	// Type is the type of the new code definition. Can be either "PREVIOUS_VERSION" or "NUMBER_OF_DAYS" or "REFERENCE_BRANCH" for projects.
	// Can be "SPECIFIC_ANALYSIS" for branches.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=PREVIOUS_VERSION;NUMBER_OF_DAYS;REFERENCE_BRANCH;SPECIFIC_ANALYSIS
	Type string `json:"type"`
	// Value is the value of the new code definition. If the type is "previous_version", this field should be empty. If the type is "reference_branch", this field should be the name of the reference branch.
	// +kubebuilder:validation:Optional
	Value *string `json:"value,omitempty"`
}

// ProjectQualityProfileReference represent the reference to a quality profile to be associated with the project.
type ProjectQualityProfileReference struct {
	// Id is the name of the quality profile to be associated with the project.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Id *string `json:"id,omitempty"`
	// QualityProfileIdRef is a reference to a QualityProfile to retrieve its id and associate it with the project.
	// +kubebuilder:validation:Optional
	IdRef *xpv1.NamespacedReference `json:"idRef,omitempty"`
	// QualityProfileIdSelector selects reference to a QualityProfile to retrieve its id and associate it with the project.
	// +kubebuilder:validation:Optional
	IdSelector *xpv1.NamespacedSelector `json:"idSelector,omitempty"`
}

// ProjectObservation are the observable fields of a Project.
type ProjectObservation struct {
	// Key is the project key.
	Key string `json:"key"`
	// Name is the project name.
	Name string `json:"name"`
	// Qualifier is the project qualifier.
	Qualifier string `json:"qualifier"`
	// Visibility is the project visibility.
	Visibility string `json:"visibility"`
	// LastAnalysisDate is the date of the last analysis.
	LastAnalysisDate *metav1.Time `json:"lastAnalysisDate,omitempty"`
	// Revision is the last analysis revision.
	Revision string `json:"revision"`
	// ProjectUuid is the project UUID.
	Uuid string `json:"uuid"`
	// Managed indicates if the project is managed.
	Managed bool `json:"managed"`
	// ProjectLinks is the list of links associated with the project.
	Links map[string]ProjectLinkObservation `json:"links,omitempty"`
	// DefaultBranch is the default branch of the project.
	DefaultBranch string `json:"defaultBranch"`
	// ProjectBranches is the list of branches associated with the project.
	Branches map[string]ProjectBranchObservation `json:"branches,omitempty"`
	// NewCodePeriod is the new code definition of the project.
	NewCodePeriod ProjectNewCodePeriodObservation `json:"newCodePeriod,omitempty"`
	// QualityGateName is the name of the quality gate associated with the project.
	QualityGateName string `json:"qualityGateName,omitempty"`
	// QualityProfiles is a map of language:quality profile associated with the project.
	// The key of the map is the language of the quality profile associated with the project, and the value is the observed state of the quality profile.
	QualityProfiles map[string]ProjectQualityProfileObservation `json:"qualityProfiles,omitempty"`
}

// ProjectLinkObservation represent the observed state of a link associated with a project.
type ProjectLinkObservation struct {
	// ID is the unique identifier of the link.
	ID string `json:"id"`
	// Name is the display name of the link.
	Name string `json:"name"`
	// Type is the type of the link (e.g., "homepage", "ci", "issue", "scm").
	Type string `json:"type"`
	// URL is the target URL of the link.
	URL string `json:"url"`
}

// ProjectQualityProfileObservation represent the observed state of a quality profile associated with a project.
type ProjectQualityProfileObservation struct {
	// Id is the unique identifier of the quality profile.
	Id string `json:"id"`
	// Name is the name of the quality profile.
	Name string `json:"name"`
}

// ProjectBranchObservation represent the observed state of a branch of a project.
type ProjectBranchObservation struct {
	// AnalysisDate is the date of the last analysis.
	AnalysisDate *metav1.Time `json:"analysisDate,omitempty"`
	// BranchID is the unique identifier of the branch.
	ID string `json:"id"`
	// ExcludedFromPurge indicates whether the branch is excluded from automatic purge.
	ExcludedFromPurge bool `json:"excludedFromPurge"`
	// IsMain indicates whether this is the main branch.
	IsMain bool `json:"isMain"`
	// Name is the name of the branch.
	Name string `json:"name"`
	// Status is the status of the branch.
	Status ProjectBranchStatusObservation `json:"status,omitempty"`
	// Type is the type of the branch.
	Type string `json:"type"`
	// NewCodePeriod is the new code definition of the branch.
	NewCodePeriod ProjectNewCodePeriodObservation `json:"newCodePeriod,omitempty"`
}

// ProjectNewCodePeriodObservation represent the observed state of the new code period of a project.
type ProjectNewCodePeriodObservation struct {
	// EffectiveValue is the effective value of the new code period.
	EffectiveValue string `json:"effectiveValue,omitempty"`
	// Inherited indicates whether the value is inherited from a parent.
	Inherited bool `json:"inherited"`
	// Type is the type of the new code period.
	Type string `json:"type"`
	// Value is the value of the new code period.
	Value string `json:"value,omitempty"`
}

// ProjectBranchStatusObservation represents the observed status of a branch of a project.
type ProjectBranchStatusObservation struct {
	// QualityGateStatus is the quality gate status of the branch.
	QualityGateStatus string `json:"qualityGateStatus,omitempty"`
}

// A ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider ProjectParameters `json:"forProvider"`
}

// A ProjectStatus represents the observed state of a Project.
type ProjectStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider ProjectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Project is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Project `json:"items"`
}

// Project type metadata.
var (
	ProjectKind             = reflect.TypeFor[Project]().Name()
	ProjectGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectKind}.String()
	ProjectKindAPIVersion   = ProjectKind + "." + SchemeGroupVersion.String()
	ProjectGroupVersionKind = SchemeGroupVersion.WithKind(ProjectKind)
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
