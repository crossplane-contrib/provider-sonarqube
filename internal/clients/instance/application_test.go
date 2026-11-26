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

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// newApplicationBranch builds an ApplicationBranchParameters from
// alternating project key / project branch pairs.
func newApplicationBranch(name string, projectBranchPairs ...string) v1alpha1.ApplicationBranchParameters {
	branch := v1alpha1.ApplicationBranchParameters{Name: name}
	for i := 0; i+1 < len(projectBranchPairs); i += 2 {
		branch.Projects = append(branch.Projects, v1alpha1.ApplicationBranchProjectParameters{
			Project: projectBranchPairs[i],
			Branch:  projectBranchPairs[i+1],
		})
	}

	return branch
}

// newApplicationBranchObservation builds an ApplicationBranchObservation
// from alternating project key / project branch pairs.
func newApplicationBranchObservation(name string, isMain bool, projectBranchPairs ...string) v1alpha1.ApplicationBranchObservation {
	branch := v1alpha1.ApplicationBranchObservation{Name: name, IsMain: isMain}
	for i := 0; i+1 < len(projectBranchPairs); i += 2 {
		branch.Projects = append(branch.Projects, v1alpha1.ApplicationBranchProjectObservation{
			Project: projectBranchPairs[i],
			Branch:  projectBranchPairs[i+1],
		})
	}

	return branch
}

// TestGenerateApplicationCreateOptions tests the
// GenerateApplicationCreateOptions function.
func TestGenerateApplicationCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.ApplicationParameters
		want *sonar.ApplicationsCreateOptions
	}{
		"WithoutDescription": {
			spec: &v1alpha1.ApplicationParameters{Key: "app", Name: "App", Visibility: "private"},
			want: &sonar.ApplicationsCreateOptions{Key: "app", Name: "App", Visibility: "private"},
		},
		"WithDescription": {
			spec: &v1alpha1.ApplicationParameters{Key: "app", Name: "App", Description: new("desc"), Visibility: "public"},
			want: &sonar.ApplicationsCreateOptions{Key: "app", Name: "App", Description: "desc", Visibility: "public"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateApplicationCreateOptions(tc.spec)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateApplicationCreateOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateApplicationUpdateOptions tests the
// GenerateApplicationUpdateOptions function.
func TestGenerateApplicationUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.ApplicationParameters
		observation *v1alpha1.ApplicationObservation
		want        *sonar.ApplicationsUpdateOptions
	}{
		"UnmanagedDescriptionKeepsObserved": {
			spec:        &v1alpha1.ApplicationParameters{Name: "New"},
			observation: &v1alpha1.ApplicationObservation{Name: "Old", Description: "observed"},
			want:        &sonar.ApplicationsUpdateOptions{Application: "app", Name: "New", Description: "observed"},
		},
		"ManagedDescription": {
			spec:        &v1alpha1.ApplicationParameters{Name: "New", Description: new("desired")},
			observation: &v1alpha1.ApplicationObservation{Name: "Old", Description: "observed"},
			want:        &sonar.ApplicationsUpdateOptions{Application: "app", Name: "New", Description: "desired"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateApplicationUpdateOptions("app", tc.spec, tc.observation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateApplicationUpdateOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateApplicationSetTagsOptions tests the
// GenerateApplicationSetTagsOptions function.
func TestGenerateApplicationSetTagsOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		tags []string
		want *sonar.ApplicationsSetTagsOptions
	}{
		"Tags": {
			tags: []string{"go", "team-a"},
			want: &sonar.ApplicationsSetTagsOptions{Application: "app", Tags: []string{"go", "team-a"}},
		},
		"EmptyTagsSendAnEmptyParameter": {
			tags: []string{},
			want: &sonar.ApplicationsSetTagsOptions{Application: "app", Tags: []string{""}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateApplicationSetTagsOptions("app", tc.tags)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateApplicationSetTagsOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateApplicationBranchOptions tests the branch create and update
// option generators, including deterministic project ordering.
func TestGenerateApplicationBranchOptions(t *testing.T) {
	t.Parallel()

	branch := newApplicationBranch("release", "proj-b", "release-b", "proj-a", "release-a")

	gotCreate := GenerateApplicationCreateBranchOptions("app", &branch)
	wantCreate := &sonar.ApplicationsCreateBranchOptions{
		Application:   "app",
		Branch:        "release",
		Project:       []string{"proj-a", "proj-b"},
		ProjectBranch: []string{"release-a", "release-b"},
	}

	if diff := cmp.Diff(wantCreate, gotCreate); diff != "" {
		t.Errorf("GenerateApplicationCreateBranchOptions() -want +got:\n%s", diff)
	}

	gotUpdate := GenerateApplicationUpdateBranchOptions("app", &branch)
	wantUpdate := &sonar.ApplicationsUpdateBranchOptions{
		Application:   "app",
		Branch:        "release",
		Name:          "release",
		Project:       []string{"proj-a", "proj-b"},
		ProjectBranch: []string{"release-a", "release-b"},
	}

	if diff := cmp.Diff(wantUpdate, gotUpdate); diff != "" {
		t.Errorf("GenerateApplicationUpdateBranchOptions() -want +got:\n%s", diff)
	}

	if branch.Projects[0].Project != "proj-b" {
		t.Errorf("GenerateApplicationCreateBranchOptions() mutated the spec project order")
	}
}

// TestGenerateApplicationObservation tests the
// GenerateApplicationObservation function.
func TestGenerateApplicationObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		details        *sonar.ApplicationDetails
		branchProjects map[string][]sonar.ApplicationProject
		want           v1alpha1.ApplicationObservation
	}{
		"NilDetails": {
			details: nil,
			want:    v1alpha1.ApplicationObservation{},
		},
		"EmptyDetails": {
			details: &sonar.ApplicationDetails{},
			want:    v1alpha1.ApplicationObservation{Projects: []string{}},
		},
		"FullDetails": {
			details: &sonar.ApplicationDetails{
				Key:         "app",
				Name:        "App",
				Description: "desc",
				Visibility:  "private",
				Tags:        []string{"team-a"},
				Projects: []sonar.ApplicationProject{
					{Key: "proj-b", Branch: "main", IsMain: true},
					{Key: "proj-a", Branch: "master", IsMain: true},
				},
				Branches: []sonar.ApplicationBranch{
					{Name: "main", IsMain: true},
					{Name: "release"},
					{Name: "empty"},
				},
			},
			branchProjects: map[string][]sonar.ApplicationProject{
				"release": {
					{Key: "proj-a", Branch: "release-a"},
					{Key: "proj-b", Branch: "main", IsMain: true},
				},
			},
			want: v1alpha1.ApplicationObservation{
				Key:         "app",
				Name:        "App",
				Description: "desc",
				Visibility:  "private",
				Tags:        []string{"team-a"},
				Projects:    []string{"proj-a", "proj-b"},
				Branches: []v1alpha1.ApplicationBranchObservation{
					{
						Name:   "main",
						IsMain: true,
						Projects: []v1alpha1.ApplicationBranchProjectObservation{
							{Project: "proj-b", Branch: "main", IsMain: true},
							{Project: "proj-a", Branch: "master", IsMain: true},
						},
					},
					{
						Name: "release",
						Projects: []v1alpha1.ApplicationBranchProjectObservation{
							{Project: "proj-a", Branch: "release-a"},
							{Project: "proj-b", Branch: "main", IsMain: true},
						},
					},
					{Name: "empty"},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateApplicationObservation(tc.details, tc.branchProjects)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateApplicationObservation() -want +got:\n%s", diff)
			}
		})
	}
}

// TestIsApplicationUpToDate tests the IsApplicationUpToDate function.
func TestIsApplicationUpToDate(t *testing.T) {
	t.Parallel()

	observation := &v1alpha1.ApplicationObservation{
		Name:        "App",
		Description: "desc",
		Projects:    []string{"proj-a", "proj-b"},
		Branches: []v1alpha1.ApplicationBranchObservation{
			newApplicationBranchObservation("main", true, "proj-a", "main", "proj-b", "main"),
			newApplicationBranchObservation("release", false, "proj-a", "release-a", "proj-b", "main"),
		},
	}

	upToDateSpec := func() *v1alpha1.ApplicationParameters {
		return &v1alpha1.ApplicationParameters{
			Name:     "App",
			Projects: []string{"proj-b", "proj-a"},
			Branches: []v1alpha1.ApplicationBranchParameters{
				newApplicationBranch("release", "proj-a", "release-a", "proj-b", "main"),
			},
		}
	}

	cases := map[string]struct {
		spec        *v1alpha1.ApplicationParameters
		observation *v1alpha1.ApplicationObservation
		want        bool
	}{
		"NilSpec": {
			spec: nil,
			want: true,
		},
		"NilObservation": {
			spec: upToDateSpec(),
			want: false,
		},
		"UpToDate": {
			spec:        upToDateSpec(),
			observation: observation,
			want:        true,
		},
		"ManagedDescriptionMatches": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Description = new("desc")

				return s
			}(),
			observation: observation,
			want:        true,
		},
		"NameDiffers": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Name = "Other"

				return s
			}(),
			observation: observation,
			want:        false,
		},
		"DescriptionDiffers": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Description = new("other")

				return s
			}(),
			observation: observation,
			want:        false,
		},
		"UnmanagedTags": {
			spec: upToDateSpec(),
			observation: func() *v1alpha1.ApplicationObservation {
				o := observation.DeepCopy()
				o.Tags = []string{"go"}

				return o
			}(),
			want: true,
		},
		"ReorderedTags": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Tags = &[]string{"team-a", "go", "go"}

				return s
			}(),
			observation: func() *v1alpha1.ApplicationObservation {
				o := observation.DeepCopy()
				o.Tags = []string{"go", "team-a"}

				return o
			}(),
			want: true,
		},
		"EmptyTagsMatchNoTags": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Tags = &[]string{}

				return s
			}(),
			observation: observation,
			want:        true,
		},
		"EmptyTagsDiffer": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Tags = &[]string{}

				return s
			}(),
			observation: func() *v1alpha1.ApplicationObservation {
				o := observation.DeepCopy()
				o.Tags = []string{"go"}

				return o
			}(),
			want: false,
		},
		"TagsDiffer": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Tags = &[]string{"go"}

				return s
			}(),
			observation: observation,
			want:        false,
		},
		"MissingProject": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Projects = append(s.Projects, "proj-c")

				return s
			}(),
			observation: observation,
			want:        false,
		},
		"ExtraProject": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Projects = []string{"proj-a"}

				return s
			}(),
			observation: observation,
			want:        false,
		},
		"BranchDiffers": {
			spec: func() *v1alpha1.ApplicationParameters {
				s := upToDateSpec()
				s.Branches = []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release", "proj-a", "other", "proj-b", "main")}

				return s
			}(),
			observation: observation,
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsApplicationUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsApplicationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAreApplicationBranchesUpToDate tests the
// AreApplicationBranchesUpToDate function.
func TestAreApplicationBranchesUpToDate(t *testing.T) {
	t.Parallel()

	mainBranch := newApplicationBranchObservation("main", true, "proj-a", "main")
	release := newApplicationBranchObservation("release", false, "proj-a", "release-a")

	cases := map[string]struct {
		spec        []v1alpha1.ApplicationBranchParameters
		observation []v1alpha1.ApplicationBranchObservation
		want        bool
	}{
		"NoBranches": {
			want: true,
		},
		"OnlyMainObserved": {
			observation: []v1alpha1.ApplicationBranchObservation{mainBranch},
			want:        true,
		},
		"MatchingBranch": {
			spec:        []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release", "proj-a", "release-a")},
			observation: []v1alpha1.ApplicationBranchObservation{mainBranch, release},
			want:        true,
		},
		"MissingBranch": {
			spec:        []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release", "proj-a", "release-a")},
			observation: []v1alpha1.ApplicationBranchObservation{mainBranch},
			want:        false,
		},
		"UnwantedBranch": {
			observation: []v1alpha1.ApplicationBranchObservation{mainBranch, release},
			want:        false,
		},
		"DriftedBranch": {
			spec:        []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release", "proj-a", "release-b")},
			observation: []v1alpha1.ApplicationBranchObservation{mainBranch, release},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreApplicationBranchesUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("AreApplicationBranchesUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsApplicationBranchUpToDate tests the IsApplicationBranchUpToDate
// function.
func TestIsApplicationBranchUpToDate(t *testing.T) {
	t.Parallel()

	observed := newApplicationBranchObservation("release", false, "proj-a", "release-a", "proj-b", "main")

	cases := map[string]struct {
		spec        *v1alpha1.ApplicationBranchParameters
		observation *v1alpha1.ApplicationBranchObservation
		want        bool
	}{
		"NilSpec": {
			want: true,
		},
		"NilObservation": {
			spec: new(newApplicationBranch("release", "proj-a", "release-a")),
			want: false,
		},
		"ExtraObservedProject": {
			spec:        new(newApplicationBranch("release", "proj-a", "release-a")),
			observation: &observed,
			want:        false,
		},
		"AllMatch": {
			spec:        new(newApplicationBranch("release", "proj-b", "main", "proj-a", "release-a")),
			observation: &observed,
			want:        true,
		},
		"BranchDiffers": {
			spec:        new(newApplicationBranch("release", "proj-a", "release-b")),
			observation: &observed,
			want:        false,
		},
		"ProjectMissing": {
			spec:        new(newApplicationBranch("release", "proj-c", "main")),
			observation: &observed,
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsApplicationBranchUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsApplicationBranchUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestFindApplicationBranchObservation tests the
// FindApplicationBranchObservation function.
func TestFindApplicationBranchObservation(t *testing.T) {
	t.Parallel()

	observation := []v1alpha1.ApplicationBranchObservation{
		newApplicationBranchObservation("main", true),
		newApplicationBranchObservation("release", false),
	}

	got, found := FindApplicationBranchObservation(observation, "release")
	if !found || got == nil || got.Name != "release" {
		t.Errorf("FindApplicationBranchObservation(release) = %v, %v, want release, true", got, found)
	}

	got, found = FindApplicationBranchObservation(observation, "missing")
	if found || got != nil {
		t.Errorf("FindApplicationBranchObservation(missing) = %v, %v, want nil, false", got, found)
	}
}

// TestApplicationBranchesToDelete tests the ApplicationBranchesToDelete
// function.
func TestApplicationBranchesToDelete(t *testing.T) {
	t.Parallel()

	observation := []v1alpha1.ApplicationBranchObservation{
		newApplicationBranchObservation("main", true),
		newApplicationBranchObservation("release", false),
		newApplicationBranchObservation("old", false),
	}

	cases := map[string]struct {
		spec []v1alpha1.ApplicationBranchParameters
		want []string
	}{
		"NoSpecDeletesAllNonMain": {
			want: []string{"release", "old"},
		},
		"KeepsDesired": {
			spec: []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release")},
			want: []string{"old"},
		},
		"NeverDeletesMain": {
			spec: []v1alpha1.ApplicationBranchParameters{newApplicationBranch("release"), newApplicationBranch("old")},
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ApplicationBranchesToDelete(tc.spec, observation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ApplicationBranchesToDelete() -want +got:\n%s", diff)
			}
		})
	}
}

// TestApplicationProjectsDiff tests the ApplicationProjectsToAdd and
// ApplicationProjectsToRemove functions.
func TestApplicationProjectsDiff(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec       []string
		observed   []string
		wantAdd    []string
		wantRemove []string
	}{
		"Empty": {},
		"InSync": {
			spec:     []string{"b", "a"},
			observed: []string{"a", "b"},
		},
		"AddAndRemove": {
			spec:       []string{"c", "a", "d", "c"},
			observed:   []string{"a", "b"},
			wantAdd:    []string{"c", "d"},
			wantRemove: []string{"b"},
		},
		"RemoveAll": {
			observed:   []string{"b", "a"},
			wantRemove: []string{"a", "b"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.wantAdd, ApplicationProjectsToAdd(tc.spec, tc.observed)); diff != "" {
				t.Errorf("ApplicationProjectsToAdd() -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantRemove, ApplicationProjectsToRemove(tc.spec, tc.observed)); diff != "" {
				t.Errorf("ApplicationProjectsToRemove() -want +got:\n%s", diff)
			}
		})
	}
}
