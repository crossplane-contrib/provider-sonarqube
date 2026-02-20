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
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

func TestGenerateAlmGithubCreateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec    v1alpha1.AlmGithubParameters
		secrets AlmGithubSecrets
		want    *sonar.AlmSettingsCreateGithubOption
	}{
		"AllFieldsWithoutWebhookSecret": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				ClientSecretSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "client-secret",
				},
				PrivateKeySecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "private-key",
				},
				URL: "https://api.github.com",
			},
			secrets: AlmGithubSecrets{
				ClientSecret: "my-client-secret",
				PrivateKey:   "my-private-key",
			},
			want: &sonar.AlmSettingsCreateGithubOption{
				Key:          "github-integration",
				AppID:        "12345",
				ClientID:     "Iv1.abc123",
				ClientSecret: "my-client-secret",
				PrivateKey:   "my-private-key",
				URL:          "https://api.github.com",
			},
		},
		"AllFieldsWithWebhookSecret": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				ClientSecretSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "client-secret",
				},
				PrivateKeySecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "private-key",
				},
				URL: "https://api.github.com",
				WebhookSecretSecretRef: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "webhook-secret",
				},
			},
			secrets: AlmGithubSecrets{
				ClientSecret:  "my-client-secret",
				PrivateKey:    "my-private-key",
				WebhookSecret: ptr.To("my-webhook-secret"),
			},
			want: &sonar.AlmSettingsCreateGithubOption{
				Key:           "github-integration",
				AppID:         "12345",
				ClientID:      "Iv1.abc123",
				ClientSecret:  "my-client-secret",
				PrivateKey:    "my-private-key",
				URL:           "https://api.github.com",
				WebhookSecret: "my-webhook-secret",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateAlmGithubCreateOptions(tc.spec, tc.secrets)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateAlmGithubCreateOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAlmGithubUpdateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec    v1alpha1.AlmGithubParameters
		secrets AlmGithubSecrets
		want    *sonar.AlmSettingsUpdateGithubOption
	}{
		"AllFieldsWithoutWebhookSecret": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				ClientSecretSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "client-secret",
				},
				PrivateKeySecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "private-key",
				},
				URL: "https://api.github.com",
			},
			secrets: AlmGithubSecrets{
				ClientSecret: "my-client-secret",
				PrivateKey:   "my-private-key",
			},
			want: &sonar.AlmSettingsUpdateGithubOption{
				Key:          "github-integration",
				AppID:        "12345",
				ClientID:     "Iv1.abc123",
				ClientSecret: "my-client-secret",
				PrivateKey:   "my-private-key",
				URL:          "https://api.github.com",
			},
		},
		"AllFieldsWithWebhookSecret": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				ClientSecretSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "client-secret",
				},
				PrivateKeySecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "private-key",
				},
				URL: "https://api.github.com",
				WebhookSecretSecretRef: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "secret", Namespace: "default"},
					Key:             "webhook-secret",
				},
			},
			secrets: AlmGithubSecrets{
				ClientSecret:  "my-client-secret",
				PrivateKey:    "my-private-key",
				WebhookSecret: ptr.To("my-webhook-secret"),
			},
			want: &sonar.AlmSettingsUpdateGithubOption{
				Key:           "github-integration",
				AppID:         "12345",
				ClientID:      "Iv1.abc123",
				ClientSecret:  "my-client-secret",
				PrivateKey:    "my-private-key",
				URL:           "https://api.github.com",
				WebhookSecret: "my-webhook-secret",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateAlmGithubUpdateOptions(tc.spec, tc.secrets)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateAlmGithubUpdateOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAlmDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key  string
		want *sonar.AlmSettingsDeleteOption
	}{
		"BasicDeleteOption": {
			key: "github-integration",
			want: &sonar.AlmSettingsDeleteOption{
				Key: "github-integration",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateAlmDeleteOptions(tc.key)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateAlmDeleteOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindGithubDefinition(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		definitions *sonar.AlmSettingsListDefinitions
		key         string
		want        *sonar.GithubDefinition
	}{
		"NilDefinitionsReturnsNil": {
			definitions: nil,
			key:         "github-integration",
			want:        nil,
		},
		"EmptyDefinitionsReturnsNil": {
			definitions: &sonar.AlmSettingsListDefinitions{},
			key:         "github-integration",
			want:        nil,
		},
		"NoMatchingKeyReturnsNil": {
			definitions: &sonar.AlmSettingsListDefinitions{
				Github: []sonar.GithubDefinition{
					{Key: "other-key", AppID: "111", ClientID: "client1", URL: "https://api.github.com"},
				},
			},
			key:  "github-integration",
			want: nil,
		},
		"MatchingKeyReturnsDefinition": {
			definitions: &sonar.AlmSettingsListDefinitions{
				Github: []sonar.GithubDefinition{
					{Key: "other-key", AppID: "111", ClientID: "client1", URL: "https://api.github.com"},
					{Key: "github-integration", AppID: "12345", ClientID: "Iv1.abc123", URL: "https://api.github.com"},
				},
			},
			key: "github-integration",
			want: &sonar.GithubDefinition{
				Key: "github-integration", AppID: "12345", ClientID: "Iv1.abc123", URL: "https://api.github.com",
			},
		},
		"FirstMatchReturned": {
			definitions: &sonar.AlmSettingsListDefinitions{
				Github: []sonar.GithubDefinition{
					{Key: "github-integration", AppID: "first", ClientID: "client1", URL: "https://url1"},
					{Key: "github-integration", AppID: "second", ClientID: "client2", URL: "https://url2"},
				},
			},
			key: "github-integration",
			want: &sonar.GithubDefinition{
				Key: "github-integration", AppID: "first", ClientID: "client1", URL: "https://url1",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindGithubDefinition(tc.definitions, tc.key)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FindGithubDefinition() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAlmGithubObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		definition *sonar.GithubDefinition
		want       v1alpha1.AlmGithubObservation
	}{
		"BasicObservation": {
			definition: &sonar.GithubDefinition{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			want: v1alpha1.AlmGithubObservation{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
		},
		"EmptyFieldsObservation": {
			definition: &sonar.GithubDefinition{
				Key: "minimal",
			},
			want: v1alpha1.AlmGithubObservation{
				Key: "minimal",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateAlmGithubObservation(tc.definition)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateAlmGithubObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsAlmGithubUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        v1alpha1.AlmGithubParameters
		observation v1alpha1.AlmGithubObservation
		want        bool
	}{
		"AllFieldsMatchReturnsTrue": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			observation: v1alpha1.AlmGithubObservation{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			want: true,
		},
		"DifferentKeyReturnsFalse": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			observation: v1alpha1.AlmGithubObservation{
				Key:      "different-key",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			want: false,
		},
		"DifferentAppIDReturnsFalse": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			observation: v1alpha1.AlmGithubObservation{
				Key:      "github-integration",
				AppID:    "99999",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			want: false,
		},
		"DifferentClientIDReturnsFalse": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			observation: v1alpha1.AlmGithubObservation{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.different",
				URL:      "https://api.github.com",
			},
			want: false,
		},
		"DifferentURLReturnsFalse": {
			spec: v1alpha1.AlmGithubParameters{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://api.github.com",
			},
			observation: v1alpha1.AlmGithubObservation{
				Key:      "github-integration",
				AppID:    "12345",
				ClientID: "Iv1.abc123",
				URL:      "https://github.example.com/api",
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsAlmGithubUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsAlmGithubUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
