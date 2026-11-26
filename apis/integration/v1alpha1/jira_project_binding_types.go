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

// JiraProjectBindingParameters are the configurable fields of a
// JiraProjectBinding.
// +kubebuilder:validation:XValidation:rule="has(self.projectKey) || has(self.projectKeyRef) || has(self.projectKeySelector)",message="one of projectKey, projectKeyRef or projectKeySelector must be set"
// +kubebuilder:validation:XValidation:rule="has(self.jiraOrganizationBindingId) || has(self.jiraOrganizationBindingIdRef) || has(self.jiraOrganizationBindingIdSelector)",message="one of jiraOrganizationBindingId, jiraOrganizationBindingIdRef or jiraOrganizationBindingIdSelector must be set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.projectKey) || has(self.projectKey)",message="projectKey cannot be unset once set"
type JiraProjectBindingParameters struct {
	// ProjectKey is the key of the SonarQube project to bind. Immutable.
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

	// JiraOrganizationBindingID is the identifier of the Jira organization
	// binding the project binding belongs to.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	JiraOrganizationBindingID *string `json:"jiraOrganizationBindingId,omitempty"`

	// JiraOrganizationBindingIDRef resolves jiraOrganizationBindingId from a
	// JiraOrganizationBinding managed resource.
	// +kubebuilder:validation:Optional
	JiraOrganizationBindingIDRef *xpv1.Reference `json:"jiraOrganizationBindingIdRef,omitempty"`

	// JiraOrganizationBindingIDSelector selects a JiraOrganizationBinding
	// managed resource to resolve jiraOrganizationBindingId.
	// +kubebuilder:validation:Optional
	JiraOrganizationBindingIDSelector *xpv1.Selector `json:"jiraOrganizationBindingIdSelector,omitempty"`

	// JiraProjectKey is the key of the Jira project to bind the SonarQube
	// project to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	JiraProjectKey string `json:"jiraProjectKey"`
}

// JiraProjectBindingObservation are the observable fields of a
// JiraProjectBinding.
type JiraProjectBindingObservation struct {
	// ID is the binding identifier assigned by SonarQube (used as
	// external-name).
	ID string `json:"id,omitempty"`
	// SonarProjectID is the identifier of the bound SonarQube project.
	SonarProjectID string `json:"sonarProjectId,omitempty"`
	// JiraOrganizationBindingID is the identifier of the Jira organization
	// binding.
	JiraOrganizationBindingID string `json:"jiraOrganizationBindingId,omitempty"`
	// JiraProjectKey is the key of the bound Jira project.
	JiraProjectKey string `json:"jiraProjectKey,omitempty"`
}

// A JiraProjectBindingSpec defines the desired state of a JiraProjectBinding.
type JiraProjectBindingSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider JiraProjectBindingParameters `json:"forProvider"`
}

// A JiraProjectBindingStatus represents the observed state of a
// JiraProjectBinding.
type JiraProjectBindingStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider JiraProjectBindingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A JiraProjectBinding is a managed resource that represents the binding
// between a SonarQube Enterprise project and a Jira project. It requires the
// instance to be connected to Jira (see JiraOrganizationBinding).
//
// A project has at most one Jira binding. An existing one is never taken over:
// to manage it, set the crossplane.io/external-name annotation to its ID.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type JiraProjectBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraProjectBindingSpec   `json:"spec"`
	Status JiraProjectBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JiraProjectBindingList contains a list of JiraProjectBinding.
type JiraProjectBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []JiraProjectBinding `json:"items"`
}

// JiraProjectBinding type metadata.
var (
	JiraProjectBindingKind             = reflect.TypeFor[JiraProjectBinding]().Name()
	JiraProjectBindingGroupKind        = schema.GroupKind{Group: Group, Kind: JiraProjectBindingKind}.String()
	JiraProjectBindingKindAPIVersion   = JiraProjectBindingKind + "." + SchemeGroupVersion.String()
	JiraProjectBindingGroupVersionKind = SchemeGroupVersion.WithKind(JiraProjectBindingKind)
)

// init registers the JiraProjectBinding resource with the Scheme.
func init() {
	register(&JiraProjectBinding{}, &JiraProjectBindingList{})
}
