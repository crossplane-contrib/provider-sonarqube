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
	// azureALMName is a test Azure ALM name.
	azureALMName = "azure-main"
	// azureALMURL is a test Azure ALM URL.
	azureALMURL = "https://dev.azure.com/example"
	// azurePATToken is a test Azure PAT token.
	azurePATToken = "pat-token"
	// azureMutatedURL is a test Azure mutated URL.
	azureMutatedURL = "https://mutated.example.com"
)

// TestLateInitializeALMAzure tests the LateInitializeALMAzure function.
func TestLateInitializeALMAzure(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeALMAzure(nil, nil)

		spec := &v1alpha1.ALMAzureParameters{}
		LateInitializeALMAzure(spec, nil)
		LateInitializeALMAzure(nil, &v1alpha1.ALMAzureObservation{})
	})

	t.Run("NoOpLateInit", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.ALMAzureParameters{
			ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName},
		}
		obs := &v1alpha1.ALMAzureObservation{
			ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: azureALMURL, Key: azureALMName},
		}

		LateInitializeALMAzure(spec, obs)

		if spec.URL != azureALMURL || spec.Key != azureALMName {
			t.Fatalf("LateInitializeALMAzure() mutated spec unexpectedly: %+v", spec)
		}
	})
}

// TestIsALMAzureLateInitialized tests the IsALMAzureLateInitialized function.
func TestIsALMAzureLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMAzureParameters
		current *v1alpha1.ALMAzureParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMAzureParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMAzureParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former: &v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName},
			},
			current: &v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName},
			},
			want: false,
		},
		"ChangedFields": {
			former: &v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName},
			},
			current: &v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: "https://azure-alt.example.com", Key: azureALMName},
			},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMAzureLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMAzureLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsALMAzureUpToDate tests the IsALMAzureUpToDate function.
func TestIsALMAzureUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMAzureParameters{ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName}}
	obs := &v1alpha1.ALMAzureObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: azureALMURL, Key: azureALMName}}

	cases := map[string]struct {
		spec          *v1alpha1.ALMAzureParameters
		specAPIToken  string
		observation   *v1alpha1.ALMAzureObservation
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
			observation:   &v1alpha1.ALMAzureObservation{ALMCommonObservation: v1alpha1.ALMCommonObservation{URL: azureALMURL, Key: "other"}},
			savedAPIToken: "token",
			want:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMAzureUpToDate(tc.spec, tc.specAPIToken, tc.observation, tc.savedAPIToken); got != tc.want {
				t.Fatalf("IsALMAzureUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateALMAzureCreateOptions tests GenerateALMAzureCreateOptions.
func TestGenerateALMAzureCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMAzureParameters{
		ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: azureALMName},
	}

	got := GenerateALMAzureCreateOptions(spec, "pat-token")
	if got == nil {
		t.Fatal("GenerateALMAzureCreateOptions() returned nil")
	}

	if got.URL != azureALMURL || got.Key != azureALMName || got.PersonalAccessToken != azurePATToken {
		t.Fatalf("GenerateALMAzureCreateOptions() unexpected options: %+v", got)
	}
}

// TestGenerateALMAzureUpdateOptions tests GenerateALMAzureUpdateOptions.
func TestGenerateALMAzureUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: azureALMName,
			specKey:    azureALMName,
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: azureALMName,
			specKey:    "azure-renamed",
			wantNewKey: "azure-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{URL: azureALMURL, Key: tc.specKey},
			}

			got := GenerateALMAzureUpdateOptions(tc.currentKey, spec, "pat-token")
			if got == nil {
				t.Fatal("GenerateALMAzureUpdateOptions() returned nil")
			}

			if got.URL != azureALMURL || got.Key != tc.currentKey || got.PersonalAccessToken != azurePATToken || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMAzureUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

// TestFindAzureALMDefinitionByKey tests FindAzureALMDefinitionByKey.
func TestFindAzureALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinitions", func(t *testing.T) {
		t.Parallel()

		if got := FindAzureALMDefinitionByKey(nil, azureALMName); got != nil {
			t.Fatalf("FindAzureALMDefinitionByKey(nil) = %+v, want nil", got)
		}
	})

	t.Run("EmptySlice", func(t *testing.T) {
		t.Parallel()

		if got := FindAzureALMDefinitionByKey(&[]sonar.AzureDefinition{}, azureALMName); got != nil {
			t.Fatalf("FindAzureALMDefinitionByKey(empty) = %+v, want nil", got)
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.AzureDefinition{{Key: "other", URL: "https://other.com"}}
		if got := FindAzureALMDefinitionByKey(defs, azureALMName); got != nil {
			t.Fatalf("FindAzureALMDefinitionByKey() = %+v, want nil", got)
		}
	})

	t.Run("KeyFoundReturnsPointerToSliceElement", func(t *testing.T) {
		t.Parallel()

		// The returned pointer must reference the original slice element, not a local copy.
		// Mutate via the returned pointer and verify the slice element was updated.
		defs := &[]sonar.AzureDefinition{
			{Key: "other", URL: "https://other.com"},
			{Key: azureALMName, URL: azureALMURL},
		}

		got := FindAzureALMDefinitionByKey(defs, azureALMName)
		if got == nil {
			t.Fatal("FindAzureALMDefinitionByKey() = nil, want non-nil")
		}

		if got.Key != azureALMName {
			t.Fatalf("FindAzureALMDefinitionByKey() key = %q, want %q", got.Key, azureALMName)
		}

		// Mutate via the returned pointer; the slice element must reflect the change.
		got.URL = azureMutatedURL
		if (*defs)[1].URL != "https://mutated.example.com" {
			t.Fatal("FindAzureALMDefinitionByKey() returned a copy, not a pointer to the slice element")
		}
	})
}

// TestGenerateALMAzureObservation tests GenerateALMAzureObservation.
func TestGenerateALMAzureObservation(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinition", func(t *testing.T) {
		t.Parallel()

		got := GenerateALMAzureObservation(nil)
		if got.Key != "" || got.URL != "" {
			t.Fatalf("GenerateALMAzureObservation(nil) = %+v, want zero value", got)
		}
	})

	t.Run("ValidDefinition", func(t *testing.T) {
		t.Parallel()

		def := &sonar.AzureDefinition{
			Key: azureALMName,
			URL: azureALMURL,
		}

		got := GenerateALMAzureObservation(def)
		if got.Key != azureALMName || got.URL != azureALMURL {
			t.Fatalf("GenerateALMAzureObservation() = %+v, unexpected values", got)
		}
	})
}

// TestAzureClientConstructors tests Azure client constructors.
func TestAzureClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	if got := NewALMIntegrationsAzureClient(config); got == nil {
		t.Fatal("NewALMIntegrationsAzureClient() returned nil")
	}

	if got := NewALMSettingsAzureClient(config); got == nil {
		t.Fatal("NewALMSettingsAzureClient() returned nil")
	}

	_ = &sonar.AlmIntegrationsSearchAzureReposOptions{}
}
