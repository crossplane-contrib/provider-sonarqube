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

// Package integration provides integration clients for ALM integrations.
package integration

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ALMIntegrationsAzureClient is the interface for interacting with
// SonarQube ALMAzure API
// It handles all the operations related to ALMAzure in SonarQube,
// such as creating, updating, deleting, and retrieving ALMAzure.
type ALMIntegrationsAzureClient interface {
	ALMIntegrationsClient
	ListAzureProjects(opt *sonar.AlmIntegrationsListAzureProjectsOptions) (v *sonar.AlmIntegrationsListAzureProjects, resp *http.Response, err error)
	SearchAzureRepos(opt *sonar.AlmIntegrationsSearchAzureReposOptions) (v *sonar.AlmIntegrationsSearchAzureRepos, resp *http.Response, err error)
}

// ALMSettingsAzureClient is the interface for interacting with
// SonarQube ALM settings API for Azure
// It handles all the operations related to ALM settings for Azure in SonarQube,
// such as creating, updating, deleting, and retrieving them.
type ALMSettingsAzureClient interface {
	ALMSettingsClient
	CreateAzure(opt *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error)
	UpdateAzure(opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error)
}

// NewALMIntegrationsAzureClient creates a new ALMIntegrationsAzureClient with
// the provided SonarQube client configuration.
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
