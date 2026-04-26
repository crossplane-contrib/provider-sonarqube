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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestLateInitializeALMGitHub(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeALMGitHub(nil, nil)

		spec := &v1alpha1.ALMGitHubParameters{}
		LateInitializeALMGitHub(spec, nil)
		LateInitializeALMGitHub(nil, &v1alpha1.ALMGitHubObservation{})
	})

	t.Run("NoOpLateInit", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.ALMGitHubParameters{
			URL:      "https://api.github.com",
			Key:      "github-main",
			AppID:    "123456",
			ClientID: "Iv1.abc",
		}
		obs := &v1alpha1.ALMGitHubObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://api.github.com", Key: "github-main"},
			AppID:                "123456",
			ClientID:             "Iv1.abc",
		}

		LateInitializeALMGitHub(spec, obs)

		if spec.URL != "https://api.github.com" || spec.Key != "github-main" || spec.AppID != "123456" || spec.ClientID != "Iv1.abc" {
			t.Fatalf("LateInitializeALMGitHub() mutated spec unexpectedly: %+v", spec)
		}
	})
}

func TestIsALMGitHubLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMGitHubParameters
		current *v1alpha1.ALMGitHubParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMGitHubParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMGitHubParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former:  &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			want:    false,
		},
		"URLChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: "https://github.example.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			want:    true,
		},
		"AppIDChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "654321", ClientID: "Iv1.abc"},
			want:    true,
		},
		"ClientIDChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.xyz"},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMGitHubLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMGitHubLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsALMGitHubUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitHubParameters{URL: "https://api.github.com", Key: "github-main", AppID: "123456", ClientID: "Iv1.abc"}
	obs := &v1alpha1.ALMGitHubObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://api.github.com", Key: "github-main"},
		AppID:                "123456",
		ClientID:             "Iv1.abc",
	}

	cases := map[string]struct {
		spec               *v1alpha1.ALMGitHubParameters
		observation        *v1alpha1.ALMGitHubObservation
		clientSecret       string
		savedClientSecret  string
		privateKey         string
		savedPrivateKey    string
		webhookSecret      string
		savedWebhookSecret string
		want               bool
	}{
		"NilSpec": {
			spec:         nil,
			observation:  obs,
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: true,
		},
		"NilObservation": {
			spec:         spec,
			observation:  nil,
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"UpToDate": {
			spec:         spec,
			observation:  obs,
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			webhookSecret: "hook", savedWebhookSecret: "hook",
			want: true,
		},
		"ClientSecretChanged": {
			spec:         spec,
			observation:  obs,
			clientSecret: "new-secret", savedClientSecret: "old-secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"PrivateKeyChanged": {
			spec:         spec,
			observation:  obs,
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "new-key", savedPrivateKey: "old-key",
			want: false,
		},
		"WebhookSecretChanged": {
			spec:         spec,
			observation:  obs,
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			webhookSecret: "new-hook", savedWebhookSecret: "old-hook",
			want: false,
		},
		"URLChanged": {
			spec:         spec,
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://other.github.com", Key: "github-main"}, AppID: "123456", ClientID: "Iv1.abc"},
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"AppIDChanged": {
			spec:         spec,
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://api.github.com", Key: "github-main"}, AppID: "other-app", ClientID: "Iv1.abc"},
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"ClientIDChanged": {
			spec:         spec,
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://api.github.com", Key: "github-main"}, AppID: "123456", ClientID: "Iv1.xyz"},
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsALMGitHubUpToDate(tc.spec, tc.observation, tc.clientSecret, tc.savedClientSecret, tc.privateKey, tc.savedPrivateKey, tc.webhookSecret, tc.savedWebhookSecret)
			if got != tc.want {
				t.Fatalf("IsALMGitHubUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateALMGitHubCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitHubParameters{
		URL:      "https://api.github.com",
		Key:      "github-main",
		AppID:    "123456",
		ClientID: "Iv1.abc",
	}

	got := GenerateALMGitHubCreateOptions(spec, "client-secret", "private-key", "webhook-secret")
	if got == nil {
		t.Fatal("GenerateALMGitHubCreateOptions() returned nil")
	}

	if got.URL != "https://api.github.com" || got.Key != "github-main" || got.AppID != "123456" || got.ClientID != "Iv1.abc" ||
		got.ClientSecret != "client-secret" || got.PrivateKey != "private-key" || got.WebhookSecret != "webhook-secret" {
		t.Fatalf("GenerateALMGitHubCreateOptions() unexpected options: %+v", got)
	}
}

func TestGenerateALMGitHubUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: "github-main",
			specKey:    "github-main",
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: "github-main",
			specKey:    "github-renamed",
			wantNewKey: "github-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMGitHubParameters{
				URL:      "https://api.github.com",
				Key:      tc.specKey,
				AppID:    "123456",
				ClientID: "Iv1.abc",
			}

			got := GenerateALMGitHubUpdateOptions(tc.currentKey, spec, "client-secret", "private-key", "webhook-secret")
			if got == nil {
				t.Fatal("GenerateALMGitHubUpdateOptions() returned nil")
			}

			if got.URL != "https://api.github.com" || got.Key != tc.currentKey || got.AppID != "123456" || got.ClientID != "Iv1.abc" ||
				got.ClientSecret != "client-secret" || got.PrivateKey != "private-key" || got.WebhookSecret != "webhook-secret" || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMGitHubUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

func TestFindGitHubALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		definitions *[]sonar.GithubDefinition
		key         string
		wantNil     bool
		wantKey     string
	}{
		"NilDefinitions": {
			definitions: nil,
			key:         "github-main",
			wantNil:     true,
		},
		"KeyFound": {
			definitions: &[]sonar.GithubDefinition{
				{Key: "other", URL: "https://other.com"},
				{Key: "github-main", URL: "https://api.github.com", AppID: "123", ClientID: "Iv1.abc"},
			},
			key:     "github-main",
			wantNil: false,
			wantKey: "github-main",
		},
		"KeyNotFound": {
			definitions: &[]sonar.GithubDefinition{
				{Key: "other", URL: "https://other.com"},
			},
			key:     "github-main",
			wantNil: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindGitHubALMDefinitionByKey(tc.definitions, tc.key)
			if tc.wantNil && got != nil {
				t.Fatalf("FindGitHubALMDefinitionByKey() = %+v, want nil", got)
			}

			if !tc.wantNil && got == nil {
				t.Fatal("FindGitHubALMDefinitionByKey() = nil, want non-nil")
			}

			if got != nil && got.Key != tc.wantKey {
				t.Fatalf("FindGitHubALMDefinitionByKey() key = %q, want %q", got.Key, tc.wantKey)
			}
		})
	}
}

func TestGenerateALMGitHubObservation(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinition", func(t *testing.T) {
		t.Parallel()

		got := GenerateALMGitHubObservation(nil)
		if got.Key != "" || got.URL != "" || got.AppID != "" || got.ClientID != "" {
			t.Fatalf("GenerateALMGitHubObservation(nil) = %+v, want zero value", got)
		}
	})

	t.Run("ValidDefinition", func(t *testing.T) {
		t.Parallel()

		def := &sonar.GithubDefinition{
			Key:      "github-main",
			URL:      "https://api.github.com",
			AppID:    "123456",
			ClientID: "Iv1.abc",
		}

		got := GenerateALMGitHubObservation(def)
		if got.Key != "github-main" || got.URL != "https://api.github.com" || got.AppID != "123456" || got.ClientID != "Iv1.abc" {
			t.Fatalf("GenerateALMGitHubObservation() = %+v, unexpected values", got)
		}
	})
}

func TestGitHubClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	if got := NewALMIntegrationsGitHubClient(config); got == nil {
		t.Fatal("NewALMIntegrationsGitHubClient() returned nil")
	}

	if got := NewALMSettingsGitHubClient(config); got == nil {
		t.Fatal("NewALMSettingsGitHubClient() returned nil")
	}
}
