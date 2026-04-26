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

// ALMIntegrationsGitHubClient is the interface for interacting with SonarQube ALMGitHub API
// It handles all the operations related to ALMGitHub in SonarQube, such as creating, updating, deleting, and retrieving ALMGitHub.
type ALMIntegrationsGitHubClient interface {
	ALMIntegrationsClient
	GetGithubClientId(opt *sonar.AlmIntegrationsGetGithubClientIdOptions) (v *sonar.AlmIntegrationsGetGithubClientId, resp *http.Response, err error)
	ListGithubOrganizations(opt *sonar.AlmIntegrationsListGithubOrganizationsOptions) (v *sonar.AlmIntegrationsListGithubOrganizations, resp *http.Response, err error)
	ListGithubRepositories(opt *sonar.AlmIntegrationsListGithubRepositoriesOptions) (v *sonar.AlmIntegrationsListGithubRepositories, resp *http.Response, err error)
}

// ALMSettingsGitHubClient is the interface for interacting with SonarQube ALM settings API for GitHub
// It handles all the operations related to ALM settings for GitHub in SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsGitHubClient interface {
	ALMSettingsClient
	CreateGithub(opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error)
	UpdateGithub(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error)
}

// NewALMIntegrationsGitHubClient creates a new ALMIntegrationsGitHubClient with the provided SonarQube client configuration.
func NewALMIntegrationsGitHubClient(clientConfig common.Config) ALMIntegrationsGitHubClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmIntegrations
}

// NewALMSettingsGitHubClient creates a new ALMSettingsGitHubClient with the provided SonarQube client configuration.
func NewALMSettingsGitHubClient(clientConfig common.Config) ALMSettingsGitHubClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// LateInitializeALMGitHub fills the empty fields in the ALMGitHub spec with the values from the SonarQube API response.
func LateInitializeALMGitHub(spec *v1alpha1.ALMGitHubParameters, observation *v1alpha1.ALMGitHubObservation) {
	if spec == nil || observation == nil {
		return
	}
	// No fields to late-initialize for GitHub ALM; all observable fields are required in the spec.
}

// IsALMGitHubLateInitialized checks if two ALMGitHub specs are equal after late initialization.
// Returns true if the specs differ (late init changed something), false if they are equal.
func IsALMGitHubLateInitialized(former, current *v1alpha1.ALMGitHubParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return former.URL != current.URL ||
		former.Key != current.Key ||
		former.AppID != current.AppID ||
		former.ClientID != current.ClientID
}

// IsALMGitHubUpToDate checks if the ALMGitHub spec is up to date with the SonarQube API response.
// It compares observable fields and all three secret pairs for changes.
func IsALMGitHubUpToDate(
	spec *v1alpha1.ALMGitHubParameters,
	observation *v1alpha1.ALMGitHubObservation,
	clientSecret, savedClientSecret,
	privateKey, savedPrivateKey,
	webhookSecret, savedWebhookSecret string,
) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.URL == observation.URL &&
		spec.Key == observation.Key &&
		spec.AppID == observation.AppID &&
		spec.ClientID == observation.ClientID &&
		clientSecret == savedClientSecret &&
		privateKey == savedPrivateKey &&
		webhookSecret == savedWebhookSecret
}

// GenerateALMGitHubCreateOptions generates the options for creating an ALMGitHub resource in SonarQube API.
func GenerateALMGitHubCreateOptions(spec *v1alpha1.ALMGitHubParameters, clientSecret, privateKey, webhookSecret string) *sonar.AlmSettingsCreateGithubOptions {
	return &sonar.AlmSettingsCreateGithubOptions{
		AppID:         spec.AppID,
		ClientID:      spec.ClientID,
		ClientSecret:  clientSecret,
		Key:           spec.Key,
		PrivateKey:    privateKey,
		URL:           spec.URL,
		WebhookSecret: webhookSecret,
	}
}

// GenerateALMGitHubUpdateOptions generates the options for updating an ALMGitHub resource in SonarQube API.
// Sets NewKey only when the key has changed.
func GenerateALMGitHubUpdateOptions(key string, spec *v1alpha1.ALMGitHubParameters, clientSecret, privateKey, webhookSecret string) *sonar.AlmSettingsUpdateGithubOptions {
	opts := &sonar.AlmSettingsUpdateGithubOptions{
		AppID:         spec.AppID,
		ClientID:      spec.ClientID,
		ClientSecret:  clientSecret,
		Key:           key,
		PrivateKey:    privateKey,
		URL:           spec.URL,
		WebhookSecret: webhookSecret,
	}

	if spec.Key != key {
		opts.NewKey = spec.Key
	}

	return opts
}

// FindGitHubALMDefinitionByKey searches for an ALM settings definition in the list of GitHub definitions by its key.
func FindGitHubALMDefinitionByKey(definitions *[]sonar.GithubDefinition, key string) *sonar.GithubDefinition {
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

// GenerateALMGitHubObservation generates the ALMGitHubObservation based on the ALM settings definition retrieved from SonarQube API.
func GenerateALMGitHubObservation(definition *sonar.GithubDefinition) v1alpha1.ALMGitHubObservation {
	if definition == nil {
		return v1alpha1.ALMGitHubObservation{}
	}

	return v1alpha1.ALMGitHubObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{
			URL: definition.URL,
			Key: definition.Key,
		},
		AppID:    definition.AppID,
		ClientID: definition.ClientID,
	}
}
