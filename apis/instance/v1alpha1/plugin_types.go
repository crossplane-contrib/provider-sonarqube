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

// PluginParameters are the configurable fields of a Plugin.
type PluginParameters struct {
	// Key is the plugin key in the SonarQube marketplace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable"
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// AutoUpdate indicates if the plugin should be automatically updated when a new version is available.
	// Defaults to true if unset
	// +kubebuilder:default=true
	AutoUpdate *bool `json:"autoUpdate,omitempty"`
}

// PluginObservation are the observable fields of a Plugin.
type PluginObservation struct {
	// Description is the plugin description.
	Description string `json:"description,omitempty"`
	// EditionBundled indicates if the plugin is bundled with an edition.
	EditionBundled bool `json:"editionBundled"`
	// Filename is the plugin filename.
	Filename string `json:"filename,omitempty"`
	// Hash is the file hash.
	Hash string `json:"hash,omitempty"`
	// HomepageURL is the plugin homepage URL.
	HomepageURL string `json:"homepageUrl,omitempty"`
	// ImplementationBuild is the implementation build.
	ImplementationBuild string `json:"implementationBuild,omitempty"`
	// IssueTrackerURL is the issue tracker URL.
	IssueTrackerURL string `json:"issueTrackerUrl,omitempty"`
	// Key is the plugin key.
	Key string `json:"key,omitempty"`
	// License is the plugin license.
	License string `json:"license,omitempty"`
	// Name is the plugin name.
	Name string `json:"name,omitempty"`
	// OrganizationName is the organization name.
	OrganizationName string `json:"organizationName,omitempty"`
	// OrganizationURL is the organization URL.
	OrganizationURL string `json:"organizationUrl,omitempty"`
	// RequiredForLanguages is the list of languages requiring this plugin.
	RequiredForLanguages []string `json:"requiredForLanguages,omitempty"`
	// SonarLintSupported indicates if SonarLint is supported.
	SonarLintSupported bool `json:"sonarLintSupported"`
	// UpdatedAt is the timestamp when the plugin was updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
	// Version is the plugin version.
	Version string `json:"version,omitempty"`
	// IsLatest indicates if the installed plugin is the latest
	// version available in the marketplace.
	IsLatest bool `json:"isLatest"`
	// Pending is true when the plugin installation has been queued
	// but SonarQube has not yet restarted to apply it.
	Pending bool `json:"pending"`
}

// A PluginSpec defines the desired state of a Plugin.
type PluginSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this Plugin.
	ForProvider PluginParameters `json:"forProvider"`
}

// A PluginStatus represents the observed state of a Plugin.
type PluginStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state of this Plugin.
	AtProvider PluginObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Plugin manages a SonarQube plugin installation.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="PENDING",type="boolean",JSONPath=".status.atProvider.pending"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of this Plugin.
	Spec PluginSpec `json:"spec"`
	// Status represents the observed state of this Plugin.
	Status PluginStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PluginList contains a list of Plugin.
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Plugin instances.
	Items []Plugin `json:"items"`
}

// Plugin type metadata.
var (
	PluginKind             = reflect.TypeFor[Plugin]().Name()
	PluginGroupKind        = schema.GroupKind{Group: Group, Kind: PluginKind}.String()
	PluginKindAPIVersion   = PluginKind + "." + SchemeGroupVersion.String()
	PluginGroupVersionKind = SchemeGroupVersion.WithKind(PluginKind)
)

// init registers the Plugin resource with the Scheme.
func init() {
	SchemeBuilder.Register(&Plugin{}, &PluginList{})
}
