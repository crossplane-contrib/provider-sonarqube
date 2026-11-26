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
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// devOpsPermissions pairs each SonarQube permission a DevOps platform role can
// be mapped to with its field in the API representation.
var devOpsPermissions = []struct {
	name  string
	field func(*sonar.DopTranslationPermissions) *bool
}{
	{"user", func(p *sonar.DopTranslationPermissions) *bool { return &p.User }},
	{"codeviewer", func(p *sonar.DopTranslationPermissions) *bool { return &p.CodeViewer }},
	{"issueadmin", func(p *sonar.DopTranslationPermissions) *bool { return &p.IssueAdmin }},
	{"securityhotspotadmin", func(p *sonar.DopTranslationPermissions) *bool { return &p.SecurityHotspotAdmin }},
	{"admin", func(p *sonar.DopTranslationPermissions) *bool { return &p.Admin }},
	{"scan", func(p *sonar.DopTranslationPermissions) *bool { return &p.Scan }},
}

// DevOpsPermissionMappingsClient handles the SonarQube DevOps platform role
// permission mapping API operations.
type DevOpsPermissionMappingsClient interface {
	Search(ctx context.Context, platform v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error)
	Create(ctx context.Context, platform v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error)
	Update(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error)
	Delete(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string) (*http.Response, error)
}

// NewDevOpsPermissionMappingsClient creates a new
// DevOpsPermissionMappingsClient from the given config.
func NewDevOpsPermissionMappingsClient(clientConfig common.Config) DevOpsPermissionMappingsClient {
	return newCachedDevOpsPermissionMappingsClient(
		&devOpsPermissionMappingsClient{client: common.NewClient(clientConfig)},
		cache.ForConfig(clientConfig),
	)
}

// devOpsPermissionMappingsClient dispatches each operation to the endpoint of
// the requested platform.
type devOpsPermissionMappingsClient struct {
	client *sonar.Client
}

// Search lists the permission mappings of the platform.
func (c *devOpsPermissionMappingsClient) Search(ctx context.Context, platform v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
	switch platform {
	case v1alpha1.DevOpsPlatformGitHub:
		return c.client.V2.DopTranslation.SearchGithubPermissionMappings(ctx)
	case v1alpha1.DevOpsPlatformGitLab:
		return c.client.V2.DopTranslation.SearchGitlabPermissionMappings(ctx)
	default:
		return nil, nil, unsupportedPlatformError(platform)
	}
}

// Create creates the permission mapping of a custom role.
func (c *devOpsPermissionMappingsClient) Create(ctx context.Context, platform v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	switch platform {
	case v1alpha1.DevOpsPlatformGitHub:
		return c.client.V2.DopTranslation.CreateGithubPermissionMapping(ctx, opt)
	case v1alpha1.DevOpsPlatformGitLab:
		return c.client.V2.DopTranslation.CreateGitlabPermissionMapping(ctx, opt)
	default:
		return nil, nil, unsupportedPlatformError(platform)
	}
}

// Update changes the permissions of the mapping of a role.
func (c *devOpsPermissionMappingsClient) Update(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	req, err := c.newRoleRequest(ctx, http.MethodPatch, platform, role, opt)
	if err != nil {
		return nil, nil, err
	}

	result := new(sonar.DopTranslationPermissionMapping)

	resp, err := c.client.Do(req, result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

// Delete deletes the permission mapping of a custom role.
func (c *devOpsPermissionMappingsClient) Delete(ctx context.Context, platform v1alpha1.DevOpsPlatform, role string) (*http.Response, error) {
	req, err := c.newRoleRequest(ctx, http.MethodDelete, platform, role, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req, nil)
}

// newRoleRequest builds a request on the mapping of one role.
//
// The SDK methods for these endpoints are not used: they escape the role
// before the request URL escapes it again, so a custom role such as
// "Security Reviewer" would never be found.
func (c *devOpsPermissionMappingsClient) newRoleRequest(ctx context.Context, method string, platform v1alpha1.DevOpsPlatform, role string, body any) (*http.Request, error) {
	if platform != v1alpha1.DevOpsPlatformGitHub && platform != v1alpha1.DevOpsPlatformGitLab {
		return nil, unsupportedPlatformError(platform)
	}

	// A slash would change the path of the request.
	if role == "" || strings.Contains(role, "/") {
		return nil, errors.Errorf("invalid role name %q", role)
	}

	req, err := c.client.NewSonarQubeV2APIRequest(ctx, method, "dop-translation/"+string(platform)+"-permission-mappings/"+role, nil, body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create request")
	}

	return req, nil
}

// unsupportedPlatformError reports a platform without permission mappings.
func unsupportedPlatformError(platform v1alpha1.DevOpsPlatform) error {
	return errors.Errorf("unsupported DevOps platform %q", platform)
}

// FindDevOpsPermissionMapping returns the mapping of the given role, and
// whether the platform has one.
func FindDevOpsPermissionMapping(ctx context.Context, client DevOpsPermissionMappingsClient, platform v1alpha1.DevOpsPlatform, role string) (sonar.DopTranslationPermissionMapping, bool, error) {
	result, resp, err := client.Search(ctx, platform) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return sonar.DopTranslationPermissionMapping{}, false, err
	}

	if result != nil {
		for _, mapping := range result.PermissionMappings {
			if mapping.Role == role {
				return mapping, true, nil
			}
		}
	}

	return sonar.DopTranslationPermissionMapping{}, false, nil
}

// GenerateDevOpsPermissionMappingObservation builds a
// DevOpsPermissionMappingObservation from the SonarQube API response.
func GenerateDevOpsPermissionMappingObservation(mapping sonar.DopTranslationPermissionMapping) v1alpha1.DevOpsPermissionMappingObservation {
	var permissions []string

	for _, permission := range devOpsPermissions {
		if *permission.field(&mapping.Permissions) {
			permissions = append(permissions, permission.name)
		}
	}

	return v1alpha1.DevOpsPermissionMappingObservation{
		ID:          mapping.Id,
		Role:        mapping.Role,
		BaseRole:    mapping.BaseRole,
		Permissions: permissions,
	}
}

// generateDevOpsPermissions converts a set of permission names to the API
// representation, in which every absent permission is false.
func generateDevOpsPermissions(names []string) sonar.DopTranslationPermissions {
	var permissions sonar.DopTranslationPermissions

	for _, permission := range devOpsPermissions {
		*permission.field(&permissions) = slices.Contains(names, permission.name)
	}

	return permissions
}

// GenerateCreateDevOpsPermissionMappingOptions builds the options creating the
// mapping described by params.
func GenerateCreateDevOpsPermissionMappingOptions(params *v1alpha1.DevOpsPermissionMappingParameters) *sonar.DopTranslationPermissionMappingCreateOptions {
	return &sonar.DopTranslationPermissionMappingCreateOptions{
		Role:        params.Role,
		Permissions: generateDevOpsPermissions(params.Permissions),
	}
}

// GenerateUpdateDevOpsPermissionMappingOptions builds the options setting every
// permission of the mapping to the state described by params.
func GenerateUpdateDevOpsPermissionMappingOptions(params *v1alpha1.DevOpsPermissionMappingParameters) *sonar.DopTranslationPermissionMappingUpdateOptions {
	permissions := generateDevOpsPermissions(params.Permissions)

	return &sonar.DopTranslationPermissionMappingUpdateOptions{
		Permissions: &sonar.DopTranslationPermissionsUpdate{
			User:                 new(permissions.User),
			CodeViewer:           new(permissions.CodeViewer),
			IssueAdmin:           new(permissions.IssueAdmin),
			SecurityHotspotAdmin: new(permissions.SecurityHotspotAdmin),
			Admin:                new(permissions.Admin),
			Scan:                 new(permissions.Scan),
		},
	}
}

// IsDevOpsPermissionMappingUpToDate returns true if the mapping grants exactly
// the desired permissions.
func IsDevOpsPermissionMappingUpToDate(params *v1alpha1.DevOpsPermissionMappingParameters, mapping sonar.DopTranslationPermissionMapping) bool {
	return generateDevOpsPermissions(params.Permissions) == mapping.Permissions
}
