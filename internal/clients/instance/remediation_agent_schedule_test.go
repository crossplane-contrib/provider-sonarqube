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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestRemediationAgentSchedulesClientPaths verifies the paths sent, the
// instance-wide one being escaped only once.
func TestRemediationAgentSchedulesClientPaths(t *testing.T) {
	t.Parallel()

	var requests []string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}

		request := req.Method + " " + req.URL.Path
		if len(body) != 0 {
			request += " " + string(body)
		}

		requests = append(requests, request)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"x"}`))
	}))
	defer server.Close()

	client := NewRemediationAgentSchedulesClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"})
	options := &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{Frequency: "DAILY"}

	for _, id := range []string{GlobalRemediationAgentScheduleID, "my:project"} {
		got, resp, err := client.GetScheduledAgentConfig(context.Background(), id) //nolint:bodyclose // closed by the SDK
		if err != nil || resp == nil || got.Id != "x" {
			t.Fatalf("GetScheduledAgentConfig(%q) = %v, %v", id, got, err)
		}

		_, _, err = client.UpdateScheduledAgentConfig(context.Background(), id, options) //nolint:bodyclose // closed by the SDK
		if err != nil {
			t.Fatalf("UpdateScheduledAgentConfig(%q): %v", id, err)
		}
	}

	want := []string{
		"GET /api/v2/remediation-agent/scheduled-agent-configs/*",
		`PUT /api/v2/remediation-agent/scheduled-agent-configs/* {"frequency":"DAILY"}`,
		"GET /api/v2/remediation-agent/scheduled-agent-configs/my:project",
		`PUT /api/v2/remediation-agent/scheduled-agent-configs/my:project {"frequency":"DAILY"}`,
	}
	if diff := cmp.Diff(want, requests); diff != "" {
		t.Errorf("requests: -want, +got:\n%s", diff)
	}
}

// TestRemediationAgentSchedulesClientGlobalError verifies errors
// of the instance-wide endpoint keep their response.
func TestRemediationAgentSchedulesClientGlobalError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := NewRemediationAgentSchedulesClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"})

	got, resp, err := client.GetScheduledAgentConfig(context.Background(), GlobalRemediationAgentScheduleID) //nolint:bodyclose // closed by the SDK
	if err == nil || got != nil || !common.IsResponseNotFound(resp) {
		t.Errorf("GetScheduledAgentConfig() = %v, %v, want a not found error", got, err)
	}
}

// TestRemediationAgentScheduleID verifies the identifier of each scope.
func TestRemediationAgentScheduleID(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec v1alpha1.RemediationAgentScheduleParameters
		want string
	}{
		"Global":             {want: "*"},
		"Project":            {spec: v1alpha1.RemediationAgentScheduleParameters{ProjectKey: new("key"), ProjectKeyRef: &xpv1.Reference{Name: "p"}}, want: "key"},
		"UnresolvedRef":      {spec: v1alpha1.RemediationAgentScheduleParameters{ProjectKeyRef: &xpv1.Reference{Name: "p"}}},
		"UnresolvedSelector": {spec: v1alpha1.RemediationAgentScheduleParameters{ProjectKeySelector: &xpv1.Selector{}}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := RemediationAgentScheduleID(&tc.spec); got != tc.want {
				t.Errorf("RemediationAgentScheduleID() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGenerateRemediationAgentScheduleObservation verifies the
// observation built from the API response.
func TestGenerateRemediationAgentScheduleObservation(t *testing.T) {
	t.Parallel()

	if diff := cmp.Diff(v1alpha1.RemediationAgentScheduleObservation{}, GenerateRemediationAgentScheduleObservation(nil)); diff != "" {
		t.Errorf("nil config: -want, +got:\n%s", diff)
	}

	got := GenerateRemediationAgentScheduleObservation(&sonar.RemediationAgentScheduledAgentConfigResource{
		Id: "*", Enabled: true, Frequency: "WEEKLY", DayOfWeek: "MONDAY", Hour: 3, Timezone: "UTC",
		NextRunUtc: 1000, MaxIssuesPerRun: 5, MaxOpenPRs: -1,
		ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"b", "a"},
	})

	want := v1alpha1.RemediationAgentScheduleObservation{
		ID: "*", Enabled: true, Frequency: "WEEKLY", DayOfWeek: "MONDAY", Hour: 3, Timezone: "UTC",
		NextRunTime: &metav1.Time{Time: time.UnixMilli(1000)}, MaxIssuesPerRun: 5, MaxOpenPRs: -1,
		ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"a", "b"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}

// TestLateInitializeRemediationAgentSchedule verifies which fields
// are late initialized.
func TestLateInitializeRemediationAgentSchedule(t *testing.T) {
	t.Parallel()

	observation := v1alpha1.RemediationAgentScheduleObservation{
		Timezone: "UTC", MaxIssuesPerRun: 5, MaxOpenPRs: 3,
		ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"a"},
	}
	filled := v1alpha1.RemediationAgentScheduleParameters{
		Timezone: new("UTC"), MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)),
	}

	withSelection := filled.DeepCopy()
	withSelection.ProjectSelectionMode = new("SELECTED")
	withSelection.SelectedProjectKeys = []string{"a"}

	emptySelection := observation.DeepCopy()
	emptySelection.SelectedProjectKeys = nil

	cases := map[string]struct {
		spec        v1alpha1.RemediationAgentScheduleParameters
		observation v1alpha1.RemediationAgentScheduleObservation
		want        v1alpha1.RemediationAgentScheduleParameters
	}{
		"Global":         {observation: observation, want: *withSelection},
		"EmptySelection": {observation: *emptySelection, want: filled},
		"Project": {
			spec:        v1alpha1.RemediationAgentScheduleParameters{ProjectKey: new("p")},
			observation: observation,
			want:        v1alpha1.RemediationAgentScheduleParameters{ProjectKey: new("p"), Timezone: new("UTC"), MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3))},
		},
		"EmptyObservation": {},
		"SpecWins": {
			spec:        v1alpha1.RemediationAgentScheduleParameters{Timezone: new("Europe/Paris"), ProjectSelectionMode: new("ALL")},
			observation: observation,
			want:        v1alpha1.RemediationAgentScheduleParameters{Timezone: new("Europe/Paris"), MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)), ProjectSelectionMode: new("ALL")},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			former := tc.spec.DeepCopy()
			LateInitializeRemediationAgentSchedule(&tc.spec, &tc.observation)

			if diff := cmp.Diff(tc.want, tc.spec); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}

			if got, want := IsRemediationAgentScheduleLateInitialized(former, &tc.spec), !cmp.Equal(*former, tc.want); got != want {
				t.Errorf("IsRemediationAgentScheduleLateInitialized() = %v, want %v", got, want)
			}
		})
	}

	LateInitializeRemediationAgentSchedule(nil, nil)

	if IsRemediationAgentScheduleLateInitialized(nil, nil) {
		t.Error("IsRemediationAgentScheduleLateInitialized(nil, nil) = true, want false")
	}
}

// TestIsRemediationAgentScheduleUpToDate verifies drift detection.
func TestIsRemediationAgentScheduleUpToDate(t *testing.T) {
	t.Parallel()

	spec := func(mutate func(*v1alpha1.RemediationAgentScheduleParameters)) *v1alpha1.RemediationAgentScheduleParameters {
		params := &v1alpha1.RemediationAgentScheduleParameters{
			Enabled: new(true), Frequency: "WEEKLY", DayOfWeek: new("MONDAY"), Hour: new(int32(0)),
			Timezone: new("UTC"), MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)),
			ProjectSelectionMode: new("SELECTED"), SelectedProjectKeys: []string{"b", "a"},
		}
		mutate(params)

		return params
	}
	observation := &v1alpha1.RemediationAgentScheduleObservation{
		Enabled: true, Frequency: "WEEKLY", DayOfWeek: "MONDAY", Timezone: "UTC", MaxIssuesPerRun: 5, MaxOpenPRs: 3,
		ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"a", "b"},
	}

	type params = v1alpha1.RemediationAgentScheduleParameters

	cases := map[string]struct {
		mutate func(*params)
		want   bool
	}{
		"UpToDate":        {mutate: func(*params) {}, want: true},
		"Enabled":         {mutate: func(p *params) { p.Enabled = new(false) }},
		"Frequency":       {mutate: func(p *params) { p.Frequency = "DAILY" }},
		"DayOfWeek":       {mutate: func(p *params) { p.DayOfWeek = new("FRIDAY") }},
		"Hour":            {mutate: func(p *params) { p.Hour = new(int32(1)) }},
		"Timezone":        {mutate: func(p *params) { p.Timezone = new("Europe/Paris") }},
		"MaxIssuesPerRun": {mutate: func(p *params) { p.MaxIssuesPerRun = new(int32(6)) }},
		"MaxOpenPRs":      {mutate: func(p *params) { p.MaxOpenPRs = new(int32(-1)) }},
		"Mode":            {mutate: func(p *params) { p.ProjectSelectionMode = new("ALL") }},
		"Projects":        {mutate: func(p *params) { p.SelectedProjectKeys = []string{"a"} }},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsRemediationAgentScheduleUpToDate(spec(tc.mutate), observation); got != tc.want {
				t.Errorf("IsRemediationAgentScheduleUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}

	if !IsRemediationAgentScheduleUpToDate(nil, observation) {
		t.Error("a nil spec must be up to date")
	}

	if IsRemediationAgentScheduleUpToDate(&params{}, nil) {
		t.Error("a nil observation must not be up to date")
	}
}

// TestGenerateRemediationAgentScheduleOptions verifies the options sent
// to SonarQube.
func TestGenerateRemediationAgentScheduleOptions(t *testing.T) {
	t.Parallel()

	observation := &v1alpha1.RemediationAgentScheduleObservation{
		Timezone: "Europe/Paris", MaxIssuesPerRun: 10, MaxOpenPRs: -1,
		ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"a"},
	}

	cases := map[string]struct {
		spec        v1alpha1.RemediationAgentScheduleParameters
		observation *v1alpha1.RemediationAgentScheduleObservation
		want        *sonar.RemediationAgentUpdateScheduledAgentConfigOptions
	}{
		"GlobalDefaults": {
			spec: v1alpha1.RemediationAgentScheduleParameters{Frequency: "DAILY"},
			want: &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
				Frequency: "DAILY", Timezone: "UTC", MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)), ProjectSelectionMode: "ALL",
			},
		},
		"GlobalKeepsObserved": {
			spec:        v1alpha1.RemediationAgentScheduleParameters{Frequency: "DAILY"},
			observation: observation,
			want: &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
				Frequency: "DAILY", Timezone: "Europe/Paris", MaxIssuesPerRun: new(int32(10)), MaxOpenPRs: new(int32(-1)),
				ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"a"},
			},
		},
		"GlobalSelectedFromSpec": {
			spec:        v1alpha1.RemediationAgentScheduleParameters{Frequency: "DAILY", ProjectSelectionMode: new("SELECTED"), SelectedProjectKeys: []string{"b"}},
			observation: observation,
			want: &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
				Frequency: "DAILY", Timezone: "Europe/Paris", MaxIssuesPerRun: new(int32(10)), MaxOpenPRs: new(int32(-1)),
				ProjectSelectionMode: "SELECTED", SelectedProjectKeys: []string{"b"},
			},
		},
		"ProjectHasNoSelection": {
			spec:        v1alpha1.RemediationAgentScheduleParameters{ProjectKey: new("p"), Frequency: "WEEKLY", DayOfWeek: new("SUNDAY")},
			observation: observation,
			want: &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
				Frequency: "WEEKLY", DayOfWeek: "SUNDAY", Timezone: "Europe/Paris", MaxIssuesPerRun: new(int32(10)), MaxOpenPRs: new(int32(-1)),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRemediationAgentScheduleUpdateOptions(&tc.spec, tc.observation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}

	got := GenerateRemediationAgentScheduleCreateOptions(&v1alpha1.RemediationAgentScheduleParameters{
		ProjectKey: new("p"), Enabled: new(true), Frequency: "DAILY", Hour: new(int32(4)),
	})

	want := &sonar.RemediationAgentCreateScheduledAgentConfigOptions{
		ProjectKey: "p", Enabled: new(true), Frequency: "DAILY", Hour: new(int32(4)),
		Timezone: "UTC", MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("create options: -want, +got:\n%s", diff)
	}
}
