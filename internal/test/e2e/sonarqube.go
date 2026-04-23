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

//go:build e2e

package e2e

import (
	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// FetchGroup returns the SonarQube group with the given ID, or (nil, nil)
// if no such group exists. Use the resource's external-name annotation —
// the provider stores the SonarQube-assigned ID there — as the lookup key.
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
// iterates the full list — fine for the handful of gates used in e2e tests.
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
// value for that key. Settings that were never set return (nil, nil) —
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
