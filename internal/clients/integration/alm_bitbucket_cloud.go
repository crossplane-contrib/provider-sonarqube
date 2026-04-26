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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ALMIntegrationsBitbucketCloudClient is the interface for interacting with
// SonarQube ALMBitbucketCloud API
// It handles all the operations related to ALMBitbucketCloud in SonarQube,
// such as creating, updating, deleting, and retrieving ALMBitbucketCloud.
type ALMIntegrationsBitbucketCloudClient interface {
	ALMIntegrationsClient
	SearchBitbucketCloudRepos(opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error)
}

// ALMSettingsBitbucketCloudClient is the interface for interacting with
// SonarQube ALM settings API for BitbucketCloud
// It handles all the operations related to ALM settings for BitbucketCloud
// in SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsBitbucketCloudClient interface {
	ALMSettingsClient
	CreateBitbucketCloud(opt *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error)
	UpdateBitbucketCloud(opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error)
}

// NewALMIntegrationsBitbucketCloudClient creates a new
// ALMIntegrationsBitbucketCloudClient with the provided
// SonarQube client configuration.
func NewALMIntegrationsBitbucketCloudClient(clientConfig common.Config) ALMIntegrationsBitbucketCloudClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmIntegrations
}

// NewALMSettingsBitbucketCloudClient creates a new
// ALMSettingsBitbucketCloudClient with the provided
// SonarQube client configuration.
func NewALMSettingsBitbucketCloudClient(clientConfig common.Config) ALMSettingsBitbucketCloudClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}
