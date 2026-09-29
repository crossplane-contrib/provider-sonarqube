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

// Package qualityprofileusergroupassociation provides a controller for
// QualityProfileUsergroupAssociation resources.
package qualityprofileusergroupassociation

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotQualityProfileUsergroupAssociation indicates the managed
	// resource is not a QualityProfileUsergroupAssociation custom
	// resource.
	errNotQualityProfileUsergroupAssociation = "managed resource is not a QualityProfileUsergroupAssociation custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errObserveQualityProfileUsergroupAssociation indicates
	// association observation failed.
	errObserveQualityProfileUsergroupAssociation = "cannot observe QualityProfileUsergroupAssociation"
	// errCreateQualityProfileUsergroupAssociation indicates association
	// creation failed.
	errCreateQualityProfileUsergroupAssociation = "cannot create QualityProfileUsergroupAssociation"
	// errUpdateQualityProfileUsergroupAssociation indicates association
	// update failed.
	errUpdateQualityProfileUsergroupAssociation = "cannot update QualityProfileUsergroupAssociation"
	// errDeleteQualityProfileUsergroupAssociation indicates association
	// deletion failed.
	errDeleteQualityProfileUsergroupAssociation = "cannot delete QualityProfileUsergroupAssociation"

	// maxPageSize is the SonarQube search page size used while looking
	// up associated groups or users.
	maxPageSize = int64(100)
)

// SetupGated adds a controller that reconciles
// QualityProfileUsergroupAssociation managed resources with safe-start
// support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup QualityProfileUsergroupAssociation controller"))
		}
	}, v1alpha1.QualityProfileUsergroupAssociationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles
// QualityProfileUsergroupAssociation managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.QualityProfileUsergroupAssociationGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewQualityProfileUsergroupAssociationClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(helpers.NewEventRecorder(mgr, name)),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		options = append(options, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		options = append(options, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		options = append(options, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.QualityProfileUsergroupAssociationList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.QualityProfileUsergroupAssociationList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.QualityProfileUsergroupAssociationGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.QualityProfileUsergroupAssociation{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.QualityProfileUsergroupAssociationClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube Quality Profiles
// client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	association, isValid := managedResource.(*v1alpha1.QualityProfileUsergroupAssociation)
	if !isValid {
		return nil, errors.New(errNotQualityProfileUsergroupAssociation)
	}

	err := c.usage.Track(ctx, association)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// external implements the ExternalClient interface for
// QualityProfileUsergroupAssociation resources.
type external struct {
	// client is used to interact with the SonarQube Quality Profiles
	// API.
	client iam.QualityProfileUsergroupAssociationClient
}

// Observe checks whether the external association exists and is up to
// date.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	association, ok := mg.(*v1alpha1.QualityProfileUsergroupAssociation)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQualityProfileUsergroupAssociation)
	}

	externalName := meta.GetExternalName(association)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// ParseQualityProfileUsergroupAssociationExternalName returns an
	// error for invalid format or unknown type. Crossplane defaults
	// the external name to metadata.name before Create runs, so an
	// unparseable name means the resource does not exist yet.
	subjectType, subject, language, qualityProfile, err := iam.ParseQualityProfileUsergroupAssociationExternalName(externalName)
	if err != nil || subjectType == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil //nolint:nilerr // An unparseable external name means the association was not created yet.
	}

	observation, found, err := c.observeAssociation(ctx, subjectType, subject, language, qualityProfile)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveQualityProfileUsergroupAssociation)
	}

	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	association.Status.AtProvider = observation
	association.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iam.IsQualityProfileUsergroupAssociationUpToDate(&association.Spec.ForProvider, &association.Status.AtProvider),
	}, nil
}

// Create grants the group or user edit rights on the Quality Profile,
// then sets the external name.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	association, ok := mg.(*v1alpha1.QualityProfileUsergroupAssociation)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotQualityProfileUsergroupAssociation)
	}

	association.SetConditions(xpv1.Creating())

	spec := association.Spec.ForProvider

	err := c.addAssociation(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateQualityProfileUsergroupAssociation)
	}

	meta.SetExternalName(association, iam.BuildQualityProfileUsergroupAssociationExternalName(&spec))

	return managed.ExternalCreation{}, nil
}

// Update grants the group or user edit rights on the Quality Profile.
// Every field of QualityProfileUsergroupAssociation is immutable, so this
// only triggers if the observed association drifts from the spec; granting
// the association again converges it.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	association, ok := mg.(*v1alpha1.QualityProfileUsergroupAssociation)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotQualityProfileUsergroupAssociation)
	}

	err := c.addAssociation(ctx, association.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateQualityProfileUsergroupAssociation)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete revokes the group or user edit rights on the Quality Profile.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	association, ok := mg.(*v1alpha1.QualityProfileUsergroupAssociation)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotQualityProfileUsergroupAssociation)
	}

	association.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(association)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	subjectType, subject, language, qualityProfile, err := iam.ParseQualityProfileUsergroupAssociationExternalName(externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityProfileUsergroupAssociation)
	}

	err = c.removeAssociation(ctx, subjectType, subject, language, qualityProfile)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityProfileUsergroupAssociation)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for stateless clients.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// observeAssociation looks up the group or user on the Quality Profile and
// returns the observation built from the SonarQube response. found is
// false when the principal is not selected on the Quality Profile.
func (c *external) observeAssociation(ctx context.Context, subjectType, subject, language, qualityProfile string) (v1alpha1.QualityProfileUsergroupAssociationObservation, bool, error) {
	if subjectType == iam.SubjectTypeGroup {
		return c.searchSelectedGroup(ctx, subject, language, qualityProfile)
	}

	return c.searchSelectedUser(ctx, subject, language, qualityProfile)
}

// searchSelectedGroup paginates SearchGroups looking for a selected group
// whose name matches groupName, and returns the observation built from it.
//
//nolint:dupl // Intentional structural similarity with searchSelectedUser; different API types prevent abstraction.
func (c *external) searchSelectedGroup(ctx context.Context, groupName, language, qualityProfile string) (v1alpha1.QualityProfileUsergroupAssociationObservation, bool, error) {
	for page := int64(1); ; page++ {
		opts := iam.GenerateQualityProfileSearchGroupsOptions(language, qualityProfile, groupName, &sonar.PaginationArgs{
			Page:     page,
			PageSize: maxPageSize,
		})

		result, resp, err := c.client.SearchGroups(ctx, opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		// The Quality Profile no longer exists, so neither does the
		// association.
		if common.IsResponseNotFound(resp) {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
		}

		if err != nil {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, errors.Wrap(err, "cannot search quality profile groups")
		}

		for idx := range result.Groups {
			if result.Groups[idx].Name == groupName {
				if !result.Groups[idx].Selected {
					return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
				}

				return iam.GenerateQualityProfileGroupAssociationObservation(language, qualityProfile, &result.Groups[idx]), true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
		}
	}
}

// searchSelectedUser paginates SearchUsers looking for a selected user
// whose login matches login, and returns the observation built from it.
//
//nolint:dupl // Intentional structural similarity with searchSelectedGroup; different API types prevent abstraction.
func (c *external) searchSelectedUser(ctx context.Context, login, language, qualityProfile string) (v1alpha1.QualityProfileUsergroupAssociationObservation, bool, error) {
	for page := int64(1); ; page++ {
		opts := iam.GenerateQualityProfileSearchUsersOptions(language, qualityProfile, login, &sonar.PaginationArgs{
			Page:     page,
			PageSize: maxPageSize,
		})

		result, resp, err := c.client.SearchUsers(ctx, opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		// The Quality Profile no longer exists, so neither does the
		// association.
		if common.IsResponseNotFound(resp) {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
		}

		if err != nil {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, errors.Wrap(err, "cannot search quality profile users")
		}

		for idx := range result.Users {
			if result.Users[idx].Login == login {
				if !result.Users[idx].Selected {
					return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
				}

				return iam.GenerateQualityProfileUserAssociationObservation(language, qualityProfile, &result.Users[idx]), true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return v1alpha1.QualityProfileUsergroupAssociationObservation{}, false, nil
		}
	}
}

// addAssociation calls AddGroup or AddUser based on the configured
// principal.
func (c *external) addAssociation(ctx context.Context, spec v1alpha1.QualityProfileUsergroupAssociationParameters) error {
	if ptr.Deref(spec.GroupName, "") != "" {
		resp, err := c.client.AddGroup(ctx, iam.GenerateQualityProfileAddGroupOptions(spec.Language, spec.QualityProfile, *spec.GroupName)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return err
	}

	if ptr.Deref(spec.Login, "") != "" {
		resp, err := c.client.AddUser(ctx, iam.GenerateQualityProfileAddUserOptions(spec.Language, spec.QualityProfile, *spec.Login)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return err
	}

	return errors.New("exactly one of groupName or login must be set")
}

// removeAssociation calls RemoveGroup or RemoveUser based on the subject
// type encoded in the external name. A 404 (profile or principal already
// gone) is treated as success.
func (c *external) removeAssociation(ctx context.Context, subjectType, subject, language, qualityProfile string) error {
	if subjectType == iam.SubjectTypeGroup {
		resp, err := c.client.RemoveGroup(ctx, iam.GenerateQualityProfileRemoveGroupOptions(language, qualityProfile, subject)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		if common.IsResponseNotFound(resp) {
			return nil
		}

		return err
	}

	resp, err := c.client.RemoveUser(ctx, iam.GenerateQualityProfileRemoveUserOptions(language, qualityProfile, subject)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if common.IsResponseNotFound(resp) {
		return nil
	}

	return err
}
