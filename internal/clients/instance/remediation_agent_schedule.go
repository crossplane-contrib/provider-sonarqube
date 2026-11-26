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

package instance

import (
	"cmp"
	"context"
	"net/http"
	"slices"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	gocmp "github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// GlobalRemediationAgentScheduleID is the identifier SonarQube gives to the
	// instance-wide scheduled agent configuration.
	GlobalRemediationAgentScheduleID = "*"

	// selectionModeSelected restricts the instance-wide schedule to a list of
	// projects.
	selectionModeSelected = "SELECTED"

	// defaultScheduleTimezone is the time zone SonarQube seeds its
	// instance-wide schedule with. Like the two limits below, it completes a
	// project schedule created without it, as the API requires the field.
	defaultScheduleTimezone = "UTC"
	// defaultMaxIssuesPerRun is the seeded limit of issues per run.
	defaultMaxIssuesPerRun = int32(5)
	// defaultMaxOpenPRs is the seeded limit of open pull requests.
	defaultMaxOpenPRs = int32(3)
)

// RemediationAgentSchedulesClient handles the SonarQube scheduled remediation
// agent configuration API.
type RemediationAgentSchedulesClient interface {
	GetScheduledAgentConfig(ctx context.Context, scheduledAgentConfigID string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	CreateScheduledAgentConfig(ctx context.Context, opt *sonar.RemediationAgentCreateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	UpdateScheduledAgentConfig(ctx context.Context, scheduledAgentConfigID string, opt *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	DeleteScheduledAgentConfig(ctx context.Context, scheduledAgentConfigID string) (*http.Response, error)
}

// NewRemediationAgentSchedulesClient creates a new
// RemediationAgentSchedulesClient from the given config.
func NewRemediationAgentSchedulesClient(clientConfig common.Config) RemediationAgentSchedulesClient {
	return common.NewClient(clientConfig).V2.RemediationAgent
}

// RemediationAgentScheduleID returns the identifier of the schedule described
// by spec: the project key, or the instance-wide identifier. It is empty while
// a project reference or selector has not been resolved yet.
func RemediationAgentScheduleID(spec *v1alpha1.RemediationAgentScheduleParameters) string {
	if spec.ProjectKey != nil {
		return *spec.ProjectKey
	}

	if spec.ProjectKeyRef != nil || spec.ProjectKeySelector != nil {
		return ""
	}

	return GlobalRemediationAgentScheduleID
}

// GenerateRemediationAgentScheduleObservation builds a
// RemediationAgentScheduleObservation from the SonarQube API response.
func GenerateRemediationAgentScheduleObservation(config *sonar.RemediationAgentScheduledAgentConfigResource) v1alpha1.RemediationAgentScheduleObservation {
	if config == nil {
		return v1alpha1.RemediationAgentScheduleObservation{}
	}

	observation := v1alpha1.RemediationAgentScheduleObservation{
		ID:                   config.Id,
		ProjectKey:           config.ProjectKey,
		Enabled:              config.Enabled,
		Frequency:            config.Frequency,
		DayOfWeek:            config.DayOfWeek,
		Hour:                 config.Hour,
		Timezone:             config.Timezone,
		MaxIssuesPerRun:      config.MaxIssuesPerRun,
		MaxOpenPRs:           config.MaxOpenPRs,
		ProjectSelectionMode: config.ProjectSelectionMode,
		SelectedProjectKeys:  slices.Sorted(slices.Values(config.SelectedProjectKeys)),
	}

	if config.NextRunUtc != 0 {
		observation.NextRunTime = helpers.IntTimestampToMetaTime(&config.NextRunUtc)
	}

	return observation
}

// LateInitializeRemediationAgentSchedule fills empty spec fields from the
// observation.
func LateInitializeRemediationAgentSchedule(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) {
	if spec == nil || observation == nil {
		return
	}

	if observation.Timezone != "" {
		helpers.AssignIfNil(&spec.Timezone, observation.Timezone)
	}

	if observation.MaxIssuesPerRun != 0 {
		helpers.AssignIfNil(&spec.MaxIssuesPerRun, observation.MaxIssuesPerRun)
	}

	if observation.MaxOpenPRs != 0 {
		helpers.AssignIfNil(&spec.MaxOpenPRs, observation.MaxOpenPRs)
	}

	lateInitializeProjectSelection(spec, observation)
}

// lateInitializeProjectSelection fills the project selection of the
// instance-wide schedule. The mode and the selected projects are only filled
// together, as the CRD rejects one without the other.
func lateInitializeProjectSelection(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) {
	if RemediationAgentScheduleID(spec) != GlobalRemediationAgentScheduleID || spec.ProjectSelectionMode != nil || observation.ProjectSelectionMode == "" {
		return
	}

	if observation.ProjectSelectionMode == selectionModeSelected {
		// SonarQube prunes deleted projects, which can leave the selection empty.
		if len(observation.SelectedProjectKeys) == 0 {
			return
		}

		spec.SelectedProjectKeys = slices.Clone(observation.SelectedProjectKeys)
	}

	spec.ProjectSelectionMode = &observation.ProjectSelectionMode
}

// IsRemediationAgentScheduleLateInitialized returns true if the spec changed
// during late initialization.
func IsRemediationAgentScheduleLateInitialized(former, current *v1alpha1.RemediationAgentScheduleParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !gocmp.Equal(former, current)
}

// IsRemediationAgentScheduleUpToDate returns true if the observed schedule
// matches the desired one.
func IsRemediationAgentScheduleUpToDate(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return isRunTimeUpToDate(spec, observation) &&
		helpers.IsComparablePtrEqualComparable(spec.Enabled, observation.Enabled) &&
		helpers.IsComparablePtrEqualComparable(spec.MaxIssuesPerRun, observation.MaxIssuesPerRun) &&
		helpers.IsComparablePtrEqualComparable(spec.MaxOpenPRs, observation.MaxOpenPRs) &&
		isProjectSelectionUpToDate(spec, observation)
}

// isRunTimeUpToDate returns true if the observed schedule runs when desired.
func isRunTimeUpToDate(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) bool {
	return spec.Frequency == observation.Frequency &&
		ptr.Deref(spec.DayOfWeek, "") == observation.DayOfWeek &&
		helpers.IsComparablePtrEqualComparable(spec.Hour, observation.Hour) &&
		helpers.IsComparablePtrEqualComparable(spec.Timezone, observation.Timezone)
}

// isProjectSelectionUpToDate returns true if the observed schedule applies to
// the desired projects.
func isProjectSelectionUpToDate(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) bool {
	if !helpers.IsComparablePtrEqualComparable(spec.ProjectSelectionMode, observation.ProjectSelectionMode) {
		return false
	}

	return ptr.Deref(spec.ProjectSelectionMode, "") != selectionModeSelected ||
		helpers.AreStringSlicesEqualDeDuped(spec.SelectedProjectKeys, observation.SelectedProjectKeys)
}

// GenerateRemediationAgentScheduleCreateOptions builds the options creating the
// schedule described by spec.
func GenerateRemediationAgentScheduleCreateOptions(spec *v1alpha1.RemediationAgentScheduleParameters) *sonar.RemediationAgentCreateScheduledAgentConfigOptions {
	options := GenerateRemediationAgentScheduleUpdateOptions(spec, nil)

	return &sonar.RemediationAgentCreateScheduledAgentConfigOptions{
		ProjectKey:           ptr.Deref(spec.ProjectKey, ""),
		Enabled:              options.Enabled,
		Frequency:            options.Frequency,
		Hour:                 options.Hour,
		DayOfWeek:            options.DayOfWeek,
		Timezone:             options.Timezone,
		MaxIssuesPerRun:      options.MaxIssuesPerRun,
		MaxOpenPRs:           options.MaxOpenPRs,
		ProjectSelectionMode: options.ProjectSelectionMode,
		SelectedProjectKeys:  options.SelectedProjectKeys,
	}
}

// GenerateRemediationAgentScheduleUpdateOptions builds the options replacing
// the schedule with the one described by spec. SonarQube replaces the whole
// configuration, so fields unset in spec keep their observed value, or take the
// SonarQube default when there is no observation.
func GenerateRemediationAgentScheduleUpdateOptions(spec *v1alpha1.RemediationAgentScheduleParameters, observation *v1alpha1.RemediationAgentScheduleObservation) *sonar.RemediationAgentUpdateScheduledAgentConfigOptions {
	observed := ptr.Deref(observation, v1alpha1.RemediationAgentScheduleObservation{})

	options := &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
		Enabled:         spec.Enabled,
		Frequency:       spec.Frequency,
		Hour:            spec.Hour,
		DayOfWeek:       ptr.Deref(spec.DayOfWeek, ""),
		Timezone:        ptr.Deref(spec.Timezone, cmp.Or(observed.Timezone, defaultScheduleTimezone)),
		MaxIssuesPerRun: new(ptr.Deref(spec.MaxIssuesPerRun, cmp.Or(observed.MaxIssuesPerRun, defaultMaxIssuesPerRun))),
		MaxOpenPRs:      new(ptr.Deref(spec.MaxOpenPRs, cmp.Or(observed.MaxOpenPRs, defaultMaxOpenPRs))),
	}

	// The project selection only exists on the instance-wide schedule.
	if RemediationAgentScheduleID(spec) != GlobalRemediationAgentScheduleID {
		return options
	}

	options.ProjectSelectionMode = ptr.Deref(spec.ProjectSelectionMode, cmp.Or(observed.ProjectSelectionMode, "ALL"))
	if options.ProjectSelectionMode == selectionModeSelected {
		options.SelectedProjectKeys = spec.SelectedProjectKeys
		if spec.ProjectSelectionMode == nil {
			options.SelectedProjectKeys = observed.SelectedProjectKeys
		}
	}

	return options
}
