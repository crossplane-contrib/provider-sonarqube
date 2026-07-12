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

package instance

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// PluginsClient handles SonarQube Plugin API operations.
type PluginsClient interface {
	// Install installs a plugin by key.
	Install(ctx context.Context, opt *sonar.PluginsInstallOptions) (resp *http.Response, err error)
	// Installed returns the list of all installed plugins.
	Installed(ctx context.Context, opt *sonar.PluginsInstalledOptions) (v *sonar.PluginsInstalled, resp *http.Response, err error)
	// Pending returns plugins queued for install/update/removal
	// (pending a SonarQube restart).
	Pending(ctx context.Context) (v *sonar.PluginsPending, resp *http.Response, err error)
	// Uninstall removes a plugin by key.
	Uninstall(ctx context.Context, opt *sonar.PluginsUninstallOptions) (resp *http.Response, err error)
	// Update updates a plugin by key.
	Update(ctx context.Context, opt *sonar.PluginsUpdateOptions) (*http.Response, error)
	// Updates retrieves all updates available for installed plugins.
	Updates(ctx context.Context) (*sonar.PluginsUpdates, *http.Response, error)
}

// NewPluginsClient creates a PluginsClient from the given config.
func NewPluginsClient(clientConfig common.Config) PluginsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Plugins
}

// GeneratePluginObservation converts a sonar.PluginInstalled to an
// observation struct.
func GeneratePluginObservation(installed *sonar.PluginInstalled) v1alpha1.PluginObservation {
	if installed == nil {
		return v1alpha1.PluginObservation{}
	}

	var updatedAt *metav1.Time

	if installed.UpdatedAt != 0 {
		updatedAt = helpers.IntTimestampToMetaTime(&installed.UpdatedAt)
	}

	return v1alpha1.PluginObservation{
		Description:          installed.Description,
		EditionBundled:       installed.EditionBundled,
		Filename:             installed.Filename,
		Hash:                 installed.Hash,
		HomepageURL:          installed.HomepageURL,
		ImplementationBuild:  installed.ImplementationBuild,
		IssueTrackerURL:      installed.IssueTrackerURL,
		Key:                  installed.Key,
		License:              installed.License,
		Name:                 installed.Name,
		OrganizationName:     installed.OrganizationName,
		OrganizationURL:      installed.OrganizationURL,
		RequiredForLanguages: installed.RequiredForLanguages,
		SonarLintSupported:   installed.SonarLintSupported,
		UpdatedAt:            updatedAt,
		Version:              installed.Version,
	}
}

// FindInstalledPlugin returns a pointer to the installed plugin with
// the given key, or nil if not found.
func FindInstalledPlugin(plugins []sonar.PluginInstalled, key string) *sonar.PluginInstalled {
	for i := range plugins {
		if plugins[i].Key == key {
			return &plugins[i]
		}
	}

	return nil
}

// FindPendingInstallPlugin returns a pointer to the plugin with the
// given key in the pending-installation list, or nil if not found.
// It only checks pending.Installing - not Updating or Removing.
func FindPendingInstallPlugin(pending *sonar.PluginsPending, key string) *sonar.PluginPending {
	if pending == nil {
		return nil
	}

	for i := range pending.Installing {
		if pending.Installing[i].Key == key {
			return &pending.Installing[i]
		}
	}

	return nil
}

// GeneratePluginObservationFromPending builds a PluginObservation from
// a PluginPending entry. Fields not available before installation
// (EditionBundled, Filename, Hash, RequiredForLanguages, SonarLintSupported,
// UpdatedAt) are left at their zero values.
func GeneratePluginObservationFromPending(pending *sonar.PluginPending) v1alpha1.PluginObservation {
	if pending == nil {
		return v1alpha1.PluginObservation{}
	}

	return v1alpha1.PluginObservation{
		Description:         pending.Description,
		HomepageURL:         pending.HomepageURL,
		ImplementationBuild: pending.ImplementationBuild,
		IssueTrackerURL:     pending.IssueTrackerURL,
		Key:                 pending.Key,
		License:             pending.License,
		Name:                pending.Name,
		OrganizationName:    pending.OrganizationName,
		OrganizationURL:     pending.OrganizationURL,
		Pending:             true,
		Version:             pending.Version,
	}
}

// IsPluginPendingUpdate returns true if the plugin is in the
// pending-update list, meaning Update was called but SonarQube
// has not yet restarted to apply it.
func IsPluginPendingUpdate(pending *sonar.PluginsPending, key string) bool {
	if pending == nil {
		return false
	}

	for _, p := range pending.Updating {
		if p.Key == key {
			return true
		}
	}

	return false
}

// IsPluginPendingRemoval returns true if the plugin is in the
// pending-removal list, meaning Uninstall was called but SonarQube
// has not yet restarted to apply it.
func IsPluginPendingRemoval(pending *sonar.PluginsPending, key string) bool {
	if pending == nil {
		return false
	}

	for _, p := range pending.Removing {
		if p.Key == key {
			return true
		}
	}

	return false
}

// IsPluginUpdatable returns true if a plugin is in the list of
// updatable plugins by its key.
func IsPluginUpdatable(plugins *[]sonar.PluginWithUpdates, key string) bool {
	if plugins == nil {
		return false
	}

	for _, p := range *plugins {
		if p.Key == key {
			return true
		}
	}

	return false
}

// LateInitializePlugin fills empty spec fields from the observation.
// Plugins have no late-initializable fields, so this is a no-op.
func LateInitializePlugin(spec *v1alpha1.PluginParameters, observation *v1alpha1.PluginObservation) {
	if spec == nil || observation == nil {
		return
	}
}

// IsPluginLateInitialized returns true when the spec changed
// as a result of late initialization.
func IsPluginLateInitialized(former, current *v1alpha1.PluginParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !cmp.Equal(former, current)
}

// IsPluginUpToDate returns true when no upgrade should be performed.
// When AutoUpdate is explicitly disabled, the plugin is always considered
// up to date. Otherwise the result follows observation.IsLatest: true when
// already at the latest marketplace version.
func IsPluginUpToDate(spec *v1alpha1.PluginParameters, observation *v1alpha1.PluginObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.AutoUpdate != nil && !*spec.AutoUpdate {
		return true
	}

	return observation.IsLatest
}
