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
	"encoding/json"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// testGitLabApplicationID is the GitLab application ID used in tests.
	testGitLabApplicationID = "gitlab-app-id"
	// testGitLabConfigurationURL is the GitLab URL used in tests.
	testGitLabConfigurationURL = "https://gitlab.example.com"
)

// newGitLabConfigurationParameters returns fully specified parameters.
func newGitLabConfigurationParameters() *v1alpha1.GitLabConfigurationParameters {
	return &v1alpha1.GitLabConfigurationParameters{
		Enabled:              new(true),
		ApplicationID:        testGitLabApplicationID,
		URL:                  testGitLabConfigurationURL,
		ApplicationSecretRef: newSecretKeySelector("gitlab", "secret"),
		SynchronizeGroups:    new(true),
		AllowedGroups:        []string{"group-a", "group-b"},
		AllowAllGroups:       new(false),
		ProvisioningType:     v1alpha1.GitLabProvisioningTypeJIT,
		AllowUsersToSignUp:   new(false),
	}
}

// newGitLabConfigurationObservation returns the observation matching
// newGitLabConfigurationParameters.
func newGitLabConfigurationObservation() *v1alpha1.GitLabConfigurationObservation {
	return &v1alpha1.GitLabConfigurationObservation{
		ID:                 "gitlab-configuration",
		Enabled:            true,
		ApplicationID:      testGitLabApplicationID,
		URL:                testGitLabConfigurationURL,
		SynchronizeGroups:  true,
		AllowedGroups:      []string{"group-b", "group-a"},
		AllowAllGroups:     false,
		ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
		AllowUsersToSignUp: false,
	}
}

// TestNewGitLabConfigurationsClient verifies that the client is built from
// the V2 DOP translation service.
func TestNewGitLabConfigurationsClient(t *testing.T) {
	t.Parallel()

	got := NewGitLabConfigurationsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000/api",
	})

	if _, ok := got.(*sonar.DopTranslationService); !ok {
		t.Errorf("NewGitLabConfigurationsClient() = %T, want *sonar.DopTranslationService", got)
	}
}

// TestLateInitializeGitLabConfiguration verifies that unset fields are
// filled from the observation and set fields are kept.
func TestLateInitializeGitLabConfiguration(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.GitLabConfigurationParameters
		observation *v1alpha1.GitLabConfigurationObservation
		want        *v1alpha1.GitLabConfigurationParameters
	}{
		"NilSpec": {
			spec:        nil,
			observation: newGitLabConfigurationObservation(),
			want:        nil,
		},
		"NilObservation": {
			spec:        &v1alpha1.GitLabConfigurationParameters{},
			observation: nil,
			want:        &v1alpha1.GitLabConfigurationParameters{},
		},
		"FillsUnsetFields": {
			spec: &v1alpha1.GitLabConfigurationParameters{ApplicationID: testGitLabApplicationID},
			observation: &v1alpha1.GitLabConfigurationObservation{
				Enabled:            true,
				SynchronizeGroups:  true,
				AllowedGroups:      []string{"group-a"},
				AllowAllGroups:     true,
				AllowUsersToSignUp: true,
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeAuto,
			},
			want: &v1alpha1.GitLabConfigurationParameters{
				Enabled:            new(true),
				ApplicationID:      testGitLabApplicationID,
				SynchronizeGroups:  new(true),
				AllowedGroups:      []string{"group-a"},
				AllowAllGroups:     new(true),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeAuto,
				AllowUsersToSignUp: new(true),
			},
		},
		"KeepsSetFields": {
			spec: newGitLabConfigurationParameters(),
			observation: &v1alpha1.GitLabConfigurationObservation{
				Enabled:            false,
				SynchronizeGroups:  false,
				AllowedGroups:      []string{"other"},
				AllowAllGroups:     true,
				AllowUsersToSignUp: true,
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeAuto,
			},
			want: newGitLabConfigurationParameters(),
		},
		"EmptyObservedGroupsLeaveGroupsUnset": {
			spec:        &v1alpha1.GitLabConfigurationParameters{},
			observation: &v1alpha1.GitLabConfigurationObservation{ProvisioningType: v1alpha1.GitLabProvisioningTypeJIT},
			want: &v1alpha1.GitLabConfigurationParameters{
				Enabled:            new(false),
				SynchronizeGroups:  new(false),
				AllowAllGroups:     new(false),
				AllowUsersToSignUp: new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeGitLabConfiguration(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.want, tc.spec); diff != "" {
				t.Errorf("LateInitializeGitLabConfiguration() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestIsGitLabConfigurationLateInitialized verifies the detection of late
// initialized fields.
func TestIsGitLabConfigurationLateInitialized(t *testing.T) {
	t.Parallel()

	withGroups := newGitLabConfigurationParameters()
	withoutGroups := newGitLabConfigurationParameters()
	withoutGroups.AllowedGroups = nil

	withoutType := newGitLabConfigurationParameters()
	withoutType.ProvisioningType = ""

	withoutSignUp := newGitLabConfigurationParameters()
	withoutSignUp.AllowUsersToSignUp = nil

	cases := map[string]struct {
		former  *v1alpha1.GitLabConfigurationParameters
		current *v1alpha1.GitLabConfigurationParameters
		want    bool
	}{
		"NilFormer":        {former: nil, current: withGroups, want: false},
		"NilCurrent":       {former: withGroups, current: nil, want: false},
		"Unchanged":        {former: withGroups, current: newGitLabConfigurationParameters(), want: false},
		"GroupsFilled":     {former: withoutGroups, current: withGroups, want: true},
		"TypeFilled":       {former: withoutType, current: withGroups, want: true},
		"BoolFieldsFilled": {former: withoutSignUp, current: withGroups, want: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGitLabConfigurationLateInitialized(tc.former, tc.current); got != tc.want {
				t.Errorf("IsGitLabConfigurationLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsGitLabConfigurationUpToDate verifies the comparison of the spec with
// the observation and of the current credentials with the saved ones.
func TestIsGitLabConfigurationUpToDate(t *testing.T) {
	t.Parallel()

	credentials := GitLabConfigurationCredentials{ApplicationSecret: "secret", ProvisioningToken: "token"}

	withSpec := func(mutate func(*v1alpha1.GitLabConfigurationParameters)) *v1alpha1.GitLabConfigurationParameters {
		spec := newGitLabConfigurationParameters()
		mutate(spec)

		return spec
	}
	withObservation := func(mutate func(*v1alpha1.GitLabConfigurationObservation)) *v1alpha1.GitLabConfigurationObservation {
		observation := newGitLabConfigurationObservation()
		mutate(observation)

		return observation
	}
	withToken := func(spec *v1alpha1.GitLabConfigurationParameters) {
		spec.ProvisioningTokenSecretRef = newSecretKeySelector("gitlab", "token")
	}

	cases := map[string]struct {
		spec        *v1alpha1.GitLabConfigurationParameters
		observation *v1alpha1.GitLabConfigurationObservation
		current     GitLabConfigurationCredentials
		saved       GitLabConfigurationCredentials
		want        bool
	}{
		"NilSpec": {
			spec: nil, observation: newGitLabConfigurationObservation(), want: true,
		},
		"NilObservation": {
			spec: newGitLabConfigurationParameters(), observation: nil, want: false,
		},
		"UpToDate": {
			spec: newGitLabConfigurationParameters(), observation: newGitLabConfigurationObservation(),
			current: credentials, saved: credentials, want: true,
		},
		"UnsetOptionalFieldsAreIgnored": {
			spec: &v1alpha1.GitLabConfigurationParameters{ApplicationID: testGitLabApplicationID, URL: testGitLabConfigurationURL},
			observation: withObservation(func(o *v1alpha1.GitLabConfigurationObservation) {
				o.AllowUsersToSignUp = true
			}),
			current: credentials, saved: credentials, want: true,
		},
		"EnabledChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.Enabled = new(false) }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"ApplicationIDChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.ApplicationID = "other" }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"URLChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.URL = "https://gitlab.com" }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"SynchronizeGroupsChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.SynchronizeGroups = new(false) }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"AllowedGroupsChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.AllowedGroups = []string{"group-a"} }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"AllowAllGroupsChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.AllowAllGroups = new(true) }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"AllowUsersToSignUpChanged": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.AllowUsersToSignUp = new(true) }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"ProvisioningTypeChanged": {
			spec: withSpec(func(s *v1alpha1.GitLabConfigurationParameters) {
				s.ProvisioningType = v1alpha1.GitLabProvisioningTypeAuto
			}),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: false,
		},
		"EmptyProvisioningTypeDefaultsToJIT": {
			spec:        withSpec(func(s *v1alpha1.GitLabConfigurationParameters) { s.ProvisioningType = "" }),
			observation: newGitLabConfigurationObservation(), current: credentials, saved: credentials, want: true,
		},
		"ApplicationSecretRotated": {
			spec: newGitLabConfigurationParameters(), observation: newGitLabConfigurationObservation(),
			current: credentials, saved: GitLabConfigurationCredentials{ApplicationSecret: "old", ProvisioningToken: "token"},
			want: false,
		},
		"ProvisioningTokenIgnoredWithoutRef": {
			spec: newGitLabConfigurationParameters(), observation: newGitLabConfigurationObservation(),
			current: credentials, saved: GitLabConfigurationCredentials{ApplicationSecret: "secret"},
			want: true,
		},
		"ProvisioningTokenRotated": {
			spec: withSpec(withToken),
			observation: withObservation(func(o *v1alpha1.GitLabConfigurationObservation) {
				o.IsProvisioningTokenSet = true
			}),
			current: credentials, saved: GitLabConfigurationCredentials{ApplicationSecret: "secret", ProvisioningToken: "old"},
			want: false,
		},
		"ProvisioningTokenNotSetInSonarQube": {
			spec:        withSpec(withToken),
			observation: newGitLabConfigurationObservation(),
			current:     credentials, saved: credentials,
			want: false,
		},
		"ProvisioningTokenUpToDate": {
			spec: withSpec(withToken),
			observation: withObservation(func(o *v1alpha1.GitLabConfigurationObservation) {
				o.IsProvisioningTokenSet = true
			}),
			current: credentials, saved: credentials,
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGitLabConfigurationUpToDate(tc.spec, tc.observation, tc.current, tc.saved); got != tc.want {
				t.Errorf("IsGitLabConfigurationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateGitLabConfigurationCreateOptions verifies the create options
// and their defaults.
func TestGenerateGitLabConfigurationCreateOptions(t *testing.T) {
	t.Parallel()

	credentials := GitLabConfigurationCredentials{ApplicationSecret: "secret", ProvisioningToken: "token"}

	cases := map[string]struct {
		spec *v1alpha1.GitLabConfigurationParameters
		want *sonar.DopTranslationGitlabConfigurationCreateOptions
	}{
		"NilSpec": {spec: nil, want: nil},
		"FullSpec": {
			spec: newGitLabConfigurationParameters(),
			want: &sonar.DopTranslationGitlabConfigurationCreateOptions{
				Enabled:            true,
				ApplicationId:      testGitLabApplicationID,
				Url:                testGitLabConfigurationURL,
				Secret:             "secret",
				SynchronizeGroups:  true,
				AllowedGroups:      []string{"group-a", "group-b"},
				AllowAllGroups:     new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
				ProvisioningToken:  "token",
				AllowUsersToSignUp: new(false),
			},
		},
		"Defaults": {
			spec: &v1alpha1.GitLabConfigurationParameters{ApplicationID: testGitLabApplicationID, URL: testGitLabConfigurationURL},
			want: &sonar.DopTranslationGitlabConfigurationCreateOptions{
				Enabled:           true,
				ApplicationId:     testGitLabApplicationID,
				Url:               testGitLabConfigurationURL,
				Secret:            "secret",
				SynchronizeGroups: false,
				AllowedGroups:     []string{},
				ProvisioningType:  v1alpha1.GitLabProvisioningTypeJIT,
				ProvisioningToken: "token",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGitLabConfigurationCreateOptions(tc.spec, credentials)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateGitLabConfigurationCreateOptions() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateGitLabConfigurationCreateOptionsSendsEmptyGroups verifies
// that unset allowed groups are sent as an empty list, which SonarQube
// requires, rather than null.
func TestGenerateGitLabConfigurationCreateOptionsSendsEmptyGroups(t *testing.T) {
	t.Parallel()

	options := GenerateGitLabConfigurationCreateOptions(&v1alpha1.GitLabConfigurationParameters{}, GitLabConfigurationCredentials{})

	body, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}

	var decoded map[string]any

	err = json.Unmarshal(body, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}

	if diff := cmp.Diff([]any{}, decoded["allowedGroups"]); diff != "" {
		t.Errorf("allowedGroups -want, +got:\n%s", diff)
	}
}

// TestGenerateGitLabConfigurationUpdateOptions verifies the update options.
func TestGenerateGitLabConfigurationUpdateOptions(t *testing.T) {
	t.Parallel()

	credentials := GitLabConfigurationCredentials{ApplicationSecret: "secret"}

	emptyGroups := newGitLabConfigurationParameters()
	emptyGroups.AllowedGroups = []string{}

	cases := map[string]struct {
		spec *v1alpha1.GitLabConfigurationParameters
		want *sonar.DopTranslationGitlabConfigurationUpdateOptions
	}{
		"NilSpec": {spec: nil, want: nil},
		"FullSpec": {
			spec: newGitLabConfigurationParameters(),
			want: &sonar.DopTranslationGitlabConfigurationUpdateOptions{
				Enabled:            new(true),
				ApplicationId:      testGitLabApplicationID,
				Url:                testGitLabConfigurationURL,
				Secret:             "secret",
				SynchronizeGroups:  new(true),
				AllowedGroups:      &sonar.UpdateFieldListStringV2{Value: []string{"group-a", "group-b"}, Defined: true},
				AllowAllGroups:     new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
				AllowUsersToSignUp: new(false),
			},
		},
		"EmptyGroupsClearTheList": {
			spec: emptyGroups,
			want: &sonar.DopTranslationGitlabConfigurationUpdateOptions{
				Enabled:            new(true),
				ApplicationId:      testGitLabApplicationID,
				Url:                testGitLabConfigurationURL,
				Secret:             "secret",
				SynchronizeGroups:  new(true),
				AllowedGroups:      &sonar.UpdateFieldListStringV2{Value: []string{}, Defined: true},
				AllowAllGroups:     new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
				AllowUsersToSignUp: new(false),
			},
		},
		"UnsetFieldsAreLeftUnchanged": {
			spec: &v1alpha1.GitLabConfigurationParameters{ApplicationID: testGitLabApplicationID, URL: testGitLabConfigurationURL},
			want: &sonar.DopTranslationGitlabConfigurationUpdateOptions{
				ApplicationId:    testGitLabApplicationID,
				Url:              testGitLabConfigurationURL,
				Secret:           "secret",
				ProvisioningType: v1alpha1.GitLabProvisioningTypeJIT,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGitLabConfigurationUpdateOptions(tc.spec, credentials)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateGitLabConfigurationUpdateOptions() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestGenerateGitLabConfigurationObservation verifies the conversion of the
// SonarQube configuration into an observation.
func TestGenerateGitLabConfigurationObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		configuration *sonar.DopTranslationGitlabConfiguration
		want          v1alpha1.GitLabConfigurationObservation
	}{
		"Nil": {configuration: nil, want: v1alpha1.GitLabConfigurationObservation{}},
		"Full": {
			configuration: &sonar.DopTranslationGitlabConfiguration{
				Id:                     "gitlab-configuration",
				Enabled:                true,
				ApplicationId:          testGitLabApplicationID,
				Url:                    testGitLabConfigurationURL,
				SynchronizeGroups:      true,
				AllowedGroups:          []string{"group-a"},
				AllowAllGroups:         true,
				AllowUsersToSignUp:     true,
				ProvisioningType:       v1alpha1.GitLabProvisioningTypeAuto,
				IsProvisioningTokenSet: true,
				ErrorMessage:           new("invalid token"),
			},
			want: v1alpha1.GitLabConfigurationObservation{
				ID:                     "gitlab-configuration",
				Enabled:                true,
				ApplicationID:          testGitLabApplicationID,
				URL:                    testGitLabConfigurationURL,
				SynchronizeGroups:      true,
				AllowedGroups:          []string{"group-a"},
				AllowAllGroups:         true,
				AllowUsersToSignUp:     true,
				ProvisioningType:       v1alpha1.GitLabProvisioningTypeAuto,
				IsProvisioningTokenSet: true,
				ErrorMessage:           "invalid token",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGitLabConfigurationObservation(tc.configuration)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateGitLabConfigurationObservation() -want, +got:\n%s", diff)
			}
		})
	}
}
