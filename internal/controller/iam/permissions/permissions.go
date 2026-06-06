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

// Package permissions provides a controller for Permissions resources.
package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
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
)

const (
	// errNotPermissions indicates managed resource is not a Permissions resource.
	errNotPermissions = "managed resource is not a Permissions custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errExternalNameNotSet indicates the external name is missing.
	errExternalNameNotSet = "external name is not set for Permissions resource %s"
	// errObservePermissions indicates Permissions observation failed.
	errObservePermissions = "cannot observe Permissions"
	// errCreatePermissions indicates Permissions creation failed.
	errCreatePermissions = "cannot create Permissions"
	// errUpdatePermissions indicates Permissions update failed.
	errUpdatePermissions = "cannot update Permissions"
	// errDeletePermissions indicates Permissions deletion failed.
	errDeletePermissions = "cannot delete Permissions"

	// subjectTypeGroup identifies group-scoped permission resources.
	subjectTypeGroup = "group"
	// subjectTypeUser identifies user-scoped permission resources.
	subjectTypeUser = "user"

	// minExternalNameParts is the minimum number of colon-separated parts
	// in a valid external name.
	minExternalNameParts = 2
	// maxExternalNameParts is the maximum number of colon-separated parts
	// allowed when splitting an external name (project key may contain colons).
	maxExternalNameParts = 3
)

// SetupGated adds a controller that reconciles Permissions managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Permissions controller"))
		}
	}, v1alpha1.PermissionsGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Permissions managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.PermissionsGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewPermissionsClient,
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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.PermissionsList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.PermissionsList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.PermissionsGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Permissions{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.PermissionsClient
}

// Connect produces an ExternalClient by resolving the ProviderConfig
// credentials.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	permissions, isValid := managedResource.(*v1alpha1.Permissions)
	if !isValid {
		return nil, errors.New(errNotPermissions)
	}

	err := c.usage.Track(ctx, permissions)
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

// An external observes, then creates, updates, or deletes external
// Permissions resources.
type external struct {
	// client is used to interact with SonarQube Permissions API.
	client iam.PermissionsClient
}

// Create adds all desired permissions to the group or user, then sets the
// external name.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	permissions, ok := mg.(*v1alpha1.Permissions)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPermissions)
	}

	permissions.SetConditions(xpv1.Creating())

	err := c.applyPermissions(permissions.Spec.ForProvider, permissions.Spec.ForProvider.Permissions, nil)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePermissions)
	}

	meta.SetExternalName(permissions, buildExternalName(&permissions.Spec.ForProvider))

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete removes all observed permissions from the group or user.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	permissions, ok := mg.(*v1alpha1.Permissions)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPermissions)
	}

	permissions.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(permissions)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := c.applyPermissions(permissions.Spec.ForProvider, nil, permissions.Status.AtProvider.Permissions)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePermissions)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for stateless clients.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// parseExternalName decodes an external name of the form:
//
//	"group:<name>" / "group:<name>:<projectKey>"
//	"user:<login>" / "user:<login>:<projectKey>"
func parseExternalName(externalName string) (subjectType, subject, projectKey string, err error) {
	parts := strings.SplitN(externalName, ":", maxExternalNameParts)
	if len(parts) < minExternalNameParts {
		return "", "", "", fmt.Errorf("invalid external name format: %q (expected <type>:<subject>[:<projectKey>])", externalName)
	}

	subjectType = parts[0]
	if subjectType != subjectTypeGroup && subjectType != subjectTypeUser {
		return "", "", "", fmt.Errorf("unknown subject type %q in external name %q", subjectType, externalName)
	}

	subject = parts[1]
	if len(parts) == maxExternalNameParts {
		projectKey = parts[2]
	}

	return subjectType, subject, projectKey, nil
}

// buildExternalName constructs the external name from spec fields.
func buildExternalName(spec *v1alpha1.PermissionsParameters) string {
	var subjectType, subject string

	if spec.GroupName != nil && *spec.GroupName != "" {
		subjectType = subjectTypeGroup
		subject = *spec.GroupName
	} else if spec.Login != nil {
		subjectType = subjectTypeUser
		subject = *spec.Login
	}

	if spec.ProjectKey != nil && *spec.ProjectKey != "" {
		return fmt.Sprintf("%s:%s:%s", subjectType, subject, *spec.ProjectKey)
	}

	return fmt.Sprintf("%s:%s", subjectType, subject)
}
