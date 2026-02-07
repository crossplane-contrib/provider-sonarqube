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

// MockQualityProfilesClient is a mock implementation of the QualityProfilesClient interface.
type MockQualityProfilesClient struct {
	ActivateRuleFn    func(opt *sonar.QualityprofilesActivateRuleOption) (resp *http.Response, err error)
	ActivateRulesFn   func(opt *sonar.QualityprofilesActivateRulesOption) (resp *http.Response, err error)
	AddGroupFn        func(opt *sonar.QualityprofilesAddGroupOption) (resp *http.Response, err error)
	AddProjectFn      func(opt *sonar.QualityprofilesAddProjectOption) (resp *http.Response, err error)
	AddUserFn         func(opt *sonar.QualityprofilesAddUserOption) (resp *http.Response, err error)
	BackupFn          func(opt *sonar.QualityprofilesBackupOption) (v *string, resp *http.Response, err error)
	ChangeParentFn    func(opt *sonar.QualityprofilesChangeParentOption) (resp *http.Response, err error)
	ChangelogFn       func(opt *sonar.QualityprofilesChangelogOption) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error)
	CompareFn         func(opt *sonar.QualityprofilesCompareOption) (v *sonar.QualityprofilesCompare, resp *http.Response, err error)
	CopyFn            func(opt *sonar.QualityprofilesCopyOption) (v *sonar.QualityprofilesCopy, resp *http.Response, err error)
	CreateFn          func(opt *sonar.QualityprofilesCreateOption) (v *sonar.QualityprofilesCreate, resp *http.Response, err error)
	DeactivateRuleFn  func(opt *sonar.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error)
	DeactivateRulesFn func(opt *sonar.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error)
	DeleteFn          func(opt *sonar.QualityprofilesDeleteOption) (resp *http.Response, err error)
	InheritanceFn     func(opt *sonar.QualityprofilesInheritanceOption) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error)
	ProjectsFn        func(opt *sonar.QualityprofilesProjectsOption) (v *sonar.QualityprofilesProjects, resp *http.Response, err error)
	RemoveGroupFn     func(opt *sonar.QualityprofilesRemoveGroupOption) (resp *http.Response, err error)
	RemoveProjectFn   func(opt *sonar.QualityprofilesRemoveProjectOption) (resp *http.Response, err error)
	RemoveUserFn      func(opt *sonar.QualityprofilesRemoveUserOption) (resp *http.Response, err error)
	RenameFn          func(opt *sonar.QualityprofilesRenameOption) (resp *http.Response, err error)
	RestoreFn         func(opt *sonar.QualityprofilesRestoreOption) (resp *http.Response, err error)
	SearchFn          func(opt *sonar.QualityprofilesSearchOption) (v *sonar.QualityprofilesSearch, resp *http.Response, err error)
	SearchGroupsFn    func(opt *sonar.QualityprofilesSearchGroupsOption) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error)
	SearchUsersFn     func(opt *sonar.QualityprofilesSearchUsersOption) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error)
	SetDefaultFn      func(opt *sonar.QualityprofilesSetDefaultOption) (resp *http.Response, err error)
	ShowFn            func(opt *sonar.QualityprofilesShowOption) (v *sonar.QualityprofilesShow, resp *http.Response, err error)
}

// Ensure MockQualityProfilesClient implements QualityProfilesClient
var _ instance.QualityProfilesClient = &MockQualityProfilesClient{}

// ActivateRule implements QualityProfilesClient.ActivateRule
func (m *MockQualityProfilesClient) ActivateRule(opt *sonar.QualityprofilesActivateRuleOption) (resp *http.Response, err error) {
	if m.ActivateRuleFn != nil {
		return m.ActivateRuleFn(opt)
	}
	return nil, nil
}

// ActivateRules implements QualityProfilesClient.ActivateRules
func (m *MockQualityProfilesClient) ActivateRules(opt *sonar.QualityprofilesActivateRulesOption) (resp *http.Response, err error) {
	if m.ActivateRulesFn != nil {
		return m.ActivateRulesFn(opt)
	}
	return nil, nil
}

// AddGroup implements QualityProfilesClient.AddGroup
func (m *MockQualityProfilesClient) AddGroup(opt *sonar.QualityprofilesAddGroupOption) (resp *http.Response, err error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}
	return nil, nil
}

// AddProject implements QualityProfilesClient.AddProject
func (m *MockQualityProfilesClient) AddProject(opt *sonar.QualityprofilesAddProjectOption) (resp *http.Response, err error) {
	if m.AddProjectFn != nil {
		return m.AddProjectFn(opt)
	}
	return nil, nil
}

// AddUser implements QualityProfilesClient.AddUser
func (m *MockQualityProfilesClient) AddUser(opt *sonar.QualityprofilesAddUserOption) (resp *http.Response, err error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}
	return nil, nil
}

// Backup implements QualityProfilesClient.Backup
func (m *MockQualityProfilesClient) Backup(opt *sonar.QualityprofilesBackupOption) (v *string, resp *http.Response, err error) {
	if m.BackupFn != nil {
		return m.BackupFn(opt)
	}
	return nil, nil, nil
}

// ChangeParent implements QualityProfilesClient.ChangeParent
func (m *MockQualityProfilesClient) ChangeParent(opt *sonar.QualityprofilesChangeParentOption) (resp *http.Response, err error) {
	if m.ChangeParentFn != nil {
		return m.ChangeParentFn(opt)
	}
	return nil, nil
}

// Changelog implements QualityProfilesClient.Changelog
func (m *MockQualityProfilesClient) Changelog(opt *sonar.QualityprofilesChangelogOption) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error) {
	if m.ChangelogFn != nil {
		return m.ChangelogFn(opt)
	}
	return nil, nil, nil
}

// Compare implements QualityProfilesClient.Compare
func (m *MockQualityProfilesClient) Compare(opt *sonar.QualityprofilesCompareOption) (v *sonar.QualityprofilesCompare, resp *http.Response, err error) {
	if m.CompareFn != nil {
		return m.CompareFn(opt)
	}
	return nil, nil, nil
}

// Copy implements QualityProfilesClient.Copy
func (m *MockQualityProfilesClient) Copy(opt *sonar.QualityprofilesCopyOption) (v *sonar.QualityprofilesCopy, resp *http.Response, err error) {
	if m.CopyFn != nil {
		return m.CopyFn(opt)
	}
	return nil, nil, nil
}

// Create implements QualityProfilesClient.Create
func (m *MockQualityProfilesClient) Create(opt *sonar.QualityprofilesCreateOption) (v *sonar.QualityprofilesCreate, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}
	return nil, nil, nil
}

// DeactivateRule implements QualityProfilesClient.DeactivateRule
func (m *MockQualityProfilesClient) DeactivateRule(opt *sonar.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error) {
	if m.DeactivateRuleFn != nil {
		return m.DeactivateRuleFn(opt)
	}
	return nil, nil
}

// DeactivateRules implements QualityProfilesClient.DeactivateRules
func (m *MockQualityProfilesClient) DeactivateRules(opt *sonar.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error) {
	if m.DeactivateRulesFn != nil {
		return m.DeactivateRulesFn(opt)
	}
	return nil, nil
}

// Delete implements QualityProfilesClient.Delete
func (m *MockQualityProfilesClient) Delete(opt *sonar.QualityprofilesDeleteOption) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}
	return nil, nil
}

// Inheritance implements QualityProfilesClient.Inheritance
func (m *MockQualityProfilesClient) Inheritance(opt *sonar.QualityprofilesInheritanceOption) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error) {
	if m.InheritanceFn != nil {
		return m.InheritanceFn(opt)
	}
	return nil, nil, nil
}

// Projects implements QualityProfilesClient.Projects
func (m *MockQualityProfilesClient) Projects(opt *sonar.QualityprofilesProjectsOption) (v *sonar.QualityprofilesProjects, resp *http.Response, err error) {
	if m.ProjectsFn != nil {
		return m.ProjectsFn(opt)
	}
	return nil, nil, nil
}

// RemoveGroup implements QualityProfilesClient.RemoveGroup
func (m *MockQualityProfilesClient) RemoveGroup(opt *sonar.QualityprofilesRemoveGroupOption) (resp *http.Response, err error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}
	return nil, nil
}

// RemoveProject implements QualityProfilesClient.RemoveProject
func (m *MockQualityProfilesClient) RemoveProject(opt *sonar.QualityprofilesRemoveProjectOption) (resp *http.Response, err error) {
	if m.RemoveProjectFn != nil {
		return m.RemoveProjectFn(opt)
	}
	return nil, nil
}

// RemoveUser implements QualityProfilesClient.RemoveUser
func (m *MockQualityProfilesClient) RemoveUser(opt *sonar.QualityprofilesRemoveUserOption) (resp *http.Response, err error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}
	return nil, nil
}

// Rename implements QualityProfilesClient.Rename
func (m *MockQualityProfilesClient) Rename(opt *sonar.QualityprofilesRenameOption) (resp *http.Response, err error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}
	return nil, nil
}

// Restore implements QualityProfilesClient.Restore
func (m *MockQualityProfilesClient) Restore(opt *sonar.QualityprofilesRestoreOption) (resp *http.Response, err error) {
	if m.RestoreFn != nil {
		return m.RestoreFn(opt)
	}
	return nil, nil
}

// Search implements QualityProfilesClient.Search
func (m *MockQualityProfilesClient) Search(opt *sonar.QualityprofilesSearchOption) (v *sonar.QualityprofilesSearch, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}
	return nil, nil, nil
}

// SearchGroups implements QualityProfilesClient.SearchGroups
func (m *MockQualityProfilesClient) SearchGroups(opt *sonar.QualityprofilesSearchGroupsOption) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}
	return nil, nil, nil
}

// SearchUsers implements QualityProfilesClient.SearchUsers
func (m *MockQualityProfilesClient) SearchUsers(opt *sonar.QualityprofilesSearchUsersOption) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error) {
	if m.SearchUsersFn != nil {
		return m.SearchUsersFn(opt)
	}
	return nil, nil, nil
}

// SetDefault implements QualityProfilesClient.SetDefault
func (m *MockQualityProfilesClient) SetDefault(opt *sonar.QualityprofilesSetDefaultOption) (resp *http.Response, err error) {
	if m.SetDefaultFn != nil {
		return m.SetDefaultFn(opt)
	}
	return nil, nil
}

// Show implements QualityProfilesClient.Show
func (m *MockQualityProfilesClient) Show(opt *sonar.QualityprofilesShowOption) (v *sonar.QualityprofilesShow, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}
	return nil, nil, nil
}
