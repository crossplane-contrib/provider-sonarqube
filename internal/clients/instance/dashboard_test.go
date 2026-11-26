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
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// dashboardsListStub is a DashboardsClient serving only List.
type dashboardsListStub struct {
	DashboardsClient

	list func(opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error)
}

// List implements DashboardsClient.List.
func (s dashboardsListStub) List(_ context.Context, opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
	return s.list(opt)
}

// TestDashboardTarget tests the DashboardTarget function.
func TestDashboardTarget(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec     v1alpha1.DashboardParameters
		wantType string
		wantKey  string
	}{
		"Project":   {spec: v1alpha1.DashboardParameters{ProjectKey: new("p")}, wantType: "project", wantKey: "p"},
		"Portfolio": {spec: v1alpha1.DashboardParameters{PortfolioKey: new("f")}, wantType: "portfolio", wantKey: "f"},
		"None":      {spec: v1alpha1.DashboardParameters{ProjectKey: new("")}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotType, gotKey := DashboardTarget(&tc.spec)
			if gotType != tc.wantType || gotKey != tc.wantKey {
				t.Errorf("DashboardTarget() = (%q, %q), want (%q, %q)", gotType, gotKey, tc.wantType, tc.wantKey)
			}
		})
	}
}

// TestGenerateDashboardOptions tests the create and update options.
func TestGenerateDashboardOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.DashboardParameters{Name: "n", Layout: "{}", Description: new("d")}

	wantCreate := &sonar.DashboardsCreateOptions{Name: "n", Layout: "{}", Description: "d", ResourceType: "project", ResourceId: "id"}
	if diff := cmp.Diff(wantCreate, GenerateDashboardCreateOptions(spec, "project", "id")); diff != "" {
		t.Errorf("GenerateDashboardCreateOptions() mismatch (-want +got):\n%s", diff)
	}

	wantUpdate := &sonar.DashboardsUpdateOptions{Name: new("n"), Layout: new("{}"), Description: new("d")}
	if diff := cmp.Diff(wantUpdate, GenerateDashboardUpdateOptions(spec)); diff != "" {
		t.Errorf("GenerateDashboardUpdateOptions() mismatch (-want +got):\n%s", diff)
	}
}

// TestFindDashboardByName tests the paginated dashboard lookup.
func TestFindDashboardByName(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	// pages serves two pages of one dashboard each.
	pages := func(opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
		if opt.ResourceId != "id" || opt.ResourceType != "project" || opt.PageSize != dashboardsPageSize {
			return nil, nil, errors.Errorf("unexpected options %+v", opt)
		}

		names := []string{"first", "second"}

		return &sonar.DashboardsDashboardsResponse{
			Dashboards: []sonar.DashboardsDashboardItem{{Id: names[opt.PageIndex-1] + "-id", Name: names[opt.PageIndex-1]}},
			Page:       sonar.PageResponseV2{PageIndex: opt.PageIndex, PageSize: 1, Total: 2},
		}, nil, nil
	}

	cases := map[string]struct {
		list      func(opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error)
		name      string
		wantID    string
		wantFound bool
		wantErr   bool
	}{
		"FoundOnSecondPage": {list: pages, name: "second", wantID: "second-id", wantFound: true},
		"NotFound":          {list: pages, name: "Second"},
		"ListError": {
			list: func(_ *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
				return nil, nil, errBoom
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, found, err := FindDashboardByName(context.Background(), dashboardsListStub{list: tc.list}, "project", "id", tc.name)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FindDashboardByName() error = %v, wantErr %v", err, tc.wantErr)
			}

			if found != tc.wantFound || got.Id != tc.wantID {
				t.Errorf("FindDashboardByName() = (%q, %v), want (%q, %v)", got.Id, found, tc.wantID, tc.wantFound)
			}
		})
	}
}

// TestGenerateDashboardObservation tests the observation generation.
func TestGenerateDashboardObservation(t *testing.T) {
	t.Parallel()

	got := GenerateDashboardObservation(&sonar.DashboardsDashboardResponse{
		Id: "id", Name: "n", Description: "d", Layout: "{}", ResourceType: "project", ResourceId: "rid", CreatedAt: 1, UpdatedAt: 2,
	})
	want := v1alpha1.DashboardObservation{
		ID: "id", Name: "n", Description: "d", Layout: "{}", ResourceType: "project", ResourceID: "rid", CreatedAt: 1, UpdatedAt: 2,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateDashboardObservation() mismatch (-want +got):\n%s", diff)
	}
}

// TestLateInitializeDashboard tests late initialization of the description.
func TestLateInitializeDashboard(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec     *string
		observed string
		want     *string
		changed  bool
	}{
		"Filled":        {observed: "d", want: new("d"), changed: true},
		"KeptWhenSet":   {spec: new(""), observed: "d", want: new("")},
		"EmptyObserved": {},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.DashboardParameters{Description: tc.spec}
			former := spec.DeepCopy()

			LateInitializeDashboard(spec, &v1alpha1.DashboardObservation{Description: tc.observed})

			if diff := cmp.Diff(tc.want, spec.Description); diff != "" {
				t.Errorf("Description mismatch (-want +got):\n%s", diff)
			}

			if got := IsDashboardLateInitialized(former, spec); got != tc.changed {
				t.Errorf("IsDashboardLateInitialized() = %v, want %v", got, tc.changed)
			}
		})
	}

	LateInitializeDashboard(nil, nil)

	if IsDashboardLateInitialized(nil, nil) {
		t.Error("IsDashboardLateInitialized(nil, nil) = true, want false")
	}
}

// TestIsDashboardUpToDate tests the up to date check.
func TestIsDashboardUpToDate(t *testing.T) {
	t.Parallel()

	observation := &v1alpha1.DashboardObservation{Name: "n", Description: "d", Layout: `{"a":1}`}

	cases := map[string]struct {
		spec        *v1alpha1.DashboardParameters
		observation *v1alpha1.DashboardObservation
		want        bool
	}{
		"NilSpec":          {observation: observation, want: true},
		"NilObservation":   {spec: &v1alpha1.DashboardParameters{}},
		"Equal":            {spec: &v1alpha1.DashboardParameters{Name: "n", Description: new("d"), Layout: `{ "a": 1 }`}, observation: observation, want: true},
		"UnsetDescription": {spec: &v1alpha1.DashboardParameters{Name: "n", Layout: `{"a":1}`}, observation: observation, want: true},
		"Name":             {spec: &v1alpha1.DashboardParameters{Name: "x", Layout: `{"a":1}`}, observation: observation},
		"Description":      {spec: &v1alpha1.DashboardParameters{Name: "n", Description: new(""), Layout: `{"a":1}`}, observation: observation},
		"Layout":           {spec: &v1alpha1.DashboardParameters{Name: "n", Layout: `{"a":2}`}, observation: observation},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsDashboardUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Errorf("IsDashboardUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAreDashboardLayoutsEquivalent tests the layout comparison.
func TestAreDashboardLayoutsEquivalent(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		desired  string
		observed string
		want     bool
	}{
		"VerbatimNonJSON":      {desired: "layout", observed: "layout", want: true},
		"DifferentNonJSON":     {desired: "layout", observed: "other"},
		"ObservedNotJSON":      {desired: "{}", observed: "other"},
		"Formatting":           {desired: "{\n  \"b\": [1, true, null],\n  \"a\": \"x\"\n}\n", observed: `{"a":"x","b":[1,true,null]}`, want: true},
		"GeneratedIDs":         {desired: `{"widgets":[{"type":"a"}]}`, observed: `{"id":"1","widgets":[{"id":"2","type":"a"}]}`, want: true},
		"MatchingIDs":          {desired: `{"widgets":[{"id":"w"}]}`, observed: `{"widgets":[{"id":"w"}]}`, want: true},
		"ChangedID":            {desired: `{"widgets":[{"id":"w"}]}`, observed: `{"widgets":[{"id":"x"}]}`},
		"MissingDesiredID":     {desired: `{"id":"w"}`, observed: `{}`},
		"ExtraObservedMember":  {desired: `{"a":1}`, observed: `{"a":1,"b":2}`},
		"MissingMember":        {desired: `{"a":1,"b":2}`, observed: `{"a":1}`},
		"ExtraObservedElement": {desired: `{"widgets":[]}`, observed: `{"widgets":[{}]}`},
		"ReorderedElements":    {desired: `[1,2]`, observed: `[2,1]`},
		"ObjectVersusArray":    {desired: `{}`, observed: `[]`},
		"ArrayVersusObject":    {desired: `[]`, observed: `{}`},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := AreDashboardLayoutsEquivalent(tc.desired, tc.observed); got != tc.want {
				t.Errorf("AreDashboardLayoutsEquivalent(%q, %q) = %v, want %v", tc.desired, tc.observed, got, tc.want)
			}
		})
	}
}

// TestNewDashboardClients tests the dashboard client constructors.
func TestNewDashboardClients(t *testing.T) {
	t.Parallel()

	if NewDashboardsClient(newTestConfig()) == nil {
		t.Error("NewDashboardsClient() returned nil")
	}

	if NewComponentNavigationClient(newTestConfig()) == nil {
		t.Error("NewComponentNavigationClient() returned nil")
	}
}
