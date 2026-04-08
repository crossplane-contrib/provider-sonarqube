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

//nolint:dupl // Provider-specific ALM wrappers intentionally share this structure.
package integration

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

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
