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

	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
)

// defaultPermissionsPageSize is the default page size for permissions.
const defaultPermissionsPageSize = 100

// MockPermissionsClient is a mock implementation of the
// iam.PermissionsClient interface.
type MockPermissionsClient struct {
	AddGroupFn    func(*sonar.PermissionsAddGroupOptions) (*http.Response, error)
	GroupsFn      func(*sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error)
	RemoveGroupFn func(*sonar.PermissionsRemoveGroupOptions) (*http.Response, error)
	AddUserFn     func(*sonar.PermissionsAddUserOptions) (*http.Response, error)
	UsersFn       func(*sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error)
	RemoveUserFn  func(*sonar.PermissionsRemoveUserOptions) (*http.Response, error)
}

// AddGroup calls the mock AddGroupFn if set, otherwise returns an error.
func (m *MockPermissionsClient) AddGroup(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}

	return nil, errNotImplemented
}

// Groups calls the mock GroupsFn if set, otherwise returns an empty result.
func (m *MockPermissionsClient) Groups(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
	if m.GroupsFn != nil {
		return m.GroupsFn(opt)
	}

	// Return empty permissions groups by default for testing convenience
	return &sonar.PermissionsGroups{Groups: []sonar.PermissionGroup{}, Paging: sonar.PermissionsPaging{Total: 0, PageIndex: 1, PageSize: defaultPermissionsPageSize}}, &http.Response{StatusCode: http.StatusOK}, nil
}

// RemoveGroup calls the mock RemoveGroupFn if set, otherwise returns an error.
func (m *MockPermissionsClient) RemoveGroup(opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}

	return nil, errNotImplemented
}

// AddUser calls the mock AddUserFn if set, otherwise returns an error.
func (m *MockPermissionsClient) AddUser(opt *sonar.PermissionsAddUserOptions) (*http.Response, error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}

	return nil, errNotImplemented
}

// Users calls the mock UsersFn if set, otherwise returns an empty result.
func (m *MockPermissionsClient) Users(opt *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
	if m.UsersFn != nil {
		return m.UsersFn(opt)
	}

	// Return empty permissions users by default for testing convenience
	return &sonar.PermissionsUsers{Users: []sonar.PermissionUser{}, Paging: sonar.PermissionsPaging{Total: 0, PageIndex: 1, PageSize: defaultPermissionsPageSize}}, &http.Response{StatusCode: http.StatusOK}, nil
}

// RemoveUser calls the mock RemoveUserFn if set, otherwise returns an error.
func (m *MockPermissionsClient) RemoveUser(opt *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}

	return nil, errNotImplemented
}

// Ensure MockPermissionsClient implements the PermissionsClient interface.
var _ iam.PermissionsClient = &MockPermissionsClient{}
