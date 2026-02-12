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

// SettingsParameters represent the desired state of SonarQube Settings.
type SettingsParameters struct {
	// Component is an identifier to specify the scope of the settings.
	// It can be projects, applications, portfolios or subportfolios scoped.
	// If set, this will be enforced for all settings.
	// WARNING: Do not use multiple Settings resources with the same component value as they will conflict with each other. It is recommended to use a single Settings resource per component.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Component is immutable."
	Component *string `json:"component,omitempty"`
	// Settings is the map of settings to be applied. The key is the unique identifier of the setting and the value is the value of the setting.
	// WARNING: Removing a setting from this map will NOT reset it to its default value in SonarQube.
	// If you want to make sure a setting is reset to its default value, you need to leave it there and properly delete the Settings resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinProperties=1
	Settings map[string]SettingParameters `json:"settings"`
}

// SettingParameters represent the desired state of a single SonarQube Setting.
type SettingParameters struct {
	// Value is the value of the setting. The format of the value depends on the type of the setting. It can be a string, a number, a boolean or a JSON object.
	// This field must be set if "Values" and "fieldValues" are not set.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=4000
	Value *string `json:"value,omitempty"`
	// Values is used for multi valued settings.
	// This field must be set if "Value" and "fieldValues" are not set.
	// +kubebuilder:validation:Optional
	Values *[]string `json:"values,omitempty"`
	// FieldValues is used for multi valued settings with predefined fields.
	// This field must be set if "Value" and "Values" are not set.
	// +kubebuilder:validation:Optional
	FieldValues *map[string]string `json:"fieldValues,omitempty"`
}

// SettingsObservation are the observable fields of a Settings.
type SettingsObservation struct {
	// Settings is the map of settings that have been applied. The key is the unique identifier of the setting and the value is the value of the setting. The format of the value depends on the type of the setting. It can be a string, a number, a boolean or a JSON object.
	Settings map[string]SettingObservation `json:"settings,omitempty"`
}

// SettingObservation are the observable fields of a single SonarQube Setting.
type SettingObservation struct {
	// Value is the value of the setting. The format of the value depends on the type of the setting. It can be a string, a number, a boolean or a JSON object.
	Value string `json:"value,omitempty"`
	// Values is used for multi valued settings.
	Values []string `json:"values,omitempty"`
	// FieldValues is used for multi valued settings with predefined fields.
	FieldValues map[string]string `json:"fieldValues,omitempty"`
}

// A SettingsSpec defines the desired state of a Settings.
type SettingsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider SettingsParameters `json:"forProvider"`
}

// A SettingsStatus represents the observed state of a Settings.
type SettingsStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider SettingsObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Settings manages SonarQube settings configuration.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Settings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SettingsSpec   `json:"spec"`
	Status SettingsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SettingsList contains a list of Settings.
type SettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Settings `json:"items"`
}

// Settings type metadata.
var (
	SettingsKind             = reflect.TypeFor[Settings]().Name()
	SettingsGroupKind        = schema.GroupKind{Group: Group, Kind: SettingsKind}.String()
	SettingsKindAPIVersion   = SettingsKind + "." + SchemeGroupVersion.String()
	SettingsGroupVersionKind = SchemeGroupVersion.WithKind(SettingsKind)
)

func init() {
	SchemeBuilder.Register(&Settings{}, &SettingsList{})
}
