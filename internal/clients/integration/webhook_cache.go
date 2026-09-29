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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

// webhooksCacheNamespace is the cache namespace of api/webhooks/list.
const webhooksCacheNamespace = "webhooks/list"

// init declares that the webhook lists, keyed by project, depend on
// projects.
func init() {
	cache.DependOn(webhooksCacheNamespace, cache.EntityProject)
}

// NewCachedWebhooksClient decorates client with scoped, or returns client
// as-is when scoped does not cache.
func NewCachedWebhooksClient(client WebhooksClient, scoped cache.Scoped) WebhooksClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedWebhooksClient{WebhooksClient: client, cache: scoped}
}

// cachedWebhooksClient is a WebhooksClient that caches List per project
// (global webhooks under an empty project), so that every Webhook resource
// on one project shares one call per TTL.
//
// SonarQube accepts several webhooks with the same name, so a stale list
// read after Create would create a duplicate: every write invalidates every
// cached list. Update and Delete take a webhook key rather than a project,
// so the whole namespace is dropped rather than one project's entry.
type cachedWebhooksClient struct {
	WebhooksClient

	// cache holds the webhook lists of the embedded client's connection.
	cache cache.Scoped
}

// List returns the cached webhooks of the requested project.
func (c *cachedWebhooksClient) List(ctx context.Context, opt *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
	var project string
	if opt != nil {
		project = opt.Project
	}

	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(webhooksCacheNamespace, project),
		func(ctx context.Context) (*sonar.WebhooksList, *http.Response, error) {
			return c.WebhooksClient.List(ctx, opt)
		})
}

// Create creates a webhook, then invalidates the cached lists.
func (c *cachedWebhooksClient) Create(ctx context.Context, opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
	defer c.cache.Invalidate(webhooksCacheNamespace)

	return c.WebhooksClient.Create(ctx, opt)
}

// Update updates a webhook, then invalidates the cached lists.
func (c *cachedWebhooksClient) Update(ctx context.Context, opt *sonar.WebhooksUpdateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(webhooksCacheNamespace)

	return c.WebhooksClient.Update(ctx, opt)
}

// Delete deletes a webhook, then invalidates the cached lists.
func (c *cachedWebhooksClient) Delete(ctx context.Context, opt *sonar.WebhooksDeleteOptions) (*http.Response, error) {
	defer c.cache.Invalidate(webhooksCacheNamespace)

	return c.WebhooksClient.Delete(ctx, opt)
}
