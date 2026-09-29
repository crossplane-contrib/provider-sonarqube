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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

// almDefinitionsCacheNamespace is the cache namespace of the ALM settings
// definitions (api/alm_settings/list_definitions), shared by every ALM
// kind.
const almDefinitionsCacheNamespace = "alm_settings/list_definitions"

// almSettingsAPI is the whole ALM settings surface used by the provider:
// every per-ALM settings client interface is a subset of it.
type almSettingsAPI interface {
	ALMSettingsClient
	CreateAzure(ctx context.Context, opt *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error)
	UpdateAzure(ctx context.Context, opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error)
	CreateBitbucket(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error)
	UpdateBitbucket(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error)
	CreateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error)
	UpdateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error)
	CreateGithub(ctx context.Context, opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error)
	UpdateGithub(ctx context.Context, opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error)
	CreateGitlab(ctx context.Context, opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error)
	UpdateGitlab(ctx context.Context, opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error)
}

// newALMSettingsAPI returns the ALM settings client of the given config.
// When the observe cache is enabled, the client caches the ALM settings
// definitions in cache.Default(), so that every ALM resource of every kind
// shares one list_definitions call per TTL.
func newALMSettingsAPI(clientConfig common.Config) almSettingsAPI {
	newClient := common.NewClient(clientConfig)

	return newCachedALMSettingsClient(newClient.AlmSettings, cache.ForConfig(clientConfig))
}

// newCachedALMSettingsClient decorates client with scoped, or returns
// client as-is when scoped does not cache.
func newCachedALMSettingsClient(client almSettingsAPI, scoped cache.Scoped) almSettingsAPI {
	if !scoped.Enabled() {
		return client
	}

	return &cachedALMSettingsClient{almSettingsAPI: client, cache: scoped}
}

// cachedALMSettingsClient is an almSettingsAPI that caches ListDefinitions
// and invalidates it on every write to an ALM setting. Project bindings do
// not appear in the definitions, so binding writes pass through, as do all
// methods not overridden.
type cachedALMSettingsClient struct {
	almSettingsAPI

	// cache holds the definitions of the embedded client's connection.
	cache cache.Scoped
}

// ListDefinitions returns the cached ALM settings definitions.
func (c *cachedALMSettingsClient) ListDefinitions(ctx context.Context) (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(almDefinitionsCacheNamespace, ""), c.almSettingsAPI.ListDefinitions)
}

// Delete deletes an ALM setting, then invalidates the definitions.
func (c *cachedALMSettingsClient) Delete(ctx context.Context, opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.Delete(ctx, opt)
}

// CreateAzure creates an Azure ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) CreateAzure(ctx context.Context, opt *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.CreateAzure(ctx, opt)
}

// UpdateAzure updates an Azure ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) UpdateAzure(ctx context.Context, opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.UpdateAzure(ctx, opt)
}

// CreateBitbucket creates a Bitbucket ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) CreateBitbucket(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.CreateBitbucket(ctx, opt)
}

// UpdateBitbucket updates a Bitbucket ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) UpdateBitbucket(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.UpdateBitbucket(ctx, opt)
}

// CreateBitbucketCloud creates a Bitbucket Cloud ALM setting, then
// invalidates the definitions.
func (c *cachedALMSettingsClient) CreateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.CreateBitbucketCloud(ctx, opt)
}

// UpdateBitbucketCloud updates a Bitbucket Cloud ALM setting, then
// invalidates the definitions.
func (c *cachedALMSettingsClient) UpdateBitbucketCloud(ctx context.Context, opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.UpdateBitbucketCloud(ctx, opt)
}

// CreateGithub creates a GitHub ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) CreateGithub(ctx context.Context, opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.CreateGithub(ctx, opt)
}

// UpdateGithub updates a GitHub ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) UpdateGithub(ctx context.Context, opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.UpdateGithub(ctx, opt)
}

// CreateGitlab creates a GitLab ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) CreateGitlab(ctx context.Context, opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.CreateGitlab(ctx, opt)
}

// UpdateGitlab updates a GitLab ALM setting, then invalidates the
// definitions.
func (c *cachedALMSettingsClient) UpdateGitlab(ctx context.Context, opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
	defer c.invalidate()

	return c.almSettingsAPI.UpdateGitlab(ctx, opt)
}

// invalidate drops the cached definitions, whether the write succeeded or
// not.
func (c *cachedALMSettingsClient) invalidate() {
	c.cache.Invalidate(almDefinitionsCacheNamespace)
}
