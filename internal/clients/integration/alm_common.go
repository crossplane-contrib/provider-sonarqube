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

// ALMIntegrationsClient is the interface for interacting with
// SonarQube ALM integrations API
// It handles all the operations related to ALM integrations in SonarQube,
// such as creating, updating, deleting, and retrieving them.
type ALMIntegrationsClient interface {
	CheckPat(ctx context.Context, opt *sonar.AlmIntegrationsCheckPatOptions) (v *sonar.AlmIntegrationsCheckPat, resp *http.Response, err error)
	SetPat(ctx context.Context, opt *sonar.AlmIntegrationsSetPatOptions) (resp *http.Response, err error)
}

// ALMSettingsClient is the interface for interacting with
// SonarQube ALM settings API
// It handles all the operations related to ALM settings in SonarQube,
// such as creating, updating, deleting, and retrieving them.
type ALMSettingsClient interface {
	CountBinding(ctx context.Context, opt *sonar.AlmSettingsCountBindingOptions) (*sonar.AlmSettingsCountBinding, *http.Response, error)
	Delete(ctx context.Context, opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error)
	DeleteBinding(ctx context.Context, opt *sonar.AlmSettingsDeleteBindingOptions) (*http.Response, error)
	GetBinding(ctx context.Context, opt *sonar.AlmSettingsGetBindingOptions) (*sonar.AlmSettingsGetBinding, *http.Response, error)
	List(ctx context.Context, opt *sonar.AlmSettingsListOptions) (*sonar.AlmSettingsList, *http.Response, error)
	ListDefinitions(ctx context.Context) (*sonar.AlmSettingsListDefinitions, *http.Response, error)
	SetAzureBinding(ctx context.Context, opt *sonar.AlmSettingsSetAzureBindingOptions) (*http.Response, error)
	SetBitbucketBinding(ctx context.Context, opt *sonar.AlmSettingsSetBitbucketBindingOptions) (*http.Response, error)
	SetBitbucketCloudBinding(ctx context.Context, opt *sonar.AlmSettingsSetBitbucketCloudBindingOptions) (*http.Response, error)
	SetGithubBinding(ctx context.Context, opt *sonar.AlmSettingsSetGithubBindingOptions) (*http.Response, error)
	SetGitlabBinding(ctx context.Context, opt *sonar.AlmSettingsSetGitlabBindingOptions) (*http.Response, error)
	Validate(ctx context.Context, opt *sonar.AlmSettingsValidateOptions) (*sonar.AlmSettingsValidation, *http.Response, error)
}

// NewALMSettingsClient creates a new ALMSettingsClient with the provided
// SonarQube client configuration.
func NewALMSettingsClient(clientConfig common.Config) ALMSettingsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// IsALMUpToDate checks if the ALM spec is up to date with the
// SonarQube API response.
// It returns true if the spec is up to date, and false if it is not.
func IsALMUpToDate(spec *v1alpha1.ALMCommonParameters, specAPIToken string, observation *v1alpha1.ALMCommonObservation, savedAPIToken string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.URL == observation.URL &&
		spec.Key == observation.Key &&
		specAPIToken == savedAPIToken
}

// LateInitializeALM fills the empty fields in the ALM spec with the
// values from the SonarQube API response.
// The API response should be the result of a "Get" operation for
// the ALM resource.
func LateInitializeALM(spec *v1alpha1.ALMCommonParameters, observation *v1alpha1.ALMCommonObservation) {
	if spec == nil || observation == nil {
		return
	}

	// This currently does not do anything because there are no fields in the ALMCommonParameters that can be late initialized. However, if in the future we add any fields to the ALMCommonParameters that can be late initialized, we should add the logic to fill in those fields here.
}

// IsALMLateInitialized checks if two ALM specs are equal after late
// initialization. It returns true if the specs are equal,
// and false if they are not.
func IsALMLateInitialized(former, current *v1alpha1.ALMCommonParameters) bool {
	if former == nil || current == nil {
		return true
	}

	return former.URL != current.URL ||
		former.Key != current.Key
}

// GenerateALMDeleteOptions generates the options for deleting an ALM resource
// in SonarQube API based on the external name of the resource.
func GenerateALMDeleteOptions(externalName string) *sonar.AlmSettingsDeleteOptions {
	return &sonar.AlmSettingsDeleteOptions{
		Key: externalName,
	}
}
