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

const (
	// qualityGateGroupsCacheNamespace is the cache namespace of
	// api/qualitygates/search_groups.
	qualityGateGroupsCacheNamespace = "qualitygates/search_groups"
	// qualityGateUsersCacheNamespace is the cache namespace of
	// api/qualitygates/search_users.
	qualityGateUsersCacheNamespace = "qualitygates/search_users"
	// qualityProfileGroupsCacheNamespace is the cache namespace of
	// api/qualityprofiles/search_groups.
	qualityProfileGroupsCacheNamespace = "qualityprofiles/search_groups"
	// qualityProfileUsersCacheNamespace is the cache namespace of
	// api/qualityprofiles/search_users.
	qualityProfileUsersCacheNamespace = "qualityprofiles/search_users"
)

// init declares the entities the selections depend on: they are keyed by
// Quality Gate or Quality Profile, and list groups by name and users by
// login.
func init() {
	cache.DependOn(qualityGateGroupsCacheNamespace, cache.EntityQualityGate, cache.EntityGroup)
	cache.DependOn(qualityGateUsersCacheNamespace, cache.EntityQualityGate, cache.EntityUser)
	cache.DependOn(qualityProfileGroupsCacheNamespace, cache.EntityQualityProfile, cache.EntityGroup)
	cache.DependOn(qualityProfileUsersCacheNamespace, cache.EntityQualityProfile, cache.EntityUser)
}

// QualityGateSelectedGroups returns the groups allowed to edit the Quality
// Gate named gateName, by group name, and whether the gate exists. See
// selectedIndex.
func QualityGateSelectedGroups(ctx context.Context, client QualityGateUsergroupAssociationClient, scoped cache.Scoped, gateName string) (selected map[string]sonar.QualityGateGroup, targetFound bool, err error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
		return client.SearchGroups(ctx, GenerateQualityGateSearchGroupsOptions(gateName, &page))
	}

	return selectedIndex(ctx, scoped, scoped.Key(qualityGateGroupsCacheNamespace, gateName), selectedPages(search, qualityGateGroupsOf), qualityGateGroupName)
}

// QualityGateSelectedUsers returns the users allowed to edit the Quality
// Gate named gateName, by login, and whether the gate exists. See
// selectedIndex.
func QualityGateSelectedUsers(ctx context.Context, client QualityGateUsergroupAssociationClient, scoped cache.Scoped, gateName string) (selected map[string]sonar.QualityGateUser, targetFound bool, err error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.QualitygatesSearchUsers, *http.Response, error) {
		return client.SearchUsers(ctx, GenerateQualityGateSearchUsersOptions(gateName, &page))
	}

	return selectedIndex(ctx, scoped, scoped.Key(qualityGateUsersCacheNamespace, gateName), selectedPages(search, qualityGateUsersOf), qualityGateUserLogin)
}

// QualityProfileSelectedGroups returns the groups allowed to edit the
// Quality Profile identified by language and qualityProfile, by group
// name, and whether the profile exists. See selectedIndex.
func QualityProfileSelectedGroups(ctx context.Context, client QualityProfileUsergroupAssociationClient, scoped cache.Scoped, language, qualityProfile string) (selected map[string]sonar.QualityprofilesProfileGroup, targetFound bool, err error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
		return client.SearchGroups(ctx, GenerateQualityProfileSearchGroupsOptions(language, qualityProfile, &page))
	}

	key := scoped.Key(qualityProfileGroupsCacheNamespace, qualityProfileParams(language, qualityProfile))

	return selectedIndex(ctx, scoped, key, selectedPages(search, qualityProfileGroupsOf), qualityProfileGroupName)
}

// QualityProfileSelectedUsers returns the users allowed to edit the
// Quality Profile identified by language and qualityProfile, by login, and
// whether the profile exists. See selectedIndex.
func QualityProfileSelectedUsers(ctx context.Context, client QualityProfileUsergroupAssociationClient, scoped cache.Scoped, language, qualityProfile string) (selected map[string]sonar.QualityprofilesProfileUser, targetFound bool, err error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.QualityprofilesSearchUsers, *http.Response, error) {
		return client.SearchUsers(ctx, GenerateQualityProfileSearchUsersOptions(language, qualityProfile, &page))
	}

	key := scoped.Key(qualityProfileUsersCacheNamespace, qualityProfileParams(language, qualityProfile))

	return selectedIndex(ctx, scoped, key, selectedPages(search, qualityProfileUsersOf), qualityProfileUserLogin)
}

// qualityGateGroupsOf returns the groups and paging of a Quality Gate
// groups search.
func qualityGateGroupsOf(result *sonar.QualitygatesSearchGroups) ([]sonar.QualityGateGroup, sonar.Paging) {
	return result.Groups, result.Paging
}

// qualityGateUsersOf returns the users and paging of a Quality Gate users
// search.
func qualityGateUsersOf(result *sonar.QualitygatesSearchUsers) ([]sonar.QualityGateUser, sonar.Paging) {
	return result.Users, result.Paging
}

// qualityProfileGroupsOf returns the groups and paging of a Quality
// Profile groups search.
func qualityProfileGroupsOf(result *sonar.QualityprofilesSearchGroups) ([]sonar.QualityprofilesProfileGroup, sonar.Paging) {
	return result.Groups, result.Paging
}

// qualityProfileUsersOf returns the users and paging of a Quality Profile
// users search.
func qualityProfileUsersOf(result *sonar.QualityprofilesSearchUsers) ([]sonar.QualityprofilesProfileUser, sonar.Paging) {
	return result.Users, result.Paging
}

// qualityGateGroupName returns the index key of a Quality Gate group.
func qualityGateGroupName(group sonar.QualityGateGroup) string { return group.Name }

// qualityGateUserLogin returns the index key of a Quality Gate user.
func qualityGateUserLogin(user sonar.QualityGateUser) string { return user.Login }

// qualityProfileGroupName returns the index key of a Quality Profile group.
func qualityProfileGroupName(group sonar.QualityprofilesProfileGroup) string { return group.Name }

// qualityProfileUserLogin returns the index key of a Quality Profile user.
func qualityProfileUserLogin(user sonar.QualityprofilesProfileUser) string { return user.Login }

// qualityProfileParams returns the cache parameters identifying a Quality
// Profile.
func qualityProfileParams(language, qualityProfile string) string {
	return cache.EncodeParams(map[string]string{"language": language, "qualityProfile": qualityProfile})
}

// NewCachedQualityGateUsergroupAssociationClient decorates client with
// scoped, so that its writes invalidate the selections cached in scoped,
// or returns client as-is when scoped does not cache.
func NewCachedQualityGateUsergroupAssociationClient(client QualityGateUsergroupAssociationClient, scoped cache.Scoped) QualityGateUsergroupAssociationClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedQualityGateUsergroupAssociationClient{QualityGateUsergroupAssociationClient: client, cache: scoped}
}

// cachedQualityGateUsergroupAssociationClient is a
// QualityGateUsergroupAssociationClient that invalidates the cached
// selections on every write. Reads go through QualityGateSelectedGroups
// and QualityGateSelectedUsers, so SearchGroups and SearchUsers pass
// through.
//
// Renaming or deleting a Quality Gate through the QualityGate controller
// invalidates these selections too (see cache.EntityQualityGate).
type cachedQualityGateUsergroupAssociationClient struct {
	QualityGateUsergroupAssociationClient

	// cache holds the selections of the embedded client's connection.
	cache cache.Scoped
}

// AddGroup grants a group edit rights on a Quality Gate, then invalidates
// the cached group selections.
func (c *cachedQualityGateUsergroupAssociationClient) AddGroup(ctx context.Context, opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityGateGroupsCacheNamespace)

	return c.QualityGateUsergroupAssociationClient.AddGroup(ctx, opt)
}

// RemoveGroup revokes a group's edit rights on a Quality Gate, then
// invalidates the cached group selections.
func (c *cachedQualityGateUsergroupAssociationClient) RemoveGroup(ctx context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityGateGroupsCacheNamespace)

	return c.QualityGateUsergroupAssociationClient.RemoveGroup(ctx, opt)
}

// AddUser grants a user edit rights on a Quality Gate, then invalidates
// the cached user selections.
func (c *cachedQualityGateUsergroupAssociationClient) AddUser(ctx context.Context, opt *sonar.QualitygatesAddUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityGateUsersCacheNamespace)

	return c.QualityGateUsergroupAssociationClient.AddUser(ctx, opt)
}

// RemoveUser revokes a user's edit rights on a Quality Gate, then
// invalidates the cached user selections.
func (c *cachedQualityGateUsergroupAssociationClient) RemoveUser(ctx context.Context, opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityGateUsersCacheNamespace)

	return c.QualityGateUsergroupAssociationClient.RemoveUser(ctx, opt)
}

// NewCachedQualityProfileUsergroupAssociationClient decorates client with
// scoped, so that its writes invalidate the selections cached in scoped,
// or returns client as-is when scoped does not cache.
func NewCachedQualityProfileUsergroupAssociationClient(client QualityProfileUsergroupAssociationClient, scoped cache.Scoped) QualityProfileUsergroupAssociationClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedQualityProfileUsergroupAssociationClient{QualityProfileUsergroupAssociationClient: client, cache: scoped}
}

// cachedQualityProfileUsergroupAssociationClient is the Quality Profile
// counterpart of cachedQualityGateUsergroupAssociationClient.
type cachedQualityProfileUsergroupAssociationClient struct {
	QualityProfileUsergroupAssociationClient

	// cache holds the selections of the embedded client's connection.
	cache cache.Scoped
}

// AddGroup grants a group edit rights on a Quality Profile, then
// invalidates the cached group selections.
func (c *cachedQualityProfileUsergroupAssociationClient) AddGroup(ctx context.Context, opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityProfileGroupsCacheNamespace)

	return c.QualityProfileUsergroupAssociationClient.AddGroup(ctx, opt)
}

// RemoveGroup revokes a group's edit rights on a Quality Profile, then
// invalidates the cached group selections.
func (c *cachedQualityProfileUsergroupAssociationClient) RemoveGroup(ctx context.Context, opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityProfileGroupsCacheNamespace)

	return c.QualityProfileUsergroupAssociationClient.RemoveGroup(ctx, opt)
}

// AddUser grants a user edit rights on a Quality Profile, then invalidates
// the cached user selections.
func (c *cachedQualityProfileUsergroupAssociationClient) AddUser(ctx context.Context, opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityProfileUsersCacheNamespace)

	return c.QualityProfileUsergroupAssociationClient.AddUser(ctx, opt)
}

// RemoveUser revokes a user's edit rights on a Quality Profile, then
// invalidates the cached user selections.
func (c *cachedQualityProfileUsergroupAssociationClient) RemoveUser(ctx context.Context, opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error) {
	defer c.cache.Invalidate(qualityProfileUsersCacheNamespace)

	return c.QualityProfileUsergroupAssociationClient.RemoveUser(ctx, opt)
}
