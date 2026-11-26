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

// Package instanceconfiguration provides a controller for
// InstanceConfiguration resources.
package instanceconfiguration

import (
	"context"
	stderrors "errors"

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
	// errObserveOption prefixes the error of an option that could not be
	// observed.
	errObserveOption = "cannot observe instance %s"
	// errApplyOption prefixes the error of an option that could not be
	// applied.
	errApplyOption = "cannot set instance %s"
)

// SetupGated adds a controller that reconciles InstanceConfiguration managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o helpers.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup InstanceConfiguration controller"))
		}
	}, v1alpha1.InstanceConfigurationGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles InstanceConfiguration managed
// resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(v1alpha1.InstanceConfigurationGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*v1alpha1.InstanceConfiguration](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newClientsFn: instance.NewInstanceConfigurationClients,
		}),
		managed.WithDeterministicExternalName(true),
	)

	err := helpers.AddMRStateRecorder(mgr, opts, &v1alpha1.InstanceConfigurationList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.InstanceConfigurationGroupVersionKind), reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.InstanceConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newClientsFn func(config common.Config) instance.InstanceConfigurationClients
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) (managed.TypedExternalClient[*v1alpha1.InstanceConfiguration], error) {
	err := c.usage.Track(ctx, instanceConfiguration)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, instanceConfiguration)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{clients: c.newClientsFn(*config), options: options}, nil
}

// external implements the managed.TypedExternalClient interface for
// InstanceConfiguration resources.
type external struct {
	// clients are used to interact with the SonarQube endpoints of each
	// option.
	clients instance.InstanceConfigurationClients
	// options are the options managed by the InstanceConfiguration.
	options []option
}

// Observe reads the options set in the spec from SonarQube. The instance
// configuration always exists, so the resource is reported as existing and
// only the set options decide whether it is up to date.
func (c *external) Observe(ctx context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) (managed.ExternalObservation, error) {
	// Deleting: the options are left in place, so report the external
	// resource as gone to let the managed reconciler remove the finalizer.
	if !instanceConfiguration.DeletionTimestamp.IsZero() {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	params := &instanceConfiguration.Spec.ForProvider
	observation := v1alpha1.InstanceConfigurationObservation{}
	upToDate := true

	for _, opt := range c.options {
		if !opt.isSet(params) {
			continue
		}

		err := opt.observe(ctx, &c.clients, &observation)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrapf(err, errObserveOption, opt.name)
		}

		upToDate = upToDate && opt.isUpToDate(params, &observation)
	}

	instanceConfiguration.Status.AtProvider = observation

	instanceConfiguration.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create applies the options set in the spec. Observe always reports the
// instance configuration as existing, so this is only a fallback.
func (c *external) Create(ctx context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) (managed.ExternalCreation, error) {
	instanceConfiguration.SetConditions(xpv1.Creating())

	return managed.ExternalCreation{}, c.applyOutdated(ctx, instanceConfiguration)
}

// Update applies the options set in the spec that differ from the values
// observed in SonarQube.
func (c *external) Update(ctx context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, c.applyOutdated(ctx, instanceConfiguration)
}

// Delete leaves the options at their current value in SonarQube: they are
// instance-wide and have no unset state to return to.
func (c *external) Delete(_ context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) (managed.ExternalDelete, error) {
	instanceConfiguration.SetConditions(xpv1.Deleting())

	return managed.ExternalDelete{}, nil
}

// Disconnect is called when the external resource is disconnected from the
// provider. There is no client cleanup to perform, so this method returns nil.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// applyOutdated applies every option set in the spec whose last observation,
// recorded by Observe in status.atProvider, does not match it. Options are
// independent, so a failing option does not prevent the others from being
// applied, and every failure is reported.
func (c *external) applyOutdated(ctx context.Context, instanceConfiguration *v1alpha1.InstanceConfiguration) error {
	params := &instanceConfiguration.Spec.ForProvider
	observation := &instanceConfiguration.Status.AtProvider

	var errs []error

	for _, opt := range c.options {
		if !opt.isSet(params) || opt.isUpToDate(params, observation) {
			continue
		}

		err := opt.apply(ctx, &c.clients, params)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, errApplyOption, opt.name))
		}
	}

	return stderrors.Join(errs...)
}
