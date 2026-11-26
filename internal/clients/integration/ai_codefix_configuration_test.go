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

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// fixSuggestionsRequest is a request recorded by the test server.
type fixSuggestionsRequest struct {
	method      string
	path        string
	contentType string
	body        map[string]any
}

// newFixSuggestionsTestServer returns a server answering the enablement
// endpoints, and the requests it received.
func newFixSuggestionsTestServer(t *testing.T) (*httptest.Server, *[]fixSuggestionsRequest) {
	t.Helper()

	var requests []fixSuggestionsRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded := fixSuggestionsRequest{method: r.Method, path: r.URL.Path, contentType: r.Header.Get("Content-Type")}

		data, _ := io.ReadAll(r.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, &recorded.body)
		}

		requests = append(requests, recorded)

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"enablement":"ENABLED_FOR_SOME_PROJECTS","enabledProjectKeys":["a"]}`))

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	return server, &requests
}

// TestFixSuggestionsClient tests that the client reads the enablement
// through the SDK and sends the projects along with the enablement.
func TestFixSuggestionsClient(t *testing.T) {
	t.Parallel()

	server, requests := newFixSuggestionsTestServer(t)
	client := NewFixSuggestionsClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"})

	got, _, err := client.GetEnablement(context.Background()) //nolint:bodyclose // fake or test server responses
	if err != nil {
		t.Fatalf("GetEnablement() error = %v", err)
	}

	if got.Enablement != v1alpha1.AICodeFixEnabledForSomeProjects || !reflect.DeepEqual(got.EnabledProjectKeys, []string{"a"}) {
		t.Errorf("GetEnablement() = %+v", got)
	}

	resp, err := client.SetEnablement(context.Background(), &sonar.FixSuggestionsSetEnablementOptions{ //nolint:bodyclose // fake or test server responses
		Enablement:         v1alpha1.AICodeFixEnabledForSomeProjects,
		EnabledProjectKeys: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("SetEnablement() error = %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("SetEnablement() status = %d, want 204", resp.StatusCode)
	}

	if len(*requests) != 2 {
		t.Fatalf("requests = %+v, want 2", *requests)
	}

	patch := (*requests)[1]
	if patch.method != http.MethodPatch || patch.path != "/api/v2/fix-suggestions/feature-enablements" || patch.contentType != "application/merge-patch+json" {
		t.Errorf("PATCH request = %+v", patch)
	}

	wantBody := map[string]any{"enablement": "ENABLED_FOR_SOME_PROJECTS", "enabledProjectKeys": []any{"a", "b"}}
	if !reflect.DeepEqual(patch.body, wantBody) {
		t.Errorf("PATCH body = %v, want %v", patch.body, wantBody)
	}
}

// TestFixSuggestionsClientSetEnablementValidation tests that invalid
// options are rejected before any request.
func TestFixSuggestionsClientSetEnablementValidation(t *testing.T) {
	t.Parallel()

	server, requests := newFixSuggestionsTestServer(t)
	client := NewFixSuggestionsClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"})

	for name, opt := range map[string]*sonar.FixSuggestionsSetEnablementOptions{
		"Nil":     nil,
		"Empty":   {},
		"Invalid": {Enablement: "SOMETIMES"},
	} {
		_, err := client.SetEnablement(context.Background(), opt) //nolint:bodyclose // fake or test server responses
		if err == nil {
			t.Errorf("SetEnablement(%s) expected an error", name)
		}
	}

	if len(*requests) != 0 {
		t.Errorf("requests = %+v, want none", *requests)
	}
}

// TestGenerateAICodeFixEnablementOptions tests building the request body.
func TestGenerateAICodeFixEnablementOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec v1alpha1.AICodeFixConfigurationParameters
		want *sonar.FixSuggestionsSetEnablementOptions
	}{
		"AllProjectsDropsKeys": {
			spec: v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForAllProjects, EnabledProjectKeys: []string{"a"}},
			want: &sonar.FixSuggestionsSetEnablementOptions{Enablement: v1alpha1.AICodeFixEnabledForAllProjects},
		},
		"SomeProjects": {
			spec: v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"a"}},
			want: &sonar.FixSuggestionsSetEnablementOptions{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"a"}},
		},
		"SomeProjectsNone": {
			spec: v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects},
			want: &sonar.FixSuggestionsSetEnablementOptions{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := GenerateAICodeFixEnablementOptions(&tc.spec); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("GenerateAICodeFixEnablementOptions() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestGenerateAICodeFixConfigurationObservation tests the observation.
func TestGenerateAICodeFixConfigurationObservation(t *testing.T) {
	t.Parallel()

	if got := GenerateAICodeFixConfigurationObservation(nil); !reflect.DeepEqual(got, v1alpha1.AICodeFixConfigurationObservation{}) {
		t.Errorf("GenerateAICodeFixConfigurationObservation(nil) = %#v, want empty", got)
	}

	got := GenerateAICodeFixConfigurationObservation(&sonar.FixSuggestionsFeatureEnablement{Enablement: v1alpha1.AICodeFixDisabled, EnabledProjectKeys: []string{"stale"}})
	if got.Enablement != v1alpha1.AICodeFixDisabled || got.EnabledProjectKeys != nil {
		t.Errorf("GenerateAICodeFixConfigurationObservation(DISABLED) = %#v, want no projects", got)
	}

	got = GenerateAICodeFixConfigurationObservation(&sonar.FixSuggestionsFeatureEnablement{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"a"}})
	if !reflect.DeepEqual(got.EnabledProjectKeys, []string{"a"}) {
		t.Errorf("GenerateAICodeFixConfigurationObservation(SOME) = %#v, want project a", got)
	}
}

// TestIsAICodeFixConfigurationUpToDate tests drift detection.
func TestIsAICodeFixConfigurationUpToDate(t *testing.T) {
	t.Parallel()

	some := func(keys ...string) *v1alpha1.AICodeFixConfigurationParameters {
		return &v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: keys}
	}

	cases := map[string]struct {
		spec        *v1alpha1.AICodeFixConfigurationParameters
		observation *v1alpha1.AICodeFixConfigurationObservation
		want        bool
	}{
		"NilSpec":        {want: true},
		"NilObservation": {spec: some("a")},
		"EnablementDiffers": {
			spec:        &v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForAllProjects},
			observation: &v1alpha1.AICodeFixConfigurationObservation{Enablement: v1alpha1.AICodeFixDisabled},
		},
		"AllProjects": {
			spec:        &v1alpha1.AICodeFixConfigurationParameters{Enablement: v1alpha1.AICodeFixEnabledForAllProjects},
			observation: &v1alpha1.AICodeFixConfigurationObservation{Enablement: v1alpha1.AICodeFixEnabledForAllProjects},
			want:        true,
		},
		"SomeProjectsReordered": {
			spec:        some("a", "b"),
			observation: &v1alpha1.AICodeFixConfigurationObservation{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"b", "a"}},
			want:        true,
		},
		"SomeProjectsDiffer": {
			spec:        some("a", "b"),
			observation: &v1alpha1.AICodeFixConfigurationObservation{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"a"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsAICodeFixConfigurationUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Errorf("IsAICodeFixConfigurationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
