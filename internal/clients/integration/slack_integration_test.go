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

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestNewSlackIntegrationsClient tests the client constructor.
func TestNewSlackIntegrationsClient(t *testing.T) {
	t.Parallel()

	cfg := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "test-token",
		BaseURL:  "http://localhost:9000",
	}

	if NewSlackIntegrationsClient(cfg) == nil {
		t.Error("NewSlackIntegrationsClient() = nil, want a client")
	}
}

// TestGenerateSlackIntegrationListOptions tests that only Slack
// configurations are listed.
func TestGenerateSlackIntegrationListOptions(t *testing.T) {
	t.Parallel()

	want := &sonar.IntegrationsListIntegrationConfigurationsOptions{IntegrationType: "SLACK"}
	if diff := cmp.Diff(want, GenerateSlackIntegrationListOptions()); diff != "" {
		t.Errorf("GenerateSlackIntegrationListOptions() -want, +got:\n%s", diff)
	}
}

// TestFindSlackIntegration tests finding a configuration by ID.
func TestFindSlackIntegration(t *testing.T) {
	t.Parallel()

	search := &sonar.IntegrationsIntegrationConfigurationSearch{
		IntegrationConfigurations: []sonar.IntegrationsIntegrationConfiguration{{Id: "first"}, {Id: "second"}},
	}

	cases := map[string]struct {
		search *sonar.IntegrationsIntegrationConfigurationSearch
		id     string
		want   *sonar.IntegrationsIntegrationConfiguration
	}{
		"NilSearch": {id: "first"},
		"NotFound":  {search: search, id: "other"},
		"Found":     {search: search, id: "second", want: &sonar.IntegrationsIntegrationConfiguration{Id: "second"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, FindSlackIntegration(tc.search, tc.id)); diff != "" {
				t.Errorf("FindSlackIntegration() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestIsSlackIntegrationUpToDate tests the up to date check.
func TestIsSlackIntegrationUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.SlackIntegrationParameters{ClientID: "client"}
	observation := &v1alpha1.SlackIntegrationObservation{ClientID: "client"}
	credentials := SlackIntegrationCredentials{ClientSecret: "secret", SigningSecret: "signing"}

	cases := map[string]struct {
		spec        *v1alpha1.SlackIntegrationParameters
		observation *v1alpha1.SlackIntegrationObservation
		saved       SlackIntegrationCredentials
		want        bool
	}{
		"NilSpec":        {observation: observation, saved: credentials, want: true},
		"NilObservation": {spec: spec, saved: credentials, want: false},
		"UpToDate":       {spec: spec, observation: observation, saved: credentials, want: true},
		"ClientIDDrift": {
			spec: spec, observation: &v1alpha1.SlackIntegrationObservation{ClientID: "other"}, saved: credentials,
		},
		"ClientSecretRotated": {
			spec: spec, observation: observation, saved: SlackIntegrationCredentials{ClientSecret: "old", SigningSecret: "signing"},
		},
		"SigningSecretRotated": {
			spec: spec, observation: observation, saved: SlackIntegrationCredentials{ClientSecret: "secret", SigningSecret: "old"},
		},
		"NeverSaved": {spec: spec, observation: observation},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsSlackIntegrationUpToDate(tc.spec, tc.observation, credentials, tc.saved)
			if got != tc.want {
				t.Errorf("IsSlackIntegrationUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestGenerateSlackIntegrationOptions tests the create and update options.
func TestGenerateSlackIntegrationOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.SlackIntegrationParameters{ClientID: "client"}
	credentials := SlackIntegrationCredentials{ClientSecret: "secret", SigningSecret: "signing"}

	if got := GenerateSlackIntegrationCreateOptions(nil, credentials); got != nil {
		t.Errorf("GenerateSlackIntegrationCreateOptions(nil) = %+v, want nil", got)
	}

	if got := GenerateSlackIntegrationUpdateOptions(nil, credentials); got != nil {
		t.Errorf("GenerateSlackIntegrationUpdateOptions(nil) = %+v, want nil", got)
	}

	wantCreate := &sonar.IntegrationsIntegrationConfigurationCreateOptions{
		IntegrationType: "SLACK", ClientId: "client", ClientSecret: "secret", SigningSecret: "signing",
	}
	if diff := cmp.Diff(wantCreate, GenerateSlackIntegrationCreateOptions(spec, credentials)); diff != "" {
		t.Errorf("GenerateSlackIntegrationCreateOptions() -want, +got:\n%s", diff)
	}

	wantUpdate := &sonar.IntegrationsIntegrationConfigurationPatchOptions{
		ClientId: "client", ClientSecret: "secret", SigningSecret: "signing",
	}
	if diff := cmp.Diff(wantUpdate, GenerateSlackIntegrationUpdateOptions(spec, credentials)); diff != "" {
		t.Errorf("GenerateSlackIntegrationUpdateOptions() -want, +got:\n%s", diff)
	}
}

// TestGenerateSlackIntegrationObservation tests the observation generation.
func TestGenerateSlackIntegrationObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		configuration *sonar.IntegrationsIntegrationConfiguration
		want          v1alpha1.SlackIntegrationObservation
	}{
		"Nil": {},
		"Full": {
			configuration: &sonar.IntegrationsIntegrationConfiguration{Id: "id", IntegrationType: "SLACK", ClientId: "client", AppId: "app"},
			want:          v1alpha1.SlackIntegrationObservation{ID: "id", ClientID: "client", AppID: "app"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, GenerateSlackIntegrationObservation(tc.configuration)); diff != "" {
				t.Errorf("GenerateSlackIntegrationObservation() -want, +got:\n%s", diff)
			}
		})
	}
}
