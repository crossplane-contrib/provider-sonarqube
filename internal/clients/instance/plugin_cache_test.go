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
	"errors"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// countingPluginsClient is a PluginsClient counting the calls of every
// method.
type countingPluginsClient struct {
	calls    map[string]int
	writeErr error
}

// newCountingPluginsClient returns a countingPluginsClient with no call.
func newCountingPluginsClient() *countingPluginsClient {
	return &countingPluginsClient{calls: map[string]int{}}
}

// okResponse returns a fresh 200 response.
func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// Install counts the call and returns writeErr.
func (c *countingPluginsClient) Install(context.Context, *sonar.PluginsInstallOptions) (*http.Response, error) {
	c.calls["Install"]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return okResponse(), nil
}

// Installed counts the call and returns an empty list.
func (c *countingPluginsClient) Installed(context.Context, *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
	c.calls["Installed"]++

	return &sonar.PluginsInstalled{}, okResponse(), nil
}

// Pending counts the call and returns an empty list.
func (c *countingPluginsClient) Pending(context.Context) (*sonar.PluginsPending, *http.Response, error) {
	c.calls["Pending"]++

	return &sonar.PluginsPending{}, okResponse(), nil
}

// Uninstall counts the call and returns writeErr.
func (c *countingPluginsClient) Uninstall(context.Context, *sonar.PluginsUninstallOptions) (*http.Response, error) {
	c.calls["Uninstall"]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return okResponse(), nil
}

// Update counts the call and returns writeErr.
func (c *countingPluginsClient) Update(context.Context, *sonar.PluginsUpdateOptions) (*http.Response, error) {
	c.calls["Update"]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return okResponse(), nil
}

// Updates counts the call and returns an empty list.
func (c *countingPluginsClient) Updates(context.Context) (*sonar.PluginsUpdates, *http.Response, error) {
	c.calls["Updates"]++

	return &sonar.PluginsUpdates{}, okResponse(), nil
}

// readAll calls every cached read of client once.
func readAll(t *testing.T, client PluginsClient) {
	t.Helper()

	ctx := context.Background()

	_, resp, err := client.Installed(ctx, nil) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil {
		t.Fatalf("Installed() = %v, %v", resp, err)
	}

	_, resp, err = client.Pending(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil {
		t.Fatalf("Pending() = %v, %v", resp, err)
	}

	_, resp, err = client.Updates(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil {
		t.Fatalf("Updates() = %v, %v", resp, err)
	}
}

// assertReads checks that every cached read reached the inner client want
// times.
func assertReads(t *testing.T, inner *countingPluginsClient, want int) {
	t.Helper()

	for _, method := range []string{"Installed", "Pending", "Updates"} {
		if got := inner.calls[method]; got != want {
			t.Errorf("inner %s called %d times, want %d", method, got, want)
		}
	}
}

// TestNewCachedPluginsClientDisabled tests that the raw client is used when
// the cache is disabled.
func TestNewCachedPluginsClientDisabled(t *testing.T) {
	t.Parallel()

	inner := newCountingPluginsClient()

	if got := newCachedPluginsClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("newCachedPluginsClient() with a noop store = %T, want the raw client", got)
	}

	// The flag is off by default, so NewPluginsClient returns the raw SDK
	// client.
	if _, cached := NewPluginsClient(newTestConfig()).(*cachedPluginsClient); cached {
		t.Error("NewPluginsClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedPluginsClientCachesReads tests that repeated list reads reach
// SonarQube once.
func TestCachedPluginsClientCachesReads(t *testing.T) {
	t.Parallel()

	inner := newCountingPluginsClient()
	client := newCachedPluginsClient(inner, cachetest.NewScoped(t))

	readAll(t, client)
	readAll(t, client)

	assertReads(t, inner, 1)

	// Calls with options are not cached.
	for range 2 {
		_, resp, err := client.Installed(context.Background(), &sonar.PluginsInstalledOptions{}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			t.Fatalf("Installed() error = %v", err)
		}
	}

	if got := inner.calls["Installed"]; got != 3 {
		t.Errorf("inner Installed called %d times, want 3", got)
	}
}

// TestCachedPluginsClientWritesInvalidate tests that every write, failed or
// not, invalidates the cached lists.
func TestCachedPluginsClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	writes := map[string]func(PluginsClient) (*http.Response, error){
		"Install": func(client PluginsClient) (*http.Response, error) {
			return client.Install(context.Background(), &sonar.PluginsInstallOptions{Key: "findbugs"})
		},
		"Uninstall": func(client PluginsClient) (*http.Response, error) {
			return client.Uninstall(context.Background(), &sonar.PluginsUninstallOptions{Key: "findbugs"})
		},
		"Update": func(client PluginsClient) (*http.Response, error) {
			return client.Update(context.Background(), &sonar.PluginsUpdateOptions{Key: "findbugs"})
		},
	}

	for name, write := range writes {
		for _, writeErr := range []error{nil, errors.New("partial failure")} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				inner := newCountingPluginsClient()
				inner.writeErr = writeErr
				client := newCachedPluginsClient(inner, cachetest.NewScoped(t))

				readAll(t, client)

				resp, err := write(client) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if !errors.Is(err, writeErr) {
					t.Fatalf("%s() error = %v, want %v", name, err, writeErr)
				}

				if got := inner.calls[name]; got != 1 {
					t.Errorf("inner %s called %d times, want 1", name, got)
				}

				readAll(t, client)
				assertReads(t, inner, 2)
			})
		}
	}
}

// TestCachedPluginsClientScopesAreIsolated tests that connections with
// different scopes share no cached data.
func TestCachedPluginsClientScopesAreIsolated(t *testing.T) {
	t.Parallel()

	store := cachetest.NewStore(t)
	inner := newCountingPluginsClient()
	first := newCachedPluginsClient(inner, cache.NewScoped(store, t.Name()+"/first"))
	second := newCachedPluginsClient(inner, cache.NewScoped(store, t.Name()+"/second"))

	readAll(t, first)
	readAll(t, second)
	assertReads(t, inner, 2)

	// A write through one scope does not invalidate the other.
	resp, err := first.Install(context.Background(), &sonar.PluginsInstallOptions{Key: "findbugs"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	readAll(t, second)
	assertReads(t, inner, 2)
}
