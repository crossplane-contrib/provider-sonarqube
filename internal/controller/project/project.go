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

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/utils/ptr"

	stderrors "errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	errNotProject   = "managed resource is not a Project custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
)

// SetupGated adds a controller that reconciles Project managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Project controller"))
		}
	}, v1alpha1.ProjectGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error { //nolint:varnamelen // consistent with other controllers
	name := managed.ControllerName(v1alpha1.ProjectGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.ProjectList{}, o.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ProjectList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ProjectGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Project{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	qualityGate, isValid := managedResource.(*v1alpha1.QualityGate)
	if !isValid {
		return nil, errors.New(errNotProject)
	}

	err := c.usage.Track(ctx, qualityGate)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{
		projectsClient:              instance.NewProjectsClient(*config),
		projectLinksClient:          instance.NewProjectLinksClient(*config),
		projectBranchesClient:       instance.NewProjectBranchesClient(*config),
		projectNewCodePeriodsClient: instance.NewNewCodePeriodsClient(*config),
		qualityGatesClient:          instance.NewQualityGatesClient(*config),
		qualityProfilesClient:       instance.NewQualityProfilesClient(*config),
		projectTagsClient:           instance.NewProjectTagsClient(*config),
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// projectsClient is used to interact with SonarQube Projects API
	projectsClient instance.ProjectsClient
	// projectLinksClient is used to interact with SonarQube Project Links API
	projectLinksClient instance.ProjectLinksClient
	// projectBranchesClient is used to interact with SonarQube Project Branches API
	projectBranchesClient instance.ProjectBranchesClient
	// projectNewCodePeriodsClient is used to interact with SonarQube Project New Code Periods API
	projectNewCodePeriodsClient instance.NewCodePeriodsClient
	// qualityGatesClient is used to interact with SonarQube Quality Gates API
	qualityGatesClient instance.QualityGatesClient
	// qualityProfilesClient is used to interact with SonarQube Quality Profiles API
	qualityProfilesClient instance.QualityProfilesClient
	// projectTagsClient is used to interact with SonarQube Project Tags API
	projectTagsClient instance.ProjectTagsClient
}

// observeResult holds the results of all concurrent observation API calls.
type observeResult struct {
	branches             map[string]v1alpha1.ProjectBranchObservation
	mainBranch           string
	links                map[string]v1alpha1.ProjectLinkObservation
	projectNewCodePeriod v1alpha1.ProjectNewCodePeriodObservation
	branchNewCodePeriods map[string]v1alpha1.ProjectNewCodePeriodObservation
	qualityGateName      string
	qualityProfiles      map[string]v1alpha1.ProjectQualityProfileObservation
	errors               []error
}

// Observe observes the external resource and updates the managed resource's status with the observed state of the external resource.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	externalName := meta.GetExternalName(project)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Check if the Project exists in SonarQube and update the observation with the project details if it exists.
	exists, err := c.observeProjectExistence(externalName, &project.Status.AtProvider)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot observe Project existence")
	}

	if !exists {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Make all observation API calls concurrently, collecting results safely.
	result := c.observeProjectDetails(externalName)

	// Safely populate observation from collected results (single-threaded)
	c.applyObserveResult(&project.Status.AtProvider, &result)

	if len(result.errors) > 0 {
		return managed.ExternalObservation{
			ResourceExists: true,
		}, stderrors.Join(result.errors...)
	}

	// Late initialize the ProjectParameters with any missing information from the observed project.
	current := project.Spec.ForProvider.DeepCopy()
	instance.LateInitializeProject(&project.Spec.ForProvider, &project.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsProjectUpToDate(&project.Spec.ForProvider, &project.Status.AtProvider),
		ResourceLateInitialized: cmp.Equal(current, &project.Status.AtProvider, cmpopts.EquateEmpty()),
	}, nil
}

// Create creates the project in SonarQube using the SonarQube API client.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	project.SetConditions(xpv1.Creating())

	// Create the SonarQube Project using the SonarQube API client
	createdProject, resp, err := c.projectsClient.Create(instance.GenerateProjectsCreateOptions(project.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create Project")
	}

	meta.SetExternalName(project, createdProject.Project.Key)

	// After creating the project, we need to update the links, branches and new code periods of the project to match the desired state in the spec.
	// We will perform these updates in the Update method since the Create method is only responsible for creating the project and not updating it.

	return managed.ExternalCreation{}, nil
}

// Update will update the external resource if it exists and is not up to date.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	projectKey := meta.GetExternalName(project)
	if projectKey == "" {
		return managed.ExternalUpdate{}, errors.New("external name is not set for the Project")
	}

	allErrors := c.updateProjectConcurrently(project, projectKey)

	if len(allErrors) > 0 {
		return managed.ExternalUpdate{}, stderrors.Join(allErrors...)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Delete will delete the external resource if it exists.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	project, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProject)
	}

	externalName := meta.GetExternalName(project)
	if externalName == "" {
		// External name is not set, so we cannot delete the resource in SonarQube.
		// However, we can consider the resource deleted since it does not exist in SonarQube.
		return managed.ExternalDelete{}, nil
	}

	project.SetConditions(xpv1.Deleting())

	// Delete the SonarQube Project using the SonarQube API client
	resp, err := c.projectsClient.Delete(instance.GenerateProjectDeleteOptions(externalName)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete Project")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Project external client.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// observeProjectExistence checks if the project with the given key exists in SonarQube.
// It populates the ProjectObservation with the project details if it exists.
func (c *external) observeProjectExistence(projectKey string, observation *v1alpha1.ProjectObservation) (bool, error) {
	projects, resp, err := c.projectsClient.Search(instance.GenerateProjectSearchOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
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

// observeProjectDetails makes all observation API calls concurrently and returns the results.
func (c *external) observeProjectDetails(projectKey string) observeResult {
	var (
		result observeResult
		mutex  sync.Mutex
		waitGr sync.WaitGroup
	)

	result.branchNewCodePeriods = make(map[string]v1alpha1.ProjectNewCodePeriodObservation)

	waitGr.Go(func() {
		c.observeBranches(projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeLinks(projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeNewCodePeriods(projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeQualityGate(projectKey, &result, &mutex)
	})

	waitGr.Go(func() {
		c.observeQualityProfiles(projectKey, &result, &mutex)
	})

	waitGr.Wait()

	return result
}

// observeBranches retrieves the project branches from SonarQube and populates the result.
func (c *external) observeBranches(projectKey string, result *observeResult, mutex *sync.Mutex) {
	branches, resp, branchErr := c.projectBranchesClient.List(instance.GenerateProjectBranchesListOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
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

// observeLinks retrieves the project links from SonarQube and populates the result.
func (c *external) observeLinks(projectKey string, result *observeResult, mutex *sync.Mutex) {
	links, resp, linkErr := c.projectLinksClient.Search(instance.GenerateProjectLinksSearchOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if linkErr != nil {
		result.errors = append(result.errors, errors.Wrap(linkErr, "cannot observe Project links"))

		return
	}

	result.links = instance.GenerateProjectLinksObservations(*links)
}

// observeNewCodePeriods observes the new code periods of the project and populates the result.
func (c *external) observeNewCodePeriods(projectKey string, result *observeResult, mutex *sync.Mutex) {
	newCodePeriod, resp, ncErr := c.projectNewCodePeriodsClient.Show(instance.GenerateNewCodePeriodsShowOptions(&projectKey, nil)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	mutex.Lock()

	if ncErr != nil {
		result.errors = append(result.errors, errors.Wrap(ncErr, "cannot show Project new code period"))
	} else {
		result.projectNewCodePeriod = instance.GenerateProjectNewCodePeriodObservation(newCodePeriod)
	}

	mutex.Unlock()

	newCodePeriodsList, ncListResp, ncListErr := c.projectNewCodePeriodsClient.List(instance.GenerateProjectNewCodePeriodsListOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
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

// observeQualityGate retrieves the quality gate associated with the project from SonarQube.
func (c *external) observeQualityGate(projectKey string, result *observeResult, mutex *sync.Mutex) {
	qualityGate, resp, qgErr := c.qualityGatesClient.GetByProject(instance.GenerateQualityGateGetByProjectOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if qgErr != nil {
		result.errors = append(result.errors, errors.Wrap(qgErr, "cannot observe Project quality gate"))

		return
	}

	result.qualityGateName = qualityGate.QualityGate.Name
}

// observeQualityProfiles retrieves the quality profiles associated with the project from SonarQube.
func (c *external) observeQualityProfiles(projectKey string, result *observeResult, mutex *sync.Mutex) {
	qualityProfiles, resp, qpErr := c.qualityProfilesClient.Search(instance.GenerateQualityProfilesSearchProjectOptions(projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	mutex.Lock()
	defer mutex.Unlock()

	if qpErr != nil || len(qualityProfiles.Profiles) == 0 {
		result.errors = append(result.errors, errors.Wrap(qpErr, "cannot observe Project quality profile"))

		return
	}

	result.qualityProfiles = instance.GenerateQualityProfilesSearchProjectObservation(qualityProfiles)
}

// applyObserveResult applies the concurrent observation results to the ProjectObservation.
func (c *external) applyObserveResult(observation *v1alpha1.ProjectObservation, result *observeResult) {
	if result.branches != nil {
		observation.Branches = result.branches
		observation.DefaultBranch = result.mainBranch
	}

	if result.links != nil {
		observation.Links = result.links
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
}

// updateProjectConcurrently launches all project update operations concurrently and collects errors.
func (c *external) updateProjectConcurrently(project *v1alpha1.Project, projectKey string) []error {
	waitGr := sync.WaitGroup{}
	errChan := make(chan error)

	waitGr.Go(func() {
		c.updateVisibility(project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateTags(project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateDefaultBranch(project, projectKey, errChan)
	})

	c.updateBranchNewCodePeriods(project, projectKey, &waitGr, errChan)

	waitGr.Go(func() {
		c.updateProjectLinks(project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectNewCodePeriod(project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectQualityGate(project, projectKey, errChan)
	})

	waitGr.Go(func() {
		c.updateProjectQualityProfiles(project, projectKey, errChan)
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

// updateVisibility updates the project visibility if it differs from the desired state.
func (c *external) updateVisibility(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.Visibility, project.Status.AtProvider.Visibility) {
		return
	}

	resp, err := c.projectsClient.UpdateVisibility(instance.GenerateProjectUpdateVisibilityOptions(projectKey, ptr.Deref(project.Spec.ForProvider.Visibility, ""))) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project visibility")
	}
}

// updateTags updates the project tags.
func (c *external) updateTags(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if project.Spec.ForProvider.Tags == nil {
		return
	}

	resp, err := c.projectTagsClient.Set(instance.GenerateProjectTagsSetOptions(projectKey, *project.Spec.ForProvider.Tags)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project tags")
	}
}

// updateDefaultBranch updates the project default branch if it differs from the desired state.
func (c *external) updateDefaultBranch(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.DefaultBranch, project.Status.AtProvider.DefaultBranch) {
		return
	}

	resp, err := c.projectBranchesClient.SetMain(instance.GenerateProjectBranchesSetMainOptions(projectKey, *project.Spec.ForProvider.DefaultBranch)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project default branch")
	}
}

// updateBranchNewCodePeriods updates the new code periods of branches that differ from the desired state.
func (c *external) updateBranchNewCodePeriods(project *v1alpha1.Project, projectKey string, waitGr *sync.WaitGroup, errChan chan<- error) {
	if instance.AreProjectBranchesUpToDate(project.Spec.ForProvider.Branches, project.Status.AtProvider.Branches) {
		return
	}

	for branch, codePeriodSpec := range project.Spec.ForProvider.Branches {
		waitGr.Add(1)

		go func(branchName string, newCodePeriodSpec *v1alpha1.ProjectNewCodePeriodParameters) {
			defer waitGr.Done()

			branchObservation, branchExists := project.Status.AtProvider.Branches[branchName]
			if branchExists && !instance.IsNewCodePeriodUpToDate(newCodePeriodSpec, &branchObservation.NewCodePeriod) {
				resp, err := c.projectNewCodePeriodsClient.Set(instance.GenerateBranchNewCodePeriodsSetOptions(projectKey, branchName, newCodePeriodSpec)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot update new code period for branch %s", branchName)
				}
			}
		}(branch, codePeriodSpec)
	}
}

// updateProjectLinks updates the links of the project if they differ from the desired state.
func (c *external) updateProjectLinks(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if instance.AreProjectLinksUpToDate(project.Spec.ForProvider.Links, project.Status.AtProvider.Links) {
		return
	}

	for _, link := range project.Spec.ForProvider.Links {
		linkObservation, linkExists := project.Status.AtProvider.Links[link.Name]
		linkUpToDate := linkExists && instance.IsProjectLinkUpToDate(link, &linkObservation)

		if linkExists && !linkUpToDate {
			resp, err := c.projectLinksClient.Delete(instance.GenerateProjectLinksDeleteOptions(linkObservation.ID)) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				errChan <- errors.Wrapf(err, "cannot delete Project link %s", link.Name)
			}
		}

		if !linkUpToDate {
			_, resp, err := c.projectLinksClient.Create(instance.GenerateProjectLinksCreateOptions(projectKey, link)) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				errChan <- errors.Wrapf(err, "cannot create Project link %s", link.Name)
			}
		}
	}
}

// updateProjectNewCodePeriod updates the project-level new code period if it differs from the desired state.
func (c *external) updateProjectNewCodePeriod(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if instance.IsNewCodePeriodUpToDate(project.Spec.ForProvider.NewCodePeriod, &project.Status.AtProvider.NewCodePeriod) {
		return
	}

	resp, err := c.projectNewCodePeriodsClient.Set(instance.GenerateProjectNewCodePeriodsSetOptions(projectKey, project.Spec.ForProvider.NewCodePeriod)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project new code period")
	}
}

// updateProjectQualityGate updates the quality gate of the project if it differs from the desired state.
func (c *external) updateProjectQualityGate(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if helpers.IsComparablePtrEqualComparable(project.Spec.ForProvider.QualityGateName, project.Status.AtProvider.QualityGateName) {
		return
	}

	resp, err := c.qualityGatesClient.Select(instance.GenerateQualityGateSelectOptions(projectKey, *project.Spec.ForProvider.QualityGateName)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		errChan <- errors.Wrap(err, "cannot update Project quality gate")
	}
}

// updateProjectQualityProfiles updates the quality profiles of the project if they differ from the desired state.
func (c *external) updateProjectQualityProfiles(project *v1alpha1.Project, projectKey string, errChan chan<- error) {
	if instance.AreProjectQualityProfilesUpToDate(project.Spec.ForProvider.QualityProfiles, project.Status.AtProvider.QualityProfiles) {
		return
	}

	for language, qualityProfile := range project.Spec.ForProvider.QualityProfiles {
		qualityProfileObservation, qualityProfileExists := project.Status.AtProvider.QualityProfiles[language]
		if qualityProfileExists && helpers.IsComparablePtrEqualComparable(qualityProfile.Id, qualityProfileObservation.Id) {
			continue
		}

		retrievedQualityProfile, resp, err := c.qualityProfilesClient.Show(instance.GenerateQualityProfileShowOptions(*qualityProfile.Id)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			errChan <- errors.Wrapf(err, "cannot retrieve quality profile for language %s", language)

			continue
		}

		resp, err = c.qualityProfilesClient.AddProject(instance.GenerateQualityProfileAddProjectOptions(projectKey, retrievedQualityProfile.Profile.Name, language)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			errChan <- errors.Wrapf(err, "cannot update Project quality profile for language %s", language)
		}
	}
}
