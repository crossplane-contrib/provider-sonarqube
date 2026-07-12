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
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// PermissionsClient is the interface for managing resource permissions
// in SonarQube.
type PermissionsClient interface {
	AddGroup(ctx context.Context, opt *sonar.PermissionsAddGroupOptions) (*http.Response, error)
	Groups(ctx context.Context, opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error)
	RemoveGroup(ctx context.Context, opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error)
	AddUser(ctx context.Context, opt *sonar.PermissionsAddUserOptions) (*http.Response, error)
	Users(ctx context.Context, opt *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error)
	RemoveUser(ctx context.Context, opt *sonar.PermissionsRemoveUserOptions) (*http.Response, error)
}

// NewPermissionsClient creates a new PermissionsClient with the provided
// SonarQube client configuration.
func NewPermissionsClient(clientConfig common.Config) PermissionsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Permissions
}

// GeneratePermissionsAddGroupOptions generates the options for adding
// a group to a permission in SonarQube based on the provided parameters.
// projectKey is optional; if nil, global permissions are managed.
func GeneratePermissionsAddGroupOptions(groupName, permission string, projectKey *string) *sonar.PermissionsAddGroupOptions {
	opts := &sonar.PermissionsAddGroupOptions{
		Permission: permission,
		GroupName:  groupName,
	}

	helpers.AssignIfNonNil(&opts.ProjectKey, projectKey)

	return opts
}

// GeneratePermissionsRemoveGroupOptions generates the options for removing
// a group from a permission in SonarQube based on the provided parameters.
// projectKey is optional; if nil, global permissions are managed.
func GeneratePermissionsRemoveGroupOptions(groupName, permission string, projectKey *string) *sonar.PermissionsRemoveGroupOptions {
	opts := &sonar.PermissionsRemoveGroupOptions{
		Permission: permission,
		GroupName:  groupName,
	}

	helpers.AssignIfNonNil(&opts.ProjectKey, projectKey)

	return opts
}

// GeneratePermissionsGroupsOptions generates the options for retrieving
// the groups associated with a permission in SonarQube.
// projectKey is optional; if nil, global permissions are queried.
func GeneratePermissionsGroupsOptions(groupName string, projectKey *string, pagination *sonar.PaginationArgs) *sonar.PermissionsGroupsOptions {
	options := &sonar.PermissionsGroupsOptions{
		Query: groupName,
	}

	helpers.AssignIfNonNil(&options.ProjectKey, projectKey)
	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// GeneratePermissionsAddUserOptions generates the options for adding
// a user to a permission in SonarQube based on the provided parameters.
// projectKey is optional; if nil, global permissions are managed.
func GeneratePermissionsAddUserOptions(login, permission string, projectKey *string) *sonar.PermissionsAddUserOptions {
	opts := &sonar.PermissionsAddUserOptions{
		Login:      login,
		Permission: permission,
	}

	helpers.AssignIfNonNil(&opts.ProjectKey, projectKey)

	return opts
}

// GeneratePermissionsRemoveUserOptions generates the options for removing
// a user from a permission in SonarQube based on the provided parameters.
// projectKey is optional; if nil, global permissions are managed.
func GeneratePermissionsRemoveUserOptions(login, permission string, projectKey *string) *sonar.PermissionsRemoveUserOptions {
	opts := &sonar.PermissionsRemoveUserOptions{
		Login:      login,
		Permission: permission,
	}

	helpers.AssignIfNonNil(&opts.ProjectKey, projectKey)

	return opts
}

// GeneratePermissionsUsersOptions generates the options for retrieving
// the users associated with a permission in SonarQube.
// projectKey is optional; if nil, global permissions are queried.
func GeneratePermissionsUsersOptions(login string, projectKey *string, pagination *sonar.PaginationArgs) *sonar.PermissionsUsersOptions {
	options := &sonar.PermissionsUsersOptions{
		Query: login,
	}

	helpers.AssignIfNonNil(&options.ProjectKey, projectKey)
	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// LateInitializePermissions is a no-op: permissions are exactly what
// the user sets.
func LateInitializePermissions(_ *v1alpha1.PermissionsParameters, _ *v1alpha1.PermissionsObservation) {
}

// IsPermissionsLateInitialized always returns false since no fields
// are late-initialized.
func IsPermissionsLateInitialized(_, _ *v1alpha1.PermissionsParameters) bool {
	return false
}

// IsPermissionsUpToDate checks if the Permissions spec is up to date with the
// SonarQube API response. It returns true if the spec matches the observation.
func IsPermissionsUpToDate(spec *v1alpha1.PermissionsParameters, obs *v1alpha1.PermissionsObservation) bool {
	if spec == nil {
		return true
	}

	if obs == nil {
		return false
	}

	return ArePermissionsEqual(&spec.Permissions, obs.Permissions)
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
