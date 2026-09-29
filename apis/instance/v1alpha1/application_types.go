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

// ApplicationParameters are the configurable fields of an Application.
type ApplicationParameters struct {
	// Key is the application key. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable"
	Key string `json:"key"`

	// Name is the application display name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is the application description.
	// When unset, the description is not managed.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description *string `json:"description,omitempty"`

	// Visibility is the application visibility. Immutable after creation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=public
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Visibility is immutable"
	Visibility string `json:"visibility,omitempty"`

	// Projects is the exhaustive list of keys of the projects that are members of the application.
	// Any member project that is not in this list is removed from the application.
	// +kubebuilder:validation:Optional
	// +listType=set
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1.Project
	// +crossplane:generate:reference:refFieldName=ProjectRefs
	// +crossplane:generate:reference:selectorFieldName=ProjectSelector
	Projects []string `json:"projects,omitempty"`

	// ProjectRefs are references to Projects used to populate Projects.
	// +kubebuilder:validation:Optional
	ProjectRefs []xpv1.NamespacedReference `json:"projectRefs,omitempty"`

	// ProjectSelector selects references to Projects used to populate Projects.
	// +kubebuilder:validation:Optional
	ProjectSelector *xpv1.NamespacedSelector `json:"projectSelector,omitempty"`

	// Branches is the exhaustive list of the application branches, excluding the main branch
	// which is managed by SonarQube. Any non-main application branch that is not in this list
	// is deleted from the application.
	// +kubebuilder:validation:Optional
	// +listType=map
	// +listMapKey=name
	Branches []ApplicationBranchParameters `json:"branches,omitempty"`
}

// ApplicationBranchParameters are the configurable fields of an
// Application branch.
type ApplicationBranchParameters struct {
	// Name is the name of the application branch.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Projects maps member projects to the project branch used in this application branch.
	// This list is exhaustive: any other project in the application branch is removed from it.
	// Every project listed here must be a member of the application.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +listType=map
	// +listMapKey=project
	Projects []ApplicationBranchProjectParameters `json:"projects"`
}

// ApplicationBranchProjectParameters select the branch of a member project
// used in an Application branch.
type ApplicationBranchProjectParameters struct {
	// Project is the key of the member project.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Project string `json:"project"`

	// Branch is the name of the project branch.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Branch string `json:"branch"`
}

// ApplicationObservation are the observable fields of an Application.
type ApplicationObservation struct {
	// Key is the application key.
	Key string `json:"key,omitempty"`
	// Name is the application name.
	Name string `json:"name,omitempty"`
	// Description is the application description.
	Description string `json:"description,omitempty"`
	// Visibility is the application visibility.
	Visibility string `json:"visibility,omitempty"`
	// Tags are the tags of the application.
	Tags []string `json:"tags,omitempty"`
	// Projects are the keys of the member projects of the application.
	Projects []string `json:"projects,omitempty"`
	// Branches are the branches of the application.
	Branches []ApplicationBranchObservation `json:"branches,omitempty"`
}

// ApplicationBranchObservation is the observed state of an Application
// branch.
type ApplicationBranchObservation struct {
	// Name is the name of the application branch.
	Name string `json:"name"`
	// IsMain indicates whether this is the main branch of the application.
	IsMain bool `json:"isMain"`
	// Projects are the project branches used in this application branch.
	Projects []ApplicationBranchProjectObservation `json:"projects,omitempty"`
}

// ApplicationBranchProjectObservation is the observed project branch used
// in an Application branch.
type ApplicationBranchProjectObservation struct {
	// Project is the key of the member project.
	Project string `json:"project"`
	// Branch is the name of the project branch.
	Branch string `json:"branch,omitempty"`
	// IsMain indicates whether the project branch is the project main branch.
	IsMain bool `json:"isMain"`
}

// An ApplicationSpec defines the desired state of an Application.
type ApplicationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider ApplicationParameters `json:"forProvider"`
}

// An ApplicationStatus represents the observed state of an Application.
type ApplicationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider ApplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Application manages a SonarQube application (Enterprise Edition only).
// An application groups several existing projects under a fixed membership,
// with its own branch definitions.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Application `json:"items"`
}

// Application type metadata.
var (
	ApplicationKind             = reflect.TypeFor[Application]().Name()
	ApplicationGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationKind}.String()
	ApplicationKindAPIVersion   = ApplicationKind + "." + SchemeGroupVersion.String()
	ApplicationGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationKind)
)

// init registers the Application resource with the Scheme.
func init() {
	register(&Application{}, &ApplicationList{})
}
