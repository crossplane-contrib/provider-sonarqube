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

// MockGroupsClient is a mock implementation of the GroupsClient interface.
type MockGroupsClient struct {
	CreateGroupFn            func(opt *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error)
	CreateGroupMembershipFn  func(opt *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error)
	DeleteGroupFn            func(groupID string) (*http.Response, error)
	DeleteGroupMembershipFn  func(membershipID string) (*http.Response, error)
	FetchGroupFn             func(groupID string) (*sonar.Group, *http.Response, error)
	SearchGroupMembershipsFn func(opt *sonar.AuthorizationsSearchGroupMembershipsOptions) (*sonar.AuthorizationsGroupMembershipsSearch, *http.Response, error)
	SearchGroupsFn           func(opt *sonar.AuthorizationsSearchGroupsOptions) (*sonar.AuthorizationsGroupsSearch, *http.Response, error)
	UpdateGroupFn            func(groupID string, opt *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error)
}

// Ensure MockGroupsClient implements GroupsClient.
var _ iam.GroupsClient = &MockGroupsClient{}

// CreateGroup implements GroupsClient.CreateGroup.
func (m *MockGroupsClient) CreateGroup(opt *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
	if m.CreateGroupFn != nil {
		return m.CreateGroupFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateGroupMembership implements GroupsClient.CreateGroupMembership.
func (m *MockGroupsClient) CreateGroupMembership(opt *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error) {
	if m.CreateGroupMembershipFn != nil {
		return m.CreateGroupMembershipFn(opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteGroup implements GroupsClient.DeleteGroup.
func (m *MockGroupsClient) DeleteGroup(groupID string) (*http.Response, error) {
	if m.DeleteGroupFn != nil {
		return m.DeleteGroupFn(groupID)
	}

	return nil, errNotImplemented
}

// DeleteGroupMembership implements GroupsClient.DeleteGroupMembership.
func (m *MockGroupsClient) DeleteGroupMembership(membershipID string) (*http.Response, error) {
	if m.DeleteGroupMembershipFn != nil {
		return m.DeleteGroupMembershipFn(membershipID)
	}

	return nil, errNotImplemented
}

// FetchGroup implements GroupsClient.FetchGroup.
func (m *MockGroupsClient) FetchGroup(groupID string) (*sonar.Group, *http.Response, error) {
	if m.FetchGroupFn != nil {
		return m.FetchGroupFn(groupID)
	}

	return nil, nil, errNotImplemented
}

// SearchGroupMemberships implements GroupsClient.SearchGroupMemberships.
func (m *MockGroupsClient) SearchGroupMemberships(opt *sonar.AuthorizationsSearchGroupMembershipsOptions) (*sonar.AuthorizationsGroupMembershipsSearch, *http.Response, error) {
	if m.SearchGroupMembershipsFn != nil {
		return m.SearchGroupMembershipsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchGroups implements GroupsClient.SearchGroups.
func (m *MockGroupsClient) SearchGroups(opt *sonar.AuthorizationsSearchGroupsOptions) (*sonar.AuthorizationsGroupsSearch, *http.Response, error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateGroup implements GroupsClient.UpdateGroup.
func (m *MockGroupsClient) UpdateGroup(groupID string, opt *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error) {
	if m.UpdateGroupFn != nil {
		return m.UpdateGroupFn(groupID, opt)
	}

	return nil, nil, errNotImplemented
}
