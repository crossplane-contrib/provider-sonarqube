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
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// selectionModeNone is the default portfolio selection mode.
	selectionModeNone = "NONE"
	// SelectionModeManual is the manual project selection mode.
	SelectionModeManual = "MANUAL"

	// qualifierApplication is the qualifier of an application added to a
	// portfolio.
	qualifierApplication = "APP"
	// qualifierPortfolio is the qualifier of a root portfolio, and of a
	// portfolio referenced in another portfolio.
	qualifierPortfolio = "VW"
	// qualifierSubPortfolio is the qualifier of a sub-portfolio created under
	// a portfolio.
	qualifierSubPortfolio = "SVW"
)

// PortfolioBranch is a branch of a project or application selected in a
// portfolio.
type PortfolioBranch struct {
	// Key is the project or application key.
	Key string
	// Branch is the branch name.
	Branch string
}

// PortfolioMembersDiff describes the changes reconciling the projects or
// applications selected in a portfolio with the desired state.
type PortfolioMembersDiff struct {
	// Add are the sorted keys of the members to add.
	Add []string
	// Remove are the sorted keys of the members to remove.
	Remove []string
	// AddBranches are the branches to select, including those of the members
	// to add.
	AddBranches []PortfolioBranch
	// RemoveBranches are the selected branches to deselect.
	RemoveBranches []PortfolioBranch
}

// IsEmpty returns true when no change is needed.
func (d PortfolioMembersDiff) IsEmpty() bool {
	return len(d.Add) == 0 && len(d.Remove) == 0 && len(d.AddBranches) == 0 && len(d.RemoveBranches) == 0
}

// PortfoliosClient is the interface for interacting with the
// SonarQube Portfolio (Views) API.
//
//nolint:interfacebloat // This interface wraps the SonarQube Views (Portfolios) API which has many methods
type PortfoliosClient interface {
	AddApplication(ctx context.Context, opt *sonar.ViewsAddApplicationOptions) (*http.Response, error)
	AddApplicationBranch(ctx context.Context, opt *sonar.ViewsAddApplicationBranchOptions) (*http.Response, error)
	AddPortfolio(ctx context.Context, opt *sonar.ViewsAddPortfolioOptions) (*http.Response, error)
	AddProject(ctx context.Context, opt *sonar.ViewsAddProjectOptions) (*http.Response, error)
	AddProjectBranch(ctx context.Context, opt *sonar.ViewsAddProjectBranchOptions) (*http.Response, error)
	Applications(ctx context.Context, opt *sonar.ViewsApplicationsOptions) (*sonar.ViewsApplications, *http.Response, error)
	Create(ctx context.Context, opt *sonar.ViewsCreateOptions) (*http.Response, error)
	Delete(ctx context.Context, opt *sonar.ViewsDeleteOptions) (*http.Response, error)
	List(ctx context.Context) (*sonar.ViewsList, *http.Response, error)
	Move(ctx context.Context, opt *sonar.ViewsMoveOptions) (*http.Response, error)
	MoveOptions(ctx context.Context, opt *sonar.ViewsMoveOptionsOptions) (*sonar.ViewsMoveDestinations, *http.Response, error)
	Projects(ctx context.Context, opt *sonar.ViewsProjectsOptions) (*sonar.ViewsProjects, *http.Response, error)
	ProjectsStatus(ctx context.Context, opt *sonar.ViewsProjectsStatusOptions) (*sonar.ViewsProjectsStatus, *http.Response, error)
	Refresh(ctx context.Context, opt *sonar.ViewsRefreshOptions) (*http.Response, error)
	RemoveApplication(ctx context.Context, opt *sonar.ViewsRemoveApplicationOptions) (*http.Response, error)
	RemoveApplicationBranch(ctx context.Context, opt *sonar.ViewsRemoveApplicationBranchOptions) (*http.Response, error)
	RemovePortfolio(ctx context.Context, opt *sonar.ViewsRemovePortfolioOptions) (*http.Response, error)
	RemoveProject(ctx context.Context, opt *sonar.ViewsRemoveProjectOptions) (*http.Response, error)
	RemoveProjectBranch(ctx context.Context, opt *sonar.ViewsRemoveProjectBranchOptions) (*http.Response, error)
	Search(ctx context.Context, opt *sonar.ViewsSearchOptions) (*sonar.ViewsSearch, *http.Response, error)
	SetManualMode(ctx context.Context, opt *sonar.ViewsSetManualModeOptions) (*http.Response, error)
	SetNoneMode(ctx context.Context, opt *sonar.ViewsSetNoneModeOptions) (*http.Response, error)
	SetRegexpMode(ctx context.Context, opt *sonar.ViewsSetRegexpModeOptions) (*http.Response, error)
	SetRemainingProjectsMode(ctx context.Context, opt *sonar.ViewsSetRemainingProjectsModeOptions) (*http.Response, error)
	SetTagsMode(ctx context.Context, opt *sonar.ViewsSetTagsModeOptions) (*http.Response, error)
	Show(ctx context.Context, opt *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error)
	SubPortfolios(ctx context.Context, opt *sonar.ViewsSubViewsOptions) (*sonar.ViewsSubViews, *http.Response, error)
	Update(ctx context.Context, opt *sonar.ViewsUpdateOptions) (*http.Response, error)
}

// NewPortfoliosClient creates a new PortfoliosClient using the
// provided SonarQube client configuration. When the observe cache is
// enabled, creating or deleting portfolios invalidates the cached datasets
// depending on projects.
func NewPortfoliosClient(clientConfig common.Config) PortfoliosClient {
	newClient := common.NewClient(clientConfig)

	return NewCachedPortfoliosClient(newClient.Views, cache.ForConfig(clientConfig))
}

// GeneratePortfolioCreateOptions generates the options for creating a
// SonarQube Portfolio from the desired state. The visibility only applies to
// root portfolios, so it is not sent when creating a sub-portfolio.
func GeneratePortfolioCreateOptions(spec *v1alpha1.PortfolioParameters) *sonar.ViewsCreateOptions {
	opts := &sonar.ViewsCreateOptions{
		Key:         spec.Key,
		Name:        spec.Name,
		Description: spec.Description,
		Visibility:  spec.Visibility,
	}

	if spec.Parent != nil {
		opts.Parent = *spec.Parent
		opts.Visibility = ""
	}

	return opts
}

// GeneratePortfolioObservation converts the SonarQube portfolio details to
// a PortfolioObservation. The parent is not returned by SonarQube: it is
// set by the caller after looking up the sub-portfolios of the desired
// parent.
func GeneratePortfolioObservation(details *sonar.ViewDetails) v1alpha1.PortfolioObservation {
	if details == nil {
		return v1alpha1.PortfolioObservation{}
	}

	observation := v1alpha1.PortfolioObservation{
		Key:           details.Key,
		Name:          details.Name,
		Description:   details.Description,
		Qualifier:     details.Qualifier,
		Visibility:    details.Visibility,
		SelectionMode: details.SelectionMode,
	}

	for _, project := range details.SelectedProjects {
		observation.Projects = append(observation.Projects, newPortfolioMemberObservation(project.ProjectKey, project.SelectedBranches))
	}

	for i := range details.SubViews {
		subView := &details.SubViews[i]

		switch subView.Qualifier {
		case qualifierApplication:
			observation.Applications = append(observation.Applications, newPortfolioMemberObservation(subViewKey(subView), subView.SelectedBranches))
		case qualifierPortfolio:
			observation.SubPortfolios = append(observation.SubPortfolios, subViewKey(subView))
		case qualifierSubPortfolio:
			observation.LocalSubPortfolios = append(observation.LocalSubPortfolios, subView.Key)
		}
	}

	sortPortfolioMembers(observation.Projects)
	sortPortfolioMembers(observation.Applications)
	slices.Sort(observation.SubPortfolios)
	slices.Sort(observation.LocalSubPortfolios)

	return observation
}

// subViewKey returns the own key of a referenced portfolio or application.
// In a portfolio, their key is prefixed by the portfolio key, and their own
// key is returned as the original key.
func subViewKey(subView *sonar.View) string {
	if subView.OriginalKey != "" {
		return subView.OriginalKey
	}

	return subView.Key
}

// newPortfolioMemberObservation builds the observation of a portfolio member
// with its sorted branches.
func newPortfolioMemberObservation(key string, branches []string) v1alpha1.PortfolioMemberObservation {
	sorted := slices.Clone(branches)
	slices.Sort(sorted)

	return v1alpha1.PortfolioMemberObservation{Key: key, Branches: sorted}
}

// sortPortfolioMembers sorts portfolio members by key.
func sortPortfolioMembers(members []v1alpha1.PortfolioMemberObservation) {
	slices.SortFunc(members, func(a, b v1alpha1.PortfolioMemberObservation) int {
		return strings.Compare(a.Key, b.Key)
	})
}

// IsPortfolioSubPortfolio returns true when the portfolio with the given
// key is a sub-portfolio created under the observed parent portfolio.
func IsPortfolioSubPortfolio(parent *sonar.ViewDetails, key string) bool {
	if parent == nil {
		return false
	}

	return slices.Contains(GeneratePortfolioObservation(parent).LocalSubPortfolios, key)
}

// DesiredPortfolioProjects returns the desired projects of a portfolio,
// keyed by project key, with the desired branches of each project. Nil
// branches mean the branches are not managed.
func DesiredPortfolioProjects(projects []v1alpha1.PortfolioProjectParameters) map[string][]string {
	desired := make(map[string][]string, len(projects))
	for _, project := range projects {
		if project.Key != nil {
			desired[*project.Key] = append(desired[*project.Key], project.Branches...)
		}
	}

	return desired
}

// DesiredPortfolioApplications returns the desired applications of a
// portfolio, keyed by application key, with the desired branches of each
// application. Nil branches mean the branches are not managed.
func DesiredPortfolioApplications(applications []v1alpha1.PortfolioApplicationParameters) map[string][]string {
	desired := make(map[string][]string, len(applications))
	for _, application := range applications {
		if application.Key != nil {
			desired[*application.Key] = append(desired[*application.Key], application.Branches...)
		}
	}

	return desired
}

// DiffPortfolioMembers computes the changes reconciling the observed
// members of a portfolio with the desired ones. Desired members with no
// branches keep the branches selected in SonarQube.
func DiffPortfolioMembers(desired map[string][]string, observation []v1alpha1.PortfolioMemberObservation) PortfolioMembersDiff {
	observed := make(map[string][]string, len(observation))
	for _, member := range observation {
		observed[member.Key] = member.Branches
	}

	var diff PortfolioMembersDiff

	for _, key := range sortedKeys(desired) {
		branches := desired[key]

		observedBranches, found := observed[key]
		if !found {
			diff.Add = append(diff.Add, key)
		}

		if len(branches) == 0 {
			continue
		}

		for _, branch := range helpers.StringSetDifference(branches, observedBranches) {
			diff.AddBranches = append(diff.AddBranches, PortfolioBranch{Key: key, Branch: branch})
		}

		for _, branch := range helpers.StringSetDifference(observedBranches, branches) {
			diff.RemoveBranches = append(diff.RemoveBranches, PortfolioBranch{Key: key, Branch: branch})
		}
	}

	for _, key := range sortedKeys(observed) {
		if _, found := desired[key]; !found {
			diff.Remove = append(diff.Remove, key)
		}
	}

	return diff
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

// PortfolioProjectsDiff returns the changes reconciling the projects
// selected in the portfolio, or an empty diff when they are not managed.
func PortfolioProjectsDiff(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) PortfolioMembersDiff {
	if spec.Projects == nil || spec.SelectionMode != SelectionModeManual {
		return PortfolioMembersDiff{}
	}

	return DiffPortfolioMembers(DesiredPortfolioProjects(*spec.Projects), observation.Projects)
}

// PortfolioApplicationsDiff returns the changes reconciling the
// applications of the portfolio, or an empty diff when they are not managed.
func PortfolioApplicationsDiff(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) PortfolioMembersDiff {
	if spec.Applications == nil {
		return PortfolioMembersDiff{}
	}

	return DiffPortfolioMembers(DesiredPortfolioApplications(*spec.Applications), observation.Applications)
}

// PortfolioSubPortfoliosToAdd returns the sorted keys of the desired
// referenced portfolios that are not yet referenced in the portfolio.
func PortfolioSubPortfoliosToAdd(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) []string {
	if spec.SubPortfolios == nil {
		return nil
	}

	return helpers.StringSetDifference(*spec.SubPortfolios, observation.SubPortfolios)
}

// PortfolioSubPortfoliosToRemove returns the sorted keys of the referenced
// portfolios that are not in the desired state.
func PortfolioSubPortfoliosToRemove(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) []string {
	if spec.SubPortfolios == nil {
		return nil
	}

	return helpers.StringSetDifference(observation.SubPortfolios, *spec.SubPortfolios)
}

// IsPortfolioParentUpToDate returns true when the portfolio is not desired
// under a parent, or is a sub-portfolio of the desired parent.
func IsPortfolioParentUpToDate(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) bool {
	return spec.Parent == nil || *spec.Parent == observation.Parent
}

// LateInitializePortfolio fills empty spec fields from the observation.
// Portfolios have no late-initializable fields; this is a no-op.
func LateInitializePortfolio(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) {
	if spec == nil || observation == nil {
		return
	}
}

// IsPortfolioLateInitialized returns true when the spec changed as a
// result of late initialization.
func IsPortfolioLateInitialized(former, current *v1alpha1.PortfolioParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !cmp.Equal(former, current)
}

// normalizeSelectionMode returns "NONE" for an empty selection mode string.
func normalizeSelectionMode(mode string) string {
	if mode == "" {
		return selectionModeNone
	}

	return mode
}

// IsPortfolioUpToDate returns true when the portfolio spec matches the
// observed state.
// Note: Regexp, Tags, and Branch are not returned by the SonarQube Show
// API, so drift in those fields is not detected.
func IsPortfolioUpToDate(spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Name != observation.Name {
		return false
	}

	if spec.Description != observation.Description {
		return false
	}

	if normalizeSelectionMode(spec.SelectionMode) != normalizeSelectionMode(observation.SelectionMode) {
		return false
	}

	if !IsPortfolioParentUpToDate(spec, observation) {
		return false
	}

	return PortfolioProjectsDiff(spec, observation).IsEmpty() &&
		PortfolioApplicationsDiff(spec, observation).IsEmpty() &&
		len(PortfolioSubPortfoliosToAdd(spec, observation)) == 0 &&
		len(PortfolioSubPortfoliosToRemove(spec, observation)) == 0
}
