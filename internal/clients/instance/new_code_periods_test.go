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

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

func TestGenerateNewCodePeriodsShowOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey *string
		branch     *string
		want       *sonar.NewCodePeriodsShowOptions
	}{
		"BothNil": {
			projectKey: nil,
			branch:     nil,
			want:       &sonar.NewCodePeriodsShowOptions{},
		},
		"ProjectKeyOnly": {
			projectKey: ptr.To("my-project"),
			branch:     nil,
			want: &sonar.NewCodePeriodsShowOptions{
				Project: "my-project",
			},
		},
		"BothSet": {
			projectKey: ptr.To("my-project"),
			branch:     ptr.To("main"),
			want: &sonar.NewCodePeriodsShowOptions{
				Project: "my-project",
				Branch:  "main",
			},
		},
		"BranchOnly": {
			projectKey: nil,
			branch:     ptr.To("develop"),
			want: &sonar.NewCodePeriodsShowOptions{
				Branch: "develop",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateNewCodePeriodsShowOptions(tc.projectKey, tc.branch)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateNewCodePeriodsShowOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateProjectNewCodePeriodsSetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		params     *v1alpha1.ProjectNewCodePeriodParameters
		want       *sonar.NewCodePeriodsSetOptions
	}{
		"NilParams": {
			projectKey: "my-project",
			params:     nil,
			want: &sonar.NewCodePeriodsSetOptions{
				Project: "my-project",
			},
		},
		"WithTypeOnly": {
			projectKey: "my-project",
			params: &v1alpha1.ProjectNewCodePeriodParameters{
				Type: "PREVIOUS_VERSION",
			},
			want: &sonar.NewCodePeriodsSetOptions{
				Project: "my-project",
				Type:    "PREVIOUS_VERSION",
			},
		},
		"WithTypeAndValue": {
			projectKey: "my-project",
			params: &v1alpha1.ProjectNewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
			want: &sonar.NewCodePeriodsSetOptions{
				Project: "my-project",
				Type:    "NUMBER_OF_DAYS",
				Value:   "30",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectNewCodePeriodsSetOptions(tc.projectKey, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectNewCodePeriodsSetOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateBranchNewCodePeriodsSetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		branchName string
		params     *v1alpha1.ProjectNewCodePeriodParameters
		want       *sonar.NewCodePeriodsSetOptions
	}{
		"NilParams": {
			projectKey: "my-project",
			branchName: "main",
			params:     nil,
			want: &sonar.NewCodePeriodsSetOptions{
				Project: "my-project",
				Branch:  "main",
			},
		},
		"WithParams": {
			projectKey: "my-project",
			branchName: "develop",
			params: &v1alpha1.ProjectNewCodePeriodParameters{
				Type:  "REFERENCE_BRANCH",
				Value: ptr.To("main"),
			},
			want: &sonar.NewCodePeriodsSetOptions{
				Project: "my-project",
				Branch:  "develop",
				Type:    "REFERENCE_BRANCH",
				Value:   "main",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateBranchNewCodePeriodsSetOptions(tc.projectKey, tc.branchName, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateBranchNewCodePeriodsSetOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateProjectNewCodePeriodsListOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.NewCodePeriodsListOptions
	}{
		"BasicListOption": {
			projectKey: "my-project",
			want: &sonar.NewCodePeriodsListOptions{
				Project: "my-project",
			},
		},
		"EmptyProjectKey": {
			projectKey: "",
			want: &sonar.NewCodePeriodsListOptions{
				Project: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectNewCodePeriodsListOptions(tc.projectKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectNewCodePeriodsListOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsNewCodePeriodUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.ProjectNewCodePeriodParameters
		observation *v1alpha1.ProjectNewCodePeriodObservation
		want        bool
	}{
		"NilObservation": {
			spec:        &v1alpha1.ProjectNewCodePeriodParameters{Type: "PREVIOUS_VERSION"},
			observation: nil,
			want:        false,
		},
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.ProjectNewCodePeriodObservation{Type: "PREVIOUS_VERSION"},
			want:        true,
		},
		"BothNil": {
			spec:        nil,
			observation: nil,
			want:        false,
		},
		"MatchingTypeNoValue": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type: "PREVIOUS_VERSION",
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Type: "PREVIOUS_VERSION",
			},
			want: true,
		},
		"MatchingTypeAndValue": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Type:  "NUMBER_OF_DAYS",
				Value: "30",
			},
			want: true,
		},
		"DifferentType": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type: "PREVIOUS_VERSION",
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Type: "NUMBER_OF_DAYS",
			},
			want: false,
		},
		"DifferentValue": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Type:  "NUMBER_OF_DAYS",
				Value: "60",
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsNewCodePeriodUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsNewCodePeriodUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateProjectNewCodePeriodObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		obs  *sonar.NewCodePeriodsShow
		want v1alpha1.ProjectNewCodePeriodObservation
	}{
		"BasicObservation": {
			obs: &sonar.NewCodePeriodsShow{
				Type:      "NUMBER_OF_DAYS",
				Value:     "30",
				Inherited: false,
			},
			want: v1alpha1.ProjectNewCodePeriodObservation{
				Type:      "NUMBER_OF_DAYS",
				Value:     "30",
				Inherited: false,
			},
		},
		"InheritedObservation": {
			obs: &sonar.NewCodePeriodsShow{
				Type:      "PREVIOUS_VERSION",
				Value:     "",
				Inherited: true,
			},
			want: v1alpha1.ProjectNewCodePeriodObservation{
				Type:      "PREVIOUS_VERSION",
				Value:     "",
				Inherited: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectNewCodePeriodObservation(tc.obs)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectNewCodePeriodObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateBranchNewCodePeriodObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		obs  *sonar.NewCodePeriod
		want v1alpha1.ProjectNewCodePeriodObservation
	}{
		"BasicObservation": {
			obs: &sonar.NewCodePeriod{
				Type:           "NUMBER_OF_DAYS",
				Value:          "30",
				Inherited:      false,
				EffectiveValue: "30",
			},
			want: v1alpha1.ProjectNewCodePeriodObservation{
				Type:           "NUMBER_OF_DAYS",
				Value:          "30",
				Inherited:      false,
				EffectiveValue: "30",
			},
		},
		"InheritedObservation": {
			obs: &sonar.NewCodePeriod{
				Type:           "PREVIOUS_VERSION",
				Value:          "",
				Inherited:      true,
				EffectiveValue: "1.0",
			},
			want: v1alpha1.ProjectNewCodePeriodObservation{
				Type:           "PREVIOUS_VERSION",
				Value:          "",
				Inherited:      true,
				EffectiveValue: "1.0",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateBranchNewCodePeriodObservation(tc.obs)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateBranchNewCodePeriodObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLateInitializeProjectNewCodePeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.ProjectNewCodePeriodParameters
		observation *v1alpha1.ProjectNewCodePeriodObservation
		wantValue   *string
	}{
		"ValueAlreadySet": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Value: "60",
			},
			wantValue: ptr.To("30"),
		},
		"ValueNilGetsInitialized": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type: "NUMBER_OF_DAYS",
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Value: "45",
			},
			wantValue: ptr.To("45"),
		},
		"EmptyObservationValue": {
			spec: &v1alpha1.ProjectNewCodePeriodParameters{
				Type: "PREVIOUS_VERSION",
			},
			observation: &v1alpha1.ProjectNewCodePeriodObservation{
				Value: "",
			},
			wantValue: ptr.To(""),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeProjectNewCodePeriod(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantValue, tc.spec.Value); diff != "" {
				t.Errorf("LateInitializeProjectNewCodePeriod() value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
