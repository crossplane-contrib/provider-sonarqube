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

const (
	// githubClientID is the test GitHub client ID.
	githubClientID = "Iv1.abc"
	// githubALMName is the test GitHub ALM name.
	githubALMName = "github-main"
	// githubALMURL is the test GitHub ALM URL.
	githubALMURL = "https://api.github.com"
	// githubAppID is the test GitHub app ID.
	githubAppID = "123456"
	// clientSecret is the test client secret.
	clientSecret = "client-secret"
	// privateKey is the test private key.
	privateKey = "private-key"
	// webhookSecret is the test webhook secret.
	webhookSecret = "webhook-secret"
	// mutatedValue is a test value for mutation checking.
	mutatedValue = "mutated"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestLateInitializeALMGitHub tests late initialization of GitHub ALM.
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
			URL:      githubALMURL,
			Key:      githubALMName,
			AppID:    "123456",
			ClientID: githubClientID,
		}
		obs := &v1alpha1.ALMGitHubObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: githubALMURL, Key: githubALMName},
			AppID:                "123456",
			ClientID:             "Iv1.abc",
		}

		LateInitializeALMGitHub(spec, obs)

		if spec.URL != githubALMURL || spec.Key != githubALMName || spec.AppID != "123456" || spec.ClientID != githubClientID {
			t.Fatalf("LateInitializeALMGitHub() mutated spec unexpectedly: %+v", spec)
		}
	})
}

// TestIsALMGitHubLateInitialized tests detecting late initialization.
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
			former:  &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			want:    false,
		},
		"URLChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: "https://github.example.com", Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			want:    true,
		},
		"AppIDChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: "654321", ClientID: "Iv1.abc"},
			want:    true,
		},
		"ClientIDChanged": {
			former:  &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"},
			current: &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.xyz"},
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

// TestIsALMGitHubUpToDate tests checking GitHub ALM up-to-date status.
func TestIsALMGitHubUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitHubParameters{URL: githubALMURL, Key: githubALMName, AppID: githubAppID, ClientID: "Iv1.abc"}
	obs := &v1alpha1.ALMGitHubObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: githubALMURL, Key: githubALMName},
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
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://other.github.com", Key: githubALMName}, AppID: githubAppID, ClientID: "Iv1.abc"},
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"AppIDChanged": {
			spec:         spec,
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: githubALMURL, Key: githubALMName}, AppID: "other-app", ClientID: "Iv1.abc"},
			clientSecret: "secret", savedClientSecret: "secret",
			privateKey: "key", savedPrivateKey: "key",
			want: false,
		},
		"ClientIDChanged": {
			spec:         spec,
			observation:  &v1alpha1.ALMGitHubObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: githubALMURL, Key: githubALMName}, AppID: githubAppID, ClientID: "Iv1.xyz"},
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

// TestGenerateALMGitHubCreateOptions tests GitHub ALM creation options.
func TestGenerateALMGitHubCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitHubParameters{
		URL:      githubALMURL,
		Key:      githubALMName,
		AppID:    "123456",
		ClientID: githubClientID,
	}

	got := GenerateALMGitHubCreateOptions(spec, "client-secret", "private-key", "webhook-secret")
	if got == nil {
		t.Fatal("GenerateALMGitHubCreateOptions() returned nil")
	}

	if got.URL != githubALMURL || got.Key != githubALMName || got.AppID != "123456" || got.ClientID != githubClientID ||
		got.ClientSecret != clientSecret || got.PrivateKey != privateKey || got.WebhookSecret != webhookSecret {
		t.Fatalf("GenerateALMGitHubCreateOptions() unexpected options: %+v", got)
	}
}

// TestGenerateALMGitHubUpdateOptions tests GitHub ALM update options.
func TestGenerateALMGitHubUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: githubALMName,
			specKey:    githubALMName,
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: githubALMName,
			specKey:    "github-renamed",
			wantNewKey: "github-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMGitHubParameters{
				URL:      githubALMURL,
				Key:      tc.specKey,
				AppID:    "123456",
				ClientID: githubClientID,
			}

			got := GenerateALMGitHubUpdateOptions(tc.currentKey, spec, "client-secret", "private-key", "webhook-secret")
			if got == nil {
				t.Fatal("GenerateALMGitHubUpdateOptions() returned nil")
			}

			if got.URL != githubALMURL || got.Key != tc.currentKey || got.AppID != "123456" || got.ClientID != githubClientID ||
				got.ClientSecret != clientSecret || got.PrivateKey != privateKey || got.WebhookSecret != webhookSecret || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMGitHubUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

// TestFindGitHubALMDefinitionByKey tests finding GitHub ALM definitions by key.
func TestFindGitHubALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinitions", func(t *testing.T) {
		t.Parallel()

		if got := FindGitHubALMDefinitionByKey(nil, githubALMName); got != nil {
			t.Fatalf("FindGitHubALMDefinitionByKey(nil) = %+v, want nil", got)
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.GithubDefinition{{Key: "other", URL: "https://other.com"}}
		if got := FindGitHubALMDefinitionByKey(defs, githubALMName); got != nil {
			t.Fatalf("FindGitHubALMDefinitionByKey() = %+v, want nil", got)
		}
	})

	t.Run("KeyFoundReturnsPointerToSliceElement", func(t *testing.T) {
		t.Parallel()

		// The returned pointer must reference the original slice element, not a local copy.
		// Mutate via the returned pointer and verify the slice element was updated.
		defs := &[]sonar.GithubDefinition{
			{Key: "other", URL: "https://other.com"},
			{Key: githubALMName, URL: githubALMURL, AppID: "123", ClientID: "Iv1.abc"},
		}

		got := FindGitHubALMDefinitionByKey(defs, githubALMName)
		if got == nil {
			t.Fatal("FindGitHubALMDefinitionByKey() = nil, want non-nil")
		}

		if got.Key != githubALMName {
			t.Fatalf("FindGitHubALMDefinitionByKey() key = %q, want %q", got.Key, githubALMName)
		}

		// Mutate via the returned pointer; the slice element must reflect the change.
		got.AppID = mutatedValue
		if (*defs)[1].AppID != "mutated" {
			t.Fatal("FindGitHubALMDefinitionByKey() returned a copy, not a pointer to the slice element")
		}
	})
}

// TestGenerateALMGitHubObservation tests generating GitHub ALM observations.
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
			Key:      githubALMName,
			URL:      githubALMURL,
			AppID:    "123456",
			ClientID: githubClientID,
		}

		got := GenerateALMGitHubObservation(def)
		if got.Key != githubALMName || got.URL != githubALMURL || got.AppID != "123456" || got.ClientID != githubClientID {
			t.Fatalf("GenerateALMGitHubObservation() = %+v, unexpected values", got)
		}
	})
}

// TestGitHubClientConstructors tests creating GitHub client instances.
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
