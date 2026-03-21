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

// RuleParameters are the configurable fields of a Rule.
type RuleParameters struct {
	// Key is the unique identifier for the custom rule (required).
	// This must be an unprefixed custom key (e.g., "myRule"), not a language-prefixed key like "language:key".
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Key is immutable."
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Key string `json:"key"`
	// Name is the display name of the rule.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// TemplateKey is the key of the template rule from which to create the custom rule.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Template Key is immutable."
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	TemplateKey string `json:"templateKey"`
	// MarkdownDescription is the Markdown-formatted description of the rule.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	MarkdownDescription string `json:"markdownDescription,omitempty"`
	// CleanCodeAttribute represents the Clean Code Attribute associated with the rule.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="CleanCodeAttribute is immutable."
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=CONVENTIONAL;FORMATTED;IDENTIFIABLE;CLEAR;COMPLETE;EFFICIENT;LOGICAL;DISTINCT;FOCUSED;MODULAR;TESTED;LAWFUL;RESPECTFUL;TRUSTWORTHY
	CleanCodeAttribute *string `json:"cleanCodeAttribute,omitempty"`
	// Impacts is a map of software quality to severity (e.g., MAINTAINABILITY: HIGH, SECURITY: LOW).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:EachKey:Enum=MAINTAINABILITY;RELIABILITY;SECURITY
	// +kubebuilder:validation:EachValue:Enum=INFO;LOW;MEDIUM;HIGH;BLOCKER
	Impacts *map[string]string `json:"impacts,omitempty"`
	// MarkdownNote is the optional note in Markdown format.
	// +kubebuilder:validation:Optional
	MarkdownNote *string `json:"markdownNote,omitempty"`
	// Parameters is a map of parameter key to its configuration for the rule.
	// +kubebuilder:validation:Optional
	Parameters *map[string]RuleParameterConfiguration `json:"parameters,omitempty"`
	// RemediationFnBaseEffort is the base effort of the remediation function (e.g., '1d').
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	RemediationFnBaseEffort *string `json:"remediationFnBaseEffort,omitempty"`
	// RemediationFnType is the type of the remediation function.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=CONSTANT_ISSUE;LINEAR;LINEAR_OFFSET
	RemediationFnType *string `json:"remediationFnType,omitempty"`
	// RemediationFyGapMultiplier is the gap multiplier of the remediation function (e.g., '2min').
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	RemediationFyGapMultiplier *string `json:"remediationFyGapMultiplier,omitempty"`
	// Severity is the rule severity.
	// +kubebuilder:validation:Enum=INFO;MINOR;MAJOR;CRITICAL;BLOCKER
	// +kubebuilder:validation:Optional
	Severity *string `json:"severity,omitempty"`
	// Status is the status of the rule.
	// +kubebuilder:validation:Enum=READY;DEPRECATED;BETA
	// +kubebuilder:validation:Optional
	Status *string `json:"status,omitempty"`
	// Type is the rule type.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
	// +kubebuilder:validation:Enum=CODE_SMELL;BUG;VULNERABILITY;SECURITY_HOTSPOT
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`
	// Tags is a list of tags to associate with the rule.
	// Use empty slice to remove current tags. Tags are not changed if parameter is not set.
	Tags *[]string `json:"tags,omitempty"`
}

// RuleParameterConfiguration represents the configuration of a parameter for a rule.
type RuleParameterConfiguration struct {
	// DefaultValue is the default value of the parameter if unspecified in implementation.
	// +kubebuilder:validation:Optional
	DefaultValue string `json:"defaultValue"`
	// HTMLDesc is the HTML-formatted description of the parameter.
	// +kubebuilder:validation:Optional
	HTMLDesc string `json:"htmlDesc"`
	// Type is the data type of the parameter (STRING, TEXT, BOOLEAN, INTEGER, FLOAT).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=STRING;TEXT;BOOLEAN;INTEGER;FLOAT
	Type string `json:"type"`
}

// RuleObservation are the observable fields of a Rule.
type RuleObservation struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Key string `json:"key"`
	// +kubebuilder:validation:Optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
	// +kubebuilder:validation:Optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
	// +kubebuilder:validation:Optional
	RemFnType string `json:"remFnType"`
	// +kubebuilder:validation:Optional
	HTMLDesc string `json:"htmlDesc"`
	// +kubebuilder:validation:Optional
	HTMLNote string `json:"htmlNote"`
	// +kubebuilder:validation:Optional
	MdNote string `json:"mdNote"`
	// +kubebuilder:validation:Optional
	NoteLogin string `json:"noteLogin"`
	// +kubebuilder:validation:Optional
	CleanCodeAttribute string `json:"cleanCodeAttribute"`
	// +kubebuilder:validation:Optional
	InternalKey string `json:"internalKey"`
	// +kubebuilder:validation:Optional
	RemFnGapMultiplier string `json:"remFnGapMultiplier"`
	// +kubebuilder:validation:Optional
	RemFnBaseEffort string `json:"remFnBaseEffort"`
	// +kubebuilder:validation:Optional
	DefaultRemFnBaseEffort string `json:"defaultRemFnBaseEffort"`
	// +kubebuilder:validation:Optional
	Language string `json:"lang"`
	// +kubebuilder:validation:Optional
	LanguageName string `json:"langName"`
	// +kubebuilder:validation:Optional
	CleanCodeAttributeCategory string `json:"cleanCodeAttributeCategory"`
	// +kubebuilder:validation:Optional
	GapDescription string `json:"gapDescription"`
	// +kubebuilder:validation:Optional
	Repo string `json:"repo"`
	// +kubebuilder:validation:Optional
	Scope string `json:"scope"`
	// +kubebuilder:validation:Optional
	Severity string `json:"severity"`
	// +kubebuilder:validation:Optional
	Status string `json:"status"`
	// +kubebuilder:validation:Optional
	DefaultRemFnType string `json:"defaultRemFnType"`
	// +kubebuilder:validation:Optional
	DefaultRemFnGapMultiplier string `json:"defaultRemFnGapMultiplier"`
	// +kubebuilder:validation:Optional
	TemplateKey string `json:"templateKey"`
	// +kubebuilder:validation:Optional
	Type string `json:"type"`
	// +kubebuilder:validation:Optional
	Impacts map[string]string `json:"impacts"`
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags"`
	// +kubebuilder:validation:Optional
	SysTags []string `json:"sysTags"`
	// +kubebuilder:validation:Optional
	Params []RuleParameterObservation `json:"params"`
	// +kubebuilder:validation:Optional
	DescriptionSections []RuleDescriptionSectionObservation `json:"descriptionSections"`
	// +kubebuilder:validation:Optional
	IsTemplate bool `json:"isTemplate"`
	// +kubebuilder:validation:Optional
	IsExternal bool `json:"isExternal"`
	// +kubebuilder:validation:Optional
	RemFnOverloaded bool `json:"remFnOverloaded"`
	// +kubebuilder:validation:Optional
	Template bool `json:"template"`
}

// RuleParameterObservation represents a parameter that can be configured for a rule.
type RuleParameterObservation struct {
	// DefaultValue is the default value of the parameter.
	// +kubebuilder:validation:Optional
	DefaultValue string `json:"defaultValue"`
	// HTMLDesc is the HTML-formatted description of the parameter.
	// +kubebuilder:validation:Optional
	HTMLDesc string `json:"htmlDesc"`
	// Desc is the plain text description of the parameter.
	// +kubebuilder:validation:Optional
	Desc string `json:"desc"`
	// Key is the unique identifier of the parameter.
	// +kubebuilder:validation:Optional
	Key string `json:"key"`
	// Type is the data type of the parameter (STRING, TEXT, BOOLEAN, INTEGER, FLOAT).
	// +kubebuilder:validation:Optional
	Type string `json:"type"`
}

// RuleDescriptionSectionObservation represents a section of a rule's description.
type RuleDescriptionSectionObservation struct {
	// Content is the HTML content of the section.
	// +kubebuilder:validation:Optional
	Content string `json:"content"`
	// Context provides additional context for the section.
	// +kubebuilder:validation:Optional
	Context RuleDescriptionContextObservation `json:"context"`
	// Key is the unique identifier of the section.
	// +kubebuilder:validation:Optional
	Key string `json:"key"`
}

// RuleDescriptionContextObservation provides context for a description section.
type RuleDescriptionContextObservation struct {
	// DisplayName is the human-readable name of the context.
	// +kubebuilder:validation:Optional
	DisplayName string `json:"displayName"`
	// Key is the unique identifier of the context.
	// +kubebuilder:validation:Optional
	Key string `json:"key"`
}

// A RuleSpec defines the desired state of a Rule.
type RuleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider RuleParameters `json:"forProvider"`
}

// A RuleStatus represents the observed state of a Rule.
type RuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider RuleObservation `json:"atProvider"`
}

// +kubebuilder:object:root=true

// A Rule is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RuleSpec   `json:"spec"`
	Status RuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleList contains a list of Rule.
type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Rule `json:"items"`
}

// Rule type metadata.
var (
	RuleKind             = reflect.TypeFor[Rule]().Name()
	RuleGroupKind        = schema.GroupKind{Group: Group, Kind: RuleKind}.String()
	RuleKindAPIVersion   = RuleKind + "." + SchemeGroupVersion.String()
	RuleGroupVersionKind = SchemeGroupVersion.WithKind(RuleKind)
)

func init() {
	SchemeBuilder.Register(&Rule{}, &RuleList{})
}
