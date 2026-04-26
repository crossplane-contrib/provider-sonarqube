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

//nolint:dupl // Provider-specific ALM mock clients intentionally share this structure.
package fake

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockALMSettingsGitHubClient is a mock implementation of the
// ALMSettingsGitHubClient interface.
type MockALMSettingsGitHubClient struct {
	CountBindingFn    func(opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error)
	DeleteFn          func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error)
	GetBindingFn      func(opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error)
	ListFn            func(opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error)
	ListDefinitionsFn func() (*sonar.AlmSettingsListDefinitions, *http.Response, error)
	ValidateFn        func(opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error)
	CreateGithubFn    func(opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error)
	UpdateGithubFn    func(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error)
}

// Ensure MockALMSettingsGitHubClient implements ALMSettingsGitHubClient.
var _ integration.ALMSettingsGitHubClient = &MockALMSettingsGitHubClient{}

// CountBinding implements ALMSettingsClient.CountBinding.
func (m *MockALMSettingsGitHubClient) CountBinding(opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error) {
	if m.CountBindingFn != nil {
		return m.CountBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements ALMSettingsClient.Delete.
func (m *MockALMSettingsGitHubClient) Delete(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// GetBinding implements ALMSettingsClient.GetBinding.
func (m *MockALMSettingsGitHubClient) GetBinding(opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error) {
	if m.GetBindingFn != nil {
		return m.GetBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// List implements ALMSettingsClient.List.
func (m *MockALMSettingsGitHubClient) List(opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ListDefinitions implements ALMSettingsClient.ListDefinitions.
func (m *MockALMSettingsGitHubClient) ListDefinitions() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	if m.ListDefinitionsFn != nil {
		return m.ListDefinitionsFn()
	}

	return nil, nil, errNotImplemented
}

// Validate implements ALMSettingsClient.Validate.
func (m *MockALMSettingsGitHubClient) Validate(opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error) {
	if m.ValidateFn != nil {
		return m.ValidateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateGithub implements ALMSettingsGitHubClient.CreateGithub.
func (m *MockALMSettingsGitHubClient) CreateGithub(opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
	if m.CreateGithubFn != nil {
		return m.CreateGithubFn(opt)
	}

	return nil, errNotImplemented
}

// UpdateGithub implements ALMSettingsGitHubClient.UpdateGithub.
func (m *MockALMSettingsGitHubClient) UpdateGithub(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
	if m.UpdateGithubFn != nil {
		return m.UpdateGithubFn(opt)
	}

	return nil, errNotImplemented
}
