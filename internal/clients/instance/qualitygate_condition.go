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
	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// GenerateQualityGateConditionObservation generates QualityGateConditionObservation from SonarQube QualitygatesShowObject_sub2
func GenerateQualityGateConditionObservation(condition *sonargo.QualitygatesShowObject_sub2) v1alpha1.QualityGateConditionObservation {
	return v1alpha1.QualityGateConditionObservation{
		Error:  condition.Error,
		ID:     condition.ID,
		Metric: condition.Metric,
		Op:     condition.Op,
	}
}

// GenerateQualityGateConditionObservationFromCreate generates QualityGateConditionObservation from SonarQube QualitygatesShowObject_sub2
func GenerateQualityGateConditionObservationFromCreate(condition *sonargo.QualitygatesCreateConditionObject) *v1alpha1.QualityGateConditionObservation {
	return &v1alpha1.QualityGateConditionObservation{
		Error:  condition.Error,
		ID:     condition.ID,
		Metric: condition.Metric,
		Op:     condition.Op,
	}
}

// GenerateQualityGateConditionsObservation generates a slice of QualityGateConditionObservation from a slice of SonarQube QualitygatesShowObject_sub2
func GenerateQualityGateConditionsObservation(conditions []sonargo.QualitygatesShowObject_sub2) []v1alpha1.QualityGateConditionObservation {
	conditionObservations := make([]v1alpha1.QualityGateConditionObservation, len(conditions))
	for i, condition := range conditions {
		conditionObservations[i] = GenerateQualityGateConditionObservation(&condition)
	}
	return conditionObservations
}

// GenerateCreateQualityGateConditionOption generates SonarQube QualitygatesCreateConditionOption from QualityGateConditionParameters
func GenerateCreateQualityGateConditionOption(gateName string, params v1alpha1.QualityGateConditionParameters) *sonargo.QualitygatesCreateConditionOption {
	option := sonargo.QualitygatesCreateConditionOption{
		GateName: gateName,
		Error:    params.Error,
		Metric:   params.Metric,
	}
	if params.Op != nil {
		option.Op = *params.Op
	}
	return &option
}

// GenerateUpdateQualityGateConditionOption generates SonarQube QualitygatesUpdateConditionOption from QualityGateConditionParameters
func GenerateUpdateQualityGateConditionOption(id string, params v1alpha1.QualityGateConditionParameters) *sonargo.QualitygatesUpdateConditionOption {
	option := sonargo.QualitygatesUpdateConditionOption{
		Id:     id,
		Error:  params.Error,
		Metric: params.Metric,
	}
	if params.Op != nil {
		option.Op = *params.Op
	}
	return &option
}

// GenerateDeleteQualityGateConditionOption generates SonarQube QualitygatesDeleteConditionOption from QualityGateConditionObservation
func GenerateDeleteQualityGateConditionOption(id string) *sonargo.QualitygatesDeleteConditionOption {
	return &sonargo.QualitygatesDeleteConditionOption{
		Id: id,
	}
}

// LateInitializeQualityGateCondition fills the empty fields in *QualityGateConditionParameters with
// the values seen in QualityGateConditionObservation.
func LateInitializeQualityGateCondition(params *v1alpha1.QualityGateConditionParameters, observation *v1alpha1.QualityGateConditionObservation) {
	if params == nil || observation == nil {
		return
	}

	// ID is always set from observation to params
	params.Id = &observation.ID

	helpers.AssignIfNil(&params.Op, observation.Op)
}

// IsQualityGateConditionUpToDate checks whether the observed QualityGateCondition is up to date with the desired QualityGateConditionParameters
func IsQualityGateConditionUpToDate(params *v1alpha1.QualityGateConditionParameters, observation *v1alpha1.QualityGateConditionObservation) bool {
	if params == nil {
		return true
	}
	if observation == nil {
		return false
	}

	if params.Error != observation.Error {
		return false
	}
	if params.Metric != observation.Metric {
		return false
	}
	if !helpers.IsComparablePtrEqualComparable(params.Op, observation.Op) {
		return false
	}

	return true
}

// QualityGateConditionAssociation associates a QualityGateConditionObservation with its corresponding QualityGateConditionParameters
type QualityGateConditionAssociation struct {
	Observation *v1alpha1.QualityGateConditionObservation
	Spec        *v1alpha1.QualityGateConditionParameters
	UpToDate    bool
}

// GenerateQualityGateConditionsAssociation generates associations between QualityGateConditionParameters and QualityGateConditionObservation
func GenerateQualityGateConditionsAssociation(specs []v1alpha1.QualityGateConditionParameters, observations []v1alpha1.QualityGateConditionObservation) map[string]QualityGateConditionAssociation {
	associations := make(map[string]QualityGateConditionAssociation)

	for i := range observations {
		associations[observations[i].ID] = QualityGateConditionAssociation{
			Observation: &observations[i],
			Spec:        nil,
			UpToDate:    false,
		}
	}

	// Create associations for specs with IDs
	for i := range specs {
		if specs[i].Id != nil {
			if assoc, exists := associations[*specs[i].Id]; exists {
				assoc.Spec = &specs[i]
				assoc.UpToDate = IsQualityGateConditionUpToDate(&specs[i], assoc.Observation)
				associations[*specs[i].Id] = assoc
			} else {
				// Spec has an ID but no matching observation (stale ID reference)
				associations[*specs[i].Id] = QualityGateConditionAssociation{
					Observation: nil,
					Spec:        &specs[i],
					UpToDate:    false,
				}
			}
		} else {
			// Spec without ID is a new condition that needs to be created
			// Use a unique placeholder key based on metric to track it
			key := "new:" + specs[i].Metric
			associations[key] = QualityGateConditionAssociation{
				Observation: nil,
				Spec:        &specs[i],
				UpToDate:    false,
			}
		}
	}

	return associations
}

// AreQualityGateConditionsUpToDate checks whether the observed QualityGateConditions are up to date with the desired QualityGateConditionParameters
func AreQualityGateConditionsUpToDate(associations map[string]QualityGateConditionAssociation) bool {
	for _, assoc := range associations {
		if !assoc.UpToDate {
			return false
		}
	}
	return true
}

// FindNonExistingQualityGateConditions finds QualityGateConditionParameters that do not have a corresponding QualityGateConditionObservation
func FindNonExistingQualityGateConditions(associations map[string]QualityGateConditionAssociation) []*v1alpha1.QualityGateConditionParameters {
	var nonExisting []*v1alpha1.QualityGateConditionParameters
	for _, assoc := range associations {
		if assoc.Observation == nil && assoc.Spec != nil {
			nonExisting = append(nonExisting, assoc.Spec)
		}
	}

	return nonExisting
}

// FindMissingQualityGateConditions finds QualityGateConditionObservations that do not have a corresponding QualityGateConditionParameters
func FindMissingQualityGateConditions(associations map[string]QualityGateConditionAssociation) []*v1alpha1.QualityGateConditionObservation {
	var missing []*v1alpha1.QualityGateConditionObservation
	for _, assoc := range associations {
		if assoc.Spec == nil && assoc.Observation != nil {
			missing = append(missing, assoc.Observation)
		}
	}
	return missing
}

// FindNotUpToDateQualityGateConditions finds QualityGateConditionParameters that are not up to date with their corresponding QualityGateConditionObservation
// This ignores associations where either Spec or Observation is nil
func FindNotUpToDateQualityGateConditions(associations map[string]QualityGateConditionAssociation) []QualityGateConditionAssociation {
	var notUpToDate []QualityGateConditionAssociation
	for _, assoc := range associations {
		if !assoc.UpToDate && assoc.Spec != nil && assoc.Observation != nil {
			notUpToDate = append(notUpToDate, assoc)
		}
	}
	return notUpToDate
}
