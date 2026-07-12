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

// MockPortfoliosClient is a mock implementation of the
// PortfoliosClient interface.
type MockPortfoliosClient struct {
	AddApplicationFn           func(opt *sonar.ViewsAddApplicationOptions) (*http.Response, error)
	AddApplicationBranchFn     func(opt *sonar.ViewsAddApplicationBranchOptions) (*http.Response, error)
	AddPortfolioFn             func(opt *sonar.ViewsAddPortfolioOptions) (*http.Response, error)
	AddProjectFn               func(opt *sonar.ViewsAddProjectOptions) (*http.Response, error)
	AddProjectBranchFn         func(opt *sonar.ViewsAddProjectBranchOptions) (*http.Response, error)
	ApplicationsFn             func(opt *sonar.ViewsApplicationsOptions) (*sonar.ViewsApplications, *http.Response, error)
	CreateFn                   func(opt *sonar.ViewsCreateOptions) (*http.Response, error)
	DeleteFn                   func(opt *sonar.ViewsDeleteOptions) (*http.Response, error)
	ListFn                     func() (*sonar.ViewsList, *http.Response, error)
	MoveFn                     func(opt *sonar.ViewsMoveOptions) (*http.Response, error)
	MoveOptionsFn              func(opt *sonar.ViewsMoveOptionsOptions) (*sonar.ViewsMoveDestinations, *http.Response, error)
	ProjectsFn                 func(opt *sonar.ViewsProjectsOptions) (*sonar.ViewsProjects, *http.Response, error)
	ProjectsStatusFn           func(opt *sonar.ViewsProjectsStatusOptions) (*sonar.ViewsProjectsStatus, *http.Response, error)
	RefreshFn                  func(opt *sonar.ViewsRefreshOptions) (*http.Response, error)
	RemoveApplicationFn        func(opt *sonar.ViewsRemoveApplicationOptions) (*http.Response, error)
	RemoveApplicationBranchFn  func(opt *sonar.ViewsRemoveApplicationBranchOptions) (*http.Response, error)
	RemovePortfolioFn          func(opt *sonar.ViewsRemovePortfolioOptions) (*http.Response, error)
	RemoveProjectFn            func(opt *sonar.ViewsRemoveProjectOptions) (*http.Response, error)
	RemoveProjectBranchFn      func(opt *sonar.ViewsRemoveProjectBranchOptions) (*http.Response, error)
	SearchFn                   func(opt *sonar.ViewsSearchOptions) (*sonar.ViewsSearch, *http.Response, error)
	SetManualModeFn            func(opt *sonar.ViewsSetManualModeOptions) (*http.Response, error)
	SetNoneModeFn              func(opt *sonar.ViewsSetNoneModeOptions) (*http.Response, error)
	SetRegexpModeFn            func(opt *sonar.ViewsSetRegexpModeOptions) (*http.Response, error)
	SetRemainingProjectsModeFn func(opt *sonar.ViewsSetRemainingProjectsModeOptions) (*http.Response, error)
	SetTagsModeFn              func(opt *sonar.ViewsSetTagsModeOptions) (*http.Response, error)
	ShowFn                     func(opt *sonar.ViewsShowOptions) (*sonar.ViewsShow, *http.Response, error)
	SubPortfoliosFn            func(opt *sonar.ViewsSubViewsOptions) (*sonar.ViewsSubViews, *http.Response, error)
	UpdateFn                   func(opt *sonar.ViewsUpdateOptions) (*http.Response, error)
}

// Ensure MockPortfoliosClient implements PortfoliosClient.
var _ instance.PortfoliosClient = &MockPortfoliosClient{}

// AddApplication implements PortfoliosClient.AddApplication.
func (m *MockPortfoliosClient) AddApplication(_ context.Context, opt *sonar.ViewsAddApplicationOptions) (*http.Response, error) {
	if m.AddApplicationFn != nil {
		return m.AddApplicationFn(opt)
	}

	return nil, errNotImplemented
}

// AddApplicationBranch implements PortfoliosClient.AddApplicationBranch.
func (m *MockPortfoliosClient) AddApplicationBranch(_ context.Context, opt *sonar.ViewsAddApplicationBranchOptions) (*http.Response, error) {
	if m.AddApplicationBranchFn != nil {
		return m.AddApplicationBranchFn(opt)
	}

	return nil, errNotImplemented
}

// AddPortfolio implements PortfoliosClient.AddPortfolio.
func (m *MockPortfoliosClient) AddPortfolio(_ context.Context, opt *sonar.ViewsAddPortfolioOptions) (*http.Response, error) {
	if m.AddPortfolioFn != nil {
		return m.AddPortfolioFn(opt)
	}

	return nil, errNotImplemented
}

// AddProject implements PortfoliosClient.AddProject.
func (m *MockPortfoliosClient) AddProject(_ context.Context, opt *sonar.ViewsAddProjectOptions) (*http.Response, error) {
	if m.AddProjectFn != nil {
		return m.AddProjectFn(opt)
	}

	return nil, errNotImplemented
}

// AddProjectBranch implements PortfoliosClient.AddProjectBranch.
func (m *MockPortfoliosClient) AddProjectBranch(_ context.Context, opt *sonar.ViewsAddProjectBranchOptions) (*http.Response, error) {
	if m.AddProjectBranchFn != nil {
		return m.AddProjectBranchFn(opt)
	}

	return nil, errNotImplemented
}

// Applications implements PortfoliosClient.Applications.
func (m *MockPortfoliosClient) Applications(_ context.Context, opt *sonar.ViewsApplicationsOptions) (*sonar.ViewsApplications, *http.Response, error) {
	if m.ApplicationsFn != nil {
		return m.ApplicationsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Create implements PortfoliosClient.Create.
func (m *MockPortfoliosClient) Create(_ context.Context, opt *sonar.ViewsCreateOptions) (*http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, errNotImplemented
}

// Delete implements PortfoliosClient.Delete.
func (m *MockPortfoliosClient) Delete(_ context.Context, opt *sonar.ViewsDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// List implements PortfoliosClient.List.
func (m *MockPortfoliosClient) List(_ context.Context) (*sonar.ViewsList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn()
	}

	return nil, nil, errNotImplemented
}

// Move implements PortfoliosClient.Move.
func (m *MockPortfoliosClient) Move(_ context.Context, opt *sonar.ViewsMoveOptions) (*http.Response, error) {
	if m.MoveFn != nil {
		return m.MoveFn(opt)
	}

	return nil, errNotImplemented
}

// MoveOptions implements PortfoliosClient.MoveOptions.
func (m *MockPortfoliosClient) MoveOptions(_ context.Context, opt *sonar.ViewsMoveOptionsOptions) (*sonar.ViewsMoveDestinations, *http.Response, error) {
	if m.MoveOptionsFn != nil {
		return m.MoveOptionsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Projects implements PortfoliosClient.Projects.
func (m *MockPortfoliosClient) Projects(_ context.Context, opt *sonar.ViewsProjectsOptions) (*sonar.ViewsProjects, *http.Response, error) {
	if m.ProjectsFn != nil {
		return m.ProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ProjectsStatus implements PortfoliosClient.ProjectsStatus.
func (m *MockPortfoliosClient) ProjectsStatus(_ context.Context, opt *sonar.ViewsProjectsStatusOptions) (*sonar.ViewsProjectsStatus, *http.Response, error) {
	if m.ProjectsStatusFn != nil {
		return m.ProjectsStatusFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Refresh implements PortfoliosClient.Refresh.
func (m *MockPortfoliosClient) Refresh(_ context.Context, opt *sonar.ViewsRefreshOptions) (*http.Response, error) {
	if m.RefreshFn != nil {
		return m.RefreshFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveApplication implements PortfoliosClient.RemoveApplication.
func (m *MockPortfoliosClient) RemoveApplication(_ context.Context, opt *sonar.ViewsRemoveApplicationOptions) (*http.Response, error) {
	if m.RemoveApplicationFn != nil {
		return m.RemoveApplicationFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveApplicationBranch implements PortfoliosClient.RemoveApplicationBranch.
func (m *MockPortfoliosClient) RemoveApplicationBranch(_ context.Context, opt *sonar.ViewsRemoveApplicationBranchOptions) (*http.Response, error) {
	if m.RemoveApplicationBranchFn != nil {
		return m.RemoveApplicationBranchFn(opt)
	}

	return nil, errNotImplemented
}

// RemovePortfolio implements PortfoliosClient.RemovePortfolio.
func (m *MockPortfoliosClient) RemovePortfolio(_ context.Context, opt *sonar.ViewsRemovePortfolioOptions) (*http.Response, error) {
	if m.RemovePortfolioFn != nil {
		return m.RemovePortfolioFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveProject implements PortfoliosClient.RemoveProject.
func (m *MockPortfoliosClient) RemoveProject(_ context.Context, opt *sonar.ViewsRemoveProjectOptions) (*http.Response, error) {
	if m.RemoveProjectFn != nil {
		return m.RemoveProjectFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveProjectBranch implements PortfoliosClient.RemoveProjectBranch.
func (m *MockPortfoliosClient) RemoveProjectBranch(_ context.Context, opt *sonar.ViewsRemoveProjectBranchOptions) (*http.Response, error) {
	if m.RemoveProjectBranchFn != nil {
		return m.RemoveProjectBranchFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements PortfoliosClient.Search.
func (m *MockPortfoliosClient) Search(_ context.Context, opt *sonar.ViewsSearchOptions) (*sonar.ViewsSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetManualMode implements PortfoliosClient.SetManualMode.
func (m *MockPortfoliosClient) SetManualMode(_ context.Context, opt *sonar.ViewsSetManualModeOptions) (*http.Response, error) {
	if m.SetManualModeFn != nil {
		return m.SetManualModeFn(opt)
	}

	return nil, errNotImplemented
}

// SetNoneMode implements PortfoliosClient.SetNoneMode.
func (m *MockPortfoliosClient) SetNoneMode(_ context.Context, opt *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
	if m.SetNoneModeFn != nil {
		return m.SetNoneModeFn(opt)
	}

	return nil, errNotImplemented
}

// SetRegexpMode implements PortfoliosClient.SetRegexpMode.
func (m *MockPortfoliosClient) SetRegexpMode(_ context.Context, opt *sonar.ViewsSetRegexpModeOptions) (*http.Response, error) {
	if m.SetRegexpModeFn != nil {
		return m.SetRegexpModeFn(opt)
	}

	return nil, errNotImplemented
}

// SetRemainingProjectsMode implements
// PortfoliosClient.SetRemainingProjectsMode.
func (m *MockPortfoliosClient) SetRemainingProjectsMode(_ context.Context, opt *sonar.ViewsSetRemainingProjectsModeOptions) (*http.Response, error) {
	if m.SetRemainingProjectsModeFn != nil {
		return m.SetRemainingProjectsModeFn(opt)
	}

	return nil, errNotImplemented
}

// SetTagsMode implements PortfoliosClient.SetTagsMode.
func (m *MockPortfoliosClient) SetTagsMode(_ context.Context, opt *sonar.ViewsSetTagsModeOptions) (*http.Response, error) {
	if m.SetTagsModeFn != nil {
		return m.SetTagsModeFn(opt)
	}

	return nil, errNotImplemented
}

// Show implements PortfoliosClient.Show.
func (m *MockPortfoliosClient) Show(_ context.Context, opt *sonar.ViewsShowOptions) (*sonar.ViewsShow, *http.Response, error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SubPortfolios implements PortfoliosClient.SubPortfolios.
func (m *MockPortfoliosClient) SubPortfolios(_ context.Context, opt *sonar.ViewsSubViewsOptions) (*sonar.ViewsSubViews, *http.Response, error) {
	if m.SubPortfoliosFn != nil {
		return m.SubPortfoliosFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Update implements PortfoliosClient.Update.
func (m *MockPortfoliosClient) Update(_ context.Context, opt *sonar.ViewsUpdateOptions) (*http.Response, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}

	return nil, errNotImplemented
}
