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

package permissionstemplate

import (
	"context"
	"fmt"

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
	errNotPermissionsTemplate = "managed resource is not a PermissionsTemplate custom resource"
	errTrackPCUsage           = "cannot track ProviderConfig usage"
	errGetPC                  = "cannot get ProviderConfig"
	errGetCPC                 = "cannot get ClusterProviderConfig"
	errGetCreds               = "cannot get credentials"
)

// SetupGated adds a controller that reconciles PermissionsTemplate managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup PermissionsTemplate controller"))
		}
	}, v1alpha1.PermissionsTemplateGroupVersionKind)

	return nil
}

// Setup configures and registers the PermissionsTemplate managed reconciler.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.PermissionsTemplateGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewPermissionsTemplatesClient,
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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.PermissionsTemplateList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.PermissionsTemplateList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.PermissionsTemplateGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.PermissionsTemplate{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.PermissionsTemplatesClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	permissionstemplate, isValid := managedResource.(*v1alpha1.PermissionsTemplate)
	if !isValid {
		return nil, errors.New(errNotPermissionsTemplate)
	}

	err := c.usage.Track(ctx, permissionstemplate)
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

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// client is used to interact with SonarQube PermissionsTemplate API
	client iam.PermissionsTemplatesClient
}

// Create creates the PermissionsTemplate external resource using the SonarQube API, based on the desired state of the managed resource. It returns an ExternalCreation which may include connection details to be stored in a secret.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	permissionsTemplate, ok := mg.(*v1alpha1.PermissionsTemplate)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPermissionsTemplate)
	}

	permissionsTemplate.SetConditions(xpv1.Creating())

	creationOptions := iam.GeneratePermissionsTemplateCreationOptions(&permissionsTemplate.Spec.ForProvider)

	createdTemplate, resp, err := c.client.CreateTemplate(creationOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create PermissionsTemplate")
	}

	meta.SetExternalName(permissionsTemplate, createdTemplate.PermissionTemplate.ID)

	return managed.ExternalCreation{}, nil
}

// Delete deletes the PermissionsTemplate external resource using the SonarQube API, based on the desired state of the managed resource. It returns an ExternalDelete which may include connection details to be stored in a secret.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	permissionsTemplate, ok := mg.(*v1alpha1.PermissionsTemplate)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPermissionsTemplate)
	}

	permissionsTemplate.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(permissionsTemplate)
	if externalName == "" {
		return managed.ExternalDelete{}, fmt.Errorf("external name is not set for PermissionsTemplate %s", permissionsTemplate.Name)
	}

	deleteOptions := iam.GeneratePermissionsTemplateDeleteOptions(externalName)
	resp, err := c.client.DeleteTemplate(deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody

	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete PermissionsTemplate")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect performs external client cleanup; this client has no persistent resources.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
