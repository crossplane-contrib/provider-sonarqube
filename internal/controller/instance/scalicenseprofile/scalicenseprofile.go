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

// Package scalicenseprofile provides a controller for SCALicenseProfile
// resources.
package scalicenseprofile

import (
	"context"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errExternalNameNotSet indicates the external name annotation is missing.
	errExternalNameNotSet = "external name is not set for SCALicenseProfile resource %s"

	// errListLicenseProfiles indicates listing the license profiles failed.
	errListLicenseProfiles = "cannot list SonarQube SCA license profiles"
	// errObserveLicenseProfile indicates license profile observation failed.
	errObserveLicenseProfile = "cannot observe SonarQube SCA license profile"
	// errObserveProjects indicates listing the assigned projects failed.
	errObserveProjects = "cannot observe the projects assigned to SonarQube SCA license profile"
	// errCreateLicenseProfile indicates license profile creation failed.
	errCreateLicenseProfile = "cannot create SonarQube SCA license profile"
	// errNoLicenseProfileID indicates SonarQube returned no identifier on creation.
	errNoLicenseProfileID = "SonarQube returned no SCA license profile identifier"
	// errUpdateLicenseProfile indicates license profile update failed.
	errUpdateLicenseProfile = "cannot update SonarQube SCA license profile"
	// errDeleteLicenseProfile indicates license profile deletion failed.
	errDeleteLicenseProfile = "cannot delete SonarQube SCA license profile"
	// errUpdateCategory indicates a license category policy update failed.
	errUpdateCategory = "cannot update the policy of license category %s"
	// errUpdateLicense indicates a license policy update failed.
	errUpdateLicense = "cannot update the policy of license %s"
	// errUnknownLicenses indicates desired licenses are not in the profile.
	errUnknownLicenses = "licenses not found in SonarQube SCA license profile: %s"
	// errAssignProject indicates assigning a project failed.
	errAssignProject = "cannot assign project %s to SonarQube SCA license profile"
	// errUnassignProject indicates unassigning a project failed.
	errUnassignProject = "cannot unassign project %s from SonarQube SCA license profile"
)

// SetupGated adds a controller that reconciles SCALicenseProfile managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o helpers.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup SCALicenseProfile controller"))
		}
	}, v1alpha1.SCALicenseProfileGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles SCALicenseProfile managed
// resources. The external name is the identifier SonarQube assigns to the
// license profile on creation.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.SCALicenseProfileGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.SCALicenseProfile](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewSCALicenseProfilesClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.SCALicenseProfileList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.SCALicenseProfileGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.SCALicenseProfile{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its
// Connect method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.SCALicenseProfilesClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube SCA license profiles
// client.
func (c *connector) Connect(ctx context.Context, profile *v1alpha1.SCALicenseProfile) (managed.TypedExternalClient[*v1alpha1.SCALicenseProfile], error) {
	err := c.usage.Track(ctx, profile)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, profile)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// external implements the ExternalClient interface for SCALicenseProfile
// resources.
type external struct {
	client instance.SCALicenseProfilesClient
}

// Observe checks if the license profile exists in SonarQube and whether its
// name, default flag, policies and assigned projects match the desired
// state.
//
// The license profile is looked up in the list of license profiles before
// being fetched: until Create sets it to the SonarQube identifier, the
// external name defaults to the resource name, which is not a valid
// license profile key.
func (c *external) Observe(ctx context.Context, profile *v1alpha1.SCALicenseProfile) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	index, resp, err := c.client.ListLicenseProfiles(ctx, &sonar.ScaLicenseProfileListOptions{}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errListLicenseProfiles)
	}

	if index == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if _, found := instance.FindSCALicenseProfile(index.LicenseProfiles, externalName); !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	details, resp, err := c.client.GetLicenseProfile(ctx, &sonar.ScaLicenseProfileGetOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObserveLicenseProfile)
	}

	if details == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	projects, err := instance.FetchSCALicenseProfileProjects(ctx, c.client, externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveProjects)
	}

	profile.Status.AtProvider = instance.GenerateSCALicenseProfileObservation(details, profile.Spec.ForProvider.Licenses, projects)
	profile.SetConditions(xpv1.Available())

	former := profile.Spec.ForProvider.DeepCopy()
	instance.LateInitializeSCALicenseProfile(&profile.Spec.ForProvider, &profile.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsSCALicenseProfileUpToDate(&profile.Spec.ForProvider, &profile.Status.AtProvider),
		ResourceLateInitialized: instance.IsSCALicenseProfileLateInitialized(former, &profile.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the license profile in SonarQube and sets the external
// name to its identifier. Policies and assigned projects are reconciled by
// the subsequent Update.
func (c *external) Create(ctx context.Context, profile *v1alpha1.SCALicenseProfile) (managed.ExternalCreation, error) {
	profile.Status.SetConditions(xpv1.Creating())

	created, resp, err := c.client.CreateLicenseProfile(ctx, instance.GenerateSCALicenseProfileCreateOptions(&profile.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLicenseProfile)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoLicenseProfileID), errCreateLicenseProfile)
	}

	meta.SetExternalName(profile, created.Id)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update reconciles the license profile name, default flag, category and
// license policies and assigned projects in SonarQube.
func (c *external) Update(ctx context.Context, profile *v1alpha1.SCALicenseProfile) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.Errorf(errExternalNameNotSet, profile.Name)
	}

	spec := &profile.Spec.ForProvider
	observation := &profile.Status.AtProvider

	if !instance.IsSCALicenseProfileSettingsUpToDate(spec, observation) {
		_, resp, err := c.client.UpdateLicenseProfile(ctx, externalName, instance.GenerateSCALicenseProfileUpdateOptions(spec)) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLicenseProfile)
		}
	}

	// Categories are updated before licenses, so that the desired license
	// policies take precedence over the policy of their category.
	categories := instance.SCALicenseProfileCategoriesToUpdate(spec.Categories, observation.Categories)

	err := c.updateCategories(ctx, externalName, categories)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	missing, err := c.updateLicenses(ctx, externalName, spec.Licenses, licensesAfterCategoryUpdates(observation.Licenses, categories))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.assignProjects(ctx, externalName, instance.SCALicenseProfileProjectsToAssign(spec.Projects, observation.Projects))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.unassignProjects(ctx, instance.SCALicenseProfileProjectsToUnassign(spec.Projects, observation.Projects))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	// Unknown licenses are reported last, so that they do not keep the
	// assigned projects from being reconciled.
	if len(missing) > 0 {
		return managed.ExternalUpdate{}, errors.Errorf(errUnknownLicenses, strings.Join(missing, ", "))
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// licensesAfterCategoryUpdates returns the observed license policies as they
// stand once the given category policies are applied: updating a category
// policy sets the policy of every license of the category, so the observed
// policy of those licenses no longer holds.
func licensesAfterCategoryUpdates(observation []v1alpha1.SCALicenseProfileLicenseObservation, categories []v1alpha1.SCALicenseProfileCategoryPolicy) []v1alpha1.SCALicenseProfileLicenseObservation {
	if len(categories) == 0 {
		return observation
	}

	updated := make(map[string]string, len(categories))
	for _, category := range categories {
		updated[category.Category] = category.Policy
	}

	licenses := make([]v1alpha1.SCALicenseProfileLicenseObservation, len(observation))
	for i, license := range observation {
		if policy, found := updated[license.Category]; found {
			license.Policy = policy
		}

		licenses[i] = license
	}

	return licenses
}

// Delete deletes the license profile from SonarQube.
func (c *external) Delete(ctx context.Context, profile *v1alpha1.SCALicenseProfile) (managed.ExternalDelete, error) {
	profile.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(profile)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.DeleteLicenseProfile(ctx, &sonar.ScaLicenseProfileGetOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	// The client resends a DELETE after a transient failure, so a 404 can
	// mean an earlier attempt succeeded.
	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLicenseProfile)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// updateCategories applies the given category policies to the license
// profile.
func (c *external) updateCategories(ctx context.Context, key string, categories []v1alpha1.SCALicenseProfileCategoryPolicy) error {
	for _, category := range categories {
		_, resp, err := c.client.UpdateLicenseProfileCategory(ctx, //nolint:bodyclose // closed via helpers.CloseBody
			&sonar.ScaLicenseProfileCategoryOptions{LicenseProfileKey: key, CategoryKey: category.Category},
			&sonar.ScaLicenseProfileCategoryUpdateRequest{Policy: category.Policy},
		)
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errUpdateCategory, category.Category)
		}
	}

	return nil
}

// updateLicenses applies the desired license policies that differ from the
// observed ones, and returns the SPDX identifiers of the desired licenses
// that are not known to the license profile.
func (c *external) updateLicenses(ctx context.Context, key string, spec []v1alpha1.SCALicenseProfileLicensePolicy, observation []v1alpha1.SCALicenseProfileLicenseObservation) ([]string, error) {
	updates, missing := instance.SCALicenseProfileLicensesToUpdate(spec, observation)

	for _, update := range updates {
		_, resp, err := c.client.UpdateLicenseProfileLicense(ctx, //nolint:bodyclose // closed via helpers.CloseBody
			&sonar.ScaLicenseProfileLicenseOptions{LicenseProfileKey: key, LicensePolicyId: update.ID},
			&sonar.ScaLicensePolicyLicenseUpdateRequest{Policy: update.Policy},
		)
		helpers.CloseBody(resp)

		if err != nil {
			return nil, errors.Wrapf(err, errUpdateLicense, update.SPDXLicenseID)
		}
	}

	return missing, nil
}

// assignProjects assigns the given projects to the license profile.
func (c *external) assignProjects(ctx context.Context, key string, projects []string) error {
	for _, project := range projects {
		_, resp, err := c.client.AssignLicenseProfileProject(ctx, &sonar.ScaLicenseProfileAssignmentRequest{LicenseProfileId: key, ProjectKey: project}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errAssignProject, project)
		}
	}

	return nil
}

// unassignProjects removes the license profile assignment of the given
// projects.
func (c *external) unassignProjects(ctx context.Context, projects []string) error {
	for _, project := range projects {
		resp, err := c.client.DeleteLicenseProfileAssignment(ctx, &sonar.ScaLicenseProfileAssignmentDeleteOptions{ProjectKey: project}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil && !common.IsResponseNotFound(resp) {
			return errors.Wrapf(err, errUnassignProject, project)
		}
	}

	return nil
}
