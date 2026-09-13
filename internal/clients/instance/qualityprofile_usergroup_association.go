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

	// qualityProfileAssociationNameParts is the number of
	// colon-separated parts in a valid association external name of the
	// form <type>:<subject>:<language>:<qualityProfile>.
	qualityProfileAssociationNameParts = 4
)

// NewQualityProfileUsergroupAssociationClient creates a
// QualityprofilesService used to manage Quality Profile group and user
// associations.
func NewQualityProfileUsergroupAssociationClient(clientConfig common.Config) *sonar.QualityprofilesService {
	newClient := common.NewClient(clientConfig)

	return newClient.Qualityprofiles
}

// GenerateQualityProfileAddGroupOptions generates options for granting a
// group edit rights on a Quality Profile identified by language and
// qualityProfile.
func GenerateQualityProfileAddGroupOptions(language, qualityProfile, groupName string) *sonar.QualityprofilesAddGroupOptions {
	return &sonar.QualityprofilesAddGroupOptions{
		Group:          groupName,
		Language:       language,
		QualityProfile: qualityProfile,
	}
}

// GenerateQualityProfileAddUserOptions generates options for granting a
// user edit rights on a Quality Profile identified by language and
// qualityProfile.
func GenerateQualityProfileAddUserOptions(language, qualityProfile, login string) *sonar.QualityprofilesAddUserOptions {
	return &sonar.QualityprofilesAddUserOptions{
		Language:       language,
		Login:          login,
		QualityProfile: qualityProfile,
	}
}

// GenerateQualityProfileRemoveGroupOptions generates options for revoking
// a group's edit rights on a Quality Profile identified by language and
// qualityProfile.
func GenerateQualityProfileRemoveGroupOptions(language, qualityProfile, groupName string) *sonar.QualityprofilesRemoveGroupOptions {
	return &sonar.QualityprofilesRemoveGroupOptions{
		Group:          groupName,
		Language:       language,
		QualityProfile: qualityProfile,
	}
}

// GenerateQualityProfileRemoveUserOptions generates options for revoking a
// user's edit rights on a Quality Profile identified by language and
// qualityProfile.
func GenerateQualityProfileRemoveUserOptions(language, qualityProfile, login string) *sonar.QualityprofilesRemoveUserOptions {
	return &sonar.QualityprofilesRemoveUserOptions{
		Language:       language,
		Login:          login,
		QualityProfile: qualityProfile,
	}
}

// GenerateQualityProfileSearchGroupsOptions generates options for
// searching groups associated with a Quality Profile identified by
// language and qualityProfile. Selected is set to "all" so both selected
// and deselected entries are returned.
func GenerateQualityProfileSearchGroupsOptions(language, qualityProfile, query string, pagination *sonar.PaginationArgs) *sonar.QualityprofilesSearchGroupsOptions {
	opts := &sonar.QualityprofilesSearchGroupsOptions{
		Language:       language,
		QualityProfile: qualityProfile,
		Query:          query,
		Selected:       selectedFilterAll,
	}
	if pagination != nil {
		opts.PaginationArgs = *pagination
	}

	return opts
}

// GenerateQualityProfileSearchUsersOptions generates options for searching
// users associated with a Quality Profile identified by language and
// qualityProfile. Selected is set to "all" so both selected and
// deselected entries are returned.
func GenerateQualityProfileSearchUsersOptions(language, qualityProfile, query string, pagination *sonar.PaginationArgs) *sonar.QualityprofilesSearchUsersOptions {
	opts := &sonar.QualityprofilesSearchUsersOptions{
		Language:       language,
		QualityProfile: qualityProfile,
		Query:          query,
		Selected:       selectedFilterAll,
	}
	if pagination != nil {
		opts.PaginationArgs = *pagination
	}

	return opts
}

// GenerateQualityProfileUsergroupAssociationObservation copies the
// quality profile, language, and the set principal (group name or login)
// from spec into observation.
func GenerateQualityProfileUsergroupAssociationObservation(params *v1alpha1.QualityProfileUsergroupAssociationParameters) v1alpha1.QualityProfileUsergroupAssociationObservation {
	if params == nil {
		return v1alpha1.QualityProfileUsergroupAssociationObservation{}
	}

	observation := v1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: params.QualityProfile,
		Language:       params.Language,
	}
	if params.GroupName != nil {
		observation.GroupName = *params.GroupName
	}

	if params.Login != nil {
		observation.Login = *params.Login
	}

	return observation
}

// IsQualityProfileUsergroupAssociationUpToDate reports whether the
// observed association matches the desired spec. It is true when spec is
// nil and false when observation is nil.
func IsQualityProfileUsergroupAssociationUpToDate(spec *v1alpha1.QualityProfileUsergroupAssociationParameters, observation *v1alpha1.QualityProfileUsergroupAssociationObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.QualityProfile != observation.QualityProfile {
		return false
	}

	if spec.Language != observation.Language {
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

// ParseQualityProfileUsergroupAssociationExternalName decodes an external
// name of the form "<type>:<subject>:<language>:<qualityProfile>" where
// type is group or user. Quality profile names may contain ":" because
// the name is split with SplitN of 4.
func ParseQualityProfileUsergroupAssociationExternalName(externalName string) (subjectType, subject, language, qualityProfile string, err error) {
	parts := strings.SplitN(externalName, ":", qualityProfileAssociationNameParts)
	if len(parts) != qualityProfileAssociationNameParts {
		return "", "", "", "", fmt.Errorf("invalid external name format: %q (expected <type>:<subject>:<language>:<qualityProfile>)", externalName)
	}

	subjectType = parts[0]
	if subjectType != SubjectTypeGroup && subjectType != SubjectTypeUser {
		return "", "", "", "", fmt.Errorf("unknown subject type %q in external name %q", subjectType, externalName)
	}

	return subjectType, parts[1], parts[2], parts[3], nil
}

// BuildQualityProfileUsergroupAssociationExternalName constructs an
// external name of the form "group:<groupName>:<language>:<qualityProfile>"
// or "user:<login>:<language>:<qualityProfile>". It returns "" when no
// principal is set.
func BuildQualityProfileUsergroupAssociationExternalName(params *v1alpha1.QualityProfileUsergroupAssociationParameters) string {
	if params == nil {
		return ""
	}

	if params.GroupName != nil && *params.GroupName != "" {
		return fmt.Sprintf("%s:%s:%s:%s", SubjectTypeGroup, *params.GroupName, params.Language, params.QualityProfile)
	}

	if params.Login != nil && *params.Login != "" {
		return fmt.Sprintf("%s:%s:%s:%s", SubjectTypeUser, *params.Login, params.Language, params.QualityProfile)
	}

	return ""
}
