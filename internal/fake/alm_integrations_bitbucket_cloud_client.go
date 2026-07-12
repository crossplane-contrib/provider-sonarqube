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

// MockALMIntegrationsBitbucketCloudClient is a mock implementation of the
// ALMIntegrationsBitbucketCloudClient interface.
type MockALMIntegrationsBitbucketCloudClient struct {
	CheckPatFn                  func(opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error)
	SetPatFn                    func(opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error)
	SearchBitbucketCloudReposFn func(opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error)
}

// Ensure MockALMIntegrationsBitbucketCloudClient implements
// ALMIntegrationsBitbucketCloudClient.
var _ integration.ALMIntegrationsBitbucketCloudClient = &MockALMIntegrationsBitbucketCloudClient{}

// CheckPat implements ALMIntegrationsClient.CheckPat.
func (m *MockALMIntegrationsBitbucketCloudClient) CheckPat(_ context.Context, opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error) {
	if m.CheckPatFn != nil {
		return m.CheckPatFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetPat implements ALMIntegrationsClient.SetPat.
func (m *MockALMIntegrationsBitbucketCloudClient) SetPat(_ context.Context, opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error) {
	if m.SetPatFn != nil {
		return m.SetPatFn(opt)
	}

	return nil, errNotImplemented
}

// SearchBitbucketCloudRepos implements
// ALMIntegrationsBitbucketCloudClient.SearchBitbucketCloudRepos.
func (m *MockALMIntegrationsBitbucketCloudClient) SearchBitbucketCloudRepos(_ context.Context, opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error) {
	if m.SearchBitbucketCloudReposFn != nil {
		return m.SearchBitbucketCloudReposFn(opt)
	}

	return nil, nil, errNotImplemented
}
