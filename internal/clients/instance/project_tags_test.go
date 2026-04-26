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
)

// TestGenerateProjectTagsSetOptions tests generating project tags set options.
func TestGenerateProjectTagsSetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		tags       []string
		want       *sonar.ProjectTagsSetOptions
	}{
		"BasicSetOption": {
			projectKey: "my-project",
			tags:       []string{"tag1", "tag2"},
			want: &sonar.ProjectTagsSetOptions{
				Project: "my-project",
				Tags:    []string{"tag1", "tag2"},
			},
		},
		"EmptyTags": {
			projectKey: "my-project",
			tags:       []string{},
			want: &sonar.ProjectTagsSetOptions{
				Project: "my-project",
				Tags:    []string{},
			},
		},
		"NilTags": {
			projectKey: "my-project",
			tags:       nil,
			want: &sonar.ProjectTagsSetOptions{
				Project: "my-project",
				Tags:    nil,
			},
		},
		"SingleTag": {
			projectKey: "another-project",
			tags:       []string{"solo"},
			want: &sonar.ProjectTagsSetOptions{
				Project: "another-project",
				Tags:    []string{"solo"},
			},
		},
		"EmptyProjectKey": {
			projectKey: "",
			tags:       []string{"tag1"},
			want: &sonar.ProjectTagsSetOptions{
				Project: "",
				Tags:    []string{"tag1"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectTagsSetOptions(tc.projectKey, tc.tags)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectTagsSetOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
