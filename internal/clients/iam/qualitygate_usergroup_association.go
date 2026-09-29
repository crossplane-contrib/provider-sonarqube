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
	"fmt"
	"net/http"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// externalNameParts is the number of colon-separated parts in a valid
	// association external name of the form <type>:<subject>:<gateName>.
	externalNameParts = 3
)

// QualityGateUsergroupAssociationClient is the interface for managing the
// groups and users allowed to edit a Quality Gate in SonarQube.
type QualityGateUsergroupAssociationClient interface { //nolint:dupl // Same shape as the Quality Profile client, but bound to distinct SonarQube API types.
	AddGroup(ctx context.Context, opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error)
	AddUser(ctx context.Context, opt *sonar.QualitygatesAddUserOptions) (*http.Response, error)
	RemoveGroup(ctx context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error)
	RemoveUser(ctx context.Context, opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error)
	SearchGroups(ctx context.Context, opt *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error)
	SearchUsers(ctx context.Context, opt *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error)
}

// NewQualityGateUsergroupAssociationClient creates a
// QualityGateUsergroupAssociationClient with the provided SonarQube client
// configuration. When the observe cache is enabled, its writes invalidate
// the selections cached in cache.Default() by QualityGateSelectedGroups
// and QualityGateSelectedUsers.
func NewQualityGateUsergroupAssociationClient(clientConfig common.Config) QualityGateUsergroupAssociationClient {
	newClient := common.NewClient(clientConfig)

	return NewCachedQualityGateUsergroupAssociationClient(newClient.Qualitygates, cache.ForConfig(clientConfig))
}

// GenerateQualityGateAddGroupOptions generates options for granting a group
// edit rights on a Quality Gate.
func GenerateQualityGateAddGroupOptions(gateName, groupName string) *sonar.QualitygatesAddGroupOptions {
	return &sonar.QualitygatesAddGroupOptions{
		GateName:  gateName,
		GroupName: groupName,
	}
}

// GenerateQualityGateAddUserOptions generates options for granting a user
// edit rights on a Quality Gate.
func GenerateQualityGateAddUserOptions(gateName, login string) *sonar.QualitygatesAddUserOptions {
	return &sonar.QualitygatesAddUserOptions{
		GateName: gateName,
		Login:    login,
	}
}

// GenerateQualityGateRemoveGroupOptions generates options for revoking a
// group's edit rights on a Quality Gate.
func GenerateQualityGateRemoveGroupOptions(gateName, groupName string) *sonar.QualitygatesRemoveGroupOptions {
	return &sonar.QualitygatesRemoveGroupOptions{
		GateName:  gateName,
		GroupName: groupName,
	}
}

// GenerateQualityGateRemoveUserOptions generates options for revoking a
// user's edit rights on a Quality Gate.
func GenerateQualityGateRemoveUserOptions(gateName, login string) *sonar.QualitygatesRemoveUserOptions {
	return &sonar.QualitygatesRemoveUserOptions{
		GateName: gateName,
		Login:    login,
	}
}

// GenerateQualityGateSearchGroupsOptions generates options for searching
// groups associated with a Quality Gate. Only the selected entries are
// returned, without name filter.
func GenerateQualityGateSearchGroupsOptions(gateName string, pagination *sonar.PaginationArgs) *sonar.QualitygatesSearchGroupsOptions {
	opts := &sonar.QualitygatesSearchGroupsOptions{
		GateName: gateName,
		Selected: sonar.SelectionFilterSelected,
	}

	helpers.AssignIfNonNil(&opts.PaginationArgs, pagination)

	return opts
}

// GenerateQualityGateSearchUsersOptions generates options for searching
// users associated with a Quality Gate. Only the selected entries are
// returned, without name filter.
func GenerateQualityGateSearchUsersOptions(gateName string, pagination *sonar.PaginationArgs) *sonar.QualitygatesSearchUsersOptions {
	opts := &sonar.QualitygatesSearchUsersOptions{
		GateName: gateName,
		Selected: sonar.SelectionFilterSelected,
	}

	helpers.AssignIfNonNil(&opts.PaginationArgs, pagination)

	return opts
}

// GenerateQualityGateGroupAssociationObservation generates an observation
// from the group returned by the SonarQube Quality Gate groups search.
func GenerateQualityGateGroupAssociationObservation(gateName string, group *sonar.QualityGateGroup) v1alpha1.QualityGateUsergroupAssociationObservation {
	if group == nil {
		return v1alpha1.QualityGateUsergroupAssociationObservation{}
	}

	return v1alpha1.QualityGateUsergroupAssociationObservation{
		GateName:  gateName,
		GroupName: group.Name,
	}
}

// GenerateQualityGateUserAssociationObservation generates an observation
// from the user returned by the SonarQube Quality Gate users search.
func GenerateQualityGateUserAssociationObservation(gateName string, user *sonar.QualityGateUser) v1alpha1.QualityGateUsergroupAssociationObservation {
	if user == nil {
		return v1alpha1.QualityGateUsergroupAssociationObservation{}
	}

	return v1alpha1.QualityGateUsergroupAssociationObservation{
		GateName: gateName,
		Login:    user.Login,
	}
}

// IsQualityGateUsergroupAssociationUpToDate reports whether the observed
// association matches the desired spec. It is true when spec is nil and
// false when observation is nil.
func IsQualityGateUsergroupAssociationUpToDate(spec *v1alpha1.QualityGateUsergroupAssociationParameters, observation *v1alpha1.QualityGateUsergroupAssociationObservation) bool {
	if spec == nil {
		return true
	}

	return observation != nil &&
		spec.GateName == observation.GateName &&
		helpers.IsComparablePtrEqualComparable(spec.GroupName, observation.GroupName) &&
		helpers.IsComparablePtrEqualComparable(spec.Login, observation.Login)
}

// ParseQualityGateUsergroupAssociationExternalName decodes an external name
// of the form "<type>:<subject>:<gateName>" where type is group or user.
// Gate names may contain ":" because the name is split with SplitN of 3.
func ParseQualityGateUsergroupAssociationExternalName(externalName string) (subjectType, subject, gateName string, err error) {
	parts := strings.SplitN(externalName, ":", externalNameParts)
	if len(parts) != externalNameParts {
		return "", "", "", fmt.Errorf("invalid external name format: %q (expected <type>:<subject>:<gateName>)", externalName)
	}

	subjectType = parts[0]
	if subjectType != SubjectTypeGroup && subjectType != SubjectTypeUser {
		return "", "", "", fmt.Errorf("unknown subject type %q in external name %q", subjectType, externalName)
	}

	return subjectType, parts[1], parts[2], nil
}

// BuildQualityGateUsergroupAssociationExternalName constructs an external
// name of the form "group:<groupName>:<gateName>" or
// "user:<login>:<gateName>". It returns "" when no principal is set.
func BuildQualityGateUsergroupAssociationExternalName(params *v1alpha1.QualityGateUsergroupAssociationParameters) string {
	if params == nil {
		return ""
	}

	if ptr.Deref(params.GroupName, "") != "" {
		return fmt.Sprintf("%s:%s:%s", SubjectTypeGroup, *params.GroupName, params.GateName)
	}

	if ptr.Deref(params.Login, "") != "" {
		return fmt.Sprintf("%s:%s:%s", SubjectTypeUser, *params.Login, params.GateName)
	}

	return ""
}
