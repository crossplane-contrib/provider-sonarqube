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
	// bitbucketCloudClientID is a test Bitbucket Cloud OAuth client ID.
	bitbucketCloudClientID = "bitbucket-client-id"
	// bitbucketCloudALMName is a test Bitbucket Cloud ALM name.
	bitbucketCloudALMName = "bitbucketcloud-main"
	// bitbucketCloudWorkspace is a test Bitbucket Cloud workspace.
	bitbucketCloudWorkspace = "my-workspace"
	// bitbucketCloudMutated is a test mutated value.
	bitbucketCloudMutated = "mutated"
)

// TestLateInitializeALMBitbucketCloud tests LateInitializeALMBitbucketCloud.
func TestLateInitializeALMBitbucketCloud(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeALMBitbucketCloud(nil, nil)

		spec := &v1alpha1.ALMBitbucketCloudParameters{}
		LateInitializeALMBitbucketCloud(spec, nil)
		LateInitializeALMBitbucketCloud(nil, &v1alpha1.ALMBitbucketCloudObservation{})
	})

	t.Run("NoOpLateInit", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.ALMBitbucketCloudParameters{
			Key:       bitbucketCloudALMName,
			ClientID:  bitbucketCloudClientID,
			Workspace: bitbucketCloudWorkspace,
		}
		obs := &v1alpha1.ALMBitbucketCloudObservation{
			Key:       bitbucketCloudALMName,
			ClientID:  bitbucketCloudClientID,
			Workspace: bitbucketCloudWorkspace,
		}

		LateInitializeALMBitbucketCloud(spec, obs)

		if spec.Key != bitbucketCloudALMName || spec.ClientID != bitbucketCloudClientID || spec.Workspace != bitbucketCloudWorkspace {
			t.Fatalf("LateInitializeALMBitbucketCloud() mutated spec unexpectedly: %+v", spec)
		}
	})
}

// TestIsALMBitbucketCloudLateInitialized tests the late-init check.
func TestIsALMBitbucketCloudLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMBitbucketCloudParameters
		current *v1alpha1.ALMBitbucketCloudParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMBitbucketCloudParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMBitbucketCloudParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former:  &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			current: &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			want:    false,
		},
		"KeyChanged": {
			former:  &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			current: &v1alpha1.ALMBitbucketCloudParameters{Key: "other", ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			want:    true,
		},
		"ClientIDChanged": {
			former:  &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			current: &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: "other-client", Workspace: bitbucketCloudWorkspace},
			want:    true,
		},
		"WorkspaceChanged": {
			former:  &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			current: &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: "other-workspace"},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMBitbucketCloudLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMBitbucketCloudLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsALMBitbucketCloudUpToDate tests the up-to-date check.
func TestIsALMBitbucketCloudUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMBitbucketCloudParameters{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace}
	obs := &v1alpha1.ALMBitbucketCloudObservation{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace}

	cases := map[string]struct {
		spec              *v1alpha1.ALMBitbucketCloudParameters
		observation       *v1alpha1.ALMBitbucketCloudObservation
		clientSecret      string
		savedClientSecret string
		want              bool
	}{
		"NilSpec": {
			spec:              nil,
			observation:       obs,
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              true,
		},
		"NilObservation": {
			spec:              spec,
			observation:       nil,
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              false,
		},
		"UpToDate": {
			spec:              spec,
			observation:       obs,
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              true,
		},
		"ClientSecretChanged": {
			spec:              spec,
			observation:       obs,
			clientSecret:      "new-secret",
			savedClientSecret: "old-secret",
			want:              false,
		},
		"KeyChanged": {
			spec:              spec,
			observation:       &v1alpha1.ALMBitbucketCloudObservation{Key: "other", ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              false,
		},
		"ClientIDChanged": {
			spec:              spec,
			observation:       &v1alpha1.ALMBitbucketCloudObservation{Key: bitbucketCloudALMName, ClientID: "other-client", Workspace: bitbucketCloudWorkspace},
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              false,
		},
		"WorkspaceChanged": {
			spec:              spec,
			observation:       &v1alpha1.ALMBitbucketCloudObservation{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: "other-workspace"},
			clientSecret:      "secret",
			savedClientSecret: "secret",
			want:              false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsALMBitbucketCloudUpToDate(tc.spec, tc.observation, tc.clientSecret, tc.savedClientSecret)
			if got != tc.want {
				t.Fatalf("IsALMBitbucketCloudUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateALMBitbucketCloudCreateOptions tests create option generation.
func TestGenerateALMBitbucketCloudCreateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMBitbucketCloudParameters{
		Key:       bitbucketCloudALMName,
		ClientID:  bitbucketCloudClientID,
		Workspace: bitbucketCloudWorkspace,
	}

	got := GenerateALMBitbucketCloudCreateOptions(spec, "client-secret")
	if got == nil {
		t.Fatal("GenerateALMBitbucketCloudCreateOptions() returned nil")
	}

	if got.Key != bitbucketCloudALMName || got.ClientID != bitbucketCloudClientID || got.Workspace != bitbucketCloudWorkspace || got.ClientSecret != "client-secret" {
		t.Fatalf("GenerateALMBitbucketCloudCreateOptions() unexpected options: %+v", got)
	}
}

// TestGenerateALMBitbucketCloudUpdateOptions tests update option generation.
func TestGenerateALMBitbucketCloudUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		currentKey string
		specKey    string
		wantNewKey string
	}{
		"KeyUnchanged": {
			currentKey: bitbucketCloudALMName,
			specKey:    bitbucketCloudALMName,
			wantNewKey: "",
		},
		"KeyChanged": {
			currentKey: bitbucketCloudALMName,
			specKey:    "bitbucketcloud-renamed",
			wantNewKey: "bitbucketcloud-renamed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := &v1alpha1.ALMBitbucketCloudParameters{
				Key:       tc.specKey,
				ClientID:  bitbucketCloudClientID,
				Workspace: bitbucketCloudWorkspace,
			}

			got := GenerateALMBitbucketCloudUpdateOptions(tc.currentKey, spec, "client-secret")
			if got == nil {
				t.Fatal("GenerateALMBitbucketCloudUpdateOptions() returned nil")
			}

			if got.Key != tc.currentKey || got.ClientID != bitbucketCloudClientID || got.Workspace != bitbucketCloudWorkspace ||
				got.ClientSecret != "client-secret" || got.NewKey != tc.wantNewKey {
				t.Fatalf("GenerateALMBitbucketCloudUpdateOptions() unexpected options: %+v", got)
			}
		})
	}
}

// TestFindBitbucketCloudALMDefinitionByKey tests the definition lookup.
func TestFindBitbucketCloudALMDefinitionByKey(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinitions", func(t *testing.T) {
		t.Parallel()

		if got := FindBitbucketCloudALMDefinitionByKey(nil, bitbucketCloudALMName); got != nil {
			t.Fatalf("FindBitbucketCloudALMDefinitionByKey(nil) = %+v, want nil", got)
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.BitbucketCloudDefinition{{Key: "other", ClientID: "other-client", Workspace: "other-workspace"}}
		if got := FindBitbucketCloudALMDefinitionByKey(defs, bitbucketCloudALMName); got != nil {
			t.Fatalf("FindBitbucketCloudALMDefinitionByKey() = %+v, want nil", got)
		}
	})

	t.Run("KeyFoundReturnsPointerToSliceElement", func(t *testing.T) {
		t.Parallel()

		defs := &[]sonar.BitbucketCloudDefinition{
			{Key: "other", ClientID: "other-client", Workspace: "other-workspace"},
			{Key: bitbucketCloudALMName, ClientID: bitbucketCloudClientID, Workspace: bitbucketCloudWorkspace},
		}

		got := FindBitbucketCloudALMDefinitionByKey(defs, bitbucketCloudALMName)
		if got == nil {
			t.Fatal("FindBitbucketCloudALMDefinitionByKey() = nil, want non-nil")
		}

		if got.Key != bitbucketCloudALMName {
			t.Fatalf("FindBitbucketCloudALMDefinitionByKey() key = %q, want %q", got.Key, bitbucketCloudALMName)
		}

		got.ClientID = bitbucketCloudMutated
		if (*defs)[1].ClientID != "mutated" {
			t.Fatal("FindBitbucketCloudALMDefinitionByKey() returned a copy, not a pointer to the slice element")
		}
	})
}

// TestGenerateALMBitbucketCloudObservation tests observation generation.
func TestGenerateALMBitbucketCloudObservation(t *testing.T) {
	t.Parallel()

	t.Run("NilDefinition", func(t *testing.T) {
		t.Parallel()

		got := GenerateALMBitbucketCloudObservation(nil)
		if got.Key != "" || got.ClientID != "" || got.Workspace != "" {
			t.Fatalf("GenerateALMBitbucketCloudObservation(nil) = %+v, want zero value", got)
		}
	})

	t.Run("ValidDefinition", func(t *testing.T) {
		t.Parallel()

		def := &sonar.BitbucketCloudDefinition{
			Key:       bitbucketCloudALMName,
			ClientID:  bitbucketCloudClientID,
			Workspace: bitbucketCloudWorkspace,
		}

		got := GenerateALMBitbucketCloudObservation(def)
		if got.Key != bitbucketCloudALMName || got.ClientID != bitbucketCloudClientID || got.Workspace != bitbucketCloudWorkspace {
			t.Fatalf("GenerateALMBitbucketCloudObservation() = %+v, unexpected values", got)
		}
	})
}

// TestBitbucketCloudClientConstructors tests the client constructors.
func TestBitbucketCloudClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	if got := NewALMIntegrationsBitbucketCloudClient(config); got == nil {
		t.Fatal("NewALMIntegrationsBitbucketCloudClient() returned nil")
	}

	if got := NewALMSettingsBitbucketCloudClient(config); got == nil {
		t.Fatal("NewALMSettingsBitbucketCloudClient() returned nil")
	}
}
