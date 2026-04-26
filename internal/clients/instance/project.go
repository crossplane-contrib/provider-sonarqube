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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// ProjectsClient is the interface for interacting with SonarQube Projects API
// It handles all the operations related to Projects in SonarQube, such as
// creating, updating, deleting, and retrieving Projects.
type ProjectsClient interface {
	BulkDelete(opt *sonar.ProjectsBulkDeleteOptions) (*http.Response, error)
	Create(opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error)
	Delete(opt *sonar.ProjectsDeleteOptions) (*http.Response, error)
	Search(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error)
	SearchMyProjects(opt *sonar.ProjectsSearchMyProjectsOptions) (*sonar.ProjectsSearchMyProjects, *http.Response, error)
	SearchMyScannableProjects(opt *sonar.ProjectsSearchMyScannableProjectsOptions) (*sonar.ProjectsSearchMyScannableProjects, *http.Response, error)
	UpdateKey(opt *sonar.ProjectsUpdateKeyOptions) (*http.Response, error)
	UpdateVisibility(opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error)
}

// NewProjectsClient creates a new ProjectsClient with the provided
// SonarQube client configuration.
func NewProjectsClient(clientConfig common.Config) ProjectsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Projects
}

// GenerateProjectsCreateOptions generates the options for creating a
// SonarQube Project based on the provided ProjectParameters.
func GenerateProjectsCreateOptions(spec v1alpha1.ProjectParameters) *sonar.ProjectsCreateOptions {
	opts := sonar.ProjectsCreateOptions{
		Name:    spec.Name,
		Project: spec.Key,
	}

	helpers.AssignIfNonNil(&opts.Visibility, spec.Visibility)
	helpers.AssignIfNonNil(&opts.MainBranch, spec.DefaultBranch)

	if spec.NewCodePeriod != nil {
		helpers.AssignIfNonNil(&opts.NewCodeDefinitionType, &spec.NewCodePeriod.Type)
		helpers.AssignIfNonNil(&opts.NewCodeDefinitionValue, spec.NewCodePeriod.Value)
	}

	return &opts
}

// GenerateProjectDeleteOptions generates the options for deleting a
// SonarQube Project based on the provided ProjectParameters.
func GenerateProjectDeleteOptions(projectKey string) *sonar.ProjectsDeleteOptions {
	opts := sonar.ProjectsDeleteOptions{
		Project: projectKey,
	}

	return &opts
}

// GenerateProjectSearchOptions generates the options for searching a
// SonarQube Project based on the provided ProjectParameters.
func GenerateProjectSearchOptions(projectKey string) *sonar.ProjectsSearchOptions {
	opts := sonar.ProjectsSearchOptions{
		Projects: []string{projectKey},
	}

	return &opts
}

// UpdateProjectAttributesObservation updates the ProjectObservation
// with the attributes of the given SonarQube ProjectSearchComponent.
func UpdateProjectAttributesObservation(observation *v1alpha1.ProjectObservation, project *sonar.ProjectSearchComponent) {
	observation.Key = project.Key
	observation.Name = project.Name
	observation.Qualifier = project.Qualifier
	observation.Visibility = project.Visibility
	observation.LastAnalysisDate = helpers.StringToMetaTime(&project.LastAnalysisDate)
	observation.Revision = project.Revision
	observation.Uuid = project.ProjectUuid
	observation.Managed = project.Managed
}

// LateInitializeProject performs late initialization of the ProjectParameters
// in the ProjectParameters based on the observed project from SonarQube.
// It updates the ProjectParameters with any missing information from the
// observed project.
func LateInitializeProject(spec *v1alpha1.ProjectParameters, observation *v1alpha1.ProjectObservation) {
	helpers.AssignIfNil(&spec.Visibility, observation.Visibility)
	LateInitializeProjectLinks(spec, observation.Links)
	LateInitializeProjectNewCodePeriod(spec.NewCodePeriod, &observation.NewCodePeriod)
}

// IsProjectLateInitialized checks if the ProjectParameters has been
// late-initialized based on the observed project from SonarQube.
func IsProjectLateInitialized(former, current *v1alpha1.ProjectParameters) bool {
	return !cmp.Equal(former.Visibility, current.Visibility) ||
		!cmp.Equal(former.Links, current.Links, cmpopts.EquateEmpty()) ||
		!cmp.Equal(former.NewCodePeriod, current.NewCodePeriod)
}

// IsProjectUpToDate checks if the observed state of a SonarQube Project is up
// to date with the desired state specified in the ProjectParameters.
func IsProjectUpToDate(spec *v1alpha1.ProjectParameters, observation *v1alpha1.ProjectObservation) bool { //nolint:gocyclo,cyclop // This function is complex due to the number of fields being compared, but it is necessary for determining if the project is up to date.
	if observation == nil {
		return false
	}

	return spec == nil ||
		spec.Key == observation.Key &&
			spec.Name == observation.Name &&
			helpers.IsComparablePtrEqualComparable(spec.Visibility, observation.Visibility) &&
			helpers.IsComparablePtrEqualComparable(spec.QualityGateName, observation.QualityGateName) &&
			AreProjectLinksUpToDate(spec.Links, observation.Links) &&
			AreProjectBranchesUpToDate(spec.Branches, observation.Branches) &&
			IsProjectMainBranchUpToDate(observation.Branches, ptr.Deref(spec.DefaultBranch, "main")) &&
			IsNewCodePeriodUpToDate(spec.NewCodePeriod, &observation.NewCodePeriod) &&
			AreProjectQualityProfilesUpToDate(spec.QualityProfiles, observation.QualityProfiles)
}

// GenerateProjectUpdateVisibilityOptions generates the options for updating the
// visibility of a SonarQube Project based on the provided project key and
// desired visibility.
func GenerateProjectUpdateVisibilityOptions(projectKey, visibility string) *sonar.ProjectsUpdateVisibilityOptions {
	return &sonar.ProjectsUpdateVisibilityOptions{
		Project:    projectKey,
		Visibility: visibility,
	}
}
