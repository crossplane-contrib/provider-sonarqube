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

// Package fake provides mock client implementations for testing.
package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockALMIntegrationsGitLabClient is a mock implementation of the
// ALMIntegrationsGitLabClient interface.
type MockALMIntegrationsGitLabClient struct {
	CheckPatFn          func(opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error)
	SetPatFn            func(opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error)
	SearchGitlabReposFn func(opt *sonar.AlmIntegrationsSearchGitlabReposOptions) (v *sonar.AlmIntegrationsSearchGitlabRepos, resp *http.Response, err error)
}

// Ensure MockALMIntegrationsGitLabClient implements
// ALMIntegrationsGitLabClient.
var _ integration.ALMIntegrationsGitLabClient = &MockALMIntegrationsGitLabClient{}

// CheckPat implements ALMIntegrationsClient.CheckPat.
func (m *MockALMIntegrationsGitLabClient) CheckPat(_ context.Context, opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error) {
	if m.CheckPatFn != nil {
		return m.CheckPatFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetPat implements ALMIntegrationsClient.SetPat.
func (m *MockALMIntegrationsGitLabClient) SetPat(_ context.Context, opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error) {
	if m.SetPatFn != nil {
		return m.SetPatFn(opt)
	}

	return nil, errNotImplemented
}

// SearchGitlabRepos implements ALMIntegrationsGitLabClient.SearchGitlabRepos.
func (m *MockALMIntegrationsGitLabClient) SearchGitlabRepos(_ context.Context, opt *sonar.AlmIntegrationsSearchGitlabReposOptions) (v *sonar.AlmIntegrationsSearchGitlabRepos, resp *http.Response, err error) {
	if m.SearchGitlabReposFn != nil {
		return m.SearchGitlabReposFn(opt)
	}

	return nil, nil, errNotImplemented
}
