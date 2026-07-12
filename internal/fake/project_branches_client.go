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

// MockProjectBranchesClient is a mock implementation of the
// ProjectBranchesClient interface.
type MockProjectBranchesClient struct {
	DeleteFn                         func(opt *sonar.ProjectBranchesDeleteOptions) (*http.Response, error)
	ListFn                           func(opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error)
	RenameFn                         func(opt *sonar.ProjectBranchesRenameOptions) (*http.Response, error)
	SetAutomaticDeletionProtectionFn func(opt *sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions) (*http.Response, error)
	SetMainFn                        func(opt *sonar.ProjectBranchesSetMainOptions) (*http.Response, error)
}

// Ensure MockProjectBranchesClient implements ProjectBranchesClient.
var _ instance.ProjectBranchesClient = &MockProjectBranchesClient{}

// Delete implements ProjectBranchesClient.Delete.
func (m *MockProjectBranchesClient) Delete(_ context.Context, opt *sonar.ProjectBranchesDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// List implements ProjectBranchesClient.List.
func (m *MockProjectBranchesClient) List(_ context.Context, opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Rename implements ProjectBranchesClient.Rename.
func (m *MockProjectBranchesClient) Rename(_ context.Context, opt *sonar.ProjectBranchesRenameOptions) (*http.Response, error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}

	return nil, errNotImplemented
}

// SetAutomaticDeletionProtection implements
// ProjectBranchesClient.SetAutomaticDeletionProtection.
func (m *MockProjectBranchesClient) SetAutomaticDeletionProtection(_ context.Context, opt *sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions) (*http.Response, error) {
	if m.SetAutomaticDeletionProtectionFn != nil {
		return m.SetAutomaticDeletionProtectionFn(opt)
	}

	return nil, errNotImplemented
}

// SetMain implements ProjectBranchesClient.SetMain.
func (m *MockProjectBranchesClient) SetMain(_ context.Context, opt *sonar.ProjectBranchesSetMainOptions) (*http.Response, error) {
	if m.SetMainFn != nil {
		return m.SetMainFn(opt)
	}

	return nil, errNotImplemented
}
