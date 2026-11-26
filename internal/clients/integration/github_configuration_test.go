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
	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// githubConfigurationID is the test GitHub configuration ID.
	githubConfigurationID = "github-configuration"
	// githubConfigurationAPIURL is the test GitHub API URL.
	githubConfigurationAPIURL = "https://api.github.com/"
	// githubConfigurationWebURL is the test GitHub web URL.
	githubConfigurationWebURL = "https://github.com/"
)

// testGitHubConfigurationCredentials returns the test GitHub App
// credentials.
func testGitHubConfigurationCredentials() GitHubConfigurationCredentials {
	return GitHubConfigurationCredentials{ClientID: githubClientID, ClientSecret: clientSecret, PrivateKey: privateKey}
}

// testGitHubConfigurationSpec returns a fully set GitHubConfiguration spec.
func testGitHubConfigurationSpec() *v1alpha1.GitHubConfigurationParameters {
	return &v1alpha1.GitHubConfigurationParameters{
		Enabled:                         new(true),
		ApplicationID:                   githubAppID,
		ClientID:                        githubClientID,
		APIURL:                          githubConfigurationAPIURL,
		WebURL:                          githubConfigurationWebURL,
		AllowedOrganizations:            []string{"org-a", "org-b"},
		ProvisioningType:                v1alpha1.GitHubProvisioningTypeJIT,
		SynchronizeGroups:               new(true),
		AllowUsersToSignUp:              new(false),
		ProjectVisibility:               new(true),
		UserConsentRequiredAfterUpgrade: new(false),
	}
}

// testGitHubConfigurationObservation returns the observation matching
// testGitHubConfigurationSpec.
func testGitHubConfigurationObservation() *v1alpha1.GitHubConfigurationObservation {
	return &v1alpha1.GitHubConfigurationObservation{
		ID:                              githubConfigurationID,
		Enabled:                         true,
		ApplicationID:                   githubAppID,
		APIURL:                          githubConfigurationAPIURL,
		WebURL:                          githubConfigurationWebURL,
		AllowedOrganizations:            []string{"org-b", "org-a"},
		ProvisioningType:                v1alpha1.GitHubProvisioningTypeJIT,
		SynchronizeGroups:               true,
		AllowUsersToSignUp:              false,
		ProjectVisibility:               true,
		UserConsentRequiredAfterUpgrade: false,
	}
}

// TestNewGitHubConfigurationsClient tests the client constructor.
func TestNewGitHubConfigurationsClient(t *testing.T) {
	t.Parallel()

	got := NewGitHubConfigurationsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})
	if got == nil {
		t.Fatal("NewGitHubConfigurationsClient() returned nil")
	}
}

// TestFindGitHubConfiguration tests finding the GitHub configuration.
func TestFindGitHubConfiguration(t *testing.T) {
	t.Parallel()

	configurations := []sonar.DopTranslationGithubConfiguration{{Id: "first"}, {Id: githubConfigurationID}}

	cases := map[string]struct {
		configurations []sonar.DopTranslationGithubConfiguration
		id             string
		wantID         string
	}{
		"NoConfigurations": {configurations: nil, id: githubConfigurationID},
		"EmptyIDReturnsFirst": {
			configurations: configurations,
			wantID:         "first",
		},
		"MatchingID": {
			configurations: configurations,
			id:             githubConfigurationID,
			wantID:         githubConfigurationID,
		},
		"UnknownID": {
			configurations: configurations,
			id:             "unknown",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindGitHubConfiguration(tc.configurations, tc.id)

			switch {
			case tc.wantID == "" && got != nil:
				t.Fatalf("FindGitHubConfiguration() = %+v, want nil", got)
			case tc.wantID != "" && got == nil:
				t.Fatalf("FindGitHubConfiguration() = nil, want ID %q", tc.wantID)
			case tc.wantID != "" && got.Id != tc.wantID:
				t.Fatalf("FindGitHubConfiguration() ID = %q, want %q", got.Id, tc.wantID)
			}
		})
	}
}

// TestGenerateGitHubConfigurationObservation tests generating the
// observation.
func TestGenerateGitHubConfigurationObservation(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()

		got := GenerateGitHubConfigurationObservation(nil)
		if diff := cmp.Diff(v1alpha1.GitHubConfigurationObservation{}, got); diff != "" {
			t.Fatalf("GenerateGitHubConfigurationObservation(nil) -want, +got:\n%s", diff)
		}
	})

	t.Run("AllFields", func(t *testing.T) {
		t.Parallel()

		got := GenerateGitHubConfigurationObservation(&sonar.DopTranslationGithubConfiguration{
			Id:                              githubConfigurationID,
			Enabled:                         true,
			ApplicationId:                   githubAppID,
			SynchronizeGroups:               true,
			ApiUrl:                          githubConfigurationAPIURL,
			WebUrl:                          githubConfigurationWebURL,
			AllowedOrganizations:            []string{"org-a"},
			ProvisioningType:                v1alpha1.GitHubProvisioningTypeAuto,
			AllowUsersToSignUp:              true,
			ProjectVisibility:               true,
			UserConsentRequiredAfterUpgrade: true,
			ErrorMessage:                    new("invalid private key"),
		})

		want := v1alpha1.GitHubConfigurationObservation{
			ID:                              githubConfigurationID,
			Enabled:                         true,
			ApplicationID:                   githubAppID,
			APIURL:                          githubConfigurationAPIURL,
			WebURL:                          githubConfigurationWebURL,
			AllowedOrganizations:            []string{"org-a"},
			ProvisioningType:                v1alpha1.GitHubProvisioningTypeAuto,
			SynchronizeGroups:               true,
			AllowUsersToSignUp:              true,
			ProjectVisibility:               true,
			UserConsentRequiredAfterUpgrade: true,
			ErrorMessage:                    "invalid private key",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("GenerateGitHubConfigurationObservation() -want, +got:\n%s", diff)
		}
	})
}

// TestLateInitializeGitHubConfiguration tests late initialization.
func TestLateInitializeGitHubConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("NilInputs", func(t *testing.T) {
		t.Parallel()

		LateInitializeGitHubConfiguration(nil, nil)
		LateInitializeGitHubConfiguration(&v1alpha1.GitHubConfigurationParameters{}, nil)
		LateInitializeGitHubConfiguration(nil, &v1alpha1.GitHubConfigurationObservation{})
	})

	t.Run("FillsUnsetFields", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GitHubConfigurationParameters{ApplicationID: githubAppID}
		observation := testGitHubConfigurationObservation()

		LateInitializeGitHubConfiguration(spec, observation)

		want := &v1alpha1.GitHubConfigurationParameters{
			Enabled:              new(true),
			ApplicationID:        githubAppID,
			AllowedOrganizations: []string{"org-b", "org-a"},
			ProvisioningType:     v1alpha1.GitHubProvisioningTypeJIT,
			SynchronizeGroups:    new(true),
			AllowUsersToSignUp:   new(false),
			ProjectVisibility:    new(true),
		}
		if diff := cmp.Diff(want, spec); diff != "" {
			t.Fatalf("LateInitializeGitHubConfiguration() -want, +got:\n%s", diff)
		}

		observation.AllowedOrganizations[0] = mutatedValue
		if spec.AllowedOrganizations[0] == mutatedValue {
			t.Fatal("LateInitializeGitHubConfiguration() shares the observed organizations slice")
		}
	})

	t.Run("KeepsSetFields", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GitHubConfigurationParameters{
			Enabled:                         new(false),
			AllowedOrganizations:            []string{},
			ProvisioningType:                v1alpha1.GitHubProvisioningTypeAuto,
			SynchronizeGroups:               new(false),
			AllowUsersToSignUp:              new(true),
			ProjectVisibility:               new(false),
			UserConsentRequiredAfterUpgrade: new(true),
		}
		want := spec.DeepCopy()

		LateInitializeGitHubConfiguration(spec, testGitHubConfigurationObservation())

		if diff := cmp.Diff(want, spec); diff != "" {
			t.Fatalf("LateInitializeGitHubConfiguration() -want, +got:\n%s", diff)
		}
	})

	t.Run("NoObservedOrganizations", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GitHubConfigurationParameters{}
		LateInitializeGitHubConfiguration(spec, &v1alpha1.GitHubConfigurationObservation{})

		if spec.AllowedOrganizations != nil {
			t.Fatalf("LateInitializeGitHubConfiguration() AllowedOrganizations = %v, want nil", spec.AllowedOrganizations)
		}
	})
}

// TestIsGitHubConfigurationLateInitialized tests detecting late
// initialization.
func TestIsGitHubConfigurationLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.GitHubConfigurationParameters
		current *v1alpha1.GitHubConfigurationParameters
		want    bool
	}{
		"NilFormer":  {former: nil, current: testGitHubConfigurationSpec(), want: false},
		"NilCurrent": {former: testGitHubConfigurationSpec(), current: nil, want: false},
		"NoChanges":  {former: testGitHubConfigurationSpec(), current: testGitHubConfigurationSpec(), want: false},
		"EnabledInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{Enabled: new(true)},
			want:    true,
		},
		"SynchronizeGroupsInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{SynchronizeGroups: new(false)},
			want:    true,
		},
		"AllowUsersToSignUpInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{AllowUsersToSignUp: new(false)},
			want:    true,
		},
		"ProjectVisibilityInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{ProjectVisibility: new(false)},
			want:    true,
		},
		"ProvisioningTypeInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{ProvisioningType: v1alpha1.GitHubProvisioningTypeJIT},
			want:    true,
		},
		"AllowedOrganizationsInitialized": {
			former:  &v1alpha1.GitHubConfigurationParameters{},
			current: &v1alpha1.GitHubConfigurationParameters{AllowedOrganizations: []string{"org-a"}},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGitHubConfigurationLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsGitHubConfigurationLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsGitHubConfigurationUpToDate tests the up-to-date check.
func TestIsGitHubConfigurationUpToDate(t *testing.T) {
	t.Parallel()

	withSpec := func(mutate func(*v1alpha1.GitHubConfigurationParameters)) *v1alpha1.GitHubConfigurationParameters {
		spec := testGitHubConfigurationSpec()
		mutate(spec)

		return spec
	}

	cases := map[string]struct {
		spec        *v1alpha1.GitHubConfigurationParameters
		observation *v1alpha1.GitHubConfigurationObservation
		saved       GitHubConfigurationCredentials
		want        bool
	}{
		"NilSpec": {
			spec: nil, observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: true,
		},
		"NilObservation": {
			spec: testGitHubConfigurationSpec(), observation: nil, saved: testGitHubConfigurationCredentials(), want: false,
		},
		"UpToDate": {
			spec: testGitHubConfigurationSpec(), observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: true,
		},
		"UpToDateIgnoringTrailingSlash": {
			spec: withSpec(func(s *v1alpha1.GitHubConfigurationParameters) {
				s.APIURL = "https://api.github.com"
				s.WebURL = "https://github.com"
			}),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: true,
		},
		"UnsetOptionalFieldsUpToDate": {
			spec: &v1alpha1.GitHubConfigurationParameters{
				ApplicationID: githubAppID, ClientID: githubClientID, APIURL: githubConfigurationAPIURL, WebURL: githubConfigurationWebURL,
			},
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: true,
		},
		"EnabledDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.Enabled = new(false) }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"ApplicationIDDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.ApplicationID = "654321" }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"APIURLDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.APIURL = "https://github.example.com/api/v3/" }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"WebURLDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.WebURL = "https://github.example.com/" }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"AllowedOrganizationsDiffer": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.AllowedOrganizations = []string{"org-a"} }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"AllowedOrganizationsCleared": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.AllowedOrganizations = []string{} }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"ProvisioningTypeDiffers": {
			spec: withSpec(func(s *v1alpha1.GitHubConfigurationParameters) {
				s.ProvisioningType = v1alpha1.GitHubProvisioningTypeAuto
			}),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"SynchronizeGroupsDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.SynchronizeGroups = new(false) }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"AllowUsersToSignUpDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.AllowUsersToSignUp = new(true) }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"ProjectVisibilityDiffers": {
			spec:        withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.ProjectVisibility = new(false) }),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: false,
		},
		"UserConsentRequired": {
			spec: testGitHubConfigurationSpec(),
			observation: func() *v1alpha1.GitHubConfigurationObservation {
				o := testGitHubConfigurationObservation()
				o.UserConsentRequiredAfterUpgrade = true

				return o
			}(),
			saved: testGitHubConfigurationCredentials(), want: false,
		},
		"UserConsentRequiredNotGiven": {
			spec: withSpec(func(s *v1alpha1.GitHubConfigurationParameters) { s.UserConsentRequiredAfterUpgrade = nil }),
			observation: func() *v1alpha1.GitHubConfigurationObservation {
				o := testGitHubConfigurationObservation()
				o.UserConsentRequiredAfterUpgrade = true

				return o
			}(),
			saved: testGitHubConfigurationCredentials(), want: true,
		},
		"UserConsentTrueCannotBeForced": {
			spec: withSpec(func(s *v1alpha1.GitHubConfigurationParameters) {
				s.UserConsentRequiredAfterUpgrade = new(true)
			}),
			observation: testGitHubConfigurationObservation(), saved: testGitHubConfigurationCredentials(), want: true,
		},
		"ClientIDChanged": {
			spec: testGitHubConfigurationSpec(), observation: testGitHubConfigurationObservation(),
			saved: GitHubConfigurationCredentials{ClientID: "Iv1.old", ClientSecret: clientSecret, PrivateKey: privateKey},
			want:  false,
		},
		"ClientSecretRotated": {
			spec: testGitHubConfigurationSpec(), observation: testGitHubConfigurationObservation(),
			saved: GitHubConfigurationCredentials{ClientID: githubClientID, ClientSecret: "old-secret", PrivateKey: privateKey},
			want:  false,
		},
		"PrivateKeyRotated": {
			spec: testGitHubConfigurationSpec(), observation: testGitHubConfigurationObservation(),
			saved: GitHubConfigurationCredentials{ClientID: githubClientID, ClientSecret: clientSecret, PrivateKey: "old-key"},
			want:  false,
		},
		"NothingSaved": {
			spec: testGitHubConfigurationSpec(), observation: testGitHubConfigurationObservation(),
			saved: GitHubConfigurationCredentials{}, want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsGitHubConfigurationUpToDate(tc.spec, tc.observation, testGitHubConfigurationCredentials(), tc.saved)
			if got != tc.want {
				t.Fatalf("IsGitHubConfigurationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateGitHubConfigurationCreateOptions tests generating the create
// options.
func TestGenerateGitHubConfigurationCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.GitHubConfigurationParameters
		want *sonar.DopTranslationGithubConfigurationCreateOptions
	}{
		"NilSpec": {spec: nil, want: nil},
		"Defaults": {
			spec: &v1alpha1.GitHubConfigurationParameters{
				ApplicationID: githubAppID, ClientID: githubClientID, APIURL: githubConfigurationAPIURL, WebURL: githubConfigurationWebURL,
			},
			want: &sonar.DopTranslationGithubConfigurationCreateOptions{
				Enabled:              true,
				ClientId:             githubClientID,
				ClientSecret:         clientSecret,
				ApplicationId:        githubAppID,
				PrivateKey:           privateKey,
				SynchronizeGroups:    false,
				ApiUrl:               githubConfigurationAPIURL,
				WebUrl:               githubConfigurationWebURL,
				AllowedOrganizations: []string{},
				ProvisioningType:     v1alpha1.GitHubProvisioningTypeJIT,
			},
		},
		"AllFields": {
			spec: testGitHubConfigurationSpec(),
			want: &sonar.DopTranslationGithubConfigurationCreateOptions{
				Enabled:                         true,
				ClientId:                        githubClientID,
				ClientSecret:                    clientSecret,
				ApplicationId:                   githubAppID,
				PrivateKey:                      privateKey,
				SynchronizeGroups:               true,
				ApiUrl:                          githubConfigurationAPIURL,
				WebUrl:                          githubConfigurationWebURL,
				AllowedOrganizations:            []string{"org-a", "org-b"},
				ProvisioningType:                v1alpha1.GitHubProvisioningTypeJIT,
				AllowUsersToSignUp:              new(false),
				ProjectVisibility:               new(true),
				UserConsentRequiredAfterUpgrade: new(false),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGitHubConfigurationCreateOptions(tc.spec, testGitHubConfigurationCredentials())
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GenerateGitHubConfigurationCreateOptions() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateGitHubConfigurationUpdateOptions tests generating the update
// options.
func TestGenerateGitHubConfigurationUpdateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.GitHubConfigurationParameters
		want *sonar.DopTranslationGithubConfigurationUpdateOptions
	}{
		"NilSpec": {spec: nil, want: nil},
		"UnsetOptionalFieldsNotSent": {
			spec: &v1alpha1.GitHubConfigurationParameters{
				ApplicationID: githubAppID, ClientID: githubClientID, APIURL: githubConfigurationAPIURL, WebURL: githubConfigurationWebURL,
			},
			want: &sonar.DopTranslationGithubConfigurationUpdateOptions{
				ClientId:      githubClientID,
				ClientSecret:  clientSecret,
				ApplicationId: githubAppID,
				PrivateKey:    privateKey,
				ApiUrl:        githubConfigurationAPIURL,
				WebUrl:        githubConfigurationWebURL,
			},
		},
		"UserConsentRequiredNotSent": {
			spec: &v1alpha1.GitHubConfigurationParameters{
				ApplicationID: githubAppID, ClientID: githubClientID, APIURL: githubConfigurationAPIURL, WebURL: githubConfigurationWebURL,
				UserConsentRequiredAfterUpgrade: new(true),
			},
			want: &sonar.DopTranslationGithubConfigurationUpdateOptions{
				ClientId:      githubClientID,
				ClientSecret:  clientSecret,
				ApplicationId: githubAppID,
				PrivateKey:    privateKey,
				ApiUrl:        githubConfigurationAPIURL,
				WebUrl:        githubConfigurationWebURL,
			},
		},
		"ClearedOrganizationsSent": {
			spec: &v1alpha1.GitHubConfigurationParameters{
				ApplicationID: githubAppID, ClientID: githubClientID, APIURL: githubConfigurationAPIURL, WebURL: githubConfigurationWebURL,
				AllowedOrganizations: []string{},
			},
			want: &sonar.DopTranslationGithubConfigurationUpdateOptions{
				ClientId:             githubClientID,
				ClientSecret:         clientSecret,
				ApplicationId:        githubAppID,
				PrivateKey:           privateKey,
				ApiUrl:               githubConfigurationAPIURL,
				WebUrl:               githubConfigurationWebURL,
				AllowedOrganizations: &sonar.UpdateFieldListStringV2{Value: []string{}, Defined: true},
			},
		},
		"AllFields": {
			spec: testGitHubConfigurationSpec(),
			want: &sonar.DopTranslationGithubConfigurationUpdateOptions{
				Enabled:                         new(true),
				ClientId:                        githubClientID,
				ClientSecret:                    clientSecret,
				ApplicationId:                   githubAppID,
				PrivateKey:                      privateKey,
				SynchronizeGroups:               new(true),
				ApiUrl:                          githubConfigurationAPIURL,
				WebUrl:                          githubConfigurationWebURL,
				AllowedOrganizations:            &sonar.UpdateFieldListStringV2{Value: []string{"org-a", "org-b"}, Defined: true},
				ProvisioningType:                v1alpha1.GitHubProvisioningTypeJIT,
				AllowUsersToSignUp:              new(false),
				ProjectVisibility:               new(true),
				UserConsentRequiredAfterUpgrade: new(false),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGitHubConfigurationUpdateOptions(tc.spec, testGitHubConfigurationCredentials())
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GenerateGitHubConfigurationUpdateOptions() -want, +got:\n%s", diff)
			}
		})
	}
}
