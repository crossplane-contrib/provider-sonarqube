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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ALMIntegrationsBitbucketClient is the interface for interacting with
// SonarQube ALMBitbucket API
// It handles all the operations related to ALMBitbucket in SonarQube,
// such as creating, updating, deleting, and retrieving ALMBitbucket.
//
//nolint:dupl // Intentional structural similarity with ALMIntegrationsGitHubClient; different provider-specific API methods prevent abstraction.
type ALMIntegrationsBitbucketClient interface {
	ALMIntegrationsClient
	ListBitbucketServerProjects(ctx context.Context, opt *sonar.AlmIntegrationsListBitbucketServerProjectsOptions) (v *sonar.AlmIntegrationsListBitbucketServerProjects, resp *http.Response, err error)
	SearchBitbucketCloudRepos(ctx context.Context, opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error)
	SearchBitbucketServerRepos(ctx context.Context, opt *sonar.AlmIntegrationsSearchBitbucketServerReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketServerRepos, resp *http.Response, err error)
}

// ALMSettingsBitbucketClient is the interface for interacting with SonarQube
// ALM settings API for Bitbucket
// It handles all the operations related to ALM settings for Bitbucket
// in SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsBitbucketClient interface {
	ALMSettingsClient
	CreateBitbucket(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error)
	UpdateBitbucket(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error)
}

// NewALMIntegrationsBitbucketClient creates a new
// ALMIntegrationsBitbucketClient with the provided
// SonarQube client configuration.
func NewALMIntegrationsBitbucketClient(clientConfig common.Config) ALMIntegrationsBitbucketClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmIntegrations
}

// NewALMSettingsBitbucketClient creates a new ALMSettingsBitbucketClient
// with the provided SonarQube client configuration.
func NewALMSettingsBitbucketClient(clientConfig common.Config) ALMSettingsBitbucketClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// LateInitializeALMBitbucket fills the empty fields in the ALMBitbucket
// spec with the values from the SonarQube API response.
// The API response should be the result of a "Get" operation for the
// ALMBitbucket resource.
func LateInitializeALMBitbucket(spec *v1alpha1.ALMBitbucketParameters, observation *v1alpha1.ALMBitbucketObservation) {
	if spec == nil || observation == nil {
		return
	}

	LateInitializeALM(&spec.ALMCommonParameters, &observation.ALMCommonObservation)
}

// IsALMBitbucketLateInitialized checks if two ALMBitbucket specs are equal
// after late initialization. It returns true if the specs are equal,
// and false if they are not.
func IsALMBitbucketLateInitialized(former, current *v1alpha1.ALMBitbucketParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return IsALMLateInitialized(&former.ALMCommonParameters, &current.ALMCommonParameters)
}

// IsALMBitbucketUpToDate checks if the ALMBitbucket spec is up to date
// with the SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsALMBitbucketUpToDate(spec *v1alpha1.ALMBitbucketParameters, specAPIToken string, observation *v1alpha1.ALMBitbucketObservation, savedAPIToken string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return IsALMUpToDate(&spec.ALMCommonParameters, specAPIToken, &observation.ALMCommonObservation, savedAPIToken)
}

// GenerateALMBitbucketCreateOptions generates the options for creating an
// ALMBitbucket resource in SonarQube API based on the desired state in the
// ALMBitbucketParameters and the provided API token.
func GenerateALMBitbucketCreateOptions(spec *v1alpha1.ALMBitbucketParameters, apiToken string) *sonar.AlmSettingsCreateBitbucketOptions {
	return &sonar.AlmSettingsCreateBitbucketOptions{
		URL:                 spec.URL,
		Key:                 spec.Key,
		PersonalAccessToken: apiToken,
	}
}

// GenerateALMBitbucketUpdateOptions generates the options for updating an
// ALMBitbucket resource in SonarQube API based on the desired state in the
// ALMBitbucketParameters, the provided API token, and the identifier of
// the ALMBitbucket resource in SonarQube.
func GenerateALMBitbucketUpdateOptions(key string, spec *v1alpha1.ALMBitbucketParameters, apiToken string) *sonar.AlmSettingsUpdateBitbucketOptions {
	updateOptions := sonar.AlmSettingsUpdateBitbucketOptions{
		URL:                 spec.URL,
		Key:                 key,
		PersonalAccessToken: apiToken,
	}

	if spec.Key != key {
		updateOptions.NewKey = spec.Key
	}

	return &updateOptions
}

// FindBitbucketALMDefinitionByKey searches for an ALM settings definition
// in the list of definitions by its key.
// It returns the definition if found, and nil if not found.
func FindBitbucketALMDefinitionByKey(definitions *[]sonar.BitbucketDefinition, key string) *sonar.BitbucketDefinition {
	if definitions == nil {
		return nil
	}

	for i := range *definitions {
		if (*definitions)[i].Key == key {
			return &(*definitions)[i]
		}
	}

	return nil
}

// GenerateALMBitbucketObservation generates the ALMBitbucketObservation
// based on the ALM settings definition retrieved from SonarQube API.
func GenerateALMBitbucketObservation(definition *sonar.BitbucketDefinition) v1alpha1.ALMBitbucketObservation {
	if definition == nil {
		return v1alpha1.ALMBitbucketObservation{}
	}

	return v1alpha1.ALMBitbucketObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{
			URL: definition.URL,
			Key: definition.Key,
		},
	}
}
