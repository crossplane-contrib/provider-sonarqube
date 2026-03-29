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

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// GroupsClient is the interface for interacting with SonarQube Group API
// It handles all the operations related to Groups in SonarQube, such as creating, updating, deleting, and retrieving Groups.
type GroupsClient interface {
	CreateGroup(opt *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error)
	CreateGroupMembership(opt *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error)
	DeleteGroup(groupID string) (*http.Response, error)
	DeleteGroupMembership(membershipID string) (*http.Response, error)
	FetchGroup(groupID string) (*sonar.Group, *http.Response, error)
	SearchGroupMemberships(opt *sonar.AuthorizationsSearchGroupMembershipsOptions) (*sonar.AuthorizationsGroupMembershipsSearch, *http.Response, error)
	SearchGroups(opt *sonar.AuthorizationsSearchGroupsOptions) (*sonar.AuthorizationsGroupsSearch, *http.Response, error)
	UpdateGroup(groupID string, opt *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error)
}

// NewGroupsClient creates a new GroupsClient with the provided SonarQube client configuration.
func NewGroupsClient(clientConfig common.Config) GroupsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.V2.Authorizations
}

// LateInitializeGroup fills the empty fields in the Group spec with the values from the SonarQube API response.
// The API response should be the result of a "Get" operation for the Group resource.
func LateInitializeGroup(spec *v1alpha1.GroupParameters, observation *v1alpha1.GroupObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Description, observation.Description)
}

// IsGroupLateInitialized checks if two Group specs are equal after late initialization. It returns true if the specs are equal, and false if they are not.
func IsGroupLateInitialized(former, current *v1alpha1.GroupParameters) bool {
	if former == nil || current == nil {
		return true
	}

	// Compare the two Group specs after late initialization.
	return former.Name != current.Name ||
		!helpers.IsComparablePtrEqualComparablePtr(former.Description, current.Description)
}

// IsGroupUpToDate checks if the Group spec is up to date with the SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsGroupUpToDate(spec *v1alpha1.GroupParameters, observation *v1alpha1.GroupObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.Name == observation.Name &&
		helpers.IsComparablePtrEqualComparable(spec.Description, observation.Description)
}

// GenerateCreateGroupOptions generates the SonarQube API options for creating a Group based on the Group spec.
func GenerateCreateGroupOptions(spec *v1alpha1.GroupParameters) *sonar.AuthorizationsCreateGroupOptions {
	if spec == nil {
		return nil
	}

	options := &sonar.AuthorizationsCreateGroupOptions{
		Name: spec.Name,
	}

	helpers.AssignIfNonNil(&options.Description, spec.Description)

	return options
}

// GenerateUpdateGroupOptions generates the SonarQube API options for updating a Group based on the Group spec.
func GenerateUpdateGroupOptions(spec *v1alpha1.GroupParameters) *sonar.AuthorizationsUpdateGroupOptions {
	if spec == nil {
		return nil
	}

	options := &sonar.AuthorizationsUpdateGroupOptions{
		Name:        spec.Name,
		Description: spec.Description,
	}

	return options
}

// GenerateGroupObservation generates the GroupObservation from the SonarQube API response for a Group resource.
func GenerateGroupObservation(group *sonar.Group) v1alpha1.GroupObservation {
	if group == nil {
		return v1alpha1.GroupObservation{}
	}

	return v1alpha1.GroupObservation{
		ID:          group.Id,
		Name:        group.Name,
		Description: group.Description,
		Managed:     group.Managed,
		Default:     group.Default,
	}
}
