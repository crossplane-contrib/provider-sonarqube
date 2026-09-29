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

// NewCachedProjectsClient decorates client with scoped, so that creating,
// deleting, re-keying or changing the visibility of a project invalidates
// the datasets cached in scoped that depend on projects (see
// cache.EntityProject), or returns client as-is when scoped does not cache.
func NewCachedProjectsClient(client ProjectsClient, scoped cache.Scoped) ProjectsClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedProjectsClient{ProjectsClient: client, cache: scoped}
}

// cachedProjectsClient is a ProjectsClient that invalidates the cached
// datasets depending on projects on every project write: creating a
// project applies a permission template, deleting it drops its permissions
// and webhooks, re-keying it moves them, and changing its visibility
// changes its permissions.
type cachedProjectsClient struct {
	ProjectsClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Create creates a project, then invalidates the datasets depending on
// projects.
func (c *cachedProjectsClient) Create(ctx context.Context, opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ProjectsClient.Create(ctx, opt)
}

// Delete deletes a project, then invalidates the datasets depending on
// projects.
func (c *cachedProjectsClient) Delete(ctx context.Context, opt *sonar.ProjectsDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ProjectsClient.Delete(ctx, opt)
}

// BulkDelete deletes projects, then invalidates the datasets depending on
// projects.
func (c *cachedProjectsClient) BulkDelete(ctx context.Context, opt *sonar.ProjectsBulkDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ProjectsClient.BulkDelete(ctx, opt)
}

// UpdateKey changes the key of a project, then invalidates the datasets
// depending on projects.
func (c *cachedProjectsClient) UpdateKey(ctx context.Context, opt *sonar.ProjectsUpdateKeyOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ProjectsClient.UpdateKey(ctx, opt)
}

// UpdateVisibility changes the visibility of a project, then invalidates
// the datasets depending on projects.
func (c *cachedProjectsClient) UpdateVisibility(ctx context.Context, opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityProject)

	return c.ProjectsClient.UpdateVisibility(ctx, opt)
}
