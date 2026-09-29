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

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

// Applications and portfolios are components like projects: permissions
// are granted on their keys, and their permission template is applied when
// they are created. Their creation and deletion therefore invalidate the
// datasets depending on projects (see cache.EntityProject).

// NewCachedApplicationsClient decorates client with scoped, so that
// creating or deleting an application invalidates the datasets cached in
// scoped that depend on projects, or returns client as-is when scoped does
// not cache.
func NewCachedApplicationsClient(client ApplicationsClient, scoped cache.Scoped) ApplicationsClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedApplicationsClient{ApplicationsClient: client, cache: scoped}
}

// cachedApplicationsClient is an ApplicationsClient that invalidates the
// cached datasets depending on projects when an application is created or
// deleted.
type cachedApplicationsClient struct {
	ApplicationsClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Create creates an application, then invalidates the datasets depending
// on projects.
func (c *cachedApplicationsClient) Create(ctx context.Context, opt *sonar.ApplicationsCreateOptions) (*sonar.ApplicationsCreate, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ApplicationsClient.Create(ctx, opt)
}

// Delete deletes an application, then invalidates the datasets depending
// on projects.
func (c *cachedApplicationsClient) Delete(ctx context.Context, opt *sonar.ApplicationsDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ApplicationsClient.Delete(ctx, opt)
}

// NewCachedPortfoliosClient decorates client with scoped, so that creating
// or deleting a portfolio invalidates the datasets cached in scoped that
// depend on projects, or returns client as-is when scoped does not cache.
func NewCachedPortfoliosClient(client PortfoliosClient, scoped cache.Scoped) PortfoliosClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedPortfoliosClient{PortfoliosClient: client, cache: scoped}
}

// cachedPortfoliosClient is a PortfoliosClient that invalidates the cached
// datasets depending on projects when a portfolio is created or deleted.
type cachedPortfoliosClient struct {
	PortfoliosClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Create creates a portfolio, then invalidates the datasets depending on
// projects.
func (c *cachedPortfoliosClient) Create(ctx context.Context, opt *sonar.ViewsCreateOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.PortfoliosClient.Create(ctx, opt)
}

// Delete deletes a portfolio, then invalidates the datasets depending on
// projects.
func (c *cachedPortfoliosClient) Delete(ctx context.Context, opt *sonar.ViewsDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.PortfoliosClient.Delete(ctx, opt)
}
