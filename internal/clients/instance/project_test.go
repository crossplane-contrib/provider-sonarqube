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

func TestGenerateProjectsCreateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec v1alpha1.ProjectParameters
		want *sonar.ProjectsCreateOption
	}{
		"BasicCreateOption": {
			spec: v1alpha1.ProjectParameters{
				Name: "test-project",
				Key:  "test-key",
			},
			want: &sonar.ProjectsCreateOption{
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
			want: &sonar.ProjectsCreateOption{
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
			want: &sonar.ProjectsCreateOption{
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
			want: &sonar.ProjectsCreateOption{
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
			want: &sonar.ProjectsCreateOption{
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
			want: &sonar.ProjectsCreateOption{
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

func TestGenerateProjectDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.ProjectsDeleteOption
	}{
		"BasicDeleteOption": {
			projectKey: "my-project",
			want: &sonar.ProjectsDeleteOption{
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

func TestGenerateProjectSearchOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.ProjectsSearchOption
	}{
		"BasicSearchOption": {
			projectKey: "my-project",
			want: &sonar.ProjectsSearchOption{
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

func TestLateInitializeProject(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.ProjectParameters
		observation *v1alpha1.ProjectObservation
		wantVisib   *string
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeProject(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantVisib, tc.spec.Visibility); diff != "" {
				t.Errorf("LateInitializeProject() visibility mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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

func TestGenerateProjectUpdateVisibilityOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		visibility string
		want       *sonar.ProjectsUpdateVisibilityOption
	}{
		"BasicUpdateVisibility": {
			projectKey: "my-project",
			visibility: "private",
			want: &sonar.ProjectsUpdateVisibilityOption{
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
