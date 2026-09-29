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
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// permissionTemplatesCacheNamespace is the cache namespace of
	// api/permissions/search_templates.
	permissionTemplatesCacheNamespace = "permissions/search_templates"
	// permissionTemplateGroupsCacheNamespace is the cache namespace of
	// api/permissions/template_groups.
	permissionTemplateGroupsCacheNamespace = "permissions/template_groups"
	// permissionTemplateUsersCacheNamespace is the cache namespace of
	// api/permissions/template_users.
	permissionTemplateUsersCacheNamespace = "permissions/template_users"

	// permissionTemplatesPageSize is the page size requested from
	// api/permissions/search_templates.
	permissionTemplatesPageSize int64 = 500
	// permissionTemplateMembersPageSize is the page size of
	// api/permissions/template_groups and api/permissions/template_users,
	// which cap it at 100.
	permissionTemplateMembersPageSize int64 = 100
)

// init declares the entities the template datasets depend on: the
// template members are listed by group name and login, and the templates
// carry per-permission group and user counts.
func init() {
	cache.DependOn(permissionTemplatesCacheNamespace, cache.EntityGroup, cache.EntityUser)
	cache.DependOn(permissionTemplateGroupsCacheNamespace, cache.EntityGroup)
	cache.DependOn(permissionTemplateUsersCacheNamespace, cache.EntityUser)
}

// PermissionTemplateIndex indexes the permission templates of a SonarQube
// instance. It is shared with other callers and must not be mutated.
type PermissionTemplateIndex struct {
	// Templates are the permission templates, by ID.
	Templates map[string]sonar.PermissionTemplate `json:"templates"`
	// IDsByName are the template IDs, by template name.
	IDsByName map[string]string `json:"idsByName"`
	// DefaultIDs are the IDs of the templates that are the default of at
	// least one qualifier.
	DefaultIDs map[string]struct{} `json:"defaultIds"`
}

// FindByID returns the template with the given ID, whether it is a default
// template, and whether it exists.
func (i *PermissionTemplateIndex) FindByID(templateID string) (template sonar.PermissionTemplate, isDefault, found bool) {
	template, found = i.Templates[templateID]
	if !found {
		return sonar.PermissionTemplate{}, false, false
	}

	_, isDefault = i.DefaultIDs[templateID]

	return template, isDefault, true
}

// FindByName returns the template with the given name, whether it is a
// default template, and whether it exists.
func (i *PermissionTemplateIndex) FindByName(name string) (template sonar.PermissionTemplate, isDefault, found bool) {
	id, found := i.IDsByName[name]
	if !found {
		return sonar.PermissionTemplate{}, false, false
	}

	return i.FindByID(id)
}

// PermissionTemplatesIndex returns the index of every permission template
// of the instance.
//
// SonarQube only filters api/permissions/search_templates by name, so
// resolving a template by ID requires the whole list: it is fetched once
// per connection and TTL, and shared by every PermissionsTemplate resource.
func PermissionTemplatesIndex(ctx context.Context, client PermissionsTemplatesClient, scoped cache.Scoped) (*PermissionTemplateIndex, error) {
	return cache.Fetch(ctx, scoped.Store, scoped.Key(permissionTemplatesCacheNamespace, ""), func(ctx context.Context) (*PermissionTemplateIndex, error) {
		return fetchPermissionTemplatesIndex(ctx, client)
	})
}

// fetchPermissionTemplatesIndex reads every permission template and
// indexes them.
//
// api/permissions/search_templates returns no paging information, and
// current SonarQube versions return every template whatever the requested
// page. Pages are therefore requested until one is short or brings no
// template not seen yet, which also guards against servers ignoring the
// page parameters.
func fetchPermissionTemplatesIndex(ctx context.Context, client PermissionsTemplatesClient) (*PermissionTemplateIndex, error) {
	index := &PermissionTemplateIndex{
		Templates:  map[string]sonar.PermissionTemplate{},
		IDsByName:  map[string]string{},
		DefaultIDs: map[string]struct{}{},
	}

	for page := int64(1); ; page++ {
		err := ctx.Err()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot search permission templates page %d", page)
		}

		result, resp, err := client.SearchTemplates(ctx, GeneratePermissionsTemplateSearchOptions("", &sonar.PaginationArgs{Page: page, PageSize: permissionTemplatesPageSize})) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, errors.Wrapf(err, "cannot search permission templates page %d", page)
		}

		if result == nil {
			return nil, errors.Errorf("received nil permission templates page %d from SonarQube", page)
		}

		for _, defaultTemplate := range result.DefaultTemplates {
			index.DefaultIDs[defaultTemplate.TemplateID] = struct{}{}
		}

		added := 0

		for _, template := range result.PermissionTemplates {
			if _, seen := index.Templates[template.ID]; seen {
				continue
			}

			index.Templates[template.ID] = template
			index.IDsByName[template.Name] = template.ID
			added++
		}

		if len(result.PermissionTemplates) < int(permissionTemplatesPageSize) || added == 0 {
			return index, nil
		}
	}
}

// PermissionTemplateGroups returns every group having at least one
// permission in the template with the given ID. The list is cached per
// connection and template.
//
// The returned slice is shared with other callers and must not be mutated.
func PermissionTemplateGroups(ctx context.Context, client PermissionsTemplatesClient, scoped cache.Scoped, templateID string) ([]sonar.PermissionsTemplateGroup, error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
		return client.TemplateGroups(ctx, GeneratePermissionsTemplateGroupsSearchOptions(templateID, &page))
	}

	return fetchAllPagesCached(ctx, scoped, scoped.Key(permissionTemplateGroupsCacheNamespace, templateID), search, templateGroupsOf)
}

// PermissionTemplateUsers returns every user having at least one
// permission in the template with the given ID. The list is cached per
// connection and template.
//
// The returned slice is shared with other callers and must not be mutated.
func PermissionTemplateUsers(ctx context.Context, client PermissionsTemplatesClient, scoped cache.Scoped, templateID string) ([]sonar.PermissionsTemplateUser, error) {
	search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
		return client.TemplateUsers(ctx, GeneratePermissionsTemplateUsersSearchOptions(templateID, &page))
	}

	return fetchAllPagesCached(ctx, scoped, scoped.Key(permissionTemplateUsersCacheNamespace, templateID), search, templateUsersOf)
}

// fetchAllPagesCached returns every entry of the paginated search, cached
// under key.
func fetchAllPagesCached[R, T any](ctx context.Context, scoped cache.Scoped, key cache.Key, search common.PageSearch[R], items func(result *R) ([]T, sonar.Paging)) ([]T, error) {
	return cache.Fetch(ctx, scoped.Store, key, func(ctx context.Context) ([]T, error) {
		return common.FetchAllPages(ctx, permissionTemplateMembersPageSize, common.SearchPages(search, items))
	})
}

// templateGroupsOf returns the groups and paging of a template groups
// response.
func templateGroupsOf(result *sonar.PermissionsTemplateGroups) ([]sonar.PermissionsTemplateGroup, sonar.Paging) {
	return result.Groups, result.Paging
}

// templateUsersOf returns the users and paging of a template users
// response.
func templateUsersOf(result *sonar.PermissionsTemplateUsers) ([]sonar.PermissionsTemplateUser, sonar.Paging) {
	return result.Users, result.Paging
}

// NewCachedPermissionsTemplatesClient decorates client with scoped, so that
// its writes invalidate the template data cached in scoped, or returns
// client as-is when scoped does not cache.
func NewCachedPermissionsTemplatesClient(client PermissionsTemplatesClient, scoped cache.Scoped) PermissionsTemplatesClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedPermissionsTemplatesClient{PermissionsTemplatesClient: client, cache: scoped}
}

// cachedPermissionsTemplatesClient is a PermissionsTemplatesClient that
// invalidates the cached template data on every write. Reads go through
// PermissionTemplatesIndex, PermissionTemplateGroups and
// PermissionTemplateUsers, so SearchTemplates, TemplateGroups and
// TemplateUsers pass through.
//
// Every write also invalidates the template index: its templates carry
// per-permission group and user counts, and the project creator
// permissions (see UpdatePermissionsTemplateObservation).
type cachedPermissionsTemplatesClient struct {
	PermissionsTemplatesClient

	// cache holds the template data of the embedded client's connection.
	cache cache.Scoped
}

// CreateTemplate creates a template, then invalidates the template index.
func (c *cachedPermissionsTemplatesClient) CreateTemplate(ctx context.Context, opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.CreateTemplate(ctx, opt)
}

// UpdateTemplate updates a template, then invalidates the template index.
func (c *cachedPermissionsTemplatesClient) UpdateTemplate(ctx context.Context, opt *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.UpdateTemplate(ctx, opt)
}

// DeleteTemplate deletes a template, then invalidates every template
// dataset.
func (c *cachedPermissionsTemplatesClient) DeleteTemplate(ctx context.Context, opt *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace, permissionTemplateGroupsCacheNamespace, permissionTemplateUsersCacheNamespace)

	return c.PermissionsTemplatesClient.DeleteTemplate(ctx, opt)
}

// SetDefaultTemplate sets a default template, then invalidates the
// template index.
func (c *cachedPermissionsTemplatesClient) SetDefaultTemplate(ctx context.Context, opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.SetDefaultTemplate(ctx, opt)
}

// AddGroupToTemplate grants a template permission to a group, then
// invalidates the template groups and index.
func (c *cachedPermissionsTemplatesClient) AddGroupToTemplate(ctx context.Context, opt *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplateGroupsCacheNamespace, permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.AddGroupToTemplate(ctx, opt)
}

// RemoveGroupFromTemplate revokes a template permission from a group, then
// invalidates the template groups and index.
func (c *cachedPermissionsTemplatesClient) RemoveGroupFromTemplate(ctx context.Context, opt *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplateGroupsCacheNamespace, permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.RemoveGroupFromTemplate(ctx, opt)
}

// AddUserToTemplate grants a template permission to a user, then
// invalidates the template users and index.
func (c *cachedPermissionsTemplatesClient) AddUserToTemplate(ctx context.Context, opt *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplateUsersCacheNamespace, permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.AddUserToTemplate(ctx, opt)
}

// RemoveUserFromTemplate revokes a template permission from a user, then
// invalidates the template users and index.
func (c *cachedPermissionsTemplatesClient) RemoveUserFromTemplate(ctx context.Context, opt *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplateUsersCacheNamespace, permissionTemplatesCacheNamespace)

	return c.PermissionsTemplatesClient.RemoveUserFromTemplate(ctx, opt)
}

// AddProjectCreatorToTemplate grants a template permission to the project
// creator, then invalidates every template dataset. The project creator
// permissions are read from the index; the member lists are invalidated
// too in case a SonarQube version reflects them there.
func (c *cachedPermissionsTemplatesClient) AddProjectCreatorToTemplate(ctx context.Context, opt *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace, permissionTemplateGroupsCacheNamespace, permissionTemplateUsersCacheNamespace)

	return c.PermissionsTemplatesClient.AddProjectCreatorToTemplate(ctx, opt)
}

// RemoveProjectCreatorFromTemplate revokes a template permission from the
// project creator, then invalidates every template dataset, see
// AddProjectCreatorToTemplate.
func (c *cachedPermissionsTemplatesClient) RemoveProjectCreatorFromTemplate(ctx context.Context, opt *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error) {
	defer c.cache.Invalidate(permissionTemplatesCacheNamespace, permissionTemplateGroupsCacheNamespace, permissionTemplateUsersCacheNamespace)

	return c.PermissionsTemplatesClient.RemoveProjectCreatorFromTemplate(ctx, opt)
}
