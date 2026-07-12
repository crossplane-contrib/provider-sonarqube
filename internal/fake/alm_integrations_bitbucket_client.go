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

// MockALMIntegrationsBitbucketClient is a mock implementation of the
// ALMIntegrationsBitbucketClient interface.
type MockALMIntegrationsBitbucketClient struct {
	CheckPatFn                    func(opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error)
	SetPatFn                      func(opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error)
	ListBitbucketServerProjectsFn func(opt *sonar.AlmIntegrationsListBitbucketServerProjectsOptions) (v *sonar.AlmIntegrationsListBitbucketServerProjects, resp *http.Response, err error)
	SearchBitbucketCloudReposFn   func(opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error)
	SearchBitbucketServerReposFn  func(opt *sonar.AlmIntegrationsSearchBitbucketServerReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketServerRepos, resp *http.Response, err error)
}

// Ensure MockALMIntegrationsBitbucketClient implements
// ALMIntegrationsBitbucketClient.
var _ integration.ALMIntegrationsBitbucketClient = &MockALMIntegrationsBitbucketClient{}

// CheckPat implements ALMIntegrationsClient.CheckPat.
func (m *MockALMIntegrationsBitbucketClient) CheckPat(_ context.Context, opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error) {
	if m.CheckPatFn != nil {
		return m.CheckPatFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetPat implements ALMIntegrationsClient.SetPat.
func (m *MockALMIntegrationsBitbucketClient) SetPat(_ context.Context, opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error) {
	if m.SetPatFn != nil {
		return m.SetPatFn(opt)
	}

	return nil, errNotImplemented
}

// ListBitbucketServerProjects implements
// ALMIntegrationsBitbucketClient.ListBitbucketServerProjects.
func (m *MockALMIntegrationsBitbucketClient) ListBitbucketServerProjects(_ context.Context, opt *sonar.AlmIntegrationsListBitbucketServerProjectsOptions) (v *sonar.AlmIntegrationsListBitbucketServerProjects, resp *http.Response, err error) {
	if m.ListBitbucketServerProjectsFn != nil {
		return m.ListBitbucketServerProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchBitbucketCloudRepos implements
// ALMIntegrationsBitbucketClient.SearchBitbucketCloudRepos.
func (m *MockALMIntegrationsBitbucketClient) SearchBitbucketCloudRepos(_ context.Context, opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error) {
	if m.SearchBitbucketCloudReposFn != nil {
		return m.SearchBitbucketCloudReposFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchBitbucketServerRepos implements
// ALMIntegrationsBitbucketClient.SearchBitbucketServerRepos.
func (m *MockALMIntegrationsBitbucketClient) SearchBitbucketServerRepos(_ context.Context, opt *sonar.AlmIntegrationsSearchBitbucketServerReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketServerRepos, resp *http.Response, err error) {
	if m.SearchBitbucketServerReposFn != nil {
		return m.SearchBitbucketServerReposFn(opt)
	}

	return nil, nil, errNotImplemented
}
