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

// MockGitHubConfigurationsClient is a mock implementation of the
// GitHubConfigurationsClient interface.
type MockGitHubConfigurationsClient struct {
	SearchGithubConfigurationsFn func() (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error)
	CreateGithubConfigurationFn  func(opt *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error)
	UpdateGithubConfigurationFn  func(id string, opt *sonar.DopTranslationGithubConfigurationUpdateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error)
	DeleteGithubConfigurationFn  func(id string) (*http.Response, error)
}

// Ensure MockGitHubConfigurationsClient implements GitHubConfigurationsClient.
var _ integration.GitHubConfigurationsClient = &MockGitHubConfigurationsClient{}

// SearchGithubConfigurations implements
// GitHubConfigurationsClient.SearchGithubConfigurations.
func (m *MockGitHubConfigurationsClient) SearchGithubConfigurations(_ context.Context) (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error) {
	if m.SearchGithubConfigurationsFn != nil {
		return m.SearchGithubConfigurationsFn()
	}

	return nil, nil, errNotImplemented
}

// CreateGithubConfiguration implements
// GitHubConfigurationsClient.CreateGithubConfiguration.
func (m *MockGitHubConfigurationsClient) CreateGithubConfiguration(_ context.Context, opt *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
	if m.CreateGithubConfigurationFn != nil {
		return m.CreateGithubConfigurationFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateGithubConfiguration implements
// GitHubConfigurationsClient.UpdateGithubConfiguration.
func (m *MockGitHubConfigurationsClient) UpdateGithubConfiguration(_ context.Context, id string, opt *sonar.DopTranslationGithubConfigurationUpdateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
	if m.UpdateGithubConfigurationFn != nil {
		return m.UpdateGithubConfigurationFn(id, opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteGithubConfiguration implements
// GitHubConfigurationsClient.DeleteGithubConfiguration.
func (m *MockGitHubConfigurationsClient) DeleteGithubConfiguration(_ context.Context, id string) (*http.Response, error) {
	if m.DeleteGithubConfigurationFn != nil {
		return m.DeleteGithubConfigurationFn(id)
	}

	return nil, errNotImplemented
}
