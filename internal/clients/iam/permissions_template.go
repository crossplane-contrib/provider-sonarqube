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

// PermissionsTemplatesClient is the interface for interacting with
// SonarQube PermissionsTemplate API
// It handles all the operations related to PermissionsTemplates
// in SonarQube, such as creating, updating, deleting,
// and retrieving PermissionsTemplates.
type PermissionsTemplatesClient interface { //nolint:interfacebloat // This interface is intentionally large to cover all operations related to PermissionsTemplates in SonarQube.
	AddGroupToTemplate(opt *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error)
	AddProjectCreatorToTemplate(opt *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error)
	AddUserToTemplate(opt *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error)
	CreateTemplate(opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error)
	DeleteTemplate(opt *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error)
	RemoveGroupFromTemplate(opt *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error)
	RemoveProjectCreatorFromTemplate(opt *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error)
	RemoveUserFromTemplate(opt *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error)
	SearchTemplates(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error)
	SetDefaultTemplate(opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error)
	TemplateGroups(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error)
	TemplateUsers(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error)
	UpdateTemplate(opt *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error)
}

// NewPermissionsTemplatesClient creates a new PermissionsTemplatesClient
// with the provided SonarQube client configuration.
func NewPermissionsTemplatesClient(clientConfig common.Config) PermissionsTemplatesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Permissions
}

// LateInitializePermissionsTemplate fills the empty fields in the
// PermissionsTemplate spec with the values from the SonarQube API response.
// The API response should be the result of a "Get" operation for
// the PermissionsTemplate resource.
func LateInitializePermissionsTemplate(spec *v1alpha1.PermissionsTemplateParameters, observation *v1alpha1.PermissionsTemplateObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Description, observation.Description)
	helpers.AssignIfNil(&spec.ProjectKeyPattern, observation.ProjectKeyPattern)
	helpers.AssignIfNil(&spec.Default, observation.Default)
	helpers.AssignIfNil(&spec.CreatorPermissions, observation.CreatorPermissions)
}

// IsPermissionsTemplateLateInitialized checks if two PermissionsTemplate
// specs are equal after late initialization.
// It returns true if the specs are equal, and false if they are not.
func IsPermissionsTemplateLateInitialized(former, current *v1alpha1.PermissionsTemplateParameters) bool {
	if former == nil || current == nil {
		return false
	}

	descriptionChanged := !helpers.IsComparablePtrEqualComparablePtr(former.Description, current.Description)
	projectKeyPatternChanged := !helpers.IsComparablePtrEqualComparablePtr(former.ProjectKeyPattern, current.ProjectKeyPattern)
	defaultChanged := !helpers.IsComparablePtrEqualComparablePtr(former.Default, current.Default)

	creatorPermissionsChanged := lenDereferencedStringSlice(former.CreatorPermissions) != lenDereferencedStringSlice(current.CreatorPermissions)
	if former.CreatorPermissions != nil && current.CreatorPermissions != nil {
		creatorPermissionsChanged = !ArePermissionsEqual(former.CreatorPermissions, *current.CreatorPermissions)
	}

	return descriptionChanged || projectKeyPatternChanged || defaultChanged || creatorPermissionsChanged
}

// IsPermissionsTemplateUpToDate checks if the PermissionsTemplate
// spec is up to date with the SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsPermissionsTemplateUpToDate(spec *v1alpha1.PermissionsTemplateParameters, observation *v1alpha1.PermissionsTemplateObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return ArePermissionsTemplateBaseFieldsUpToDate(spec, observation) &&
		helpers.IsComparablePtrEqualComparable(spec.Default, observation.Default) &&
		ArePermissionsEqual(spec.CreatorPermissions, observation.CreatorPermissions) &&
		ArePermissionsTemplateGroupsUpToDate(spec.GroupPermissions, &observation.GroupPermissions) &&
		ArePermissionsTemplateUsersUpToDate(spec.UserPermissions, &observation.UserPermissions)
}

// ArePermissionsTemplateBaseFieldsUpToDate checks if the base fields of
// the PermissionsTemplate spec are up to date with the SonarQube API response.
// Base fields include name, description and project key pattern.
// It returns true if the base fields are up to date, and false if they are not.
func ArePermissionsTemplateBaseFieldsUpToDate(spec *v1alpha1.PermissionsTemplateParameters, observation *v1alpha1.PermissionsTemplateObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.Name == observation.Name &&
		helpers.IsComparablePtrEqualComparable(spec.Description, observation.Description) &&
		helpers.IsComparablePtrEqualComparable(spec.ProjectKeyPattern, observation.ProjectKeyPattern)
}

// GeneratePermissionsTemplateSearchOptions generates the options for searching
// PermissionsTemplates based on the provided template name.
func GeneratePermissionsTemplateSearchOptions(templateName string, pagination *sonar.PaginationArgs) *sonar.PermissionsSearchTemplatesOptions {
	options := &sonar.PermissionsSearchTemplatesOptions{
		Query: templateName,
	}

	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// GeneratePermissionsTemplateUpdateOptions generates the options for updating
// a PermissionsTemplate based on the provided template ID and spec.
func GeneratePermissionsTemplateUpdateOptions(templateID string, spec *v1alpha1.PermissionsTemplateParameters) *sonar.PermissionsUpdateTemplateOptions {
	options := &sonar.PermissionsUpdateTemplateOptions{
		ID:   templateID,
		Name: spec.Name,
	}

	helpers.AssignIfNonNil(&options.Description, spec.Description)
	helpers.AssignIfNonNil(&options.ProjectKeyPattern, spec.ProjectKeyPattern)

	return options
}

// ArePermissionsTemplateGroupsUpToDate determines if the permission templates
// for groups matches the actual observation in SonarQube.
func ArePermissionsTemplateGroupsUpToDate(spec *[]v1alpha1.PermissionsTemplateGroupParameters, observation *[]v1alpha1.PermissionsTemplateGroupObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	groupsMapping := CreatePermissionsTemplateGroupMapping(spec, observation)
	for _, indices := range groupsMapping {
		specIndex, obsIndex := indices[0], indices[1]

		if specIndex == -1 || obsIndex == -1 || !ArePermissionsEqual((*spec)[specIndex].Permissions, (*observation)[obsIndex].Permissions) {
			return false
		}
	}

	return true
}

// CreatePermissionsTemplateGroupMapping builds a mapping between
// the spec and the observed groups.
// First element of the list is the index of the group in the spec,
// and the second element is the index of the group in the observation.
// If a group from the spec does not exist in the observation,
// its second element will be -1, and if a group from the observation
// does not exist in the spec, its first element will be -1.
func CreatePermissionsTemplateGroupMapping(spec *[]v1alpha1.PermissionsTemplateGroupParameters, observation *[]v1alpha1.PermissionsTemplateGroupObservation) map[string][2]int {
	mapping := make(map[string][2]int)

	for i, group := range *spec {
		if group.Name == "" {
			continue
		}

		mapping[group.Name] = [2]int{i, -1}
	}

	for observedIndex, observedGroup := range *observation {
		if observedGroup.Name == "" {
			continue
		}

		if len(observedGroup.Permissions) == 0 {
			if _, exists := mapping[observedGroup.Name]; !exists {
				continue
			}
		}

		if indices, exists := mapping[observedGroup.Name]; exists {
			mapping[observedGroup.Name] = [2]int{indices[0], observedIndex}
		} else {
			mapping[observedGroup.Name] = [2]int{-1, observedIndex}
		}
	}

	return mapping
}

// ArePermissionsTemplateUsersUpToDate determines if the permission templates
// for users matches the actual observation in SonarQube.
func ArePermissionsTemplateUsersUpToDate(spec *[]v1alpha1.PermissionsTemplateUserParameters, observation *[]v1alpha1.PermissionsTemplateUserObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	usersMapping := CreatePermissionsTemplateUserMapping(spec, observation)
	for _, indices := range usersMapping {
		specIndex, obsIndex := indices[0], indices[1]

		if specIndex == -1 || obsIndex == -1 || !ArePermissionsEqual((*spec)[specIndex].Permissions, (*observation)[obsIndex].Permissions) {
			return false
		}
	}

	return true
}

// CreatePermissionsTemplateUserMapping builds a mapping between the spec and
// the observed users.
// First element of the list is the index of the user in the spec,
// and the second element is the index of the user in the observation.
// If a user from the spec does not exist in the observation,
// its second element will be -1, and if a user from the observation does not
// exist in the spec, its first element will be -1.
func CreatePermissionsTemplateUserMapping(spec *[]v1alpha1.PermissionsTemplateUserParameters, observation *[]v1alpha1.PermissionsTemplateUserObservation) map[string][2]int {
	mapping := make(map[string][2]int)

	for i, user := range *spec {
		mapping[user.Login] = [2]int{i, -1}
	}

	for observedIndex, observedUser := range *observation {
		if len(observedUser.Permissions) == 0 {
			if _, exists := mapping[observedUser.Login]; !exists {
				continue
			}
		}

		if indices, exists := mapping[observedUser.Login]; exists {
			mapping[observedUser.Login] = [2]int{indices[0], observedIndex}
		} else {
			mapping[observedUser.Login] = [2]int{-1, observedIndex}
		}
	}

	return mapping
}

// GeneratePermissionsTemplateDeleteOptions generates the options for deleting
// a PermissionsTemplate based on the provided template ID.
func GeneratePermissionsTemplateDeleteOptions(templateID string) *sonar.PermissionsDeleteTemplateOptions {
	return &sonar.PermissionsDeleteTemplateOptions{
		TemplateID: templateID,
	}
}

// GeneratePermissionsTemplateCreationOptions generates the options for
// creating a PermissionsTemplate based on the provided spec.
func GeneratePermissionsTemplateCreationOptions(spec *v1alpha1.PermissionsTemplateParameters) *sonar.PermissionsCreateTemplateOptions {
	options := &sonar.PermissionsCreateTemplateOptions{
		Name: spec.Name,
	}

	helpers.AssignIfNonNil(&options.Description, spec.Description)
	helpers.AssignIfNonNil(&options.ProjectKeyPattern, spec.ProjectKeyPattern)

	return options
}

// GeneratePermissionsTemplateSetAsDefaultOptions generates the options for
// setting a PermissionsTemplate as default based on the provided template ID.
func GeneratePermissionsTemplateSetAsDefaultOptions(templateID string) *sonar.PermissionsSetDefaultTemplateOptions {
	return &sonar.PermissionsSetDefaultTemplateOptions{
		TemplateID: templateID,
	}
}

// GeneratePermissionsTemplateGroupsSearchOptions generates the options for
// searching the groups permissions of a PermissionsTemplate based on the
// provided template ID and pagination options.
func GeneratePermissionsTemplateGroupsSearchOptions(templateID string, pagination *sonar.PaginationArgs) *sonar.PermissionsTemplateGroupsOptions {
	options := &sonar.PermissionsTemplateGroupsOptions{
		TemplateID: templateID,
	}

	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// GeneratePermissionsTemplateUsersSearchOptions generates the options for
// searching the users permissions of a PermissionsTemplate based on the
// provided template ID and pagination options.
func GeneratePermissionsTemplateUsersSearchOptions(templateID string, pagination *sonar.PaginationArgs) *sonar.PermissionsTemplateUsersOptions {
	options := &sonar.PermissionsTemplateUsersOptions{
		TemplateID: templateID,
	}

	helpers.AssignIfNonNil(&options.PaginationArgs, pagination)

	return options
}

// UpdatePermissionsTemplateObservation updates the
// PermissionsTemplateObservation based on the provided PermissionTemplate
// from the SonarQube API response and if the template is default or not.
func UpdatePermissionsTemplateObservation(observation *v1alpha1.PermissionsTemplateObservation, template *sonar.PermissionTemplate, isDefault bool) {
	observation.ID = template.ID
	observation.Name = template.Name
	observation.Description = template.Description
	observation.ProjectKeyPattern = template.ProjectKeyPattern
	observation.Default = isDefault
	observation.CreatedAt = helpers.StringToMetaTime(&template.CreatedAt)
	observation.UpdatedAt = helpers.StringToMetaTime(&template.UpdatedAt)

	// update creator permissions
	creatorPermissions := []string{}

	for _, permission := range template.Permissions {
		if permission.WithProjectCreator {
			creatorPermissions = append(creatorPermissions, permission.Key)
		}
	}

	observation.CreatorPermissions = creatorPermissions
}

// GeneratePermissionsTemplateGroupObservations generates a list of
// PermissionsTemplateGroupObservation based on the provided list of
// TemplateGroup from the SonarQube API response.
func GeneratePermissionsTemplateGroupObservations(groups *[]sonar.TemplateGroup) []v1alpha1.PermissionsTemplateGroupObservation {
	if groups == nil {
		return nil
	}

	observations := make([]v1alpha1.PermissionsTemplateGroupObservation, len(*groups))
	for i, group := range *groups {
		observations[i] = *GeneratePermissionsTemplateGroupObservation(&group)
	}

	return observations
}

// GeneratePermissionsTemplateGroupObservation generates a
// PermissionsTemplateGroupObservation based on the provided TemplateGroup
// from the SonarQube API response.
func GeneratePermissionsTemplateGroupObservation(group *sonar.TemplateGroup) *v1alpha1.PermissionsTemplateGroupObservation {
	return &v1alpha1.PermissionsTemplateGroupObservation{
		Name:        group.Name,
		Permissions: group.Permissions,
	}
}

// GeneratePermissionsTemplateUserObservations generates a list of
// PermissionsTemplateUserObservation based on the provided list of TemplateUser
// from the SonarQube API response.
func GeneratePermissionsTemplateUserObservations(users *[]sonar.TemplateUser) []v1alpha1.PermissionsTemplateUserObservation {
	if users == nil {
		return nil
	}

	observations := make([]v1alpha1.PermissionsTemplateUserObservation, len(*users))
	for i, user := range *users {
		observations[i] = *GeneratePermissionsTemplateUserObservation(&user)
	}

	return observations
}

// GeneratePermissionsTemplateUserObservation generates a
// PermissionsTemplateUserObservation based on the provided TemplateUser
// from the SonarQube API response.
func GeneratePermissionsTemplateUserObservation(user *sonar.TemplateUser) *v1alpha1.PermissionsTemplateUserObservation {
	return &v1alpha1.PermissionsTemplateUserObservation{
		Login:       user.Login,
		Permissions: user.Permissions,
	}
}

// GeneratePermissionsTemplateAddGroupPermissionOptions generates the options
// for adding a group permission to a PermissionsTemplate based on the provided
// template ID, group name and permission.
func GeneratePermissionsTemplateAddGroupPermissionOptions(templateID, groupName, permission string) *sonar.PermissionsAddGroupToTemplateOptions {
	return &sonar.PermissionsAddGroupToTemplateOptions{
		TemplateID: templateID,
		GroupName:  groupName,
		Permission: permission,
	}
}

// GeneratePermissionsTemplateRemoveGroupPermissionOptions generates the options
// for removing a group permission from a PermissionsTemplate based on the
// provided template ID, group name and permission.
func GeneratePermissionsTemplateRemoveGroupPermissionOptions(templateID, groupName, permission string) *sonar.PermissionsRemoveGroupFromTemplateOptions {
	return &sonar.PermissionsRemoveGroupFromTemplateOptions{
		TemplateID: templateID,
		GroupName:  groupName,
		Permission: permission,
	}
}

// GeneratePermissionsTemplateAddUserPermissionOptions generates the options for
// adding a user permission to a PermissionsTemplate based on the provided
// template ID, user login and permission.
func GeneratePermissionsTemplateAddUserPermissionOptions(templateID, userLogin, permission string) *sonar.PermissionsAddUserToTemplateOptions {
	return &sonar.PermissionsAddUserToTemplateOptions{
		TemplateID: templateID,
		Login:      userLogin,
		Permission: permission,
	}
}

// GeneratePermissionsTemplateRemoveUserPermissionOptions generates the options
// for removing a user permission from a PermissionsTemplate based on the
// provided template ID, user login and permission.
func GeneratePermissionsTemplateRemoveUserPermissionOptions(templateID, userLogin, permission string) *sonar.PermissionsRemoveUserFromTemplateOptions {
	return &sonar.PermissionsRemoveUserFromTemplateOptions{
		TemplateID: templateID,
		Login:      userLogin,
		Permission: permission,
	}
}

// GeneratePermissionsTemplateAddCreatorPermissionOptions creates options
// for adding a permission to a template creator.
func GeneratePermissionsTemplateAddCreatorPermissionOptions(templateID, permission string) *sonar.PermissionsAddProjectCreatorToTemplateOptions {
	return &sonar.PermissionsAddProjectCreatorToTemplateOptions{
		TemplateID: templateID,
		Permission: permission,
	}
}

// GeneratePermissionsTemplateRemoveCreatorPermissionOptions creates options
// for removing a permission from a template creator.
func GeneratePermissionsTemplateRemoveCreatorPermissionOptions(templateID, permission string) *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions {
	return &sonar.PermissionsRemoveProjectCreatorFromTemplateOptions{
		TemplateID: templateID,
		Permission: permission,
	}
}

// lenDereferencedStringSlice returns the length of a dereferenced
// string slice or 0 if nil.
func lenDereferencedStringSlice(values *[]string) int {
	if values == nil {
		return 0
	}

	return len(*values)
}
