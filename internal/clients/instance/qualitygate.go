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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// QualityGatesClient is the interface for interacting with SonarQube Quality Gates API
// It handles all the operations related to Quality Gates in SonarQube, such as creating, updating, deleting, and retrieving Quality Gates and their conditions.
// It also handles users / groups / projects association with Quality Gates.
// It also interacts with Quality Gate Conditions.
type QualityGatesClient interface {
	AddGroup(opt *sonar.QualitygatesAddGroupOption) (resp *http.Response, err error)
	AddUser(opt *sonar.QualitygatesAddUserOption) (resp *http.Response, err error)
	Copy(opt *sonar.QualitygatesCopyOption) (resp *http.Response, err error)
	Create(opt *sonar.QualitygatesCreateOption) (v *sonar.QualitygatesCreate, resp *http.Response, err error)
	CreateCondition(opt *sonar.QualitygatesCreateConditionOption) (v *sonar.QualitygatesCreateCondition, resp *http.Response, err error)
	DeleteCondition(opt *sonar.QualitygatesDeleteConditionOption) (resp *http.Response, err error)
	Deselect(opt *sonar.QualitygatesDeselectOption) (resp *http.Response, err error)
	Destroy(opt *sonar.QualitygatesDestroyOption) (resp *http.Response, err error)
	GetByProject(opt *sonar.QualitygatesGetByProjectOption) (v *sonar.QualitygatesGetByProject, resp *http.Response, err error)
	List() (v *sonar.QualitygatesList, resp *http.Response, err error)
	ProjectStatus(opt *sonar.QualitygatesProjectStatusOption) (v *sonar.QualitygatesProjectStatus, resp *http.Response, err error)
	RemoveGroup(opt *sonar.QualitygatesRemoveGroupOption) (resp *http.Response, err error)
	RemoveUser(opt *sonar.QualitygatesRemoveUserOption) (resp *http.Response, err error)
	Rename(opt *sonar.QualitygatesRenameOption) (resp *http.Response, err error)
	Search(opt *sonar.QualitygatesSearchOption) (v *sonar.QualitygatesSearch, resp *http.Response, err error)
	SearchGroups(opt *sonar.QualitygatesSearchGroupsOption) (v *sonar.QualitygatesSearchGroups, resp *http.Response, err error)
	SearchUsers(opt *sonar.QualitygatesSearchUsersOption) (v *sonar.QualitygatesSearchUsers, resp *http.Response, err error)
	Select(opt *sonar.QualitygatesSelectOption) (resp *http.Response, err error)
	SetAsDefault(opt *sonar.QualitygatesSetAsDefaultOption) (resp *http.Response, err error)
	Show(opt *sonar.QualitygatesShowOption) (v *sonar.QualitygatesShow, resp *http.Response, err error)
	UpdateCondition(opt *sonar.QualitygatesUpdateConditionOption) (resp *http.Response, err error)
}

// NewQualityGatesClient creates a new QualityGatesClient with the provided SonarQube client configuration.
func NewQualityGatesClient(clientConfig common.Config) QualityGatesClient {
	newClient := common.NewClient(clientConfig)
	return newClient.Qualitygates
}

// GenerateQualityGateCreateOptions generates SonarQube QualitygatesCreateOption from QualityGateParameters
func GenerateQualityGateCreateOptions(spec v1alpha1.QualityGateParameters) *sonar.QualitygatesCreateOption {
	return &sonar.QualitygatesCreateOption{
		Name: spec.Name,
	}
}

// GenerateQualityGateObservation generates QualityGateObservation from SonarQube QualityGate
// observation should not be nil, else it will panic
func GenerateQualityGateObservation(observation *sonar.QualitygatesShow) v1alpha1.QualityGateObservation {
	return v1alpha1.QualityGateObservation{
		Actions:           GenerateQualityGateActionsObservation(&observation.Actions),
		CaycStatus:        observation.CaycStatus,
		Conditions:        GenerateQualityGateConditionsObservation(observation.Conditions),
		IsAiCodeSupported: observation.IsAiCodeSupported,
		IsBuiltIn:         observation.IsBuiltIn,
		IsDefault:         observation.IsDefault,
		Name:              observation.Name,
	}
}

// GenerateQualityGateActionsObservation generates QualityGatesActions from SonarQube QualityGateActions
// actions should not be nil, else it will panic
func GenerateQualityGateActionsObservation(actions *sonar.QualityGateActions) v1alpha1.QualityGatesActions {
	return v1alpha1.QualityGatesActions{
		AssociateProjects:     actions.AssociateProjects,
		Copy:                  actions.Copy,
		Delegate:              actions.Delegate,
		Delete:                actions.Delete,
		ManageAiCodeAssurance: actions.ManageAiCodeAssurance,
		ManageConditions:      actions.ManageConditions,
		Rename:                actions.Rename,
		SetAsDefault:          actions.SetAsDefault,
	}
}

// IsQualityGateUpToDate checks if the Quality Gate spec is up to date with the observed state
func IsQualityGateUpToDate(spec *v1alpha1.QualityGateParameters, observation *v1alpha1.QualityGateObservation, associations map[string]QualityGateConditionAssociation) bool {
	if spec == nil {
		return true
	}
	if observation == nil {
		return false
	}

	if spec.Name != observation.Name {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(spec.Default, observation.IsDefault) {
		return false
	}

	if !AreQualityGateConditionsUpToDate(associations) {
		return false
	}

	return true
}

// buildObservationIdSet creates a map of all observation condition IDs for quick lookup
func buildObservationIdSet(conditions []v1alpha1.QualityGateConditionObservation) map[string]bool {
	idSet := make(map[string]bool)
	for i := range conditions {
		idSet[conditions[i].ID] = true
	}
	return idSet
}

// findMatchingObservationId searches for an observation condition that matches the spec condition
// by metric, error, and op, and returns its ID
func findMatchingObservationId(specCondition v1alpha1.QualityGateConditionParameters, observations []v1alpha1.QualityGateConditionObservation) *string {
	for i := range observations {
		if specCondition.Metric == observations[i].Metric &&
			specCondition.Error == observations[i].Error &&
			helpers.IsComparablePtrEqualComparable(specCondition.Op, observations[i].Op) {
			return &observations[i].ID
		}
	}
	return nil
}

// LateInitializeQualityGate fills the spec with the observed state if the spec fields are nil
// It also late-initializes condition IDs by matching conditions by their metric, error, and op fields
// If a condition has a stale ID (doesn't exist in observations), it will be updated to the correct ID
func LateInitializeQualityGate(spec *v1alpha1.QualityGateParameters, observation *v1alpha1.QualityGateObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Default, observation.IsDefault)

	// Build a map of observation IDs for quick lookup
	observationIdSet := buildObservationIdSet(observation.Conditions)

	// Late-initialize or update condition IDs by matching on metric, error, and op
	for i := range spec.Conditions {
		// Check if the spec has an ID that still exists in observations
		hasValidId := spec.Conditions[i].Id != nil && observationIdSet[*spec.Conditions[i].Id]

		if hasValidId {
			// Already has a valid ID that exists in observations, skip
			continue
		}

		// Either no ID or stale ID - find matching observation by metric, error, and op
		if matchingId := findMatchingObservationId(spec.Conditions[i], observation.Conditions); matchingId != nil {
			spec.Conditions[i].Id = matchingId
		}
	}
}

// WereQualityGateConditionsLateInitialized checks if any conditions had their IDs late-initialized
// by comparing the before and after states
func WereQualityGateConditionsLateInitialized(before, after []v1alpha1.QualityGateConditionParameters) bool {
	if len(before) != len(after) {
		return true
	}

	// Check if any condition that didn't have an ID now has one
	for i := range before {
		if !helpers.IsComparablePtrEqualComparablePtr(before[i].Id, after[i].Id) || !helpers.IsComparablePtrEqualComparablePtr(before[i].Op, after[i].Op) {
			return true
		}
	}

	return false
}
