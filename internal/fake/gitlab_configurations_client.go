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

// MockGitLabConfigurationsClient is a mock implementation of the
// GitLabConfigurationsClient interface.
type MockGitLabConfigurationsClient struct {
	SearchGitlabConfigurationsFn func() (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error)
	CreateGitlabConfigurationFn  func(opt *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	GetGitlabConfigurationFn     func(id string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	UpdateGitlabConfigurationFn  func(id string, opt *sonar.DopTranslationGitlabConfigurationUpdateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	DeleteGitlabConfigurationFn  func(id string) (*http.Response, error)
}

// Ensure MockGitLabConfigurationsClient implements GitLabConfigurationsClient.
var _ integration.GitLabConfigurationsClient = &MockGitLabConfigurationsClient{}

// SearchGitlabConfigurations implements
// GitLabConfigurationsClient.SearchGitlabConfigurations.
func (m *MockGitLabConfigurationsClient) SearchGitlabConfigurations(_ context.Context) (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error) {
	if m.SearchGitlabConfigurationsFn != nil {
		return m.SearchGitlabConfigurationsFn()
	}

	return nil, nil, errNotImplemented
}

// CreateGitlabConfiguration implements
// GitLabConfigurationsClient.CreateGitlabConfiguration.
func (m *MockGitLabConfigurationsClient) CreateGitlabConfiguration(_ context.Context, opt *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
	if m.CreateGitlabConfigurationFn != nil {
		return m.CreateGitlabConfigurationFn(opt)
	}

	return nil, nil, errNotImplemented
}

// GetGitlabConfiguration implements
// GitLabConfigurationsClient.GetGitlabConfiguration.
func (m *MockGitLabConfigurationsClient) GetGitlabConfiguration(_ context.Context, id string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
	if m.GetGitlabConfigurationFn != nil {
		return m.GetGitlabConfigurationFn(id)
	}

	return nil, nil, errNotImplemented
}

// UpdateGitlabConfiguration implements
// GitLabConfigurationsClient.UpdateGitlabConfiguration.
func (m *MockGitLabConfigurationsClient) UpdateGitlabConfiguration(_ context.Context, id string, opt *sonar.DopTranslationGitlabConfigurationUpdateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
	if m.UpdateGitlabConfigurationFn != nil {
		return m.UpdateGitlabConfigurationFn(id, opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteGitlabConfiguration implements
// GitLabConfigurationsClient.DeleteGitlabConfiguration.
func (m *MockGitLabConfigurationsClient) DeleteGitlabConfiguration(_ context.Context, id string) (*http.Response, error) {
	if m.DeleteGitlabConfigurationFn != nil {
		return m.DeleteGitlabConfigurationFn(id)
	}

	return nil, errNotImplemented
}
