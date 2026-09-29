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

// NewCachedGroupsClient decorates client with scoped, so that creating,
// renaming or deleting a group invalidates the datasets cached in scoped
// that depend on groups (see cache.EntityGroup), or returns client as-is
// when scoped does not cache.
func NewCachedGroupsClient(client GroupsClient, scoped cache.Scoped) GroupsClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedGroupsClient{GroupsClient: client, cache: scoped}
}

// cachedGroupsClient is a GroupsClient that invalidates the cached
// datasets depending on groups whenever a group is created, renamed or
// deleted: those writes change which names the cached searches return.
// Group memberships do not appear in any cached dataset, so their writes
// pass through.
type cachedGroupsClient struct {
	GroupsClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// CreateGroup creates a group, then invalidates the datasets depending on
// groups: a search by its name may have cached it as not found.
func (c *cachedGroupsClient) CreateGroup(ctx context.Context, opt *sonar.AuthorizationsCreateGroupOptions) (*sonar.AuthorizationsGroup, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityGroup)

	return c.GroupsClient.CreateGroup(ctx, opt)
}

// UpdateGroup updates (possibly renames) a group, then invalidates the
// datasets depending on groups.
func (c *cachedGroupsClient) UpdateGroup(ctx context.Context, groupID string, opt *sonar.AuthorizationsUpdateGroupOptions) (*sonar.AuthorizationsGroup, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityGroup)

	return c.GroupsClient.UpdateGroup(ctx, groupID, opt)
}

// DeleteGroup deletes a group, then invalidates the datasets depending on
// groups.
func (c *cachedGroupsClient) DeleteGroup(ctx context.Context, groupID string) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityGroup)

	return c.GroupsClient.DeleteGroup(ctx, groupID)
}

// NewCachedUsersClient decorates client with scoped, so that creating,
// updating or deactivating a user invalidates the datasets cached in scoped
// that depend on users (see cache.EntityUser), or returns client as-is
// when scoped does not cache.
func NewCachedUsersClient(client UsersClient, scoped cache.Scoped) UsersClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedUsersClient{UsersClient: client, cache: scoped}
}

// cachedUsersClient is a UsersClient that invalidates the cached datasets
// depending on users whenever a user is created (a search by its login may
// have cached it as not found), updated (possibly changing its login) or
// deactivated (which drops its permissions and tokens).
type cachedUsersClient struct {
	UsersClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Create creates a user, then invalidates the datasets depending on users.
func (c *cachedUsersClient) Create(ctx context.Context, opt *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityUser)

	return c.UsersClient.Create(ctx, opt)
}

// Update updates a user, then invalidates the datasets depending on users.
func (c *cachedUsersClient) Update(ctx context.Context, userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityUser)

	return c.UsersClient.Update(ctx, userID, opt)
}

// Deactivate deactivates a user, then invalidates the datasets depending
// on users.
func (c *cachedUsersClient) Deactivate(ctx context.Context, opt *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityUser)

	return c.UsersClient.Deactivate(ctx, opt)
}
