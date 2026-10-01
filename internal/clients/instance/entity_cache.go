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

// NewCachedQualityGatesClient decorates client with scoped, so that deleting
// or renaming a Quality Gate, or changing who may edit it, invalidates the
// datasets cached in scoped that depend on Quality Gates (see
// cache.EntityQualityGate), or returns client as-is when scoped does not
// cache.
func NewCachedQualityGatesClient(client QualityGatesClient, scoped cache.Scoped) QualityGatesClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedQualityGatesClient{QualityGatesClient: client, cache: scoped}
}

// cachedQualityGatesClient is a QualityGatesClient that invalidates the
// cached datasets depending on Quality Gates whenever a Quality Gate is
// deleted or renamed, or its editors change.
type cachedQualityGatesClient struct {
	QualityGatesClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Delete deletes a Quality Gate, then invalidates the datasets depending on
// Quality Gates.
func (c *cachedQualityGatesClient) Delete(ctx context.Context, opt *sonar.QualitygatesDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.Delete(ctx, opt)
}

// Rename renames a Quality Gate, then invalidates the datasets depending on
// Quality Gates.
func (c *cachedQualityGatesClient) Rename(ctx context.Context, opt *sonar.QualitygatesRenameOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.Rename(ctx, opt)
}

// AddGroup grants a group edit rights on a Quality Gate, then invalidates
// the datasets depending on Quality Gates.
func (c *cachedQualityGatesClient) AddGroup(ctx context.Context, opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.AddGroup(ctx, opt)
}

// RemoveGroup revokes a group's edit rights on a Quality Gate, then
// invalidates the datasets depending on Quality Gates.
func (c *cachedQualityGatesClient) RemoveGroup(ctx context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.RemoveGroup(ctx, opt)
}

// AddUser grants a user edit rights on a Quality Gate, then invalidates the
// datasets depending on Quality Gates.
func (c *cachedQualityGatesClient) AddUser(ctx context.Context, opt *sonar.QualitygatesAddUserOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.AddUser(ctx, opt)
}

// RemoveUser revokes a user's edit rights on a Quality Gate, then
// invalidates the datasets depending on Quality Gates.
func (c *cachedQualityGatesClient) RemoveUser(ctx context.Context, opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityGate)

	return c.QualityGatesClient.RemoveUser(ctx, opt)
}

// NewCachedQualityProfilesClient decorates client with scoped, so that
// deleting or renaming a Quality Profile, or changing who may edit it,
// invalidates the datasets cached in scoped that depend on Quality Profiles
// (see cache.EntityQualityProfile), or returns client as-is when scoped
// does not cache.
func NewCachedQualityProfilesClient(client QualityProfilesClient, scoped cache.Scoped) QualityProfilesClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedQualityProfilesClient{QualityProfilesClient: client, cache: scoped}
}

// cachedQualityProfilesClient is a QualityProfilesClient that invalidates
// the cached datasets depending on Quality Profiles whenever a Quality
// Profile is deleted or renamed, or its editors change.
type cachedQualityProfilesClient struct {
	QualityProfilesClient

	// cache holds the datasets of the embedded client's connection.
	cache cache.Scoped
}

// Delete deletes a Quality Profile, then invalidates the datasets depending
// on Quality Profiles.
func (c *cachedQualityProfilesClient) Delete(ctx context.Context, opt *sonar.QualityprofilesDeleteOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.Delete(ctx, opt)
}

// Rename renames a Quality Profile, then invalidates the datasets depending
// on Quality Profiles.
func (c *cachedQualityProfilesClient) Rename(ctx context.Context, opt *sonar.QualityprofilesRenameOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.Rename(ctx, opt)
}

// AddGroup grants a group edit rights on a Quality Profile, then invalidates
// the datasets depending on Quality Profiles.
func (c *cachedQualityProfilesClient) AddGroup(ctx context.Context, opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.AddGroup(ctx, opt)
}

// RemoveGroup revokes a group's edit rights on a Quality Profile, then
// invalidates the datasets depending on Quality Profiles.
func (c *cachedQualityProfilesClient) RemoveGroup(ctx context.Context, opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.RemoveGroup(ctx, opt)
}

// AddUser grants a user edit rights on a Quality Profile, then invalidates
// the datasets depending on Quality Profiles.
func (c *cachedQualityProfilesClient) AddUser(ctx context.Context, opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.AddUser(ctx, opt)
}

// RemoveUser revokes a user's edit rights on a Quality Profile, then
// invalidates the datasets depending on Quality Profiles.
func (c *cachedQualityProfilesClient) RemoveUser(ctx context.Context, opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error) {
	defer c.cache.InvalidateEntities(cache.EntityQualityProfile)

	return c.QualityProfilesClient.RemoveUser(ctx, opt)
}
