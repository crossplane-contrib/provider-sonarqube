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
	"k8s.io/utils/ptr"
)

// QualityProfilesClient is the interface for the Quality Profiles SonarQube client.
type QualityProfilesClient interface {
	ActivateRule(opt *sonar.QualityprofilesActivateRuleOption) (resp *http.Response, err error)
	ActivateRules(opt *sonar.QualityprofilesActivateRulesOption) (resp *http.Response, err error)
	AddGroup(opt *sonar.QualityprofilesAddGroupOption) (resp *http.Response, err error)
	AddProject(opt *sonar.QualityprofilesAddProjectOption) (resp *http.Response, err error)
	AddUser(opt *sonar.QualityprofilesAddUserOption) (resp *http.Response, err error)
	Backup(opt *sonar.QualityprofilesBackupOption) (v *string, resp *http.Response, err error)
	ChangeParent(opt *sonar.QualityprofilesChangeParentOption) (resp *http.Response, err error)
	Changelog(opt *sonar.QualityprofilesChangelogOption) (v *sonar.QualityprofilesChangelog, resp *http.Response, err error)
	Compare(opt *sonar.QualityprofilesCompareOption) (v *sonar.QualityprofilesCompare, resp *http.Response, err error)
	Copy(opt *sonar.QualityprofilesCopyOption) (v *sonar.QualityprofilesCopy, resp *http.Response, err error)
	Create(opt *sonar.QualityprofilesCreateOption) (v *sonar.QualityprofilesCreate, resp *http.Response, err error)
	DeactivateRule(opt *sonar.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error)
	DeactivateRules(opt *sonar.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error)
	Delete(opt *sonar.QualityprofilesDeleteOption) (resp *http.Response, err error)
	Inheritance(opt *sonar.QualityprofilesInheritanceOption) (v *sonar.QualityprofilesInheritance, resp *http.Response, err error)
	Projects(opt *sonar.QualityprofilesProjectsOption) (v *sonar.QualityprofilesProjects, resp *http.Response, err error)
	RemoveGroup(opt *sonar.QualityprofilesRemoveGroupOption) (resp *http.Response, err error)
	RemoveProject(opt *sonar.QualityprofilesRemoveProjectOption) (resp *http.Response, err error)
	RemoveUser(opt *sonar.QualityprofilesRemoveUserOption) (resp *http.Response, err error)
	Rename(opt *sonar.QualityprofilesRenameOption) (resp *http.Response, err error)
	Restore(opt *sonar.QualityprofilesRestoreOption) (resp *http.Response, err error)
	Search(opt *sonar.QualityprofilesSearchOption) (v *sonar.QualityprofilesSearch, resp *http.Response, err error)
	SearchGroups(opt *sonar.QualityprofilesSearchGroupsOption) (v *sonar.QualityprofilesSearchGroups, resp *http.Response, err error)
	SearchUsers(opt *sonar.QualityprofilesSearchUsersOption) (v *sonar.QualityprofilesSearchUsers, resp *http.Response, err error)
	SetDefault(opt *sonar.QualityprofilesSetDefaultOption) (resp *http.Response, err error)
	Show(opt *sonar.QualityprofilesShowOption) (v *sonar.QualityprofilesShow, resp *http.Response, err error)
}

// NewQualityProfilesClient creates a new QualityProfilesClient with the provided SonarQube client configuration.
func NewQualityProfilesClient(clientConfig common.Config) QualityProfilesClient {
	newClient := common.NewClient(clientConfig)
	return newClient.Qualityprofiles
}

// GenerateCreateQualityProfileOption generates SonarQube QualityprofilesCreateOption from QualityProfileParameters
func GenerateCreateQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesCreateOption {
	return &sonar.QualityprofilesCreateOption{
		Name:     params.Name,
		Language: params.Language,
	}
}

// GenerateDeleteQualityProfileOption generates SonarQube QualityprofilesDeleteOption from QualityProfileParameters
func GenerateDeleteQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesDeleteOption {
	return &sonar.QualityprofilesDeleteOption{
		Language:       params.Language,
		QualityProfile: params.Name,
	}
}

// GenerateRenameQualityProfileOption generates SonarQube QualityprofilesRenameOption from QualityProfileParameters
func GenerateRenameQualityProfileOption(key string, params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesRenameOption {
	return &sonar.QualityprofilesRenameOption{
		Key:  key,
		Name: params.Name,
	}
}

// GenerateQualityProfileObservation generates QualityProfileObservation from SonarQube QualityprofilesShow
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
		Rules:                     GenerateQualityProfileRulesObservation(rules),
	}
}

// GenerateQualityprofilesSetDefaultOption generates SonarQube QualityprofilesSetDefaultOption from QualityProfileParameters
func GenerateQualityprofilesSetDefaultOption(params v1alpha1.QualityProfileParameters) *sonar.QualityprofilesSetDefaultOption {
	return &sonar.QualityprofilesSetDefaultOption{
		QualityProfile: params.Name,
		Language:       params.Language,
	}
}

// IsQualityProfileUpToDate checks whether the observed QualityProfile is up to date with the desired QualityProfileParameters
// It also checks that all rule associations are up to date
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

// LateInitializeQualityProfile fills the empty fields in *QualityProfileParameters with
// the values seen in QualityProfileObservation.
func LateInitializeQualityProfile(spec *v1alpha1.QualityProfileParameters, observation *v1alpha1.QualityProfileObservation) {
	if spec == nil || observation == nil {
		return
	}
	helpers.AssignIfNil(&spec.Default, observation.IsDefault)
}

// GenerateQualityProfileActivateRuleOption generates SonarQube QualityprofilesActivateRuleOption from QualityProfileRuleParameters
// Note: Per SonarQube API, impacts and severity cannot be used at the same time. If both are provided, impacts takes precedence.
func GenerateQualityProfileActivateRuleOption(qualityProfileKey string, params v1alpha1.QualityProfileRuleParameters) *sonar.QualityprofilesActivateRuleOption {
	activateRulesOption := &sonar.QualityprofilesActivateRuleOption{
		Key:             qualityProfileKey,
		Rule:            params.Rule,
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

// GenerateQualityProfileDeactivateRuleOption generates SonarQube QualityprofilesDeactivateRuleOption from rule key
func GenerateQualityProfileDeactivateRuleOption(qualityProfileKey string, ruleKey string) *sonar.QualityprofilesDeactivateRuleOption {
	return &sonar.QualityprofilesDeactivateRuleOption{
		Key:  qualityProfileKey,
		Rule: ruleKey,
	}
}

// QualityProfileRuleAssociation associates a QualityProfileRuleObservation with its corresponding QualityProfileRuleParameters
type QualityProfileRuleAssociation struct {
	Observation *v1alpha1.QualityProfileRuleObservation
	Spec        *v1alpha1.QualityProfileRuleParameters
	UpToDate    bool
}

// GenerateQualityProfileRulesAssociation generates associations between QualityProfileRuleParameters and QualityProfileRuleObservation
// The key in the returned map is the rule key (which is unique per rule)
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
	for i := range specs {
		ruleKey := specs[i].Rule
		if assoc, exists := associations[ruleKey]; exists {
			// Rule exists in observation - check if up to date
			assoc.Spec = &specs[i]
			assoc.UpToDate = IsQualityProfileRuleUpToDate(&specs[i], assoc.Observation)
			associations[ruleKey] = assoc
		} else {
			// Rule not currently active - needs activation
			associations[ruleKey] = QualityProfileRuleAssociation{
				Observation: nil,
				Spec:        &specs[i],
				UpToDate:    false,
			}
		}
	}

	return associations
}

// AreQualityProfileRulesUpToDate checks whether all rule associations are up to date
func AreQualityProfileRulesUpToDate(associations map[string]QualityProfileRuleAssociation) bool {
	for _, assoc := range associations {
		if !assoc.UpToDate {
			return false
		}
	}
	return true
}

// FindNonExistingQualityProfileRules finds QualityProfileRuleParameters that do not have a corresponding QualityProfileRuleObservation
// These are rules that need to be activated
func FindNonExistingQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []*v1alpha1.QualityProfileRuleParameters {
	var nonExisting []*v1alpha1.QualityProfileRuleParameters
	for _, assoc := range associations {
		if assoc.Observation == nil && assoc.Spec != nil {
			nonExisting = append(nonExisting, assoc.Spec)
		}
	}
	return nonExisting
}

// FindMissingQualityProfileRules finds QualityProfileRuleObservations that do not have a corresponding QualityProfileRuleParameters
// These are rules that need to be deactivated
func FindMissingQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []*v1alpha1.QualityProfileRuleObservation {
	var missing []*v1alpha1.QualityProfileRuleObservation
	for _, assoc := range associations {
		if assoc.Spec == nil && assoc.Observation != nil {
			missing = append(missing, assoc.Observation)
		}
	}
	return missing
}

// FindNotUpToDateQualityProfileRules finds QualityProfileRuleAssociations that are not up to date
// These are rules where both spec and observation exist but have different parameters
func FindNotUpToDateQualityProfileRules(associations map[string]QualityProfileRuleAssociation) []QualityProfileRuleAssociation {
	var notUpToDate []QualityProfileRuleAssociation
	for _, assoc := range associations {
		if !assoc.UpToDate && assoc.Spec != nil && assoc.Observation != nil {
			notUpToDate = append(notUpToDate, assoc)
		}
	}
	return notUpToDate
}

// WereQualityProfileRulesLateInitialized checks if any rule parameters were updated during late initialization
// This compares the original rules slice with the updated one after late initialization
func WereQualityProfileRulesLateInitialized(original, updated []v1alpha1.QualityProfileRuleParameters) bool {
	if len(original) != len(updated) {
		return true
	}

	// Build a map for quick lookup of original rules
	originalMap := make(map[string]*v1alpha1.QualityProfileRuleParameters, len(original))
	for i := range original {
		originalMap[original[i].Rule] = &original[i]
	}

	for i := range updated {
		orig, exists := originalMap[updated[i].Rule]
		if !exists {
			return true
		}
		// Compare the spec fields that could be late-initialized
		if !helpers.IsComparablePtrEqualComparablePtr(orig.Severity, updated[i].Severity) {
			return true
		}
		if !helpers.IsComparablePtrEqualComparablePtr(orig.Prioritized, updated[i].Prioritized) {
			return true
		}
	}

	return false
}
