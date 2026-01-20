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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"k8s.io/utils/ptr"
)

// QualityProfilesClient is the interface for the Quality Profiles SonarQube client.
type QualityProfilesClient interface {
	ActivateRule(opt *sonargo.QualityprofilesActivateRuleOption) (resp *http.Response, err error)
	ActivateRules(opt *sonargo.QualityprofilesActivateRulesOption) (resp *http.Response, err error)
	AddGroup(opt *sonargo.QualityprofilesAddGroupOption) (resp *http.Response, err error)
	AddProject(opt *sonargo.QualityprofilesAddProjectOption) (resp *http.Response, err error)
	AddUser(opt *sonargo.QualityprofilesAddUserOption) (resp *http.Response, err error)
	Backup(opt *sonargo.QualityprofilesBackupOption) (v *string, resp *http.Response, err error)
	ChangeParent(opt *sonargo.QualityprofilesChangeParentOption) (resp *http.Response, err error)
	Changelog(opt *sonargo.QualityprofilesChangelogOption) (v *sonargo.QualityprofilesChangelogObject, resp *http.Response, err error)
	Compare(opt *sonargo.QualityprofilesCompareOption) (v *sonargo.QualityprofilesCompareObject, resp *http.Response, err error)
	Copy(opt *sonargo.QualityprofilesCopyOption) (v *sonargo.QualityprofilesCopyObject, resp *http.Response, err error)
	Create(opt *sonargo.QualityprofilesCreateOption) (v *sonargo.QualityprofilesCreateObject, resp *http.Response, err error)
	DeactivateRule(opt *sonargo.QualityprofilesDeactivateRuleOption) (resp *http.Response, err error)
	DeactivateRules(opt *sonargo.QualityprofilesDeactivateRulesOption) (resp *http.Response, err error)
	Delete(opt *sonargo.QualityprofilesDeleteOption) (resp *http.Response, err error)
	Export(opt *sonargo.QualityprofilesExportOption) (v *string, resp *http.Response, err error)
	Exporters() (v *sonargo.QualityprofilesExportersObject, resp *http.Response, err error)
	Importers() (v *sonargo.QualityprofilesImportersObject, resp *http.Response, err error)
	Inheritance(opt *sonargo.QualityprofilesInheritanceOption) (v *sonargo.QualityprofilesInheritanceObject, resp *http.Response, err error)
	Projects(opt *sonargo.QualityprofilesProjectsOption) (v *sonargo.QualityprofilesProjectsObject, resp *http.Response, err error)
	RemoveGroup(opt *sonargo.QualityprofilesRemoveGroupOption) (resp *http.Response, err error)
	RemoveProject(opt *sonargo.QualityprofilesRemoveProjectOption) (resp *http.Response, err error)
	RemoveUser(opt *sonargo.QualityprofilesRemoveUserOption) (resp *http.Response, err error)
	Rename(opt *sonargo.QualityprofilesRenameOption) (resp *http.Response, err error)
	Restore(opt *sonargo.QualityprofilesRestoreOption) (resp *http.Response, err error)
	Search(opt *sonargo.QualityprofilesSearchOption) (v *sonargo.QualityprofilesSearchObject, resp *http.Response, err error)
	SearchGroups(opt *sonargo.QualityprofilesSearchGroupsOption) (v *sonargo.QualityprofilesSearchGroupsObject, resp *http.Response, err error)
	SearchUsers(opt *sonargo.QualityprofilesSearchUsersOption) (v *sonargo.QualityprofilesSearchUsersObject, resp *http.Response, err error)
	SetDefault(opt *sonargo.QualityprofilesSetDefaultOption) (resp *http.Response, err error)
	Show(opt *sonargo.QualityprofilesShowOption) (v *sonargo.QualityprofilesShowObject, resp *http.Response, err error)
}

// NewQualityProfilesClient creates a new QualityProfilesClient with the provided SonarQube client configuration.
func NewQualityProfilesClient(clientConfig common.Config) QualityProfilesClient {
	newClient := common.NewClient(clientConfig)
	return newClient.Qualityprofiles
}

// GenerateCreateQualityProfileOption generates SonarQube QualityprofilesCreateOption from QualityProfileParameters
func GenerateCreateQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonargo.QualityprofilesCreateOption {
	return &sonargo.QualityprofilesCreateOption{
		Name:     params.Name,
		Language: params.Language,
	}
}

// GenerateDeleteQualityProfileOption generates SonarQube QualityprofilesDeleteOption from QualityProfileParameters
func GenerateDeleteQualityProfileOption(params v1alpha1.QualityProfileParameters) *sonargo.QualityprofilesDeleteOption {
	return &sonargo.QualityprofilesDeleteOption{
		Language:       params.Language,
		QualityProfile: params.Name,
	}
}

// GenerateRenameQualityProfileOption generates SonarQube QualityprofilesRenameOption from QualityProfileParameters
func GenerateRenameQualityProfileOption(key string, params v1alpha1.QualityProfileParameters) *sonargo.QualityprofilesRenameOption {
	return &sonargo.QualityprofilesRenameOption{
		Key:  key,
		Name: params.Name,
	}
}

// GenerateQualityProfileObservation generates QualityProfileObservation from SonarQube QualityprofilesShowObject
func GenerateQualityProfileObservation(observation *sonargo.QualityprofilesShowObject, rules *sonargo.RulesSearchObject) v1alpha1.QualityProfileObservation {
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
func GenerateQualityprofilesSetDefaultOption(params v1alpha1.QualityProfileParameters) *sonargo.QualityprofilesSetDefaultOption {
	return &sonargo.QualityprofilesSetDefaultOption{
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
func GenerateQualityProfileActivateRuleOption(qualityProfileKey string, params v1alpha1.QualityProfileRuleParameters) *sonargo.QualityprofilesActivateRuleOption {
	activateRulesOption := &sonargo.QualityprofilesActivateRuleOption{
		Key:             qualityProfileKey,
		Rule:            params.Rule,
		PrioritizedRule: common.STR_FALSE,
	}

	if ptr.Deref(params.Prioritized, false) {
		activateRulesOption.PrioritizedRule = common.STR_TRUE
	}

	// Per SonarQube API: impacts and severity cannot be used together
	// If impacts is provided, use it; otherwise use severity
	if impacts := helpers.MapToSemicolonSeparatedString(params.Impacts); impacts != "" {
		activateRulesOption.Impacts = impacts
	} else if params.Severity != nil {
		activateRulesOption.Severity = *params.Severity
	}

	if parameters := helpers.MapToSemicolonSeparatedString(params.Parameters); parameters != "" {
		activateRulesOption.Params = parameters
	}

	return activateRulesOption
}

// GenerateQualityProfileDeactivateRuleOption generates SonarQube QualityprofilesDeactivateRuleOption from rule key
func GenerateQualityProfileDeactivateRuleOption(qualityProfileKey string, ruleKey string) *sonargo.QualityprofilesDeactivateRuleOption {
	return &sonargo.QualityprofilesDeactivateRuleOption{
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
