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

// MockPermissionsTemplatesClient is a mock implementation of the iam.PermissionsTemplatesClient interface.
type MockPermissionsTemplatesClient struct {
	AddGroupFn             func(*sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error)
	AddProjectCreatorFn    func(*sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error)
	AddUserFn              func(*sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error)
	CreateFn               func(*sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error)
	DeleteFn               func(*sonar.PermissionsDeleteTemplateOptions) (*http.Response, error)
	RemoveGroupFn          func(*sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error)
	RemoveProjectCreatorFn func(*sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error)
	RemoveUserFn           func(*sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error)
	SearchFn               func(*sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error)
	SetDefaultFn           func(*sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error)
	TemplateGroupsFn       func(*sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error)
	TemplateUsersFn        func(*sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error)
	UpdateTemplateFn       func(*sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error)
}

// Ensure MockPermissionsTemplatesClient implements PermissionsTemplatesClient.
var _ iam.PermissionsTemplatesClient = &MockPermissionsTemplatesClient{}

// AddGroupToTemplate implements PermissionsTemplatesClient.AddGroupToTemplate.
func (m *MockPermissionsTemplatesClient) AddGroupToTemplate(opt *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}

	return nil, errNotImplemented
}

// AddProjectCreatorToTemplate implements PermissionsTemplatesClient.AddProjectCreatorToTemplate.
func (m *MockPermissionsTemplatesClient) AddProjectCreatorToTemplate(opt *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error) {
	if m.AddProjectCreatorFn != nil {
		return m.AddProjectCreatorFn(opt)
	}

	return nil, errNotImplemented
}

// AddUserToTemplate implements PermissionsTemplatesClient.AddUserToTemplate.
func (m *MockPermissionsTemplatesClient) AddUserToTemplate(opt *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}

	return nil, errNotImplemented
}

// CreateTemplate implements PermissionsTemplatesClient.CreateTemplate.
func (m *MockPermissionsTemplatesClient) CreateTemplate(opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteTemplate implements PermissionsTemplatesClient.DeleteTemplate.
func (m *MockPermissionsTemplatesClient) DeleteTemplate(opt *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveGroupFromTemplate implements PermissionsTemplatesClient.RemoveGroupFromTemplate.
func (m *MockPermissionsTemplatesClient) RemoveGroupFromTemplate(opt *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveProjectCreatorFromTemplate implements PermissionsTemplatesClient.RemoveProjectCreatorFromTemplate.
func (m *MockPermissionsTemplatesClient) RemoveProjectCreatorFromTemplate(opt *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error) {
	if m.RemoveProjectCreatorFn != nil {
		return m.RemoveProjectCreatorFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveUserFromTemplate implements PermissionsTemplatesClient.RemoveUserFromTemplate.
func (m *MockPermissionsTemplatesClient) RemoveUserFromTemplate(opt *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}

	return nil, errNotImplemented
}

// SearchTemplates implements PermissionsTemplatesClient.SearchTemplates.
func (m *MockPermissionsTemplatesClient) SearchTemplates(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetDefaultTemplate implements PermissionsTemplatesClient.SetDefaultTemplate.
func (m *MockPermissionsTemplatesClient) SetDefaultTemplate(opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error) {
	if m.SetDefaultFn != nil {
		return m.SetDefaultFn(opt)
	}

	return nil, errNotImplemented
}

// TemplateGroups implements PermissionsTemplatesClient.TemplateGroups.
func (m *MockPermissionsTemplatesClient) TemplateGroups(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
	if m.TemplateGroupsFn != nil {
		return m.TemplateGroupsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// TemplateUsers implements PermissionsTemplatesClient.TemplateUsers.
func (m *MockPermissionsTemplatesClient) TemplateUsers(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
	if m.TemplateUsersFn != nil {
		return m.TemplateUsersFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateTemplate implements PermissionsTemplatesClient.UpdateTemplate.
func (m *MockPermissionsTemplatesClient) UpdateTemplate(opt *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error) {
	if m.UpdateTemplateFn != nil {
		return m.UpdateTemplateFn(opt)
	}

	return nil, nil, errNotImplemented
}
