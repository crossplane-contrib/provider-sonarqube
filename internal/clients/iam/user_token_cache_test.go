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

package iam

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// countingUserTokensClient is a UserTokensClient counting the calls of
// every method, and the Search calls per login.
type countingUserTokensClient struct {
	calls        map[string]int
	searchLogins map[string]int
	writeErr     error
}

// newCountingUserTokensClient returns a countingUserTokensClient with no
// call.
func newCountingUserTokensClient() *countingUserTokensClient {
	return &countingUserTokensClient{calls: map[string]int{}, searchLogins: map[string]int{}}
}

// Generate counts the call and returns writeErr.
func (c *countingUserTokensClient) Generate(context.Context, *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
	c.calls["Generate"]++

	if c.writeErr != nil {
		return nil, nil, c.writeErr
	}

	return &sonar.UserTokensGenerate{}, okResponse(), nil
}

// Revoke counts the call and returns writeErr.
func (c *countingUserTokensClient) Revoke(context.Context, *sonar.UserTokensRevokeOptions) (*http.Response, error) {
	c.calls["Revoke"]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return okResponse(), nil
}

// Search counts the call and returns an empty list.
func (c *countingUserTokensClient) Search(_ context.Context, opt *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
	c.calls["Search"]++
	c.searchLogins[opt.Login]++

	return &sonar.UserTokensSearch{}, okResponse(), nil
}

// searchTokens searches the tokens of login through client.
func searchTokens(t *testing.T, client UserTokensClient, login string) {
	t.Helper()

	result, resp, err := client.Search(context.Background(), &sonar.UserTokensSearchOptions{Login: login}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil || result == nil {
		t.Fatalf("Search(%q) = %v, %v, %v", login, result, resp, err)
	}
}

// TestNewCachedUserTokensClientDisabled tests that the raw client is used
// when the cache is disabled.
func TestNewCachedUserTokensClientDisabled(t *testing.T) {
	t.Parallel()

	inner := newCountingUserTokensClient()
	if got := NewCachedUserTokensClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedUserTokensClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewUserTokensClient(newTestConfig()).(*cachedUserTokensClient); cached {
		t.Error("NewUserTokensClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedUserTokensClientCachesSearchPerLogin tests that K tokens of one
// user share one Search per TTL.
func TestCachedUserTokensClientCachesSearchPerLogin(t *testing.T) {
	t.Parallel()

	inner := newCountingUserTokensClient()
	client := NewCachedUserTokensClient(inner, cachetest.NewScoped(t))

	for range 5 {
		searchTokens(t, client, "alice")
		searchTokens(t, client, "")
	}

	if inner.searchLogins["alice"] != 1 || inner.searchLogins[""] != 1 {
		t.Errorf("Search calls per login = %v, want 1 each", inner.searchLogins)
	}
}

// TestCachedUserTokensClientWritesInvalidate tests that every write, failed
// or not, invalidates every cached search, whatever its login.
func TestCachedUserTokensClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]func(UserTokensClient) error{
		"Generate": func(c UserTokensClient) error {
			_, resp, err := c.Generate(ctx, &sonar.UserTokensGenerateOptions{Name: "token", Login: "alice"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
		"Revoke": func(c UserTokensClient) error {
			resp, err := c.Revoke(ctx, &sonar.UserTokensRevokeOptions{Name: "token"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
	}

	for name, write := range writes {
		for _, writeErr := range []error{nil, errors.New("partial failure")} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				inner := newCountingUserTokensClient()
				inner.writeErr = writeErr
				client := NewCachedUserTokensClient(inner, cachetest.NewScoped(t))

				searchTokens(t, client, "alice")
				searchTokens(t, client, "")

				err := write(client)
				if !errors.Is(err, writeErr) {
					t.Fatalf("%s() error = %v, want %v", name, err, writeErr)
				}

				searchTokens(t, client, "alice")
				searchTokens(t, client, "")

				if inner.searchLogins["alice"] != 2 || inner.searchLogins[""] != 2 {
					t.Errorf("Search calls per login = %v, want 2 each", inner.searchLogins)
				}
			})
		}
	}
}

// TestEntityWritesInvalidateUserTokens tests that deactivating a user or
// deleting a project, whichever controller does it, invalidates the cached
// token lists.
func TestEntityWritesInvalidateUserTokens(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]func(scoped cache.Scoped) (*http.Response, error){
		"UserDeactivate": func(scoped cache.Scoped) (*http.Response, error) {
			return NewCachedUsersClient(&stubUsersClient{calls: map[string]int{}}, scoped).Deactivate(ctx, &sonar.UsersDeactivateOptionsV2{Id: "id"})
		},
		"ProjectDelete": func(scoped cache.Scoped) (*http.Response, error) {
			return instance.NewCachedProjectsClient(&stubProjectsClient{calls: map[string]int{}}, scoped).Delete(ctx, &sonar.ProjectsDeleteOptions{Project: "p"})
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inner := newCountingUserTokensClient()
			scoped := cachetest.NewScoped(t)
			client := NewCachedUserTokensClient(inner, scoped)

			searchTokens(t, client, "alice")

			resp, err := write(scoped) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}

			searchTokens(t, client, "alice")

			if got := inner.searchLogins["alice"]; got != 2 {
				t.Errorf("Search calls = %d, want 2", got)
			}
		})
	}
}
