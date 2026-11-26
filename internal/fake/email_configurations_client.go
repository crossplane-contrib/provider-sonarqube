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

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockEmailConfigurationsClient is a mock implementation of the
// EmailConfigurationsClient interface.
type MockEmailConfigurationsClient struct {
	SearchEmailConfigurationsFn func() (*sonar.SystemEmailConfigurationsSearch, *http.Response, error)
	CreateEmailConfigurationFn  func(opt *sonar.SystemEmailConfigurationCreateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error)
	GetEmailConfigurationFn     func(id string) (*sonar.SystemEmailConfiguration, *http.Response, error)
	UpdateEmailConfigurationFn  func(id string, opt *sonar.SystemEmailConfigurationUpdateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error)
	DeleteEmailConfigurationFn  func(id string) (*http.Response, error)
}

// Ensure MockEmailConfigurationsClient implements EmailConfigurationsClient.
var _ instance.EmailConfigurationsClient = &MockEmailConfigurationsClient{}

// SearchEmailConfigurations implements
// EmailConfigurationsClient.SearchEmailConfigurations.
func (m *MockEmailConfigurationsClient) SearchEmailConfigurations(_ context.Context) (*sonar.SystemEmailConfigurationsSearch, *http.Response, error) {
	if m.SearchEmailConfigurationsFn != nil {
		return m.SearchEmailConfigurationsFn()
	}

	return nil, nil, errNotImplemented
}

// CreateEmailConfiguration implements
// EmailConfigurationsClient.CreateEmailConfiguration.
func (m *MockEmailConfigurationsClient) CreateEmailConfiguration(_ context.Context, opt *sonar.SystemEmailConfigurationCreateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error) {
	if m.CreateEmailConfigurationFn != nil {
		return m.CreateEmailConfigurationFn(opt)
	}

	return nil, nil, errNotImplemented
}

// GetEmailConfiguration implements
// EmailConfigurationsClient.GetEmailConfiguration.
func (m *MockEmailConfigurationsClient) GetEmailConfiguration(_ context.Context, id string) (*sonar.SystemEmailConfiguration, *http.Response, error) {
	if m.GetEmailConfigurationFn != nil {
		return m.GetEmailConfigurationFn(id)
	}

	return nil, nil, errNotImplemented
}

// UpdateEmailConfiguration implements
// EmailConfigurationsClient.UpdateEmailConfiguration.
func (m *MockEmailConfigurationsClient) UpdateEmailConfiguration(_ context.Context, id string, opt *sonar.SystemEmailConfigurationUpdateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error) {
	if m.UpdateEmailConfigurationFn != nil {
		return m.UpdateEmailConfigurationFn(id, opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteEmailConfiguration implements
// EmailConfigurationsClient.DeleteEmailConfiguration.
func (m *MockEmailConfigurationsClient) DeleteEmailConfiguration(_ context.Context, id string) (*http.Response, error) {
	if m.DeleteEmailConfigurationFn != nil {
		return m.DeleteEmailConfigurationFn(id)
	}

	return nil, errNotImplemented
}
