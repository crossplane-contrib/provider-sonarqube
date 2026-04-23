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
