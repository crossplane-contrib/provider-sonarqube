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
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestObserve covers early returns and the installed-plugin flow.
func TestObserve(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client *fake.MockPluginsClient
		args   observeArgs
		want   observeWant
	}{
		"NotPluginError": {
			client: &fake.MockPluginsClient{},
			args:   observeArgs{ctx: context.Background(), mg: &notPlugin{}},
			want:   observeWant{o: managed.ExternalObservation{}, err: errors.New(errNotPlugin)},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockPluginsClient{},
			args:   observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want:   observeWant{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"InstalledAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{o: managed.ExternalObservation{}, err: errors.Wrap(errBoom, errObservePlugin)},
		},
		"UpdatesAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{
				o:   managed.ExternalObservation{ResourceExists: true},
				err: errors.Wrap(errBoom, "cannot get plugin updates from SonarQube"),
			},
		},
		"PluginAtLatestVersion": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{Plugins: []sonar.PluginWithUpdates{}}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"PluginHasUpdateAutoUpdateEnabled": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{
				ctx: context.Background(),
				mg: func() *v1alpha1.Plugin {
					p := newPlugin(testPluginKey, testPluginKey)
					autoUpdate := true
					p.Spec.ForProvider.AutoUpdate = &autoUpdate

					return p
				}(),
			},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  false,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"PluginHasUpdateAutoUpdateDisabled": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{
				ctx: context.Background(),
				mg: func() *v1alpha1.Plugin {
					p := newPlugin(testPluginKey, testPluginKey)
					autoUpdate := false
					p.Spec.ForProvider.AutoUpdate = &autoUpdate

					return p
				}(),
			},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
	}

	runObserveCases(t, cases)
}

// TestObservePendingInstall covers the pending-installation path.
func TestObservePendingInstall(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client *fake.MockPluginsClient
		args   observeArgs
		want   observeWant
	}{
		"PluginNotInInstalledNorPending": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: "other-plugin"}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"PluginPendingInstall": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{Plugins: []sonar.PluginInstalled{}}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{
						Installing: []sonar.PluginPending{{Key: testPluginKey, Name: "FindBugs"}},
					}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"PendingInstallAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{Plugins: []sonar.PluginInstalled{}}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{o: managed.ExternalObservation{}, err: errors.Wrap(errBoom, errObservePlugin)},
		},
	}

	runObserveCases(t, cases)
}

// TestObservePendingRemoval covers the pending-removal path (plugin queued
// for uninstall but SonarQube has not yet restarted).
func TestObservePendingRemoval(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client *fake.MockPluginsClient
		args   observeArgs
		want   observeWant
	}{
		"PendingRemovalDuringDeletion": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{
						Removing: []sonar.PluginPending{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPluginDeleting(testPluginKey, testPluginKey)},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"PendingRemovalAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPluginDeleting(testPluginKey, testPluginKey)},
			want: observeWant{o: managed.ExternalObservation{}, err: errors.Wrap(errBoom, errObservePlugin)},
		},
	}

	runObserveCases(t, cases)
}

// TestObservePendingUpdate covers the pending-upgrade path (plugin queued
// for update but SonarQube has not yet restarted).
func TestObservePendingUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client *fake.MockPluginsClient
		args   observeArgs
		want   observeWant
	}{
		"PluginPendingUpdate": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{
						Updating: []sonar.PluginPendingUpdate{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"PendingUpdateAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: observeArgs{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: observeWant{o: managed.ExternalObservation{}, err: errors.Wrap(errBoom, errObservePlugin)},
		},
	}

	runObserveCases(t, cases)
}

// TestObservePopulatesAtProvider verifies Observe writes all observation
// fields from the installed plugin.
func TestObservePopulatesAtProvider(t *testing.T) {
	t.Parallel()

	mg := newPlugin(testPluginKey, testPluginKey)
	e := &external{client: &fake.MockPluginsClient{
		InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
			return &sonar.PluginsInstalled{
				Plugins: []sonar.PluginInstalled{
					{
						Key:         testPluginKey,
						Name:        "FindBugs",
						Version:     "4.0.0",
						Description: "Find bugs",
						Filename:    "findbugs.jar",
						License:     "LGPL",
					},
				},
			}, mockHTTPOK(), nil
		},
		UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
			return &sonar.PluginsUpdates{Plugins: []sonar.PluginWithUpdates{}}, mockHTTPOK(), nil
		},
	}}

	_, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	obs := mg.Status.AtProvider
	if obs.Name != "FindBugs" {
		t.Errorf("AtProvider.Name = %q, want %q", obs.Name, "FindBugs")
	}

	if obs.Version != "4.0.0" {
		t.Errorf("AtProvider.Version = %q, want %q", obs.Version, "4.0.0")
	}

	if !obs.IsLatest {
		t.Errorf("AtProvider.IsLatest = false, want true (no updates available)")
	}
}
