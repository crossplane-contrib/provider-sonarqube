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
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	// maxRulesPerPage is the maximum number of rules that can be fetched per page.
	maxRulesPerPage = 500
)

// RulesClient is the client for SonarQube Rules API.
type RulesClient interface {
	App() (v *sonar.RulesApp, resp *http.Response, err error)
	Create(opt *sonar.RulesCreateOption) (v *sonar.RulesCreate, resp *http.Response, err error)
	Delete(opt *sonar.RulesDeleteOption) (resp *http.Response, err error)
	List(opt *sonar.RulesListOption) (v *string, resp *http.Response, err error)
	Repositories(opt *sonar.RulesRepositoriesOption) (v *sonar.RulesRepositories, resp *http.Response, err error)
	Search(opt *sonar.RulesSearchOption) (v *sonar.RulesSearch, resp *http.Response, err error)
	Show(opt *sonar.RulesShowOption) (v *sonar.RulesShow, resp *http.Response, err error)
	Tags(opt *sonar.RulesTagsOption) (v *sonar.RulesTags, resp *http.Response, err error)
	Update(opt *sonar.RulesUpdateOption) (v *sonar.RulesUpdate, resp *http.Response, err error)
}

// NewRulesClient creates a new RulesClient with the provided SonarQube client configuration.
func NewRulesClient(clientConfig common.Config) RulesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Rules
}

// GenerateQualityProfileRulesSearchOption generates SonarQube RulesSearchOption for a given quality profile key
// to fetch activated rules in that quality profile.
func GenerateQualityProfileRulesSearchOption(key string, page int) *sonar.RulesSearchOption {
	return &sonar.RulesSearchOption{
		Qprofile: key,
		// We want only activated rules in the quality profile
		Activation: true,
		PaginationArgs: sonar.PaginationArgs{
			// Set page size to maximum allowed
			PageSize: maxRulesPerPage,
			// Set page number (1-based)
			Page: int64(page),
		},
		// Retrieve all fields, including "actives"
		Fields: []string{
			"actives",
			"createdAt",
			"internalKey",
			"name",
			"params",
			"severity",
			"status",
			"updatedAt",
		},
	}
}

// FetchAllQualityProfileRules fetches all activated rules for a quality profile using pagination.
// It iterates through all pages until all rules are fetched.
func FetchAllQualityProfileRules(rulesClient RulesClient, qualityProfileKey string) (*sonar.RulesSearch, error) {
	var allRules []sonar.RuleDetails

	allActives := make(map[string][]sonar.RuleActivation)

	page := 1

	for {
		rules, resp, err := rulesClient.Search(GenerateQualityProfileRulesSearchOption(qualityProfileKey, page)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, err
		}

		if rules.Rules != nil {
			allRules = append(allRules, rules.Rules...)
			for key, activations := range rules.Actives {
				allActives[key] = append(allActives[key], activations...)
			}
		}

		// Check if we've fetched all rules
		// Paging.Total is the total number of rules, we compare with what we've collected
		if int64(len(allRules)) >= rules.Paging.Total {
			// Return aggregated result with the paging info from last response
			return &sonar.RulesSearch{
				Actives: allActives,
				Facets:  rules.Facets,
				Paging: sonar.Paging{
					PageIndex: int64(page),
					PageSize:  int64(len(allRules)),
					Total:     rules.Paging.Total,
				},
				Rules: allRules,
			}, nil
		}

		page++
	}
}

// GenerateQualityProfileRulesObservation generates observations for Quality Profile Rules.
func GenerateQualityProfileRulesObservation(qualityProfileId string, rules *sonar.RulesSearch) []v1alpha1.QualityProfileRuleObservation {
	if rules == nil || rules.Rules == nil {
		return []v1alpha1.QualityProfileRuleObservation{}
	}

	observations := make([]v1alpha1.QualityProfileRuleObservation, len(rules.Rules))

	for index, rule := range rules.Rules {
		var activatedRuleList *[]sonar.RuleActivation

		activatedRuleFetch, exists := rules.Actives[rule.Key]
		if exists {
			activatedRuleList = &activatedRuleFetch
		}

		ruleActivatedSettings := findQualityProfileActiveRuleSettings(qualityProfileId, activatedRuleList)
		observations[index] = GenerateQualityProfileRuleObservation(rule, ruleActivatedSettings)
	}

	return observations
}

// ruleActiveSettings holds the activated settings for a rule, including severity and parameters.
type ruleActiveSettings struct {
	Severity    *string
	Params      *map[string]string
	Impacts     []v1alpha1.QualityProfileRuleImpact
	Prioritized *bool
}

// findQualityProfileActiveRuleSettings parses the activated rules, confirms that they belong to the quality profile, and returns a map of rule key to its activated settings (severity and parameters).
func findQualityProfileActiveRuleSettings(qualityProfileId string, activeRules *[]sonar.RuleActivation) *ruleActiveSettings {
	if activeRules == nil {
		return nil
	}

	for _, activeRule := range *activeRules {
		if activeRule.QProfile == qualityProfileId {
			params := make(map[string]string, len(activeRule.Params))
			for _, param := range activeRule.Params {
				params[param.Key] = param.Value
			}

			return &ruleActiveSettings{
				Severity:    &activeRule.Severity,
				Params:      &params,
				Prioritized: &activeRule.PrioritizedRule,
				Impacts:     GenerateQualityProfileImpactsObservation(&activeRule.Impacts),
			}
		}
	}

	return nil
}

// GenerateQualityProfileRuleObservation generates observation for a Quality Profile Rule.
func GenerateQualityProfileRuleObservation(rule sonar.RuleDetails, activatedSettings *ruleActiveSettings) v1alpha1.QualityProfileRuleObservation {
	ruleObservation := v1alpha1.QualityProfileRuleObservation{
		Key:       rule.Key,
		CreatedAt: helpers.StringToMetaTime(&rule.CreatedAt),
		Impacts:   GenerateQualityProfileImpactsObservation(&rule.Impacts),
		Name:      rule.Name,
		UpdatedAt: helpers.StringToMetaTime(&rule.UpdatedAt),
		Severity:  rule.Severity,
	}

	if activatedSettings != nil {
		if activatedSettings.Severity != nil {
			ruleObservation.Severity = *activatedSettings.Severity
		}

		if activatedSettings.Params != nil {
			ruleObservation.Parameters = *activatedSettings.Params
		}

		if activatedSettings.Impacts != nil {
			ruleObservation.Impacts = activatedSettings.Impacts
		}

		if activatedSettings.Prioritized != nil {
			ruleObservation.Prioritized = *activatedSettings.Prioritized
		}
	}

	return ruleObservation
}

// GenerateQualityProfileImpactsObservation generates observations for Quality Profile Rule Impacts.
func GenerateQualityProfileImpactsObservation(impacts *[]sonar.RuleImpact) []v1alpha1.QualityProfileRuleImpact {
	if impacts == nil {
		return []v1alpha1.QualityProfileRuleImpact{}
	}

	observations := make([]v1alpha1.QualityProfileRuleImpact, len(*impacts))
	for i, impact := range *impacts {
		observations[i] = GenerateQualityProfileRuleImpactObservation(impact)
	}

	return observations
}

// GenerateQualityProfileRuleImpactObservation generates observation for Quality Profile Rule Impact.
func GenerateQualityProfileRuleImpactObservation(impact sonar.RuleImpact) v1alpha1.QualityProfileRuleImpact {
	return v1alpha1.QualityProfileRuleImpact{
		Severity:        impact.Severity,
		SoftwareQuality: impact.SoftwareQuality,
	}
}

// IsQualityProfileRuleUpToDate checks whether the observed QualityProfileRule is up to date with the desired QualityProfileRuleParameters
// Compares rule key, severity (if specified), and parameters (if specified).
func IsQualityProfileRuleUpToDate(spec *v1alpha1.QualityProfileRuleParameters, observation *v1alpha1.QualityProfileRuleObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	// Rule key must match
	if spec.Rule != observation.Key {
		return false
	}

	// Check severity if specified
	if !helpers.IsComparablePtrEqualComparable(spec.Severity, observation.Severity) {
		return false
	}

	// Check prioritized if specified
	if !helpers.IsComparablePtrEqualComparable(spec.Prioritized, observation.Prioritized) {
		return false
	}

	// Check impacts if specified
	if !areQualityProfileRuleImpactsUpToDate(spec.Impacts, observation.Impacts) {
		return false
	}

	// Check parameters if specified
	if spec.Parameters != nil {
		if !cmp.Equal(*spec.Parameters, observation.Parameters, cmpopts.EquateEmpty()) {
			return false
		}
	}

	return true
}

// areQualityProfileRuleImpactsUpToDate checks whether the observed QualityProfileRule impacts are up to date with the desired impacts.
func areQualityProfileRuleImpactsUpToDate(spec *map[string]string, observation []v1alpha1.QualityProfileRuleImpact) bool {
	if spec == nil {
		return true
	}

	// Build an impact map from observation for easy lookup
	impactMap := make(map[string]string, len(observation))
	for _, impact := range observation {
		impactMap[impact.SoftwareQuality] = impact.Severity
	}

	return cmp.Equal(*spec, impactMap, cmpopts.EquateEmpty())
}
