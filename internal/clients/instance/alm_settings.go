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

package instance

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// AlmClient is the interface for interacting with SonarQube ALM Settings API.
// It handles operations related to ALM integrations in SonarQube (GitHub, GitLab, Bitbucket, etc.),
// such as creating, updating, deleting, and listing ALM settings.
type AlmClient interface {
	// Shared methods for all ALM types
	Delete(opt *sonar.AlmSettingsDeleteOption) (*http.Response, error)
	ListDefinitions() (*sonar.AlmSettingsListDefinitions, *http.Response, error)

	// GitHub-specific methods
	CreateGithub(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error)
	UpdateGithub(opt *sonar.AlmSettingsUpdateGithubOption) (*http.Response, error)
}

// NewAlmClient creates a new AlmClient with the provided SonarQube client configuration.
func NewAlmClient(clientConfig common.Config) AlmClient {
	newClient := common.NewClient(clientConfig)

	return newClient.AlmSettings
}

// AlmGithubSecrets holds the resolved secret values for ALM GitHub operations.
type AlmGithubSecrets struct {
	ClientSecret  string
	PrivateKey    string
	WebhookSecret *string
}

// GenerateAlmGithubCreateOptions generates SonarQube AlmSettingsCreateGithubOption from AlmGithubParameters and resolved secrets.
func GenerateAlmGithubCreateOptions(spec v1alpha1.AlmGithubParameters, secrets AlmGithubSecrets) *sonar.AlmSettingsCreateGithubOption {
	opt := &sonar.AlmSettingsCreateGithubOption{
		AppID:        spec.AppID,
		ClientID:     spec.ClientID,
		ClientSecret: secrets.ClientSecret,
		Key:          spec.Key,
		PrivateKey:   secrets.PrivateKey,
		URL:          spec.URL,
	}

	if secrets.WebhookSecret != nil {
		opt.WebhookSecret = *secrets.WebhookSecret
	}

	return opt
}

// GenerateAlmGithubUpdateOptions generates SonarQube AlmSettingsUpdateGithubOption from AlmGithubParameters and resolved secrets.
func GenerateAlmGithubUpdateOptions(spec v1alpha1.AlmGithubParameters, secrets AlmGithubSecrets) *sonar.AlmSettingsUpdateGithubOption {
	opt := &sonar.AlmSettingsUpdateGithubOption{
		AppID:        spec.AppID,
		ClientID:     spec.ClientID,
		ClientSecret: secrets.ClientSecret,
		Key:          spec.Key,
		PrivateKey:   secrets.PrivateKey,
		URL:          spec.URL,
	}

	if secrets.WebhookSecret != nil {
		opt.WebhookSecret = *secrets.WebhookSecret
	}

	return opt
}

// GenerateAlmDeleteOptions generates SonarQube AlmSettingsDeleteOption from a key.
func GenerateAlmDeleteOptions(key string) *sonar.AlmSettingsDeleteOption {
	return &sonar.AlmSettingsDeleteOption{
		Key: key,
	}
}

// FindGithubDefinition finds a GitHub ALM definition by key from the list of definitions.
// Returns nil if no matching definition is found.
func FindGithubDefinition(definitions *sonar.AlmSettingsListDefinitions, key string) *sonar.GithubDefinition {
	if definitions == nil {
		return nil
	}

	for i := range definitions.Github {
		if definitions.Github[i].Key == key {
			return &definitions.Github[i]
		}
	}

	return nil
}

// GenerateAlmGithubObservation generates AlmGithubObservation from a SonarQube GithubDefinition.
func GenerateAlmGithubObservation(definition *sonar.GithubDefinition) v1alpha1.AlmGithubObservation {
	return v1alpha1.AlmGithubObservation{
		Key:      definition.Key,
		AppID:    definition.AppID,
		ClientID: definition.ClientID,
		URL:      definition.URL,
	}
}

// IsAlmGithubUpToDate checks if the observed ALM GitHub definition is up to date with the desired parameters.
// Note: Secret fields (ClientSecret, PrivateKey, WebhookSecret) cannot be compared
// as they are not returned by the SonarQube API.
func IsAlmGithubUpToDate(spec v1alpha1.AlmGithubParameters, observation v1alpha1.AlmGithubObservation) bool {
	return spec.Key == observation.Key &&
		spec.AppID == observation.AppID &&
		spec.ClientID == observation.ClientID &&
		spec.URL == observation.URL
}
