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
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// ApplicationsClient is the interface for interacting with the
// SonarQube Applications API.
//
//nolint:interfacebloat // This interface wraps the SonarQube Applications API which has many methods
type ApplicationsClient interface {
	AddProject(ctx context.Context, opt *sonar.ApplicationsAddProjectOptions) (*http.Response, error)
	Create(ctx context.Context, opt *sonar.ApplicationsCreateOptions) (*sonar.ApplicationsCreate, *http.Response, error)
	CreateBranch(ctx context.Context, opt *sonar.ApplicationsCreateBranchOptions) (*http.Response, error)
	Delete(ctx context.Context, opt *sonar.ApplicationsDeleteOptions) (*http.Response, error)
	DeleteBranch(ctx context.Context, opt *sonar.ApplicationsDeleteBranchOptions) (*http.Response, error)
	Refresh(ctx context.Context, opt *sonar.ApplicationsRefreshOptions) (*http.Response, error)
	RemoveProject(ctx context.Context, opt *sonar.ApplicationsRemoveProjectOptions) (*http.Response, error)
	SearchAllProjects(ctx context.Context, opt *sonar.ApplicationsSearchProjectsOptions) ([]sonar.ApplicationProject, *http.Response, error)
	SearchProjects(ctx context.Context, opt *sonar.ApplicationsSearchProjectsOptions) (*sonar.ApplicationsSearchProjects, *http.Response, error)
	SetTags(ctx context.Context, opt *sonar.ApplicationsSetTagsOptions) (*http.Response, error)
	Show(ctx context.Context, opt *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error)
	ShowLeak(ctx context.Context, opt *sonar.ApplicationsShowLeakOptions) (*sonar.ApplicationsShowLeak, *http.Response, error)
	Update(ctx context.Context, opt *sonar.ApplicationsUpdateOptions) (*http.Response, error)
	UpdateBranch(ctx context.Context, opt *sonar.ApplicationsUpdateBranchOptions) (*http.Response, error)
}

// NewApplicationsClient creates a new ApplicationsClient using the
// provided SonarQube client configuration. When the observe cache is
// enabled, creating or deleting applications invalidates the cached datasets
// depending on projects.
func NewApplicationsClient(clientConfig common.Config) ApplicationsClient {
	newClient := common.NewClient(clientConfig)

	return NewCachedApplicationsClient(newClient.Applications, cache.ForConfig(clientConfig))
}

// GenerateApplicationCreateOptions generates the options for creating a
// SonarQube Application from the desired state.
func GenerateApplicationCreateOptions(spec *v1alpha1.ApplicationParameters) *sonar.ApplicationsCreateOptions {
	return &sonar.ApplicationsCreateOptions{
		Key:         spec.Key,
		Name:        spec.Name,
		Description: ptr.Deref(spec.Description, ""),
		Visibility:  spec.Visibility,
	}
}

// GenerateApplicationUpdateOptions generates the options for updating the
// name and description of a SonarQube Application. When the description is
// not managed, the observed description is sent back unchanged.
func GenerateApplicationUpdateOptions(key string, spec *v1alpha1.ApplicationParameters, observation *v1alpha1.ApplicationObservation) *sonar.ApplicationsUpdateOptions {
	description := observation.Description
	if spec.Description != nil {
		description = *spec.Description
	}

	return &sonar.ApplicationsUpdateOptions{
		Application: key,
		Name:        spec.Name,
		Description: description,
	}
}

// GenerateApplicationCreateBranchOptions generates the options for creating
// an Application branch from the desired state.
func GenerateApplicationCreateBranchOptions(key string, branch *v1alpha1.ApplicationBranchParameters) *sonar.ApplicationsCreateBranchOptions {
	projects, projectBranches := splitApplicationBranchProjects(branch.Projects)

	return &sonar.ApplicationsCreateBranchOptions{
		Application:   key,
		Branch:        branch.Name,
		Project:       projects,
		ProjectBranch: projectBranches,
	}
}

// GenerateApplicationUpdateBranchOptions generates the options for updating
// the project branches of an existing Application branch.
func GenerateApplicationUpdateBranchOptions(key string, branch *v1alpha1.ApplicationBranchParameters) *sonar.ApplicationsUpdateBranchOptions {
	projects, projectBranches := splitApplicationBranchProjects(branch.Projects)

	return &sonar.ApplicationsUpdateBranchOptions{
		Application:   key,
		Branch:        branch.Name,
		Name:          branch.Name,
		Project:       projects,
		ProjectBranch: projectBranches,
	}
}

// splitApplicationBranchProjects splits the project branch selections into
// the two parallel lists expected by the SonarQube API, sorted by project key
// so requests are deterministic.
func splitApplicationBranchProjects(selections []v1alpha1.ApplicationBranchProjectParameters) (projects, projectBranches []string) {
	sorted := slices.Clone(selections)
	slices.SortFunc(sorted, func(a, b v1alpha1.ApplicationBranchProjectParameters) int {
		return strings.Compare(a.Project, b.Project)
	})

	projects = make([]string, 0, len(sorted))
	projectBranches = make([]string, 0, len(sorted))

	for _, selection := range sorted {
		projects = append(projects, selection.Project)
		projectBranches = append(projectBranches, selection.Branch)
	}

	return projects, projectBranches
}

// GenerateApplicationObservation converts SonarQube application details to
// an ApplicationObservation. branchProjects maps each non-main branch name to
// the project branches returned when showing the application on that branch.
func GenerateApplicationObservation(details *sonar.ApplicationDetails, branchProjects map[string][]sonar.ApplicationProject) v1alpha1.ApplicationObservation {
	if details == nil {
		return v1alpha1.ApplicationObservation{}
	}

	projects := make([]string, 0, len(details.Projects))
	for _, project := range details.Projects {
		projects = append(projects, project.Key)
	}

	slices.Sort(projects)

	var branches []v1alpha1.ApplicationBranchObservation

	for _, branch := range details.Branches {
		source := branchProjects[branch.Name]
		if branch.IsMain {
			source = details.Projects
		}

		branches = append(branches, v1alpha1.ApplicationBranchObservation{
			Name:     branch.Name,
			IsMain:   branch.IsMain,
			Projects: generateApplicationBranchProjectObservations(source),
		})
	}

	return v1alpha1.ApplicationObservation{
		Key:         details.Key,
		Name:        details.Name,
		Description: details.Description,
		Visibility:  details.Visibility,
		Tags:        details.Tags,
		Projects:    projects,
		Branches:    branches,
	}
}

// generateApplicationBranchProjectObservations converts the projects of an
// application branch to their observations.
func generateApplicationBranchProjectObservations(projects []sonar.ApplicationProject) []v1alpha1.ApplicationBranchProjectObservation {
	if len(projects) == 0 {
		return nil
	}

	observations := make([]v1alpha1.ApplicationBranchProjectObservation, 0, len(projects))
	for _, project := range projects {
		observations = append(observations, v1alpha1.ApplicationBranchProjectObservation{
			Project: project.Key,
			Branch:  project.Branch,
			IsMain:  project.IsMain,
		})
	}

	return observations
}

// IsApplicationUpToDate returns true when the application spec matches the
// observed state.
func IsApplicationUpToDate(spec *v1alpha1.ApplicationParameters, observation *v1alpha1.ApplicationObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Name != observation.Name {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(spec.Description, observation.Description) {
		return false
	}

	if !helpers.AreStringSlicesEqualDeDuped(spec.Projects, observation.Projects) {
		return false
	}

	return AreApplicationBranchesUpToDate(spec.Branches, observation.Branches)
}

// AreApplicationBranchesUpToDate returns true when every desired branch exists
// with the desired project branches, and no unwanted non-main branch exists.
func AreApplicationBranchesUpToDate(spec []v1alpha1.ApplicationBranchParameters, observation []v1alpha1.ApplicationBranchObservation) bool {
	if len(ApplicationBranchesToDelete(spec, observation)) > 0 {
		return false
	}

	for i := range spec {
		observed, found := FindApplicationBranchObservation(observation, spec[i].Name)
		if !found || !IsApplicationBranchUpToDate(&spec[i], observed) {
			return false
		}
	}

	return true
}

// IsApplicationBranchUpToDate returns true when the observed branch uses
// exactly the project branch selections of the desired branch, with no
// extra project.
func IsApplicationBranchUpToDate(spec *v1alpha1.ApplicationBranchParameters, observation *v1alpha1.ApplicationBranchObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	observed := make(map[string]string, len(observation.Projects))
	for _, project := range observation.Projects {
		observed[project.Project] = project.Branch
	}

	if len(observed) != len(spec.Projects) {
		return false
	}

	for _, selection := range spec.Projects {
		branch, found := observed[selection.Project]
		if !found || branch != selection.Branch {
			return false
		}
	}

	return true
}

// FindApplicationBranchObservation returns the observed application branch
// with the given name, if any.
func FindApplicationBranchObservation(observation []v1alpha1.ApplicationBranchObservation, name string) (*v1alpha1.ApplicationBranchObservation, bool) {
	for i := range observation {
		if observation[i].Name == name {
			return &observation[i], true
		}
	}

	return nil, false
}

// ApplicationBranchesToDelete returns the names of the observed non-main
// application branches that are not in the desired state.
func ApplicationBranchesToDelete(spec []v1alpha1.ApplicationBranchParameters, observation []v1alpha1.ApplicationBranchObservation) []string {
	desired := make(map[string]struct{}, len(spec))
	for _, branch := range spec {
		desired[branch.Name] = struct{}{}
	}

	var toDelete []string

	for _, branch := range observation {
		if branch.IsMain {
			continue
		}

		if _, found := desired[branch.Name]; !found {
			toDelete = append(toDelete, branch.Name)
		}
	}

	return toDelete
}

// ApplicationProjectsToAdd returns the sorted keys of the desired projects
// that are not yet members of the application.
func ApplicationProjectsToAdd(spec, observation []string) []string {
	return helpers.StringSetDifference(spec, observation)
}

// ApplicationProjectsToRemove returns the sorted keys of the member projects
// that are not in the desired state.
func ApplicationProjectsToRemove(spec, observation []string) []string {
	return helpers.StringSetDifference(observation, spec)
}
