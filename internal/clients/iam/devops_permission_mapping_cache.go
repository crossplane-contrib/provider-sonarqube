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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

// devOpsPermissionMappingsCacheNamespace is the cache namespace of the
// api/v2/dop-translation/{platform}-permission-mappings lists.
const devOpsPermissionMappingsCacheNamespace = "dop-translation/permission-mappings"

// newCachedDevOpsPermissionMappingsClient decorates client with scoped, or
// returns client as-is when scoped does not cache.
func newCachedDevOpsPermissionMappingsClient(client DevOpsPermissionMappingsClient, scoped cache.Scoped) DevOpsPermissionMappingsClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedDevOpsPermissionMappingsClient{DevOpsPermissionMappingsClient: client, cache: scoped}
}

// cachedDevOpsPermissionMappingsClient is a DevOpsPermissionMappingsClient that
// caches Search per platform, so that every DevOpsPermissionMapping of one
// platform shares one call per TTL. Every write invalidates the cached lists.
type cachedDevOpsPermissionMappingsClient struct {
	DevOpsPermissionMappingsClient

	// cache holds the mapping lists of the embedded client's connection.
	cache cache.Scoped
}

// Search returns the cached mappings of the platform.
func (c *cachedDevOpsPermissionMappingsClient) Search(ctx context.Context, platform v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(devOpsPermissionMappingsCacheNamespace, string(platform)),
		func(ctx context.Context) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
			return c.DevOpsPermissionMappingsClient.Search(ctx, platform)
		})
}

// Create creates a mapping, then invalidates the cached lists.
func (c *cachedDevOpsPermissionMappingsClient) Create(ctx context.Context, platform v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	defer c.cache.Invalidate(devOpsPermissionMappingsCacheNamespace)

	return c.DevOpsPermissionMappingsClient.Create(ctx, platform, opt)
}

// Update updates a mapping, then invalidates the cached lists.
func (c *cachedDevOpsPermissionMappingsClient) Update(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	defer c.cache.Invalidate(devOpsPermissionMappingsCacheNamespace)

	return c.DevOpsPermissionMappingsClient.Update(ctx, platform, role, opt)
}

// Delete deletes a mapping, then invalidates the cached lists.
func (c *cachedDevOpsPermissionMappingsClient) Delete(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string) (*http.Response, error) {
	defer c.cache.Invalidate(devOpsPermissionMappingsCacheNamespace)

	return c.DevOpsPermissionMappingsClient.Delete(ctx, platform, role)
}
