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
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// GitHubConfigurationsClient is the interface for interacting with the
// SonarQube V2 GitHub configurations API
// (api/v2/dop-translation/github-configurations), which manages the GitHub
// authentication and provisioning configuration of the instance.
type GitHubConfigurationsClient interface {
	SearchGithubConfigurations(ctx context.Context) (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error)
	CreateGithubConfiguration(ctx context.Context, opt *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error)
	UpdateGithubConfiguration(ctx context.Context, id string, opt *sonar.DopTranslationGithubConfigurationUpdateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error)
	DeleteGithubConfiguration(ctx context.Context, id string) (*http.Response, error)
}

// NewGitHubConfigurationsClient creates a new GitHubConfigurationsClient
// with the provided SonarQube client configuration.
// It is not cached: SonarQube supports a single GitHub configuration per
// instance, so there is no dataset to share between resources.
func NewGitHubConfigurationsClient(clientConfig common.Config) GitHubConfigurationsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.V2.DopTranslation
}

// FindGitHubConfiguration returns the configuration with the given ID, or,
// when id is empty, the first configuration, since SonarQube supports a
// single one per instance. It returns nil if there is no such configuration.
func FindGitHubConfiguration(configurations []sonar.DopTranslationGithubConfiguration, id string) *sonar.DopTranslationGithubConfiguration {
	for i := range configurations {
		if id == "" || configurations[i].Id == id {
			return &configurations[i]
		}
	}

	return nil
}

// GenerateGitHubConfigurationObservation generates the
// GitHubConfigurationObservation of the given SonarQube configuration.
func GenerateGitHubConfigurationObservation(configuration *sonar.DopTranslationGithubConfiguration) v1alpha1.GitHubConfigurationObservation {
	if configuration == nil {
		return v1alpha1.GitHubConfigurationObservation{}
	}

	return v1alpha1.GitHubConfigurationObservation{
		ID:                              configuration.Id,
		Enabled:                         configuration.Enabled,
		ApplicationID:                   configuration.ApplicationId,
		APIURL:                          configuration.ApiUrl,
		WebURL:                          configuration.WebUrl,
		AllowedOrganizations:            configuration.AllowedOrganizations,
		ProvisioningType:                configuration.ProvisioningType,
		SynchronizeGroups:               configuration.SynchronizeGroups,
		AllowUsersToSignUp:              configuration.AllowUsersToSignUp,
		ProjectVisibility:               configuration.ProjectVisibility,
		UserConsentRequiredAfterUpgrade: configuration.UserConsentRequiredAfterUpgrade,
		ErrorMessage:                    ptr.Deref(configuration.ErrorMessage, ""),
	}
}

// LateInitializeGitHubConfiguration fills the unset optional fields of the
// GitHubConfiguration spec with the values observed in SonarQube.
func LateInitializeGitHubConfiguration(spec *v1alpha1.GitHubConfigurationParameters, observation *v1alpha1.GitHubConfigurationObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Enabled, observation.Enabled)
	helpers.AssignIfNil(&spec.SynchronizeGroups, observation.SynchronizeGroups)
	helpers.AssignIfNil(&spec.AllowUsersToSignUp, observation.AllowUsersToSignUp)
	helpers.AssignIfNil(&spec.ProjectVisibility, observation.ProjectVisibility)
	helpers.AssignIfNil(&spec.UserConsentRequiredAfterUpgrade, observation.UserConsentRequiredAfterUpgrade)

	if spec.ProvisioningType == "" {
		spec.ProvisioningType = observation.ProvisioningType
	}

	if spec.AllowedOrganizations == nil && len(observation.AllowedOrganizations) > 0 {
		spec.AllowedOrganizations = append([]string(nil), observation.AllowedOrganizations...)
	}
}

// IsGitHubConfigurationLateInitialized reports whether late initialization
// changed the GitHubConfiguration spec.
func IsGitHubConfigurationLateInitialized(former, current *v1alpha1.GitHubConfigurationParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !helpers.IsComparablePtrEqualComparablePtr(former.Enabled, current.Enabled) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.SynchronizeGroups, current.SynchronizeGroups) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.AllowUsersToSignUp, current.AllowUsersToSignUp) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.ProjectVisibility, current.ProjectVisibility) ||
		!helpers.IsComparablePtrEqualComparablePtr(former.UserConsentRequiredAfterUpgrade, current.UserConsentRequiredAfterUpgrade) ||
		former.ProvisioningType != current.ProvisioningType ||
		(former.AllowedOrganizations == nil) != (current.AllowedOrganizations == nil)
}

// GitHubConfigurationCredentials are the GitHub App credentials that
// SonarQube does not return: the values resolved from the spec, or the
// values last applied, saved to the connection secret.
type GitHubConfigurationCredentials struct {
	// ClientID is the client ID of the GitHub App.
	ClientID string
	// ClientSecret is the client secret of the GitHub App.
	ClientSecret string
	// PrivateKey is the private key of the GitHub App.
	PrivateKey string
}

// IsGitHubConfigurationUpToDate reports whether the GitHub configuration
// observed in SonarQube matches the spec. The credentials SonarQube does
// not return are compared with the ones last applied.
func IsGitHubConfigurationUpToDate(spec *v1alpha1.GitHubConfigurationParameters, observation *v1alpha1.GitHubConfigurationObservation, desired, saved GitHubConfigurationCredentials) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.ApplicationID == observation.ApplicationID &&
		isSameURL(spec.APIURL, observation.APIURL) &&
		isSameURL(spec.WebURL, observation.WebURL) &&
		areGitHubConfigurationProvisioningSettingsUpToDate(spec, observation) &&
		areGitHubConfigurationTogglesUpToDate(spec, observation) &&
		desired == saved
}

// areGitHubConfigurationProvisioningSettingsUpToDate reports whether the
// allowed organizations and provisioning type set in the spec match the
// observed ones.
func areGitHubConfigurationProvisioningSettingsUpToDate(spec *v1alpha1.GitHubConfigurationParameters, observation *v1alpha1.GitHubConfigurationObservation) bool {
	return (spec.AllowedOrganizations == nil || helpers.AreStringSlicesEqualDeDuped(spec.AllowedOrganizations, observation.AllowedOrganizations)) &&
		(spec.ProvisioningType == "" || spec.ProvisioningType == observation.ProvisioningType)
}

// areGitHubConfigurationTogglesUpToDate reports whether the boolean settings
// set in the spec match the observed ones.
func areGitHubConfigurationTogglesUpToDate(spec *v1alpha1.GitHubConfigurationParameters, observation *v1alpha1.GitHubConfigurationObservation) bool {
	return helpers.IsComparablePtrEqualComparable(spec.Enabled, observation.Enabled) &&
		helpers.IsComparablePtrEqualComparable(spec.SynchronizeGroups, observation.SynchronizeGroups) &&
		helpers.IsComparablePtrEqualComparable(spec.AllowUsersToSignUp, observation.AllowUsersToSignUp) &&
		helpers.IsComparablePtrEqualComparable(spec.ProjectVisibility, observation.ProjectVisibility) &&
		helpers.IsComparablePtrEqualComparable(spec.UserConsentRequiredAfterUpgrade, observation.UserConsentRequiredAfterUpgrade)
}

// isSameURL reports whether two URLs are equal, ignoring a trailing slash.
func isSameURL(urlA, urlB string) bool {
	return strings.TrimSuffix(urlA, "/") == strings.TrimSuffix(urlB, "/")
}

// GenerateGitHubConfigurationCreateOptions generates the options for
// creating the GitHub configuration in SonarQube. Unset fields that
// SonarQube requires get their defaults: enabled, JIT provisioning, no group
// synchronization and no organization restriction.
func GenerateGitHubConfigurationCreateOptions(spec *v1alpha1.GitHubConfigurationParameters, credentials GitHubConfigurationCredentials) *sonar.DopTranslationGithubConfigurationCreateOptions {
	if spec == nil {
		return nil
	}

	provisioningType := spec.ProvisioningType
	if provisioningType == "" {
		provisioningType = v1alpha1.GitHubProvisioningTypeJIT
	}

	return &sonar.DopTranslationGithubConfigurationCreateOptions{
		Enabled:                         ptr.Deref(spec.Enabled, true),
		ClientId:                        credentials.ClientID,
		ClientSecret:                    credentials.ClientSecret,
		ApplicationId:                   spec.ApplicationID,
		PrivateKey:                      credentials.PrivateKey,
		SynchronizeGroups:               ptr.Deref(spec.SynchronizeGroups, false),
		ApiUrl:                          spec.APIURL,
		WebUrl:                          spec.WebURL,
		AllowedOrganizations:            append([]string{}, spec.AllowedOrganizations...),
		ProvisioningType:                provisioningType,
		AllowUsersToSignUp:              spec.AllowUsersToSignUp,
		ProjectVisibility:               spec.ProjectVisibility,
		UserConsentRequiredAfterUpgrade: spec.UserConsentRequiredAfterUpgrade,
	}
}

// GenerateGitHubConfigurationUpdateOptions generates the options for
// updating the GitHub configuration in SonarQube. Every field set in the
// spec is sent, credentials included: SonarQube requires the private key
// when the URLs change, and cannot tell whether the others changed.
func GenerateGitHubConfigurationUpdateOptions(spec *v1alpha1.GitHubConfigurationParameters, credentials GitHubConfigurationCredentials) *sonar.DopTranslationGithubConfigurationUpdateOptions {
	if spec == nil {
		return nil
	}

	opts := &sonar.DopTranslationGithubConfigurationUpdateOptions{
		Enabled:                         spec.Enabled,
		ClientId:                        credentials.ClientID,
		ClientSecret:                    credentials.ClientSecret,
		ApplicationId:                   spec.ApplicationID,
		PrivateKey:                      credentials.PrivateKey,
		SynchronizeGroups:               spec.SynchronizeGroups,
		ApiUrl:                          spec.APIURL,
		WebUrl:                          spec.WebURL,
		ProvisioningType:                spec.ProvisioningType,
		AllowUsersToSignUp:              spec.AllowUsersToSignUp,
		ProjectVisibility:               spec.ProjectVisibility,
		UserConsentRequiredAfterUpgrade: spec.UserConsentRequiredAfterUpgrade,
	}

	if spec.AllowedOrganizations != nil {
		opts.AllowedOrganizations = &sonar.UpdateFieldListStringV2{
			Value:   append([]string{}, spec.AllowedOrganizations...),
			Defined: true,
		}
	}

	return opts
}
