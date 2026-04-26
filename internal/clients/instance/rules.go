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
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// maxRulesPerPage is the maximum number of rules that can be fetched per page.
	maxRulesPerPage = 500
)

// RulesClient is the client for SonarQube Rules API.
type RulesClient interface {
	App() (v *sonar.RulesApp, resp *http.Response, err error)
	Create(opt *sonar.RulesCreateOptions) (v *sonar.RulesCreate, resp *http.Response, err error)
	Delete(opt *sonar.RulesDeleteOptions) (resp *http.Response, err error)
	List(opt *sonar.RulesListOptions) (v *string, resp *http.Response, err error)
	Repositories(opt *sonar.RulesRepositoriesOptions) (v *sonar.RulesRepositories, resp *http.Response, err error)
	Search(opt *sonar.RulesSearchOptions) (v *sonar.RulesSearch, resp *http.Response, err error)
	Show(opt *sonar.RulesShowOptions) (v *sonar.RulesShow, resp *http.Response, err error)
	Tags(opt *sonar.RulesTagsOptions) (v *sonar.RulesTags, resp *http.Response, err error)
	Update(opt *sonar.RulesUpdateOptions) (v *sonar.RulesUpdate, resp *http.Response, err error)
}

// NewRulesClient creates a new RulesClient with the provided SonarQube client
// configuration.
func NewRulesClient(clientConfig common.Config) RulesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Rules
}

// GenerateQualityProfileRulesSearchOption generates SonarQube RulesSearchOption
// for a given quality profile key
// to fetch activated rules in that quality profile.
func GenerateQualityProfileRulesSearchOption(key string, page int) *sonar.RulesSearchOptions {
	return &sonar.RulesSearchOptions{
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

// FetchAllQualityProfileRules fetches all activated rules for a quality profile
// using pagination.
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

// GenerateQualityProfileRulesObservation generates observations for
// Quality Profile Rules.
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

// ruleActiveSettings holds the activated settings for a rule,
// including severity and parameters.
type ruleActiveSettings struct {
	Severity    *string
	Params      *map[string]string
	Impacts     []v1alpha1.QualityProfileRuleImpact
	Prioritized *bool
}

// findQualityProfileActiveRuleSettings parses the activated rules,
// confirms that they belong to the quality profile,
// and returns a map of rule key to its activated settings
// (severity and parameters).
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

// GenerateQualityProfileRuleObservation generates observation for a
// Quality Profile Rule.
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

// GenerateQualityProfileImpactsObservation generates observations for
// Quality Profile Rule Impacts.
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

// GenerateQualityProfileRuleImpactObservation generates observation for
// Quality Profile Rule Impact.
func GenerateQualityProfileRuleImpactObservation(impact sonar.RuleImpact) v1alpha1.QualityProfileRuleImpact {
	return v1alpha1.QualityProfileRuleImpact{
		Severity:        impact.Severity,
		SoftwareQuality: impact.SoftwareQuality,
	}
}

// IsQualityProfileRuleUpToDate checks whether the observed QualityProfileRule
// is up to date with the desired QualityProfileRuleParameters
// Compares rule key, severity (if specified), and parameters (if specified).
func IsQualityProfileRuleUpToDate(spec *v1alpha1.QualityProfileRuleParameters, observation *v1alpha1.QualityProfileRuleObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	// Rule key must match
	if !helpers.IsComparablePtrEqualComparable(spec.Rule, observation.Key) {
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

// areQualityProfileRuleImpactsUpToDate checks whether the observed
// QualityProfileRule impacts are up to date with the desired impacts.
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

// GenerateRuleCreateOptions generates SonarQube RulesCreateOption based on the
// provided Rule parameters.
// Note: Per SonarQube API, impacts and severity are mutually exclusive. If both
// are provided, impacts takes precedence.
func GenerateRuleCreateOptions(parameters *v1alpha1.RuleParameters) *sonar.RulesCreateOptions {
	rulesCreateOptions := &sonar.RulesCreateOptions{}
	if parameters == nil {
		return rulesCreateOptions
	}

	helpers.AssignIfNonNil(&rulesCreateOptions.CleanCodeAttribute, parameters.CleanCodeAttribute)
	rulesCreateOptions.CustomKey = parameters.Key
	rulesCreateOptions.MarkdownDescription = parameters.MarkdownDescription
	rulesCreateOptions.Name = parameters.Name
	rulesCreateOptions.Params = generateRuleParametersOptions(parameters.Parameters)
	// Per SonarQube API: impacts and severity cannot be used together
	// If impacts is provided and non-empty, use it; otherwise use severity
	if parameters.Impacts != nil && len(*parameters.Impacts) > 0 {
		rulesCreateOptions.Impacts = *parameters.Impacts
		// Do not set Severity when Impacts is used
	} else {
		helpers.AssignIfNonNil(&rulesCreateOptions.Severity, parameters.Severity)
	}

	helpers.AssignIfNonNil(&rulesCreateOptions.Status, parameters.Status)
	rulesCreateOptions.TemplateKey = parameters.TemplateKey
	helpers.AssignIfNonNil(&rulesCreateOptions.Type, parameters.Type)

	return rulesCreateOptions
}

// GenerateRuleUpdateOptions generates SonarQube RulesUpdateOption based on the
// provided Rule parameters and observation.
// Note: Per SonarQube API, impacts and severity are mutually exclusive.
// If both are provided, impacts takes precedence.
func GenerateRuleUpdateOptions(key string, parameters *v1alpha1.RuleParameters) *sonar.RulesUpdateOptions {
	rulesUpdateOptions := &sonar.RulesUpdateOptions{}
	if parameters == nil {
		return rulesUpdateOptions
	}

	rulesUpdateOptions.Key = key
	rulesUpdateOptions.MarkdownDescription = parameters.MarkdownDescription
	helpers.AssignIfNonNil(&rulesUpdateOptions.MarkdownNote, parameters.MarkdownNote)
	rulesUpdateOptions.Name = parameters.Name
	rulesUpdateOptions.Params = generateRuleParametersOptions(parameters.Parameters)
	helpers.AssignIfNonNil(&rulesUpdateOptions.RemediationFnBaseEffort, parameters.RemediationFnBaseEffort)
	helpers.AssignIfNonNil(&rulesUpdateOptions.RemediationFnType, parameters.RemediationFnType)
	helpers.AssignIfNonNil(&rulesUpdateOptions.RemediationFyGapMultiplier, parameters.RemediationFyGapMultiplier)
	// Per SonarQube API: impacts and severity cannot be used together
	// If impacts is provided and non-empty, use it; otherwise use severity
	if parameters.Impacts != nil && len(*parameters.Impacts) > 0 {
		rulesUpdateOptions.Impacts = *parameters.Impacts
		// Do not set Severity when Impacts is used
	} else {
		helpers.AssignIfNonNil(&rulesUpdateOptions.Severity, parameters.Severity)
	}

	helpers.AssignIfNonNil(&rulesUpdateOptions.Status, parameters.Status)
	helpers.AssignIfNonNil(&rulesUpdateOptions.Tags, parameters.Tags)

	return rulesUpdateOptions
}

// generateRuleParametersOptions generates SonarQube RuleParameters based on the
// provided Rule parameters.
func generateRuleParametersOptions(spec *map[string]v1alpha1.RuleParameterConfiguration) map[string]string {
	if spec == nil {
		return map[string]string{}
	}

	params := make(map[string]string, len(*spec))
	for key, config := range *spec {
		params[key] = config.DefaultValue
	}

	return params
}

// GenerateRuleDeleteOptions generates SonarQube RulesDeleteOption based on the
// provided Rule observation.
func GenerateRuleDeleteOptions(key string) *sonar.RulesDeleteOptions {
	return &sonar.RulesDeleteOptions{
		Key: key,
	}
}

// GenerateRuleGetOptions generates SonarQube RulesShowOption based on the
// provided Rule observation.
func GenerateRuleGetOptions(key string) *sonar.RulesShowOptions {
	return &sonar.RulesShowOptions{
		Key: key,
	}
}

// GenerateRuleObservation generates the observation for a Rule based on the
// SonarQube RulesShow response.
func GenerateRuleObservation(rule *sonar.RulesShow) v1alpha1.RuleObservation {
	if rule == nil {
		return v1alpha1.RuleObservation{}
	}

	return v1alpha1.RuleObservation{
		Name:                       rule.Rule.Name,
		Key:                        rule.Rule.Key,
		CreatedAt:                  helpers.StringToMetaTime(&rule.Rule.CreatedAt),
		UpdatedAt:                  helpers.StringToMetaTime(&rule.Rule.UpdatedAt),
		RemFnType:                  rule.Rule.RemFnType,
		RemFnBaseEffort:            rule.Rule.RemFnBaseEffort,
		RemFnGapMultiplier:         rule.Rule.RemFnGapMultiplier,
		RemFnOverloaded:            rule.Rule.RemFnOverloaded,
		HTMLNote:                   rule.Rule.HTMLNote,
		HTMLDesc:                   rule.Rule.HTMLDesc,
		MdNote:                     rule.Rule.MdNote,
		NoteLogin:                  rule.Rule.NoteLogin,
		CleanCodeAttribute:         rule.Rule.CleanCodeAttribute,
		CleanCodeAttributeCategory: rule.Rule.CleanCodeAttributeCategory,
		InternalKey:                rule.Rule.InternalKey,
		DefaultRemFnBaseEffort:     rule.Rule.DefaultRemFnBaseEffort,
		DefaultRemFnType:           rule.Rule.DefaultRemFnType,
		DefaultRemFnGapMultiplier:  rule.Rule.DefaultRemFnGapMultiplier,
		Language:                   rule.Rule.Lang,
		LanguageName:               rule.Rule.LangName,
		GapDescription:             rule.Rule.GapDescription,
		Repo:                       rule.Rule.Repo,
		Scope:                      rule.Rule.Scope,
		Severity:                   rule.Rule.Severity,
		Status:                     rule.Rule.Status,
		TemplateKey:                rule.Rule.TemplateKey,
		Type:                       rule.Rule.Type,
		Impacts:                    GenerateRuleImpactsObservation(&rule.Rule.Impacts),
		Tags:                       helpers.AnySliceToStringSlice(rule.Rule.Tags),
		SysTags:                    rule.Rule.SysTags,
		Params:                     GenerateRuleParametersObservation(&rule.Rule.Params),
		DescriptionSections:        GenerateRuleDescriptionSectionsObservation(&rule.Rule.DescriptionSections),
		IsTemplate:                 rule.Rule.IsTemplate,
		IsExternal:                 rule.Rule.IsExternal,
		Template:                   rule.Rule.Template,
	}
}

// GenerateRuleImpactsObservation generates observations for Rule impacts.
func GenerateRuleImpactsObservation(impacts *[]sonar.RuleImpact) map[string]string {
	if impacts == nil {
		return map[string]string{}
	}

	observations := make(map[string]string)
	for _, impact := range *impacts {
		observations[impact.SoftwareQuality] = impact.Severity
	}

	return observations
}

// GenerateRuleParametersObservation generates the observation of RuleParameters
// from the SonarQube Rule Show response.
func GenerateRuleParametersObservation(rule *[]sonar.RuleParam) []v1alpha1.RuleParameterObservation {
	if rule == nil {
		return []v1alpha1.RuleParameterObservation{}
	}

	observations := make([]v1alpha1.RuleParameterObservation, len(*rule))
	for i, param := range *rule {
		observations[i] = v1alpha1.RuleParameterObservation{
			DefaultValue: param.DefaultValue,
			HTMLDesc:     param.HTMLDesc,
			Desc:         param.Desc,
			Key:          param.Key,
			Type:         param.Type,
		}
	}

	return observations
}

// GenerateRuleDescriptionSectionsObservation generates the observation of
// RuleDescriptionsSections from the SonarQube rule Show response.
func GenerateRuleDescriptionSectionsObservation(sections *[]sonar.RuleDescriptionSection) []v1alpha1.RuleDescriptionSectionObservation {
	if sections == nil {
		return []v1alpha1.RuleDescriptionSectionObservation{}
	}

	observations := make([]v1alpha1.RuleDescriptionSectionObservation, len(*sections))
	for i, section := range *sections {
		observations[i] = v1alpha1.RuleDescriptionSectionObservation{
			Content: section.Content,
			Key:     section.Key,
			Context: GenerateRuleDescriptionContextObservation(&section.Context),
		}
	}

	return observations
}

// GenerateRuleDescriptionContextObservation generates the observation of
// RuleDescriptionContext from the SonarQube rule Show response.
func GenerateRuleDescriptionContextObservation(context *sonar.RuleDescriptionSectionContext) v1alpha1.RuleDescriptionContextObservation {
	if context == nil {
		return v1alpha1.RuleDescriptionContextObservation{}
	}

	return v1alpha1.RuleDescriptionContextObservation{
		DisplayName: context.DisplayName,
		Key:         context.Key,
	}
}

// IsRuleUpToDate checks whether the observed Rule is up to date with the
// desired RuleParameters.
// Compares rule key, severity (if specified), status (if specified),
// and parameters (if specified).
func IsRuleUpToDate(spec *v1alpha1.RuleParameters, observation *v1alpha1.RuleObservation) bool { //nolint:gocyclo,cyclop // This function is inherently complex due to the number of fields that need to be compared, and breaking it down would reduce readability.
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	// Keep precedence consistent with create/update behavior:
	// non-empty impacts takes precedence over severity.
	impactsAndSeverityUpToDate := helpers.IsComparablePtrEqualComparable(spec.Severity, observation.Severity)
	if spec.Impacts != nil && len(*spec.Impacts) > 0 {
		impactsAndSeverityUpToDate = helpers.IsComparableMapPtrEqualComparableMap(spec.Impacts, observation.Impacts)
	}

	return isRuleKeyUpToDate(spec.Key, observation.Key) &&
		spec.Name == observation.Name &&
		spec.TemplateKey == observation.TemplateKey &&
		helpers.IsComparablePtrEqualComparable(spec.CleanCodeAttribute, observation.CleanCodeAttribute) &&
		impactsAndSeverityUpToDate &&
		helpers.IsComparablePtrEqualComparable(spec.RemediationFnBaseEffort, observation.RemFnBaseEffort) &&
		helpers.IsComparablePtrEqualComparable(spec.RemediationFnType, observation.RemFnType) &&
		helpers.IsComparablePtrEqualComparable(spec.RemediationFyGapMultiplier, observation.RemFnGapMultiplier) &&
		helpers.IsComparablePtrEqualComparable(spec.Status, observation.Status) &&
		helpers.IsComparablePtrEqualComparable(spec.Type, observation.Type) &&
		areRuleTagsUpToDate(spec.Tags, observation.Tags) &&
		areRuleParametersUpToDate(spec.Parameters, &observation.Params)
}

// isRuleKeyUpToDate checks whether the observed rule key is up to date with the
// desired rule key.
// The rule key in SonarQube is returned as "language:key" (e.g., "java:S1234"),
// while the desired rule key in our spec is just the unprefixed key
// (e.g., "S1234").
func isRuleKeyUpToDate(specKey, observationKey string) bool {
	if specKey == observationKey {
		return true
	}

	if specKey == "" || observationKey == "" {
		return false
	}

	return strings.HasSuffix(observationKey, ":"+specKey)
}

// areRuleTagsUpToDate checks if rule tags match between spec and observation.
func areRuleTagsUpToDate(specTags *[]string, observationTags []string) bool {
	if specTags == nil {
		return true
	}

	return cmp.Equal(*specTags, observationTags, cmpopts.EquateEmpty(), cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	}))
}

// areRuleParametersUpToDate checks whether the observed Rule parameters are up
// to date with the desired parameters.
func areRuleParametersUpToDate(spec *map[string]v1alpha1.RuleParameterConfiguration, observation *[]v1alpha1.RuleParameterObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if len(*spec) != len(*observation) {
		return false
	}

	// Build a map from observation for easy lookup
	observationMap := make(map[string]v1alpha1.RuleParameterConfiguration, len(*observation))
	for _, param := range *observation {
		observationMap[param.Key] = v1alpha1.RuleParameterConfiguration{
			DefaultValue: param.DefaultValue,
			HTMLDesc:     param.HTMLDesc,
			Type:         param.Type,
		}
	}

	return cmp.Equal(*spec, observationMap, cmpopts.EquateEmpty())
}

// LateInitializeRule fills the empty fields in the RuleParameters with the
// values from RuleObservation. This is used to update the spec with default
// values from the API response during late initialization.
func LateInitializeRule(spec *v1alpha1.RuleParameters, observation *v1alpha1.RuleObservation) { //nolint:gocyclo,cyclop // complexity comes from guarding each field for non-empty values
	if spec == nil || observation == nil {
		return
	}

	if observation.CleanCodeAttribute != "" {
		helpers.AssignIfNil(&spec.CleanCodeAttribute, observation.CleanCodeAttribute)
	}

	if observation.MdNote != "" {
		helpers.AssignIfNil(&spec.MarkdownNote, observation.MdNote)
	}

	if observation.RemFnBaseEffort != "" {
		helpers.AssignIfNil(&spec.RemediationFnBaseEffort, observation.RemFnBaseEffort)
	}

	if observation.RemFnType != "" {
		helpers.AssignIfNil(&spec.RemediationFnType, observation.RemFnType)
	}

	if observation.RemFnGapMultiplier != "" {
		helpers.AssignIfNil(&spec.RemediationFyGapMultiplier, observation.RemFnGapMultiplier)
	}

	if observation.Status != "" {
		helpers.AssignIfNil(&spec.Status, observation.Status)
	}

	if observation.Type != "" {
		helpers.AssignIfNil(&spec.Type, observation.Type)
	}

	if spec.Tags == nil && len(observation.Tags) != 0 {
		tags := make([]string, len(observation.Tags))
		copy(tags, observation.Tags)

		spec.Tags = &tags
	}
}

// IsRuleLateInitialized checks whether a rule was late initialized by
// comparing the before and after RuleParameters.
// It returns true if the only differences between before and after are fields
// that can be late initialized
// (i.e., fields that are nil in before and non-nil in after),
// and false otherwise.
func IsRuleLateInitialized(before, after *v1alpha1.RuleParameters) bool {
	return !cmp.Equal(before, after, cmpopts.EquateEmpty())
}
