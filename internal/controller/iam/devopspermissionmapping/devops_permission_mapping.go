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

// Package devopspermissionmapping provides a controller for
// DevOpsPermissionMapping resources.
package devopspermissionmapping

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errObserveMapping indicates the permission mappings could not be read.
	errObserveMapping = "cannot get DevOps platform permission mappings"
	// errCreateMapping indicates permission mapping creation failed.
	errCreateMapping = "cannot create DevOps platform permission mapping"
	// errUpdateMapping indicates permission mapping update failed.
	errUpdateMapping = "cannot update DevOps platform permission mapping"
	// errDeleteMapping indicates permission mapping deletion failed.
	errDeleteMapping = "cannot delete DevOps platform permission mapping"
	// errExternalName indicates the external name does not designate the role
	// of the spec.
	errExternalName = "external name %q does not match spec.forProvider.role %q"
)

// SetupGated adds a controller that reconciles DevOpsPermissionMapping managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, options helpers.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup DevOpsPermissionMapping controller"))
		}
	}, v1alpha1.DevOpsPermissionMappingGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles DevOpsPermissionMapping managed
// resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.DevOpsPermissionMappingGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.DevOpsPermissionMapping](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewDevOpsPermissionMappingsClient,
		}),
		// The external name is the role, never metadata.name: a resource named
		// after a role must not take over its mapping.
		helpers.WithSpecExternalName(mgr.GetClient(), func(mg *v1alpha1.DevOpsPermissionMapping) string {
			return mg.Spec.ForProvider.Role
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.DevOpsPermissionMappingList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.DevOpsPermissionMappingGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.DevOpsPermissionMapping{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.DevOpsPermissionMappingsClient
}

// Connect tracks the ProviderConfig usage and builds a client from the
// ProviderConfig of the managed resource.
func (c *connector) Connect(ctx context.Context, mapping *v1alpha1.DevOpsPermissionMapping) (managed.TypedExternalClient[*v1alpha1.DevOpsPermissionMapping], error) {
	err := c.usage.Track(ctx, mapping)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, mapping)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	if config == nil {
		return nil, errors.Wrap(errors.New("empty configuration returned"), errGetPC)
	}

	return &external{client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for
// DevOpsPermissionMapping resources.
type external struct {
	// client is used to interact with the SonarQube permission mapping API.
	client iam.DevOpsPermissionMappingsClient
}

// Observe observes the permission mapping of the role.
//
// The mapping of a built-in role cannot be deleted, so it is reported as
// absent while the resource is being deleted, which lets the deletion complete.
func (c *external) Observe(ctx context.Context, mapping *v1alpha1.DevOpsPermissionMapping) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(mapping)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	params := &mapping.Spec.ForProvider
	if externalName != params.Role {
		return managed.ExternalObservation{}, errors.Errorf(errExternalName, externalName, params.Role)
	}

	observed, found, err := iam.FindDevOpsPermissionMapping(ctx, c.client, params.Platform, params.Role)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveMapping)
	}

	if !found || (observed.BaseRole && meta.WasDeleted(mapping)) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	mapping.Status.AtProvider = iam.GenerateDevOpsPermissionMappingObservation(observed)
	mapping.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  iam.IsDevOpsPermissionMappingUpToDate(params, observed),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create creates the permission mapping of a custom role. Built-in roles always
// have a mapping, so they are never created.
func (c *external) Create(ctx context.Context, mapping *v1alpha1.DevOpsPermissionMapping) (managed.ExternalCreation, error) {
	mapping.SetConditions(xpv1.Creating())

	params := &mapping.Spec.ForProvider

	_, resp, err := c.client.Create(ctx, params.Platform, iam.GenerateCreateDevOpsPermissionMappingOptions(params)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateMapping)
	}

	meta.SetExternalName(mapping, params.Role)

	return managed.ExternalCreation{}, nil
}

// Update sets the permissions of the mapping to the desired set.
func (c *external) Update(ctx context.Context, mapping *v1alpha1.DevOpsPermissionMapping) (managed.ExternalUpdate, error) {
	params := &mapping.Spec.ForProvider

	_, resp, err := c.client.Update(ctx, params.Platform, params.Role, iam.GenerateUpdateDevOpsPermissionMappingOptions(params)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateMapping)
}

// Delete deletes the permission mapping of a custom role. The mapping of a
// built-in role is left unchanged.
func (c *external) Delete(ctx context.Context, mapping *v1alpha1.DevOpsPermissionMapping) (managed.ExternalDelete, error) {
	mapping.SetConditions(xpv1.Deleting())

	if mapping.Status.AtProvider.BaseRole {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(ctx, mapping.Spec.ForProvider.Platform, mapping.Spec.ForProvider.Role) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	// The client resends a DELETE after a transient failure, so a 404 can mean
	// an earlier attempt succeeded.
	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteMapping)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}
