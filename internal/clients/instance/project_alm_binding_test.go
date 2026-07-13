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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestGenerateProjectALMBindingObservation tests generating the ALM binding
// observation from a SonarQube API response.
func TestGenerateProjectALMBindingObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		binding *sonar.AlmSettingsGetBinding
		want    *v1alpha1.ProjectALMBindingObservation
	}{
		"Nil": {
			binding: nil,
			want:    nil,
		},
		"GitHubBinding": {
			binding: &sonar.AlmSettingsGetBinding{
				Alm:                   "github",
				Key:                   "my-github",
				Repository:            "org/repo",
				RepositoryURL:         "https://github.com/org/repo",
				Monorepo:              true,
				SummaryCommentEnabled: true,
			},
			want: &v1alpha1.ProjectALMBindingObservation{
				Alm:                   "github",
				Key:                   "my-github",
				Repository:            "org/repo",
				RepositoryURL:         "https://github.com/org/repo",
				Monorepo:              true,
				SummaryCommentEnabled: true,
			},
		},
		"BitbucketBinding": {
			binding: &sonar.AlmSettingsGetBinding{
				Alm:        "bitbucket",
				Key:        "my-bitbucket",
				Repository: "PROJ",
				Slug:       "repo-slug",
			},
			want: &v1alpha1.ProjectALMBindingObservation{
				Alm:        "bitbucket",
				Key:        "my-bitbucket",
				Repository: "PROJ",
				Slug:       "repo-slug",
			},
		},
		"AzureBinding": {
			binding: &sonar.AlmSettingsGetBinding{
				Alm:                      "azure",
				Key:                      "my-azure",
				Monorepo:                 true,
				InlineAnnotationsEnabled: true,
			},
			want: &v1alpha1.ProjectALMBindingObservation{
				Alm:                      "azure",
				Key:                      "my-azure",
				Monorepo:                 true,
				InlineAnnotationsEnabled: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectALMBindingObservation(tc.binding)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectALMBindingObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectALMBindingGetOptions tests generating get options.
func TestGenerateProjectALMBindingGetOptions(t *testing.T) {
	t.Parallel()

	got := GenerateProjectALMBindingGetOptions("my-project")
	want := &sonar.AlmSettingsGetBindingOptions{Project: "my-project"}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectALMBindingGetOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectALMBindingDeleteOptions tests generating delete options.
func TestGenerateProjectALMBindingDeleteOptions(t *testing.T) {
	t.Parallel()

	got := GenerateProjectALMBindingDeleteOptions("my-project")
	want := &sonar.AlmSettingsDeleteBindingOptions{Project: "my-project"}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectALMBindingDeleteOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectGitHubBindingOptions tests generating GitHub bind
// options.
func TestGenerateProjectGitHubBindingOptions(t *testing.T) {
	t.Parallel()

	binding := &v1alpha1.ProjectALMBindingParameters{
		Monorepo: true,
		GitHub: &v1alpha1.ProjectGitHubBindingParameters{
			AlmSettingKey:         "my-github",
			Repository:            "org/repo",
			SummaryCommentEnabled: new(true),
		},
	}

	got := GenerateProjectGitHubBindingOptions("my-project", binding)
	want := &sonar.AlmSettingsSetGithubBindingOptions{
		AlmSetting:            "my-github",
		Project:               "my-project",
		Repository:            "org/repo",
		Monorepo:              true,
		SummaryCommentEnabled: new(true),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectGitHubBindingOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectGitLabBindingOptions tests generating GitLab bind
// options.
func TestGenerateProjectGitLabBindingOptions(t *testing.T) {
	t.Parallel()

	binding := &v1alpha1.ProjectALMBindingParameters{
		Monorepo: true,
		GitLab: &v1alpha1.ProjectGitLabBindingParameters{
			AlmSettingKey: "my-gitlab",
			Repository:    "42",
		},
	}

	got := GenerateProjectGitLabBindingOptions("my-project", binding)
	want := &sonar.AlmSettingsSetGitlabBindingOptions{
		AlmSetting: "my-gitlab",
		Project:    "my-project",
		Repository: "42",
		Monorepo:   true,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectGitLabBindingOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectAzureBindingOptions tests generating Azure bind
// options.
func TestGenerateProjectAzureBindingOptions(t *testing.T) {
	t.Parallel()

	binding := &v1alpha1.ProjectALMBindingParameters{
		Monorepo: true,
		Azure: &v1alpha1.ProjectAzureBindingParameters{
			AlmSettingKey:            "my-azure",
			ProjectName:              "azure-project",
			RepositoryName:           "azure-repo",
			InlineAnnotationsEnabled: new(true),
		},
	}

	got := GenerateProjectAzureBindingOptions("my-project", binding)
	want := &sonar.AlmSettingsSetAzureBindingOptions{
		AlmSetting:               "my-azure",
		Project:                  "my-project",
		ProjectName:              "azure-project",
		RepositoryName:           "azure-repo",
		Monorepo:                 true,
		InlineAnnotationsEnabled: new(true),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectAzureBindingOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectBitbucketBindingOptions tests generating Bitbucket
// Server bind options.
func TestGenerateProjectBitbucketBindingOptions(t *testing.T) {
	t.Parallel()

	binding := &v1alpha1.ProjectALMBindingParameters{
		Monorepo: true,
		Bitbucket: &v1alpha1.ProjectBitbucketBindingParameters{
			AlmSettingKey: "my-bitbucket",
			Repository:    "PROJ",
			Slug:          "repo-slug",
		},
	}

	got := GenerateProjectBitbucketBindingOptions("my-project", binding)
	want := &sonar.AlmSettingsSetBitbucketBindingOptions{
		AlmSetting: "my-bitbucket",
		Project:    "my-project",
		Repository: "PROJ",
		Slug:       "repo-slug",
		Monorepo:   true,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectBitbucketBindingOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateProjectBitbucketCloudBindingOptions tests generating
// Bitbucket Cloud bind options.
func TestGenerateProjectBitbucketCloudBindingOptions(t *testing.T) {
	t.Parallel()

	binding := &v1alpha1.ProjectALMBindingParameters{
		Monorepo: true,
		BitbucketCloud: &v1alpha1.ProjectBitbucketCloudBindingParameters{
			AlmSettingKey: "my-bitbucket-cloud",
			Repository:    "workspace/repo",
		},
	}

	got := GenerateProjectBitbucketCloudBindingOptions("my-project", binding)
	want := &sonar.AlmSettingsSetBitbucketCloudBindingOptions{
		AlmSetting: "my-bitbucket-cloud",
		Project:    "my-project",
		Repository: "workspace/repo",
		Monorepo:   true,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateProjectBitbucketCloudBindingOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestIsProjectALMBindingUpToDate tests checking whether a Project's ALM
// binding is up to date.
func TestIsProjectALMBindingUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec     *v1alpha1.ProjectALMBindingParameters
		observed *v1alpha1.ProjectALMBindingObservation
		want     bool
	}{
		"BothNil": {
			spec:     nil,
			observed: nil,
			want:     true,
		},
		"SpecNilObservedSet": {
			spec: nil,
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "github",
			},
			want: false,
		},
		"SpecSetObservedNil": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				GitHub: &v1alpha1.ProjectGitHubBindingParameters{AlmSettingKey: "gh", Repository: "org/repo"},
			},
			observed: nil,
			want:     false,
		},
		"NoDiscriminatorSet": {
			spec:     &v1alpha1.ProjectALMBindingParameters{},
			observed: &v1alpha1.ProjectALMBindingObservation{Alm: "github"},
			want:     false,
		},
		"GitHubMatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Monorepo: true,
				GitHub: &v1alpha1.ProjectGitHubBindingParameters{
					AlmSettingKey:         "gh",
					Repository:            "org/repo",
					SummaryCommentEnabled: new(true),
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm:                   "github",
				Key:                   "gh",
				Repository:            "org/repo",
				Monorepo:              true,
				SummaryCommentEnabled: true,
			},
			want: true,
		},
		"GitHubRepositoryMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				GitHub: &v1alpha1.ProjectGitHubBindingParameters{AlmSettingKey: "gh", Repository: "org/repo"},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "github", Key: "gh", Repository: "org/other",
			},
			want: false,
		},
		"GitHubOptionalFieldNilAlwaysMatches": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				GitHub: &v1alpha1.ProjectGitHubBindingParameters{AlmSettingKey: "gh", Repository: "org/repo"},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "github", Key: "gh", Repository: "org/repo", SummaryCommentEnabled: true,
			},
			want: true,
		},
		"GitLabMatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				GitLab: &v1alpha1.ProjectGitLabBindingParameters{AlmSettingKey: "gl", Repository: "42"},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{Alm: "gitlab", Key: "gl", Repository: "42"},
			want:     true,
		},
		"GitLabAlmTypeMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				GitLab: &v1alpha1.ProjectGitLabBindingParameters{AlmSettingKey: "gl", Repository: "42"},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{Alm: "github", Key: "gl", Repository: "42"},
			want:     false,
		},
		"AzureMatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Azure: &v1alpha1.ProjectAzureBindingParameters{
					AlmSettingKey:            "az",
					ProjectName:              "azure-project",
					RepositoryName:           "azure-repo",
					InlineAnnotationsEnabled: new(true),
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "azure", Key: "az", InlineAnnotationsEnabled: true,
			},
			want: true,
		},
		"AzureKeyMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Azure: &v1alpha1.ProjectAzureBindingParameters{
					AlmSettingKey:  "az",
					ProjectName:    "azure-project",
					RepositoryName: "azure-repo",
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{Alm: "azure", Key: "other-az"},
			want:     false,
		},
		"BitbucketMatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Bitbucket: &v1alpha1.ProjectBitbucketBindingParameters{
					AlmSettingKey: "bb", Repository: "PROJ", Slug: "repo-slug",
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "bitbucket", Key: "bb", Repository: "PROJ", Slug: "repo-slug",
			},
			want: true,
		},
		"BitbucketSlugMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Bitbucket: &v1alpha1.ProjectBitbucketBindingParameters{
					AlmSettingKey: "bb", Repository: "PROJ", Slug: "repo-slug",
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "bitbucket", Key: "bb", Repository: "PROJ", Slug: "other-slug",
			},
			want: false,
		},
		"BitbucketCloudMatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				BitbucketCloud: &v1alpha1.ProjectBitbucketCloudBindingParameters{
					AlmSettingKey: "bbc", Repository: "workspace/repo",
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "bitbucketcloud", Key: "bbc", Repository: "workspace/repo",
			},
			want: true,
		},
		"BitbucketCloudRepositoryMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				BitbucketCloud: &v1alpha1.ProjectBitbucketCloudBindingParameters{
					AlmSettingKey: "bbc", Repository: "workspace/repo",
				},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "bitbucketcloud", Key: "bbc", Repository: "workspace/other",
			},
			want: false,
		},
		"MonorepoMismatch": {
			spec: &v1alpha1.ProjectALMBindingParameters{
				Monorepo: true,
				GitHub:   &v1alpha1.ProjectGitHubBindingParameters{AlmSettingKey: "gh", Repository: "org/repo"},
			},
			observed: &v1alpha1.ProjectALMBindingObservation{
				Alm: "github", Key: "gh", Repository: "org/repo", Monorepo: false,
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectALMBindingUpToDate(tc.spec, tc.observed)
			if got != tc.want {
				t.Errorf("IsProjectALMBindingUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
