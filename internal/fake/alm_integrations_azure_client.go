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

// MockALMIntegrationsAzureClient is a mock implementation of the
// ALMIntegrationsAzureClient interface.
type MockALMIntegrationsAzureClient struct {
	CheckPatFn          func(opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error)
	SetPatFn            func(opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error)
	ListAzureProjectsFn func(opt *sonar.AlmIntegrationsListAzureProjectsOptions) (v *sonar.AlmIntegrationsListAzureProjects, resp *http.Response, err error)
	SearchAzureReposFn  func(opt *sonar.AlmIntegrationsSearchAzureReposOptions) (v *sonar.AlmIntegrationsSearchAzureRepos, resp *http.Response, err error)
}

// Ensure MockALMIntegrationsAzureClient implements
// ALMIntegrationsAzureClient.
var _ integration.ALMIntegrationsAzureClient = &MockALMIntegrationsAzureClient{}

// CheckPat implements ALMIntegrationsClient.CheckPat.
func (m *MockALMIntegrationsAzureClient) CheckPat(_ context.Context, opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error) {
	if m.CheckPatFn != nil {
		return m.CheckPatFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetPat implements ALMIntegrationsClient.SetPat.
func (m *MockALMIntegrationsAzureClient) SetPat(_ context.Context, opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error) {
	if m.SetPatFn != nil {
		return m.SetPatFn(opt)
	}

	return nil, errNotImplemented
}

// ListAzureProjects implements ALMIntegrationsAzureClient.ListAzureProjects.
func (m *MockALMIntegrationsAzureClient) ListAzureProjects(_ context.Context, opt *sonar.AlmIntegrationsListAzureProjectsOptions) (v *sonar.AlmIntegrationsListAzureProjects, resp *http.Response, err error) {
	if m.ListAzureProjectsFn != nil {
		return m.ListAzureProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchAzureRepos implements ALMIntegrationsAzureClient.SearchAzureRepos.
func (m *MockALMIntegrationsAzureClient) SearchAzureRepos(_ context.Context, opt *sonar.AlmIntegrationsSearchAzureReposOptions) (v *sonar.AlmIntegrationsSearchAzureRepos, resp *http.Response, err error) {
	if m.SearchAzureReposFn != nil {
		return m.SearchAzureReposFn(opt)
	}

	return nil, nil, errNotImplemented
}
