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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

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

const (
	// gitlabALMName is a test GitLab ALM name.
	gitlabALMName = "gitlab-main"
	// gitlabALMURL is a test GitLab ALM URL.
	gitlabALMURL = "https://gitlab.example.com"
	// gitlabPATToken is a test GitLab PAT token.
	gitlabPATToken = "pat-token"
	// gitlabMutatedURL is a test GitLab mutated URL.
	gitlabMutatedURL = "https://mutated.example.com"
)

// TestLateInitializeALMGitLab tests the LateInitializeALMGitLab function.
func TestLateInitializeALMGitLab(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeALMGitLab(nil, nil)

		spec := &v1alpha1.ALMGitLabParameters{}
		LateInitializeALMGitLab(spec, nil)
		LateInitializeALMGitLab(nil, &v1alpha1.ALMGitLabObservation{})
	})

	t.Run("NoOpLateInit", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.ALMGitLabParameters{
			ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName},
		}
		obs := &v1alpha1.ALMGitLabObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: gitlabALMURL, Key: gitlabALMName},
		}

		LateInitializeALMGitLab(spec, obs)

		if spec.URL != gitlabALMURL || spec.Key != gitlabALMName {
			t.Fatalf("LateInitializeALMGitLab() mutated spec unexpectedly: %+v", spec)
		}
	})
}

// TestIsALMGitLabLateInitialized tests the IsALMGitLabLateInitialized function.
func TestIsALMGitLabLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMGitLabParameters
		current *v1alpha1.ALMGitLabParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMGitLabParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMGitLabParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName},
			},
			current: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName},
			},
			want: false,
		},
		"ChangedFields": {
			former: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName},
			},
			current: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab-alt.example.com", Key: gitlabALMName},
			},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMGitLabLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMGitLabLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsALMGitLabUpToDate tests the IsALMGitLabUpToDate function.
func TestIsALMGitLabUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitLabParameters{ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName}}
	obs := &v1alpha1.ALMGitLabObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: gitlabALMURL, Key: gitlabALMName}}

	cases := map[string]struct {
		spec          *v1alpha1.ALMGitLabParameters
		specAPIToken  string
		observation   *v1alpha1.ALMGitLabObservation
		savedAPIToken string
		want          bool
	}{
		"NilSpec": {
			spec:          nil,
			specAPIToken:  "token",
			observation:   obs,
			savedAPIToken: "token",
			want:          true,
		},
		"NilObservation": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   nil,
			savedAPIToken: "token",
			want:          false,
		},
		"UpToDate": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   obs,
			savedAPIToken: "token",
			want:          true,
		},
		"TokenChanged": {
			spec:          spec,
			specAPIToken:  "token-a",
			observation:   obs,
			savedAPIToken: "token-b",
			want:          false,
		},
		"KeyChanged": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   &v1alpha1.ALMGitLabObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: gitlabALMURL, Key: "other"}},
			savedAPIToken: "token",
			want:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMGitLabUpToDate(tc.spec, tc.specAPIToken, tc.observation, tc.savedAPIToken); got != tc.want {
				t.Fatalf("IsALMGitLabUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateALMGitLabCreateOptions tests GenerateALMGitLabCreateOptions.
func TestGenerateALMGitLabCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitLabParameters{
		ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: gitlabALMName},
	}

	got := GenerateALMGitLabCreateOptions(spec, "pat-token")
	if got == nil {
		t.Fatal("GenerateALMGitLabCreateOptions() returned nil")
	}

	if got.URL != gitlabALMURL || got.Key != gitlabALMName || got.PersonalAccessToken != gitlabPATToken {
		t.Fatalf("GenerateALMGitLabCreateOptions() unexpected options: %+v", got)
	}
}

// TestGenerateALMGitLabUpdateOptions tests GenerateALMGitLabUpdateOptions.
func TestGenerateALMGitLabUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: gitlabALMName,
			specKey:    gitlabALMName,
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: gitlabALMName,
			specKey:    "gitlab-renamed",
			wantNewKey: "gitlab-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: gitlabALMURL, Key: tc.specKey},
			}

			got := GenerateALMGitLabUpdateOptions(tc.currentKey, spec, "pat-token")
			if got == nil {
				t.Fatal("GenerateALMGitLabUpdateOptions() returned nil")
			}

			if got.URL != gitlabALMURL || got.Key != tc.currentKey || got.PersonalAccessToken != gitlabPATToken || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMGitLabUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

// TestFindGitLabALMDefinitionByKey tests FindGitLabALMDefinitionByKey.
func TestFindGitLabALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinitions", func(t *testing.T) {
		t.Parallel()

		if got := FindGitLabALMDefinitionByKey(nil, gitlabALMName); got != nil {
			t.Fatalf("FindGitLabALMDefinitionByKey(nil) = %+v, want nil", got)
		}
	})

	t.Run("EmptySlice", func(t *testing.T) {
		t.Parallel()

		if got := FindGitLabALMDefinitionByKey(&[]sonar.GitlabDefinition{}, gitlabALMName); got != nil {
			t.Fatalf("FindGitLabALMDefinitionByKey(empty) = %+v, want nil", got)
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.GitlabDefinition{{Key: "other", URL: "https://other.com"}}
		if got := FindGitLabALMDefinitionByKey(defs, gitlabALMName); got != nil {
			t.Fatalf("FindGitLabALMDefinitionByKey() = %+v, want nil", got)
		}
	})

	t.Run("KeyFoundReturnsPointerToSliceElement", func(t *testing.T) {
		t.Parallel()

		// The returned pointer must reference the original slice element, not a local copy.
		// Mutate via the returned pointer and verify the slice element was updated.
		defs := &[]sonar.GitlabDefinition{
			{Key: "other", URL: "https://other.com"},
			{Key: gitlabALMName, URL: gitlabALMURL},
		}

		got := FindGitLabALMDefinitionByKey(defs, gitlabALMName)
		if got == nil {
			t.Fatal("FindGitLabALMDefinitionByKey() = nil, want non-nil")
		}

		if got.Key != gitlabALMName {
			t.Fatalf("FindGitLabALMDefinitionByKey() key = %q, want %q", got.Key, gitlabALMName)
		}

		// Mutate via the returned pointer; the slice element must reflect the change.
		got.URL = gitlabMutatedURL
		if (*defs)[1].URL != "https://mutated.example.com" {
			t.Fatal("FindGitLabALMDefinitionByKey() returned a copy, not a pointer to the slice element")
		}
	})
}

// TestGenerateALMGitLabObservation tests GenerateALMGitLabObservation.
func TestGenerateALMGitLabObservation(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinition", func(t *testing.T) {
		t.Parallel()

		got := GenerateALMGitLabObservation(nil)
		if got.Key != "" || got.URL != "" {
			t.Fatalf("GenerateALMGitLabObservation(nil) = %+v, want zero value", got)
		}
	})

	t.Run("ValidDefinition", func(t *testing.T) {
		t.Parallel()

		def := &sonar.GitlabDefinition{
			Key: gitlabALMName,
			URL: gitlabALMURL,
		}

		got := GenerateALMGitLabObservation(def)
		if got.Key != gitlabALMName || got.URL != gitlabALMURL {
			t.Fatalf("GenerateALMGitLabObservation() = %+v, unexpected values", got)
		}
	})
}

// TestGitLabClientConstructors tests GitLab client constructors.
func TestGitLabClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	if got := NewALMIntegrationsGitLabClient(config); got == nil {
		t.Fatal("NewALMIntegrationsGitLabClient() returned nil")
	}

	if got := NewALMSettingsGitLabClient(config); got == nil {
		t.Fatal("NewALMSettingsGitLabClient() returned nil")
	}

	_ = &sonar.AlmIntegrationsSearchGitlabReposOptions{}
}
