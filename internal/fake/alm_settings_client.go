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
	"errors"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

var errAlmNotImplemented = errors.New("alm operation not implemented")

// MockAlmClient is a mock implementation of the AlmClient interface.
type MockAlmClient struct {
	CreateGithubFn    func(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error)
	UpdateGithubFn    func(opt *sonar.AlmSettingsUpdateGithubOption) (*http.Response, error)
	DeleteFn          func(opt *sonar.AlmSettingsDeleteOption) (*http.Response, error)
	ListDefinitionsFn func() (*sonar.AlmSettingsListDefinitions, *http.Response, error)
}

// Ensure MockAlmClient implements AlmClient.
var _ instance.AlmClient = &MockAlmClient{}

// CreateGithub implements AlmClient.CreateGithub.
func (m *MockAlmClient) CreateGithub(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error) {
	if m.CreateGithubFn != nil {
		return m.CreateGithubFn(opt)
	}

	return nil, errAlmNotImplemented
}

// UpdateGithub implements AlmClient.UpdateGithub.
func (m *MockAlmClient) UpdateGithub(opt *sonar.AlmSettingsUpdateGithubOption) (*http.Response, error) {
	if m.UpdateGithubFn != nil {
		return m.UpdateGithubFn(opt)
	}

	return nil, errAlmNotImplemented
}

// Delete implements AlmClient.Delete.
func (m *MockAlmClient) Delete(opt *sonar.AlmSettingsDeleteOption) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errAlmNotImplemented
}

// ListDefinitions implements AlmClient.ListDefinitions.
func (m *MockAlmClient) ListDefinitions() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	if m.ListDefinitionsFn != nil {
		return m.ListDefinitionsFn()
	}

	return nil, nil, errAlmNotImplemented
}
