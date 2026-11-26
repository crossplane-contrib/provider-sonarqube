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

package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockSlackIntegrationsClient is a mock implementation of the
// SlackIntegrationsClient interface.
type MockSlackIntegrationsClient struct {
	ListIntegrationConfigurationsFn  func(opt *sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error)
	CreateIntegrationConfigurationFn func(opt *sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error)
	UpdateIntegrationConfigurationFn func(configurationID string, opt *sonar.IntegrationsIntegrationConfigurationPatchOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error)
	DeleteIntegrationConfigurationFn func(configurationID string) (*http.Response, error)
}

// Ensure MockSlackIntegrationsClient implements SlackIntegrationsClient.
var _ integration.SlackIntegrationsClient = &MockSlackIntegrationsClient{}

// ListIntegrationConfigurations implements
// SlackIntegrationsClient.ListIntegrationConfigurations.
func (m *MockSlackIntegrationsClient) ListIntegrationConfigurations(_ context.Context, opt *sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error) {
	if m.ListIntegrationConfigurationsFn != nil {
		return m.ListIntegrationConfigurationsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateIntegrationConfiguration implements
// SlackIntegrationsClient.CreateIntegrationConfiguration.
func (m *MockSlackIntegrationsClient) CreateIntegrationConfiguration(_ context.Context, opt *sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
	if m.CreateIntegrationConfigurationFn != nil {
		return m.CreateIntegrationConfigurationFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateIntegrationConfiguration implements
// SlackIntegrationsClient.UpdateIntegrationConfiguration.
func (m *MockSlackIntegrationsClient) UpdateIntegrationConfiguration(_ context.Context, configurationID string, opt *sonar.IntegrationsIntegrationConfigurationPatchOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
	if m.UpdateIntegrationConfigurationFn != nil {
		return m.UpdateIntegrationConfigurationFn(configurationID, opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteIntegrationConfiguration implements
// SlackIntegrationsClient.DeleteIntegrationConfiguration.
func (m *MockSlackIntegrationsClient) DeleteIntegrationConfiguration(_ context.Context, configurationID string) (*http.Response, error) {
	if m.DeleteIntegrationConfigurationFn != nil {
		return m.DeleteIntegrationConfigurationFn(configurationID)
	}

	return nil, errNotImplemented
}
