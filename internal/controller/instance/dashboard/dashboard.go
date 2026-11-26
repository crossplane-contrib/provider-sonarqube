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

// Package dashboard provides a controller for Dashboard resources.
package dashboard

import (
	"context"

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
	errExternalNameNotSet = "external name is not set for Dashboard resource %s"

	// errObserveDashboard indicates dashboard observation failed.
	errObserveDashboard = "cannot observe SonarQube dashboard"
	// errCreateDashboard indicates dashboard creation in SonarQube failed.
	errCreateDashboard = "cannot create SonarQube dashboard"
	// errUpdateDashboard indicates dashboard update in SonarQube failed.
	errUpdateDashboard = "cannot update SonarQube dashboard"
	// errDeleteDashboard indicates dashboard deletion in SonarQube failed.
	errDeleteDashboard = "cannot delete SonarQube dashboard"
	// errListDashboards indicates listing the dashboards of the target failed.
	errListDashboards = "cannot list the SonarQube dashboards of the target"
	// errLookupTarget indicates the target component lookup failed.
	errLookupTarget = "cannot look up SonarQube %s %q"
	// errNoTargetID indicates SonarQube returned no identifier for a
	// component.
	errNoTargetID = "SonarQube returned no identifier"
	// errTargetNotFound indicates the target project or portfolio does not
	// exist.
	errTargetNotFound = "SonarQube %s %q does not exist"
	// errNoTarget indicates the spec holds no resolved target.
	errNoTarget = "neither projectKey nor portfolioKey is set"
	// errTargetMismatch indicates the dashboard identified by the external
	// name does not belong to the target of the spec.
	errTargetMismatch = "dashboard %s does not belong to the %s %q: the target of a dashboard cannot be changed"
	// errDashboardExists indicates a dashboard with the desired name already
	// exists.
	errDashboardExists = "a dashboard named %q already exists on the %s %q with ID %s: set the crossplane.io/external-name annotation to this ID to manage it"
	// errNoDashboardID indicates SonarQube returned no identifier for the
	// created dashboard.
	errNoDashboardID = "SonarQube returned no dashboard ID"
)

// SetupGated adds a controller that reconciles Dashboard managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o helpers.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Dashboard controller"))
		}
	}, v1alpha1.DashboardGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Dashboard managed resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.DashboardGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		// No initializer: the default one would use the resource name as the
		// external name, and a resource named like a dashboard ID would take
		// over that dashboard.
		managed.WithInitializers(),
		managed.WithTypedExternalConnector[*v1alpha1.Dashboard](&connector{
			kube:            mgr.GetClient(),
			usage:           resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn:    instance.NewDashboardsClient,
			newNavigationFn: instance.NewComponentNavigationClient,
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.DashboardList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.DashboardGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Dashboard{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its
// Connect method is called.
type connector struct {
	kube            client.Client
	usage           *resource.ProviderConfigUsageTracker
	newServiceFn    func(config common.Config) instance.DashboardsClient
	newNavigationFn func(config common.Config) instance.ComponentNavigationClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing the SonarQube clients.
func (c *connector) Connect(ctx context.Context, dashboard *v1alpha1.Dashboard) (managed.TypedExternalClient[*v1alpha1.Dashboard], error) {
	err := c.usage.Track(ctx, dashboard)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, dashboard)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config), navigation: c.newNavigationFn(*config)}, nil
}

// external implements the ExternalClient interface for Dashboard resources.
type external struct {
	client     instance.DashboardsClient
	navigation instance.ComponentNavigationClient
}

// Observe checks whether the Dashboard exists in SonarQube and whether it is
// up to date.
//
// The external name is the dashboard ID assigned by SonarQube. A dashboard
// that does not belong to the target of the spec is never updated nor
// deleted: it is reported as an error, or as missing once the resource is
// being deleted.
func (c *external) Observe(ctx context.Context, dashboard *v1alpha1.Dashboard) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(dashboard)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, resp, err := c.client.Get(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObserveDashboard)
	}

	if observed == nil {
		return managed.ExternalObservation{}, errors.New(errObserveDashboard)
	}

	owned, err := c.isOwnedByTarget(ctx, dashboard, observed)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if !owned {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	dashboard.Status.AtProvider = instance.GenerateDashboardObservation(observed)

	former := dashboard.Spec.ForProvider.DeepCopy()
	instance.LateInitializeDashboard(&dashboard.Spec.ForProvider, &dashboard.Status.AtProvider)

	dashboard.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsDashboardUpToDate(&dashboard.Spec.ForProvider, &dashboard.Status.AtProvider),
		ResourceLateInitialized: instance.IsDashboardLateInitialized(former, &dashboard.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the Dashboard in SonarQube and sets the external name to the
// ID assigned by SonarQube.
//
// It fails when the target already has a dashboard with the same name,
// instead of taking it over or duplicating it. This also covers a previous
// creation whose response was lost.
func (c *external) Create(ctx context.Context, dashboard *v1alpha1.Dashboard) (managed.ExternalCreation, error) {
	dashboard.SetConditions(xpv1.Creating())

	spec := &dashboard.Spec.ForProvider

	resourceType, key := instance.DashboardTarget(spec)
	if key == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoTarget), errCreateDashboard)
	}

	targetID, err := c.findTargetID(ctx, resourceType, key)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDashboard)
	}

	if targetID == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.Errorf(errTargetNotFound, resourceType, key), errCreateDashboard)
	}

	existing, found, err := instance.FindDashboardByName(ctx, c.client, resourceType, targetID, spec.Name)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errListDashboards)
	}

	if found {
		return managed.ExternalCreation{}, errors.Errorf(errDashboardExists, spec.Name, resourceType, key, existing.Id)
	}

	created, resp, err := c.client.Create(ctx, instance.GenerateDashboardCreateOptions(spec, resourceType, targetID)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDashboard)
	}

	if created == nil || created.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoDashboardID), errCreateDashboard)
	}

	meta.SetExternalName(dashboard, created.Id)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update updates the name, description and layout of the Dashboard in
// SonarQube.
func (c *external) Update(ctx context.Context, dashboard *v1alpha1.Dashboard) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(dashboard)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.Errorf(errExternalNameNotSet, dashboard.Name)
	}

	_, resp, err := c.client.Update(ctx, externalName, instance.GenerateDashboardUpdateOptions(&dashboard.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateDashboard)
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete deletes the Dashboard from SonarQube.
func (c *external) Delete(ctx context.Context, dashboard *v1alpha1.Dashboard) (managed.ExternalDelete, error) {
	dashboard.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(dashboard)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(ctx, externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	// The client resends a DELETE after a transient failure, so a 404 can mean
	// an earlier attempt succeeded.
	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDashboard)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// isOwnedByTarget returns true when the observed dashboard belongs to the
// target of the spec. Otherwise it returns an error, unless the resource is
// being deleted, which then leaves the dashboard in place.
func (c *external) isOwnedByTarget(ctx context.Context, dashboard *v1alpha1.Dashboard, observed *sonar.DashboardsDashboardResponse) (bool, error) {
	resourceType, key := instance.DashboardTarget(&dashboard.Spec.ForProvider)

	targetID, err := c.findTargetID(ctx, resourceType, key)
	if err != nil {
		return false, errors.Wrap(err, errObserveDashboard)
	}

	switch {
	case targetID != "" && observed.ResourceId == targetID && observed.ResourceType == resourceType:
		return true, nil
	case meta.WasDeleted(dashboard):
		return false, nil
	case key == "":
		return false, errors.New(errNoTarget)
	default:
		return false, errors.Errorf(errTargetMismatch, meta.GetExternalName(dashboard), resourceType, key)
	}
}

// findTargetID returns the identifier of the project or portfolio with the
// given key, which the dashboards API identifies them with. It is empty when
// the key is empty or no component has it.
func (c *external) findTargetID(ctx context.Context, resourceType, key string) (string, error) {
	// Without a key, SonarQube cannot designate any component.
	if key == "" {
		return "", nil
	}

	component, resp, err := c.navigation.Component(ctx, &sonar.NavigationComponentOptions{Component: key}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return "", nil
		}

		return "", errors.Wrapf(err, errLookupTarget, resourceType, key)
	}

	if component == nil || component.ID == "" {
		return "", errors.Wrapf(errors.New(errNoTargetID), errLookupTarget, resourceType, key)
	}

	return component.ID, nil
}
