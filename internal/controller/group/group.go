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

package group

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	errNotGroup     = "managed resource is not a Group custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errNoGroupID    = "created Group has empty ID"
	errDeleteGroup  = "cannot delete Group"
	errUpdateGroup  = "cannot update Group"
	errCreateGroup  = "cannot create Group"
	errObserveGroup = "cannot observe Group"
)

// SetupGated adds a controller that reconciles Group managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options controller.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Group controller"))
		}
	}, v1alpha1.GroupGroupVersionKind)

	return nil
}

func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.GroupGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewGroupsClient,
		}),
		managed.WithLogger(options.Logger.WithValues("controller", name)),
		managed.WithPollInterval(options.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor is marked as deprecated but is not yet replaced with an alternative in controller-runtime, and the APIRecorder is still required for recording events.*/)),
	}

	if options.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if options.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(options.ChangeLogOptions.ChangeLogger))
	}

	if options.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(options.MetricOptions.MRMetrics))
	}

	if options.MetricOptions != nil && options.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.GroupList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.GroupList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.GroupGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Group{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.GroupsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	group, isValid := managedResource.(*v1alpha1.Group)
	if !isValid {
		return nil, errors.New(errNotGroup)
	}

	err := c.usage.Track(ctx, group)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	if config == nil {
		return nil, errors.Wrap(errors.New("empty configuration returned"), errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// client is used to interact with SonarQube Group API
	client iam.GroupsClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	group, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGroup)
	}

	// Use external name as the identifier to observe the resource
	externalName := meta.GetExternalName(group)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	retrievedGroup, resp, err := c.client.FetchGroup(externalName) //nolint:bodyclose // closed via helpers.CloseBody
	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObserveGroup)
	}
	defer helpers.CloseBody(resp)

	group.Status.AtProvider = iam.GenerateGroupObservation(retrievedGroup)

	former := group.Spec.ForProvider.DeepCopy()
	iam.LateInitializeGroup(&group.Spec.ForProvider, &group.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        iam.IsGroupUpToDate(&group.Spec.ForProvider, &group.Status.AtProvider),
		ResourceLateInitialized: iam.IsGroupLateInitialized(former, &group.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create a new external resource based on the managed resource spec. The external resource should be created in a way that it will be observed as up to date by Observe for the managed resource.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	group, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGroup)
	}

	group.Status.SetConditions(xpv1.Creating())

	createdGroup, resp, err := c.client.CreateGroup(iam.GenerateCreateGroupOptions(&group.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateGroup)
	}
	defer helpers.CloseBody(resp)

	if createdGroup == nil || createdGroup.Id == "" {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNoGroupID), errCreateGroup)
	}

	// Set the external name to the identifier of the created resource.
	meta.SetExternalName(group, createdGroup.Id)

	return managed.ExternalCreation{}, nil
}

// Update is required to update an existing external resource to match the desired state of the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	group, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGroup)
	}

	externalName := meta.GetExternalName(group)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New("cannot update Group without external name")
	}

	_, resp, err := c.client.UpdateGroup(externalName, iam.GenerateUpdateGroupOptions(&group.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateGroup)
	}
	defer helpers.CloseBody(resp)

	return managed.ExternalUpdate{}, nil
}

// Delete will delete the external resource if it exists. After deletion, Observe should return ResourceExists=false to indicate that the external resource no longer exists.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	group, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGroup)
	}

	group.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(group)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	destroyResp, err := c.client.DeleteGroup(externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(destroyResp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteGroup)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
