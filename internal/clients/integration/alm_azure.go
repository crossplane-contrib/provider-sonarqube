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

// Package integration provides clients for managing SonarQube ALM
// (Application Lifecycle Management) and webhook integration resources.
package integration

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ALMIntegrationsAzureClient is the interface for interacting with
// SonarQube ALMAzure API
// It handles all the operations related to ALMAzure in SonarQube,
// such as creating, updating, deleting, and retrieving ALMAzure.
type ALMIntegrationsAzureClient interface {
	ALMIntegrationsClient
	ListAzureProjects(ctx context.Context, opt *sonar.AlmIntegrationsListAzureProjectsOptions) (v *sonar.AlmIntegrationsListAzureProjects, resp *http.Response, err error)
	SearchAzureRepos(ctx context.Context, opt *sonar.AlmIntegrationsSearchAzureReposOptions) (v *sonar.AlmIntegrationsSearchAzureRepos, resp *http.Response, err error)
}

// ALMSettingsAzureClient is the interface for interacting with
// SonarQube ALM settings API for Azure
// It handles all the operations related to ALM settings for Azure in
// SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsAzureClient interface {
	ALMSettingsClient
	CreateAzure(ctx context.Context, opt *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error)
	UpdateAzure(ctx context.Context, opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error)
}

// NewALMIntegrationsAzureClient creates a new ALMIntegrationsAzureClient
// with the provided SonarQube client configuration.
func NewALMIntegrationsAzureClient(clientConfig common.Config) ALMIntegrationsAzureClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmIntegrations
}

// NewALMSettingsAzureClient creates a new ALMSettingsAzureClient with the
// provided SonarQube client configuration.
func NewALMSettingsAzureClient(clientConfig common.Config) ALMSettingsAzureClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// LateInitializeALMAzure fills the empty fields in the ALMAzure spec with
// the values from the SonarQube API response.
// The API response should be the result of a "Get" operation for the
// ALMAzure resource.
func LateInitializeALMAzure(spec *v1alpha1.ALMAzureParameters, observation *v1alpha1.ALMAzureObservation) {
	if spec == nil || observation == nil {
		return
	}

	LateInitializeALM(&spec.ALMCommonParameters, &observation.ALMCommonObservation)
}

// IsALMAzureLateInitialized checks if two ALMAzure specs are equal after late
// initialization. It returns true if the specs are equal,
// and false if they are not.
func IsALMAzureLateInitialized(former, current *v1alpha1.ALMAzureParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return IsALMLateInitialized(&former.ALMCommonParameters, &current.ALMCommonParameters)
}

// IsALMAzureUpToDate checks if the ALMAzure spec is up to date with
// the SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsALMAzureUpToDate(spec *v1alpha1.ALMAzureParameters, specAPIToken string, observation *v1alpha1.ALMAzureObservation, savedAPIToken string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return IsALMUpToDate(&spec.ALMCommonParameters, specAPIToken, &observation.ALMCommonObservation, savedAPIToken)
}

// GenerateALMAzureCreateOptions generates the options for creating an
// ALMAzure resource in SonarQube API based on the desired state in the
// ALMAzureParameters and the provided API token.
func GenerateALMAzureCreateOptions(spec *v1alpha1.ALMAzureParameters, apiToken string) *sonar.AlmSettingsCreateAzureOptions {
	return &sonar.AlmSettingsCreateAzureOptions{
		URL:                 spec.URL,
		Key:                 spec.Key,
		PersonalAccessToken: apiToken,
	}
}

// GenerateALMAzureUpdateOptions generates the options for updating an
// ALMAzure resource in SonarQube API based on the desired state in the
// ALMAzureParameters, the provided API token, and the identifier of the
// ALMAzure resource in SonarQube.
func GenerateALMAzureUpdateOptions(key string, spec *v1alpha1.ALMAzureParameters, apiToken string) *sonar.AlmSettingsUpdateAzureOptions {
	updateOptions := sonar.AlmSettingsUpdateAzureOptions{
		URL:                 spec.URL,
		Key:                 key,
		PersonalAccessToken: apiToken,
	}

	if spec.Key != key {
		updateOptions.NewKey = spec.Key
	}

	return &updateOptions
}

// FindAzureALMDefinitionByKey searches for an ALM settings definition
// in the list of definitions by its key and devops platform.
// It returns the definition if found, and nil if not found.
func FindAzureALMDefinitionByKey(definitions *[]sonar.AzureDefinition, key string) *sonar.AzureDefinition {
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

// GenerateALMAzureObservation generates the ALMAzureObservation
// based on the ALM settings definition retrieved from SonarQube API.
func GenerateALMAzureObservation(definition *sonar.AzureDefinition) v1alpha1.ALMAzureObservation {
	if definition == nil {
		return v1alpha1.ALMAzureObservation{}
	}

	return v1alpha1.ALMAzureObservation{
		ALMCommonObservation: v1alpha1.ALMCommonObservation{
			URL: definition.URL,
			Key: definition.Key,
		},
	}
}
