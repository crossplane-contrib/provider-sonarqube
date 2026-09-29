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
	"slices"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

const (
	// permissionsGroupsCacheNamespace is the cache namespace of
	// api/permissions/groups.
	permissionsGroupsCacheNamespace = "permissions/groups"
	// permissionsUsersCacheNamespace is the cache namespace of
	// api/permissions/users.
	permissionsUsersCacheNamespace = "permissions/users"

	// permissionsPageSize is the page size of api/permissions/groups and
	// api/permissions/users, which cap it at 100.
	permissionsPageSize int64 = 100

	// paramProjectKey is the cache parameter holding the project key, empty
	// for global permissions.
	paramProjectKey = "projectKey"
	// paramQuery is the cache parameter holding the q search filter.
	paramQuery = "q"
)

// init declares the entities the permission searches depend on.
func init() {
	// Group permissions list groups by name, on projects; user permissions
	// list users by login, on projects.
	cache.DependOn(permissionsGroupsCacheNamespace, cache.EntityGroup, cache.EntityProject)
	cache.DependOn(permissionsUsersCacheNamespace, cache.EntityUser, cache.EntityProject)
}

// permissionsLookup is the cached result of a search for one group or user.
type permissionsLookup struct {
	// Permissions are the permissions of the group or user.
	Permissions []string `json:"permissions"`
	// Found reports whether SonarQube returned the group or user.
	Found bool `json:"found"`
}

// GroupPermissionsIndex returns the permissions of every group having at
// least one permission, by group name: global permissions when projectKey
// is empty, the permissions on that project otherwise.
//
// It reads the unfiltered, paginated api/permissions/groups list, which
// SonarQube restricts to the groups having at least one permission. The
// index is cached per connection and project, so that every Permissions
// resource on one project shares a single scan per TTL.
//
// The returned map is shared with other callers and must not be mutated.
func GroupPermissionsIndex(ctx context.Context, client PermissionsClient, scoped cache.Scoped, projectKey string) (map[string][]string, error) {
	key := scoped.Key(permissionsGroupsCacheNamespace, cache.EncodeParams(map[string]string{paramProjectKey: projectKey}))

	return cache.Fetch(ctx, scoped.Store, key, func(ctx context.Context) (map[string][]string, error) {
		groups, err := common.FetchAllPages(ctx, permissionsPageSize, groupPermissionsPages(client, "", projectKey))
		if err != nil {
			return nil, errors.Wrap(err, "cannot list group permissions")
		}

		index := make(map[string][]string, len(groups))
		for _, group := range groups {
			index[group.Name] = group.Permissions
		}

		return index, nil
	})
}

// GroupPermissions returns the permissions of the group named groupName,
// global when projectKey is empty or on that project otherwise, and whether
// SonarQube returned the group.
//
// SonarQube drops project-scoped results when the q filter is combined with
// projectKey, so project permissions are read from GroupPermissionsIndex.
// Global permissions are searched by name instead, and cached per name:
// indexing every group of a large instance would cost more than one
// filtered search per Group or Permissions resource.
//
// The returned slice is a copy owned by the caller.
func GroupPermissions(ctx context.Context, client PermissionsClient, scoped cache.Scoped, groupName, projectKey string) (permissions []string, found bool, err error) {
	if projectKey != "" {
		index, indexErr := GroupPermissionsIndex(ctx, client, scoped, projectKey)
		if indexErr != nil {
			return nil, false, indexErr
		}

		permissions, found = index[groupName]

		return slices.Clone(permissions), found, nil
	}

	key := scoped.Key(permissionsGroupsCacheNamespace, cache.EncodeParams(map[string]string{paramProjectKey: projectKey, paramQuery: groupName}))

	lookup, err := cache.Fetch(ctx, scoped.Store, key, func(ctx context.Context) (permissionsLookup, error) {
		group, groupFound, searchErr := common.FindInPages(ctx, permissionsPageSize, groupPermissionsPages(client, groupName, projectKey),
			func(group sonar.PermissionGroup) bool { return group.Name == groupName })
		if searchErr != nil {
			return permissionsLookup{}, errors.Wrap(searchErr, "cannot search group permissions")
		}

		return permissionsLookup{Permissions: group.Permissions, Found: groupFound}, nil
	})
	if err != nil {
		return nil, false, err
	}

	return slices.Clone(lookup.Permissions), lookup.Found, nil
}

// UserPermissions returns the permissions of the user with the given
// login, global when projectKey is empty or on that project otherwise, and
// whether SonarQube returned the user.
//
// Users are searched by login and cached per (project, login): user counts
// are usually too large for an unfiltered index to pay off.
//
// The returned slice is a copy owned by the caller.
func UserPermissions(ctx context.Context, client PermissionsClient, scoped cache.Scoped, login, projectKey string) (permissions []string, found bool, err error) {
	key := scoped.Key(permissionsUsersCacheNamespace, cache.EncodeParams(map[string]string{paramProjectKey: projectKey, paramQuery: login}))

	lookup, err := cache.Fetch(ctx, scoped.Store, key, func(ctx context.Context) (permissionsLookup, error) {
		user, userFound, searchErr := common.FindInPages(ctx, permissionsPageSize, userPermissionsPages(client, login, projectKey),
			func(user sonar.PermissionUser) bool { return user.Login == login })
		if searchErr != nil {
			return permissionsLookup{}, errors.Wrap(searchErr, "cannot search user permissions")
		}

		return permissionsLookup{Permissions: user.Permissions, Found: userFound}, nil
	})
	if err != nil {
		return nil, false, err
	}

	return slices.Clone(lookup.Permissions), lookup.Found, nil
}

// groupPermissionsPages returns a PageFetcher of api/permissions/groups
// filtered by query (none when empty) and projectKey (global when empty).
func groupPermissionsPages(client PermissionsClient, query, projectKey string) common.PageFetcher[sonar.PermissionGroup] {
	return common.SearchPages(
		func(ctx context.Context, page sonar.PaginationArgs) (*sonar.PermissionsGroups, *http.Response, error) {
			return client.Groups(ctx, GeneratePermissionsGroupsOptions(query, optionalString(projectKey), &page))
		},
		func(result *sonar.PermissionsGroups) ([]sonar.PermissionGroup, sonar.Paging) {
			return result.Groups, result.Paging
		})
}

// userPermissionsPages returns a PageFetcher of api/permissions/users
// filtered by query (none when empty) and projectKey (global when empty).
func userPermissionsPages(client PermissionsClient, query, projectKey string) common.PageFetcher[sonar.PermissionUser] {
	return common.SearchPages(
		func(ctx context.Context, page sonar.PaginationArgs) (*sonar.PermissionsUsers, *http.Response, error) {
			return client.Users(ctx, GeneratePermissionsUsersOptions(query, optionalString(projectKey), &page))
		},
		func(result *sonar.PermissionsUsers) ([]sonar.PermissionUser, sonar.Paging) {
			return result.Users, result.Paging
		})
}

// optionalString returns nil for an empty value, a pointer to it otherwise.
func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

// NewCachedPermissionsClient decorates client with scoped, so that its
// writes invalidate the permission searches cached in scoped, or returns
// client as-is when scoped does not cache.
func NewCachedPermissionsClient(client PermissionsClient, scoped cache.Scoped) PermissionsClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedPermissionsClient{PermissionsClient: client, cache: scoped}
}

// cachedPermissionsClient is a PermissionsClient that invalidates the
// cached permission searches on every write. Reads go through
// GroupPermissions, GroupPermissionsIndex and UserPermissions, so Groups
// and Users pass through.
type cachedPermissionsClient struct {
	PermissionsClient

	// cache holds the permission searches of the embedded client's
	// connection.
	cache cache.Scoped
}

// AddGroup grants a permission to a group, then invalidates the cached
// group permissions.
func (c *cachedPermissionsClient) AddGroup(ctx context.Context, opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionsGroupsCacheNamespace)

	return c.PermissionsClient.AddGroup(ctx, opt)
}

// RemoveGroup revokes a permission from a group, then invalidates the
// cached group permissions.
func (c *cachedPermissionsClient) RemoveGroup(ctx context.Context, opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionsGroupsCacheNamespace)

	return c.PermissionsClient.RemoveGroup(ctx, opt)
}

// AddUser grants a permission to a user, then invalidates the cached user
// permissions.
func (c *cachedPermissionsClient) AddUser(ctx context.Context, opt *sonar.PermissionsAddUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionsUsersCacheNamespace)

	return c.PermissionsClient.AddUser(ctx, opt)
}

// RemoveUser revokes a permission from a user, then invalidates the cached
// user permissions.
func (c *cachedPermissionsClient) RemoveUser(ctx context.Context, opt *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionsUsersCacheNamespace)

	return c.PermissionsClient.RemoveUser(ctx, opt)
}
