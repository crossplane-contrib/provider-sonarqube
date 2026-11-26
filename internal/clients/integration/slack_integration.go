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
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// SlackIntegrationsClient is the interface for interacting with the SonarQube
// integration configurations API
// (api/v2/integrations/integration-configurations). The API is keyed by
// integration type, Slack being the only one it supports.
type SlackIntegrationsClient interface {
	ListIntegrationConfigurations(ctx context.Context, opt *sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error)
	CreateIntegrationConfiguration(ctx context.Context, opt *sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error)
	UpdateIntegrationConfiguration(ctx context.Context, configurationID string, opt *sonar.IntegrationsIntegrationConfigurationPatchOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error)
	DeleteIntegrationConfiguration(ctx context.Context, configurationID string) (*http.Response, error)
}

// SlackIntegrationCredentials are the write-only values of a Slack
// integration, which SonarQube never returns.
type SlackIntegrationCredentials struct {
	// ClientSecret is the client secret of the Slack app.
	ClientSecret string
	// SigningSecret is the signing secret of the Slack app.
	SigningSecret string
}

// NewSlackIntegrationsClient creates a new SlackIntegrationsClient with the
// provided SonarQube client configuration. SonarQube holds at most one Slack
// configuration, read by a single managed resource, so its responses are not
// cached.
func NewSlackIntegrationsClient(clientConfig common.Config) SlackIntegrationsClient {
	return common.NewClient(clientConfig).V2.Integrations
}

// GenerateSlackIntegrationListOptions generates the SonarQube API options for
// listing the Slack configurations.
func GenerateSlackIntegrationListOptions() *sonar.IntegrationsListIntegrationConfigurationsOptions {
	return &sonar.IntegrationsListIntegrationConfigurationsOptions{IntegrationType: sonar.IntegrationTypeSlack}
}

// FindSlackIntegration returns the configuration with the given ID, or nil if
// the search results do not contain it. The API has no endpoint to get a
// single configuration.
func FindSlackIntegration(search *sonar.IntegrationsIntegrationConfigurationSearch, configurationID string) *sonar.IntegrationsIntegrationConfiguration {
	if search == nil {
		return nil
	}

	for i := range search.IntegrationConfigurations {
		if search.IntegrationConfigurations[i].Id == configurationID {
			return &search.IntegrationConfigurations[i]
		}
	}

	return nil
}

// IsSlackIntegrationUpToDate reports whether the SlackIntegration spec
// matches the configuration observed in SonarQube.
//
// SonarQube never returns the client secret nor the signing secret, so the
// values read from the spec's secret references (current) are compared with
// those last applied by the provider (saved).
func IsSlackIntegrationUpToDate(spec *v1alpha1.SlackIntegrationParameters, observation *v1alpha1.SlackIntegrationObservation, current, saved SlackIntegrationCredentials) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.ClientID == observation.ClientID && current == saved
}

// GenerateSlackIntegrationCreateOptions generates the SonarQube API options
// for creating a Slack configuration from the spec and the values of its
// secret references.
func GenerateSlackIntegrationCreateOptions(spec *v1alpha1.SlackIntegrationParameters, credentials SlackIntegrationCredentials) *sonar.IntegrationsIntegrationConfigurationCreateOptions {
	if spec == nil {
		return nil
	}

	return &sonar.IntegrationsIntegrationConfigurationCreateOptions{
		IntegrationType: sonar.IntegrationTypeSlack,
		ClientId:        spec.ClientID,
		ClientSecret:    credentials.ClientSecret,
		SigningSecret:   credentials.SigningSecret,
	}
}

// GenerateSlackIntegrationUpdateOptions generates the SonarQube API options
// for updating a Slack configuration from the spec and the values of its
// secret references.
func GenerateSlackIntegrationUpdateOptions(spec *v1alpha1.SlackIntegrationParameters, credentials SlackIntegrationCredentials) *sonar.IntegrationsIntegrationConfigurationPatchOptions {
	if spec == nil {
		return nil
	}

	return &sonar.IntegrationsIntegrationConfigurationPatchOptions{
		ClientId:      spec.ClientID,
		ClientSecret:  credentials.ClientSecret,
		SigningSecret: credentials.SigningSecret,
	}
}

// GenerateSlackIntegrationObservation generates the
// SlackIntegrationObservation from the configuration returned by SonarQube.
func GenerateSlackIntegrationObservation(configuration *sonar.IntegrationsIntegrationConfiguration) v1alpha1.SlackIntegrationObservation {
	if configuration == nil {
		return v1alpha1.SlackIntegrationObservation{}
	}

	return v1alpha1.SlackIntegrationObservation{
		ID:       configuration.Id,
		ClientID: configuration.ClientId,
		AppID:    configuration.AppId,
	}
}
