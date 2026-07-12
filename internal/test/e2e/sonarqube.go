//go:build e2e

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

package e2e

import (
	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// FetchGroup returns the SonarQube group with the given ID, or (nil, nil)
// if no such group exists. Use the resource's external-name annotation -
// the provider stores the SonarQube-assigned ID there - as the lookup key.
func (f *Framework) FetchGroup(id string) (*sonar.Group, error) {
	g, resp, err := f.Sonar.V2.Authorizations.FetchGroup(id)
	defer helpers.CloseBody(resp)
	if common.IsResponseNotFound(resp) {
		return nil, nil //nolint:nilnil // intentional: 404 is the natural absence sentinel
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// FindGroupByName returns the unique SonarQube group whose name exactly
// matches name, or (nil, nil) if no such group exists. Useful when the
// resource's external-name has not been observed yet (e.g. during creation
// failures), since SonarQube exposes search by name but Fetch requires ID.
func (f *Framework) FindGroupByName(name string) (*sonar.Group, error) {
	res, resp, err := f.Sonar.V2.Authorizations.SearchGroups(&sonar.AuthorizationsSearchGroupsOptions{Query: name})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Groups {
		if res.Groups[i].Name == name {
			return &res.Groups[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional: no match is not an error condition for assertions
}

// FindProjectByKey returns the SonarQube project component with the given
// key, or (nil, nil) if no such project exists.
func (f *Framework) FindProjectByKey(key string) (*sonar.ProjectSearchComponent, error) {
	res, resp, err := f.Sonar.Projects.Search(&sonar.ProjectsSearchOptions{Projects: []string{key}})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Components {
		if res.Components[i].Key == key {
			return &res.Components[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// FindQualityGate returns the quality gate with the given name, or (nil, nil)
// if no such gate exists. SonarQube exposes only a List endpoint, so this
// iterates the full list - fine for the handful of gates used in e2e tests.
func (f *Framework) FindQualityGate(name string) (*sonar.QualityGate, error) {
	res, resp, err := f.Sonar.Qualitygates.List()
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Qualitygates {
		if res.Qualitygates[i].Name == name {
			return &res.Qualitygates[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// FindQualityProfile returns the quality profile matching name + language,
// or (nil, nil) if no such profile exists.
func (f *Framework) FindQualityProfile(name, language string) (*sonar.QualityProfile, error) {
	res, resp, err := f.Sonar.Qualityprofiles.Search(&sonar.QualityprofilesSearchOptions{Language: language})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Profiles {
		if res.Profiles[i].Name == name && res.Profiles[i].Language == language {
			return &res.Profiles[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// FetchRule returns the SonarQube rule with the given fully-qualified key
// (e.g. "java:my-custom-rule"), or (nil, nil) if no such rule exists.
func (f *Framework) FetchRule(key string) (*sonar.RuleDetails, error) {
	res, resp, err := f.Sonar.Rules.Show(&sonar.RulesShowOptions{Key: key})
	defer helpers.CloseBody(resp)
	if common.IsResponseNotFound(resp) {
		return nil, nil //nolint:nilnil // intentional
	}
	if err != nil {
		return nil, err
	}
	return &res.Rule, nil
}

// FetchSettingValue returns the SettingValue for the given key (scoped to
// component when non-empty), or (nil, nil) if SonarQube did not return a
// value for that key. Settings that were never set return (nil, nil) -
// callers treat that as the natural "not yet reconciled" state.
func (f *Framework) FetchSettingValue(component, key string) (*sonar.SettingValue, error) {
	opts := &sonar.SettingsValuesOptions{Keys: []string{key}}
	if component != "" {
		opts.Component = component
	}
	res, resp, err := f.Sonar.Settings.Values(opts)
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Settings {
		if res.Settings[i].Key == key {
			return &res.Settings[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// FetchUser returns the SonarQube user with the given ID, or (nil, nil)
// if no such user exists. The provider stores the user ID in the
// external-name annotation, so the typical lookup is
// f.FetchUser(meta.GetExternalName(user)).
func (f *Framework) FetchUser(id string) (*sonar.UserV2, error) {
	u, resp, err := f.Sonar.V2.UsersManagement.Fetch(id)
	defer helpers.CloseBody(resp)
	if common.IsResponseNotFound(resp) {
		return nil, nil //nolint:nilnil // intentional
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// FindPermissionsTemplate returns the permissions template whose name
// exactly matches name, or (nil, nil) if no such template exists.
func (f *Framework) FindPermissionsTemplate(name string) (*sonar.PermissionTemplate, error) {
	res, resp, err := f.Sonar.Permissions.SearchTemplates(&sonar.PermissionsSearchTemplatesOptions{Query: name})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.PermissionTemplates {
		if res.PermissionTemplates[i].Name == name {
			return &res.PermissionTemplates[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// GroupPermissions returns the global permissions assigned to the named
// group. Returns an empty (non-nil) slice when SonarQube knows the group
// but it has no global permissions, and an error when the group is unknown
// or the API call fails.
func (f *Framework) GroupPermissions(groupName string) ([]string, error) {
	res, resp, err := f.Sonar.Permissions.Groups(&sonar.PermissionsGroupsOptions{Query: groupName})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Groups {
		if res.Groups[i].Name == groupName {
			perms := make([]string, len(res.Groups[i].Permissions))
			copy(perms, res.Groups[i].Permissions)
			return perms, nil
		}
	}
	return []string{}, nil
}

// UserPermissions returns the global permissions assigned to the named user.
// Returns an empty (non-nil) slice when SonarQube knows the user but it has
// no global permissions, and an error when the API call fails.
func (f *Framework) UserPermissions(login string) ([]string, error) {
	res, resp, err := f.Sonar.Permissions.Users(&sonar.PermissionsUsersOptions{Query: login})
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Users {
		if res.Users[i].Login == login {
			perms := make([]string, len(res.Users[i].Permissions))
			copy(perms, res.Users[i].Permissions)
			return perms, nil
		}
	}
	return []string{}, nil
}

// FindALMGitLabDefinitionByKey returns the GitLab ALM setting definition
// whose key exactly matches key, or (nil, nil) if no such definition exists.
func (f *Framework) FindALMGitLabDefinitionByKey(key string) (*sonar.GitlabDefinition, error) {
	res, resp, err := f.Sonar.AlmSettings.ListDefinitions()
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Gitlab {
		if res.Gitlab[i].Key == key {
			return &res.Gitlab[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional: absence of definition is the natural "not yet created" sentinel
}

// FindGlobalWebhookByKey returns the global SonarQube webhook whose key
// exactly matches key, or (nil, nil) if no such webhook exists.
func (f *Framework) FindGlobalWebhookByKey(key string) (*sonar.Webhook, error) {
	res, resp, err := f.Sonar.Webhooks.List(&sonar.WebhooksListOptions{})
	defer helpers.CloseBody(resp)

	if err != nil {
		return nil, err
	}

	for i := range res.Webhooks {
		if res.Webhooks[i].Key == key {
			return &res.Webhooks[i], nil
		}
	}

	return nil, nil //nolint:nilnil // intentional: absence is the natural "not found" sentinel
}

// FindInstalledPlugin returns the installed plugin with the given key,
// or (nil, nil) if no such plugin is installed.
func (f *Framework) FindInstalledPlugin(key string) (*sonar.PluginInstalled, error) {
	res, resp, err := f.Sonar.Plugins.Installed(nil)
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}

	for i := range res.Plugins {
		if res.Plugins[i].Key == key {
			return &res.Plugins[i], nil
		}
	}

	return nil, nil //nolint:nilnil // intentional: absence is the natural "not installed" sentinel
}

// FindPendingInstallPlugin returns the pending-install entry for the given key,
// or (nil, nil) if no such plugin is queued. Plugins remain pending until
// SonarQube is restarted, so this is the expected state in e2e tests.
func (f *Framework) FindPendingInstallPlugin(key string) (*sonar.PluginPending, error) {
	res, resp, err := f.Sonar.Plugins.Pending()
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}

	for i := range res.Installing {
		if res.Installing[i].Key == key {
			return &res.Installing[i], nil
		}
	}

	return nil, nil //nolint:nilnil // intentional: absence is the natural "not queued" sentinel
}

// FindALMGitHubDefinitionByKey returns the GitHub ALM setting definition
// whose key exactly matches key, or (nil, nil) if no such definition exists.
func (f *Framework) FindALMGitHubDefinitionByKey(key string) (*sonar.GithubDefinition, error) {
	res, resp, err := f.Sonar.AlmSettings.ListDefinitions()
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	for i := range res.Github {
		if res.Github[i].Key == key {
			return &res.Github[i], nil
		}
	}
	return nil, nil //nolint:nilnil // intentional
}

// FindPortfolioByKey returns the SonarQube portfolio with the given key,
// or (nil, nil) if no such portfolio exists. Portfolios are an Enterprise
// Edition feature; SonarQube returns 404 both when the key is unknown and
// when the running edition/license does not support portfolios at all.
func (f *Framework) FindPortfolioByKey(key string) (*sonar.ViewDetails, error) {
	res, resp, err := f.Sonar.Views.Show(&sonar.ViewsShowOptions{Key: key})
	defer helpers.CloseBody(resp)
	if common.IsResponseNotFound(resp) {
		return nil, nil //nolint:nilnil // intentional: 404 is the natural absence sentinel
	}
	if err != nil {
		return nil, err
	}
	return &res.Portfolio, nil
}

// FetchLicense returns the license currently applied to the SonarQube
// instance. SonarQube always returns 200 from this endpoint - even an
// unlicensed Enterprise/Data Center instance reports a License with empty
// fields - so callers should inspect the returned fields rather than
// treating an error as "no license".
func (f *Framework) FetchLicense() (*sonar.License, error) {
	res, resp, err := f.Sonar.Editions.Get()
	defer helpers.CloseBody(resp)
	if err != nil {
		return nil, err
	}
	return &res.License, nil
}
