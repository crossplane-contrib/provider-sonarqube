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

// Package license provides a controller for License resources.
package license

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotLicense indicates managed resource is not a License.
	errNotLicense = "managed resource is not a License custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errGetLicense indicates the SonarQube license could not be retrieved.
	errGetLicense = "cannot get SonarQube license"
	// errSetLicense indicates the SonarQube license could not be set.
	errSetLicense = "cannot set SonarQube license"
	// errUnsetLicense indicates the SonarQube license could not be unset.
	errUnsetLicense = "cannot unset SonarQube license"
	// errGetLicenseKey indicates the license key could not be resolved
	// from its configured source (secret reference or endpoint).
	errGetLicenseKey = "cannot resolve license key"

	// externalLicenseName is the fixed external name for the
	// singleton license resource.
	externalLicenseName = "license"

	// connectionDetailLicenseKeyKey is the connection secret key under which
	// the license key that was last applied to SonarQube is stored.
	connectionDetailLicenseKeyKey = "licenseKey"
)

// SetupGated adds a controller that reconciles License resources,
// registered behind a gate so it only runs when the gate is opened.
func SetupGated(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.LicenseGroupKind)

	opts.Gate.Register(func() {
		err := Setup(mgr, opts, name)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup License controller"))
		}
	}, v1alpha1.LicenseGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles License resources.
func Setup(mgr ctrl.Manager, opts controller.Options, name string) error {
	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewLicensesClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor marked deprecated but not yet replaced alternative in controller-runtime, APIRecorder still required for recording events.*/)),
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
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.LicenseList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.LicenseList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.LicenseGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.License{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.LicensesClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	license, isValid := managedResource.(*v1alpha1.License)
	if !isValid {
		return nil, errors.New(errNotLicense)
	}

	err := c.usage.Track(ctx, license)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{
		kube:   c.kube,
		client: c.newServiceFn(*config),
	}, nil
}

// external implements the ExternalClient interface for License resources.
type external struct {
	kube   client.Client
	client instance.LicensesClient
}

// Create creates the License external resource using the SonarQube API.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	license, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLicense)
	}

	license.SetConditions(xpv1.Creating())

	licenseKey, err := c.resolveLicenseKey(ctx, license)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetLicenseKey)
	}

	resp, err := c.client.Set(ctx, &sonar.LicenseSetOptions{License: ptr.Deref(licenseKey, "")}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSetLicense)
	}

	meta.SetExternalName(license, externalLicenseName)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailLicenseKeyKey: []byte(ptr.Deref(licenseKey, "")),
		},
	}, nil
}

// Update updates the License corresponding external resource to reflect the
// desired state of the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	license, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLicense)
	}

	licenseKey, err := c.resolveLicenseKey(ctx, license)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetLicenseKey)
	}

	resp, err := c.client.Set(ctx, &sonar.LicenseSetOptions{License: ptr.Deref(licenseKey, "")}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errSetLicense)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailLicenseKeyKey: []byte(ptr.Deref(licenseKey, "")),
		},
	}, nil
}

// Delete deletes the License corresponding external resource using the
// SonarQube API.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	license, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLicense)
	}

	license.SetConditions(xpv1.Deleting())

	resp, err := c.client.UnsetLicense(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUnsetLicense)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}
