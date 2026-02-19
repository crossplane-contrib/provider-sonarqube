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

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockProjectLinksClient is a mock implementation of the ProjectLinksClient interface.
type MockProjectLinksClient struct {
	CreateFn func(opt *sonar.ProjectLinksCreateOption) (*sonar.ProjectLinksCreate, *http.Response, error)
	DeleteFn func(opt *sonar.ProjectLinksDeleteOption) (*http.Response, error)
	SearchFn func(opt *sonar.ProjectLinksSearchOption) (*sonar.ProjectLinksSearch, *http.Response, error)
}

// Ensure MockProjectLinksClient implements ProjectLinksClient.
var _ instance.ProjectLinksClient = &MockProjectLinksClient{}

// Create implements ProjectLinksClient.Create.
func (m *MockProjectLinksClient) Create(opt *sonar.ProjectLinksCreateOption) (*sonar.ProjectLinksCreate, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements ProjectLinksClient.Delete.
func (m *MockProjectLinksClient) Delete(opt *sonar.ProjectLinksDeleteOption) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements ProjectLinksClient.Search.
func (m *MockProjectLinksClient) Search(opt *sonar.ProjectLinksSearchOption) (*sonar.ProjectLinksSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}
