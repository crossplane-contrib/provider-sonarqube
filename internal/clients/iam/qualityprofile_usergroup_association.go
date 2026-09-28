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
	"net/url"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// qualityProfileAssociationNameParts is the number of colon-separated parts
// in a valid association external name of the form
// <type>:<subject>:<language>:<qualityProfile>.
const qualityProfileAssociationNameParts = 4

// subjectEscaper escapes the characters that would make the subject segment
// of an association external name ambiguous. SonarQube group names may
// contain ":", which is also the external name separator.
var subjectEscaper = strings.NewReplacer("%", "%25", ":", "%3A")

// QualityProfileUsergroupAssociationClient is the interface for managing
// the groups and users allowed to edit a Quality Profile in SonarQube.
type QualityProfileUsergroupAssociationClient interface {
	AddGroup(ctx context.Context, opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error)
	AddUser(ctx context.Context, opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error)
	RemoveGroup(ctx context.Context, opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error)
	RemoveUser(ctx context.Context, opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error)
	SearchGroups(ctx context.Context, opt *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error)
	SearchUsers(ctx context.Context, opt *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error)
}

// NewQualityProfileUsergroupAssociationClient creates a
// QualityProfileUsergroupAssociationClient with the provided SonarQube
// client configuration.
func NewQualityProfileUsergroupAssociationClient(clientConfig common.Config) QualityProfileUsergroupAssociationClient {
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
// language and qualityProfile. Both selected and deselected entries are
// returned.
func GenerateQualityProfileSearchGroupsOptions(language, qualityProfile, query string, pagination *sonar.PaginationArgs) *sonar.QualityprofilesSearchGroupsOptions {
	opts := &sonar.QualityprofilesSearchGroupsOptions{
		Language:       language,
		QualityProfile: qualityProfile,
		Query:          query,
		Selected:       sonar.SelectionFilterAll,
	}

	helpers.AssignIfNonNil(&opts.PaginationArgs, pagination)

	return opts
}

// GenerateQualityProfileSearchUsersOptions generates options for searching
// users associated with a Quality Profile identified by language and
// qualityProfile. Both selected and deselected entries are returned.
func GenerateQualityProfileSearchUsersOptions(language, qualityProfile, query string, pagination *sonar.PaginationArgs) *sonar.QualityprofilesSearchUsersOptions {
	opts := &sonar.QualityprofilesSearchUsersOptions{
		Language:       language,
		QualityProfile: qualityProfile,
		Query:          query,
		Selected:       sonar.SelectionFilterAll,
	}

	helpers.AssignIfNonNil(&opts.PaginationArgs, pagination)

	return opts
}

// GenerateQualityProfileGroupAssociationObservation generates an
// observation from the group returned by the SonarQube Quality Profile
// groups search.
func GenerateQualityProfileGroupAssociationObservation(language, qualityProfile string, group *sonar.QualityprofilesProfileGroup) v1alpha1.QualityProfileUsergroupAssociationObservation {
	if group == nil {
		return v1alpha1.QualityProfileUsergroupAssociationObservation{}
	}

	return v1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: qualityProfile,
		Language:       language,
		GroupName:      group.Name,
	}
}

// GenerateQualityProfileUserAssociationObservation generates an
// observation from the user returned by the SonarQube Quality Profile
// users search.
func GenerateQualityProfileUserAssociationObservation(language, qualityProfile string, user *sonar.QualityprofilesProfileUser) v1alpha1.QualityProfileUsergroupAssociationObservation {
	if user == nil {
		return v1alpha1.QualityProfileUsergroupAssociationObservation{}
	}

	return v1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: qualityProfile,
		Language:       language,
		Login:          user.Login,
	}
}

// IsQualityProfileUsergroupAssociationUpToDate reports whether the
// observed association matches the desired spec. It is true when spec is
// nil and false when observation is nil.
func IsQualityProfileUsergroupAssociationUpToDate(spec *v1alpha1.QualityProfileUsergroupAssociationParameters, observation *v1alpha1.QualityProfileUsergroupAssociationObservation) bool {
	if spec == nil {
		return true
	}

	return observation != nil &&
		spec.QualityProfile == observation.QualityProfile &&
		spec.Language == observation.Language &&
		helpers.IsComparablePtrEqualComparable(spec.GroupName, observation.GroupName) &&
		helpers.IsComparablePtrEqualComparable(spec.Login, observation.Login)
}

// ParseQualityProfileUsergroupAssociationExternalName decodes an external
// name of the form "<type>:<subject>:<language>:<qualityProfile>" where
// type is group or user. Quality profile names may contain ":" because
// the name is split with SplitN of 4. The subject has "%" and ":" percent
// encoded (see BuildQualityProfileUsergroupAssociationExternalName) and is
// returned decoded.
func ParseQualityProfileUsergroupAssociationExternalName(externalName string) (subjectType, subject, language, qualityProfile string, err error) {
	parts := strings.SplitN(externalName, ":", qualityProfileAssociationNameParts)
	if len(parts) != qualityProfileAssociationNameParts {
		return "", "", "", "", fmt.Errorf("invalid external name format: %q (expected <type>:<subject>:<language>:<qualityProfile>)", externalName)
	}

	subjectType = parts[0]
	if subjectType != SubjectTypeGroup && subjectType != SubjectTypeUser {
		return "", "", "", "", fmt.Errorf("unknown subject type %q in external name %q", subjectType, externalName)
	}

	subject, err = url.PathUnescape(parts[1])
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid subject encoding in external name %q: %w", externalName, err)
	}

	return subjectType, subject, parts[2], parts[3], nil
}

// BuildQualityProfileUsergroupAssociationExternalName constructs an
// external name of the form "group:<groupName>:<language>:<qualityProfile>"
// or "user:<login>:<language>:<qualityProfile>". "%" and ":" in the
// subject are percent encoded so that group names containing ":" round-trip.
// It returns "" when no principal is set.
func BuildQualityProfileUsergroupAssociationExternalName(params *v1alpha1.QualityProfileUsergroupAssociationParameters) string {
	if params == nil {
		return ""
	}

	if ptr.Deref(params.GroupName, "") != "" {
		return fmt.Sprintf("%s:%s:%s:%s", SubjectTypeGroup, subjectEscaper.Replace(*params.GroupName), params.Language, params.QualityProfile)
	}

	if ptr.Deref(params.Login, "") != "" {
		return fmt.Sprintf("%s:%s:%s:%s", SubjectTypeUser, subjectEscaper.Replace(*params.Login), params.Language, params.QualityProfile)
	}

	return ""
}
