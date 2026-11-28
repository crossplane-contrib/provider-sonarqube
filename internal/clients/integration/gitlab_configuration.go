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

package integration

import (
	"context"
	"net/http"
	"slices"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// GitLabConfigurationsClient is the interface for interacting with the
// SonarQube GitLab configuration API
// (api/v2/dop-translation/gitlab-configurations). It handles the GitLab
// authentication and provisioning configuration of a SonarQube instance.
type GitLabConfigurationsClient interface {
	SearchGitlabConfigurations(ctx context.Context) (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error)
	CreateGitlabConfiguration(ctx context.Context, opt *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	GetGitlabConfiguration(ctx context.Context, id string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	UpdateGitlabConfiguration(ctx context.Context, id string, opt *sonar.DopTranslationGitlabConfigurationUpdateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error)
	DeleteGitlabConfiguration(ctx context.Context, id string) (*http.Response, error)
}

// GitLabConfigurationCredentials are the write-only values of a GitLab
// configuration, which SonarQube never returns.
type GitLabConfigurationCredentials struct {
	// ApplicationSecret is the secret of the GitLab OAuth application.
	ApplicationSecret string
	// ProvisioningToken is the GitLab token used for auto-provisioning.
	ProvisioningToken string
}

// NewGitLabConfigurationsClient creates a new GitLabConfigurationsClient with
// the provided SonarQube client configuration. SonarQube holds at most one
// GitLab configuration, read by a single managed resource, so its responses
// are not cached.
func NewGitLabConfigurationsClient(clientConfig common.Config) GitLabConfigurationsClient {
	return common.NewClient(clientConfig).V2.DopTranslation
}

// gitLabProvisioningType returns the provisioning type of the spec,
// defaulting to JIT as the CRD does.
func gitLabProvisioningType(spec *v1alpha1.GitLabConfigurationParameters) string {
	if spec.ProvisioningType == "" {
		return v1alpha1.GitLabProvisioningTypeJIT
	}

	return spec.ProvisioningType
}

// LateInitializeGitLabConfiguration fills the empty fields in the
// GitLabConfiguration spec with the values observed in SonarQube.
func LateInitializeGitLabConfiguration(spec *v1alpha1.GitLabConfigurationParameters, observation *v1alpha1.GitLabConfigurationObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Enabled, observation.Enabled)
	helpers.AssignIfNil(&spec.SynchronizeGroups, observation.SynchronizeGroups)
	helpers.AssignIfNil(&spec.AllowAllGroups, observation.AllowAllGroups)
	helpers.AssignIfNil(&spec.AllowUsersToSignUp, observation.AllowUsersToSignUp)

	if spec.ProvisioningType == "" {
		spec.ProvisioningType = observation.ProvisioningType
	}

	if spec.AllowedGroups == nil && len(observation.AllowedGroups) > 0 {
		spec.AllowedGroups = slices.Clone(observation.AllowedGroups)
	}
}

// IsGitLabConfigurationLateInitialized reports whether late initialization
// changed the GitLabConfiguration spec.
func IsGitLabConfigurationLateInitialized(former, current *v1alpha1.GitLabConfigurationParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !helpers.IsComparablePtrEqualComparablePtr(former.Enabled, current.Enabled) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.SynchronizeGroups, current.SynchronizeGroups) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.AllowAllGroups, current.AllowAllGroups) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.AllowUsersToSignUp, current.AllowUsersToSignUp) ||
		former.ProvisioningType != current.ProvisioningType ||
		(former.AllowedGroups == nil) != (current.AllowedGroups == nil)
}

// IsGitLabConfigurationUpToDate reports whether the GitLabConfiguration spec
// matches the configuration observed in SonarQube.
//
// SonarQube never returns the application secret nor the provisioning token,
// so the values read from the spec's secret references (current) are
// compared with those last applied by the provider (saved). The provisioning
// token is only compared when the spec references one: the SonarQube API
// cannot remove a provisioning token.
func IsGitLabConfigurationUpToDate(spec *v1alpha1.GitLabConfigurationParameters, observation *v1alpha1.GitLabConfigurationObservation, current, saved GitLabConfigurationCredentials) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return areGitLabConfigurationFieldsUpToDate(spec, observation) &&
		areGitLabConfigurationCredentialsUpToDate(spec, observation, current, saved)
}

// areGitLabConfigurationFieldsUpToDate reports whether the readable fields
// of the spec match the observation. Unset optional fields match any value.
func areGitLabConfigurationFieldsUpToDate(spec *v1alpha1.GitLabConfigurationParameters, observation *v1alpha1.GitLabConfigurationObservation) bool {
	return helpers.IsComparablePtrEqualComparable(spec.Enabled, observation.Enabled) &&
		spec.ApplicationID == observation.ApplicationID &&
		spec.URL == observation.URL &&
		helpers.IsComparablePtrEqualComparable(spec.SynchronizeGroups, observation.SynchronizeGroups) &&
		(spec.AllowedGroups == nil || helpers.AreStringSlicesEqualDeDuped(spec.AllowedGroups, observation.AllowedGroups)) &&
		helpers.IsComparablePtrEqualComparable(spec.AllowAllGroups, observation.AllowAllGroups) &&
		helpers.IsComparablePtrEqualComparable(spec.AllowUsersToSignUp, observation.AllowUsersToSignUp) &&
		gitLabProvisioningType(spec) == observation.ProvisioningType
}

// areGitLabConfigurationCredentialsUpToDate reports whether the write-only
// values were last applied with their current value.
func areGitLabConfigurationCredentialsUpToDate(spec *v1alpha1.GitLabConfigurationParameters, observation *v1alpha1.GitLabConfigurationObservation, current, saved GitLabConfigurationCredentials) bool {
	if current.ApplicationSecret != saved.ApplicationSecret {
		return false
	}

	if spec.ProvisioningTokenSecretRef == nil {
		return true
	}

	return observation.IsProvisioningTokenSet && current.ProvisioningToken == saved.ProvisioningToken
}

// GenerateGitLabConfigurationCreateOptions generates the SonarQube API
// options for creating a GitLab configuration from the spec and the values
// of its secret references.
func GenerateGitLabConfigurationCreateOptions(spec *v1alpha1.GitLabConfigurationParameters, credentials GitLabConfigurationCredentials) *sonar.DopTranslationGitlabConfigurationCreateOptions {
	if spec == nil {
		return nil
	}

	// allowedGroups is required by SonarQube, as a list: send an empty one
	// rather than null.
	allowedGroups := slices.Clone(spec.AllowedGroups)
	if allowedGroups == nil {
		allowedGroups = []string{}
	}

	return &sonar.DopTranslationGitlabConfigurationCreateOptions{
		Enabled:            ptr.Deref(spec.Enabled, true),
		ApplicationId:      spec.ApplicationID,
		Url:                spec.URL,
		Secret:             credentials.ApplicationSecret,
		SynchronizeGroups:  ptr.Deref(spec.SynchronizeGroups, false),
		AllowedGroups:      allowedGroups,
		AllowAllGroups:     spec.AllowAllGroups,
		ProvisioningType:   gitLabProvisioningType(spec),
		ProvisioningToken:  credentials.ProvisioningToken,
		AllowUsersToSignUp: spec.AllowUsersToSignUp,
	}
}

// GenerateGitLabConfigurationUpdateOptions generates the SonarQube API
// options for updating a GitLab configuration from the spec and the values
// of its secret references. Unset optional fields are left unchanged.
func GenerateGitLabConfigurationUpdateOptions(spec *v1alpha1.GitLabConfigurationParameters, credentials GitLabConfigurationCredentials) *sonar.DopTranslationGitlabConfigurationUpdateOptions {
	if spec == nil {
		return nil
	}

	options := &sonar.DopTranslationGitlabConfigurationUpdateOptions{
		Enabled:            spec.Enabled,
		ApplicationId:      spec.ApplicationID,
		Url:                spec.URL,
		Secret:             credentials.ApplicationSecret,
		SynchronizeGroups:  spec.SynchronizeGroups,
		AllowAllGroups:     spec.AllowAllGroups,
		ProvisioningType:   gitLabProvisioningType(spec),
		AllowUsersToSignUp: spec.AllowUsersToSignUp,
		ProvisioningToken:  credentials.ProvisioningToken,
	}

	if spec.AllowedGroups != nil {
		options.AllowedGroups = &sonar.UpdateFieldListStringV2{Value: slices.Clone(spec.AllowedGroups), Defined: true}
	}

	return options
}

// GenerateGitLabConfigurationObservation generates the
// GitLabConfigurationObservation from the configuration returned by
// SonarQube.
func GenerateGitLabConfigurationObservation(configuration *sonar.DopTranslationGitlabConfiguration) v1alpha1.GitLabConfigurationObservation {
	if configuration == nil {
		return v1alpha1.GitLabConfigurationObservation{}
	}

	return v1alpha1.GitLabConfigurationObservation{
		ID:                     configuration.Id,
		Enabled:                configuration.Enabled,
		ApplicationID:          configuration.ApplicationId,
		URL:                    configuration.Url,
		SynchronizeGroups:      configuration.SynchronizeGroups,
		AllowedGroups:          slices.Clone(configuration.AllowedGroups),
		AllowAllGroups:         configuration.AllowAllGroups,
		AllowUsersToSignUp:     configuration.AllowUsersToSignUp,
		ProvisioningType:       configuration.ProvisioningType,
		IsProvisioningTokenSet: configuration.IsProvisioningTokenSet,
		ErrorMessage:           ptr.Deref(configuration.ErrorMessage, ""),
	}
}
