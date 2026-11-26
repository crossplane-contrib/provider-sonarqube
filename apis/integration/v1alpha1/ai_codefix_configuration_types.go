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

// AI CodeFix enablement states.
const (
	// AICodeFixDisabled disables AI CodeFix on every project.
	AICodeFixDisabled = "DISABLED"
	// AICodeFixEnabledForAllProjects enables AI CodeFix on every project.
	AICodeFixEnabledForAllProjects = "ENABLED_FOR_ALL_PROJECTS"
	// AICodeFixEnabledForSomeProjects enables AI CodeFix on the selected
	// projects only.
	AICodeFixEnabledForSomeProjects = "ENABLED_FOR_SOME_PROJECTS"
)

// AICodeFixConfigurationParameters are the configurable fields of an
// AICodeFixConfiguration.
//
// +kubebuilder:validation:XValidation:rule="self.enablement == 'ENABLED_FOR_SOME_PROJECTS' || ((!has(self.enabledProjectKeys) || size(self.enabledProjectKeys) == 0) && (!has(self.enabledProjectKeyRefs) || size(self.enabledProjectKeyRefs) == 0) && !has(self.enabledProjectKeySelector))",message="enabledProjectKeys, enabledProjectKeyRefs and enabledProjectKeySelector are only allowed when enablement is ENABLED_FOR_SOME_PROJECTS"
type AICodeFixConfigurationParameters struct {
	// Enablement is the AI CodeFix enablement scope.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DISABLED;ENABLED_FOR_ALL_PROJECTS;ENABLED_FOR_SOME_PROJECTS
	Enablement string `json:"enablement"`

	// EnabledProjectKeys is the exhaustive list of keys of the projects AI
	// CodeFix is enabled for. Only allowed when Enablement is
	// ENABLED_FOR_SOME_PROJECTS.
	// +optional
	// +listType=set
	EnabledProjectKeys []string `json:"enabledProjectKeys,omitempty"`

	// EnabledProjectKeyRefs are references to Projects used to populate
	// EnabledProjectKeys.
	// +optional
	EnabledProjectKeyRefs []xpv1.Reference `json:"enabledProjectKeyRefs,omitempty"`

	// EnabledProjectKeySelector selects references to Projects used to
	// populate EnabledProjectKeys.
	// +optional
	EnabledProjectKeySelector *xpv1.Selector `json:"enabledProjectKeySelector,omitempty"`
}

// AICodeFixConfigurationObservation are the observable fields of an
// AICodeFixConfiguration.
type AICodeFixConfigurationObservation struct {
	// Enablement is the observed AI CodeFix enablement scope.
	Enablement string `json:"enablement,omitempty"`
	// EnabledProjectKeys are the keys of the projects AI CodeFix is enabled
	// for, when Enablement is ENABLED_FOR_SOME_PROJECTS.
	EnabledProjectKeys []string `json:"enabledProjectKeys,omitempty"`
}

// An AICodeFixConfigurationSpec defines the desired state of an
// AICodeFixConfiguration.
type AICodeFixConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider AICodeFixConfigurationParameters `json:"forProvider"`
}

// An AICodeFixConfigurationStatus represents the observed state of an
// AICodeFixConfiguration.
type AICodeFixConfigurationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider AICodeFixConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An AICodeFixConfiguration manages the instance-wide AI CodeFix enablement
// of a SonarQube instance: whether AI CodeFix is disabled, enabled for every
// project, or enabled for selected projects only. The LLM it uses is
// configured with an LLMProvider mapped to the AI_CODEFIX capability.
//
// It is a singleton: manage at most one AICodeFixConfiguration per SonarQube
// instance. Deleting it disables AI CodeFix. Requires SonarQube Enterprise
// Edition or above; the underlying API is marked internal by SonarQube.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLEMENT",type="string",JSONPath=".status.atProvider.enablement"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type AICodeFixConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AICodeFixConfigurationSpec   `json:"spec"`
	Status AICodeFixConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AICodeFixConfigurationList contains a list of AICodeFixConfiguration.
type AICodeFixConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AICodeFixConfiguration `json:"items"`
}

// AICodeFixConfiguration type metadata.
var (
	AICodeFixConfigurationKind             = reflect.TypeFor[AICodeFixConfiguration]().Name()
	AICodeFixConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: AICodeFixConfigurationKind}.String()
	AICodeFixConfigurationKindAPIVersion   = AICodeFixConfigurationKind + "." + SchemeGroupVersion.String()
	AICodeFixConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(AICodeFixConfigurationKind)
)

// init registers the AICodeFixConfiguration resource with the Scheme.
func init() {
	register(&AICodeFixConfiguration{}, &AICodeFixConfigurationList{})
}
