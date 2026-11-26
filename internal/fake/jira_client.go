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

// MockJiraClient is a mock implementation of the JiraClient interface.
type MockJiraClient struct {
	GetOrganizationBindingFn    func(opt *sonar.JiraOrganizationBindingOptions) (*sonar.JiraOrganizationBinding, *http.Response, error)
	UpdateOrganizationBindingFn func(body sonar.JiraOrganizationBinding) (*sonar.JiraOrganizationBinding, *http.Response, error)
	DeleteOrganizationBindingFn func(opt *sonar.JiraOrganizationBindingOptions) (*http.Response, error)
	GetProjectBindingFn         func(opt *sonar.JiraProjectBindingOptions) (*sonar.JiraProjectBinding, *http.Response, error)
	CreateProjectBindingFn      func(body sonar.JiraProjectBinding) (*sonar.JiraProjectBinding, *http.Response, error)
	DeleteProjectBindingFn      func(opt *sonar.JiraProjectBindingOptions) (*http.Response, error)
}

// Ensure MockJiraClient implements JiraClient.
var _ integration.JiraClient = &MockJiraClient{}

// GetOrganizationBinding implements JiraClient.GetOrganizationBinding.
func (m *MockJiraClient) GetOrganizationBinding(_ context.Context, opt *sonar.JiraOrganizationBindingOptions) (*sonar.JiraOrganizationBinding, *http.Response, error) {
	if m.GetOrganizationBindingFn != nil {
		return m.GetOrganizationBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateOrganizationBinding implements JiraClient.UpdateOrganizationBinding.
func (m *MockJiraClient) UpdateOrganizationBinding(_ context.Context, body sonar.JiraOrganizationBinding) (*sonar.JiraOrganizationBinding, *http.Response, error) {
	if m.UpdateOrganizationBindingFn != nil {
		return m.UpdateOrganizationBindingFn(body)
	}

	return nil, nil, errNotImplemented
}

// DeleteOrganizationBinding implements JiraClient.DeleteOrganizationBinding.
func (m *MockJiraClient) DeleteOrganizationBinding(_ context.Context, opt *sonar.JiraOrganizationBindingOptions) (*http.Response, error) {
	if m.DeleteOrganizationBindingFn != nil {
		return m.DeleteOrganizationBindingFn(opt)
	}

	return nil, errNotImplemented
}

// GetProjectBinding implements JiraClient.GetProjectBinding.
func (m *MockJiraClient) GetProjectBinding(_ context.Context, opt *sonar.JiraProjectBindingOptions) (*sonar.JiraProjectBinding, *http.Response, error) {
	if m.GetProjectBindingFn != nil {
		return m.GetProjectBindingFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateProjectBinding implements JiraClient.CreateProjectBinding.
func (m *MockJiraClient) CreateProjectBinding(_ context.Context, body sonar.JiraProjectBinding) (*sonar.JiraProjectBinding, *http.Response, error) {
	if m.CreateProjectBindingFn != nil {
		return m.CreateProjectBindingFn(body)
	}

	return nil, nil, errNotImplemented
}

// DeleteProjectBinding implements JiraClient.DeleteProjectBinding.
func (m *MockJiraClient) DeleteProjectBinding(_ context.Context, opt *sonar.JiraProjectBindingOptions) (*http.Response, error) {
	if m.DeleteProjectBindingFn != nil {
		return m.DeleteProjectBindingFn(opt)
	}

	return nil, errNotImplemented
}
