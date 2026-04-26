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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// UsersClient is the interface for interacting with the SonarQube v2 user API.
type UsersClient interface {
	Create(opt *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error)
	Deactivate(opt *sonar.UsersDeactivateOptionsV2) (*http.Response, error)
	Fetch(userID string) (*sonar.UserV2, *http.Response, error)
	Search(opt *sonar.UsersSearchOptionV2) (*sonar.UsersSearchV2, *http.Response, error)
	Update(userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error)
}

// NewUsersClient creates a new UsersClient with the provided SonarQube
// client configuration.
func NewUsersClient(clientConfig common.Config) UsersClient {
	newClient := common.NewClient(clientConfig)

	return newClient.V2.UsersManagement
}

// LateInitializeUser fills the empty fields in the User spec with the
// values from the SonarQube API response.
func LateInitializeUser(spec *v1alpha1.UserParameters, observation *v1alpha1.UserObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Email, observation.Email)
	helpers.AssignIfNil(&spec.Local, observation.Local)

	if spec.ScmAccounts == nil && len(observation.ScmAccounts) > 0 {
		accounts := append([]string(nil), observation.ScmAccounts...)
		spec.ScmAccounts = &accounts
	}
}

// IsUserLateInitialized checks if two User specs are equal after late
// initialization.
func IsUserLateInitialized(former, current *v1alpha1.UserParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return former.Login != current.Login ||
		former.Name != current.Name ||
		!helpers.IsComparablePtrEqualComparablePtr(former.Email, current.Email) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.Local, current.Local) ||
		!AreUserScmAccountsUpToDate(former.ScmAccounts, current.ScmAccounts)
}

// IsUserUpToDate checks if the User spec is up to date with the
// SonarQube API response.
func IsUserUpToDate(spec *v1alpha1.UserParameters, observation *v1alpha1.UserObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.Login == observation.Login &&
		spec.Name == observation.Name &&
		helpers.IsComparablePtrEqualComparablePtr(spec.Email, &observation.Email) &&
		helpers.IsComparablePtrEqualComparablePtr(spec.Local, &observation.Local) &&
		AreUserScmAccountsUpToDate(spec.ScmAccounts, &observation.ScmAccounts) &&
		AreUserGroupsUpToDate(spec.Groups, &observation.Groups)
}

// AreUserScmAccountsUpToDate checks if the User's SCM accounts are up to date
// with the SonarQube API response.
func AreUserScmAccountsUpToDate(spec *[]string, observation *[]string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return helpers.AreStringSlicesEqualDeDuped(*spec, *observation)
}

// AreUserGroupsUpToDate checks if the User's group associations are up to date
// with the SonarQube API response.
func AreUserGroupsUpToDate(spec *[]v1alpha1.UserGroupsParameters, observation *map[string]string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	specGroups := make([]string, 0, len(*spec))
	for _, group := range *spec {
		if group.GroupId == nil || *group.GroupId == "" {
			continue
		}

		specGroups = append(specGroups, *group.GroupId)
	}

	observationGroups := make([]string, 0, len(*observation))
	for groupID := range *observation {
		observationGroups = append(observationGroups, groupID)
	}

	return helpers.AreStringSlicesEqualDeDuped(specGroups, observationGroups)
}

// GenerateCreateUserOptions generates the SonarQube API options for
// creating a user.
func GenerateCreateUserOptions(spec *v1alpha1.UserParameters, password *string) *sonar.UsersCreateOptionsV2 {
	if spec == nil {
		return nil
	}

	options := &sonar.UsersCreateOptionsV2{
		Login: spec.Login,
		Name:  spec.Name,
	}

	helpers.AssignIfNonNil(&options.Password, password)
	helpers.AssignIfNonNil(&options.Email, spec.Email)

	if spec.Local != nil {
		local := *spec.Local
		options.Local = &local
	}

	helpers.AssignIfNonNil(&options.ScmAccounts, spec.ScmAccounts)

	return options
}

// GenerateUpdateUserOptions generates the SonarQube API options for
// updating a user.
func GenerateUpdateUserOptions(spec *v1alpha1.UserParameters) *sonar.UsersUpdateOptionsV2 {
	if spec == nil {
		return nil
	}

	options := &sonar.UsersUpdateOptionsV2{
		Login: spec.Login,
		Name:  spec.Name,
	}

	helpers.AssignIfNonNil(&options.Email, spec.Email)
	helpers.AssignIfNonNil(&options.ExternalId, spec.ExternalId)
	helpers.AssignIfNonNil(&options.ExternalLogin, spec.ExternalLogin)
	helpers.AssignIfNonNil(&options.ExternalProvider, spec.ExternalProvider)

	if spec.ScmAccounts != nil {
		options.ScmAccounts = &sonar.UpdateFieldListStringV2{
			Value:   append([]string(nil), (*spec.ScmAccounts)...),
			Defined: true,
		}
	}

	return options
}

// GenerateUserObservation generates the UserObservation from the SonarQube API
// response.
func GenerateUserObservation(user *sonar.UserV2) v1alpha1.UserObservation {
	if user == nil {
		return v1alpha1.UserObservation{}
	}

	return v1alpha1.UserObservation{
		Id:                          user.Id,
		Active:                      user.Active,
		Email:                       user.Email,
		ExternalId:                  user.ExternalId,
		ExternalLogin:               user.ExternalLogin,
		ExternalProvider:            user.ExternalProvider,
		Local:                       user.Local,
		Login:                       user.Login,
		Managed:                     user.Managed,
		Name:                        user.Name,
		ScmAccounts:                 append([]string(nil), user.ScmAccounts...),
		SonarLintLastConnectionDate: user.SonarLintLastConnectionDate,
		SonarQubeLastConnectionDate: user.SonarQubeLastConnectionDate,
		Avatar:                      user.Avatar,
	}
}
