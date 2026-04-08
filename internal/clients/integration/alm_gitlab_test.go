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
			ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
		}
		obs := &v1alpha1.ALMGitLabObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://gitlab.example.com", Key: "gitlab-main"},
		}

		LateInitializeALMGitLab(spec, obs)

		if spec.URL != "https://gitlab.example.com" || spec.Key != "gitlab-main" {
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
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			},
			current: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			},
			want: false,
		},
		"ChangedFields": {
			former: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			},
			current: &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab-alt.example.com", Key: "gitlab-main"},
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

	spec := &v1alpha1.ALMGitLabParameters{ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"}}
	obs := &v1alpha1.ALMGitLabObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://gitlab.example.com", Key: "gitlab-main"}}

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
			observation:   &v1alpha1.ALMGitLabObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: "https://gitlab.example.com", Key: "other"}},
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

func TestGenerateALMGitLabCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMGitLabParameters{
		ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
	}

	got := GenerateALMGitLabCreateOptions(spec, "pat-token")
	if got == nil {
		t.Fatal("GenerateALMGitLabCreateOptions() returned nil")
	}

	if got.URL != "https://gitlab.example.com" || got.Key != "gitlab-main" || got.PersonalAccessToken != "pat-token" {
		t.Fatalf("GenerateALMGitLabCreateOptions() unexpected options: %+v", got)
	}
}

func TestGenerateALMGitLabUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: "gitlab-main",
			specKey:    "gitlab-main",
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: "gitlab-main",
			specKey:    "gitlab-renamed",
			wantNewKey: "gitlab-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: tc.specKey},
			}

			got := GenerateALMGitLabUpdateOptions(tc.currentKey, spec, "pat-token")
			if got == nil {
				t.Fatal("GenerateALMGitLabUpdateOptions() returned nil")
			}

			if got.URL != "https://gitlab.example.com" || got.Key != tc.currentKey || got.PersonalAccessToken != "pat-token" || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMGitLabUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

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
