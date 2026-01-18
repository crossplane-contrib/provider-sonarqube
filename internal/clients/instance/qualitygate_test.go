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
	"testing"

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

func TestGenerateQualityGateCreateOptions(t *testing.T) {
	tests := map[string]struct {
		spec v1alpha1.QualityGateParameters
		want *sonargo.QualitygatesCreateOption
	}{
		"BasicCreateOption": {
			spec: v1alpha1.QualityGateParameters{
				Name: "my-quality-gate",
			},
			want: &sonargo.QualitygatesCreateOption{
				Name: "my-quality-gate",
			},
		},
		"CreateOptionWithDefault": {
			spec: v1alpha1.QualityGateParameters{
				Name:    "default-gate",
				Default: ptr.To(true),
			},
			want: &sonargo.QualitygatesCreateOption{
				Name: "default-gate",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateCreateOptions(tc.spec)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateCreateOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateQualityGateObservation(t *testing.T) {
	tests := map[string]struct {
		observation *sonargo.QualitygatesShowObject
		want        v1alpha1.QualityGateObservation
	}{
		"BasicObservation": {
			observation: &sonargo.QualitygatesShowObject{
				Name:              "test-gate",
				CaycStatus:        "compliant",
				IsBuiltIn:         false,
				IsDefault:         true,
				IsAiCodeSupported: false,
				Conditions:        []sonargo.QualitygatesShowObject_sub2{},
				Actions: sonargo.QualitygatesShowObject_sub1{
					AssociateProjects:     true,
					Copy:                  true,
					Delete:                true,
					ManageConditions:      true,
					Rename:                true,
					SetAsDefault:          true,
					Delegate:              false,
					ManageAiCodeAssurance: false,
				},
			},
			want: v1alpha1.QualityGateObservation{
				Name:              "test-gate",
				CaycStatus:        "compliant",
				IsBuiltIn:         false,
				IsDefault:         true,
				IsAiCodeSupported: false,
				Conditions:        []v1alpha1.QualityGateConditionObservation{},
				Actions: v1alpha1.QualityGatesActions{
					AssociateProjects:     true,
					Copy:                  true,
					Delete:                true,
					ManageConditions:      true,
					Rename:                true,
					SetAsDefault:          true,
					Delegate:              false,
					ManageAiCodeAssurance: false,
				},
			},
		},
		"ObservationWithConditions": {
			observation: &sonargo.QualitygatesShowObject{
				Name:       "gate-with-conditions",
				CaycStatus: "non_compliant",
				IsBuiltIn:  true,
				IsDefault:  false,
				Conditions: []sonargo.QualitygatesShowObject_sub2{
					{
						ID:     "1",
						Metric: "coverage",
						Op:     "LT",
						Error:  "80",
					},
					{
						ID:     "2",
						Metric: "duplicated_lines_density",
						Op:     "GT",
						Error:  "3",
					},
				},
				Actions: sonargo.QualitygatesShowObject_sub1{},
			},
			want: v1alpha1.QualityGateObservation{
				Name:       "gate-with-conditions",
				CaycStatus: "non_compliant",
				IsBuiltIn:  true,
				IsDefault:  false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{
						ID:     "1",
						Metric: "coverage",
						Op:     "LT",
						Error:  "80",
					},
					{
						ID:     "2",
						Metric: "duplicated_lines_density",
						Op:     "GT",
						Error:  "3",
					},
				},
				Actions: v1alpha1.QualityGatesActions{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateObservation(tc.observation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateQualityGateActionsObservation(t *testing.T) {
	tests := map[string]struct {
		actions *sonargo.QualitygatesShowObject_sub1
		want    v1alpha1.QualityGatesActions
	}{
		"AllActionsEnabled": {
			actions: &sonargo.QualitygatesShowObject_sub1{
				AssociateProjects:     true,
				Copy:                  true,
				Delegate:              true,
				Delete:                true,
				ManageAiCodeAssurance: true,
				ManageConditions:      true,
				Rename:                true,
				SetAsDefault:          true,
			},
			want: v1alpha1.QualityGatesActions{
				AssociateProjects:     true,
				Copy:                  true,
				Delegate:              true,
				Delete:                true,
				ManageAiCodeAssurance: true,
				ManageConditions:      true,
				Rename:                true,
				SetAsDefault:          true,
			},
		},
		"NoActionsEnabled": {
			actions: &sonargo.QualitygatesShowObject_sub1{},
			want:    v1alpha1.QualityGatesActions{},
		},
		"PartialActionsEnabled": {
			actions: &sonargo.QualitygatesShowObject_sub1{
				Copy:   true,
				Rename: true,
			},
			want: v1alpha1.QualityGatesActions{
				Copy:   true,
				Rename: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateActionsObservation(tc.actions)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateActionsObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsQualityGateUpToDate(t *testing.T) {
	tests := map[string]struct {
		spec         *v1alpha1.QualityGateParameters
		observation  *v1alpha1.QualityGateObservation
		associations map[string]QualityGateConditionAssociation
		want         bool
	}{
		"NilSpecReturnsTrue": {
			spec:         nil,
			observation:  &v1alpha1.QualityGateObservation{Name: "test"},
			associations: nil,
			want:         true,
		},
		"NilObservationReturnsFalse": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test"},
			observation:  nil,
			associations: nil,
			want:         false,
		},
		"MatchingNameReturnsTrue": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test"},
			observation:  &v1alpha1.QualityGateObservation{Name: "test"},
			associations: nil,
			want:         true,
		},
		"DifferentNameReturnsFalse": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test"},
			observation:  &v1alpha1.QualityGateObservation{Name: "different"},
			associations: nil,
			want:         false,
		},
		"MatchingDefaultReturnsTrue": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test", Default: ptr.To(true)},
			observation:  &v1alpha1.QualityGateObservation{Name: "test", IsDefault: true},
			associations: nil,
			want:         true,
		},
		"DifferentDefaultReturnsFalse": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test", Default: ptr.To(true)},
			observation:  &v1alpha1.QualityGateObservation{Name: "test", IsDefault: false},
			associations: nil,
			want:         false,
		},
		"NilDefaultWithObservedFalseReturnsTrue": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test", Default: nil},
			observation:  &v1alpha1.QualityGateObservation{Name: "test", IsDefault: false},
			associations: nil,
			want:         true,
		},
		"NilDefaultWithObservedTrueReturnsTrue": {
			spec:         &v1alpha1.QualityGateParameters{Name: "test", Default: nil},
			observation:  &v1alpha1.QualityGateObservation{Name: "test", IsDefault: true},
			associations: nil,
			want:         true,
		},
		"ConditionsNotUpToDateReturnsFalse": {
			spec:        &v1alpha1.QualityGateParameters{Name: "test"},
			observation: &v1alpha1.QualityGateObservation{Name: "test"},
			associations: map[string]QualityGateConditionAssociation{
				"1": {UpToDate: false},
			},
			want: false,
		},
		"ConditionsUpToDateReturnsTrue": {
			spec:        &v1alpha1.QualityGateParameters{Name: "test"},
			observation: &v1alpha1.QualityGateObservation{Name: "test"},
			associations: map[string]QualityGateConditionAssociation{
				"1": {UpToDate: true},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsQualityGateUpToDate(tc.spec, tc.observation, tc.associations)
			if got != tc.want {
				t.Errorf("IsQualityGateUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLateInitializeQualityGate(t *testing.T) {
	tests := map[string]struct {
		spec           *v1alpha1.QualityGateParameters
		observation    *v1alpha1.QualityGateObservation
		wantDefault    *bool
		wantConditions []v1alpha1.QualityGateConditionParameters
	}{
		"NilSpecDoesNothing": {
			spec:        nil,
			observation: &v1alpha1.QualityGateObservation{IsDefault: true},
			wantDefault: nil,
		},
		"NilObservationDoesNothing": {
			spec:        &v1alpha1.QualityGateParameters{Name: "test"},
			observation: nil,
			wantDefault: nil,
		},
		"NilDefaultGetsInitialized": {
			spec:        &v1alpha1.QualityGateParameters{Name: "test", Default: nil},
			observation: &v1alpha1.QualityGateObservation{IsDefault: true},
			wantDefault: ptr.To(true),
		},
		"ExistingDefaultNotOverwritten": {
			spec:        &v1alpha1.QualityGateParameters{Name: "test", Default: ptr.To(false)},
			observation: &v1alpha1.QualityGateObservation{IsDefault: true},
			wantDefault: ptr.To(false),
		},
		"ConditionWithoutIdGetsInitialized": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "condition-123", Metric: "coverage", Error: "80", Op: "LT"},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("condition-123"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
		"ConditionWithValidIdNotOverwritten": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Id: ptr.To("existing-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "existing-id", Metric: "coverage", Error: "80", Op: "LT"}, // same ID in observation
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("existing-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
		"ConditionWithStaleIdGetsUpdated": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Id: ptr.To("stale-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "new-id", Metric: "coverage", Error: "80", Op: "LT"}, // different ID in observation
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("new-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
		"MultipleConditionsMatchedCorrectly": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Metric: "coverage", Error: "80", Op: ptr.To("LT")},
					{Metric: "bugs", Error: "0", Op: ptr.To("GT")},
					{Id: ptr.To("dup-id"), Metric: "duplicated_lines", Error: "3"}, // valid ID
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "cov-id", Metric: "coverage", Error: "80", Op: "LT"},
					{ID: "bugs-id", Metric: "bugs", Error: "0", Op: "GT"},
					{ID: "dup-id", Metric: "duplicated_lines", Error: "3", Op: ""},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("cov-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				{Id: ptr.To("bugs-id"), Metric: "bugs", Error: "0", Op: ptr.To("GT")},
				{Id: ptr.To("dup-id"), Metric: "duplicated_lines", Error: "3"}, // kept because valid
			},
		},
		"NoMatchingConditionLeavesIdNil": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "other-id", Metric: "bugs", Error: "0", Op: "GT"},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: nil, Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
		"StaleIdGetsUpdated": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Id: ptr.To("old-stale-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "new-id-456", Metric: "coverage", Error: "80", Op: "LT"},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("new-id-456"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
		"MultipleConditionsOneWithStaleId": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Id: ptr.To("valid-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
					{Id: ptr.To("stale-id"), Metric: "bugs", Error: "0", Op: ptr.To("GT")},
					{Metric: "duplicated_lines", Error: "3"},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "valid-id", Metric: "coverage", Error: "80", Op: "LT"},
					{ID: "new-bugs-id", Metric: "bugs", Error: "0", Op: "GT"},
					{ID: "dup-id", Metric: "duplicated_lines", Error: "3", Op: ""},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("valid-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				{Id: ptr.To("new-bugs-id"), Metric: "bugs", Error: "0", Op: ptr.To("GT")},
				{Id: ptr.To("dup-id"), Metric: "duplicated_lines", Error: "3"},
			},
		},
		"StaleIdWithNoMatchingConditionBecomesNil": {
			spec: &v1alpha1.QualityGateParameters{
				Name: "test",
				Conditions: []v1alpha1.QualityGateConditionParameters{
					{Id: ptr.To("stale-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
				},
			},
			observation: &v1alpha1.QualityGateObservation{
				IsDefault: false,
				Conditions: []v1alpha1.QualityGateConditionObservation{
					{ID: "other-id", Metric: "bugs", Error: "0", Op: "GT"},
				},
			},
			wantDefault: ptr.To(false),
			wantConditions: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("stale-id"), Metric: "coverage", Error: "80", Op: ptr.To("LT")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			LateInitializeQualityGate(tc.spec, tc.observation)
			if tc.spec == nil {
				return
			}
			if tc.wantDefault == nil && tc.spec.Default != nil {
				t.Errorf("LateInitializeQualityGate() Default = %v, want nil", *tc.spec.Default)
				return
			}
			if tc.wantDefault != nil && tc.spec.Default == nil {
				t.Errorf("LateInitializeQualityGate() Default = nil, want %v", *tc.wantDefault)
				return
			}
			if tc.wantDefault != nil && tc.spec.Default != nil && *tc.spec.Default != *tc.wantDefault {
				t.Errorf("LateInitializeQualityGate() Default = %v, want %v", *tc.spec.Default, *tc.wantDefault)
			}

			// Check conditions if expected
			if tc.wantConditions != nil {
				if diff := cmp.Diff(tc.wantConditions, tc.spec.Conditions); diff != "" {
					t.Errorf("LateInitializeQualityGate() Conditions mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestWereQualityGateConditionsLateInitialized(t *testing.T) {
	tests := map[string]struct {
		before []v1alpha1.QualityGateConditionParameters
		after  []v1alpha1.QualityGateConditionParameters
		want   bool
	}{
		"NoConditionsReturnsFalse": {
			before: []v1alpha1.QualityGateConditionParameters{},
			after:  []v1alpha1.QualityGateConditionParameters{},
			want:   false,
		},
		"DifferentLengthReturnsTrue": {
			before: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80"},
			},
			after: []v1alpha1.QualityGateConditionParameters{},
			want:  true,
		},
		"IdAddedReturnsTrue": {
			before: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: nil},
			},
			after: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("new-id")},
			},
			want: true,
		},
		"IdAlreadyPresentReturnsFalse": {
			before: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("existing-id")},
			},
			after: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("existing-id")},
			},
			want: false,
		},
		"MultipleConditionsOneInitializedReturnsTrue": {
			before: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("id-1")},
				{Metric: "bugs", Error: "0", Id: nil},
			},
			after: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("id-1")},
				{Metric: "bugs", Error: "0", Id: ptr.To("id-2")},
			},
			want: true,
		},
		"NoChangesReturnsFalse": {
			before: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("id-1")},
				{Metric: "bugs", Error: "0", Id: ptr.To("id-2")},
			},
			after: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80", Id: ptr.To("id-1")},
				{Metric: "bugs", Error: "0", Id: ptr.To("id-2")},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := WereQualityGateConditionsLateInitialized(tc.before, tc.after)
			if got != tc.want {
				t.Errorf("WereQualityGateConditionsLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}
