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

// Package plugin provides a controller for Plugin resources.
package plugin

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotPlugin indicates the managed resource is not a Plugin.
	errNotPlugin = "managed resource is not a Plugin custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errObservePlugin indicates plugin observation from SonarQube failed.
	errObservePlugin = "cannot observe SonarQube Plugin"
	// errInstallPlugin indicates plugin installation in SonarQube failed.
	errInstallPlugin = "cannot install SonarQube Plugin"
	// errUninstallPlugin indicates plugin removal from SonarQube failed.
	errUninstallPlugin = "cannot uninstall SonarQube Plugin"
)

// SetupGated adds a controller that reconciles Plugin managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Plugin controller"))
		}
	}, v1alpha1.PluginGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Plugin managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.PluginGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewPluginsClient,
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
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.PluginList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.PluginList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.PluginGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Plugin{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.PluginsClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube plugins client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	managedPlugin, isValid := managedResource.(*v1alpha1.Plugin)
	if !isValid {
		return nil, errors.New(errNotPlugin)
	}

	err := c.usage.Track(ctx, managedPlugin)
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

// external implements the ExternalClient interface for Plugin resources.
type external struct {
	// client interacts with the SonarQube Plugins API.
	client instance.PluginsClient
}

// Observe checks whether the plugin is installed in SonarQube and
// populates status.atProvider.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	managedPlugin, ok := mg.(*v1alpha1.Plugin)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPlugin)
	}

	externalName := meta.GetExternalName(managedPlugin)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	installedList, resp, err := c.client.Installed(nil) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObservePlugin)
	}

	found := instance.FindInstalledPlugin(installedList.Plugins, externalName)
	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	managedPlugin.Status.AtProvider = instance.GeneratePluginObservation(found)

	pluginsToUpdate, resp, err := c.client.Updates() //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{
			ResourceExists: true,
		}, errors.Wrap(err, "cannot get plugin updates from SonarQube")
	}

	// IsPluginUpdatable returns true when updates are available, so IsLatest is
	// the inverse: true means already at the latest version.
	managedPlugin.Status.AtProvider.IsLatest = !instance.IsPluginUpdatable(&pluginsToUpdate.Plugins, found.Key)

	former := managedPlugin.Spec.ForProvider.DeepCopy()
	instance.LateInitializePlugin(&managedPlugin.Spec.ForProvider, &managedPlugin.Status.AtProvider)

	managedPlugin.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsPluginUpToDate(&managedPlugin.Spec.ForProvider, &managedPlugin.Status.AtProvider),
		ResourceLateInitialized: !instance.IsPluginLateInitialized(former, &managedPlugin.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create installs the plugin in SonarQube and sets the external name to
// the plugin key.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	managedPlugin, ok := mg.(*v1alpha1.Plugin)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPlugin)
	}

	managedPlugin.Status.SetConditions(xpv1.Creating())

	resp, err := c.client.Install(&sonar.PluginsInstallOptions{Key: managedPlugin.Spec.ForProvider.Key}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errInstallPlugin)
	}

	meta.SetExternalName(managedPlugin, managedPlugin.Spec.ForProvider.Key)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Update is responsible for upgrading the plugin in SonarQube when
// a new version is available and AutoUpdate is not explicitly disabled.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	plugin, ok := mg.(*v1alpha1.Plugin)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPlugin)
	}

	// Use external name as the identifier to update the resource
	externalName := meta.GetExternalName(plugin)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New("cannot update SonarQube Plugin without external name")
	}

	resp, err := c.client.Update(&sonar.PluginsUpdateOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update SonarQube Plugin")
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Delete uninstalls the plugin from SonarQube.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	managedPlugin, ok := mg.(*v1alpha1.Plugin)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPlugin)
	}

	managedPlugin.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(managedPlugin)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Uninstall(&sonar.PluginsUninstallOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUninstallPlugin)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
