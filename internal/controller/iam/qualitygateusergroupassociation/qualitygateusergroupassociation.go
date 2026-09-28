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

// Package qualitygateusergroupassociation provides a controller for
// QualityGateUsergroupAssociation resources.
package qualitygateusergroupassociation

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
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
	// errNotQualityGateUsergroupAssociation indicates the managed resource
	// is not a QualityGateUsergroupAssociation custom resource.
	errNotQualityGateUsergroupAssociation = "managed resource is not a QualityGateUsergroupAssociation custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errObserveQualityGateUsergroupAssociation indicates association
	// observation failed.
	errObserveQualityGateUsergroupAssociation = "cannot observe QualityGateUsergroupAssociation"
	// errCreateQualityGateUsergroupAssociation indicates association
	// creation failed.
	errCreateQualityGateUsergroupAssociation = "cannot create QualityGateUsergroupAssociation"
	// errUpdateQualityGateUsergroupAssociation indicates association
	// update failed.
	errUpdateQualityGateUsergroupAssociation = "cannot update QualityGateUsergroupAssociation"
	// errDeleteQualityGateUsergroupAssociation indicates association
	// deletion failed.
	errDeleteQualityGateUsergroupAssociation = "cannot delete QualityGateUsergroupAssociation"

	// maxPageSize is the SonarQube search page size used while looking up
	// associated groups or users.
	maxPageSize = int64(100)
)

// SetupGated adds a controller that reconciles
// QualityGateUsergroupAssociation managed resources with safe-start
// support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup QualityGateUsergroupAssociation controller"))
		}
	}, v1alpha1.QualityGateUsergroupAssociationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles
// QualityGateUsergroupAssociation managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.QualityGateUsergroupAssociationGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewQualityGateUsergroupAssociationClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor is marked as deprecated but is not yet replaced with an alternative in controller-runtime, and the APIRecorder is still required for recording events.*/)),
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
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.QualityGateUsergroupAssociationList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.QualityGateUsergroupAssociationList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.QualityGateUsergroupAssociationGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.QualityGateUsergroupAssociation{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.QualityGateUsergroupAssociationClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube Quality Gates
// client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	association, isValid := managedResource.(*v1alpha1.QualityGateUsergroupAssociation)
	if !isValid {
		return nil, errors.New(errNotQualityGateUsergroupAssociation)
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
// QualityGateUsergroupAssociation resources.
type external struct {
	// client is used to interact with the SonarQube Quality Gates API.
	client iam.QualityGateUsergroupAssociationClient
}

// Observe checks whether the external association exists and is up to
// date.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	association, ok := mg.(*v1alpha1.QualityGateUsergroupAssociation)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQualityGateUsergroupAssociation)
	}

	externalName := meta.GetExternalName(association)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// resolveTarget returns an error for invalid format or unknown type.
	// Crossplane defaults the external name to metadata.name before Create
	// runs, so an unparseable name means the resource does not exist yet.
	subjectType, subject, gateName, err := resolveTarget(association, externalName)
	if err != nil || subjectType == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil //nolint:nilerr // An unparseable external name means the association was not created yet.
	}

	observation, found, err := c.observeAssociation(ctx, subjectType, subject, gateName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveQualityGateUsergroupAssociation)
	}

	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	association.Status.AtProvider = observation
	association.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iam.IsQualityGateUsergroupAssociationUpToDate(&association.Spec.ForProvider, &association.Status.AtProvider),
	}, nil
}

// Create grants the group or user edit rights on the Quality Gate, then
// sets the external name.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	association, ok := mg.(*v1alpha1.QualityGateUsergroupAssociation)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotQualityGateUsergroupAssociation)
	}

	association.SetConditions(xpv1.Creating())

	spec := association.Spec.ForProvider

	err := c.addAssociation(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateQualityGateUsergroupAssociation)
	}

	meta.SetExternalName(association, iam.BuildQualityGateUsergroupAssociationExternalName(&spec))

	return managed.ExternalCreation{}, nil
}

// Update grants the group or user edit rights on the Quality Gate.
// Every field of QualityGateUsergroupAssociation is immutable, so this only
// triggers if the observed association drifts from the spec; granting the
// association again converges it.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	association, ok := mg.(*v1alpha1.QualityGateUsergroupAssociation)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotQualityGateUsergroupAssociation)
	}

	err := c.addAssociation(ctx, association.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateQualityGateUsergroupAssociation)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete revokes the group or user edit rights on the Quality Gate.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	association, ok := mg.(*v1alpha1.QualityGateUsergroupAssociation)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotQualityGateUsergroupAssociation)
	}

	association.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(association)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	subjectType, subject, gateName, err := resolveTarget(association, externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityGateUsergroupAssociation)
	}

	err = c.removeAssociation(ctx, subjectType, subject, gateName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteQualityGateUsergroupAssociation)
	}

	return managed.ExternalDelete{}, nil
}

// resolveTarget returns the subject type, subject and gate name identified
// by the external name. When the external name matches the one built from
// the spec, the spec values are used directly so that group or gate names
// containing ":" are not split ambiguously; otherwise the external name is
// parsed (e.g. for imported resources).
func resolveTarget(association *v1alpha1.QualityGateUsergroupAssociation, externalName string) (subjectType, subject, gateName string, err error) {
	spec := association.Spec.ForProvider
	if externalName == iam.BuildQualityGateUsergroupAssociationExternalName(&spec) {
		if ptr.Deref(spec.GroupName, "") != "" {
			return iam.SubjectTypeGroup, *spec.GroupName, spec.GateName, nil
		}

		return iam.SubjectTypeUser, *spec.Login, spec.GateName, nil
	}

	return iam.ParseQualityGateUsergroupAssociationExternalName(externalName)
}

// Disconnect is a no-op for stateless clients.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// observeAssociation looks up the group or user on the Quality Gate and
// returns the observation built from the SonarQube response. found is
// false when the principal is not selected on the Quality Gate.
func (c *external) observeAssociation(ctx context.Context, subjectType, subject, gateName string) (v1alpha1.QualityGateUsergroupAssociationObservation, bool, error) {
	if subjectType == iam.SubjectTypeGroup {
		return c.searchSelectedGroup(ctx, subject, gateName)
	}

	return c.searchSelectedUser(ctx, subject, gateName)
}

// searchSelectedGroup paginates SearchGroups looking for a selected group
// whose name matches groupName, and returns the observation built from it.
//
//nolint:dupl // Intentional structural similarity with searchSelectedUser; different API types prevent abstraction.
func (c *external) searchSelectedGroup(ctx context.Context, groupName, gateName string) (v1alpha1.QualityGateUsergroupAssociationObservation, bool, error) {
	for page := int64(1); ; page++ {
		opts := iam.GenerateQualityGateSearchGroupsOptions(gateName, groupName, &sonar.PaginationArgs{
			Page:     page,
			PageSize: maxPageSize,
		})

		result, resp, err := c.client.SearchGroups(ctx, opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			// The Quality Gate no longer exists, so neither does the
			// association.
			if common.IsResponseNotFound(resp) {
				return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
			}

			return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, errors.Wrap(err, "cannot search quality gate groups")
		}

		for idx := range result.Groups {
			if result.Groups[idx].Name == groupName {
				if !result.Groups[idx].Selected {
					return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
				}

				return iam.GenerateQualityGateGroupAssociationObservation(gateName, &result.Groups[idx]), true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
		}
	}
}

// searchSelectedUser paginates SearchUsers looking for a selected user
// whose login matches login, and returns the observation built from it.
//
//nolint:dupl // Intentional structural similarity with searchSelectedGroup; different API types prevent abstraction.
func (c *external) searchSelectedUser(ctx context.Context, login, gateName string) (v1alpha1.QualityGateUsergroupAssociationObservation, bool, error) {
	for page := int64(1); ; page++ {
		opts := iam.GenerateQualityGateSearchUsersOptions(gateName, login, &sonar.PaginationArgs{
			Page:     page,
			PageSize: maxPageSize,
		})

		result, resp, err := c.client.SearchUsers(ctx, opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			// The Quality Gate no longer exists, so neither does the
			// association.
			if common.IsResponseNotFound(resp) {
				return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
			}

			return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, errors.Wrap(err, "cannot search quality gate users")
		}

		for idx := range result.Users {
			if result.Users[idx].Login == login {
				if !result.Users[idx].Selected {
					return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
				}

				return iam.GenerateQualityGateUserAssociationObservation(gateName, &result.Users[idx]), true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return v1alpha1.QualityGateUsergroupAssociationObservation{}, false, nil
		}
	}
}

// addAssociation calls AddGroup or AddUser based on the configured
// principal.
func (c *external) addAssociation(ctx context.Context, spec v1alpha1.QualityGateUsergroupAssociationParameters) error {
	if ptr.Deref(spec.GroupName, "") != "" {
		resp, err := c.client.AddGroup(ctx, iam.GenerateQualityGateAddGroupOptions(spec.GateName, *spec.GroupName)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return err
	}

	if ptr.Deref(spec.Login, "") != "" {
		resp, err := c.client.AddUser(ctx, iam.GenerateQualityGateAddUserOptions(spec.GateName, *spec.Login)) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return err
	}

	return errors.New("exactly one of groupName or login must be set")
}

// removeAssociation calls RemoveGroup or RemoveUser based on the subject
// type encoded in the external name. A not-found response means the Quality
// Gate, group or user is already gone, so the association no longer exists.
func (c *external) removeAssociation(ctx context.Context, subjectType, subject, gateName string) error {
	var (
		resp *http.Response
		err  error
	)

	if subjectType == iam.SubjectTypeGroup {
		resp, err = c.client.RemoveGroup(ctx, iam.GenerateQualityGateRemoveGroupOptions(gateName, subject)) //nolint:bodyclose // closed via helpers.CloseBody
	} else {
		resp, err = c.client.RemoveUser(ctx, iam.GenerateQualityGateRemoveUserOptions(gateName, subject)) //nolint:bodyclose // closed via helpers.CloseBody
	}

	defer helpers.CloseBody(resp)

	if err != nil && common.IsResponseNotFound(resp) {
		return nil
	}

	return err
}
