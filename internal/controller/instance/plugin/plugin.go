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

	// errObservePlugin indicates plugin observation from SonarQube failed.
	errObservePlugin = "cannot observe SonarQube Plugin"
	// errInstallPlugin indicates plugin installation in SonarQube failed.
	errInstallPlugin = "cannot install SonarQube Plugin"
	// errUninstallPlugin indicates plugin removal from SonarQube failed.
	errUninstallPlugin = "cannot uninstall SonarQube Plugin"
	// errCancelPluginInstall indicates cancelling a pending installation
	// failed.
	errCancelPluginInstall = "cannot cancel the pending installation of SonarQube Plugin"
	// errPluginInstallNotCancellable is returned when deleting a plugin whose
	// installation is pending alongside other plugin operations.
	errPluginInstallNotCancellable = "the installation of SonarQube Plugin %s is pending a SonarQube restart alongside other plugin operations, and SonarQube can only cancel them all together: restart SonarQube or resolve the other pending operations"
)

// SetupGated adds a controller that reconciles Plugin managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o helpers.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Plugin controller"))
		}
	}, v1alpha1.PluginGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Plugin managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(v1alpha1.PluginGroupKind)

	options := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*v1alpha1.Plugin](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewPluginsClient,
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(mg *v1alpha1.Plugin) string {
			return mg.Spec.ForProvider.Key
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, opts, &v1alpha1.PluginList{})
	if err != nil {
		return err
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
func (c *connector) Connect(ctx context.Context, managedPlugin *v1alpha1.Plugin) (managed.TypedExternalClient[*v1alpha1.Plugin], error) {
	err := c.usage.Track(ctx, managedPlugin)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, managedPlugin)
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

// Create installs the plugin in SonarQube and sets the external name to
// the plugin key.
func (c *external) Create(ctx context.Context, managedPlugin *v1alpha1.Plugin) (managed.ExternalCreation, error) {
	managedPlugin.Status.SetConditions(xpv1.Creating())

	resp, err := c.client.Install(ctx, &sonar.PluginsInstallOptions{Key: managedPlugin.Spec.ForProvider.Key}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errInstallPlugin)
	}

	meta.SetExternalName(managedPlugin, managedPlugin.Spec.ForProvider.Key)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update upgrades the plugin in SonarQube. If the upgrade is already
// queued (pending a restart), the Update call is skipped to avoid
// re-queuing the same operation.
func (c *external) Update(ctx context.Context, plugin *v1alpha1.Plugin) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(plugin)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New("cannot update SonarQube Plugin without external name")
	}

	pendingList, pendingResp, err := c.client.Pending(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(pendingResp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update SonarQube Plugin")
	}

	if instance.IsPluginPendingUpdate(pendingList, externalName) {
		return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
	}

	resp, err := c.client.Update(ctx, &sonar.PluginsUpdateOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update SonarQube Plugin")
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete uninstalls the plugin from SonarQube. If the removal is already
// queued (pending a restart), the Uninstall call is skipped to avoid
// re-queuing the same operation. A plugin whose installation is still
// pending is not installed yet, so it cannot be uninstalled: its installation
// is cancelled instead. SonarQube can only cancel every pending operation at
// once, so this is only done when no other plugin operation is pending.
func (c *external) Delete(ctx context.Context, managedPlugin *v1alpha1.Plugin) (managed.ExternalDelete, error) {
	managedPlugin.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(managedPlugin)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	pendingList, pendingResp, err := c.client.Pending(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(pendingResp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUninstallPlugin)
	}

	if instance.IsPluginPendingRemoval(pendingList, externalName) {
		return managed.ExternalDelete{}, nil
	}

	if instance.FindPendingInstallPlugin(pendingList, externalName) != nil {
		return managed.ExternalDelete{}, c.cancelPendingInstall(ctx, pendingList, externalName)
	}

	resp, err := c.client.Uninstall(ctx, &sonar.PluginsUninstallOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
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

// cancelPendingInstall cancels the pending installation of the plugin with
// the given key, provided it is the only pending plugin operation.
func (c *external) cancelPendingInstall(ctx context.Context, pendingList *sonar.PluginsPending, key string) error {
	if !instance.IsOnlyPendingInstall(pendingList, key) {
		return errors.Errorf(errPluginInstallNotCancellable, key)
	}

	resp, err := c.client.CancelAll(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	return errors.Wrap(err, errCancelPluginInstall)
}
