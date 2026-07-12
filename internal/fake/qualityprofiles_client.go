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

// MockQualityProfilesClient is a mock implementation of the
// QualityProfilesClient interface.
type MockQualityProfilesClient struct {
	ActivateRuleFn    func(opt *sonar.QualityprofilesActivateRuleOptions) (resp *http.Response, err error)
	ActivateRulesFn   func(opt *sonar.QualityprofilesActivateRulesOptions) (resp *http.Response, err error)
	AddGroupFn        func(opt *sonar.QualityprofilesAddGroupOptions) (resp *http.Response, err error)
	AddProjectFn      func(opt *sonar.QualityprofilesAddProjectOptions) (resp *http.Response, err error)
	AddUserFn         func(opt *sonar.QualityprofilesAddUserOptions) (resp *http.Response, err error)
	BackupFn          func(opt *sonar.QualityprofilesBackupOptions) (v *string, resp *http.Response, err error)
	ChangeParentFn    func(opt *sonar.QualityprofilesChangeParentOptions) (resp *http.Response, err error)
	ChangelogFn       func(opt *sonar.QualityprofilesChangelogOptions) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error)
	CompareFn         func(opt *sonar.QualityprofilesCompareOptions) (v *sonar.QualityprofilesCompare, resp *http.Response, err error)
	CopyFn            func(opt *sonar.QualityprofilesCopyOptions) (v *sonar.QualityprofilesCopy, resp *http.Response, err error)
	CreateFn          func(opt *sonar.QualityprofilesCreateOptions) (v *sonar.QualityprofilesCreate, resp *http.Response, err error)
	DeactivateRuleFn  func(opt *sonar.QualityprofilesDeactivateRuleOptions) (resp *http.Response, err error)
	DeactivateRulesFn func(opt *sonar.QualityprofilesDeactivateRulesOptions) (resp *http.Response, err error)
	DeleteFn          func(opt *sonar.QualityprofilesDeleteOptions) (resp *http.Response, err error)
	InheritanceFn     func(opt *sonar.QualityprofilesInheritanceOptions) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error)
	ProjectsFn        func(opt *sonar.QualityprofilesProjectsOptions) (v *sonar.QualityprofilesProjects, resp *http.Response, err error)
	RemoveGroupFn     func(opt *sonar.QualityprofilesRemoveGroupOptions) (resp *http.Response, err error)
	RemoveProjectFn   func(opt *sonar.QualityprofilesRemoveProjectOptions) (resp *http.Response, err error)
	RemoveUserFn      func(opt *sonar.QualityprofilesRemoveUserOptions) (resp *http.Response, err error)
	RenameFn          func(opt *sonar.QualityprofilesRenameOptions) (resp *http.Response, err error)
	RestoreFn         func(opt *sonar.QualityprofilesRestoreOptions) (resp *http.Response, err error)
	SearchFn          func(opt *sonar.QualityprofilesSearchOptions) (v *sonar.QualityprofilesSearch, resp *http.Response, err error)
	SearchGroupsFn    func(opt *sonar.QualityprofilesSearchGroupsOptions) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error)
	SearchUsersFn     func(opt *sonar.QualityprofilesSearchUsersOptions) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error)
	SetDefaultFn      func(opt *sonar.QualityprofilesSetDefaultOptions) (resp *http.Response, err error)
	ShowFn            func(opt *sonar.QualityprofilesShowOptions) (v *sonar.QualityprofilesShow, resp *http.Response, err error)
}

// Ensure MockQualityProfilesClient implements QualityProfilesClient.
var _ instance.QualityProfilesClient = &MockQualityProfilesClient{}

// ActivateRule implements QualityProfilesClient.ActivateRule.
func (m *MockQualityProfilesClient) ActivateRule(_ context.Context, opt *sonar.QualityprofilesActivateRuleOptions) (resp *http.Response, err error) {
	if m.ActivateRuleFn != nil {
		return m.ActivateRuleFn(opt)
	}

	return nil, errNotImplemented
}

// ActivateRules implements QualityProfilesClient.ActivateRules.
func (m *MockQualityProfilesClient) ActivateRules(_ context.Context, opt *sonar.QualityprofilesActivateRulesOptions) (resp *http.Response, err error) {
	if m.ActivateRulesFn != nil {
		return m.ActivateRulesFn(opt)
	}

	return nil, errNotImplemented
}

// AddGroup implements QualityProfilesClient.AddGroup.
func (m *MockQualityProfilesClient) AddGroup(_ context.Context, opt *sonar.QualityprofilesAddGroupOptions) (resp *http.Response, err error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}

	return nil, errNotImplemented
}

// AddProject implements QualityProfilesClient.AddProject.
func (m *MockQualityProfilesClient) AddProject(_ context.Context, opt *sonar.QualityprofilesAddProjectOptions) (resp *http.Response, err error) {
	if m.AddProjectFn != nil {
		return m.AddProjectFn(opt)
	}

	return nil, errNotImplemented
}

// AddUser implements QualityProfilesClient.AddUser.
func (m *MockQualityProfilesClient) AddUser(_ context.Context, opt *sonar.QualityprofilesAddUserOptions) (resp *http.Response, err error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}

	return nil, errNotImplemented
}

// Backup implements QualityProfilesClient.Backup.
func (m *MockQualityProfilesClient) Backup(_ context.Context, opt *sonar.QualityprofilesBackupOptions) (v *string, resp *http.Response, err error) {
	if m.BackupFn != nil {
		return m.BackupFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ChangeParent implements QualityProfilesClient.ChangeParent.
func (m *MockQualityProfilesClient) ChangeParent(_ context.Context, opt *sonar.QualityprofilesChangeParentOptions) (resp *http.Response, err error) {
	if m.ChangeParentFn != nil {
		return m.ChangeParentFn(opt)
	}

	return nil, errNotImplemented
}

// Changelog implements QualityProfilesClient.Changelog.
func (m *MockQualityProfilesClient) Changelog(_ context.Context, opt *sonar.QualityprofilesChangelogOptions) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error) {
	if m.ChangelogFn != nil {
		return m.ChangelogFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Compare implements QualityProfilesClient.Compare.
func (m *MockQualityProfilesClient) Compare(_ context.Context, opt *sonar.QualityprofilesCompareOptions) (v *sonar.QualityprofilesCompare, resp *http.Response, err error) {
	if m.CompareFn != nil {
		return m.CompareFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Copy implements QualityProfilesClient.Copy.
func (m *MockQualityProfilesClient) Copy(_ context.Context, opt *sonar.QualityprofilesCopyOptions) (v *sonar.QualityprofilesCopy, resp *http.Response, err error) {
	if m.CopyFn != nil {
		return m.CopyFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Create implements QualityProfilesClient.Create.
func (m *MockQualityProfilesClient) Create(_ context.Context, opt *sonar.QualityprofilesCreateOptions) (v *sonar.QualityprofilesCreate, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// DeactivateRule implements QualityProfilesClient.DeactivateRule.
func (m *MockQualityProfilesClient) DeactivateRule(_ context.Context, opt *sonar.QualityprofilesDeactivateRuleOptions) (resp *http.Response, err error) {
	if m.DeactivateRuleFn != nil {
		return m.DeactivateRuleFn(opt)
	}

	return nil, errNotImplemented
}

// DeactivateRules implements QualityProfilesClient.DeactivateRules.
func (m *MockQualityProfilesClient) DeactivateRules(_ context.Context, opt *sonar.QualityprofilesDeactivateRulesOptions) (resp *http.Response, err error) {
	if m.DeactivateRulesFn != nil {
		return m.DeactivateRulesFn(opt)
	}

	return nil, errNotImplemented
}

// Delete implements QualityProfilesClient.Delete.
func (m *MockQualityProfilesClient) Delete(_ context.Context, opt *sonar.QualityprofilesDeleteOptions) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// Inheritance implements QualityProfilesClient.Inheritance.
func (m *MockQualityProfilesClient) Inheritance(_ context.Context, opt *sonar.QualityprofilesInheritanceOptions) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error) {
	if m.InheritanceFn != nil {
		return m.InheritanceFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Projects implements QualityProfilesClient.Projects.
func (m *MockQualityProfilesClient) Projects(_ context.Context, opt *sonar.QualityprofilesProjectsOptions) (v *sonar.QualityprofilesProjects, resp *http.Response, err error) {
	if m.ProjectsFn != nil {
		return m.ProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// RemoveGroup implements QualityProfilesClient.RemoveGroup.
func (m *MockQualityProfilesClient) RemoveGroup(_ context.Context, opt *sonar.QualityprofilesRemoveGroupOptions) (resp *http.Response, err error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveProject implements QualityProfilesClient.RemoveProject.
func (m *MockQualityProfilesClient) RemoveProject(_ context.Context, opt *sonar.QualityprofilesRemoveProjectOptions) (resp *http.Response, err error) {
	if m.RemoveProjectFn != nil {
		return m.RemoveProjectFn(opt)
	}

	return nil, errNotImplemented
}

// RemoveUser implements QualityProfilesClient.RemoveUser.
func (m *MockQualityProfilesClient) RemoveUser(_ context.Context, opt *sonar.QualityprofilesRemoveUserOptions) (resp *http.Response, err error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}

	return nil, errNotImplemented
}

// Rename implements QualityProfilesClient.Rename.
func (m *MockQualityProfilesClient) Rename(_ context.Context, opt *sonar.QualityprofilesRenameOptions) (resp *http.Response, err error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}

	return nil, errNotImplemented
}

// Restore implements QualityProfilesClient.Restore.
func (m *MockQualityProfilesClient) Restore(_ context.Context, opt *sonar.QualityprofilesRestoreOptions) (resp *http.Response, err error) {
	if m.RestoreFn != nil {
		return m.RestoreFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements QualityProfilesClient.Search.
func (m *MockQualityProfilesClient) Search(_ context.Context, opt *sonar.QualityprofilesSearchOptions) (v *sonar.QualityprofilesSearch, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchGroups implements QualityProfilesClient.SearchGroups.
func (m *MockQualityProfilesClient) SearchGroups(_ context.Context, opt *sonar.QualityprofilesSearchGroupsOptions) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SearchUsers implements QualityProfilesClient.SearchUsers.
func (m *MockQualityProfilesClient) SearchUsers(_ context.Context, opt *sonar.QualityprofilesSearchUsersOptions) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error) {
	if m.SearchUsersFn != nil {
		return m.SearchUsersFn(opt)
	}

	return nil, nil, errNotImplemented
}

// SetDefault implements QualityProfilesClient.SetDefault.
func (m *MockQualityProfilesClient) SetDefault(_ context.Context, opt *sonar.QualityprofilesSetDefaultOptions) (resp *http.Response, err error) {
	if m.SetDefaultFn != nil {
		return m.SetDefaultFn(opt)
	}

	return nil, errNotImplemented
}

// Show implements QualityProfilesClient.Show.
func (m *MockQualityProfilesClient) Show(_ context.Context, opt *sonar.QualityprofilesShowOptions) (v *sonar.QualityprofilesShow, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errNotImplemented
}
