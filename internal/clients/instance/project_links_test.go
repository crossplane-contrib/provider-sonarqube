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

// TestGenerateProjectLinksCreateOptions tests generating create options.
func TestGenerateProjectLinksCreateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectID string
		link      v1alpha1.ProjectLinkParameters
		want      *sonar.ProjectLinksCreateOptions
	}{
		"BasicCreateOption": {
			projectID: "my-project",
			link: v1alpha1.ProjectLinkParameters{
				Name: "homepage",
				URL:  "https://example.com",
			},
			want: &sonar.ProjectLinksCreateOptions{
				ProjectID: "my-project",
				Name:      "homepage",
				URL:       "https://example.com",
			},
		},
		"WithID": {
			projectID: "my-project",
			link: v1alpha1.ProjectLinkParameters{
				ID:   new("link-1"),
				Name: "ci",
				URL:  "https://ci.example.com",
			},
			want: &sonar.ProjectLinksCreateOptions{
				ProjectID: "my-project",
				Name:      "ci",
				URL:       "https://ci.example.com",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectLinksCreateOptions(tc.projectID, tc.link)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectLinksCreateOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectLinksDeleteOptions tests generating delete options.
func TestGenerateProjectLinksDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		linkID string
		want   *sonar.ProjectLinksDeleteOptions
	}{
		"BasicDeleteOption": {
			linkID: "link-123",
			want: &sonar.ProjectLinksDeleteOptions{
				ID: "link-123",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectLinksDeleteOptions(tc.linkID)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectLinksDeleteOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectLinksSearchOptions tests generating search options.
func TestGenerateProjectLinksSearchOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectID string
		want      *sonar.ProjectLinksSearchOptions
	}{
		"BasicSearchOption": {
			projectID: "my-project",
			want: &sonar.ProjectLinksSearchOptions{
				ProjectID: "my-project",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectLinksSearchOptions(tc.projectID)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectLinksSearchOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLateInitializeProjectLinks tests late initialization of links.
func TestLateInitializeProjectLinks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.ProjectParameters
		observation map[string]v1alpha1.ProjectLinkObservation
		wantLinks   []v1alpha1.ProjectLinkParameters
	}{
		"NoLinks": {
			spec:        &v1alpha1.ProjectParameters{},
			observation: nil,
			wantLinks:   nil,
		},
		"LinkAlreadyHasID": {
			spec: &v1alpha1.ProjectParameters{
				Links: []v1alpha1.ProjectLinkParameters{
					{ID: new("existing-id"), Name: "homepage", URL: "https://example.com"},
				},
			},
			observation: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "observed-id", Name: "homepage", URL: "https://example.com"},
			},
			wantLinks: []v1alpha1.ProjectLinkParameters{
				{ID: new("existing-id"), Name: "homepage", URL: "https://example.com"},
			},
		},
		"LinkGetsIDFromObservation": {
			spec: &v1alpha1.ProjectParameters{
				Links: []v1alpha1.ProjectLinkParameters{
					{Name: "homepage", URL: "https://example.com"},
				},
			},
			observation: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "observed-id", Name: "homepage", URL: "https://example.com"},
			},
			wantLinks: []v1alpha1.ProjectLinkParameters{
				{ID: new("observed-id"), Name: "homepage", URL: "https://example.com"},
			},
		},
		"NoMatchingObservation": {
			spec: &v1alpha1.ProjectParameters{
				Links: []v1alpha1.ProjectLinkParameters{
					{Name: "homepage", URL: "https://example.com"},
				},
			},
			observation: map[string]v1alpha1.ProjectLinkObservation{
				"ci": {ID: "ci-id", Name: "ci", URL: "https://ci.example.com"},
			},
			wantLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com"},
			},
		},
		"URLMismatchNoIDAssigned": {
			spec: &v1alpha1.ProjectParameters{
				Links: []v1alpha1.ProjectLinkParameters{
					{Name: "homepage", URL: "https://example.com"},
				},
			},
			observation: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "observed-id", Name: "homepage", URL: "https://other.com"},
			},
			wantLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeProjectLinks(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantLinks, tc.spec.Links); diff != "" {
				t.Errorf("LateInitializeProjectLinks() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateProjectLinksObservations tests generating observations.
func TestGenerateProjectLinksObservations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		links sonar.ProjectLinksSearch
		want  map[string]v1alpha1.ProjectLinkObservation
	}{
		"EmptyLinks": {
			links: sonar.ProjectLinksSearch{},
			want:  map[string]v1alpha1.ProjectLinkObservation{},
		},
		"SingleLink": {
			links: sonar.ProjectLinksSearch{
				Links: []sonar.ProjectLink{
					{ID: "link-1", Name: "homepage", Type: "homepage", URL: "https://example.com"},
				},
			},
			want: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "link-1", Name: "homepage", Type: "homepage", URL: "https://example.com"},
			},
		},
		"MultipleLinks": {
			links: sonar.ProjectLinksSearch{
				Links: []sonar.ProjectLink{
					{ID: "link-1", Name: "homepage", Type: "homepage", URL: "https://example.com"},
					{ID: "link-2", Name: "ci", Type: "ci", URL: "https://ci.example.com"},
				},
			},
			want: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "link-1", Name: "homepage", Type: "homepage", URL: "https://example.com"},
				"ci":       {ID: "link-2", Name: "ci", Type: "ci", URL: "https://ci.example.com"},
			},
		},
		"LinkWithoutType": {
			links: sonar.ProjectLinksSearch{
				Links: []sonar.ProjectLink{
					{ID: "link-3", Name: "GitHub Repository", URL: "https://github.com/example/repo"},
				},
			},
			want: map[string]v1alpha1.ProjectLinkObservation{
				"GitHub Repository": {ID: "link-3", Name: "GitHub Repository", Type: "", URL: "https://github.com/example/repo"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateProjectLinksObservations(tc.links)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateProjectLinksObservations() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAreProjectLinksUpToDate tests checking links up to date.
func TestAreProjectLinksUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specLinks     []v1alpha1.ProjectLinkParameters
		observedLinks map[string]v1alpha1.ProjectLinkObservation
		want          bool
	}{
		"BothEmpty": {
			specLinks:     nil,
			observedLinks: nil,
			want:          true,
		},
		"DifferentLengths": {
			specLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com"},
			},
			observedLinks: nil,
			want:          false,
		},
		"MatchingLinks": {
			specLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com", ID: new("link-1")},
			},
			observedLinks: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "link-1", Name: "homepage", URL: "https://example.com"},
			},
			want: true,
		},
		"LinkNotFoundInObservation": {
			specLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com"},
			},
			observedLinks: map[string]v1alpha1.ProjectLinkObservation{
				"ci": {ID: "link-2", Name: "ci", URL: "https://ci.example.com"},
			},
			want: false,
		},
		"URLMismatch": {
			specLinks: []v1alpha1.ProjectLinkParameters{
				{Name: "homepage", URL: "https://example.com"},
			},
			observedLinks: map[string]v1alpha1.ProjectLinkObservation{
				"homepage": {ID: "link-1", Name: "homepage", URL: "https://other.com"},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreProjectLinksUpToDate(tc.specLinks, tc.observedLinks)
			if got != tc.want {
				t.Errorf("AreProjectLinksUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsProjectLinkUpToDate tests checking link up to date.
func TestIsProjectLinkUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specLink     v1alpha1.ProjectLinkParameters
		observedLink *v1alpha1.ProjectLinkObservation
		want         bool
	}{
		"NilObservation": {
			specLink:     v1alpha1.ProjectLinkParameters{Name: "homepage", URL: "https://example.com"},
			observedLink: nil,
			want:         false,
		},
		"MatchingWithoutID": {
			specLink: v1alpha1.ProjectLinkParameters{Name: "homepage", URL: "https://example.com"},
			observedLink: &v1alpha1.ProjectLinkObservation{
				ID: "link-1", Name: "homepage", URL: "https://example.com",
			},
			want: true,
		},
		"MatchingWithID": {
			specLink: v1alpha1.ProjectLinkParameters{
				ID: new("link-1"), Name: "homepage", URL: "https://example.com",
			},
			observedLink: &v1alpha1.ProjectLinkObservation{
				ID: "link-1", Name: "homepage", URL: "https://example.com",
			},
			want: true,
		},
		"IDMismatch": {
			specLink: v1alpha1.ProjectLinkParameters{
				ID: new("link-2"), Name: "homepage", URL: "https://example.com",
			},
			observedLink: &v1alpha1.ProjectLinkObservation{
				ID: "link-1", Name: "homepage", URL: "https://example.com",
			},
			want: false,
		},
		"NameMismatch": {
			specLink: v1alpha1.ProjectLinkParameters{Name: "ci", URL: "https://example.com"},
			observedLink: &v1alpha1.ProjectLinkObservation{
				ID: "link-1", Name: "homepage", URL: "https://example.com",
			},
			want: false,
		},
		"URLMismatch": {
			specLink: v1alpha1.ProjectLinkParameters{Name: "homepage", URL: "https://other.com"},
			observedLink: &v1alpha1.ProjectLinkObservation{
				ID: "link-1", Name: "homepage", URL: "https://example.com",
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsProjectLinkUpToDate(tc.specLink, tc.observedLink)
			if got != tc.want {
				t.Errorf("IsProjectLinkUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
