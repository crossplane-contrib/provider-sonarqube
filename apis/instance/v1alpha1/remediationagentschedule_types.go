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

// RemediationAgentScheduleParameters are the configurable fields of a
// RemediationAgentSchedule.
//
// Without projectKey, projectKeyRef or projectKeySelector the resource manages
// the instance-wide schedule. With one of them it manages the schedule of that
// project, which overrides the instance-wide one. SonarQube holds at most one
// schedule per scope.
//
// +kubebuilder:validation:XValidation:rule="self.frequency != 'WEEKLY' || has(self.dayOfWeek)",message="dayOfWeek is required when frequency is WEEKLY"
// +kubebuilder:validation:XValidation:rule="self.frequency != 'DAILY' || !has(self.dayOfWeek)",message="dayOfWeek must not be set when frequency is DAILY"
// +kubebuilder:validation:XValidation:rule="!(has(self.projectKey) || has(self.projectKeyRef) || has(self.projectKeySelector)) || !(has(self.projectSelectionMode) || has(self.selectedProjectKeys) || has(self.selectedProjectKeyRefs) || has(self.selectedProjectKeySelector))",message="projectSelectionMode and selected projects are only valid on the instance-wide schedule"
// +kubebuilder:validation:XValidation:rule="(has(self.projectSelectionMode) && self.projectSelectionMode == 'SELECTED') == (has(self.selectedProjectKeys) || has(self.selectedProjectKeyRefs) || has(self.selectedProjectKeySelector))",message="selected projects must be set if, and only if, projectSelectionMode is SELECTED"
type RemediationAgentScheduleParameters struct {
	// ProjectKey is the key of the project whose schedule is managed. Leave it,
	// ProjectKeyRef and ProjectKeySelector unset to manage the instance-wide
	// schedule.
	// +optional
	// +immutable
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self != '*'",message="omit projectKey to manage the instance-wide schedule"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectKey is immutable"
	ProjectKey *string `json:"projectKey,omitempty"`

	// ProjectKeyRef references a Project to retrieve its key.
	// +optional
	ProjectKeyRef *xpv1.Reference `json:"projectKeyRef,omitempty"`

	// ProjectKeySelector selects a Project to retrieve its key.
	// +optional
	ProjectKeySelector *xpv1.Selector `json:"projectKeySelector,omitempty"`

	// Enabled turns scheduled remediation on or off.
	// +kubebuilder:validation:Required
	Enabled *bool `json:"enabled"`

	// Frequency is how often the agent runs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DAILY;WEEKLY
	Frequency string `json:"frequency"`

	// DayOfWeek is the day the agent runs on. Required when Frequency is
	// WEEKLY, and forbidden when it is DAILY.
	// +optional
	// +kubebuilder:validation:Enum=MONDAY;TUESDAY;WEDNESDAY;THURSDAY;FRIDAY;SATURDAY;SUNDAY
	DayOfWeek *string `json:"dayOfWeek,omitempty"`

	// Hour is the hour of the day, in Timezone, at which the agent runs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	Hour *int32 `json:"hour"`

	// Timezone is the IANA time zone of the schedule, such as Europe/Paris.
	// Late-initialized from SonarQube, and UTC when a project schedule is
	// created without it.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Timezone *string `json:"timezone,omitempty"`

	// MaxIssuesPerRun is the maximum number of issues a run fixes.
	// Late-initialized from SonarQube, and 5 when a project schedule is created
	// without it.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=50
	MaxIssuesPerRun *int32 `json:"maxIssuesPerRun,omitempty"`

	// MaxOpenPRs pauses scheduled runs while this many pull requests opened by
	// them are still open. -1 never pauses. Late-initialized from SonarQube,
	// and 3 when a project schedule is created without it.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == -1 || (self >= 1 && self <= 100)",message="maxOpenPRs must be between 1 and 100, or -1 for unlimited"
	MaxOpenPRs *int32 `json:"maxOpenPRs,omitempty"`

	// ProjectSelectionMode selects the projects the instance-wide schedule
	// applies to. Only valid on the instance-wide schedule, and
	// late-initialized from SonarQube.
	// +optional
	// +kubebuilder:validation:Enum=ALL;SELECTED
	ProjectSelectionMode *string `json:"projectSelectionMode,omitempty"`

	// SelectedProjectKeys are the keys of the projects the instance-wide
	// schedule applies to when ProjectSelectionMode is SELECTED. This list is
	// exhaustive: any other project is removed from the selection.
	// +optional
	// +listType=set
	// +kubebuilder:validation:MinItems=1
	SelectedProjectKeys []string `json:"selectedProjectKeys,omitempty"`

	// SelectedProjectKeyRefs references Projects to retrieve their keys.
	// +optional
	// +kubebuilder:validation:MinItems=1
	SelectedProjectKeyRefs []xpv1.Reference `json:"selectedProjectKeyRefs,omitempty"`

	// SelectedProjectKeySelector selects Projects to retrieve their keys.
	// +optional
	SelectedProjectKeySelector *xpv1.Selector `json:"selectedProjectKeySelector,omitempty"`
}

// RemediationAgentScheduleObservation are the observable fields of a
// RemediationAgentSchedule.
type RemediationAgentScheduleObservation struct {
	// ID is the identifier of the schedule: the project key, or "*" for the
	// instance-wide schedule.
	ID string `json:"id,omitempty"`
	// ProjectKey is the key of the project, empty for the instance-wide
	// schedule.
	ProjectKey string `json:"projectKey,omitempty"`
	// Enabled reports whether scheduled remediation is on.
	Enabled bool `json:"enabled"`
	// Frequency is how often the agent runs.
	Frequency string `json:"frequency,omitempty"`
	// DayOfWeek is the day a weekly schedule runs on.
	DayOfWeek string `json:"dayOfWeek,omitempty"`
	// Hour is the hour of the day at which the agent runs.
	Hour int32 `json:"hour"`
	// Timezone is the time zone of the schedule.
	Timezone string `json:"timezone,omitempty"`
	// NextRunTime is the time of the next scheduled run.
	NextRunTime *metav1.Time `json:"nextRunTime,omitempty"`
	// MaxIssuesPerRun is the maximum number of issues a run fixes.
	MaxIssuesPerRun int32 `json:"maxIssuesPerRun,omitempty"`
	// MaxOpenPRs is the open pull request limit, -1 when unlimited.
	MaxOpenPRs int32 `json:"maxOpenPRs,omitempty"`
	// ProjectSelectionMode is the project selection of the instance-wide
	// schedule.
	ProjectSelectionMode string `json:"projectSelectionMode,omitempty"`
	// SelectedProjectKeys are the projects selected by the instance-wide
	// schedule.
	SelectedProjectKeys []string `json:"selectedProjectKeys,omitempty"`
}

// A RemediationAgentScheduleSpec defines the desired state of a
// RemediationAgentSchedule.
//
// +kubebuilder:validation:XValidation:rule="(has(self.forProvider.projectKey) || has(self.forProvider.projectKeyRef) || has(self.forProvider.projectKeySelector)) == (has(oldSelf.forProvider.projectKey) || has(oldSelf.forProvider.projectKeyRef) || has(oldSelf.forProvider.projectKeySelector))",message="a schedule cannot change between instance-wide and project scope"
type RemediationAgentScheduleSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider RemediationAgentScheduleParameters `json:"forProvider"`
}

// A RemediationAgentScheduleStatus represents the observed state of a
// RemediationAgentSchedule.
type RemediationAgentScheduleStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider RemediationAgentScheduleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A RemediationAgentSchedule manages when the SonarQube Remediation Agent
// fixes the issue backlog, either instance-wide or for a single project.
// Enterprise Edition only.
//
// Deleting the instance-wide schedule disables it, as SonarQube cannot delete
// it. Deleting a project schedule makes the project follow the instance-wide
// one again.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type RemediationAgentSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemediationAgentScheduleSpec   `json:"spec"`
	Status RemediationAgentScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RemediationAgentScheduleList contains a list of RemediationAgentSchedule.
type RemediationAgentScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RemediationAgentSchedule `json:"items"`
}

var (
	// RemediationAgentScheduleKind is the kind name of a
	// RemediationAgentSchedule.
	RemediationAgentScheduleKind = reflect.TypeFor[RemediationAgentSchedule]().Name()
	// RemediationAgentScheduleGroupKind is the group-kind name of a
	// RemediationAgentSchedule.
	RemediationAgentScheduleGroupKind = schema.GroupKind{Group: Group, Kind: RemediationAgentScheduleKind}.String()
	// RemediationAgentScheduleKindAPIVersion is the kind and API version of a
	// RemediationAgentSchedule.
	RemediationAgentScheduleKindAPIVersion = RemediationAgentScheduleKind + "." + SchemeGroupVersion.String()
	// RemediationAgentScheduleGroupVersionKind is the GroupVersionKind of a
	// RemediationAgentSchedule.
	RemediationAgentScheduleGroupVersionKind = SchemeGroupVersion.WithKind(RemediationAgentScheduleKind)
)

// init registers the RemediationAgentSchedule and
// RemediationAgentScheduleList types with the SchemeBuilder.
func init() {
	register(&RemediationAgentSchedule{}, &RemediationAgentScheduleList{})
}
