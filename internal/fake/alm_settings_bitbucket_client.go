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
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockALMSettingsBitbucketClient is a mock implementation of the
// ALMSettingsBitbucketClient interface.
type MockALMSettingsBitbucketClient struct {
	CountBindingFn    func(opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error)
	DeleteFn          func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error)
	GetBindingFn      func(opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error)
	ListFn            func(opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error)
	ListDefinitionsFn func() (*sonar.AlmSettingsListDefinitions, *http.Response, error)
	ValidateFn        func(opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error)
	CreateBitbucketFn func(opt *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error)
	UpdateBitbucketFn func(opt *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error)
}

// Ensure MockALMSettingsBitbucketClient implements ALMSettingsBitbucketClient.
var _ integration.ALMSettingsBitbucketClient = &MockALMSettingsBitbucketClient{}

// CountBinding implements ALMSettingsClient.CountBinding.
func (m *MockALMSettingsBitbucketClient) CountBinding(_ context.Context, opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error) {
	if m.CountBindingFn != nil {
		return m.CountBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements ALMSettingsClient.Delete.
func (m *MockALMSettingsBitbucketClient) Delete(_ context.Context, opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// GetBinding implements ALMSettingsClient.GetBinding.
func (m *MockALMSettingsBitbucketClient) GetBinding(_ context.Context, opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error) {
	if m.GetBindingFn != nil {
		return m.GetBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// List implements ALMSettingsClient.List.
func (m *MockALMSettingsBitbucketClient) List(_ context.Context, opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ListDefinitions implements ALMSettingsClient.ListDefinitions.
func (m *MockALMSettingsBitbucketClient) ListDefinitions(_ context.Context) (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	if m.ListDefinitionsFn != nil {
		return m.ListDefinitionsFn()
	}

	return nil, nil, errNotImplemented
}

// Validate implements ALMSettingsClient.Validate.
func (m *MockALMSettingsBitbucketClient) Validate(_ context.Context, opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error) {
	if m.ValidateFn != nil {
		return m.ValidateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateBitbucket implements ALMSettingsBitbucketClient.CreateBitbucket.
func (m *MockALMSettingsBitbucketClient) CreateBitbucket(_ context.Context, opt *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error) {
	if m.CreateBitbucketFn != nil {
		return m.CreateBitbucketFn(opt)
	}

	return nil, errNotImplemented
}

// UpdateBitbucket implements ALMSettingsBitbucketClient.UpdateBitbucket.
func (m *MockALMSettingsBitbucketClient) UpdateBitbucket(_ context.Context, opt *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error) {
	if m.UpdateBitbucketFn != nil {
		return m.UpdateBitbucketFn(opt)
	}

	return nil, errNotImplemented
}
