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

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

// userTokensCacheNamespace is the cache namespace of
// api/user_tokens/search.
const userTokensCacheNamespace = "user_tokens/search"

// init declares that the token lists, keyed by login, depend on users, and
// on projects: project analysis tokens are dropped with their project.
func init() {
	cache.DependOn(userTokensCacheNamespace, cache.EntityUser, cache.EntityProject)
}

// NewCachedUserTokensClient decorates client with scoped, or returns client
// as-is when scoped does not cache.
func NewCachedUserTokensClient(client UserTokensClient, scoped cache.Scoped) UserTokensClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedUserTokensClient{UserTokensClient: client, cache: scoped}
}

// cachedUserTokensClient is a UserTokensClient that caches Search per
// login (the connection's own user under an empty login), so that every
// UserToken resource of one user shares one call per TTL.
//
// Every Generate mints a new token, so a stale list read after Generate
// would generate another one: every write invalidates every cached list.
// An empty login and an explicit login can designate the same user, so the
// whole namespace is dropped rather than one login's entry.
type cachedUserTokensClient struct {
	UserTokensClient

	// cache holds the token lists of the embedded client's connection.
	cache cache.Scoped
}

// Search returns the cached tokens of the requested user.
func (c *cachedUserTokensClient) Search(ctx context.Context, opt *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
	var login string
	if opt != nil {
		login = opt.Login
	}

	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(userTokensCacheNamespace, login),
		func(ctx context.Context) (*sonar.UserTokensSearch, *http.Response, error) {
			return c.UserTokensClient.Search(ctx, opt)
		})
}

// Generate generates a token, then invalidates the cached lists.
func (c *cachedUserTokensClient) Generate(ctx context.Context, opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
	defer c.cache.Invalidate(userTokensCacheNamespace)

	return c.UserTokensClient.Generate(ctx, opt)
}

// Revoke revokes a token, then invalidates the cached lists.
func (c *cachedUserTokensClient) Revoke(ctx context.Context, opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
	defer c.cache.Invalidate(userTokensCacheNamespace)

	return c.UserTokensClient.Revoke(ctx, opt)
}
