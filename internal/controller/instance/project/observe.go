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

// Package project handles project observation and management.
package project

import (
	"context"
	"sync"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// observeProjectExistence checks if the project with the given key exists
// in SonarQube.
// It populates the ProjectObservation with the project details if it exists.
func (c *external) observeProjectExistence(ctx context.Context, projectKey string, observation *v1alpha1.ProjectObservation) (bool, error) {
	projects, resp, err := c.projectsClient.Search(ctx, instance.GenerateProjectSearchOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil || projects == nil || len(projects.Components) == 0 {
		return false, errors.Wrap(err, "cannot search for Project")
	}

	var project *sonar.ProjectSearchComponent

	for _, searchProject := range projects.Components {
		if searchProject.Key == projectKey {
			project = &searchProject

			break
		}
	}

	if project == nil {
		return false, nil
	}

	instance.UpdateProjectAttributesObservation(observation, project)

	return true, nil
}

// observeProjectDetails makes all observation API calls concurrently and
// returns the results.
func (c *external) observeProjectDetails(ctx context.Context, projectKey, projectId string) observeResult {
	var (
		result observeResult
		mutex  sync.Mutex
		waitGr sync.WaitGroup
	)

	result.branchNewCodePeriods = make(map[string]v1alpha1.ProjectNewCodePeriodObservation)

	waitGr.Go(func() {
		c.observeBranches(ctx, projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeLinks(ctx, projectId, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeNewCodePeriods(ctx, projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeQualityGate(ctx, projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeQualityProfiles(ctx, projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeALMBinding(ctx, projectKey, &result, &mutex)
	})

	waitGr.Wait()

	return result
}

// observeBranches retrieves the project branches from SonarQube and
// populates the result.
func (c *external) observeBranches(ctx context.Context, projectKey string, result *observeResult, mutex *sync.Mutex) {
	branches, resp, branchErr := c.projectBranchesClient.List(ctx, instance.GenerateProjectBranchesListOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if branchErr != nil {
		result.errors = append(result.errors, errors.Wrap(branchErr, "cannot observe Project branches"))

		return
	}

	result.branches = instance.GenerateBranchesObservations(*branches)

	for branchName, branchObs := range result.branches {
		if branchObs.IsMain {
			result.mainBranch = branchName

			break
		}
	}
}

// observeLinks retrieves the project links from SonarQube and
// populates the result.
func (c *external) observeLinks(ctx context.Context, projectId string, result *observeResult, mutex *sync.Mutex) {
	links, resp, linkErr := c.projectLinksClient.Search(ctx, instance.GenerateProjectLinksSearchOptions(projectId)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if linkErr != nil {
		result.errors = append(result.errors, errors.Wrap(linkErr, "cannot observe Project links"))

		return
	}

	result.links = instance.GenerateProjectLinksObservations(*links)
}

// observeNewCodePeriods observes the new code periods of the project
// and populates the result.
func (c *external) observeNewCodePeriods(ctx context.Context, projectKey string, result *observeResult, mutex *sync.Mutex) {
	newCodePeriod, resp, ncErr := c.projectNewCodePeriodsClient.Show(ctx, instance.GenerateNewCodePeriodsShowOptions(&projectKey, nil)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	mutex.Lock()

	if ncErr != nil {
		result.errors = append(result.errors, errors.Wrap(ncErr, "cannot show Project new code period"))
	} else {
		result.projectNewCodePeriod = instance.GenerateProjectNewCodePeriodObservation(newCodePeriod)
	}

	mutex.Unlock()

	newCodePeriodsList, ncListResp, ncListErr := c.projectNewCodePeriodsClient.List(ctx, instance.GenerateProjectNewCodePeriodsListOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(ncListResp)

	mutex.Lock()
	defer mutex.Unlock()

	if ncListErr != nil {
		result.errors = append(result.errors, errors.Wrap(ncListErr, "cannot list Project new code periods"))

		return
	}

	for _, ncp := range newCodePeriodsList.NewCodePeriods {
		result.branchNewCodePeriods[ncp.BranchKey] = instance.GenerateBranchNewCodePeriodObservation(&ncp)
	}
}

// observeQualityGate retrieves the quality gate associated with
// the project from SonarQube.
func (c *external) observeQualityGate(ctx context.Context, projectKey string, result *observeResult, mutex *sync.Mutex) {
	qualityGate, resp, qgErr := c.qualityGatesClient.GetByProject(ctx, instance.GenerateQualityGateGetByProjectOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if qgErr != nil {
		result.errors = append(result.errors, errors.Wrap(qgErr, "cannot observe Project quality gate"))

		return
	}

	result.qualityGateName = qualityGate.QualityGate.Name
}

// observeQualityProfiles retrieves the quality profiles associated
// with the project from SonarQube.
func (c *external) observeQualityProfiles(ctx context.Context, projectKey string, result *observeResult, mutex *sync.Mutex) {
	qualityProfiles, resp, qpErr := c.qualityProfilesClient.Search(ctx, instance.GenerateQualityProfilesSearchProjectOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if qpErr != nil {
		result.errors = append(result.errors, errors.Wrap(qpErr, "cannot observe Project quality profile"))

		return
	}

	result.qualityProfiles = instance.GenerateQualityProfilesSearchProjectObservation(qualityProfiles)
}

// observeALMBinding retrieves the ALM binding associated with the project
// from SonarQube, if any.
func (c *external) observeALMBinding(ctx context.Context, projectKey string, result *observeResult, mutex *sync.Mutex) {
	binding, resp, bindingErr := c.almSettingsClient.GetBinding(ctx, instance.GenerateProjectALMBindingGetOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if bindingErr != nil {
		if common.IsResponseNotFound(resp) {
			return
		}

		result.errors = append(result.errors, errors.Wrap(bindingErr, "cannot observe Project ALM binding"))

		return
	}

	result.almBinding = instance.GenerateProjectALMBindingObservation(binding)
}

// applyObserveResult applies the concurrent observation results
// to the ProjectObservation.
func (c *external) applyObserveResult(observation *v1alpha1.ProjectObservation, result *observeResult) {
	if result.branches != nil {
		observation.Branches = result.branches
		observation.DefaultBranch = result.mainBranch
	} else if observation.Branches == nil {
		observation.Branches = make(map[string]v1alpha1.ProjectBranchObservation)
	}

	if result.links != nil {
		observation.Links = result.links
	} else if observation.Links == nil {
		observation.Links = make(map[string]v1alpha1.ProjectLinkObservation)
	}

	observation.NewCodePeriod = result.projectNewCodePeriod

	for branchKey, ncpObs := range result.branchNewCodePeriods {
		if branchObs, exists := observation.Branches[branchKey]; exists {
			branchObs.NewCodePeriod = ncpObs
			observation.Branches[branchKey] = branchObs
		}
	}

	observation.QualityGateName = result.qualityGateName

	if result.qualityProfiles != nil {
		observation.QualityProfiles = result.qualityProfiles
	}

	observation.AlmBinding = result.almBinding
}
