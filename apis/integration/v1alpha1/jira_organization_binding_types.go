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

// JiraOrganizationBindingParameters are the configurable fields of a
// JiraOrganizationBinding.
type JiraOrganizationBindingParameters struct {
	// SonarOrganizationUUID is the identifier of the SonarQube organization
	// the Jira Cloud instance is bound to. SonarQube Server has a single,
	// implicit organization, whose identifier is the default value.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="00000000-0000-4000-0000-000000000000"
	// +kubebuilder:validation:Format=uuid
	SonarOrganizationUUID string `json:"sonarOrganizationUuid,omitempty"`

	// IsTokenShared indicates whether the Jira OAuth token of the connection
	// is shared across all users. Late initialized from SonarQube when
	// omitted.
	// +kubebuilder:validation:Optional
	IsTokenShared *bool `json:"isTokenShared,omitempty"`
}

// JiraOrganizationBindingObservation are the observable fields of a
// JiraOrganizationBinding.
type JiraOrganizationBindingObservation struct {
	// ID is the binding identifier assigned by SonarQube (used as
	// external-name).
	ID string `json:"id,omitempty"`
	// SonarOrganizationUUID is the identifier of the bound SonarQube
	// organization.
	SonarOrganizationUUID string `json:"sonarOrganizationUuid,omitempty"`
	// JiraInstanceURL is the URL of the bound Jira Cloud instance.
	JiraInstanceURL string `json:"jiraInstanceUrl,omitempty"`
	// CreatedBy is the user who established or reauthorized the connection.
	CreatedBy string `json:"createdBy,omitempty"`
	// CreatedAt is the creation date of the binding.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
	// IsTokenShared indicates whether the Jira OAuth token is shared across
	// all users.
	IsTokenShared bool `json:"isTokenShared,omitempty"`
}

// A JiraOrganizationBindingSpec defines the desired state of a
// JiraOrganizationBinding.
type JiraOrganizationBindingSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	// +kubebuilder:default={}
	ForProvider JiraOrganizationBindingParameters `json:"forProvider"`
}

// A JiraOrganizationBindingStatus represents the observed state of a
// JiraOrganizationBinding.
type JiraOrganizationBindingStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider JiraOrganizationBindingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A JiraOrganizationBinding is a managed resource that represents the binding
// between a SonarQube Enterprise instance and a Jira Cloud instance.
//
// The binding cannot be created through the API: it requires an Atlassian OAuth
// consent in a browser, from the SonarQube user interface. This resource only
// manages an existing binding, which must be imported by setting the
// crossplane.io/external-name annotation to its ID.
//
// Deleting the resource deletes the binding, along with every Jira project
// binding and the related data, and the binding cannot be restored by the
// provider. Remove "Delete" from spec.managementPolicies to keep it.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type JiraOrganizationBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraOrganizationBindingSpec   `json:"spec"`
	Status JiraOrganizationBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JiraOrganizationBindingList contains a list of JiraOrganizationBinding.
type JiraOrganizationBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []JiraOrganizationBinding `json:"items"`
}

// JiraOrganizationBinding type metadata.
var (
	JiraOrganizationBindingKind             = reflect.TypeFor[JiraOrganizationBinding]().Name()
	JiraOrganizationBindingGroupKind        = schema.GroupKind{Group: Group, Kind: JiraOrganizationBindingKind}.String()
	JiraOrganizationBindingKindAPIVersion   = JiraOrganizationBindingKind + "." + SchemeGroupVersion.String()
	JiraOrganizationBindingGroupVersionKind = SchemeGroupVersion.WithKind(JiraOrganizationBindingKind)
)

// init registers the JiraOrganizationBinding resource with the Scheme.
func init() {
	register(&JiraOrganizationBinding{}, &JiraOrganizationBindingList{})
}
