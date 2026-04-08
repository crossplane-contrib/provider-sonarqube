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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockALMSettingsGitLabClient is a mock implementation of the ALMSettingsGitLabClient interface.
type MockALMSettingsGitLabClient struct {
	CountBindingFn    func(opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error)
	DeleteFn          func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error)
	GetBindingFn      func(opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error)
	ListFn            func(opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error)
	ListDefinitionsFn func() (*sonar.AlmSettingsListDefinitions, *http.Response, error)
	ValidateFn        func(opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error)
	CreateGitlabFn    func(opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error)
	UpdateGitlabFn    func(opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error)
}

// Ensure MockALMSettingsGitLabClient implements ALMSettingsGitLabClient.
var _ integration.ALMSettingsGitLabClient = &MockALMSettingsGitLabClient{}

// CountBinding implements ALMSettingsClient.CountBinding.
func (m *MockALMSettingsGitLabClient) CountBinding(opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error) {
	if m.CountBindingFn != nil {
		return m.CountBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements ALMSettingsClient.Delete.
func (m *MockALMSettingsGitLabClient) Delete(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// GetBinding implements ALMSettingsClient.GetBinding.
func (m *MockALMSettingsGitLabClient) GetBinding(opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error) {
	if m.GetBindingFn != nil {
		return m.GetBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// List implements ALMSettingsClient.List.
func (m *MockALMSettingsGitLabClient) List(opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ListDefinitions implements ALMSettingsClient.ListDefinitions.
func (m *MockALMSettingsGitLabClient) ListDefinitions() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	if m.ListDefinitionsFn != nil {
		return m.ListDefinitionsFn()
	}

	return nil, nil, errNotImplemented
}

// Validate implements ALMSettingsClient.Validate.
func (m *MockALMSettingsGitLabClient) Validate(opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error) {
	if m.ValidateFn != nil {
		return m.ValidateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateGitlab implements ALMSettingsGitLabClient.CreateGitlab.
func (m *MockALMSettingsGitLabClient) CreateGitlab(opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error) {
	if m.CreateGitlabFn != nil {
		return m.CreateGitlabFn(opt)
	}

	return nil, errNotImplemented
}

// UpdateGitlab implements ALMSettingsGitLabClient.UpdateGitlab.
func (m *MockALMSettingsGitLabClient) UpdateGitlab(opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
	if m.UpdateGitlabFn != nil {
		return m.UpdateGitlabFn(opt)
	}

	return nil, errNotImplemented
}
