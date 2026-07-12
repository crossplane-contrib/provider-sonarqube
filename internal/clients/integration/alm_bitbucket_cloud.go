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

// ALMIntegrationsBitbucketCloudClient is the interface for interacting with
// SonarQube ALMBitbucketCloud API
// It handles all the operations related to ALMBitbucketCloud in SonarQube,
// such as creating, updating, deleting, and retrieving ALMBitbucketCloud.
type ALMIntegrationsBitbucketCloudClient interface {
	ALMIntegrationsClient
	SearchBitbucketCloudRepos(ctx context.Context, opt *sonar.AlmIntegrationsSearchBitbucketCloudReposOptions) (v *sonar.AlmIntegrationsSearchBitbucketCloudRepos, resp *http.Response, err error)
}

// ALMSettingsBitbucketCloudClient is the interface for interacting with
// SonarQube ALM settings API for BitbucketCloud
// It handles all the operations related to ALM settings for BitbucketCloud
// in SonarQube, such as creating, updating, deleting, and retrieving them.
type ALMSettingsBitbucketCloudClient interface {
	ALMSettingsClient
	CreateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error)
	UpdateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error)
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

// LateInitializeALMBitbucketCloud fills the empty fields in the
// ALMBitbucketCloud spec with the values from the SonarQube API response.
func LateInitializeALMBitbucketCloud(spec *v1alpha1.ALMBitbucketCloudParameters, observation *v1alpha1.ALMBitbucketCloudObservation) {
	if spec == nil || observation == nil {
		return
	}
	// No fields to late-initialize for Bitbucket Cloud ALM; all observable fields are required in the spec.
}

// IsALMBitbucketCloudLateInitialized checks if two ALMBitbucketCloud specs
// are equal after late initialization.
// Returns true if the specs differ (late init changed something),
// false if they are equal.
func IsALMBitbucketCloudLateInitialized(former, current *v1alpha1.ALMBitbucketCloudParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return former.Key != current.Key ||
		former.ClientID != current.ClientID ||
		former.Workspace != current.Workspace
}

// IsALMBitbucketCloudUpToDate checks if the ALMBitbucketCloud spec is up
// to date with the SonarQube API response.
func IsALMBitbucketCloudUpToDate(
	spec *v1alpha1.ALMBitbucketCloudParameters,
	observation *v1alpha1.ALMBitbucketCloudObservation,
	clientSecret, savedClientSecret string,
) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.Key == observation.Key &&
		spec.ClientID == observation.ClientID &&
		spec.Workspace == observation.Workspace &&
		clientSecret == savedClientSecret
}

// GenerateALMBitbucketCloudCreateOptions generates the options for creating
// an ALMBitbucketCloud resource in SonarQube API.
func GenerateALMBitbucketCloudCreateOptions(spec *v1alpha1.ALMBitbucketCloudParameters, clientSecret string) *sonar.AlmSettingsCreateBitbucketCloudOptions {
	return &sonar.AlmSettingsCreateBitbucketCloudOptions{
		ClientID:     spec.ClientID,
		ClientSecret: clientSecret,
		Key:          spec.Key,
		Workspace:    spec.Workspace,
	}
}

// GenerateALMBitbucketCloudUpdateOptions generates the options for updating
// an ALMBitbucketCloud resource in SonarQube API.
// Sets NewKey only when the key has changed.
func GenerateALMBitbucketCloudUpdateOptions(key string, spec *v1alpha1.ALMBitbucketCloudParameters, clientSecret string) *sonar.AlmSettingsUpdateBitbucketCloudOptions {
	opts := &sonar.AlmSettingsUpdateBitbucketCloudOptions{
		ClientID:     spec.ClientID,
		ClientSecret: clientSecret,
		Key:          key,
		Workspace:    spec.Workspace,
	}

	if spec.Key != key {
		opts.NewKey = spec.Key
	}

	return opts
}

// FindBitbucketCloudALMDefinitionByKey searches for an ALM settings
// definition in the list of Bitbucket Cloud definitions by its key.
func FindBitbucketCloudALMDefinitionByKey(definitions *[]sonar.BitbucketCloudDefinition, key string) *sonar.BitbucketCloudDefinition {
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

// GenerateALMBitbucketCloudObservation generates the
// ALMBitbucketCloudObservation based on the ALM settings definition
// retrieved from SonarQube API.
func GenerateALMBitbucketCloudObservation(definition *sonar.BitbucketCloudDefinition) v1alpha1.ALMBitbucketCloudObservation {
	if definition == nil {
		return v1alpha1.ALMBitbucketCloudObservation{}
	}

	return v1alpha1.ALMBitbucketCloudObservation{
		Key:       definition.Key,
		ClientID:  definition.ClientID,
		Workspace: definition.Workspace,
	}
}
