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

package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// countingWebhooksClient is a WebhooksClient counting the calls of every
// method, and the List calls per project.
type countingWebhooksClient struct {
	calls        map[string]int
	listProjects map[string]int
	writeErr     error
}

// newCountingWebhooksClient returns a countingWebhooksClient with no call.
func newCountingWebhooksClient() *countingWebhooksClient {
	return &countingWebhooksClient{calls: map[string]int{}, listProjects: map[string]int{}}
}

// List counts the call and returns an empty list.
func (c *countingWebhooksClient) List(_ context.Context, opt *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
	c.calls["List"]++
	c.listProjects[opt.Project]++

	return &sonar.WebhooksList{}, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// Create counts the call and returns writeErr.
func (c *countingWebhooksClient) Create(context.Context, *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
	resp, err := c.write("Create")

	return &sonar.WebhooksCreate{}, resp, err
}

// Update counts the call and returns writeErr.
func (c *countingWebhooksClient) Update(context.Context, *sonar.WebhooksUpdateOptions) (*http.Response, error) {
	return c.write("Update")
}

// Delete counts the call and returns writeErr.
func (c *countingWebhooksClient) Delete(context.Context, *sonar.WebhooksDeleteOptions) (*http.Response, error) {
	return c.write("Delete")
}

// write counts a call to method and returns writeErr.
func (c *countingWebhooksClient) write(method string) (*http.Response, error) {
	c.calls[method]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
}

// listWebhooks lists the webhooks of project through client.
func listWebhooks(t *testing.T, client WebhooksClient, project string) {
	t.Helper()

	list, resp, err := client.List(context.Background(), &sonar.WebhooksListOptions{Project: project}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil || list == nil {
		t.Fatalf("List(%q) = %v, %v, %v", project, list, resp, err)
	}
}

// TestNewCachedWebhooksClientDisabled tests that the raw client is used
// when the cache is disabled.
func TestNewCachedWebhooksClientDisabled(t *testing.T) {
	t.Parallel()

	inner := newCountingWebhooksClient()
	if got := NewCachedWebhooksClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedWebhooksClient() with a noop store = %T, want the raw client", got)
	}

	config := common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: "http://localhost:9000"}
	if _, cached := NewWebhooksClient(config).(*cachedWebhooksClient); cached {
		t.Error("NewWebhooksClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedWebhooksClientCachesListPerProject tests that K webhooks on one
// project share one List per TTL, and that global and project lists are
// distinct entries.
func TestCachedWebhooksClientCachesListPerProject(t *testing.T) {
	t.Parallel()

	inner := newCountingWebhooksClient()
	client := NewCachedWebhooksClient(inner, cachetest.NewScoped(t))

	for range 5 {
		listWebhooks(t, client, "project")
		listWebhooks(t, client, "")
	}

	if inner.listProjects["project"] != 1 || inner.listProjects[""] != 1 {
		t.Errorf("List calls per project = %v, want 1 for the project and 1 global", inner.listProjects)
	}
}

// TestCachedWebhooksClientWritesInvalidate tests that every write, failed
// or not, invalidates every cached list.
func TestCachedWebhooksClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]func(WebhooksClient) error{
		"Create": func(c WebhooksClient) error {
			_, resp, err := c.Create(ctx, &sonar.WebhooksCreateOptions{Name: "hook", Project: "project"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
		"Update": func(c WebhooksClient) error {
			resp, err := c.Update(ctx, &sonar.WebhooksUpdateOptions{Webhook: "key"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
		"Delete": func(c WebhooksClient) error {
			resp, err := c.Delete(ctx, &sonar.WebhooksDeleteOptions{Webhook: "key"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
	}

	for name, write := range writes {
		for _, writeErr := range []error{nil, errors.New("partial failure")} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				inner := newCountingWebhooksClient()
				inner.writeErr = writeErr
				client := NewCachedWebhooksClient(inner, cachetest.NewScoped(t))

				listWebhooks(t, client, "project")
				listWebhooks(t, client, "")

				err := write(client)
				if !errors.Is(err, writeErr) {
					t.Fatalf("%s() error = %v, want %v", name, err, writeErr)
				}

				listWebhooks(t, client, "project")
				listWebhooks(t, client, "")

				if inner.listProjects["project"] != 2 || inner.listProjects[""] != 2 {
					t.Errorf("List calls per project = %v, want 2 each", inner.listProjects)
				}
			})
		}
	}
}

// stubProjectsClient is an instance.ProjectsClient accepting deletions.
// Other methods panic through the nil embedded interface.
type stubProjectsClient struct {
	instance.ProjectsClient
}

// Delete succeeds.
func (*stubProjectsClient) Delete(context.Context, *sonar.ProjectsDeleteOptions) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
}

// TestProjectDeleteInvalidatesWebhooks tests that deleting a project,
// whichever controller does it, invalidates the cached webhook lists.
func TestProjectDeleteInvalidatesWebhooks(t *testing.T) {
	t.Parallel()

	inner := newCountingWebhooksClient()
	scoped := cachetest.NewScoped(t)
	client := NewCachedWebhooksClient(inner, scoped)

	listWebhooks(t, client, "project")

	resp, err := instance.NewCachedProjectsClient(&stubProjectsClient{}, scoped).Delete(context.Background(), &sonar.ProjectsDeleteOptions{Project: "project"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	listWebhooks(t, client, "project")

	if got := inner.listProjects["project"]; got != 2 {
		t.Errorf("List calls = %d, want 2", got)
	}
}
