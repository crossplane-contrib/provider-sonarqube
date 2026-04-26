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

package iam

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

type PermissionsClient interface {
	AddGroup(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error)
	Groups(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error)
	RemoveGroup(opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error)
}

// NewPermissionsClient creates a new PermissionsClient with the provided
// SonarQube client configuration.
func NewPermissionsClient(clientConfig common.Config) PermissionsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Permissions
}

// GeneratePermissionsAddGroupOptions generates the options for adding
// a group to a permission in SonarQube based on the provided parameters.
func GeneratePermissionsAddGroupOptions(groupName, permission string) *sonar.PermissionsAddGroupOptions {
	return &sonar.PermissionsAddGroupOptions{
		Permission: permission,
		GroupName:  groupName,
	}
}

// GeneratePermissionsRemoveGroupOptions generates the options for removing
// a group from a permission in SonarQube based on the provided parameters.
func GeneratePermissionsRemoveGroupOptions(groupName, permission string) *sonar.PermissionsRemoveGroupOptions {
	return &sonar.PermissionsRemoveGroupOptions{
		Permission: permission,
		GroupName:  groupName,
	}
}

// GeneratePermissionsGroupsOptions generates the options for retrieving
// the groups associated with a permission in SonarQube
// based on the provided parameters.
func GeneratePermissionsGroupsOptions(groupName string, pagination *sonar.PaginationArgs) *sonar.PermissionsGroupsOptions {
	options := &sonar.PermissionsGroupsOptions{
		Query: groupName,
	}

	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// ArePermissionsEqual compares the permissions specified
// in the desired state with the permissions observed from SonarQube.
// It returns true if the permissions are equal,
// ignoring order and duplicates, and false otherwise.
func ArePermissionsEqual(spec *[]string, observed []string) bool {
	if spec == nil {
		return true
	}

	return helpers.AreStringSlicesEqualDeDuped(*spec, observed)
}
