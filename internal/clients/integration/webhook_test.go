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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// newSecretKeySelector creates a test LocalSecretKeySelector.
func newSecretKeySelector(name, key string) *xpv1.LocalSecretKeySelector {
	return &xpv1.LocalSecretKeySelector{
		LocalSecretReference: xpv1.LocalSecretReference{Name: name},
		Key:                  key,
	}
}

// TestLateInitializeWebhook verifies that LateInitializeWebhook does not panic
// for any input combination.
func TestLateInitializeWebhook(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.WebhookParameters
		observation *v1alpha1.WebhookObservation
	}{
		"NilSpec":        {spec: nil, observation: &v1alpha1.WebhookObservation{}},
		"NilObservation": {spec: &v1alpha1.WebhookParameters{}, observation: nil},
		"BothNil":        {spec: nil, observation: nil},
		"BothSet": {
			spec:        &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			observation: &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// LateInitializeWebhook is a no-op; just verify it does not panic.
			LateInitializeWebhook(tc.spec, tc.observation)
		})
	}
}

// TestIsWebhookLateInitialized verifies that the function always returns false
// because LateInitializeWebhook is a no-op.
func TestIsWebhookLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.WebhookParameters
		current *v1alpha1.WebhookParameters
		want    bool
	}{
		"BothNil": {former: nil, current: nil, want: false},
		"SameValues": {
			former:  &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			current: &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			want:    false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsWebhookLateInitialized(tc.former, tc.current)

			if got != tc.want {
				t.Errorf("IsWebhookLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsWebhookUpToDate verifies the up-to-date check for all relevant fields.
func TestIsWebhookUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec          *v1alpha1.WebhookParameters
		observation   *v1alpha1.WebhookObservation
		currentSecret string
		savedSecret   string
		want          bool
	}{
		"NilSpecIsUpToDate": {
			spec:        nil,
			observation: &v1alpha1.WebhookObservation{},
			want:        true,
		},
		"NilObservationIsNotUpToDate": {
			spec:        &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			observation: nil,
			want:        false,
		},
		"NameChangedIsNotUpToDate": {
			spec:        &v1alpha1.WebhookParameters{Name: "new-name", URL: "https://example.com"},
			observation: &v1alpha1.WebhookObservation{Name: "old-name", URL: "https://example.com"},
			want:        false,
		},
		"URLChangedIsNotUpToDate": {
			spec:        &v1alpha1.WebhookParameters{Name: "hook", URL: "https://new.example.com"},
			observation: &v1alpha1.WebhookObservation{Name: "hook", URL: "https://old.example.com"},
			want:        false,
		},
		"SecretAddedIsNotUpToDate": {
			spec: &v1alpha1.WebhookParameters{
				Name:      "hook",
				URL:       "https://example.com",
				SecretRef: newSecretKeySelector("my-secret", "value"),
			},
			observation: &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com", HasSecret: false},
			want:        false,
		},
		"SecretRemovedIsNotUpToDate": {
			spec:        &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			observation: &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com", HasSecret: true},
			want:        false,
		},
		"UpToDateNoSecret": {
			spec:        &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			observation: &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com", HasSecret: false},
			want:        true,
		},
		"UpToDateWithSecretMatches": {
			spec: &v1alpha1.WebhookParameters{
				Name:      "hook",
				URL:       "https://example.com",
				SecretRef: newSecretKeySelector("my-secret", "value"),
			},
			observation:   &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com", HasSecret: true},
			currentSecret: "abc",
			savedSecret:   "abc",
			want:          true,
		},
		"SecretRotatedIsNotUpToDate": {
			spec: &v1alpha1.WebhookParameters{
				Name:      "hook",
				URL:       "https://example.com",
				SecretRef: newSecretKeySelector("my-secret", "value"),
			},
			observation:   &v1alpha1.WebhookObservation{Name: "hook", URL: "https://example.com", HasSecret: true},
			currentSecret: "new-secret",
			savedSecret:   "old-secret",
			want:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsWebhookUpToDate(tc.spec, tc.observation, tc.currentSecret, tc.savedSecret)

			if got != tc.want {
				t.Errorf("IsWebhookUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestFindWebhookByKey verifies that the correct webhook is found by key.
func TestFindWebhookByKey(t *testing.T) {
	t.Parallel()

	webhooks := []sonar.Webhook{
		{Key: "abc123", Name: "hook-a"},
		{Key: "def456", Name: "hook-b"},
	}

	cases := map[string]struct {
		key  string
		want *sonar.Webhook
	}{
		"Found":    {key: "abc123", want: &webhooks[0]},
		"NotFound": {key: "missing", want: nil},
		"EmptyKey": {key: "", want: nil},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindWebhookByKey(webhooks, tc.key)

			if tc.want == nil {
				if got != nil {
					t.Errorf("FindWebhookByKey() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("FindWebhookByKey() = nil, want %v", tc.want)
			}

			if got.Key != tc.want.Key {
				t.Errorf("FindWebhookByKey().Key = %q, want %q", got.Key, tc.want.Key)
			}
		})
	}
}

// TestNewWebhooksClient verifies that NewWebhooksClient returns a non-nil
// client for a valid configuration.
func TestNewWebhooksClient(t *testing.T) {
	t.Parallel()

	cfg := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "test-token",
		BaseURL:  "http://localhost:9000",
	}

	got := NewWebhooksClient(cfg)

	if got == nil {
		t.Error("NewWebhooksClient() returned nil")
	}
}

// TestGenerateWebhookCreateOptions verifies that the options struct is
// correctly built from spec, projectKey and secret.
func TestGenerateWebhookCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec       *v1alpha1.WebhookParameters
		projectKey string
		secret     string
		want       *sonar.WebhooksCreateOptions
	}{
		"GlobalWebhookNoSecret": {
			spec:       &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			projectKey: "",
			secret:     "",
			want:       &sonar.WebhooksCreateOptions{Name: "hook", URL: "https://example.com", Project: "", Secret: ""},
		},
		"ProjectScopedWithSecret": {
			spec:       &v1alpha1.WebhookParameters{Name: "proj-hook", URL: "https://ci.example.com/hook"},
			projectKey: "my-project",
			secret:     "s3cr3t",
			want:       &sonar.WebhooksCreateOptions{Name: "proj-hook", URL: "https://ci.example.com/hook", Project: "my-project", Secret: "s3cr3t"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateWebhookCreateOptions(tc.spec, tc.projectKey, tc.secret)

			if got == nil {
				t.Fatal("GenerateWebhookCreateOptions() returned nil")
			}

			if got.Name != tc.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tc.want.Name)
			}

			if got.URL != tc.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tc.want.URL)
			}

			if got.Project != tc.want.Project {
				t.Errorf("Project = %q, want %q", got.Project, tc.want.Project)
			}

			if got.Secret != tc.want.Secret {
				t.Errorf("Secret = %q, want %q", got.Secret, tc.want.Secret)
			}
		})
	}
}

// TestGenerateWebhookUpdateOptions verifies that the options struct is
// correctly built from webhookKey, spec and secret.
func TestGenerateWebhookUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		webhookKey string
		spec       *v1alpha1.WebhookParameters
		secret     string
		want       *sonar.WebhooksUpdateOptions
	}{
		"UpdateNameAndURL": {
			webhookKey: "key-abc",
			spec:       &v1alpha1.WebhookParameters{Name: "new-name", URL: "https://new.example.com"},
			secret:     "",
			want:       &sonar.WebhooksUpdateOptions{Webhook: "key-abc", Name: "new-name", URL: "https://new.example.com", Secret: ""},
		},
		"UpdateWithSecret": {
			webhookKey: "key-xyz",
			spec:       &v1alpha1.WebhookParameters{Name: "hook", URL: "https://example.com"},
			secret:     "rotated-secret",
			want:       &sonar.WebhooksUpdateOptions{Webhook: "key-xyz", Name: "hook", URL: "https://example.com", Secret: "rotated-secret"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateWebhookUpdateOptions(tc.webhookKey, tc.spec, tc.secret)

			if got == nil {
				t.Fatal("GenerateWebhookUpdateOptions() returned nil")
			}

			if got.Webhook != tc.want.Webhook {
				t.Errorf("Webhook = %q, want %q", got.Webhook, tc.want.Webhook)
			}

			if got.Name != tc.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tc.want.Name)
			}

			if got.URL != tc.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tc.want.URL)
			}

			if got.Secret != tc.want.Secret {
				t.Errorf("Secret = %q, want %q", got.Secret, tc.want.Secret)
			}
		})
	}
}

// TestGenerateWebhookObservation verifies that the observation is correctly
// populated from the SonarQube API response.
func TestGenerateWebhookObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		sonarWebhook *sonar.Webhook
		want         v1alpha1.WebhookObservation
	}{
		"Nil": {sonarWebhook: nil, want: v1alpha1.WebhookObservation{}},
		"Full": {
			sonarWebhook: &sonar.Webhook{Key: "k1", Name: "hook", URL: "https://example.com", HasSecret: true},
			want:         v1alpha1.WebhookObservation{Key: "k1", Name: "hook", URL: "https://example.com", HasSecret: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateWebhookObservation(tc.sonarWebhook)

			if got != tc.want {
				t.Errorf("GenerateWebhookObservation() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
