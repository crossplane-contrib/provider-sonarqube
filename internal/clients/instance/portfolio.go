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

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// selectionModeNone is the default portfolio selection mode.
	selectionModeNone = "NONE"
)

// PortfoliosClient is the interface for interacting with the
// SonarQube Portfolio (Views) API.
type PortfoliosClient interface {
	AddApplication(opt *sonar.ViewsAddApplicationOptions) (*http.Response, error)
	AddApplicationBranch(opt *sonar.ViewsAddApplicationBranchOptions) (*http.Response, error)
	AddPortfolio(opt *sonar.ViewsAddPortfolioOptions) (*http.Response, error)
	AddProject(opt *sonar.ViewsAddProjectOptions) (*http.Response, error)
	AddProjectBranch(opt *sonar.ViewsAddProjectBranchOptions) (*http.Response, error)
	Applications(opt *sonar.ViewsApplicationsOptions) (*sonar.ViewsApplications, *http.Response, error)
	Create(opt *sonar.ViewsCreateOptions) (*http.Response, error)
	Delete(opt *sonar.ViewsDeleteOptions) (*http.Response, error)
	List() (*sonar.ViewsList, *http.Response, error)
	Move(opt *sonar.ViewsMoveOptions) (*http.Response, error)
	MoveOptions(opt *sonar.ViewsMoveOptionsOptions) (*sonar.ViewsMoveDestinations, *http.Response, error)
	Projects(opt *sonar.ViewsProjectsOptions) (*sonar.ViewsProjects, *http.Response, error)
	ProjectsStatus(opt *sonar.ViewsProjectsStatusOptions) (*sonar.ViewsProjectsStatus, *http.Response, error)
	Refresh(opt *sonar.ViewsRefreshOptions) (*http.Response, error)
	RemoveApplication(opt *sonar.ViewsRemoveApplicationOptions) (*http.Response, error)
	RemoveApplicationBranch(opt *sonar.ViewsRemoveApplicationBranchOptions) (*http.Response, error)
	RemovePortfolio(opt *sonar.ViewsRemovePortfolioOptions) (*http.Response, error)
	RemoveProject(opt *sonar.ViewsRemoveProjectOptions) (*http.Response, error)
	RemoveProjectBranch(opt *sonar.ViewsRemoveProjectBranchOptions) (*http.Response, error)
	Search(opt *sonar.ViewsSearchOptions) (*sonar.ViewsSearch, *http.Response, error)
	SetManualMode(opt *sonar.ViewsSetManualModeOptions) (*http.Response, error)
	SetNoneMode(opt *sonar.ViewsSetNoneModeOptions) (*http.Response, error)
	SetRegexpMode(opt *sonar.ViewsSetRegexpModeOptions) (*http.Response, error)
	SetRemainingProjectsMode(opt *sonar.ViewsSetRemainingProjectsModeOptions) (*http.Response, error)
	SetTagsMode(opt *sonar.ViewsSetTagsModeOptions) (*http.Response, error)
	Show(opt *sonar.ViewsShowOptions) (*sonar.ViewsShow, *http.Response, error)
	SubPortfolios(opt *sonar.ViewsSubViewsOptions) (*sonar.ViewsSubViews, *http.Response, error)
	Update(opt *sonar.ViewsUpdateOptions) (*http.Response, error)
}

// NewPortfoliosClient creates a new PortfoliosClient using the
// provided SonarQube client configuration.
func NewPortfoliosClient(clientConfig common.Config) PortfoliosClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Views
}

// GeneratePortfolioObservation converts a sonar.ViewDetails to a
// PortfolioObservation.
func GeneratePortfolioObservation(details *sonar.ViewDetails) v1alpha1.PortfolioObservation {
	if details == nil {
		return v1alpha1.PortfolioObservation{}
	}

	return v1alpha1.PortfolioObservation{
		Key:           details.Key,
		Name:          details.Name,
		Description:   details.Description,
		Qualifier:     details.Qualifier,
		Visibility:    details.Visibility,
		SelectionMode: details.SelectionMode,
	}
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

	return normalizeSelectionMode(spec.SelectionMode) == normalizeSelectionMode(observation.SelectionMode)
}
