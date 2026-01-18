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

// Package fake provides mock implementations for testing.
package fake

import (
	"net/http"

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockQualityGatesClient is a mock implementation of the QualityGatesClient interface.
type MockQualityGatesClient struct {
	AddGroupFn        func(opt *sonargo.QualitygatesAddGroupOption) (resp *http.Response, err error)
	AddUserFn         func(opt *sonargo.QualitygatesAddUserOption) (resp *http.Response, err error)
	CopyFn            func(opt *sonargo.QualitygatesCopyOption) (resp *http.Response, err error)
	CreateFn          func(opt *sonargo.QualitygatesCreateOption) (v *sonargo.QualitygatesCreateObject, resp *http.Response, err error)
	CreateConditionFn func(opt *sonargo.QualitygatesCreateConditionOption) (v *sonargo.QualitygatesCreateConditionObject, resp *http.Response, err error)
	DeleteConditionFn func(opt *sonargo.QualitygatesDeleteConditionOption) (resp *http.Response, err error)
	DeselectFn        func(opt *sonargo.QualitygatesDeselectOption) (resp *http.Response, err error)
	DestroyFn         func(opt *sonargo.QualitygatesDestroyOption) (resp *http.Response, err error)
	GetByProjectFn    func(opt *sonargo.QualitygatesGetByProjectOption) (v *sonargo.QualitygatesGetByProjectObject, resp *http.Response, err error)
	ListFn            func() (v *sonargo.QualitygatesListObject, resp *http.Response, err error)
	ProjectStatusFn   func(opt *sonargo.QualitygatesProjectStatusOption) (v *sonargo.QualitygatesProjectStatusObject, resp *http.Response, err error)
	RemoveGroupFn     func(opt *sonargo.QualitygatesRemoveGroupOption) (resp *http.Response, err error)
	RemoveUserFn      func(opt *sonargo.QualitygatesRemoveUserOption) (resp *http.Response, err error)
	RenameFn          func(opt *sonargo.QualitygatesRenameOption) (resp *http.Response, err error)
	SearchFn          func(opt *sonargo.QualitygatesSearchOption) (v *sonargo.QualitygatesSearchObject, resp *http.Response, err error)
	SearchGroupsFn    func(opt *sonargo.QualitygatesSearchGroupsOption) (v *sonargo.QualitygatesSearchGroupsObject, resp *http.Response, err error)
	SearchUsersFn     func(opt *sonargo.QualitygatesSearchUsersOption) (v *sonargo.QualitygatesSearchUsersObject, resp *http.Response, err error)
	SelectFn          func(opt *sonargo.QualitygatesSelectOption) (resp *http.Response, err error)
	SetAsDefaultFn    func(opt *sonargo.QualitygatesSetAsDefaultOption) (resp *http.Response, err error)
	ShowFn            func(opt *sonargo.QualitygatesShowOption) (v *sonargo.QualitygatesShowObject, resp *http.Response, err error)
	UpdateConditionFn func(opt *sonargo.QualitygatesUpdateConditionOption) (resp *http.Response, err error)
}

// Ensure MockQualityGatesClient implements QualityGatesClient
var _ instance.QualityGatesClient = &MockQualityGatesClient{}

// AddGroup implements QualityGatesClient.AddGroup
func (m *MockQualityGatesClient) AddGroup(opt *sonargo.QualitygatesAddGroupOption) (resp *http.Response, err error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}
	return nil, nil
}

// AddUser implements QualityGatesClient.AddUser
func (m *MockQualityGatesClient) AddUser(opt *sonargo.QualitygatesAddUserOption) (resp *http.Response, err error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}
	return nil, nil
}

// Copy implements QualityGatesClient.Copy
func (m *MockQualityGatesClient) Copy(opt *sonargo.QualitygatesCopyOption) (resp *http.Response, err error) {
	if m.CopyFn != nil {
		return m.CopyFn(opt)
	}
	return nil, nil
}

// Create implements QualityGatesClient.Create
func (m *MockQualityGatesClient) Create(opt *sonargo.QualitygatesCreateOption) (v *sonargo.QualitygatesCreateObject, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}
	return nil, nil, nil
}

// CreateCondition implements QualityGatesClient.CreateCondition
func (m *MockQualityGatesClient) CreateCondition(opt *sonargo.QualitygatesCreateConditionOption) (v *sonargo.QualitygatesCreateConditionObject, resp *http.Response, err error) {
	if m.CreateConditionFn != nil {
		return m.CreateConditionFn(opt)
	}
	return nil, nil, nil
}

// DeleteCondition implements QualityGatesClient.DeleteCondition
func (m *MockQualityGatesClient) DeleteCondition(opt *sonargo.QualitygatesDeleteConditionOption) (resp *http.Response, err error) {
	if m.DeleteConditionFn != nil {
		return m.DeleteConditionFn(opt)
	}
	return nil, nil
}

// Deselect implements QualityGatesClient.Deselect
func (m *MockQualityGatesClient) Deselect(opt *sonargo.QualitygatesDeselectOption) (resp *http.Response, err error) {
	if m.DeselectFn != nil {
		return m.DeselectFn(opt)
	}
	return nil, nil
}

// Destroy implements QualityGatesClient.Destroy
func (m *MockQualityGatesClient) Destroy(opt *sonargo.QualitygatesDestroyOption) (resp *http.Response, err error) {
	if m.DestroyFn != nil {
		return m.DestroyFn(opt)
	}
	return nil, nil
}

// GetByProject implements QualityGatesClient.GetByProject
func (m *MockQualityGatesClient) GetByProject(opt *sonargo.QualitygatesGetByProjectOption) (v *sonargo.QualitygatesGetByProjectObject, resp *http.Response, err error) {
	if m.GetByProjectFn != nil {
		return m.GetByProjectFn(opt)
	}
	return nil, nil, nil
}

// List implements QualityGatesClient.List
func (m *MockQualityGatesClient) List() (v *sonargo.QualitygatesListObject, resp *http.Response, err error) {
	if m.ListFn != nil {
		return m.ListFn()
	}
	return nil, nil, nil
}

// ProjectStatus implements QualityGatesClient.ProjectStatus
func (m *MockQualityGatesClient) ProjectStatus(opt *sonargo.QualitygatesProjectStatusOption) (v *sonargo.QualitygatesProjectStatusObject, resp *http.Response, err error) {
	if m.ProjectStatusFn != nil {
		return m.ProjectStatusFn(opt)
	}
	return nil, nil, nil
}

// RemoveGroup implements QualityGatesClient.RemoveGroup
func (m *MockQualityGatesClient) RemoveGroup(opt *sonargo.QualitygatesRemoveGroupOption) (resp *http.Response, err error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}
	return nil, nil
}

// RemoveUser implements QualityGatesClient.RemoveUser
func (m *MockQualityGatesClient) RemoveUser(opt *sonargo.QualitygatesRemoveUserOption) (resp *http.Response, err error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}
	return nil, nil
}

// Rename implements QualityGatesClient.Rename
func (m *MockQualityGatesClient) Rename(opt *sonargo.QualitygatesRenameOption) (resp *http.Response, err error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}
	return nil, nil
}

// Search implements QualityGatesClient.Search
func (m *MockQualityGatesClient) Search(opt *sonargo.QualitygatesSearchOption) (v *sonargo.QualitygatesSearchObject, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}
	return nil, nil, nil
}

// SearchGroups implements QualityGatesClient.SearchGroups
func (m *MockQualityGatesClient) SearchGroups(opt *sonargo.QualitygatesSearchGroupsOption) (v *sonargo.QualitygatesSearchGroupsObject, resp *http.Response, err error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}
	return nil, nil, nil
}

// SearchUsers implements QualityGatesClient.SearchUsers
func (m *MockQualityGatesClient) SearchUsers(opt *sonargo.QualitygatesSearchUsersOption) (v *sonargo.QualitygatesSearchUsersObject, resp *http.Response, err error) {
	if m.SearchUsersFn != nil {
		return m.SearchUsersFn(opt)
	}
	return nil, nil, nil
}

// Select implements QualityGatesClient.Select
func (m *MockQualityGatesClient) Select(opt *sonargo.QualitygatesSelectOption) (resp *http.Response, err error) {
	if m.SelectFn != nil {
		return m.SelectFn(opt)
	}
	return nil, nil
}

// SetAsDefault implements QualityGatesClient.SetAsDefault
func (m *MockQualityGatesClient) SetAsDefault(opt *sonargo.QualitygatesSetAsDefaultOption) (resp *http.Response, err error) {
	if m.SetAsDefaultFn != nil {
		return m.SetAsDefaultFn(opt)
	}
	return nil, nil
}

// Show implements QualityGatesClient.Show
func (m *MockQualityGatesClient) Show(opt *sonargo.QualitygatesShowOption) (v *sonargo.QualitygatesShowObject, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}
	return nil, nil, nil
}

// UpdateCondition implements QualityGatesClient.UpdateCondition
func (m *MockQualityGatesClient) UpdateCondition(opt *sonargo.QualitygatesUpdateConditionOption) (resp *http.Response, err error) {
	if m.UpdateConditionFn != nil {
		return m.UpdateConditionFn(opt)
	}
	return nil, nil
}
