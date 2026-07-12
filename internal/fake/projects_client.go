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

// MockProjectsClient is a mock implementation of the ProjectsClient interface.
type MockProjectsClient struct {
	BulkDeleteFn                func(opt *sonar.ProjectsBulkDeleteOptions) (*http.Response, error)
	CreateFn                    func(opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error)
	DeleteFn                    func(opt *sonar.ProjectsDeleteOptions) (*http.Response, error)
	SearchFn                    func(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error)
	SearchMyProjectsFn          func(opt *sonar.ProjectsSearchMyProjectsOptions) (*sonar.ProjectsSearchMyProjects, *http.Response, error)
	SearchMyScannableProjectsFn func(opt *sonar.ProjectsSearchMyScannableProjectsOptions) (*sonar.ProjectsSearchMyScannableProjects, *http.Response, error)
	UpdateKeyFn                 func(opt *sonar.ProjectsUpdateKeyOptions) (*http.Response, error)
	UpdateVisibilityFn          func(opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error)
}

// Ensure MockProjectsClient implements ProjectsClient.
var _ instance.ProjectsClient = &MockProjectsClient{}

// BulkDelete implements ProjectsClient.BulkDelete.
func (m *MockProjectsClient) BulkDelete(_ context.Context, opt *sonar.ProjectsBulkDeleteOptions) (*http.Response, error) {
	if m.BulkDeleteFn != nil {
		return m.BulkDeleteFn(opt)
	}

	return nil, errNotImplemented
}

// Create implements ProjectsClient.Create.
func (m *MockProjectsClient) Create(_ context.Context, opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements ProjectsClient.Delete.
func (m *MockProjectsClient) Delete(_ context.Context, opt *sonar.ProjectsDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements ProjectsClient.Search.
func (m *MockProjectsClient) Search(_ context.Context, opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchMyProjects implements ProjectsClient.SearchMyProjects.
func (m *MockProjectsClient) SearchMyProjects(_ context.Context, opt *sonar.ProjectsSearchMyProjectsOptions) (*sonar.ProjectsSearchMyProjects, *http.Response, error) {
	if m.SearchMyProjectsFn != nil {
		return m.SearchMyProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchMyScannableProjects implements
// ProjectsClient.SearchMyScannableProjects.
func (m *MockProjectsClient) SearchMyScannableProjects(_ context.Context, opt *sonar.ProjectsSearchMyScannableProjectsOptions) (*sonar.ProjectsSearchMyScannableProjects, *http.Response, error) {
	if m.SearchMyScannableProjectsFn != nil {
		return m.SearchMyScannableProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateKey implements ProjectsClient.UpdateKey.
func (m *MockProjectsClient) UpdateKey(_ context.Context, opt *sonar.ProjectsUpdateKeyOptions) (*http.Response, error) {
	if m.UpdateKeyFn != nil {
		return m.UpdateKeyFn(opt)
	}

	return nil, errNotImplemented
}

// UpdateVisibility implements ProjectsClient.UpdateVisibility.
func (m *MockProjectsClient) UpdateVisibility(_ context.Context, opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error) {
	if m.UpdateVisibilityFn != nil {
		return m.UpdateVisibilityFn(opt)
	}

	return nil, errNotImplemented
}
