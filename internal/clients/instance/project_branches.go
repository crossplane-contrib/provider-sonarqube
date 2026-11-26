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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// ProjectBranchesClient is the interface for interacting with
// SonarQube Project Branches API.
type ProjectBranchesClient interface {
	Delete(ctx context.Context, opt *sonar.ProjectBranchesDeleteOptions) (*http.Response, error)
	List(ctx context.Context, opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error)
	Rename(ctx context.Context, opt *sonar.ProjectBranchesRenameOptions) (*http.Response, error)
	SetAutomaticDeletionProtection(ctx context.Context, opt *sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions) (*http.Response, error)
	SetMain(ctx context.Context, opt *sonar.ProjectBranchesSetMainOptions) (*http.Response, error)
}

// NewProjectBranchesClient creates a new ProjectBranchesClient with
// the provided SonarQube client configuration.
func NewProjectBranchesClient(clientConfig common.Config) ProjectBranchesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.ProjectBranches
}

// GenerateProjectBranchesListOptions generates the options for listing branches
// of a SonarQube Project based on the provided project key.
func GenerateProjectBranchesListOptions(projectKey string) *sonar.ProjectBranchesListOptions {
	return &sonar.ProjectBranchesListOptions{
		Project: projectKey,
	}
}

// GenerateProjectBranchesDeleteOptions generates the options for deleting a
// branch of a SonarQube Project based on the provided project key
// and branch key.
func GenerateProjectBranchesDeleteOptions(projectKey, branchKey string) *sonar.ProjectBranchesDeleteOptions {
	return &sonar.ProjectBranchesDeleteOptions{
		Project: projectKey,
		Branch:  branchKey,
	}
}

// GenerateBranchesObservations generates the observations for the branches of
// a SonarQube Project based on the provided list of branches.
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

// AreProjectBranchesUpToDate checks if the observed branches of a
// SonarQube Project are up to date with the branches specified in the managed
// resource.
// Only the branches present in both the spec and the observation are compared:
// a branch of the spec that does not exist yet cannot be configured.
func AreProjectBranchesUpToDate(spec map[string]*v1alpha1.ProjectBranchParameters, observation map[string]v1alpha1.ProjectBranchObservation) bool {
	for branchName, branchObservation := range observation {
		branchSpec := spec[branchName]
		if !IsBranchNewCodePeriodUpToDate(branchSpec, &branchObservation) ||
			!IsBranchPurgeProtectionUpToDate(branchSpec, &branchObservation) {
			return false
		}
	}

	return true
}

// IsBranchNewCodePeriodUpToDate checks if the observed new code period of a
// branch matches the one specified for it, if any.
func IsBranchNewCodePeriodUpToDate(spec *v1alpha1.ProjectBranchParameters, observation *v1alpha1.ProjectBranchObservation) bool {
	if observation == nil {
		return false
	}

	return IsNewCodePeriodUpToDate(GenerateBranchNewCodePeriodParameters(spec), &observation.NewCodePeriod)
}

// IsBranchPurgeProtectionUpToDate checks if the observed automatic deletion
// protection of a branch matches the one specified for it, if any.
//
// SonarQube does not allow removing the protection of the main branch, so such
// a request is considered up to date: Update could never converge on it.
func IsBranchPurgeProtectionUpToDate(spec *v1alpha1.ProjectBranchParameters, observation *v1alpha1.ProjectBranchObservation) bool {
	if observation == nil {
		return false
	}

	if spec == nil || spec.ExcludedFromPurge == nil {
		return true
	}

	if observation.IsMain && !*spec.ExcludedFromPurge {
		return true
	}

	return *spec.ExcludedFromPurge == observation.ExcludedFromPurge
}

// GenerateBranchNewCodePeriodParameters extracts the new code definition from
// the parameters of a branch. It returns nil when the new code definition of
// the branch is not managed.
func GenerateBranchNewCodePeriodParameters(spec *v1alpha1.ProjectBranchParameters) *v1alpha1.ProjectNewCodePeriodParameters {
	if spec == nil || spec.Type == nil {
		return nil
	}

	return &v1alpha1.ProjectNewCodePeriodParameters{
		Type:  *spec.Type,
		Value: spec.Value,
	}
}

// GenerateProjectBranchesSetAutomaticDeletionProtectionOptions generates the
// options for protecting a branch of a SonarQube Project from automatic
// deletion, or for removing that protection.
func GenerateProjectBranchesSetAutomaticDeletionProtectionOptions(projectKey, branchName string, protected bool) *sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions {
	return &sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions{
		Project: projectKey,
		Branch:  branchName,
		Value:   protected,
	}
}

// IsProjectMainBranchUpToDate checks if the main branch of a SonarQube Project
// is up to date with the main branch specified in the managed resource.
func IsProjectMainBranchUpToDate(observedBranches map[string]v1alpha1.ProjectBranchObservation, mainBranchName string) bool {
	for branchName, branchObservation := range observedBranches {
		if branchObservation.IsMain && branchName != mainBranchName {
			return false
		}
	}

	return true
}

// GenerateProjectBranchesSetMainOptions generates the options for setting
// the main branch of a SonarQube Project based on the provided
// project key and main branch name.
func GenerateProjectBranchesSetMainOptions(projectKey, mainBranchName string) *sonar.ProjectBranchesSetMainOptions {
	return &sonar.ProjectBranchesSetMainOptions{
		Project: projectKey,
		Branch:  mainBranchName,
	}
}
