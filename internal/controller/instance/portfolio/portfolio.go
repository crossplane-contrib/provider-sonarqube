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

// Package portfolio provides a controller for Portfolio resources.
package portfolio

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotPortfolio indicates the managed resource is not a Portfolio.
	errNotPortfolio = "managed resource is not a Portfolio custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errExternalNameNotSet indicates the external name annotation is missing.
	errExternalNameNotSet = "external name is not set for Portfolio resource %s"

	// errObservePortfolio indicates portfolio observation from SonarQube failed.
	errObservePortfolio = "cannot observe SonarQube Portfolio"
	// errCreatePortfolio indicates portfolio creation in SonarQube failed.
	errCreatePortfolio = "cannot create SonarQube Portfolio"
	// errUpdatePortfolio indicates portfolio update in SonarQube failed.
	errUpdatePortfolio = "cannot update SonarQube Portfolio"
	// errDeletePortfolio indicates portfolio deletion in SonarQube failed.
	errDeletePortfolio = "cannot delete SonarQube Portfolio"
	// errSetSelectionMode indicates selection mode configuration failed.
	errSetSelectionMode = "cannot set selection mode for SonarQube Portfolio"

	// selectionModeNone is the default portfolio selection mode.
	selectionModeNone = "NONE"
	// selectionModeManual is the manual project selection mode.
	selectionModeManual = "MANUAL"
	// selectionModeRegexp is the regexp-based project selection mode.
	selectionModeRegexp = "REGEXP"
	// selectionModeRemaining is the remaining-projects selection mode.
	selectionModeRemaining = "REMAINING"
	// selectionModeTags is the tag-based project selection mode.
	selectionModeTags = "TAGS"
)

// SetupGated adds a controller that reconciles Portfolio managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Portfolio controller"))
		}
	}, v1alpha1.PortfolioGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Portfolio managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.PortfolioGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewPortfoliosClient,
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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.PortfolioList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.PortfolioList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.PortfolioGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Portfolio{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its
// Connect method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.PortfoliosClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube portfolios client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	portfolio, isValid := managedResource.(*v1alpha1.Portfolio)
	if !isValid {
		return nil, errors.New(errNotPortfolio)
	}

	err := c.usage.Track(ctx, portfolio)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, portfolio)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// external implements the ExternalClient interface for Portfolio resources.
type external struct {
	client instance.PortfoliosClient
}

// Observe checks if the Portfolio exists in SonarQube and whether it
// matches the desired state.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	portfolio, ok := mg.(*v1alpha1.Portfolio)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPortfolio)
	}

	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	result, resp, err := c.client.Show(&sonar.ViewsShowOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObservePortfolio)
	}

	defer helpers.CloseBody(resp)

	portfolio.Status.AtProvider = instance.GeneratePortfolioObservation(&result.Portfolio)
	portfolio.SetConditions(xpv1.Available())

	former := portfolio.Spec.ForProvider.DeepCopy()
	instance.LateInitializePortfolio(&portfolio.Spec.ForProvider, &portfolio.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsPortfolioUpToDate(&portfolio.Spec.ForProvider, &portfolio.Status.AtProvider),
		ResourceLateInitialized: instance.IsPortfolioLateInitialized(former, &portfolio.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the Portfolio in SonarQube and sets the external
// name to the portfolio key.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	portfolio, ok := mg.(*v1alpha1.Portfolio)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPortfolio)
	}

	portfolio.Status.SetConditions(xpv1.Creating())

	resp, err := c.client.Create(&sonar.ViewsCreateOptions{ //nolint:bodyclose // closed via helpers.CloseBody
		Key:         portfolio.Spec.ForProvider.Key,
		Name:        portfolio.Spec.ForProvider.Name,
		Description: portfolio.Spec.ForProvider.Description,
		Visibility:  portfolio.Spec.ForProvider.Visibility,
	})
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePortfolio)
	}

	err = c.setSelectionMode(portfolio.Spec.ForProvider.Key, &portfolio.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSetSelectionMode)
	}

	meta.SetExternalName(portfolio, portfolio.Spec.ForProvider.Key)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update updates the Portfolio name, description, and selection
// mode in SonarQube.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	portfolio, ok := mg.(*v1alpha1.Portfolio)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPortfolio)
	}

	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.Errorf(errExternalNameNotSet, portfolio.Name)
	}

	resp, err := c.client.Update(&sonar.ViewsUpdateOptions{ //nolint:bodyclose // closed via helpers.CloseBody
		Key:         externalName,
		Name:        portfolio.Spec.ForProvider.Name,
		Description: portfolio.Spec.ForProvider.Description,
	})
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePortfolio)
	}

	err = c.setSelectionMode(externalName, &portfolio.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errSetSelectionMode)
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete deletes the Portfolio from SonarQube.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	portfolio, ok := mg.(*v1alpha1.Portfolio)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPortfolio)
	}

	portfolio.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(&sonar.ViewsDeleteOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePortfolio)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// setSelectionMode sets the project selection mode for the portfolio
// in SonarQube.
func (c *external) setSelectionMode(key string, spec *v1alpha1.PortfolioParameters) error {
	mode := spec.SelectionMode
	if mode == "" {
		mode = selectionModeNone
	}

	switch mode {
	case selectionModeNone:
		resp, err := c.client.SetNoneMode(&sonar.ViewsSetNoneModeOptions{Portfolio: key}) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set NONE selection mode")
	case selectionModeManual:
		resp, err := c.client.SetManualMode(&sonar.ViewsSetManualModeOptions{Portfolio: key}) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set MANUAL selection mode")
	case selectionModeRegexp:
		resp, err := c.client.SetRegexpMode(&sonar.ViewsSetRegexpModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Regexp:    spec.Regexp,
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set REGEXP selection mode")
	case selectionModeRemaining:
		resp, err := c.client.SetRemainingProjectsMode(&sonar.ViewsSetRemainingProjectsModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set REMAINING selection mode")
	case selectionModeTags:
		resp, err := c.client.SetTagsMode(&sonar.ViewsSetTagsModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Tags:      spec.Tags,
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set TAGS selection mode")
	default:
		return nil
	}
}
