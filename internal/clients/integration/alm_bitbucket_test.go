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

const (
	// bitbucketALMName is a test Bitbucket ALM name.
	bitbucketALMName = "bitbucket-main"
	// bitbucketALMURL is a test Bitbucket ALM URL.
	bitbucketALMURL = "https://bitbucket.example.com"
	// bitbucketPATToken is a test Bitbucket personal access token.
	bitbucketPATToken = "pat-token"
	// bitbucketMutatedURL is a test mutated Bitbucket ALM URL.
	bitbucketMutatedURL = "https://mutated.example.com"
)

// TestLateInitializeALMBitbucket tests LateInitializeALMBitbucket.
func TestLateInitializeALMBitbucket(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeALMBitbucket(nil, nil)

		spec := &v1alpha1.ALMBitbucketParameters{}
		LateInitializeALMBitbucket(spec, nil)
		LateInitializeALMBitbucket(nil, &v1alpha1.ALMBitbucketObservation{})
	})

	t.Run("NoOpLateInit", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.ALMBitbucketParameters{
			ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName},
		}
		obs := &v1alpha1.ALMBitbucketObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: bitbucketALMURL, Key: bitbucketALMName},
		}

		LateInitializeALMBitbucket(spec, obs)

		if spec.URL != bitbucketALMURL || spec.Key != bitbucketALMName {
			t.Fatalf("LateInitializeALMBitbucket() mutated spec unexpectedly: %+v", spec)
		}
	})
}

// TestIsALMBitbucketLateInitialized tests IsALMBitbucketLateInitialized.
func TestIsALMBitbucketLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMBitbucketParameters
		current *v1alpha1.ALMBitbucketParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMBitbucketParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMBitbucketParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former: &v1alpha1.ALMBitbucketParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName},
			},
			current: &v1alpha1.ALMBitbucketParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName},
			},
			want: false,
		},
		"ChangedFields": {
			former: &v1alpha1.ALMBitbucketParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName},
			},
			current: &v1alpha1.ALMBitbucketParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://bitbucket-alt.example.com", Key: bitbucketALMName},
			},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMBitbucketLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMBitbucketLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsALMBitbucketUpToDate tests IsALMBitbucketUpToDate.
func TestIsALMBitbucketUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMBitbucketParameters{ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName}}
	obs := &v1alpha1.ALMBitbucketObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: bitbucketALMURL, Key: bitbucketALMName}}

	cases := map[string]struct {
		spec          *v1alpha1.ALMBitbucketParameters
		specAPIToken  string
		observation   *v1alpha1.ALMBitbucketObservation
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
			observation:   &v1alpha1.ALMBitbucketObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: bitbucketALMURL, Key: "other"}},
			savedAPIToken: "token",
			want:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMBitbucketUpToDate(tc.spec, tc.specAPIToken, tc.observation, tc.savedAPIToken); got != tc.want {
				t.Fatalf("IsALMBitbucketUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateALMBitbucketCreateOptions tests create option generation.
func TestGenerateALMBitbucketCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMBitbucketParameters{
		ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: bitbucketALMName},
	}

	got := GenerateALMBitbucketCreateOptions(spec, "pat-token")
	if got == nil {
		t.Fatal("GenerateALMBitbucketCreateOptions() returned nil")
	}

	if got.URL != bitbucketALMURL || got.Key != bitbucketALMName || got.PersonalAccessToken != bitbucketPATToken {
		t.Fatalf("GenerateALMBitbucketCreateOptions() unexpected options: %+v", got)
	}
}

// TestGenerateALMBitbucketUpdateOptions tests update option generation.
func TestGenerateALMBitbucketUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: bitbucketALMName,
			specKey:    bitbucketALMName,
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: bitbucketALMName,
			specKey:    "bitbucket-renamed",
			wantNewKey: "bitbucket-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMBitbucketParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: bitbucketALMURL, Key: tc.specKey},
			}

			got := GenerateALMBitbucketUpdateOptions(tc.currentKey, spec, "pat-token")
			if got == nil {
				t.Fatal("GenerateALMBitbucketUpdateOptions() returned nil")
			}

			if got.URL != bitbucketALMURL || got.Key != tc.currentKey || got.PersonalAccessToken != bitbucketPATToken || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMBitbucketUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

// TestFindBitbucketALMDefinitionByKey tests the definition lookup.
func TestFindBitbucketALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinitions", func(t *testing.T) {
		t.Parallel()

		if got := FindBitbucketALMDefinitionByKey(nil, bitbucketALMName); got != nil {
			t.Fatalf("FindBitbucketALMDefinitionByKey(nil) = %+v, want nil", got)
		}
	})

	t.Run("EmptySlice", func(t *testing.T) {
		t.Parallel()

		if got := FindBitbucketALMDefinitionByKey(&[]sonar.BitbucketDefinition{}, bitbucketALMName); got != nil {
			t.Fatalf("FindBitbucketALMDefinitionByKey(empty) = %+v, want nil", got)
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.BitbucketDefinition{{Key: "other", URL: "https://other.com"}}
		if got := FindBitbucketALMDefinitionByKey(defs, bitbucketALMName); got != nil {
			t.Fatalf("FindBitbucketALMDefinitionByKey() = %+v, want nil", got)
		}
	})

	t.Run("KeyFoundReturnsPointerToSliceElement", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.BitbucketDefinition{
			{Key: "other", URL: "https://other.com"},
			{Key: bitbucketALMName, URL: bitbucketALMURL},
		}

		got := FindBitbucketALMDefinitionByKey(defs, bitbucketALMName)
		if got == nil {
			t.Fatal("FindBitbucketALMDefinitionByKey() = nil, want non-nil")
		}

		if got.Key != bitbucketALMName {
			t.Fatalf("FindBitbucketALMDefinitionByKey() key = %q, want %q", got.Key, bitbucketALMName)
		}

		got.URL = bitbucketMutatedURL
		if (*defs)[1].URL != "https://mutated.example.com" {
			t.Fatal("FindBitbucketALMDefinitionByKey() returned a copy, not a pointer to the slice element")
		}
	})
}

// TestGenerateALMBitbucketObservation tests observation generation.
func TestGenerateALMBitbucketObservation(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinition", func(t *testing.T) {
		t.Parallel()

		got := GenerateALMBitbucketObservation(nil)
		if got.Key != "" || got.URL != "" {
			t.Fatalf("GenerateALMBitbucketObservation(nil) = %+v, want zero value", got)
		}
	})

	t.Run("ValidDefinition", func(t *testing.T) {
		t.Parallel()

		def := &sonar.BitbucketDefinition{
			Key: bitbucketALMName,
			URL: bitbucketALMURL,
		}

		got := GenerateALMBitbucketObservation(def)
		if got.Key != bitbucketALMName || got.URL != bitbucketALMURL {
			t.Fatalf("GenerateALMBitbucketObservation() = %+v, unexpected values", got)
		}
	})
}

// TestBitbucketClientConstructors tests the Bitbucket client constructors.
func TestBitbucketClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	if got := NewALMIntegrationsBitbucketClient(config); got == nil {
		t.Fatal("NewALMIntegrationsBitbucketClient() returned nil")
	}

	if got := NewALMSettingsBitbucketClient(config); got == nil {
		t.Fatal("NewALMSettingsBitbucketClient() returned nil")
	}

	_ = &sonar.AlmIntegrationsSearchBitbucketServerReposOptions{}
}
