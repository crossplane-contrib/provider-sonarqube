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
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// QualityProfilesClient is the interface for the Quality Profiles
// SonarQube client.
//
//nolint:interfacebloat // This interface wraps the SonarQube Quality Profiles API which has 26 methods
type QualityProfilesClient interface {
	ActivateRule(opt *sonar.QualityprofilesActivateRuleOptions) (resp *http.Response, err error)
	ActivateRules(opt *sonar.QualityprofilesActivateRulesOptions) (resp *http.Response, err error)
	AddGroup(opt *sonar.QualityprofilesAddGroupOptions) (resp *http.Response, err error)
	AddProject(opt *sonar.QualityprofilesAddProjectOptions) (resp *http.Response, err error)
	AddUser(opt *sonar.QualityprofilesAddUserOptions) (resp *http.Response, err error)
	Backup(opt *sonar.QualityprofilesBackupOptions) (v *string, resp *http.Response, err error)
	ChangeParent(opt *sonar.QualityprofilesChangeParentOptions) (resp *http.Response, err error)
	Changelog(opt *sonar.QualityprofilesChangelogOptions) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error)
	Compare(opt *sonar.QualityprofilesCompareOptions) (v *sonar.QualityprofilesCompare, resp *http.Response, err error)
	Copy(opt *sonar.QualityprofilesCopyOptions) (v *sonar.QualityprofilesCopy, resp *http.Response, err error)
	Create(opt *sonar.QualityprofilesCreateOptions) (v *sonar.QualityprofilesCreate, resp *http.Response, err error)
	DeactivateRule(opt *sonar.QualityprofilesDeactivateRuleOptions) (resp *http.Response, err error)
	DeactivateRules(opt *sonar.QualityprofilesDeactivateRulesOptions) (resp *http.Response, err error)
	Delete(opt *sonar.QualityprofilesDeleteOptions) (resp *http.Response, err error)
	Inheritance(opt *sonar.QualityprofilesInheritanceOptions) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error)
	Projects(opt *sonar.QualityprofilesProjectsOptions) (v *sonar.QualityprofilesProjects, resp *http.Response, err error)
	RemoveGroup(opt *sonar.QualityprofilesRemoveGroupOptions) (resp *http.Response, err error)
	RemoveProject(opt *sonar.QualityprofilesRemoveProjectOptions) (resp *http.Response, err error)
	RemoveUser(opt *sonar.QualityprofilesRemoveUserOptions) (resp *http.Response, err error)
	Rename(opt *sonar.QualityprofilesRenameOptions) (resp *http.Response, err error)
	Restore(opt *sonar.QualityprofilesRestoreOptions) (resp *http.Response, err error)
	Search(opt *sonar.QualityprofilesSearchOptions) (v *sonar.QualityprofilesSearch, resp *http.Response, err error)
	SearchGroups(opt *sonar.QualityprofilesSearchGroupsOptions) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error)
	SearchUsers(opt *sonar.QualityprofilesSearchUsersOptions) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error)
	SetDefault(opt *sonar.QualityprofilesSetDefaultOptions) (resp *http.Response, err error)
	Show(opt *sonar.QualityprofilesShowOptions) (v *sonar.QualityprofilesShow, resp *http.Response, err error)
}

// NewQualityProfilesClient creates a new QualityProfilesClient with the
// provided SonarQube client configuration.
func NewQualityProfilesClient(clientConfig common.Config) QualityProfilesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Qualityprofiles
}

// GenerateCreateQualityProfileOption generates SonarQube
// QualityprofilesCreateOption from QualityProfileParameters.
func GenerateCreateQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesCreateOptions {
	return &sonar.QualityprofilesCreateOptions{
		Name:     params.Name,
		Language: params.Language,
	}
}

// GenerateDeleteQualityProfileOption generates SonarQube
// QualityprofilesDeleteOption from QualityProfileParameters.
func GenerateDeleteQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesDeleteOptions {
	return &sonar.QualityprofilesDeleteOptions{
		Language:       params.Language,
		QualityProfile: params.Name,
	}
}

// GenerateRenameQualityProfileOption generates SonarQube
// QualityprofilesRenameOption from QualityProfileParameters.
func GenerateRenameQualityProfileOption(key string, params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesRenameOptions {
	return &sonar.QualityprofilesRenameOptions{
		Key:  key,
		Name: params.Name,
	}
}

// GenerateQualityProfileObservation generates QualityProfileObservation
// from SonarQube QualityprofilesShow.
func GenerateQualityProfileObservation(observation *sonar.QualityprofilesShow, rules *sonar.RulesSearch) v1alpha1.QualityProfileObservation {
	return v1alpha1.QualityProfileObservation{
		ActiveDeprecatedRuleCount: observation.Profile.ActiveDeprecatedRuleCount,
		ActiveRuleCount:           observation.Profile.ActiveRuleCount,
		IsBuiltIn:                 observation.Profile.IsBuiltIn,
		IsDefault:                 observation.Profile.IsDefault,
		IsInherited:               observation.Profile.IsInherited,
		Key:                       observation.Profile.Key,
		Language:                  observation.Profile.Language,
		LanguageName:              observation.Profile.LanguageName,
		LastUsed:                  helpers.StringToMetaTime(&observation.Profile.LastUsed),
		Name:                      observation.Profile.Name,
		ProjectCount:              observation.Profile.ProjectCount,
		RulesUpdatedAt:            helpers.StringToMetaTime(&observation.Profile.RulesUpdatedAt),
		Rules:                     GenerateQualityProfileRulesObservation(observation.Profile.Key, rules),
	}
}

// GenerateQualityprofilesSetDefaultOption generates SonarQube
// QualityprofilesSetDefaultOption from QualityProfileParameters.
func GenerateQualityprofilesSetDefaultOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesSetDefaultOptions {
	return &sonar.QualityprofilesSetDefaultOptions{
		QualityProfile: params.Name,
		Language:       params.Language,
	}
}

// IsQualityProfileUpToDate checks whether the observed QualityProfile is
// up to date with the desired QualityProfileParameters
// It also checks that all rule associations are up to date.
func IsQualityProfileUpToDate(spec *v1alpha1.QualityProfileParameters, observation *v1alpha1.QualityProfileObservation, associations map[string]QualityProfileRuleAssociation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Name != observation.Name {
		return false
	}

	if spec.Language != observation.Language {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(spec.Default, observation.IsDefault) {
		return false
	}

	// Check if all rules are up to date
	if !AreQualityProfileRulesUpToDate(associations) {
		return false
	}

	return true
}

// LateInitializeQualityProfile fills the empty fields in
// *QualityProfileParameters with the values seen in QualityProfileObservation.
func LateInitializeQualityProfile(spec *v1alpha1.QualityProfileParameters, observation *v1alpha1.QualityProfileObservation, associations map[string]QualityProfileRuleAssociation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Default, observation.IsDefault)

	// Late initialize rules
	LateInitializeQualityProfileRules(associations)
}

// GenerateQualityProfileActivateRuleOption generates SonarQube
// QualityprofilesActivateRuleOption from QualityProfileRuleParameters
// Per SonarQube API, impacts and severity cannot be used at the same time.
// If both are provided, impacts takes precedence.
func GenerateQualityProfileActivateRuleOption(qualityProfileKey string, params v1alpha1.QualityProfileRuleParameters) *sonar.QualityprofilesActivateRuleOptions {
	activateRulesOption := &sonar.QualityprofilesActivateRuleOptions{
		Key:             qualityProfileKey,
		Rule:            ptr.Deref(params.Rule, ""),
		PrioritizedRule: false,
	}

	if ptr.Deref(params.Prioritized, false) {
		activateRulesOption.PrioritizedRule = true
	}

	// Per SonarQube API: impacts and severity cannot be used together
	// If impacts is provided, use it; otherwise use severity
	if params.Impacts != nil && len(*params.Impacts) > 0 {
		activateRulesOption.Impacts = *params.Impacts
	} else if params.Severity != nil {
		activateRulesOption.Severity = *params.Severity
	}

	if params.Parameters != nil && len(*params.Parameters) > 0 {
		activateRulesOption.Params = *params.Parameters
	}

	return activateRulesOption
}

// GenerateQualityProfileDeactivateRuleOption generates SonarQube
// QualityprofilesDeactivateRuleOption from rule key.
func GenerateQualityProfileDeactivateRuleOption(qualityProfileKey, ruleKey string) *sonar.QualityprofilesDeactivateRuleOptions {
	return &sonar.QualityprofilesDeactivateRuleOptions{
		Key:  qualityProfileKey,
		Rule: ruleKey,
	}
}

// QualityProfileRuleAssociation associates a QualityProfileRuleObservation with
// its corresponding QualityProfileRuleParameters.
type QualityProfileRuleAssociation struct {
	Observation *v1alpha1.QualityProfileRuleObservation
	Spec        *v1alpha1.QualityProfileRuleParameters
	UpToDate    bool
}

// GenerateQualityProfileRulesAssociation generates associations between
// QualityProfileRuleParameters and QualityProfileRuleObservation
// The key in the returned map is the rule key (which is unique per rule).
func GenerateQualityProfileRulesAssociation(specs []v1alpha1.QualityProfileRuleParameters, observations []v1alpha1.QualityProfileRuleObservation) map[string]QualityProfileRuleAssociation {
	associations := make(map[string]QualityProfileRuleAssociation)

	// First, populate with all observations using rule key as the map key
	for i := range observations {
		associations[observations[i].Key] = QualityProfileRuleAssociation{
			Observation: &observations[i],
			Spec:        nil,
			UpToDate:    false,
		}
	}

	// Then, match specs to observations
	for idx := range specs {
		ruleKey := ptr.Deref(specs[idx].Rule, "")
		if ruleKey == "" {
			continue
		}

		if assoc, exists := associations[ruleKey]; exists {
			// Rule exists in observation - check if up to date
			assoc.Spec = &specs[idx]
			assoc.UpToDate = IsQualityProfileRuleUpToDate(&specs[idx], assoc.Observation)
			associations[ruleKey] = assoc
		} else {
			// Rule not currently active - needs activation
			associations[ruleKey] = QualityProfileRuleAssociation{
				Observation: nil,
				Spec:        &specs[idx],
				UpToDate:    false,
			}
		}
	}

	return associations
}

// AreQualityProfileRulesUpToDate checks whether all rule associations are
// up to date.
func AreQualityProfileRulesUpToDate(associations map[string]QualityProfileRuleAssociation) bool {
	for _, assoc := range associations {
		if !assoc.UpToDate {
			return false
		}
	}

	return true
}

// FindNonExistingQualityProfileRules finds QualityProfileRuleParameters that
// do not have a corresponding QualityProfileRuleObservation
// These are rules that need to be activated.
func FindNonExistingQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []*v1alpha1.QualityProfileRuleParameters {
	var nonExisting []*v1alpha1.QualityProfileRuleParameters

	for _, assoc := range associations {
		if assoc.Observation == nil && assoc.Spec != nil {
			nonExisting = append(nonExisting, assoc.Spec)
		}
	}

	return nonExisting
}

// FindMissingQualityProfileRules finds QualityProfileRuleObservations that
// do not have a corresponding QualityProfileRuleParameters
// These are rules that need to be deactivated.
func FindMissingQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []*v1alpha1.QualityProfileRuleObservation {
	var missing []*v1alpha1.QualityProfileRuleObservation

	for _, assoc := range associations {
		if assoc.Spec == nil && assoc.Observation != nil {
			missing = append(missing, assoc.Observation)
		}
	}

	return missing
}

// FindNotUpToDateQualityProfileRules finds QualityProfileRuleAssociations
// that are not up to date
// These are rules where both spec and observation exist but have
// different parameters.
func FindNotUpToDateQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []QualityProfileRuleAssociation {
	var notUpToDate []QualityProfileRuleAssociation

	for _, assoc := range associations {
		if !assoc.UpToDate && assoc.Spec != nil && assoc.Observation != nil {
			notUpToDate = append(notUpToDate, assoc)
		}
	}

	return notUpToDate
}

// LateInitializeQualityProfileRules fills the empty fields in
// QualityProfileRuleParameters with the values seen in
// QualityProfileRuleObservation for the rules that exist in both
// spec and observation.
func LateInitializeQualityProfileRules(associations map[string]QualityProfileRuleAssociation) {
	for key, assoc := range associations {
		if assoc.Spec != nil && assoc.Observation != nil {
			helpers.AssignIfNil(&assoc.Spec.Severity, assoc.Observation.Severity)
			helpers.AssignIfNil(&assoc.Spec.Prioritized, assoc.Observation.Prioritized)
			// Note: impacts is a map, so we only late initialize it if it's currently nil to avoid overwriting any existing values
			if assoc.Spec.Impacts == nil && len(assoc.Observation.Impacts) > 0 {
				impactsMap := make(map[string]string)
				for idx := range assoc.Observation.Impacts {
					impactsMap[assoc.Observation.Impacts[idx].SoftwareQuality] = assoc.Observation.Impacts[idx].Severity
				}

				assoc.Spec.Impacts = &impactsMap
			}

			associations[key] = assoc
		}
	}
}

// WereQualityProfileRulesLateInitialized checks if any rule parameters
// were updated during late initialization
// This compares the original rules slice with the updated one after
// late initialization.
func WereQualityProfileRulesLateInitialized(original, updated []v1alpha1.QualityProfileRuleParameters) bool {
	if len(original) != len(updated) {
		return true
	}

	// Build a map for quick lookup of original rules
	originalMap := make(map[string]*v1alpha1.QualityProfileRuleParameters, len(original))
	for idx := range original {
		ruleKey := ptr.Deref(original[idx].Rule, "")
		if ruleKey == "" {
			continue
		}

		originalMap[ruleKey] = &original[idx]
	}

	for idx := range updated {
		ruleKey := ptr.Deref(updated[idx].Rule, "")
		if ruleKey == "" {
			return true
		}

		orig, exists := originalMap[ruleKey]
		if !exists {
			return true
		}
		// Compare the spec fields that could be late-initialized
		if !helpers.IsComparablePtrEqualComparablePtr(orig.Severity, updated[idx].Severity) {
			return true
		}

		if !helpers.IsComparablePtrEqualComparablePtr(orig.Prioritized, updated[idx].Prioritized) {
			return true
		}
	}

	return false
}

// GenerateQualityProfilesSearchProjectOptions generates the options for
// searching a SonarQube Quality Profile by project based on the
// provided project key.
func GenerateQualityProfilesSearchProjectOptions(projectKey string) *sonar.QualityprofilesSearchOptions {
	return &sonar.QualityprofilesSearchOptions{
		Project: projectKey,
	}
}

// GenerateQualityProfileAddProjectOptions generates the options
// for adding a SonarQube Quality Profile to a project based on the provided
// project key, quality profile name, and language.
func GenerateQualityProfileAddProjectOptions(projectKey, qualityProfileName, language string) *sonar.QualityprofilesAddProjectOptions {
	return &sonar.QualityprofilesAddProjectOptions{
		Project:        projectKey,
		QualityProfile: qualityProfileName,
		Language:       language,
	}
}

// GenerateQualityProfilesSearchProjectObservation generates the observation
// for a SonarQube Quality Profile associated with a project based on the
// provided QualityprofilesSearch response.
// The observation is keyed by language so that
// AreProjectQualityProfilesUpToDate can look up profiles by language
// (matching the spec map key).
func GenerateQualityProfilesSearchProjectObservation(observation *sonar.QualityprofilesSearch) map[string]v1alpha1.ProjectQualityProfileObservation {
	qualityProfiles := make(map[string]v1alpha1.ProjectQualityProfileObservation, len(observation.Profiles))

	for _, profile := range observation.Profiles {
		qualityProfiles[profile.Language] = v1alpha1.ProjectQualityProfileObservation{
			Id:   profile.Key,
			Name: profile.Name,
		}
	}

	return qualityProfiles
}

// AreProjectQualityProfilesUpToDate checks whether the observed project quality
// profiles are up to date with the desired project quality profile references.
// When the spec is empty, the quality profiles are considered up to date
// because no explicit assignment is required.
// The observation may contain more entries than the spec
// (e.g. default built-in profiles for languages not referenced in the spec);
// only the languages present in the spec are checked.
func AreProjectQualityProfilesUpToDate(spec map[string]v1alpha1.ProjectQualityProfileReference, observation map[string]v1alpha1.ProjectQualityProfileObservation) bool {
	if len(spec) == 0 {
		return true
	}

	for language, specProfile := range spec {
		obsProfile, exists := observation[language]
		if !exists || !helpers.IsComparablePtrEqualComparable(specProfile.Id, obsProfile.Id) {
			return false
		}
	}

	return true
}

// GenerateQualityProfileShowOptions generates the options for showing a
// SonarQube Quality Profile based on the provided quality profile key.
func GenerateQualityProfileShowOptions(qualityProfileKey string) *sonar.QualityprofilesShowOptions {
	return &sonar.QualityprofilesShowOptions{
		Key: qualityProfileKey,
	}
}
