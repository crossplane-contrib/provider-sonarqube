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

package instance

import (
	"fmt"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// SubjectTypeGroup identifies a group principal in an association
	// external name.
	SubjectTypeGroup = "group"
	// SubjectTypeUser identifies a user principal in an association
	// external name.
	SubjectTypeUser = "user"

	// selectedFilterAll requests both selected and deselected search
	// results from SonarQube.
	selectedFilterAll = "all"

	// externalNameParts is the number of colon-separated parts in a valid
	// association external name of the form <type>:<subject>:<gateName>.
	externalNameParts = 3
)

// NewQualityGateUsergroupAssociationClient creates a QualityGatesClient
// used to manage Quality Gate group and user associations.
func NewQualityGateUsergroupAssociationClient(clientConfig common.Config) QualityGatesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Qualitygates
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
// groups associated with a Quality Gate. Selected is set to "all" so both
// selected and deselected entries are returned.
func GenerateQualityGateSearchGroupsOptions(gateName, query string, pagination *sonar.PaginationArgs) *sonar.QualitygatesSearchGroupsOptions {
	opts := &sonar.QualitygatesSearchGroupsOptions{
		GateName: gateName,
		Query:    query,
		Selected: selectedFilterAll,
	}
	if pagination != nil {
		opts.PaginationArgs = *pagination
	}

	return opts
}

// GenerateQualityGateSearchUsersOptions generates options for searching
// users associated with a Quality Gate. Selected is set to "all" so both
// selected and deselected entries are returned.
func GenerateQualityGateSearchUsersOptions(gateName, query string, pagination *sonar.PaginationArgs) *sonar.QualitygatesSearchUsersOptions {
	opts := &sonar.QualitygatesSearchUsersOptions{
		GateName: gateName,
		Query:    query,
		Selected: selectedFilterAll,
	}
	if pagination != nil {
		opts.PaginationArgs = *pagination
	}

	return opts
}

// GenerateQualityGateUsergroupAssociationObservation copies the gate name
// and the set principal (group name or login) from spec into observation.
func GenerateQualityGateUsergroupAssociationObservation(params *v1alpha1.QualityGateUsergroupAssociationParameters) v1alpha1.QualityGateUsergroupAssociationObservation {
	if params == nil {
		return v1alpha1.QualityGateUsergroupAssociationObservation{}
	}

	observation := v1alpha1.QualityGateUsergroupAssociationObservation{
		GateName: params.GateName,
	}
	if params.GroupName != nil {
		observation.GroupName = *params.GroupName
	}

	if params.Login != nil {
		observation.Login = *params.Login
	}

	return observation
}

// IsQualityGateUsergroupAssociationUpToDate reports whether the observed
// association matches the desired spec. It is true when spec is nil and
// false when observation is nil.
func IsQualityGateUsergroupAssociationUpToDate(spec *v1alpha1.QualityGateUsergroupAssociationParameters, observation *v1alpha1.QualityGateUsergroupAssociationObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.GateName != observation.GateName {
		return false
	}

	return derefString(spec.GroupName) == observation.GroupName &&
		derefString(spec.Login) == observation.Login
}

// derefString returns the pointed-to string, or "" when ptr is nil.
func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}

	return *ptr
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

	if params.GroupName != nil && *params.GroupName != "" {
		return fmt.Sprintf("%s:%s:%s", SubjectTypeGroup, *params.GroupName, params.GateName)
	}

	if params.Login != nil && *params.Login != "" {
		return fmt.Sprintf("%s:%s:%s", SubjectTypeUser, *params.Login, params.GateName)
	}

	return ""
}
