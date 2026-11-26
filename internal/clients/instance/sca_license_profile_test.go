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
	"errors"
	"net/http"
	"strings"
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

// errSCABoom is a generic test error.
var errSCABoom = errors.New("boom")

// scaAssignableProjectsClient is a SCALicenseProfilesClient serving the
// assignable projects of a license profile from pages.
type scaAssignableProjectsClient struct {
	SCALicenseProfilesClient

	pages []*sonar.ScaLicenseProfileAssignableProjectsResponse
	err   error
	calls []*sonar.ScaLicenseProfileAssignableProjectsOptions
}

// ListLicenseProfileAssignableProjects returns the page requested by opt.
func (c *scaAssignableProjectsClient) ListLicenseProfileAssignableProjects(_ context.Context, opt *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error) {
	c.calls = append(c.calls, opt)
	if c.err != nil {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError}, c.err
	}

	index := int(opt.PageIndex) - 1
	if index >= len(c.pages) {
		return nil, &http.Response{StatusCode: http.StatusOK}, nil
	}

	return c.pages[index], &http.Response{StatusCode: http.StatusOK}, nil
}

// scaAssignedProjectsPage builds a page of assigned projects.
func scaAssignedProjectsPage(pageIndex, total int32, keys ...string) *sonar.ScaLicenseProfileAssignableProjectsResponse {
	page := &sonar.ScaLicenseProfileAssignableProjectsResponse{
		Page: sonar.PageResponseV2{PageIndex: pageIndex, PageSize: scaLicenseProfileProjectsPageSize, Total: total},
	}
	for _, key := range keys {
		page.AssignableProjects = append(page.AssignableProjects, sonar.ScaLicenseProfileAssignableProject{
			ProjectKey:               key,
			AssignedToLicenseProfile: true,
		})
	}

	return page
}

// TestGenerateSCALicenseProfileCreateOptions tests the
// GenerateSCALicenseProfileCreateOptions function.
func TestGenerateSCALicenseProfileCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.SCALicenseProfileParameters
		want *sonar.ScaLicenseProfileCreateRequest
	}{
		"UnmanagedDefault": {
			spec: &v1alpha1.SCALicenseProfileParameters{Name: "Profile", Organization: "default-organization"},
			want: &sonar.ScaLicenseProfileCreateRequest{Name: "Profile", Organization: "default-organization"},
		},
		"Default": {
			spec: &v1alpha1.SCALicenseProfileParameters{Name: "Profile", Organization: "org", Default: new(true)},
			want: &sonar.ScaLicenseProfileCreateRequest{Name: "Profile", Organization: "org", Default: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSCALicenseProfileCreateOptions(tc.spec)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSCALicenseProfileCreateOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateSCALicenseProfileUpdateOptions tests the
// GenerateSCALicenseProfileUpdateOptions function.
func TestGenerateSCALicenseProfileUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.SCALicenseProfileParameters
		want *sonar.ScaLicenseProfileUpdateRequest
	}{
		"UnmanagedDefault": {
			spec: &v1alpha1.SCALicenseProfileParameters{Name: "Profile"},
			want: &sonar.ScaLicenseProfileUpdateRequest{Name: "Profile"},
		},
		"NotDefault": {
			spec: &v1alpha1.SCALicenseProfileParameters{Name: "Profile", Default: new(false)},
			want: &sonar.ScaLicenseProfileUpdateRequest{Name: "Profile", Default: new(false)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSCALicenseProfileUpdateOptions(tc.spec)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSCALicenseProfileUpdateOptions() -want +got:\n%s", diff)
			}
		})
	}
}

// TestFindSCALicenseProfile tests the FindSCALicenseProfile function.
func TestFindSCALicenseProfile(t *testing.T) {
	t.Parallel()

	profiles := []sonar.ScaLicenseProfile{{Id: "a", Name: "A"}, {Id: "b", Name: "B"}}

	got, found := FindSCALicenseProfile(profiles, "b")
	if !found || got.Name != "B" {
		t.Errorf("FindSCALicenseProfile(b) = %v, %v; want B, true", got, found)
	}

	got, found = FindSCALicenseProfile(profiles, "my-profile")
	if found || got != nil {
		t.Errorf("FindSCALicenseProfile(my-profile) = %v, %v; want nil, false", got, found)
	}
}

// TestFetchSCALicenseProfileProjects tests the
// FetchSCALicenseProfileProjects function.
func TestFetchSCALicenseProfileProjects(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client    *scaAssignableProjectsClient
		want      []string
		wantCalls int
		errSubstr string
	}{
		"SinglePage": {
			client: &scaAssignableProjectsClient{pages: []*sonar.ScaLicenseProfileAssignableProjectsResponse{
				scaAssignedProjectsPage(1, 2, "proj-b", "proj-a"),
			}},
			want:      []string{"proj-a", "proj-b"},
			wantCalls: 1,
		},
		"SeveralPages": {
			client: &scaAssignableProjectsClient{pages: []*sonar.ScaLicenseProfileAssignableProjectsResponse{
				scaAssignedProjectsPage(1, scaLicenseProfileProjectsPageSize+1, "proj-b"),
				scaAssignedProjectsPage(2, scaLicenseProfileProjectsPageSize+1, "proj-a"),
			}},
			want:      []string{"proj-a", "proj-b"},
			wantCalls: 2,
		},
		"SkipsUnassignedProjects": {
			client: &scaAssignableProjectsClient{pages: []*sonar.ScaLicenseProfileAssignableProjectsResponse{
				{
					Page: sonar.PageResponseV2{PageIndex: 1, PageSize: scaLicenseProfileProjectsPageSize, Total: 2},
					AssignableProjects: []sonar.ScaLicenseProfileAssignableProject{
						{ProjectKey: "proj-a", AssignedToLicenseProfile: true},
						{ProjectKey: "proj-b"},
					},
				},
			}},
			want:      []string{"proj-a"},
			wantCalls: 1,
		},
		"NoProjects": {
			client: &scaAssignableProjectsClient{pages: []*sonar.ScaLicenseProfileAssignableProjectsResponse{
				scaAssignedProjectsPage(1, 0),
			}},
			want:      []string{},
			wantCalls: 1,
		},
		"APIError": {
			client:    &scaAssignableProjectsClient{err: errSCABoom},
			errSubstr: "boom",
			wantCalls: 1,
		},
		"NilResult": {
			client:    &scaAssignableProjectsClient{},
			errSubstr: "nil assignable projects result",
			wantCalls: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := FetchSCALicenseProfileProjects(context.Background(), tc.client, "profile-id")
			if tc.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("FetchSCALicenseProfileProjects() error = %v, want to contain %q", err, tc.errSubstr)
				}
			} else if err != nil {
				t.Fatalf("FetchSCALicenseProfileProjects() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FetchSCALicenseProfileProjects() -want +got:\n%s", diff)
			}

			if len(tc.client.calls) != tc.wantCalls {
				t.Fatalf("FetchSCALicenseProfileProjects() made %d calls, want %d", len(tc.client.calls), tc.wantCalls)
			}

			first := tc.client.calls[0]
			if first.LicenseProfileId != "profile-id" || first.AssignedToLicenseProfile == nil || !*first.AssignedToLicenseProfile {
				t.Errorf("FetchSCALicenseProfileProjects() options = %+v, want the assigned projects of profile-id", first)
			}
		})
	}
}

// TestGenerateSCALicenseProfileObservation tests the
// GenerateSCALicenseProfileObservation function.
func TestGenerateSCALicenseProfileObservation(t *testing.T) {
	t.Parallel()

	details := &sonar.ScaLicenseProfileDetails{
		Profile: sonar.ScaLicenseProfile{Id: "id", Name: "Profile", Default: true, UpdatedAt: "2026-01-01T00:00:00Z"},
		Categories: []sonar.ScaLicenseProfileCategory{
			{Id: "c1", Key: "COPYLEFT_STRONG", Policy: "DENY"},
			{Id: "c2", Key: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		},
		Licenses: []sonar.ScaLicensePolicyLicense{
			{Id: "l1", SpdxLicenseId: "MIT", Name: "MIT License", Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
			{Id: "l2", SpdxLicenseId: "GPL-3.0-only", Name: "GPL 3.0", Category: "COPYLEFT_STRONG", Policy: "DENY"},
		},
	}

	cases := map[string]struct {
		details  *sonar.ScaLicenseProfileDetails
		licenses []v1alpha1.SCALicenseProfileLicensePolicy
		projects []string
		want     v1alpha1.SCALicenseProfileObservation
	}{
		"Nil": {
			want: v1alpha1.SCALicenseProfileObservation{},
		},
		"OnlyListedLicenses": {
			details:  details,
			licenses: []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "GPL-3.0-only", Policy: "ALLOW"}, {SPDXLicenseID: "Unknown-1.0", Policy: "DENY"}},
			projects: []string{"proj-a"},
			want: v1alpha1.SCALicenseProfileObservation{
				ID:        "id",
				Name:      "Profile",
				Default:   true,
				UpdatedAt: "2026-01-01T00:00:00Z",
				Categories: []v1alpha1.SCALicenseProfileCategoryObservation{
					{Category: "COPYLEFT_STRONG", Policy: "DENY"},
					{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
				},
				Licenses: []v1alpha1.SCALicenseProfileLicenseObservation{
					{ID: "l2", SPDXLicenseID: "GPL-3.0-only", Name: "GPL 3.0", Category: "COPYLEFT_STRONG", Policy: "DENY"},
				},
				Projects: []string{"proj-a"},
			},
		},
		"NoLicensesNoProjects": {
			details:  &sonar.ScaLicenseProfileDetails{Profile: sonar.ScaLicenseProfile{Id: "id", Name: "Profile"}, Licenses: details.Licenses},
			projects: []string{},
			want:     v1alpha1.SCALicenseProfileObservation{ID: "id", Name: "Profile"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSCALicenseProfileObservation(tc.details, tc.licenses, tc.projects)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSCALicenseProfileObservation() -want +got:\n%s", diff)
			}
		})
	}
}

// TestLateInitializeSCALicenseProfile tests the
// LateInitializeSCALicenseProfile and IsSCALicenseProfileLateInitialized
// functions.
func TestLateInitializeSCALicenseProfile(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.SCALicenseProfileParameters
		observation *v1alpha1.SCALicenseProfileObservation
		wantDefault *bool
		wantChanged bool
	}{
		"UnsetDefault": {
			spec:        &v1alpha1.SCALicenseProfileParameters{},
			observation: &v1alpha1.SCALicenseProfileObservation{Default: true},
			wantDefault: new(true),
			wantChanged: true,
		},
		"SetDefaultIsKept": {
			spec:        &v1alpha1.SCALicenseProfileParameters{Default: new(false)},
			observation: &v1alpha1.SCALicenseProfileObservation{Default: true},
			wantDefault: new(false),
		},
		"NilObservation": {
			spec: &v1alpha1.SCALicenseProfileParameters{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			former := tc.spec.DeepCopy()
			LateInitializeSCALicenseProfile(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantDefault, tc.spec.Default); diff != "" {
				t.Errorf("LateInitializeSCALicenseProfile() Default -want +got:\n%s", diff)
			}

			if got := IsSCALicenseProfileLateInitialized(former, tc.spec); got != tc.wantChanged {
				t.Errorf("IsSCALicenseProfileLateInitialized() = %v, want %v", got, tc.wantChanged)
			}
		})
	}

	LateInitializeSCALicenseProfile(nil, nil)

	if IsSCALicenseProfileLateInitialized(nil, &v1alpha1.SCALicenseProfileParameters{}) {
		t.Error("IsSCALicenseProfileLateInitialized(nil, spec) = true, want false")
	}
}

// TestIsSCALicenseProfileUpToDate tests the IsSCALicenseProfileUpToDate
// function.
func TestIsSCALicenseProfileUpToDate(t *testing.T) {
	t.Parallel()

	spec := func(modify func(*v1alpha1.SCALicenseProfileParameters)) *v1alpha1.SCALicenseProfileParameters {
		params := &v1alpha1.SCALicenseProfileParameters{
			Name:       "Profile",
			Default:    new(true),
			Categories: []v1alpha1.SCALicenseProfileCategoryPolicy{{Category: "COPYLEFT_STRONG", Policy: "DENY"}},
			Licenses:   []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "MIT", Policy: "ALLOW"}},
			Projects:   []string{"proj-b", "proj-a"},
		}
		if modify != nil {
			modify(params)
		}

		return params
	}

	observation := &v1alpha1.SCALicenseProfileObservation{
		Name:    "Profile",
		Default: true,
		Categories: []v1alpha1.SCALicenseProfileCategoryObservation{
			{Category: "COPYLEFT_STRONG", Policy: "DENY"},
			{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		},
		Licenses: []v1alpha1.SCALicenseProfileLicenseObservation{{ID: "l1", SPDXLicenseID: "MIT", Policy: "ALLOW"}},
		Projects: []string{"proj-a", "proj-b"},
	}

	cases := map[string]struct {
		spec        *v1alpha1.SCALicenseProfileParameters
		observation *v1alpha1.SCALicenseProfileObservation
		want        bool
	}{
		"NilSpec":        {observation: observation, want: true},
		"NilObservation": {spec: spec(nil)},
		"UpToDate":       {spec: spec(nil), observation: observation, want: true},
		"UnmanagedDefault": {
			spec:        spec(func(p *v1alpha1.SCALicenseProfileParameters) { p.Default = nil }),
			observation: observation,
			want:        true,
		},
		"NameDiffers": {
			spec:        spec(func(p *v1alpha1.SCALicenseProfileParameters) { p.Name = "Other" }),
			observation: observation,
		},
		"DefaultDiffers": {
			spec:        spec(func(p *v1alpha1.SCALicenseProfileParameters) { p.Default = new(false) }),
			observation: observation,
		},
		"CategoryDiffers": {
			spec: spec(func(p *v1alpha1.SCALicenseProfileParameters) {
				p.Categories = []v1alpha1.SCALicenseProfileCategoryPolicy{{Category: "PERMISSIVE_STANDARD", Policy: "DENY"}}
			}),
			observation: observation,
		},
		"LicenseDiffers": {
			spec: spec(func(p *v1alpha1.SCALicenseProfileParameters) {
				p.Licenses = []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "MIT", Policy: "DENY"}}
			}),
			observation: observation,
		},
		"LicenseMissing": {
			spec: spec(func(p *v1alpha1.SCALicenseProfileParameters) {
				p.Licenses = []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "Unknown-1.0", Policy: "DENY"}}
			}),
			observation: observation,
		},
		"ProjectsDiffer": {
			spec:        spec(func(p *v1alpha1.SCALicenseProfileParameters) { p.Projects = []string{"proj-a"} }),
			observation: observation,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsSCALicenseProfileUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Errorf("IsSCALicenseProfileUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSCALicenseProfileCategoriesToUpdate tests the
// SCALicenseProfileCategoriesToUpdate function.
func TestSCALicenseProfileCategoriesToUpdate(t *testing.T) {
	t.Parallel()

	spec := []v1alpha1.SCALicenseProfileCategoryPolicy{
		{Category: "COPYLEFT_STRONG", Policy: "DENY"},
		{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		{Category: "UNKNOWN", Policy: "DENY"},
	}
	observation := []v1alpha1.SCALicenseProfileCategoryObservation{
		{Category: "COPYLEFT_STRONG", Policy: "DENY"},
		{Category: "PERMISSIVE_STANDARD", Policy: "DENY"},
	}

	want := []v1alpha1.SCALicenseProfileCategoryPolicy{
		{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		{Category: "UNKNOWN", Policy: "DENY"},
	}

	if diff := cmp.Diff(want, SCALicenseProfileCategoriesToUpdate(spec, observation)); diff != "" {
		t.Errorf("SCALicenseProfileCategoriesToUpdate() -want +got:\n%s", diff)
	}
}

// TestSCALicenseProfileLicensesToUpdate tests the
// SCALicenseProfileLicensesToUpdate function.
func TestSCALicenseProfileLicensesToUpdate(t *testing.T) {
	t.Parallel()

	spec := []v1alpha1.SCALicenseProfileLicensePolicy{
		{SPDXLicenseID: "MIT", Policy: "ALLOW"},
		{SPDXLicenseID: "GPL-3.0-only", Policy: "ALLOW"},
		{SPDXLicenseID: "Unknown-1.0", Policy: "DENY"},
	}
	observation := []v1alpha1.SCALicenseProfileLicenseObservation{
		{ID: "l1", SPDXLicenseID: "MIT", Policy: "ALLOW"},
		{ID: "l2", SPDXLicenseID: "GPL-3.0-only", Policy: "DENY"},
	}

	updates, missing := SCALicenseProfileLicensesToUpdate(spec, observation)

	wantUpdates := []SCALicenseProfileLicenseUpdate{{ID: "l2", SPDXLicenseID: "GPL-3.0-only", Policy: "ALLOW"}}
	if diff := cmp.Diff(wantUpdates, updates); diff != "" {
		t.Errorf("SCALicenseProfileLicensesToUpdate() updates -want +got:\n%s", diff)
	}

	if diff := cmp.Diff([]string{"Unknown-1.0"}, missing); diff != "" {
		t.Errorf("SCALicenseProfileLicensesToUpdate() missing -want +got:\n%s", diff)
	}
}

// TestSCALicenseProfileProjectsDelta tests the
// SCALicenseProfileProjectsToAssign and SCALicenseProfileProjectsToUnassign
// functions.
func TestSCALicenseProfileProjectsDelta(t *testing.T) {
	t.Parallel()

	spec := []string{"proj-c", "proj-a", "proj-a"}
	observation := []string{"proj-a", "proj-b"}

	if diff := cmp.Diff([]string{"proj-c"}, SCALicenseProfileProjectsToAssign(spec, observation)); diff != "" {
		t.Errorf("SCALicenseProfileProjectsToAssign() -want +got:\n%s", diff)
	}

	if diff := cmp.Diff([]string{"proj-b"}, SCALicenseProfileProjectsToUnassign(spec, observation)); diff != "" {
		t.Errorf("SCALicenseProfileProjectsToUnassign() -want +got:\n%s", diff)
	}
}
