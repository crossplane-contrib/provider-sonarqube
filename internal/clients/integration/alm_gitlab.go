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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ALMIntegrationsGitLabClient is the interface for interacting with SonarQube ALMGitLab API
// It handles all the operations related to ALMGitLab in SonarQube, such as creating, updating, deleting, and retrieving ALMGitLab.
type ALMIntegrationsGitLabClient interface {
	ALMIntegrationsClient
	SearchGitlabRepos(opt *sonar.AlmIntegrationsSearchGitlabReposOptions) (v *sonar.AlmIntegrationsSearchGitlabRepos, resp *http.Response, err error)
}

// ALMSettingsGitLabClient is the interface for interacting with SonarQube ALM settings API for GitLab
// It handles all the operations related to ALM settings for GitLab in SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsGitLabClient interface {
	ALMSettingsClient
	CreateGitlab(opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error)
	UpdateGitlab(opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error)
}

// NewALMIntegrationsGitLabClient creates a new ALMIntegrationsGitLabClient with the provided SonarQube client configuration.
func NewALMIntegrationsGitLabClient(clientConfig common.Config) ALMIntegrationsGitLabClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmIntegrations
}

// NewALMSettingsGitLabClient creates a new ALMSettingsGitLabClient with the provided SonarQube client configuration.
func NewALMSettingsGitLabClient(clientConfig common.Config) ALMSettingsGitLabClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// LateInitializeALMGitLab fills the empty fields in the ALMGitLab spec with the values from the SonarQube API response.
// The API response should be the result of a "Get" operation for the ALMGitLab resource.
func LateInitializeALMGitLab(spec *v1alpha1.ALMGitLabParameters, observation *v1alpha1.ALMGitLabObservation) {
	if spec == nil || observation == nil {
		return
	}

	LateInitializeALM(&spec.ALMCommonParameters, &observation.ALMCommonObservation)
}

// IsALMGitLabLateInitialized checks if two ALMGitLab specs are equal after late initialization. It returns true if the specs are equal, and false if they are not.
func IsALMGitLabLateInitialized(former, current *v1alpha1.ALMGitLabParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return IsALMLateInitialized(&former.ALMCommonParameters, &current.ALMCommonParameters)
}

// IsALMGitLabUpToDate checks if the ALMGitLab spec is up to date with the SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsALMGitLabUpToDate(spec *v1alpha1.ALMGitLabParameters, specAPIToken string, observation *v1alpha1.ALMGitLabObservation, savedAPIToken string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return IsALMUpToDate(&spec.ALMCommonParameters, specAPIToken, &observation.ALMCommonObservation, savedAPIToken)
}

// GenerateALMGitLabCreateOptions generates the options for creating an ALMGitLab resource in SonarQube API based on the desired state in the ALMGitLabParameters and the provided API token.
func GenerateALMGitLabCreateOptions(spec *v1alpha1.ALMGitLabParameters, apiToken string) *sonar.AlmSettingsCreateGitlabOptions {
	return &sonar.AlmSettingsCreateGitlabOptions{
		URL:                 spec.URL,
		Key:                 spec.Key,
		PersonalAccessToken: apiToken,
	}
}

// GenerateALMGitLabUpdateOptions generates the options for updating an ALMGitLab resource in SonarQube API based on the desired state in the ALMGitLabParameters, the provided API token, and the identifier of the ALMGitLab resource in SonarQube.
func GenerateALMGitLabUpdateOptions(key string, spec *v1alpha1.ALMGitLabParameters, apiToken string) *sonar.AlmSettingsUpdateGitlabOptions {
	updateOptions := sonar.AlmSettingsUpdateGitlabOptions{
		URL:                 spec.URL,
		Key:                 key,
		PersonalAccessToken: apiToken,
	}

	if spec.Key != key {
		updateOptions.NewKey = spec.Key
	}

	return &updateOptions
}

// FindGitLabALMDefinitionByKey searches for an ALM settings definition in the list of definitions by its key and devops platform. It returns the definition if found, and nil if not found.
func FindGitLabALMDefinitionByKey(definitions *[]sonar.GitlabDefinition, key string) *sonar.GitlabDefinition {
	if definitions == nil {
		return nil
	}

	for i := range *definitions {
		def := (*definitions)[i]
		if def.Key == key {
			return &def
		}
	}

	return nil
}

// GenerateALMGitLabObservation generates the ALMGitLabObservation based on the ALM settings definition retrieved from SonarQube API.
func GenerateALMGitLabObservation(definition *sonar.GitlabDefinition) v1alpha1.ALMGitLabObservation {
	if definition == nil {
		return v1alpha1.ALMGitLabObservation{}
	}

	return v1alpha1.ALMGitLabObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{
			URL: definition.URL,
			Key: definition.Key,
		},
	}
}
