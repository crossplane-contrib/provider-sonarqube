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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestGeneratePortfolioObservation tests the
// GeneratePortfolioObservation function.
func TestGeneratePortfolioObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		details *PortfolioDetails
		want    v1alpha1.PortfolioObservation
	}{
		"NilDetails": {
			details: nil,
			want:    v1alpha1.PortfolioObservation{},
		},
		"EmptyDetails": {
			details: &PortfolioDetails{},
			want:    v1alpha1.PortfolioObservation{},
		},
		"FullDetails": {
			details: &PortfolioDetails{
				ViewDetails: sonar.ViewDetails{
					Key:           "my-portfolio",
					Name:          "My Portfolio",
					Description:   "A test portfolio",
					Qualifier:     "VW",
					Visibility:    "public",
					SelectionMode: "MANUAL",
					SelectedProjects: []sonar.ViewSelectedProject{
						{ProjectKey: "project-b"},
						{ProjectKey: "project-a", SelectedBranches: []string{"main", "feature"}},
					},
				},
				SubViews: []PortfolioSubView{
					{View: sonar.View{Key: "child-b", Qualifier: "SVW"}},
					{View: sonar.View{Key: "my-portfolio:app", OriginalKey: "app", Qualifier: "APP"}, SelectedBranches: []string{"main"}},
					{View: sonar.View{Key: "my-portfolio:referenced", OriginalKey: "referenced", Qualifier: "VW"}},
					{View: sonar.View{Key: "child-a", Qualifier: "SVW"}},
					{View: sonar.View{Key: "unknown", Qualifier: "TRK"}},
				},
			},
			want: v1alpha1.PortfolioObservation{
				Key:           "my-portfolio",
				Name:          "My Portfolio",
				Description:   "A test portfolio",
				Qualifier:     "VW",
				Visibility:    "public",
				SelectionMode: "MANUAL",
				Projects: []v1alpha1.PortfolioMemberObservation{
					{Key: "project-a", Branches: []string{"feature", "main"}},
					{Key: "project-b"},
				},
				Applications:       []v1alpha1.PortfolioMemberObservation{{Key: "app", Branches: []string{"main"}}},
				SubPortfolios:      []string{"referenced"},
				LocalSubPortfolios: []string{"child-a", "child-b"},
			},
		},
		"SubViewWithoutOriginalKey": {
			details: &PortfolioDetails{
				SubViews: []PortfolioSubView{
					{View: sonar.View{Key: "app", Qualifier: "APP"}},
				},
			},
			want: v1alpha1.PortfolioObservation{
				Applications: []v1alpha1.PortfolioMemberObservation{{Key: "app"}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GeneratePortfolioObservation(tc.details)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GeneratePortfolioObservation() -want +got:\n%s", diff)
			}
		})
	}
}

// TestPortfolioDetailsDecoding verifies the show response decodes the
// selected branches of the applications, which sonar.View does not expose.
func TestPortfolioDetailsDecoding(t *testing.T) {
	t.Parallel()

	payload := `{"key":"pf","name":"PF","qualifier":"VW","selectionMode":"MANUAL",` +
		`"selectedProjects":[{"projectKey":"p","selectedBranches":["dev"]}],` +
		`"subViews":[{"key":"pf:app","originalKey":"app","qualifier":"APP","selectedBranches":["main","dev"]}]}`

	var details PortfolioDetails

	err := json.Unmarshal([]byte(payload), &details)
	if err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}

	want := v1alpha1.PortfolioObservation{
		Key:           "pf",
		Name:          "PF",
		Qualifier:     "VW",
		SelectionMode: "MANUAL",
		Projects:      []v1alpha1.PortfolioMemberObservation{{Key: "p", Branches: []string{"dev"}}},
		Applications:  []v1alpha1.PortfolioMemberObservation{{Key: "app", Branches: []string{"dev", "main"}}},
	}

	if diff := cmp.Diff(want, GeneratePortfolioObservation(&details)); diff != "" {
		t.Errorf("GeneratePortfolioObservation() -want +got:\n%s", diff)
	}
}

// TestShowDetails tests the ShowDetails request against a fake SonarQube
// server.
func TestShowDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/views/show" || r.URL.Query().Get("key") != "pf" {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"pf","subViews":[{"key":"pf:app","originalKey":"app","qualifier":"APP","selectedBranches":["main"]}]}`))
	}))
	t.Cleanup(server.Close)

	config := newTestConfig()
	config.BaseURL = server.URL + "/api/"
	client := NewPortfoliosClient(config)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		details, resp, err := client.ShowDetails(context.Background(), &sonar.ViewsShowOptions{Key: "pf"})
		if resp != nil {
			defer resp.Body.Close()
		}

		if err != nil {
			t.Fatalf("ShowDetails() unexpected error: %v", err)
		}

		want := []PortfolioSubView{{View: sonar.View{Key: "pf:app", OriginalKey: "app", Qualifier: "APP"}, SelectedBranches: []string{"main"}}}
		if diff := cmp.Diff(want, details.SubViews); diff != "" {
			t.Errorf("ShowDetails() SubViews -want +got:\n%s", diff)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		_, resp, err := client.ShowDetails(context.Background(), &sonar.ViewsShowOptions{Key: "missing"})
		if resp != nil {
			defer resp.Body.Close()
		}

		if err == nil {
			t.Fatal("ShowDetails() expected error, got nil")
		}

		if resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("ShowDetails() response = %v, want a 404 response", resp)
		}
	})

	t.Run("InvalidOptions", func(t *testing.T) {
		t.Parallel()

		_, resp, err := client.ShowDetails(context.Background(), &sonar.ViewsShowOptions{})
		if resp != nil {
			defer resp.Body.Close()
		}

		if err == nil {
			t.Fatal("ShowDetails() expected a validation error, got nil")
		}
	})
}

// TestGeneratePortfolioCreateOptions tests the
// GeneratePortfolioCreateOptions function.
func TestGeneratePortfolioCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.PortfolioParameters
		want *sonar.ViewsCreateOptions
	}{
		"RootPortfolio": {
			spec: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", Description: "d", Visibility: "private"},
			want: &sonar.ViewsCreateOptions{Key: "k", Name: "n", Description: "d", Visibility: "private"},
		},
		"SubPortfolioOmitsVisibility": {
			spec: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", Visibility: "public", Parent: new("parent")},
			want: &sonar.ViewsCreateOptions{Key: "k", Name: "n", Parent: "parent"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, GeneratePortfolioCreateOptions(tc.spec)); diff != "" {
				t.Errorf("GeneratePortfolioCreateOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestIsPortfolioSubPortfolio tests the IsPortfolioSubPortfolio function.
func TestIsPortfolioSubPortfolio(t *testing.T) {
	t.Parallel()

	parent := &PortfolioDetails{
		SubViews: []PortfolioSubView{
			{View: sonar.View{Key: "child", Qualifier: "SVW"}},
			{View: sonar.View{Key: "parent:referenced", OriginalKey: "referenced", Qualifier: "VW"}},
		},
	}

	cases := map[string]struct {
		parent *PortfolioDetails
		key    string
		want   bool
	}{
		"NilParent":           {parent: nil, key: "child", want: false},
		"SubPortfolio":        {parent: parent, key: "child", want: true},
		"ReferencedPortfolio": {parent: parent, key: "referenced", want: false},
		"Unrelated":           {parent: parent, key: "other", want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsPortfolioSubPortfolio(tc.parent, tc.key); got != tc.want {
				t.Errorf("IsPortfolioSubPortfolio() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDesiredPortfolioMembers tests the DesiredPortfolioProjects and
// DesiredPortfolioApplications functions.
func TestDesiredPortfolioMembers(t *testing.T) {
	t.Parallel()

	want := map[string][]string{"a": {"main", "dev"}, "b": nil}

	projects := DesiredPortfolioProjects([]v1alpha1.PortfolioProjectParameters{
		{Key: new("a"), Branches: []string{"main"}},
		{Key: new("b")},
		{Key: new("a"), Branches: []string{"dev"}},
		{KeyRef: &xpv1.Reference{Name: "unresolved"}},
	})
	if diff := cmp.Diff(want, projects); diff != "" {
		t.Errorf("DesiredPortfolioProjects() -want +got:\n%s", diff)
	}

	applications := DesiredPortfolioApplications([]v1alpha1.PortfolioApplicationParameters{
		{Key: new("a"), Branches: []string{"main"}},
		{Key: new("b")},
		{Key: new("a"), Branches: []string{"dev"}},
		{KeyRef: &xpv1.Reference{Name: "unresolved"}},
	})
	if diff := cmp.Diff(want, applications); diff != "" {
		t.Errorf("DesiredPortfolioApplications() -want +got:\n%s", diff)
	}
}

// TestDiffPortfolioMembers tests the DiffPortfolioMembers function.
func TestDiffPortfolioMembers(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		desired     map[string][]string
		observation []v1alpha1.PortfolioMemberObservation
		want        PortfolioMembersDiff
	}{
		"UpToDate": {
			desired:     map[string][]string{"a": nil, "b": {"dev", "main"}},
			observation: []v1alpha1.PortfolioMemberObservation{{Key: "a", Branches: []string{"x"}}, {Key: "b", Branches: []string{"main", "dev"}}},
			want:        PortfolioMembersDiff{},
		},
		"AddMembersWithBranches": {
			desired: map[string][]string{"b": {"dev"}, "a": nil},
			want: PortfolioMembersDiff{
				Add:         []string{"a", "b"},
				AddBranches: []PortfolioBranch{{Key: "b", Branch: "dev"}},
			},
		},
		"ReconcileBranches": {
			desired:     map[string][]string{"a": {"dev", "feature"}},
			observation: []v1alpha1.PortfolioMemberObservation{{Key: "a", Branches: []string{"dev", "main"}}},
			want: PortfolioMembersDiff{
				AddBranches:    []PortfolioBranch{{Key: "a", Branch: "feature"}},
				RemoveBranches: []PortfolioBranch{{Key: "a", Branch: "main"}},
			},
		},
		"RemoveMembers": {
			desired:     map[string][]string{},
			observation: []v1alpha1.PortfolioMemberObservation{{Key: "b"}, {Key: "a"}},
			want:        PortfolioMembersDiff{Remove: []string{"a", "b"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := DiffPortfolioMembers(tc.desired, tc.observation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("DiffPortfolioMembers() -want +got:\n%s", diff)
			}

			if got.IsEmpty() != cmp.Equal(tc.want, PortfolioMembersDiff{}) {
				t.Errorf("PortfolioMembersDiff.IsEmpty() = %v", got.IsEmpty())
			}
		})
	}
}

// TestPortfolioMembersDiffs tests the PortfolioProjectsDiff,
// PortfolioApplicationsDiff and referenced portfolio helpers.
func TestPortfolioMembersDiffs(t *testing.T) {
	t.Parallel()

	observation := &v1alpha1.PortfolioObservation{
		Projects:      []v1alpha1.PortfolioMemberObservation{{Key: "old-project"}},
		Applications:  []v1alpha1.PortfolioMemberObservation{{Key: "old-app"}},
		SubPortfolios: []string{"old-ref", "kept-ref"},
	}

	t.Run("Unmanaged", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.PortfolioParameters{SelectionMode: SelectionModeManual}

		if diff := PortfolioProjectsDiff(spec, observation); !diff.IsEmpty() {
			t.Errorf("PortfolioProjectsDiff() = %+v, want empty", diff)
		}

		if diff := PortfolioApplicationsDiff(spec, observation); !diff.IsEmpty() {
			t.Errorf("PortfolioApplicationsDiff() = %+v, want empty", diff)
		}

		if got := PortfolioSubPortfoliosToAdd(spec, observation); got != nil {
			t.Errorf("PortfolioSubPortfoliosToAdd() = %v, want nil", got)
		}

		if got := PortfolioSubPortfoliosToRemove(spec, observation); got != nil {
			t.Errorf("PortfolioSubPortfoliosToRemove() = %v, want nil", got)
		}
	})

	t.Run("ProjectsIgnoredOutsideManualMode", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.PortfolioParameters{SelectionMode: "TAGS", Projects: &[]v1alpha1.PortfolioProjectParameters{}}

		if diff := PortfolioProjectsDiff(spec, observation); !diff.IsEmpty() {
			t.Errorf("PortfolioProjectsDiff() = %+v, want empty", diff)
		}
	})

	t.Run("Managed", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.PortfolioParameters{
			SelectionMode: SelectionModeManual,
			Projects:      &[]v1alpha1.PortfolioProjectParameters{{Key: new("new-project")}},
			Applications:  &[]v1alpha1.PortfolioApplicationParameters{{Key: new("new-app")}},
			SubPortfolios: &[]string{"kept-ref", "new-ref"},
		}

		wantProjects := PortfolioMembersDiff{Add: []string{"new-project"}, Remove: []string{"old-project"}}
		if diff := cmp.Diff(wantProjects, PortfolioProjectsDiff(spec, observation)); diff != "" {
			t.Errorf("PortfolioProjectsDiff() -want +got:\n%s", diff)
		}

		wantApplications := PortfolioMembersDiff{Add: []string{"new-app"}, Remove: []string{"old-app"}}
		if diff := cmp.Diff(wantApplications, PortfolioApplicationsDiff(spec, observation)); diff != "" {
			t.Errorf("PortfolioApplicationsDiff() -want +got:\n%s", diff)
		}

		if diff := cmp.Diff([]string{"new-ref"}, PortfolioSubPortfoliosToAdd(spec, observation)); diff != "" {
			t.Errorf("PortfolioSubPortfoliosToAdd() -want +got:\n%s", diff)
		}

		if diff := cmp.Diff([]string{"old-ref"}, PortfolioSubPortfoliosToRemove(spec, observation)); diff != "" {
			t.Errorf("PortfolioSubPortfoliosToRemove() -want +got:\n%s", diff)
		}
	})
}

// TestLateInitializePortfolio tests the LateInitializePortfolio function.
func TestLateInitializePortfolio(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PortfolioParameters
		observation *v1alpha1.PortfolioObservation
	}{
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.PortfolioObservation{Name: "obs"},
		},
		"NilObservation": {
			spec:        &v1alpha1.PortfolioParameters{Name: "spec"},
			observation: nil,
		},
		"BothNil": {
			spec:        nil,
			observation: nil,
		},
		"NoOpWithValues": {
			spec:        &v1alpha1.PortfolioParameters{Key: "k", Name: "n"},
			observation: &v1alpha1.PortfolioObservation{Key: "k", Name: "n"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specCopy := tc.spec
			if tc.spec != nil {
				specVal := *tc.spec
				specCopy = &specVal
			}

			LateInitializePortfolio(tc.spec, tc.observation)

			if tc.spec != nil && !cmp.Equal(tc.spec, specCopy) {
				t.Errorf("LateInitializePortfolio() modified spec unexpectedly: got %+v, want %+v", *tc.spec, *specCopy)
			}
		})
	}
}

// TestIsPortfolioLateInitialized tests the IsPortfolioLateInitialized function.
func TestIsPortfolioLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.PortfolioParameters
		current *v1alpha1.PortfolioParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.PortfolioParameters{Name: "n"},
			want:    false,
		},
		"NilCurrent": {
			former:  &v1alpha1.PortfolioParameters{Name: "n"},
			current: nil,
			want:    false,
		},
		"BothNil": {
			former:  nil,
			current: nil,
			want:    false,
		},
		"EqualParameters": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			want:    false,
		},
		"DifferentName": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "old"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "new"},
			want:    true,
		},
		"DifferentSelectionMode": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "MANUAL"},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPortfolioLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsPortfolioLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsPortfolioUpToDate tests the IsPortfolioUpToDate function.
func TestIsPortfolioUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PortfolioParameters
		observation *v1alpha1.PortfolioObservation
		want        bool
	}{
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.PortfolioObservation{Name: "n"},
			want:        true,
		},
		"NilObservation": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n"},
			observation: nil,
			want:        false,
		},
		"UpToDateAllFieldsMatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				Description:   "desc",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				Description:   "desc",
				SelectionMode: "NONE",
			},
			want: true,
		},
		"SelectionModeNormalizedEmptyEqualsNONE": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "NONE",
			},
			want: true,
		},
		"SelectionModeNormalizedBothEmpty": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			want: true,
		},
		"NameMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "New Name",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "Old Name",
				SelectionMode: "NONE",
			},
			want: false,
		},
		"DescriptionMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				Description:   "new desc",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				Description:   "old desc",
				SelectionMode: "NONE",
			},
			want: false,
		},
		"SelectionModeMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "MANUAL",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "NONE",
			},
			want: false,
		},
		"ParentUpToDate": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n", Parent: new("parent")},
			observation: &v1alpha1.PortfolioObservation{Name: "n", Parent: "parent"},
			want:        true,
		},
		"ParentMismatch": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n", Parent: new("parent")},
			observation: &v1alpha1.PortfolioObservation{Name: "n"},
			want:        false,
		},
		"MembersUpToDate": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "n",
				SelectionMode: "MANUAL",
				Projects:      &[]v1alpha1.PortfolioProjectParameters{{Key: new("p"), Branches: []string{"dev"}}},
				Applications:  &[]v1alpha1.PortfolioApplicationParameters{{Key: new("a")}},
				SubPortfolios: &[]string{"r"},
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "n",
				SelectionMode: "MANUAL",
				Projects:      []v1alpha1.PortfolioMemberObservation{{Key: "p", Branches: []string{"dev"}}},
				Applications:  []v1alpha1.PortfolioMemberObservation{{Key: "a", Branches: []string{"main"}}},
				SubPortfolios: []string{"r"},
			},
			want: true,
		},
		"ProjectBranchMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "n",
				SelectionMode: "MANUAL",
				Projects:      &[]v1alpha1.PortfolioProjectParameters{{Key: new("p"), Branches: []string{"dev"}}},
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "n",
				SelectionMode: "MANUAL",
				Projects:      []v1alpha1.PortfolioMemberObservation{{Key: "p"}},
			},
			want: false,
		},
		"ApplicationMismatch": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n", Applications: &[]v1alpha1.PortfolioApplicationParameters{}},
			observation: &v1alpha1.PortfolioObservation{Name: "n", Applications: []v1alpha1.PortfolioMemberObservation{{Key: "a"}}},
			want:        false,
		},
		"SubPortfolioMissing": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n", SubPortfolios: &[]string{"r"}},
			observation: &v1alpha1.PortfolioObservation{Name: "n"},
			want:        false,
		},
		"SubPortfolioExtra": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n", SubPortfolios: &[]string{}},
			observation: &v1alpha1.PortfolioObservation{Name: "n", SubPortfolios: []string{"r"}},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPortfolioUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsPortfolioUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNormalizeSelectionMode tests the normalizeSelectionMode function.
func TestNormalizeSelectionMode(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mode string
		want string
	}{
		"EmptyBecomesNONE":   {mode: "", want: "NONE"},
		"NONEUnchanged":      {mode: "NONE", want: "NONE"},
		"MANUALUnchanged":    {mode: "MANUAL", want: "MANUAL"},
		"REGEXPUnchanged":    {mode: "REGEXP", want: "REGEXP"},
		"REMAININGUnchanged": {mode: "REMAINING", want: "REMAINING"},
		"TAGSUnchanged":      {mode: "TAGS", want: "TAGS"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := normalizeSelectionMode(tc.mode)
			if got != tc.want {
				t.Errorf("normalizeSelectionMode(%q) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}
