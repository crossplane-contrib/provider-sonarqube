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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockQualityProfilesClient is a mock implementation of the QualityProfilesClient interface.
type MockQualityProfilesClient struct {
	ActivateRuleFn    func(opt *sonargo.QualityprofilesActivateRuleOption) (resp *http.Response, err error)
	ActivateRulesFn   func(opt *sonargo.QualityprofilesActivateRulesOption) (resp *http.Response, err error)
	AddGroupFn        func(opt *sonargo.QualityprofilesAddGroupOption) (resp *http.Response, err error)
	AddProjectFn      func(opt *sonargo.QualityprofilesAddProjectOption) (resp *http.Response, err error)
	AddUserFn         func(opt *sonargo.QualityprofilesAddUserOption) (resp *http.Response, err error)
	BackupFn          func(opt *sonargo.QualityprofilesBackupOption) (v *string, resp *http.Response, err error)
	ChangeParentFn    func(opt *sonargo.QualityprofilesChangeParentOption) (resp *http.Response, err error)
	ChangelogFn       func(opt *sonargo.QualityprofilesChangelogOption) (v *sonargo.QualityprofilesChangelogObject, resp *http.Response, err error)
	CompareFn         func(opt *sonargo.QualityprofilesCompareOption) (v *sonargo.QualityprofilesCompareObject, resp *http.Response, err error)
	CopyFn            func(opt *sonargo.QualityprofilesCopyOption) (v *sonargo.QualityprofilesCopyObject, resp *http.Response, err error)
	CreateFn          func(opt *sonargo.QualityprofilesCreateOption) (v *sonargo.QualityprofilesCreateObject, resp *http.Response, err error)
	DeactivateRuleFn  func(opt *sonargo.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error)
	DeactivateRulesFn func(opt *sonargo.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error)
	DeleteFn          func(opt *sonargo.QualityprofilesDeleteOption) (resp *http.Response, err error)
	ExportFn          func(opt *sonargo.QualityprofilesExportOption) (v *string, resp *http.Response, err error)
	ExportersFn       func() (v *sonargo.QualityprofilesExportersObject, resp *http.Response, err error)
	ImportersFn       func() (v *sonargo.QualityprofilesImportersObject, resp *http.Response, err error)
	InheritanceFn     func(opt *sonargo.QualityprofilesInheritanceOption) (v *sonargo.QualityprofilesInheritanceObject, resp *http.Response, err error)
	ProjectsFn        func(opt *sonargo.QualityprofilesProjectsOption) (v *sonargo.QualityprofilesProjectsObject, resp *http.Response, err error)
	RemoveGroupFn     func(opt *sonargo.QualityprofilesRemoveGroupOption) (resp *http.Response, err error)
	RemoveProjectFn   func(opt *sonargo.QualityprofilesRemoveProjectOption) (resp *http.Response, err error)
	RemoveUserFn      func(opt *sonargo.QualityprofilesRemoveUserOption) (resp *http.Response, err error)
	RenameFn          func(opt *sonargo.QualityprofilesRenameOption) (resp *http.Response, err error)
	RestoreFn         func(opt *sonargo.QualityprofilesRestoreOption) (resp *http.Response, err error)
	SearchFn          func(opt *sonargo.QualityprofilesSearchOption) (v *sonargo.QualityprofilesSearchObject, resp *http.Response, err error)
	SearchGroupsFn    func(opt *sonargo.QualityprofilesSearchGroupsOption) (v *sonargo.QualityprofilesSearchGroupsObject, resp *http.Response, err error)
	SearchUsersFn     func(opt *sonargo.QualityprofilesSearchUsersOption) (v *sonargo.QualityprofilesSearchUsersObject, resp *http.Response, err error)
	SetDefaultFn      func(opt *sonargo.QualityprofilesSetDefaultOption) (resp *http.Response, err error)
	ShowFn            func(opt *sonargo.QualityprofilesShowOption) (v *sonargo.QualityprofilesShowObject, resp *http.Response, err error)
}

// Ensure MockQualityProfilesClient implements QualityProfilesClient
var _ instance.QualityProfilesClient = &MockQualityProfilesClient{}

// ActivateRule implements QualityProfilesClient.ActivateRule
func (m *MockQualityProfilesClient) ActivateRule(opt *sonargo.QualityprofilesActivateRuleOption) (resp *http.Response, err error) {
	if m.ActivateRuleFn != nil {
		return m.ActivateRuleFn(opt)
	}
	return nil, nil
}

// ActivateRules implements QualityProfilesClient.ActivateRules
func (m *MockQualityProfilesClient) ActivateRules(opt *sonargo.QualityprofilesActivateRulesOption) (resp *http.Response, err error) {
	if m.ActivateRulesFn != nil {
		return m.ActivateRulesFn(opt)
	}
	return nil, nil
}

// AddGroup implements QualityProfilesClient.AddGroup
func (m *MockQualityProfilesClient) AddGroup(opt *sonargo.QualityprofilesAddGroupOption) (resp *http.Response, err error) {
	if m.AddGroupFn != nil {
		return m.AddGroupFn(opt)
	}
	return nil, nil
}

// AddProject implements QualityProfilesClient.AddProject
func (m *MockQualityProfilesClient) AddProject(opt *sonargo.QualityprofilesAddProjectOption) (resp *http.Response, err error) {
	if m.AddProjectFn != nil {
		return m.AddProjectFn(opt)
	}
	return nil, nil
}

// AddUser implements QualityProfilesClient.AddUser
func (m *MockQualityProfilesClient) AddUser(opt *sonargo.QualityprofilesAddUserOption) (resp *http.Response, err error) {
	if m.AddUserFn != nil {
		return m.AddUserFn(opt)
	}
	return nil, nil
}

// Backup implements QualityProfilesClient.Backup
func (m *MockQualityProfilesClient) Backup(opt *sonargo.QualityprofilesBackupOption) (v *string, resp *http.Response, err error) {
	if m.BackupFn != nil {
		return m.BackupFn(opt)
	}
	return nil, nil, nil
}

// ChangeParent implements QualityProfilesClient.ChangeParent
func (m *MockQualityProfilesClient) ChangeParent(opt *sonargo.QualityprofilesChangeParentOption) (resp *http.Response, err error) {
	if m.ChangeParentFn != nil {
		return m.ChangeParentFn(opt)
	}
	return nil, nil
}

// Changelog implements QualityProfilesClient.Changelog
func (m *MockQualityProfilesClient) Changelog(opt *sonargo.QualityprofilesChangelogOption) (v *sonargo.QualityprofilesChangelogObject, resp *http.Response, err error) {
	if m.ChangelogFn != nil {
		return m.ChangelogFn(opt)
	}
	return nil, nil, nil
}

// Compare implements QualityProfilesClient.Compare
func (m *MockQualityProfilesClient) Compare(opt *sonargo.QualityprofilesCompareOption) (v *sonargo.QualityprofilesCompareObject, resp *http.Response, err error) {
	if m.CompareFn != nil {
		return m.CompareFn(opt)
	}
	return nil, nil, nil
}

// Copy implements QualityProfilesClient.Copy
func (m *MockQualityProfilesClient) Copy(opt *sonargo.QualityprofilesCopyOption) (v *sonargo.QualityprofilesCopyObject, resp *http.Response, err error) {
	if m.CopyFn != nil {
		return m.CopyFn(opt)
	}
	return nil, nil, nil
}

// Create implements QualityProfilesClient.Create
func (m *MockQualityProfilesClient) Create(opt *sonargo.QualityprofilesCreateOption) (v *sonargo.QualityprofilesCreateObject, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}
	return nil, nil, nil
}

// DeactivateRule implements QualityProfilesClient.DeactivateRule
func (m *MockQualityProfilesClient) DeactivateRule(opt *sonargo.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error) {
	if m.DeactivateRuleFn != nil {
		return m.DeactivateRuleFn(opt)
	}
	return nil, nil
}

// DeactivateRules implements QualityProfilesClient.DeactivateRules
func (m *MockQualityProfilesClient) DeactivateRules(opt *sonargo.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error) {
	if m.DeactivateRulesFn != nil {
		return m.DeactivateRulesFn(opt)
	}
	return nil, nil
}

// Delete implements QualityProfilesClient.Delete
func (m *MockQualityProfilesClient) Delete(opt *sonargo.QualityprofilesDeleteOption) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}
	return nil, nil
}

// Export implements QualityProfilesClient.Export
func (m *MockQualityProfilesClient) Export(opt *sonargo.QualityprofilesExportOption) (v *string, resp *http.Response, err error) {
	if m.ExportFn != nil {
		return m.ExportFn(opt)
	}
	return nil, nil, nil
}

// Exporters implements QualityProfilesClient.Exporters
func (m *MockQualityProfilesClient) Exporters() (v *sonargo.QualityprofilesExportersObject, resp *http.Response, err error) {
	if m.ExportersFn != nil {
		return m.ExportersFn()
	}
	return nil, nil, nil
}

// Importers implements QualityProfilesClient.Importers
func (m *MockQualityProfilesClient) Importers() (v *sonargo.QualityprofilesImportersObject, resp *http.Response, err error) {
	if m.ImportersFn != nil {
		return m.ImportersFn()
	}
	return nil, nil, nil
}

// Inheritance implements QualityProfilesClient.Inheritance
func (m *MockQualityProfilesClient) Inheritance(opt *sonargo.QualityprofilesInheritanceOption) (v *sonargo.QualityprofilesInheritanceObject, resp *http.Response, err error) {
	if m.InheritanceFn != nil {
		return m.InheritanceFn(opt)
	}
	return nil, nil, nil
}

// Projects implements QualityProfilesClient.Projects
func (m *MockQualityProfilesClient) Projects(opt *sonargo.QualityprofilesProjectsOption) (v *sonargo.QualityprofilesProjectsObject, resp *http.Response, err error) {
	if m.ProjectsFn != nil {
		return m.ProjectsFn(opt)
	}
	return nil, nil, nil
}

// RemoveGroup implements QualityProfilesClient.RemoveGroup
func (m *MockQualityProfilesClient) RemoveGroup(opt *sonargo.QualityprofilesRemoveGroupOption) (resp *http.Response, err error) {
	if m.RemoveGroupFn != nil {
		return m.RemoveGroupFn(opt)
	}
	return nil, nil
}

// RemoveProject implements QualityProfilesClient.RemoveProject
func (m *MockQualityProfilesClient) RemoveProject(opt *sonargo.QualityprofilesRemoveProjectOption) (resp *http.Response, err error) {
	if m.RemoveProjectFn != nil {
		return m.RemoveProjectFn(opt)
	}
	return nil, nil
}

// RemoveUser implements QualityProfilesClient.RemoveUser
func (m *MockQualityProfilesClient) RemoveUser(opt *sonargo.QualityprofilesRemoveUserOption) (resp *http.Response, err error) {
	if m.RemoveUserFn != nil {
		return m.RemoveUserFn(opt)
	}
	return nil, nil
}

// Rename implements QualityProfilesClient.Rename
func (m *MockQualityProfilesClient) Rename(opt *sonargo.QualityprofilesRenameOption) (resp *http.Response, err error) {
	if m.RenameFn != nil {
		return m.RenameFn(opt)
	}
	return nil, nil
}

// Restore implements QualityProfilesClient.Restore
func (m *MockQualityProfilesClient) Restore(opt *sonargo.QualityprofilesRestoreOption) (resp *http.Response, err error) {
	if m.RestoreFn != nil {
		return m.RestoreFn(opt)
	}
	return nil, nil
}

// Search implements QualityProfilesClient.Search
func (m *MockQualityProfilesClient) Search(opt *sonargo.QualityprofilesSearchOption) (v *sonargo.QualityprofilesSearchObject, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}
	return nil, nil, nil
}

// SearchGroups implements QualityProfilesClient.SearchGroups
func (m *MockQualityProfilesClient) SearchGroups(opt *sonargo.QualityprofilesSearchGroupsOption) (v *sonargo.QualityprofilesSearchGroupsObject, resp *http.Response, err error) {
	if m.SearchGroupsFn != nil {
		return m.SearchGroupsFn(opt)
	}
	return nil, nil, nil
}

// SearchUsers implements QualityProfilesClient.SearchUsers
func (m *MockQualityProfilesClient) SearchUsers(opt *sonargo.QualityprofilesSearchUsersOption) (v *sonargo.QualityprofilesSearchUsersObject, resp *http.Response, err error) {
	if m.SearchUsersFn != nil {
		return m.SearchUsersFn(opt)
	}
	return nil, nil, nil
}

// SetDefault implements QualityProfilesClient.SetDefault
func (m *MockQualityProfilesClient) SetDefault(opt *sonargo.QualityprofilesSetDefaultOption) (resp *http.Response, err error) {
	if m.SetDefaultFn != nil {
		return m.SetDefaultFn(opt)
	}
	return nil, nil
}

// Show implements QualityProfilesClient.Show
func (m *MockQualityProfilesClient) Show(opt *sonargo.QualityprofilesShowOption) (v *sonargo.QualityprofilesShowObject, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}
	return nil, nil, nil
}
