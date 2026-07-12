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

package project

import (
	"context"
	"sync"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// updateProjectConcurrently launches all project update operations
// concurrently and collects errors.
func (c *external) updateProjectConcurrently(ctx context.Context, project *v1alpha1.Project, projectKey, projectId string) []error {
	waitGr := sync.WaitGroup{}
	errChan := make(chan error)

	waitGr.Go(func() {
		c.updateVisibility(ctx, project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateTags(ctx, project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateDefaultBranch(ctx, project, projectKey, errChan)
	})

	c.updateBranchNewCodePeriods(ctx, project, projectKey, &waitGr, errChan)

	waitGr.Go(func() {
		c.updateProjectLinks(ctx, project, projectId, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectNewCodePeriod(ctx, project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectQualityGate(ctx, project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectQualityProfiles(ctx, project, projectKey, errChan)
	})

	// Drain error channel concurrently to avoid deadlock on unbuffered sends.
	var allErrors []error

	done := make(chan struct{})

	go func() {
		for err := range errChan {
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}

		close(done)
	}()

	waitGr.Wait()
	close(errChan)
	<-done

	return allErrors
}

// updateVisibility updates the project visibility if it differs from the
// desired state.
func (c *external) updateVisibility(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.Visibility, project.Status.AtProvider.Visibility) {
		return
	}

	resp, err := c.projectsClient.UpdateVisibility(ctx, instance.GenerateProjectUpdateVisibilityOptions(projectKey, ptr.Deref(project.Spec.ForProvider.Visibility, ""))) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project visibility")
	}
}

// updateTags updates the project tags.
func (c *external) updateTags(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if project.Spec.ForProvider.Tags == nil {
		return
	}

	resp, err := c.projectTagsClient.Set(ctx, instance.GenerateProjectTagsSetOptions(projectKey, *project.Spec.ForProvider.Tags)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project tags")
	}
}

// updateDefaultBranch updates the project default branch if it differs from
// the desired state.
func (c *external) updateDefaultBranch(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.DefaultBranch, project.Status.AtProvider.DefaultBranch) {
		return
	}

	resp, err := c.projectBranchesClient.SetMain(ctx, instance.GenerateProjectBranchesSetMainOptions(projectKey, *project.Spec.ForProvider.DefaultBranch)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project default branch")
	}
}

// updateBranchNewCodePeriods updates the new code periods of branches that
// differ from the desired state.
func (c *external) updateBranchNewCodePeriods(ctx context.Context, project *v1alpha1.Project, projectKey string, waitGr *sync.WaitGroup, errChan chan<- error) {
	if instance.AreProjectBranchesUpToDate(project.Spec.ForProvider.Branches, project.Status.AtProvider.Branches) {
		return
	}

	for branch, codePeriodSpec := range project.Spec.ForProvider.Branches {
		waitGr.Add(1)

		go func(branchName string, newCodePeriodSpec *v1alpha1.ProjectNewCodePeriodParameters) {
			defer waitGr.Done()

			branchObservation, branchExists := project.Status.AtProvider.Branches[branchName]
			if branchExists && !instance.IsNewCodePeriodUpToDate(newCodePeriodSpec, &branchObservation.NewCodePeriod) {
				resp, err := c.projectNewCodePeriodsClient.Set(ctx, instance.GenerateBranchNewCodePeriodsSetOptions(projectKey, branchName, newCodePeriodSpec)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot update new code period for branch %s", branchName)
				}
			}
		}(branch, codePeriodSpec)
	}
}

// updateProjectLinks updates the links of the project if they differ from the
// desired state.
func (c *external) updateProjectLinks(ctx context.Context, project *v1alpha1.Project, projectId string, errChan chan<- error) {
	if instance.AreProjectLinksUpToDate(project.Spec.ForProvider.Links, project.Status.AtProvider.Links) {
		return
	}

	linksUpdateWaitGroup := sync.WaitGroup{}
	linksUpdateWaitGroup.Go(func() {
		c.deleteNonSpecLinks(ctx, project, errChan)
	})

	for _, link := range project.Spec.ForProvider.Links {
		linkObservation, linkExists := project.Status.AtProvider.Links[link.Name]
		linkUpToDate := linkExists && instance.IsProjectLinkUpToDate(link, &linkObservation)

		linksUpdateWaitGroup.Add(1)

		go func(linkId string, linkSpec v1alpha1.ProjectLinkParameters, deleteLink bool, createLink bool) {
			defer linksUpdateWaitGroup.Done()

			if deleteLink {
				resp, err := c.projectLinksClient.Delete(ctx, instance.GenerateProjectLinksDeleteOptions(linkId)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot delete Project link %s", linkSpec.Name)
				}
			}

			if createLink {
				_, resp, err := c.projectLinksClient.Create(ctx, instance.GenerateProjectLinksCreateOptions(projectId, linkSpec)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot create Project link %s", linkSpec.Name)
				}
			}
		}(linkObservation.ID, link, linkExists && !linkUpToDate, !linkUpToDate)
	}

	linksUpdateWaitGroup.Wait()
}

// deleteNonSpecLinks deletes the project links that exist in the observed state
// but not in the desired state, or that differ between the observed
// and desired state.
func (c *external) deleteNonSpecLinks(ctx context.Context, project *v1alpha1.Project, errChan chan<- error) {
	deleteWaitGroup := sync.WaitGroup{}
	// Delete links that exist in the observed state but not in the desired state, or that differ between the observed and desired state.
	for linkName, linkObservation := range project.Status.AtProvider.Links {
		if !c.isProjectLinkInSpec(project.Spec.ForProvider.Links, linkName) {
			deleteWaitGroup.Add(1)

			go func(linkName string, linkId string) {
				defer deleteWaitGroup.Done()

				resp, err := c.projectLinksClient.Delete(ctx, instance.GenerateProjectLinksDeleteOptions(linkId)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot delete Project link %s", linkName)
				}
			}(linkName, linkObservation.ID)
		}
	}

	deleteWaitGroup.Wait()
}

// isProjectLinkInSpec checks if a link with the given name exists in the spec.
func (c *external) isProjectLinkInSpec(spec []v1alpha1.ProjectLinkParameters, linkName string) bool {
	for _, linSpec := range spec {
		if linSpec.Name == linkName {
			return true
		}
	}

	return false
}

// updateProjectNewCodePeriod updates the project-level new code period
// if it differs from the desired state.
func (c *external) updateProjectNewCodePeriod(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if instance.IsNewCodePeriodUpToDate(project.Spec.ForProvider.NewCodePeriod, &project.Status.AtProvider.NewCodePeriod) {
		return
	}

	resp, err := c.projectNewCodePeriodsClient.Set(ctx, instance.GenerateProjectNewCodePeriodsSetOptions(projectKey, project.Spec.ForProvider.NewCodePeriod)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project new code period")
	}
}

// updateProjectQualityGate updates the quality gate of the project
// if it differs from the desired state.
func (c *external) updateProjectQualityGate(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.QualityGateName, project.Status.AtProvider.QualityGateName) {
		return
	}

	resp, err := c.qualityGatesClient.Assign(ctx, instance.GenerateQualityGateSelectOptions(projectKey, *project.Spec.ForProvider.QualityGateName)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project quality gate")
	}
}

// updateProjectQualityProfiles updates the quality profiles of the project
// if they differ from the desired state.
func (c *external) updateProjectQualityProfiles(ctx context.Context, project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if instance.AreProjectQualityProfilesUpToDate(project.Spec.ForProvider.QualityProfiles, project.Status.AtProvider.QualityProfiles) {
		return
	}

	for language, qualityProfile := range project.Spec.ForProvider.QualityProfiles {
		qualityProfileObservation, qualityProfileExists := project.Status.AtProvider.QualityProfiles[language]
		if qualityProfileExists && helpers.IsComparablePtrEqualComparable(qualityProfile.Id, qualityProfileObservation.Id) {
			continue
		}

		// Guard against a nil or unresolved Id (reference not yet resolved).
		// This can occur on the first reconcile after resource creation, before
		// the resolved external-name has been written back to the spec.
		if qualityProfile.Id == nil || *qualityProfile.Id == "" {
			errChan <- errors.Errorf("quality profile ID for language %s is not yet resolved, skipping update", language)

			continue
		}

		retrievedQualityProfile, resp, err := c.qualityProfilesClient.Show(ctx, instance.GenerateQualityProfileShowOptions(*qualityProfile.Id)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			errChan <- errors.Wrapf(err, "cannot retrieve quality profile for language %s", language)

			continue
		}

		resp, err = c.qualityProfilesClient.AddProject(ctx, instance.GenerateQualityProfileAddProjectOptions(projectKey, retrievedQualityProfile.Profile.Name, language)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			errChan <- errors.Wrapf(err, "cannot update Project quality profile for language %s", language)
		}
	}
}
