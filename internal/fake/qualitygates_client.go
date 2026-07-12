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
	"errors"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// errNotImplemented indicates function is not implemented.
var errNotImplemented = errors.New("mock function not implemented")

// MockQualityGatesClient is a mock implementation of the
// QualityGatesClient interface.
type MockQualityGatesClient struct {
	AddGroupFn        func(opt *sonar.QualitygatesAddGroupOptions) (resp *http.Response, err error)
	AddUserFn         func(opt *sonar.QualitygatesAddUserOptions) (resp *http.Response, err error)
	CopyFn            func(opt *sonar.QualitygatesCopyOptions) (resp *http.Response, err error)
	CreateFn          func(opt *sonar.QualitygatesCreateOptions) (v *sonar.QualitygatesCreate, resp *http.Response, err error)
	CreateConditionFn func(opt *sonar.QualitygatesCreateConditionOptions) (v *sonar.QualitygatesCreateCondition, resp *http.Response, err error)
	DeleteConditionFn func(opt *sonar.QualitygatesDeleteConditionOptions) (resp *http.Response, err error)
	UnassignFn        func(opt *sonar.QualitygatesUnassignOptions) (resp *http.Response, err error)
	DeleteFn          func(opt *sonar.QualitygatesDeleteOptions) (resp *http.Response, err error)
	GetByProjectFn    func(opt *sonar.QualitygatesGetByProjectOptions) (v *sonar.QualitygatesGetByProject, resp *http.Response, err error)
	ListFn            func() (v *sonar.QualitygatesList, resp *http.Response, err error)
	ProjectStatusFn   func(opt *sonar.QualitygatesProjectStatusOptions) (v *sonar.QualitygatesProjectStatus, resp *http.Response, err error)
	RemoveGroupFn     func(opt *sonar.QualitygatesRemoveGroupOptions) (resp *http.Response, err error)
	RemoveUserFn      func(opt *sonar.QualitygatesRemoveUserOptions) (resp *http.Response, err error)
	RenameFn          func(opt *sonar.QualitygatesRenameOptions) (resp *http.Response, err error)
	SearchFn          func(opt *sonar.QualitygatesSearchOptions) (v *sonar.QualitygatesSearch, resp *http.Response, err error)
	SearchGroupsFn    func(opt *sonar.QualitygatesSearchGroupsOptions) (v *sonar.QualitygatesSearchGroups, resp *http.Response, err error)
	SearchUsersFn     func(opt *sonar.QualitygatesSearchUsersOptions) (v *sonar.QualitygatesSearchUsers, resp *http.Response, err error)
	AssignFn          func(opt *sonar.QualitygatesAssignOptions) (resp *http.Response, err error)
	SetDefaultFn      func(opt *sonar.QualitygatesSetDefaultOptions) (resp *http.Response, err error)
	ShowFn            func(opt *sonar.QualitygatesShowOptions) (v *sonar.QualitygatesShow, resp *http.Response, err error)
	UpdateConditionFn func(opt *sonar.QualitygatesUpdateConditionOptions) (resp *http.Response, err error)
}

// Ensure MockQualityGatesClient implements QualityGatesClient.
var _ instance.QualityGatesClient = &MockQualityGatesClient{}

// AddGroup implements QualityGatesClient.AddGroup.
func (m *MockQualityGatesClient) AddGroup(_ context.Context, opt *sonar.QualitygatesAddGroupOptions) (resp *http.Response, err error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}

	return nil, errNotImplemented
}

// AddUser implements QualityGatesClient.AddUser.
func (m *MockQualityGatesClient) AddUser(_ context.Context, opt *sonar.QualitygatesAddUserOptions) (resp *http.Response, err error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}

	return nil, errNotImplemented
}

// Copy implements QualityGatesClient.Copy.
func (m *MockQualityGatesClient) Copy(_ context.Context, opt *sonar.QualitygatesCopyOptions) (resp *http.Response, err error) {
	if m.CopyFn != nil {
		return m.CopyFn(opt)
	}

	return nil, errNotImplemented
}

// Create implements QualityGatesClient.Create.
func (m *MockQualityGatesClient) Create(_ context.Context, opt *sonar.QualitygatesCreateOptions) (v *sonar.QualitygatesCreate, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateCondition implements QualityGatesClient.CreateCondition.
func (m *MockQualityGatesClient) CreateCondition(_ context.Context, opt *sonar.QualitygatesCreateConditionOptions) (v *sonar.QualitygatesCreateCondition, resp *http.Response, err error) {
	if m.CreateConditionFn != nil {
		return m.CreateConditionFn(opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteCondition implements QualityGatesClient.DeleteCondition.
func (m *MockQualityGatesClient) DeleteCondition(_ context.Context, opt *sonar.QualitygatesDeleteConditionOptions) (resp *http.Response, err error) {
	if m.DeleteConditionFn != nil {
		return m.DeleteConditionFn(opt)
	}

	return nil, errNotImplemented
}

// Unassign implements QualityGatesClient.Unassign.
func (m *MockQualityGatesClient) Unassign(_ context.Context, opt *sonar.QualitygatesUnassignOptions) (resp *http.Response, err error) {
	if m.UnassignFn != nil {
		return m.UnassignFn(opt)
	}

	return nil, errNotImplemented
}

// Delete implements QualityGatesClient.Delete.
func (m *MockQualityGatesClient) Delete(_ context.Context, opt *sonar.QualitygatesDeleteOptions) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// GetByProject implements QualityGatesClient.GetByProject.
func (m *MockQualityGatesClient) GetByProject(_ context.Context, opt *sonar.QualitygatesGetByProjectOptions) (v *sonar.QualitygatesGetByProject, resp *http.Response, err error) {
	if m.GetByProjectFn != nil {
		return m.GetByProjectFn(opt)
	}

	return nil, nil, errNotImplemented
}

// List implements QualityGatesClient.List.
func (m *MockQualityGatesClient) List(_ context.Context) (v *sonar.QualitygatesList, resp *http.Response, err error) {
	if m.ListFn != nil {
		return m.ListFn()
	}

	return nil, nil, errNotImplemented
}

// ProjectStatus implements QualityGatesClient.ProjectStatus.
func (m *MockQualityGatesClient) ProjectStatus(_ context.Context, opt *sonar.QualitygatesProjectStatusOptions) (v *sonar.QualitygatesProjectStatus, resp *http.Response, err error) {
	if m.ProjectStatusFn != nil {
		return m.ProjectStatusFn(opt)
	}

	return nil, nil, errNotImplemented
}

// RemoveGroup implements QualityGatesClient.RemoveGroup.
func (m *MockQualityGatesClient) RemoveGroup(_ context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (resp *http.Response, err error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveUser implements QualityGatesClient.RemoveUser.
func (m *MockQualityGatesClient) RemoveUser(_ context.Context, opt *sonar.QualitygatesRemoveUserOptions) (resp *http.Response, err error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}

	return nil, errNotImplemented
}

// Rename implements QualityGatesClient.Rename.
func (m *MockQualityGatesClient) Rename(_ context.Context, opt *sonar.QualitygatesRenameOptions) (resp *http.Response, err error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements QualityGatesClient.Search.
func (m *MockQualityGatesClient) Search(_ context.Context, opt *sonar.QualitygatesSearchOptions) (v *sonar.QualitygatesSearch, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchGroups implements QualityGatesClient.SearchGroups.
func (m *MockQualityGatesClient) SearchGroups(_ context.Context, opt *sonar.QualitygatesSearchGroupsOptions) (v *sonar.QualitygatesSearchGroups, resp *http.Response, err error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchUsers implements QualityGatesClient.SearchUsers.
func (m *MockQualityGatesClient) SearchUsers(_ context.Context, opt *sonar.QualitygatesSearchUsersOptions) (v *sonar.QualitygatesSearchUsers, resp *http.Response, err error) {
	if m.SearchUsersFn != nil {
		return m.SearchUsersFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Assign implements QualityGatesClient.Assign.
func (m *MockQualityGatesClient) Assign(_ context.Context, opt *sonar.QualitygatesAssignOptions) (resp *http.Response, err error) {
	if m.AssignFn != nil {
		return m.AssignFn(opt)
	}

	return nil, errNotImplemented
}

// SetDefault implements QualityGatesClient.SetDefault.
func (m *MockQualityGatesClient) SetDefault(_ context.Context, opt *sonar.QualitygatesSetDefaultOptions) (resp *http.Response, err error) {
	if m.SetDefaultFn != nil {
		return m.SetDefaultFn(opt)
	}

	return nil, errNotImplemented
}

// Show implements QualityGatesClient.Show.
func (m *MockQualityGatesClient) Show(_ context.Context, opt *sonar.QualitygatesShowOptions) (v *sonar.QualitygatesShow, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateCondition implements QualityGatesClient.UpdateCondition.
func (m *MockQualityGatesClient) UpdateCondition(_ context.Context, opt *sonar.QualitygatesUpdateConditionOptions) (resp *http.Response, err error) {
	if m.UpdateConditionFn != nil {
		return m.UpdateConditionFn(opt)
	}

	return nil, errNotImplemented
}
