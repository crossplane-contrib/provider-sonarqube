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

// TestGenerateProjectsCreateOptions tests creating project creation options.
func TestGenerateProjectsCreateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec v1alpha1.ProjectParameters
		want *sonar.ProjectsCreateOptions
	}{
		"BasicCreateOption": {
			spec: v1alpha1.ProjectParameters{
				Name: "test-project",
				Key:  "test-key",
			},
			want: &sonar.ProjectsCreateOptions{
				Name:    "test-project",
				Project: "test-key",
			},
		},
		"WithVisibility": {
			spec: v1alpha1.ProjectParameters{
				Name:       "test-project",
				Key:        "test-key",
				Visibility: ptr.To("private"),
			},
			want: &sonar.ProjectsCreateOptions{
				Name:       "test-project",
				Project:    "test-key",
				Visibility: "private",
			},
		},
		"WithDefaultBranch": {
			spec: v1alpha1.ProjectParameters{
				Name:          "test-project",
				Key:           "test-key",
				DefaultBranch: ptr.To("develop"),
			},
			want: &sonar.ProjectsCreateOptions{
				Name:       "test-project",
				Project:    "test-key",
				MainBranch: "develop",
			},
		},
		"WithNewCodePeriod": {
			spec: v1alpha1.ProjectParameters{
				Name: "test-project",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			want: &sonar.ProjectsCreateOptions{
				Name:                   "test-project",
				Project:                "test-key",
				NewCodeDefinitionType:  "NUMBER_OF_DAYS",
				NewCodeDefinitionValue: "30",
			},
		},
		"WithNewCodePeriodNoValue": {
			spec: v1alpha1.ProjectParameters{
				Name: "test-project",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type: "PREVIOUS_VERSION",
				},
			},
			want: &sonar.ProjectsCreateOptions{
				Name:                  "test-project",
				Project:               "test-key",
				NewCodeDefinitionType: "PREVIOUS_VERSION",
			},
		},
		"AllFields": {
			spec: v1alpha1.ProjectParameters{
				Name:          "full-project",
				Key:           "full-key",
				Visibility:    ptr.To("public"),
				DefaultBranch: ptr.To("main"),
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "REFERENCE_BRANCH",
					Value: ptr.To("main"),
				},
			},
			want: &sonar.ProjectsCreateOptions{
				Name:                   "full-project",
				Project:                "full-key",
				Visibility:             "public",
				MainBranch:             "main",
				NewCodeDefinitionType:  "REFERENCE_BRANCH",
				NewCodeDefinitionValue: "main",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectsCreateOptions(tc.spec)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectsCreateOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectDeleteOptions tests creating project deletion options.
func TestGenerateProjectDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.ProjectsDeleteOptions
	}{
		"BasicDeleteOption": {
			projectKey: "my-project",
			want: &sonar.ProjectsDeleteOptions{
				Project: "my-project",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectDeleteOptions(tc.projectKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectDeleteOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectSearchOptions tests creating project search options.
func TestGenerateProjectSearchOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.ProjectsSearchOptions
	}{
		"BasicSearchOption": {
			projectKey: "my-project",
			want: &sonar.ProjectsSearchOptions{
				Projects: []string{"my-project"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectSearchOptions(tc.projectKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectSearchOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestUpdateProjectAttributesObservation tests updating project observations.
func TestUpdateProjectAttributesObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		project *sonar.ProjectSearchComponent
		want    v1alpha1.ProjectObservation
	}{
		"BasicUpdate": {
			project: &sonar.ProjectSearchComponent{
				Key:        "test-key",
				Name:       "test-name",
				Qualifier:  "TRK",
				Visibility: "public",
				Revision:   "abc123",
				Managed:    false,
			},
			want: v1alpha1.ProjectObservation{
				Key:        "test-key",
				Name:       "test-name",
				Qualifier:  "TRK",
				Visibility: "public",
				Revision:   "abc123",
				Managed:    false,
			},
		},
		"ManagedProject": {
			project: &sonar.ProjectSearchComponent{
				Key:         "managed-key",
				Name:        "managed-name",
				Qualifier:   "TRK",
				Visibility:  "private",
				Revision:    "def456",
				ProjectUuid: "uuid-123",
				Managed:     true,
			},
			want: v1alpha1.ProjectObservation{
				Key:        "managed-key",
				Name:       "managed-name",
				Qualifier:  "TRK",
				Visibility: "private",
				Revision:   "def456",
				Uuid:       "uuid-123",
				Managed:    true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := v1alpha1.ProjectObservation{}
			UpdateProjectAttributesObservation(&got, tc.project)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("UpdateProjectAttributesObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLateInitializeProject tests late initialization of projects.
func TestLateInitializeProject(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec          *v1alpha1.ProjectParameters
		observation   *v1alpha1.ProjectObservation
		wantVisib     *string
		wantNCPValue  *string // nil means spec.NewCodePeriod is nil or has no change
		checkNCPValue bool    // set true to assert spec.NewCodePeriod.Value after call
	}{
		"VisibilityAlreadySet": {
			spec: &v1alpha1.ProjectParameters{
				Visibility: ptr.To("private"),
			},
			observation: &v1alpha1.ProjectObservation{
				Visibility: "public",
			},
			wantVisib: ptr.To("private"),
		},
		"VisibilityNilGetsInitialized": {
			spec: &v1alpha1.ProjectParameters{},
			observation: &v1alpha1.ProjectObservation{
				Visibility: "public",
			},
			wantVisib: ptr.To("public"),
		},
		// When spec.NewCodePeriod is nil, LateInitializeProject must not panic or change
		// the field (the nil guard in LateInitializeProjectNewCodePeriod should short-circuit).
		"NewCodePeriodNilSpecIsNoOp": {
			spec: &v1alpha1.ProjectParameters{
				Visibility: ptr.To("public"),
				// NewCodePeriod intentionally left nil
			},
			observation: &v1alpha1.ProjectObservation{
				Visibility: "public",
				NewCodePeriod: v1alpha1.ProjectNewCodePeriodObservation{
					Type:      "PREVIOUS_VERSION",
					Inherited: true,
				},
			},
			wantVisib:     ptr.To("public"),
			checkNCPValue: false, // spec.NewCodePeriod stays nil; nothing to assert
		},
		// When spec.NewCodePeriod is set but Value is nil, the observation's value should
		// be back-filled so that drift detection works correctly.
		"NewCodePeriodValueNilIsInitialized": {
			spec: &v1alpha1.ProjectParameters{
				Visibility: ptr.To("public"),
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type: "NUMBER_OF_DAYS",
					// Value intentionally nil: should be set from observation
				},
			},
			observation: &v1alpha1.ProjectObservation{
				Visibility: "public",
				NewCodePeriod: v1alpha1.ProjectNewCodePeriodObservation{
					Type:  "NUMBER_OF_DAYS",
					Value: "30",
				},
			},
			wantVisib:     ptr.To("public"),
			wantNCPValue:  ptr.To("30"),
			checkNCPValue: true,
		},
		// When spec.NewCodePeriod.Value is already set, it must not be overwritten.
		"NewCodePeriodValueAlreadySetIsPreserved": {
			spec: &v1alpha1.ProjectParameters{
				Visibility: ptr.To("public"),
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("14"),
				},
			},
			observation: &v1alpha1.ProjectObservation{
				Visibility: "public",
				NewCodePeriod: v1alpha1.ProjectNewCodePeriodObservation{
					Type:  "NUMBER_OF_DAYS",
					Value: "30",
				},
			},
			wantVisib:     ptr.To("public"),
			wantNCPValue:  ptr.To("14"),
			checkNCPValue: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeProject(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantVisib, tc.spec.Visibility); diff != "" {
				t.Errorf("LateInitializeProject() visibility mismatch (-want +got):\n%s", diff)
			}

			if tc.checkNCPValue {
				if tc.spec.NewCodePeriod == nil {
					t.Errorf("LateInitializeProject() spec.NewCodePeriod is nil, expected non-nil")
				} else if diff := cmp.Diff(tc.wantNCPValue, tc.spec.NewCodePeriod.Value); diff != "" {
					t.Errorf("LateInitializeProject() NewCodePeriod.Value mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestIsProjectUpToDate tests checking if project is up to date.
func TestIsProjectUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.ProjectParameters
		observation *v1alpha1.ProjectObservation
		want        bool
	}{
		"NilObservation": {
			spec:        &v1alpha1.ProjectParameters{Name: "proj", Key: "key"},
			observation: nil,
			want:        false,
		},
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.ProjectObservation{Name: "proj", Key: "key"},
			want:        true,
		},
		"MatchingBasicFields": {
			spec: &v1alpha1.ProjectParameters{
				Name: "proj",
				Key:  "key",
			},
			observation: &v1alpha1.ProjectObservation{
				Name: "proj",
				Key:  "key",
			},
			want: true,
		},
		"NameMismatch": {
			spec: &v1alpha1.ProjectParameters{
				Name: "proj-new",
				Key:  "key",
			},
			observation: &v1alpha1.ProjectObservation{
				Name: "proj",
				Key:  "key",
			},
			want: false,
		},
		"KeyMismatch": {
			spec: &v1alpha1.ProjectParameters{
				Name: "proj",
				Key:  "new-key",
			},
			observation: &v1alpha1.ProjectObservation{
				Name: "proj",
				Key:  "key",
			},
			want: false,
		},
		"VisibilityMismatch": {
			spec: &v1alpha1.ProjectParameters{
				Name:       "proj",
				Key:        "key",
				Visibility: ptr.To("private"),
			},
			observation: &v1alpha1.ProjectObservation{
				Name:       "proj",
				Key:        "key",
				Visibility: "public",
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsProjectUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateProjectUpdateVisibilityOptions tests generating
// visibility options.
func TestGenerateProjectUpdateVisibilityOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		visibility string
		want       *sonar.ProjectsUpdateVisibilityOptions
	}{
		"BasicUpdateVisibility": {
			projectKey: "my-project",
			visibility: "private",
			want: &sonar.ProjectsUpdateVisibilityOptions{
				Project:    "my-project",
				Visibility: "private",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectUpdateVisibilityOptions(tc.projectKey, tc.visibility)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectUpdateVisibilityOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsProjectLateInitializedNoChanges tests no changes case.
func TestIsProjectLateInitializedNoChanges(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"NoChanges": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			want: false,
		},
		"OnlyNothingElseSet": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLateInitializedVisibility tests visibility initialization.
func TestIsProjectLateInitializedVisibility(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"VisibilityChanged": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("private"),
			},
			want: true,
		},
		"VisibilityNilToSet": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
			},
			want: true,
		},
		"VisibilitySetToNil": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLateInitializedLinks tests links initialization.
func TestIsProjectLateInitializedLinks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"LinksAdded": {
			former: &v1alpha1.ProjectParameters{
				Name:  "test-proj",
				Key:   "test-key",
				Links: []v1alpha1.ProjectLinkParameters{},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
				},
			},
			want: true,
		},
		"LinksRemoved": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name:  "test-proj",
				Key:   "test-key",
				Links: []v1alpha1.ProjectLinkParameters{},
			},
			want: true,
		},
		"LinksModified": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://old.example.com",
					},
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://new.example.com",
					},
				},
			},
			want: true,
		},
		"LinksNilEqualsEmpty": {
			former: &v1alpha1.ProjectParameters{
				Name:  "test-proj",
				Key:   "test-key",
				Links: nil,
			},
			current: &v1alpha1.ProjectParameters{
				Name:  "test-proj",
				Key:   "test-key",
				Links: []v1alpha1.ProjectLinkParameters{},
			},
			want: false, // EquateEmpty treats nil and empty slice as equal
		},
		"MultipleLinks": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
					{
						Name: "CI",
						URL:  "https://ci.example.com",
					},
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLateInitializedNewCodePeriod tests new code period init.
func TestIsProjectLateInitializedNewCodePeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"NewCodePeriodChanged": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			want: true,
		},
		"NewCodePeriodValueAdded": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			want: true,
		},
		"NewCodePeriodValueRemoved": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: nil,
				},
			},
			want: true,
		},
		"NewCodePeriodNilToSet": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			want: true,
		},
		"NewCodePeriodSetToNil": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
			},
			want: true,
		},
		"NewCodePeriodTypeChanged": {
			former: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name: "test-proj",
				Key:  "test-key",
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "REFERENCE_BRANCH",
					Value: ptr.To("main"),
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLateInitializedCombined tests combined initialization.
func TestIsProjectLateInitializedCombined(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"VisibilityAndLinksChanged": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("private"),
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
				},
			},
			want: true,
		},
		"VisibilityAndNewCodePeriodChanged": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("private"),
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			want: true,
		},
		"AllFieldsChanged": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "PREVIOUS_VERSION",
					Value: nil,
				},
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("private"),
				Links: []v1alpha1.ProjectLinkParameters{
					{
						Name: "Homepage",
						URL:  "https://example.com",
					},
				},
				NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
					Type:  "NUMBER_OF_DAYS",
					Value: ptr.To("30"),
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLateInitializedEdgeCases tests edge cases.
func TestIsProjectLateInitializedEdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		former  *v1alpha1.ProjectParameters
		current *v1alpha1.ProjectParameters
		want    bool
	}{
		"SameValuesNoInitialization": {
			former: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
			},
			current: &v1alpha1.ProjectParameters{
				Name:       "test-proj",
				Key:        "test-key",
				Visibility: ptr.To("public"),
				Links:      []v1alpha1.ProjectLinkParameters{},
			},
			want: false,
		},
		"DifferentStringValues": {
			former: &v1alpha1.ProjectParameters{
				Name: "proj-a",
				Key:  "key-a",
			},
			current: &v1alpha1.ProjectParameters{
				Name: "proj-b",
				Key:  "key-b",
			},
			want: false, // Only Visibility, Links, and NewCodePeriod are checked
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsProjectLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}
