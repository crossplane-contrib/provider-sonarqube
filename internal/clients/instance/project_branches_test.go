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

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestGenerateProjectBranchesListOptions tests generating list options.
func TestGenerateProjectBranchesListOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.ProjectBranchesListOptions
	}{
		"BasicListOption": {
			projectKey: "my-project",
			want: &sonar.ProjectBranchesListOptions{
				Project: "my-project",
			},
		},
		"EmptyProjectKey": {
			projectKey: "",
			want: &sonar.ProjectBranchesListOptions{
				Project: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectBranchesListOptions(tc.projectKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectBranchesListOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectBranchesDeleteOptions tests generating delete options.
func TestGenerateProjectBranchesDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		branchKey  string
		want       *sonar.ProjectBranchesDeleteOptions
	}{
		"BasicDeleteOption": {
			projectKey: "my-project",
			branchKey:  "feature-branch",
			want: &sonar.ProjectBranchesDeleteOptions{
				Project: "my-project",
				Branch:  "feature-branch",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectBranchesDeleteOptions(tc.projectKey, tc.branchKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectBranchesDeleteOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateBranchesObservations tests generating branch observations.
func TestGenerateBranchesObservations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		branches sonar.ProjectBranchesList
		want     map[string]v1alpha1.ProjectBranchObservation
	}{
		"EmptyBranches": {
			branches: sonar.ProjectBranchesList{},
			want:     map[string]v1alpha1.ProjectBranchObservation{},
		},
		"SingleMainBranch": {
			branches: sonar.ProjectBranchesList{
				Branches: []sonar.Branch{
					{
						Name:              "main",
						BranchID:          "branch-id-1",
						IsMain:            true,
						ExcludedFromPurge: true,
						Type:              "LONG",
						Status:            sonar.BranchStatus{QualityGateStatus: "OK"},
					},
				},
			},
			want: map[string]v1alpha1.ProjectBranchObservation{
				"main": {
					Name:              "main",
					ID:                "branch-id-1",
					IsMain:            true,
					ExcludedFromPurge: true,
					Type:              "LONG",
					Status:            v1alpha1.ProjectBranchStatusObservation{QualityGateStatus: "OK"},
				},
			},
		},
		"MultipleBranches": {
			branches: sonar.ProjectBranchesList{
				Branches: []sonar.Branch{
					{
						Name:              "main",
						BranchID:          "id-1",
						IsMain:            true,
						ExcludedFromPurge: true,
						Type:              "LONG",
						Status:            sonar.BranchStatus{QualityGateStatus: "OK"},
					},
					{
						Name:              "develop",
						BranchID:          "id-2",
						IsMain:            false,
						ExcludedFromPurge: false,
						Type:              "LONG",
						Status:            sonar.BranchStatus{QualityGateStatus: "ERROR"},
					},
				},
			},
			want: map[string]v1alpha1.ProjectBranchObservation{
				"main": {
					Name:              "main",
					ID:                "id-1",
					IsMain:            true,
					ExcludedFromPurge: true,
					Type:              "LONG",
					Status:            v1alpha1.ProjectBranchStatusObservation{QualityGateStatus: "OK"},
				},
				"develop": {
					Name:              "develop",
					ID:                "id-2",
					IsMain:            false,
					ExcludedFromPurge: false,
					Type:              "LONG",
					Status:            v1alpha1.ProjectBranchStatusObservation{QualityGateStatus: "ERROR"},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateBranchesObservations(tc.branches)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateBranchesObservations() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAreProjectBranchesUpToDate tests checking branches up to date.
func TestAreProjectBranchesUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        map[string]*v1alpha1.ProjectNewCodePeriodParameters
		observation map[string]v1alpha1.ProjectBranchObservation
		want        bool
	}{
		"BothNil": {
			spec:        nil,
			observation: nil,
			want:        true,
		},
		"NilSpecWithObservation": {
			spec: nil,
			observation: map[string]v1alpha1.ProjectBranchObservation{
				"main": {},
			},
			want: true,
		},
		"SpecBranchNotInObservation": {
			spec: map[string]*v1alpha1.ProjectNewCodePeriodParameters{
				"feature": {Type: "NUMBER_OF_DAYS", Value: new("30")},
			},
			observation: map[string]v1alpha1.ProjectBranchObservation{
				"main": {},
			},
			want: true,
		},
		"MatchingBranchNewCodePeriod": {
			spec: map[string]*v1alpha1.ProjectNewCodePeriodParameters{
				"main": {Type: "NUMBER_OF_DAYS", Value: new("30")},
			},
			observation: map[string]v1alpha1.ProjectBranchObservation{
				"main": {
					NewCodePeriod: v1alpha1.ProjectNewCodePeriodObservation{
						Type:  "NUMBER_OF_DAYS",
						Value: "30",
					},
				},
			},
			want: true,
		},
		"MismatchedBranchNewCodePeriod": {
			spec: map[string]*v1alpha1.ProjectNewCodePeriodParameters{
				"main": {Type: "NUMBER_OF_DAYS", Value: new("30")},
			},
			observation: map[string]v1alpha1.ProjectBranchObservation{
				"main": {
					NewCodePeriod: v1alpha1.ProjectNewCodePeriodObservation{
						Type:  "NUMBER_OF_DAYS",
						Value: "60",
					},
				},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreProjectBranchesUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("AreProjectBranchesUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectMainBranchUpToDate tests checking main branch up to date.
func TestIsProjectMainBranchUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		observedBranches map[string]v1alpha1.ProjectBranchObservation
		mainBranchName   string
		want             bool
	}{
		"EmptyObservation": {
			observedBranches: map[string]v1alpha1.ProjectBranchObservation{},
			mainBranchName:   "main",
			want:             true,
		},
		"MainBranchMatches": {
			observedBranches: map[string]v1alpha1.ProjectBranchObservation{
				"main": {IsMain: true},
			},
			mainBranchName: "main",
			want:           true,
		},
		"MainBranchDoesNotMatch": {
			observedBranches: map[string]v1alpha1.ProjectBranchObservation{
				"master": {IsMain: true},
			},
			mainBranchName: "main",
			want:           false,
		},
		"NoMainBranchMarked": {
			observedBranches: map[string]v1alpha1.ProjectBranchObservation{
				"main":    {IsMain: false},
				"develop": {IsMain: false},
			},
			mainBranchName: "main",
			want:           true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectMainBranchUpToDate(tc.observedBranches, tc.mainBranchName)
			if got != tc.want {
				t.Errorf("IsProjectMainBranchUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateProjectBranchesSetMainOptions tests generating set main options.
func TestGenerateProjectBranchesSetMainOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey     string
		mainBranchName string
		want           *sonar.ProjectBranchesSetMainOptions
	}{
		"BasicSetMainOption": {
			projectKey:     "my-project",
			mainBranchName: "main",
			want: &sonar.ProjectBranchesSetMainOptions{
				Project: "my-project",
				Branch:  "main",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectBranchesSetMainOptions(tc.projectKey, tc.mainBranchName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectBranchesSetMainOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
