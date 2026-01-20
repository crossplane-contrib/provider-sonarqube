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

// QualityProfileParameters are the configurable fields of a QualityProfile.
type QualityProfileParameters struct {
	// Name is the Display name of the Quality Profile.
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Language defines the programming language of the Quality Profile.
	// +kubebuilder:validation:Enum=azureresourcemanager;cloudformation;cs;css;docker;flex;go;ipynb;java;js;json;jsp;kotlin;kubernetes;php;py;ruby;rust;scala;scala;secrets;terraform;text;ts;vbnet;web;xml;yaml
	// +kubebuilder:validation:Required
	Language string `json:"language"`
	// Default indicates whether this Quality Profile is the default one.
	// +kubebuilder:validation:Optional
	Default *bool `json:"default,omitempty"`
	// Rules is the list of rules to be activated in the Quality Profile.
	// +kubebuilder:validation:Optional
	Rules []QualityProfileRuleParameters `json:"rules,omitempty"`
}

// QualityProfileObservation are the observable fields of a QualityProfile.
type QualityProfileObservation struct {
	// ActiveDeprecatedRuleCount represents the number of active deprecated rules in the Quality Profile.
	ActiveDeprecatedRuleCount int64 `json:"activeDeprecatedRuleCount"`
	// ActiveRuleCount represents the number of active rules in the Quality Profile.
	ActiveRuleCount int64 `json:"activeRuleCount"`
	// IsBuiltIn indicates whether the Quality Profile is built-in.
	IsBuiltIn bool `json:"isBuiltIn"`
	// IsDefault indicates whether the Quality Profile is the default one.
	IsDefault bool `json:"isDefault"`
	// IsInherited indicates whether the Quality Profile is inherited.
	IsInherited bool `json:"isInherited"`
	// Key is the unique key (identifier) of the Quality Profile.
	Key string `json:"key"`
	// Language is the programming language to which the Quality Profile applies.
	Language string `json:"language"`
	// LanguageName is the display name of the programming language to which the Quality Profile applies.
	LanguageName string `json:"languageName"`
	// LastUsed is the last time the Quality Profile was used.
	LastUsed *metav1.Time `json:"lastUsed,omitempty"`
	// Name is the display name of the Quality Profile.
	Name string `json:"name"`
	// ProjectCount is the number of projects associated with the Quality Profile.
	ProjectCount int64 `json:"projectCount"`
	// RulesUpdatedAt is the last time the rules in the Quality Profile were updated.
	RulesUpdatedAt *metav1.Time `json:"rulesUpdatedAt,omitempty"`
	// Rules represents the list of rules activated in the Quality Profile.
	Rules []QualityProfileRuleObservation `json:"rules,omitempty"`
}

// QualityProfileRuleParameters are the configurable fields of a QualityProfile Rule.
type QualityProfileRuleParameters struct {
	// Impacts overrides severities for the rule. Cannot be used as the same time as 'severity'.
	// If used together with 'severity', 'impacts' will take precedence.
	// WARNING: This field is currently not reconciled against manual edits, as it is not possible to read its value via SonarQube API.
	// +kubebuilder:validation:Optional
	Impacts *map[string]string `json:"impacts,omitempty"`
	// Parameters.
	// WARNING: This field is currently not reconciled against manual edits, as it is not possible to read the activated values via SonarQube API.
	// +kubebuilder:validation:Optional
	Parameters *map[string]string `json:"params,omitempty"`
	// Prioritized marks activated rule as prioritized, so all corresponding Issues will have to be fixed.
	// WARNING: This field is currently not reconciled against manual edits, as it is not possible to read its value via SonarQube API.
	// +kubebuilder:validation:Optional
	Prioritized *bool `json:"prioritized,omitempty"`
	// Rule is the unique key (identifier) of the rule to be activated in the Quality Profile.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Rule string `json:"rule"`
	// Severity. Cannot be used as the same time as 'impacts'.
	// If used together with 'impacts', 'impacts' will take precedence.
	// WARNING: This field is currently not reconciled against manual edits, as it is not possible to read the activated value via SonarQube API.
	// +kubebuilder:validation:Enum=INFO;MINOR;MAJOR;CRITICAL;BLOCKER
	// +kubebuilder:validation:Optional
	Severity *string `json:"severity,omitempty"`
}

// QualityProfileRuleObservation are the observable fields of a QualityProfile Rule.
type QualityProfileRuleObservation struct {
	Key                        string                          `json:"key"`
	Repo                       string                          `json:"repo"`
	Name                       string                          `json:"name"`
	NoteLogin                  string                          `json:"noteLogin"`
	MdNote                     string                          `json:"mdNote"`
	HTMLNote                   string                          `json:"htmlNote"`
	CreatedAt                  *metav1.Time                    `json:"createdAt,omitempty"`
	UpdatedAt                  *metav1.Time                    `json:"updatedAt,omitempty"`
	HTMLDesc                   string                          `json:"htmlDesc"`
	Severity                   string                          `json:"severity"`
	Status                     string                          `json:"status"`
	InternalKey                string                          `json:"internalKey"`
	IsTemplate                 bool                            `json:"isTemplate"`
	Tags                       []string                        `json:"tags,omitempty"`
	TemplateKey                string                          `json:"templateKey"`
	SysTags                    []string                        `json:"sysTags,omitempty"`
	Language                   string                          `json:"language"`
	LanguageName               string                          `json:"languageName"`
	Scope                      string                          `json:"scope"`
	IsExternal                 bool                            `json:"isExternal"`
	Type                       string                          `json:"type"`
	CleanCodeAttributeCategory string                          `json:"cleanCodeAttributeCategory"`
	CleanCodeAttribute         string                          `json:"cleanCodeAttribute"`
	Impacts                    []QualityProfileRuleImpact      `json:"impacts,omitempty"`
	DescriptionSections        []QualityProfileRuleDescription `json:"descriptionSections,omitempty"`
	Parameters                 []QualityProfileRuleParameter   `json:"parameters,omitempty"`
}

type QualityProfileRuleDescription struct {
	Content string                                       `json:"content,omitempty"`
	Context QualityProfileRuleDescriptionSectionsContext `json:"context,omitempty"`
	Key     string                                       `json:"key,omitempty"`
}

type QualityProfileRuleDescriptionSectionsContext struct {
	DisplayName string `json:"displayName,omitempty"`
	Key         string `json:"key,omitempty"`
}

type QualityProfileRuleImpact struct {
	Severity        string `json:"severity,omitempty"`
	SoftwareQuality string `json:"softwareQuality,omitempty"`
}

type QualityProfileRuleParameter struct {
	DefaultValue string `json:"defaultValue,omitempty"`
	Desc         string `json:"desc,omitempty"`
	Key          string `json:"key,omitempty"`
}

// A QualityProfileSpec defines the desired state of a QualityProfile.
type QualityProfileSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              QualityProfileParameters `json:"forProvider"`
}

// A QualityProfileStatus represents the observed state of a QualityProfile.
type QualityProfileStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          QualityProfileObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A QualityProfile is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type QualityProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QualityProfileSpec   `json:"spec"`
	Status QualityProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QualityProfileList contains a list of QualityProfile
type QualityProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QualityProfile `json:"items"`
}

// QualityProfile type metadata.
var (
	QualityProfileKind             = reflect.TypeOf(QualityProfile{}).Name()
	QualityProfileGroupKind        = schema.GroupKind{Group: Group, Kind: QualityProfileKind}.String()
	QualityProfileKindAPIVersion   = QualityProfileKind + "." + SchemeGroupVersion.String()
	QualityProfileGroupVersionKind = SchemeGroupVersion.WithKind(QualityProfileKind)
)

func init() {
	SchemeBuilder.Register(&QualityProfile{}, &QualityProfileList{})
}
