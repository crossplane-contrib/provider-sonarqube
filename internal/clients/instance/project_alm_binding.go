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
	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// GenerateProjectALMBindingObservation generates the observation for a
// Project's ALM binding based on the provided SonarQube API response.
func GenerateProjectALMBindingObservation(binding *sonar.AlmSettingsGetBinding) *v1alpha1.ProjectALMBindingObservation {
	if binding == nil {
		return nil
	}

	return &v1alpha1.ProjectALMBindingObservation{
		Alm:                      binding.Alm,
		Key:                      binding.Key,
		Repository:               binding.Repository,
		RepositoryURL:            binding.RepositoryURL,
		Slug:                     binding.Slug,
		Monorepo:                 binding.Monorepo,
		InlineAnnotationsEnabled: binding.InlineAnnotationsEnabled,
		SummaryCommentEnabled:    binding.SummaryCommentEnabled,
	}
}

// GenerateProjectALMBindingGetOptions generates the options for retrieving
// a Project's ALM binding based on the provided project key.
func GenerateProjectALMBindingGetOptions(projectKey string) *sonar.AlmSettingsGetBindingOptions {
	return &sonar.AlmSettingsGetBindingOptions{
		Project: projectKey,
	}
}

// GenerateProjectALMBindingDeleteOptions generates the options for deleting
// a Project's ALM binding based on the provided project key.
func GenerateProjectALMBindingDeleteOptions(projectKey string) *sonar.AlmSettingsDeleteBindingOptions {
	return &sonar.AlmSettingsDeleteBindingOptions{
		Project: projectKey,
	}
}

// GenerateProjectGitHubBindingOptions generates the options for binding a
// Project to a GitHub repository based on the provided project key and
// binding spec.
func GenerateProjectGitHubBindingOptions(projectKey string, binding *v1alpha1.ProjectALMBindingParameters) *sonar.AlmSettingsSetGithubBindingOptions {
	return &sonar.AlmSettingsSetGithubBindingOptions{
		AlmSetting:            binding.GitHub.AlmSettingKey,
		Project:               projectKey,
		Repository:            binding.GitHub.Repository,
		Monorepo:              binding.Monorepo,
		SummaryCommentEnabled: binding.GitHub.SummaryCommentEnabled,
	}
}

// GenerateProjectGitLabBindingOptions generates the options for binding a
// Project to a GitLab project based on the provided project key and
// binding spec.
func GenerateProjectGitLabBindingOptions(projectKey string, binding *v1alpha1.ProjectALMBindingParameters) *sonar.AlmSettingsSetGitlabBindingOptions {
	return &sonar.AlmSettingsSetGitlabBindingOptions{
		AlmSetting: binding.GitLab.AlmSettingKey,
		Project:    projectKey,
		Repository: binding.GitLab.Repository,
		Monorepo:   binding.Monorepo,
	}
}

// GenerateProjectAzureBindingOptions generates the options for binding a
// Project to an Azure DevOps repository based on the provided project key
// and binding spec.
func GenerateProjectAzureBindingOptions(projectKey string, binding *v1alpha1.ProjectALMBindingParameters) *sonar.AlmSettingsSetAzureBindingOptions {
	return &sonar.AlmSettingsSetAzureBindingOptions{
		AlmSetting:               binding.Azure.AlmSettingKey,
		Project:                  projectKey,
		ProjectName:              binding.Azure.ProjectName,
		RepositoryName:           binding.Azure.RepositoryName,
		Monorepo:                 binding.Monorepo,
		InlineAnnotationsEnabled: binding.Azure.InlineAnnotationsEnabled,
	}
}

// GenerateProjectBitbucketBindingOptions generates the options for binding
// a Project to a Bitbucket Server repository based on the provided project
// key and binding spec.
func GenerateProjectBitbucketBindingOptions(projectKey string, binding *v1alpha1.ProjectALMBindingParameters) *sonar.AlmSettingsSetBitbucketBindingOptions {
	return &sonar.AlmSettingsSetBitbucketBindingOptions{
		AlmSetting: binding.Bitbucket.AlmSettingKey,
		Project:    projectKey,
		Repository: binding.Bitbucket.Repository,
		Slug:       binding.Bitbucket.Slug,
		Monorepo:   binding.Monorepo,
	}
}

// GenerateProjectBitbucketCloudBindingOptions generates the options for
// binding a Project to a Bitbucket Cloud repository based on the provided
// project key and binding spec.
func GenerateProjectBitbucketCloudBindingOptions(projectKey string, binding *v1alpha1.ProjectALMBindingParameters) *sonar.AlmSettingsSetBitbucketCloudBindingOptions {
	return &sonar.AlmSettingsSetBitbucketCloudBindingOptions{
		AlmSetting: binding.BitbucketCloud.AlmSettingKey,
		Project:    projectKey,
		Repository: binding.BitbucketCloud.Repository,
		Monorepo:   binding.Monorepo,
	}
}

// ALM type identifiers as reported by the SonarQube get_binding endpoint.
const (
	almTypeGitHub         = "github"
	almTypeGitLab         = "gitlab"
	almTypeAzure          = "azure"
	almTypeBitbucket      = "bitbucket"
	almTypeBitbucketCloud = "bitbucketcloud"
)

// IsProjectALMBindingUpToDate compares the desired state of a Project's ALM
// binding in the spec with the observed state of the binding from
// SonarQube.
// Returns true if they are up to date, or false if they are not.
func IsProjectALMBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	if spec == nil {
		return observed == nil
	}

	if observed == nil {
		return false
	}

	switch {
	case spec.GitHub != nil:
		return isGitHubBindingUpToDate(spec, observed)
	case spec.GitLab != nil:
		return isGitLabBindingUpToDate(spec, observed)
	case spec.Azure != nil:
		return isAzureBindingUpToDate(spec, observed)
	case spec.Bitbucket != nil:
		return isBitbucketBindingUpToDate(spec, observed)
	case spec.BitbucketCloud != nil:
		return isBitbucketCloudBindingUpToDate(spec, observed)
	default:
		return false
	}
}

// isGitHubBindingUpToDate compares the GitHub binding spec against the
// observed binding.
func isGitHubBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	return observed.Alm == almTypeGitHub &&
		observed.Key == spec.GitHub.AlmSettingKey &&
		observed.Repository == spec.GitHub.Repository &&
		observed.Monorepo == spec.Monorepo &&
		helpers.IsComparablePtrEqualComparable(spec.GitHub.SummaryCommentEnabled, observed.SummaryCommentEnabled)
}

// isGitLabBindingUpToDate compares the GitLab binding spec against the
// observed binding.
func isGitLabBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	return observed.Alm == almTypeGitLab &&
		observed.Key == spec.GitLab.AlmSettingKey &&
		observed.Repository == spec.GitLab.Repository &&
		observed.Monorepo == spec.Monorepo
}

// isAzureBindingUpToDate compares the Azure binding spec against the
// observed binding. The SonarQube get_binding response has no dedicated
// ProjectName/RepositoryName fields for Azure, so a change to either can't
// be detected here and won't trigger a re-bind.
func isAzureBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	return observed.Alm == almTypeAzure &&
		observed.Key == spec.Azure.AlmSettingKey &&
		observed.Monorepo == spec.Monorepo &&
		helpers.IsComparablePtrEqualComparable(spec.Azure.InlineAnnotationsEnabled, observed.InlineAnnotationsEnabled)
}

// isBitbucketBindingUpToDate compares the Bitbucket Server binding spec
// against the observed binding.
func isBitbucketBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	return observed.Alm == almTypeBitbucket &&
		observed.Key == spec.Bitbucket.AlmSettingKey &&
		observed.Repository == spec.Bitbucket.Repository &&
		observed.Slug == spec.Bitbucket.Slug &&
		observed.Monorepo == spec.Monorepo
}

// isBitbucketCloudBindingUpToDate compares the Bitbucket Cloud binding spec
// against the observed binding.
func isBitbucketCloudBindingUpToDate(spec *v1alpha1.ProjectALMBindingParameters, observed *v1alpha1.ProjectALMBindingObservation) bool {
	return observed.Alm == almTypeBitbucketCloud &&
		observed.Key == spec.BitbucketCloud.AlmSettingKey &&
		observed.Repository == spec.BitbucketCloud.Repository &&
		observed.Monorepo == spec.Monorepo
}
