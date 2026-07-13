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
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityGate
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
	// AlmBinding binds the project to a repository on a DevOps platform (GitHub, GitLab, Azure DevOps, Bitbucket, or Bitbucket Cloud), enabling Pull Request decoration.
	// +kubebuilder:validation:Optional
	AlmBinding *ProjectALMBindingParameters `json:"almBinding,omitempty"`
}

// ProjectALMBindingParameters configure a Project's binding to a repository on
// a DevOps platform. Exactly one of GitHub, GitLab, Azure, Bitbucket, or
// BitbucketCloud must be set.
// +kubebuilder:validation:XValidation:rule="[has(self.gitHub),has(self.gitLab),has(self.azure),has(self.bitbucket),has(self.bitbucketCloud)].filter(x,x).size()==1",message="exactly one of gitHub, gitLab, azure, bitbucket, bitbucketCloud must be set"
type ProjectALMBindingParameters struct {
	// Monorepo indicates the bound repository is part of a monorepo.
	// +kubebuilder:validation:Optional
	Monorepo bool `json:"monorepo,omitempty"`
	// GitHub binds the project to a GitHub repository.
	// +kubebuilder:validation:Optional
	GitHub *ProjectGitHubBindingParameters `json:"gitHub,omitempty"`
	// GitLab binds the project to a GitLab project.
	// +kubebuilder:validation:Optional
	GitLab *ProjectGitLabBindingParameters `json:"gitLab,omitempty"`
	// Azure binds the project to an Azure DevOps repository.
	// +kubebuilder:validation:Optional
	Azure *ProjectAzureBindingParameters `json:"azure,omitempty"`
	// Bitbucket binds the project to a Bitbucket Server repository.
	// +kubebuilder:validation:Optional
	Bitbucket *ProjectBitbucketBindingParameters `json:"bitbucket,omitempty"`
	// BitbucketCloud binds the project to a Bitbucket Cloud repository.
	// +kubebuilder:validation:Optional
	BitbucketCloud *ProjectBitbucketCloudBindingParameters `json:"bitbucketCloud,omitempty"`
}

// ProjectGitHubBindingParameters are the configurable fields of a Project's
// GitHub binding.
type ProjectGitHubBindingParameters struct {
	// AlmSettingKey is the key of the ALMGitHub setting to bind to. Must match the spec.forProvider.key of an existing ALMGitHub resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AlmSettingKey string `json:"almSettingKey"`
	// Repository is the GitHub repository (e.g. "organization/repository").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Repository string `json:"repository"`
	// SummaryCommentEnabled enables the analysis summary in the pull request discussion tab.
	// +kubebuilder:validation:Optional
	SummaryCommentEnabled *bool `json:"summaryCommentEnabled,omitempty"`
}

// ProjectGitLabBindingParameters are the configurable fields of a Project's
// GitLab binding.
type ProjectGitLabBindingParameters struct {
	// AlmSettingKey is the key of the ALMGitLab setting to bind to. Must match the spec.forProvider.key of an existing ALMGitLab resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AlmSettingKey string `json:"almSettingKey"`
	// Repository is the GitLab project ID.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`
}

// ProjectAzureBindingParameters are the configurable fields of a Project's
// Azure DevOps binding.
type ProjectAzureBindingParameters struct {
	// AlmSettingKey is the key of the ALMAzure setting to bind to. Must match the spec.forProvider.key of an existing ALMAzure resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AlmSettingKey string `json:"almSettingKey"`
	// ProjectName is the Azure DevOps project name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ProjectName string `json:"projectName"`
	// RepositoryName is the Azure DevOps repository name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	RepositoryName string `json:"repositoryName"`
	// InlineAnnotationsEnabled enables inline annotations during Pull Request decoration.
	// +kubebuilder:validation:Optional
	InlineAnnotationsEnabled *bool `json:"inlineAnnotationsEnabled,omitempty"`
}

// ProjectBitbucketBindingParameters are the configurable fields of a
// Project's Bitbucket Server binding.
type ProjectBitbucketBindingParameters struct {
	// AlmSettingKey is the key of the ALMBitbucket setting to bind to. Must match the spec.forProvider.key of an existing ALMBitbucket resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AlmSettingKey string `json:"almSettingKey"`
	// Repository is the Bitbucket Server repository key.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`
	// Slug is the Bitbucket Server repository slug.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Slug string `json:"slug"`
}

// ProjectBitbucketCloudBindingParameters are the configurable fields of a
// Project's Bitbucket Cloud binding.
type ProjectBitbucketCloudBindingParameters struct {
	// AlmSettingKey is the key of the ALMBitbucketCloud setting to bind to. Must match the spec.forProvider.key of an existing ALMBitbucketCloud resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AlmSettingKey string `json:"almSettingKey"`
	// Repository is the Bitbucket Cloud repository key.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`
}

// ProjectLinkParameters represent the parameters of a link
// associated with a project.
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

// ProjectNewCodePeriodParameters represent the parameters of the
// new code definition of a project.
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

// ProjectQualityProfileReference represent the reference to a quality profile
// to be associated with the project.
type ProjectQualityProfileReference struct {
	// Id is the name of the quality profile to be associated with the project.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.QualityProfile
	Id *string `json:"id,omitempty"`
	// IdRef is a reference to a QualityProfile to retrieve its id and associate it with the project.
	// +kubebuilder:validation:Optional
	IdRef *xpv1.NamespacedReference `json:"idRef,omitempty"`
	// IdSelector selects reference to a QualityProfile to retrieve its id and associate it with the project.
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
	// AlmBinding is the observed state of the project's DevOps platform binding.
	AlmBinding *ProjectALMBindingObservation `json:"almBinding,omitempty"`
}

// ProjectALMBindingObservation is the observed state of a Project's DevOps
// platform binding.
type ProjectALMBindingObservation struct {
	// Alm is the type of DevOps platform (azure, bitbucket, bitbucketcloud, github, gitlab).
	Alm string `json:"alm,omitempty"`
	// Key is the DevOps platform setting key.
	Key string `json:"key,omitempty"`
	// Repository is the bound repository identifier.
	Repository string `json:"repository,omitempty"`
	// RepositoryURL is the URL of the bound repository.
	RepositoryURL string `json:"repositoryUrl,omitempty"`
	// Slug is the bound repository slug (Bitbucket Server only).
	Slug string `json:"slug,omitempty"`
	// Monorepo indicates the bound repository is part of a monorepo.
	Monorepo bool `json:"monorepo,omitempty"`
	// InlineAnnotationsEnabled indicates inline annotations are enabled (Azure only).
	InlineAnnotationsEnabled bool `json:"inlineAnnotationsEnabled,omitempty"`
	// SummaryCommentEnabled indicates the pull request summary comment is enabled.
	SummaryCommentEnabled bool `json:"summaryCommentEnabled,omitempty"`
}

// ProjectLinkObservation represent the observed state of a link associated with
// a project.
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

// ProjectQualityProfileObservation represent the observed state of a
// quality profile associated with a project.
type ProjectQualityProfileObservation struct {
	// Id is the unique identifier of the quality profile.
	Id string `json:"id"`
	// Name is the name of the quality profile.
	Name string `json:"name"`
}

// ProjectBranchObservation represent the observed state of a
// branch of a project.
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

// ProjectNewCodePeriodObservation represent the observed state of the new code
// period of a project.
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

// ProjectBranchStatusObservation represents the observed status of a
// branch of a project.
type ProjectBranchStatusObservation struct {
	// QualityGateStatus is the quality gate status of the branch.
	QualityGateStatus string `json:"qualityGateStatus,omitempty"`
}

// A ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ProjectParameters `json:"forProvider"`
}

// A ProjectStatus represents the observed state of a Project.
type ProjectStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

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

// init registers the Project resource with the Scheme.
func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
