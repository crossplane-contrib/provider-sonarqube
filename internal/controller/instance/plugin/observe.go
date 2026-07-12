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

package plugin

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

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

	installedList, resp, err := c.client.Installed(ctx, nil) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObservePlugin)
	}

	found := instance.FindInstalledPlugin(installedList.Plugins, externalName)
	if found == nil {
		return c.observePending(ctx, managedPlugin, externalName)
	}

	if managedPlugin.DeletionTimestamp != nil {
		obs, isPendingRemoval, pendingErr := c.observePendingRemoval(ctx, managedPlugin, externalName)
		if pendingErr != nil || isPendingRemoval {
			return obs, pendingErr
		}
	}

	return c.observeInstalled(ctx, managedPlugin, found)
}

// observeInstalled handles the full observation path for a plugin that
// is already present in the installed list.
func (c *external) observeInstalled(
	ctx context.Context,
	managedPlugin *v1alpha1.Plugin,
	found *sonar.PluginInstalled,
) (managed.ExternalObservation, error) {
	managedPlugin.Status.AtProvider = instance.GeneratePluginObservation(found)

	pluginsToUpdate, resp, err := c.client.Updates(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{ResourceExists: true},
			errors.Wrap(err, "cannot get plugin updates from SonarQube")
	}

	// IsPluginUpdatable returns true when updates are available, so IsLatest
	// is the inverse: true means already at the latest version.
	isUpdatable := instance.IsPluginUpdatable(&pluginsToUpdate.Plugins, found.Key)
	managedPlugin.Status.AtProvider.IsLatest = !isUpdatable

	if isUpdatable {
		obs, isPendingUpdate, pendingErr := c.observePendingUpdate(ctx, managedPlugin, found.Key)
		if pendingErr != nil || isPendingUpdate {
			return obs, pendingErr
		}
	}

	former := managedPlugin.Spec.ForProvider.DeepCopy()
	instance.LateInitializePlugin(&managedPlugin.Spec.ForProvider, &managedPlugin.Status.AtProvider)
	managedPlugin.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsPluginUpToDate(&managedPlugin.Spec.ForProvider, &managedPlugin.Status.AtProvider),
		ResourceLateInitialized: instance.IsPluginLateInitialized(former, &managedPlugin.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// observePending checks whether the plugin is queued for installation
// (pending a SonarQube restart). It returns a not-exists observation
// when the key is absent from the pending list.
func (c *external) observePending(ctx context.Context, managedPlugin *v1alpha1.Plugin, key string) (managed.ExternalObservation, error) {
	pending, resp, err := c.client.Pending(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObservePlugin)
	}

	pendingPlugin := instance.FindPendingInstallPlugin(pending, key)
	if pendingPlugin == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	managedPlugin.Status.AtProvider = instance.GeneratePluginObservationFromPending(pendingPlugin)
	managedPlugin.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// observePendingRemoval checks whether the plugin is queued for removal
// (pending a SonarQube restart). It returns (obs, true, nil) when found,
// signalling the caller to return obs immediately.
func (c *external) observePendingRemoval(ctx context.Context, managedPlugin *v1alpha1.Plugin, key string) (managed.ExternalObservation, bool, error) {
	return c.observePendingOp(ctx, managedPlugin, key, instance.IsPluginPendingRemoval, xpv1.Deleting())
}

// observePendingUpdate checks whether the plugin is queued for an upgrade
// (pending a SonarQube restart). It returns (obs, true, nil) when found,
// signalling the caller to return obs immediately.
func (c *external) observePendingUpdate(ctx context.Context, managedPlugin *v1alpha1.Plugin, key string) (managed.ExternalObservation, bool, error) {
	return c.observePendingOp(ctx, managedPlugin, key, instance.IsPluginPendingUpdate, xpv1.Available())
}

// observePendingOp is the shared core for observePendingRemoval
// and observePendingUpdate. It calls Pending(), uses checkFn to
// test whether key is in the relevant list, then marks the plugin
// pending and applies cond. Returns (obs, true, nil) when found,
// (zero, false, nil) otherwise.
func (c *external) observePendingOp(
	ctx context.Context,
	managedPlugin *v1alpha1.Plugin,
	key string,
	checkFn func(*sonar.PluginsPending, string) bool,
	cond xpv1.Condition,
) (managed.ExternalObservation, bool, error) {
	pendingList, resp, err := c.client.Pending(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, false, errors.Wrap(err, errObservePlugin)
	}

	if !checkFn(pendingList, key) {
		return managed.ExternalObservation{}, false, nil
	}

	managedPlugin.Status.AtProvider.Pending = true
	managedPlugin.SetConditions(cond)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, true, nil
}
