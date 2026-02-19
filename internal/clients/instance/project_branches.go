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

// ProjectBranchesClient is the interface for interacting with SonarQube Project Branches API.
type ProjectBranchesClient interface {
	Delete(opt *sonar.ProjectBranchesDeleteOption) (*http.Response, error)
	List(opt *sonar.ProjectBranchesListOption) (*sonar.ProjectBranchesList, *http.Response, error)
	Rename(opt *sonar.ProjectBranchesRenameOption) (*http.Response, error)
	SetAutomaticDeletionProtection(opt *sonar.ProjectBranchesSetAutomaticDeletionProtectionOption) (*http.Response, error)
	SetMain(opt *sonar.ProjectBranchesSetMainOption) (*http.Response, error)
}

// NewProjectBranchesClient creates a new ProjectBranchesClient with the provided SonarQube client configuration.
func NewProjectBranchesClient(clientConfig common.Config) ProjectBranchesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.ProjectBranches
}

// GenerateProjectBranchesListOptions generates the options for listing branches of a SonarQube Project based on the provided project key.
func GenerateProjectBranchesListOptions(projectKey string) *sonar.ProjectBranchesListOption {
	return &sonar.ProjectBranchesListOption{
		Project: projectKey,
	}
}

// GenerateProjectBranchesDeleteOptions generates the options for deleting a branch of a SonarQube Project based on the provided project key and branch key.
func GenerateProjectBranchesDeleteOptions(projectKey string, branchKey string) *sonar.ProjectBranchesDeleteOption {
	return &sonar.ProjectBranchesDeleteOption{
		Project: projectKey,
		Branch:  branchKey,
	}
}

// GenerateBranchesObservations generates the observations for the branches of a SonarQube Project based on the provided list of branches.
func GenerateBranchesObservations(branches sonar.ProjectBranchesList) map[string]v1alpha1.ProjectBranchObservation {
	observations := make(map[string]v1alpha1.ProjectBranchObservation)
	for _, branch := range branches.Branches {
		observations[branch.Name] = v1alpha1.ProjectBranchObservation{
			AnalysisDate:      helpers.StringToMetaTime(&branch.AnalysisDate),
			ID:                branch.BranchID,
			ExcludedFromPurge: branch.ExcludedFromPurge,
			IsMain:            branch.IsMain,
			Name:              branch.Name,
			Status:            v1alpha1.ProjectBranchStatusObservation{QualityGateStatus: branch.Status.QualityGateStatus},
			Type:              branch.Type,
		}
	}

	return observations
}

// AreProjectBranchesUpToDate checks if the observed branches of a SonarQube Project are up to date with the authorized branches specified in the managed resource.
// Only confirms that each branch new code definition matches the branch status in the observation for each branch specified in the spec.
func AreProjectBranchesUpToDate(spec map[string]*v1alpha1.ProjectNewCodePeriodParameters, observation map[string]v1alpha1.ProjectBranchObservation) bool {
	for branchName, branchObservation := range observation {
		// Check if the branch exists in the spec and if its new code period is up to date with the observation. If the branch does not exist in the spec, we consider it up to date since we are only concerned with branches specified in the spec.
		branchSpec, ok := spec[branchName]
		if ok && !IsNewCodePeriodUpToDate(branchSpec, &branchObservation.NewCodePeriod) {
			return false
		}
	}

	return true
}

// IsProjectMainBranchUpToDate checks if the main branch of a SonarQube Project is up to date with the main branch specified in the managed resource.
func IsProjectMainBranchUpToDate(observedBranches map[string]v1alpha1.ProjectBranchObservation, mainBranchName string) bool {
	for branchName, branchObservation := range observedBranches {
		if branchObservation.IsMain && branchName != mainBranchName {
			return false
		}
	}

	return true
}

// GenerateProjectBranchesSetMainOptions generates the options for setting the main branch of a SonarQube Project based on the provided project key and main branch name.
func GenerateProjectBranchesSetMainOptions(projectKey string, mainBranchName string) *sonar.ProjectBranchesSetMainOption {
	return &sonar.ProjectBranchesSetMainOption{
		Project: projectKey,
		Branch:  mainBranchName,
	}
}
