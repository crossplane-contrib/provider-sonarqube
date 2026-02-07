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
)

const (
	// maxRulesPerPage is the maximum number of rules that can be fetched per page
	maxRulesPerPage = 500
)

// RulesClient is the client for SonarQube Rules API
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

// GenerateQualityProfileRuleSearchOption generates SonarQube RulesSearchOption for a given quality profile key
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
			"cleanCodeAttribute",
			"createdAt",
			"debtRemFn",
			"defaultDebtRemFn",
			"defaultRemFn",
			"deprecatedKeys",
			"descriptionSections",
			"educationPrinciples",
			"gapDescription",
			"htmlDesc",
			"htmlNote",
			"internalKey",
			"isExternal",
			"isTemplate",
			"lang",
			"langName",
			"mdDesc",
			"mdNote",
			"name",
			"noteLogin",
			"params",
			"remFn",
			"remFnOverloaded",
			"repo",
			"scope",
			"severity",
			"status",
			"sysTags",
			"tags",
			"templateKey",
			"updatedAt",
		},
	}
}

// FetchAllQualityProfileRules fetches all activated rules for a quality profile using pagination.
// It iterates through all pages until all rules are fetched.
func FetchAllQualityProfileRules(rulesClient RulesClient, qualityProfileKey string) (*sonar.RulesSearch, error) {
	var allRules []sonar.RuleDetails
	page := 1

	for {
		rules, resp, err := rulesClient.Search(GenerateQualityProfileRulesSearchOption(qualityProfileKey, page)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)
		if err != nil {
			return nil, err
		}

		if rules.Rules != nil {
			allRules = append(allRules, rules.Rules...)
		}

		// Check if we've fetched all rules
		// Paging.Total is the total number of rules, we compare with what we've collected
		if int64(len(allRules)) >= rules.Paging.Total {
			// Return aggregated result with the paging info from last response
			return &sonar.RulesSearch{
				Actives: rules.Actives,
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

// GenerateQualityProfileRulesObservation generates observations for Quality Profile Rules
func GenerateQualityProfileRulesObservation(rules *sonar.RulesSearch) []v1alpha1.QualityProfileRuleObservation {
	if rules == nil || rules.Rules == nil {
		return []v1alpha1.QualityProfileRuleObservation{}
	}

	observations := make([]v1alpha1.QualityProfileRuleObservation, len(rules.Rules))

	for i, rule := range rules.Rules {
		observations[i] = GenerateQualityProfileRuleObservation(rule)
	}

	return observations
}

// GenerateQualityProfileRuleObservation generates observation for a Quality Profile Rule
func GenerateQualityProfileRuleObservation(rule sonar.RuleDetails) v1alpha1.QualityProfileRuleObservation {
	return v1alpha1.QualityProfileRuleObservation{
		CleanCodeAttribute:         rule.CleanCodeAttribute,
		CleanCodeAttributeCategory: rule.CleanCodeAttributeCategory,
		CreatedAt:                  helpers.StringToMetaTime(&rule.CreatedAt),
		DescriptionSections:        GenerateQualityProfileRuleDescriptionSectionsObservation(&rule.DescriptionSections),
		HTMLDesc:                   rule.HTMLDesc,
		HTMLNote:                   rule.HTMLNote,
		Impacts:                    GenerateQualityProfileImpactsObservation(&rule.Impacts),
		InternalKey:                rule.InternalKey,
		IsExternal:                 rule.IsExternal,
		IsTemplate:                 rule.IsTemplate,
		Key:                        rule.Key,
		Language:                   rule.Lang,
		LanguageName:               rule.LangName,
		MdNote:                     rule.MdNote,
		Name:                       rule.Name,
		NoteLogin:                  rule.NoteLogin,
		Parameters:                 GenerateQualityProfileRuleParametersObservation(rule.Params),
		Repo:                       rule.Repo,
		Scope:                      rule.Scope,
		Severity:                   rule.Severity,
		Status:                     rule.Status,
		SysTags:                    rule.SysTags,
		Tags:                       helpers.AnySliceToStringSlice(rule.Tags),
		TemplateKey:                rule.TemplateKey,
		Type:                       rule.Type,
		UpdatedAt:                  helpers.StringToMetaTime(&rule.UpdatedAt),
	}
}

// GenerateQualityProfileRuleDescriptionSectionsObservation generates observations for Quality Profile Rule Descriptions
func GenerateQualityProfileRuleDescriptionSectionsObservation(descriptionSections *[]sonar.DescriptionSection) []v1alpha1.QualityProfileRuleDescription {
	if descriptionSections == nil {
		return []v1alpha1.QualityProfileRuleDescription{}
	}

	observations := make([]v1alpha1.QualityProfileRuleDescription, len(*descriptionSections))

	for i, section := range *descriptionSections {
		observations[i] = GenerateQualityProfileRuleDescriptionObservation(section)
	}

	return observations
}

// GenerateQualityProfileRuleDescriptionObservation generates observation for Quality Profile Rule Description
func GenerateQualityProfileRuleDescriptionObservation(descriptionSections sonar.DescriptionSection) v1alpha1.QualityProfileRuleDescription {
	return v1alpha1.QualityProfileRuleDescription{
		Content: descriptionSections.Content,
		Context: GenerateQualityProfileRuleDescriptionSectionsContextObservation(descriptionSections.Context),
		Key:     descriptionSections.Key,
	}
}

// GenerateQualityProfileRuleDescriptionSectionsContextObservation generates observation for Quality Profile Rule Description Context
func GenerateQualityProfileRuleDescriptionSectionsContextObservation(contextSection sonar.DescriptionContext) v1alpha1.QualityProfileRuleDescriptionSectionsContext {
	return v1alpha1.QualityProfileRuleDescriptionSectionsContext{
		DisplayName: contextSection.DisplayName,
		Key:         contextSection.Key,
	}
}

// GenerateQualityProfileImpactsObservation generates observations for Quality Profile Rule Impacts
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

// GenerateQualityProfileRuleImpactObservation generates observation for Quality Profile Rule Impact
func GenerateQualityProfileRuleImpactObservation(impact sonar.RuleImpact) v1alpha1.QualityProfileRuleImpact {
	return v1alpha1.QualityProfileRuleImpact{
		Severity:        impact.Severity,
		SoftwareQuality: impact.SoftwareQuality,
	}
}

// GenerateQualityProfileRuleParametersObservation generates observations for Quality Profile Rule Parameters
func GenerateQualityProfileRuleParametersObservation(parameters []sonar.RuleParam) []v1alpha1.QualityProfileRuleParameter {
	observations := make([]v1alpha1.QualityProfileRuleParameter, len(parameters))
	for i, parameter := range parameters {
		observations[i] = GenerateQualityProfileRuleParameterObservation(parameter)
	}
	return observations
}

// GenerateQualityProfileRuleParameterObservation generates observation for Quality Profile Rule Parameter
func GenerateQualityProfileRuleParameterObservation(parameter sonar.RuleParam) v1alpha1.QualityProfileRuleParameter {
	return v1alpha1.QualityProfileRuleParameter{
		DefaultValue: parameter.DefaultValue,
		Desc:         parameter.Desc,
		Key:          parameter.Key,
	}
}

// IsQualityProfileRuleUpToDate checks whether the observed QualityProfileRule is up to date with the desired QualityProfileRuleParameters
// Note: We only compare the rule key since the SonarQube API does not return the activated severity, impacts, or parameter values.
// The API returns rule definitions with default values, not the customized activation configuration.
func IsQualityProfileRuleUpToDate(spec *v1alpha1.QualityProfileRuleParameters, observation *v1alpha1.QualityProfileRuleObservation) bool {
	if spec == nil {
		return true
	}
	if observation == nil {
		return false
	}

	// Only compare rule keys - we cannot reliably compare severity, impacts, or parameters
	// because the API returns defaults, not activated values
	if spec.Rule != observation.Key {
		return false
	}

	// if !helpers.IsComparablePtrEqualComparable(spec.Severity, observation.Severity) {
	// 	return false
	// }

	// if !areQualityProfileRuleImpactsUpToDate(spec.Impacts, observation.Impacts) {
	// 	return false
	// }

	// if !areQualityProfileRuleParametersUpToDate(spec.Parameters, observation.Parameters) {
	// 	return false
	// }

	return true
}

// // areQualityProfileRuleImpactsUpToDate checks whether the observed QualityProfileRule impacts are up to date with the desired impacts
// // Not functional yet because SonarQube API does not return proper values for impacts (SonarQube 25.12.X)
// func areQualityProfileRuleImpactsUpToDate(spec *map[string]string, observation []v1alpha1.QualityProfileRuleImpact) bool {
// 	if spec == nil {
// 		return true
// 	}

// 	// Build an impact map from observation for easy lookup
// 	impactMap := make(map[string]string, len(observation))
// 	for _, impact := range observation {
// 		impactMap[impact.SoftwareQuality] = impact.Severity
// 	}

// 	for k, v := range *spec {
// 		if observedSeverity, ok := impactMap[k]; !ok || observedSeverity != v {
// 			return false
// 		}
// 	}
// 	return true
// }

// // areQualityProfileRuleParametersUpToDate checks whether the observed QualityProfileRule parameters are up to date with the desired parameters
// func areQualityProfileRuleParametersUpToDate(spec *map[string]string, observation []v1alpha1.QualityProfileRuleParameter) bool {
// 	if spec == nil {
// 		return true
// 	}

// 	// Build a parameter map from observation for easy lookup
// 	parameterMap := make(map[string]string, len(observation))
// 	for _, parameter := range observation {
// 		parameterMap[parameter.Key] = parameter.DefaultValue
// 	}

// 	for k, v := range *spec {
// 		if observedValue, ok := parameterMap[k]; !ok || observedValue != v {
// 			return false
// 		}
// 	}
// 	return true
// }
