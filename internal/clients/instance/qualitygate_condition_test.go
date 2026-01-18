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

func TestGenerateQualityGateConditionObservation(t *testing.T) {
	tests := map[string]struct {
		condition *sonargo.QualitygatesShowObject_sub2
		want      v1alpha1.QualityGateConditionObservation
	}{
		"BasicCondition": {
			condition: &sonargo.QualitygatesShowObject_sub2{
				ID:     "123",
				Metric: "coverage",
				Op:     "LT",
				Error:  "80",
			},
			want: v1alpha1.QualityGateConditionObservation{
				ID:     "123",
				Metric: "coverage",
				Op:     "LT",
				Error:  "80",
			},
		},
		"EmptyCondition": {
			condition: &sonargo.QualitygatesShowObject_sub2{},
			want:      v1alpha1.QualityGateConditionObservation{},
		},
		"ConditionWithGTOperator": {
			condition: &sonargo.QualitygatesShowObject_sub2{
				ID:     "456",
				Metric: "duplicated_lines_density",
				Op:     "GT",
				Error:  "3",
			},
			want: v1alpha1.QualityGateConditionObservation{
				ID:     "456",
				Metric: "duplicated_lines_density",
				Op:     "GT",
				Error:  "3",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateConditionObservation(tc.condition)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateConditionObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateQualityGateConditionsObservation(t *testing.T) {
	tests := map[string]struct {
		conditions []sonargo.QualitygatesShowObject_sub2
		want       []v1alpha1.QualityGateConditionObservation
	}{
		"EmptySlice": {
			conditions: []sonargo.QualitygatesShowObject_sub2{},
			want:       []v1alpha1.QualityGateConditionObservation{},
		},
		"SingleCondition": {
			conditions: []sonargo.QualitygatesShowObject_sub2{
				{ID: "1", Metric: "coverage", Op: "LT", Error: "80"},
			},
			want: []v1alpha1.QualityGateConditionObservation{
				{ID: "1", Metric: "coverage", Op: "LT", Error: "80"},
			},
		},
		"MultipleConditions": {
			conditions: []sonargo.QualitygatesShowObject_sub2{
				{ID: "1", Metric: "coverage", Op: "LT", Error: "80"},
				{ID: "2", Metric: "duplicated_lines_density", Op: "GT", Error: "3"},
				{ID: "3", Metric: "new_coverage", Op: "LT", Error: "90"},
			},
			want: []v1alpha1.QualityGateConditionObservation{
				{ID: "1", Metric: "coverage", Op: "LT", Error: "80"},
				{ID: "2", Metric: "duplicated_lines_density", Op: "GT", Error: "3"},
				{ID: "3", Metric: "new_coverage", Op: "LT", Error: "90"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateConditionsObservation(tc.conditions)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateConditionsObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateCreateQualityGateConditionOption(t *testing.T) {
	tests := map[string]struct {
		params v1alpha1.QualityGateConditionParameters
		want   *sonargo.QualitygatesCreateConditionOption
	}{
		"BasicCondition": {
			params: v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
			},
			want: &sonargo.QualitygatesCreateConditionOption{
				GateName: "my-gate",
				Metric:   "coverage",
				Error:    "80",
			},
		},
		"ConditionWithOperator": {
			params: v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
				Op:     ptr.To("LT"),
			},
			want: &sonargo.QualitygatesCreateConditionOption{
				GateName: "my-gate",
				Metric:   "coverage",
				Error:    "80",
				Op:       "LT",
			},
		},
		"ConditionWithGTOperator": {
			params: v1alpha1.QualityGateConditionParameters{
				Metric: "duplicated_lines_density",
				Error:  "3",
				Op:     ptr.To("GT"),
			},
			want: &sonargo.QualitygatesCreateConditionOption{
				GateName: "another-gate",
				Metric:   "duplicated_lines_density",
				Error:    "3",
				Op:       "GT",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateCreateQualityGateConditionOption(tc.want.GateName, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateCreateQualityGateConditionOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateUpdateQualityGateConditionOption(t *testing.T) {
	tests := map[string]struct {
		id     string
		params v1alpha1.QualityGateConditionParameters
		want   *sonargo.QualitygatesUpdateConditionOption
	}{
		"BasicUpdate": {
			id: "123",
			params: v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "85",
			},
			want: &sonargo.QualitygatesUpdateConditionOption{
				Id:     "123",
				Metric: "coverage",
				Error:  "85",
			},
		},
		"UpdateWithOperator": {
			id: "456",
			params: v1alpha1.QualityGateConditionParameters{
				Metric: "duplicated_lines_density",
				Error:  "5",
				Op:     ptr.To("GT"),
			},
			want: &sonargo.QualitygatesUpdateConditionOption{
				Id:     "456",
				Metric: "duplicated_lines_density",
				Error:  "5",
				Op:     "GT",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateUpdateQualityGateConditionOption(tc.id, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateUpdateQualityGateConditionOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateDeleteQualityGateConditionOption(t *testing.T) {
	tests := map[string]struct {
		id   string
		want *sonargo.QualitygatesDeleteConditionOption
	}{
		"BasicDelete": {
			id:   "123",
			want: &sonargo.QualitygatesDeleteConditionOption{Id: "123"},
		},
		"EmptyID": {
			id:   "",
			want: &sonargo.QualitygatesDeleteConditionOption{Id: ""},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateDeleteQualityGateConditionOption(tc.id)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateDeleteQualityGateConditionOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsQualityGateConditionUpToDate(t *testing.T) {
	tests := map[string]struct {
		params      *v1alpha1.QualityGateConditionParameters
		observation *v1alpha1.QualityGateConditionObservation
		want        bool
	}{
		"NilParamsReturnsTrue": {
			params:      nil,
			observation: &v1alpha1.QualityGateConditionObservation{},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			params:      &v1alpha1.QualityGateConditionParameters{},
			observation: nil,
			want:        false,
		},
		"MatchingValuesReturnsTrue": {
			params: &v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
				Op:     ptr.To("LT"),
			},
			observation: &v1alpha1.QualityGateConditionObservation{
				Metric: "coverage",
				Error:  "80",
				Op:     "LT",
			},
			want: true,
		},
		"DifferentErrorReturnsFalse": {
			params: &v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
				Op:     ptr.To("LT"),
			},
			observation: &v1alpha1.QualityGateConditionObservation{
				Metric: "coverage",
				Error:  "85",
				Op:     "LT",
			},
			want: false,
		},
		"DifferentMetricReturnsFalse": {
			params: &v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
			},
			observation: &v1alpha1.QualityGateConditionObservation{
				Metric: "new_coverage",
				Error:  "80",
			},
			want: false,
		},
		"DifferentOpReturnsFalse": {
			params: &v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
				Op:     ptr.To("LT"),
			},
			observation: &v1alpha1.QualityGateConditionObservation{
				Metric: "coverage",
				Error:  "80",
				Op:     "GT",
			},
			want: false,
		},
		"NilOpMatchesAnyObservedOp": {
			params: &v1alpha1.QualityGateConditionParameters{
				Metric: "coverage",
				Error:  "80",
				Op:     nil,
			},
			observation: &v1alpha1.QualityGateConditionObservation{
				Metric: "coverage",
				Error:  "80",
				Op:     "LT",
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsQualityGateConditionUpToDate(tc.params, tc.observation)
			if got != tc.want {
				t.Errorf("IsQualityGateConditionUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLateInitializeQualityGateCondition(t *testing.T) {
	tests := map[string]struct {
		params      *v1alpha1.QualityGateConditionParameters
		observation *v1alpha1.QualityGateConditionObservation
		wantOp      *string
	}{
		"NilParamsDoesNothing": {
			params:      nil,
			observation: &v1alpha1.QualityGateConditionObservation{Op: "LT"},
			wantOp:      nil,
		},
		"NilObservationDoesNothing": {
			params:      &v1alpha1.QualityGateConditionParameters{Metric: "coverage"},
			observation: nil,
			wantOp:      nil,
		},
		"NilOpGetsInitialized": {
			params:      &v1alpha1.QualityGateConditionParameters{Metric: "coverage", Op: nil},
			observation: &v1alpha1.QualityGateConditionObservation{Op: "LT"},
			wantOp:      ptr.To("LT"),
		},
		"ExistingOpNotOverwritten": {
			params:      &v1alpha1.QualityGateConditionParameters{Metric: "coverage", Op: ptr.To("GT")},
			observation: &v1alpha1.QualityGateConditionObservation{Op: "LT"},
			wantOp:      ptr.To("GT"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			LateInitializeQualityGateCondition(tc.params, tc.observation)
			if tc.params == nil {
				return
			}
			if tc.wantOp == nil && tc.params.Op != nil {
				t.Errorf("LateInitializeQualityGateCondition() Op = %v, want nil", *tc.params.Op)
				return
			}
			if tc.wantOp != nil && tc.params.Op == nil {
				t.Errorf("LateInitializeQualityGateCondition() Op = nil, want %v", *tc.wantOp)
				return
			}
			if tc.wantOp != nil && tc.params.Op != nil && *tc.params.Op != *tc.wantOp {
				t.Errorf("LateInitializeQualityGateCondition() Op = %v, want %v", *tc.params.Op, *tc.wantOp)
			}
		})
	}
}

func TestGenerateQualityGateConditionsAssociation(t *testing.T) {
	tests := map[string]struct {
		specs        []v1alpha1.QualityGateConditionParameters
		observations []v1alpha1.QualityGateConditionObservation
		wantKeys     []string
	}{
		"EmptyInputsReturnsEmptyMap": {
			specs:        []v1alpha1.QualityGateConditionParameters{},
			observations: []v1alpha1.QualityGateConditionObservation{},
			wantKeys:     []string{},
		},
		"OnlyObservationsCreatesOrphanedEntries": {
			specs: []v1alpha1.QualityGateConditionParameters{},
			observations: []v1alpha1.QualityGateConditionObservation{
				{ID: "1", Metric: "coverage"},
				{ID: "2", Metric: "duplicated_lines"},
			},
			wantKeys: []string{"1", "2"},
		},
		"SpecsWithIDsMatchObservations": {
			specs: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("1"), Metric: "coverage", Error: "80"},
			},
			observations: []v1alpha1.QualityGateConditionObservation{
				{ID: "1", Metric: "coverage", Error: "80"},
			},
			wantKeys: []string{"1"},
		},
		"SpecsWithoutIDsCreateNewEntries": {
			specs: []v1alpha1.QualityGateConditionParameters{
				{Metric: "coverage", Error: "80"},
			},
			observations: []v1alpha1.QualityGateConditionObservation{},
			wantKeys:     []string{"new:coverage"},
		},
		"MixedSpecsAndObservations": {
			specs: []v1alpha1.QualityGateConditionParameters{
				{Id: ptr.To("1"), Metric: "coverage", Error: "80"},
				{Metric: "new_metric", Error: "50"},
			},
			observations: []v1alpha1.QualityGateConditionObservation{
				{ID: "1", Metric: "coverage", Error: "80"},
				{ID: "2", Metric: "orphaned", Error: "10"},
			},
			wantKeys: []string{"1", "2", "new:new_metric"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateConditionsAssociation(tc.specs, tc.observations)
			if len(got) != len(tc.wantKeys) {
				t.Errorf("GenerateQualityGateConditionsAssociation() returned %d associations, want %d", len(got), len(tc.wantKeys))
			}
			for _, key := range tc.wantKeys {
				if _, exists := got[key]; !exists {
					t.Errorf("GenerateQualityGateConditionsAssociation() missing key %q", key)
				}
			}
		})
	}
}

func TestAreQualityGateConditionsUpToDate(t *testing.T) {
	tests := map[string]struct {
		associations map[string]QualityGateConditionAssociation
		want         bool
	}{
		"EmptyMapReturnsTrue": {
			associations: map[string]QualityGateConditionAssociation{},
			want:         true,
		},
		"NilMapReturnsTrue": {
			associations: nil,
			want:         true,
		},
		"AllUpToDateReturnsTrue": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {UpToDate: true},
				"2": {UpToDate: true},
			},
			want: true,
		},
		"AnyNotUpToDateReturnsFalse": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {UpToDate: true},
				"2": {UpToDate: false},
			},
			want: false,
		},
		"SingleNotUpToDateReturnsFalse": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {UpToDate: false},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := AreQualityGateConditionsUpToDate(tc.associations)
			if got != tc.want {
				t.Errorf("AreQualityGateConditionsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindNonExistingQualityGateConditions(t *testing.T) {
	tests := map[string]struct {
		associations map[string]QualityGateConditionAssociation
		wantCount    int
	}{
		"EmptyMapReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{},
			wantCount:    0,
		},
		"AllExistingReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1")},
				},
			},
			wantCount: 0,
		},
		"NonExistingReturnsSpecs": {
			associations: map[string]QualityGateConditionAssociation{
				"new:coverage": {
					Observation: nil,
					Spec:        &v1alpha1.QualityGateConditionParameters{Metric: "coverage"},
				},
			},
			wantCount: 1,
		},
		"MixedReturnsOnlyNonExisting": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1")},
				},
				"new:metric": {
					Observation: nil,
					Spec:        &v1alpha1.QualityGateConditionParameters{Metric: "metric"},
				},
			},
			wantCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FindNonExistingQualityGateConditions(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNonExistingQualityGateConditions() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFindMissingQualityGateConditions(t *testing.T) {
	tests := map[string]struct {
		associations map[string]QualityGateConditionAssociation
		wantCount    int
	}{
		"EmptyMapReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{},
			wantCount:    0,
		},
		"AllMatchedReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1")},
				},
			},
			wantCount: 0,
		},
		"OrphanedObservationsReturnsMissing": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        nil,
				},
			},
			wantCount: 1,
		},
		"MixedReturnsOnlyOrphaned": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1")},
				},
				"2": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "2"},
					Spec:        nil,
				},
			},
			wantCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FindMissingQualityGateConditions(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindMissingQualityGateConditions() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFindNotUpToDateQualityGateConditions(t *testing.T) {
	tests := map[string]struct {
		associations map[string]QualityGateConditionAssociation
		wantCount    int
	}{
		"EmptyMapReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{},
			wantCount:    0,
		},
		"AllUpToDateReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1")},
					UpToDate:    true,
				},
			},
			wantCount: 0,
		},
		"NotUpToDateWithBothSpecAndObservation": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1", Error: "80"},
					Spec:        &v1alpha1.QualityGateConditionParameters{Id: ptr.To("1"), Error: "90"},
					UpToDate:    false,
				},
			},
			wantCount: 1,
		},
		"NotUpToDateButMissingObservationReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{
				"new:coverage": {
					Observation: nil,
					Spec:        &v1alpha1.QualityGateConditionParameters{Metric: "coverage"},
					UpToDate:    false,
				},
			},
			wantCount: 0,
		},
		"NotUpToDateButMissingSpecReturnsEmpty": {
			associations: map[string]QualityGateConditionAssociation{
				"1": {
					Observation: &v1alpha1.QualityGateConditionObservation{ID: "1"},
					Spec:        nil,
					UpToDate:    false,
				},
			},
			wantCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FindNotUpToDateQualityGateConditions(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNotUpToDateQualityGateConditions() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestGenerateQualityGateConditionObservationFromCreate(t *testing.T) {
	tests := map[string]struct {
		condition *sonargo.QualitygatesCreateConditionObject
		want      *v1alpha1.QualityGateConditionObservation
	}{
		"BasicCondition": {
			condition: &sonargo.QualitygatesCreateConditionObject{
				ID:     "123",
				Metric: "coverage",
				Op:     "LT",
				Error:  "80",
			},
			want: &v1alpha1.QualityGateConditionObservation{
				ID:     "123",
				Metric: "coverage",
				Op:     "LT",
				Error:  "80",
			},
		},
		"EmptyCondition": {
			condition: &sonargo.QualitygatesCreateConditionObject{},
			want:      &v1alpha1.QualityGateConditionObservation{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityGateConditionObservationFromCreate(tc.condition)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateConditionObservationFromCreate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
