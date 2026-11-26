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
	"fmt"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// fixSuggestionsEnablementPath is the V2 path of the AI CodeFix feature
// enablement.
const fixSuggestionsEnablementPath = "fix-suggestions/feature-enablements"

// AICodeFixEnablementOptions is the request body of
// PATCH api/v2/fix-suggestions/feature-enablements.
//
// The SDK's sonar.FixSuggestionsSetEnablementOptions only has the
// enablement, while the endpoint also takes the projects to enable AI
// CodeFix for, so the provider sends this body itself.
type AICodeFixEnablementOptions struct {
	// Enablement is the AI CodeFix enablement scope. This field is required.
	// Allowed values: DISABLED, ENABLED_FOR_ALL_PROJECTS,
	// ENABLED_FOR_SOME_PROJECTS.
	Enablement string `json:"enablement"`
	// EnabledProjectKeys are the keys of the projects AI CodeFix is enabled
	// for. Only used when Enablement is ENABLED_FOR_SOME_PROJECTS.
	EnabledProjectKeys []string `json:"enabledProjectKeys,omitempty"`
}

// FixSuggestionsClient is the interface for interacting with the SonarQube
// V2 AI CodeFix enablement API (api/v2/fix-suggestions/feature-enablements).
type FixSuggestionsClient interface {
	GetEnablement(ctx context.Context) (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error)
	SetEnablement(ctx context.Context, opt *AICodeFixEnablementOptions) (*http.Response, error)
}

// NewFixSuggestionsClient creates a new FixSuggestionsClient using the
// provided SonarQube client configuration.
func NewFixSuggestionsClient(clientConfig common.Config) FixSuggestionsClient {
	return &fixSuggestionsClient{client: common.NewClient(clientConfig)}
}

// fixSuggestionsClient implements FixSuggestionsClient on top of the SDK
// client: reads go through the SDK service, and writes build the V2 request
// directly, see AICodeFixEnablementOptions.
type fixSuggestionsClient struct {
	client *sonar.Client
}

// GetEnablement returns the AI CodeFix feature enablement.
func (c *fixSuggestionsClient) GetEnablement(ctx context.Context) (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error) {
	return c.client.V2.FixSuggestions.GetEnablement(ctx)
}

// SetEnablement sets the AI CodeFix feature enablement, with the projects
// to enable it for.
func (c *fixSuggestionsClient) SetEnablement(ctx context.Context, opt *AICodeFixEnablementOptions) (*http.Response, error) {
	if opt == nil {
		return nil, sonar.NewValidationError("opt", "option struct is required", sonar.ErrMissingRequired)
	}

	err := c.client.V2.FixSuggestions.ValidateSetEnablementOpt(&sonar.FixSuggestionsSetEnablementOptions{Enablement: opt.Enablement})
	if err != nil {
		return nil, err
	}

	req, err := c.client.NewSonarQubeV2APIRequest(ctx, http.MethodPatch, fixSuggestionsEnablementPath, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.client.Do(req, nil)
}

// GenerateAICodeFixEnablementOptions generates the options setting the
// enablement of the spec. Projects are only sent when AI CodeFix is enabled
// for some projects.
func GenerateAICodeFixEnablementOptions(spec *v1alpha1.AICodeFixConfigurationParameters) *AICodeFixEnablementOptions {
	opts := &AICodeFixEnablementOptions{Enablement: spec.Enablement}

	if spec.Enablement == v1alpha1.AICodeFixEnabledForSomeProjects {
		opts.EnabledProjectKeys = spec.EnabledProjectKeys
		if opts.EnabledProjectKeys == nil {
			opts.EnabledProjectKeys = []string{}
		}
	}

	return opts
}

// GenerateAICodeFixConfigurationObservation generates the observation of the
// AI CodeFix feature enablement. Projects are only reported when AI CodeFix
// is enabled for some projects, since SonarQube ignores them otherwise.
func GenerateAICodeFixConfigurationObservation(enablement *sonar.FixSuggestionsFeatureEnablement) v1alpha1.AICodeFixConfigurationObservation {
	if enablement == nil {
		return v1alpha1.AICodeFixConfigurationObservation{}
	}

	observation := v1alpha1.AICodeFixConfigurationObservation{Enablement: enablement.Enablement}

	if enablement.Enablement == v1alpha1.AICodeFixEnabledForSomeProjects {
		observation.EnabledProjectKeys = enablement.EnabledProjectKeys
	}

	return observation
}

// IsAICodeFixConfigurationUpToDate reports whether the observed enablement
// matches the spec. Projects are compared as sets, and only when AI CodeFix
// is enabled for some projects.
func IsAICodeFixConfigurationUpToDate(spec *v1alpha1.AICodeFixConfigurationParameters, observation *v1alpha1.AICodeFixConfigurationObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil || spec.Enablement != observation.Enablement {
		return false
	}

	if spec.Enablement != v1alpha1.AICodeFixEnabledForSomeProjects {
		return true
	}

	return helpers.AreStringSlicesEqualDeDuped(spec.EnabledProjectKeys, observation.EnabledProjectKeys)
}
