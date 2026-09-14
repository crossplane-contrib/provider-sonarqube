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

// Package newcodeperiod provides a controller for NewCodePeriod resources.
package newcodeperiod

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
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
	// errNotNewCodePeriod indicates managed resource is not NewCodePeriod.
	errNotNewCodePeriod = "managed resource is not a NewCodePeriod custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
)

// SetupGated adds a controller that reconciles NewCodePeriod managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup NewCodePeriod controller"))
		}
	}, v1alpha1.NewCodePeriodGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles NewCodePeriod managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.NewCodePeriodGroupKind)

	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewNewCodePeriodsClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		reconcilerOpts = append(reconcilerOpts, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		reconcilerOpts = append(reconcilerOpts, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		reconcilerOpts = append(reconcilerOpts, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.NewCodePeriodList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.NewCodePeriodList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.NewCodePeriodGroupVersionKind), reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.NewCodePeriod{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.NewCodePeriodsClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	newCodePeriod, ok := managedResource.(*v1alpha1.NewCodePeriod)
	if !ok {
		return nil, errors.New(errNotNewCodePeriod)
	}

	err := c.usage.Track(ctx, newCodePeriod)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	m, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, m)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{newCodePeriodsClient: svc}, nil
}

// external implements the managed.ExternalClient interface for NewCodePeriod
// resources.
type external struct {
	// newCodePeriodsClient is used to interact with SonarQube New Code Periods API
	newCodePeriodsClient instance.NewCodePeriodsClient
}

// Observe checks if the external resource exists and if it matches the desired
// state specified by the managed resource. It returns an ExternalObservation
// indicating whether the resource exists,
// whether it is up to date, and any connection details or errors.
func (c *external) Observe(ctx context.Context, managedResource resource.Managed) (managed.ExternalObservation, error) {
	newCodePeriod, ok := managedResource.(*v1alpha1.NewCodePeriod)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNewCodePeriod)
	}

	// Deleting: the instance-wide default new code period is reset on delete;
	// mark the external resource as non-existent so the managed reconciler can
	// remove the finalizer and allow the CR to be deleted.
	if !newCodePeriod.DeletionTimestamp.IsZero() {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	sonarNewCodePeriod, resp, err := c.newCodePeriodsClient.Show(ctx, instance.GenerateInstanceNewCodePeriodsShowOptions()) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get instance new code period")
	}

	observation := instance.GenerateInstanceNewCodePeriodObservation(sonarNewCodePeriod)
	newCodePeriod.Status.AtProvider = observation

	newCodePeriod.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.AreInstanceNewCodePeriodsUpToDate(&newCodePeriod.Spec.ForProvider, &observation),
	}, nil
}

// Create sets the instance-wide default new code period based on the desired
// state in the managed resource. It should return an error if the creation
// failed, or nil if it succeeded.
func (c *external) Create(ctx context.Context, managedResource resource.Managed) (managed.ExternalCreation, error) {
	newCodePeriod, ok := managedResource.(*v1alpha1.NewCodePeriod)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNewCodePeriod)
	}

	newCodePeriod.SetConditions(xpv1.Creating())

	resp, err := c.newCodePeriodsClient.Set(ctx, instance.GenerateInstanceNewCodePeriodsSetOptions(&newCodePeriod.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to set instance new code period")
	}

	return managed.ExternalCreation{}, nil
}

// Update sets the instance-wide default new code period to match the desired
// state in the managed resource. It returns an error if the update failed, or
// nil if it succeeded.
func (c *external) Update(ctx context.Context, managedResource resource.Managed) (managed.ExternalUpdate, error) {
	newCodePeriod, ok := managedResource.(*v1alpha1.NewCodePeriod)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNewCodePeriod)
	}

	resp, err := c.newCodePeriodsClient.Set(ctx, instance.GenerateInstanceNewCodePeriodsSetOptions(&newCodePeriod.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update instance new code period")
	}

	return managed.ExternalUpdate{}, nil
}

// Delete unsets the instance-wide default new code period in SonarQube based on
// the given managed resource. It returns any error that occurred during
// deletion.
func (c *external) Delete(ctx context.Context, managedResource resource.Managed) (managed.ExternalDelete, error) {
	newCodePeriod, ok := managedResource.(*v1alpha1.NewCodePeriod)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotNewCodePeriod)
	}

	newCodePeriod.SetConditions(xpv1.Deleting())

	resp, err := c.newCodePeriodsClient.Unset(ctx, instance.GenerateInstanceNewCodePeriodsUnsetOptions()) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to unset instance new code period")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is called when the external resource is disconnected from the
// provider. There is no client cleanup to perform, so this method returns nil.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
