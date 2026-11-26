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
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// FixSuggestionsClient is the interface for interacting with the SonarQube
// V2 AI CodeFix enablement API (api/v2/fix-suggestions/feature-enablements).
type FixSuggestionsClient interface {
	GetEnablement(ctx context.Context) (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error)
	SetEnablement(ctx context.Context, opt *sonar.FixSuggestionsSetEnablementOptions) (*http.Response, error)
}

// NewFixSuggestionsClient creates a new FixSuggestionsClient using the
// provided SonarQube client configuration.
func NewFixSuggestionsClient(clientConfig common.Config) FixSuggestionsClient {
	return common.NewClient(clientConfig).V2.FixSuggestions
}

// GenerateAICodeFixEnablementOptions generates the options setting the
// enablement of the spec. Projects are only sent when AI CodeFix is enabled
// for some projects.
func GenerateAICodeFixEnablementOptions(spec *v1alpha1.AICodeFixConfigurationParameters) *sonar.FixSuggestionsSetEnablementOptions {
	opts := &sonar.FixSuggestionsSetEnablementOptions{Enablement: spec.Enablement}

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
