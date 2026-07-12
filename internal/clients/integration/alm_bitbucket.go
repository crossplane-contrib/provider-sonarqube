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
