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

package permissions

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Observe checks whether the external Permissions resource exists and
// is up to date.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	permissions, ok := mg.(*v1alpha1.Permissions)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPermissions)
	}

	externalName := meta.GetExternalName(permissions)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// parseExternalName returns "" for subjectType on any error (invalid
	// format or unknown type). Crossplane defaults the external name to
	// metadata.name before Create runs, so an unparseable name means the
	// resource does not exist yet - trigger Create to set the proper name.
	subjectType, subject, projectKey, _ := parseExternalName(externalName)
	if subjectType == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	var (
		observedPermissions []string
		found               bool
		err                 error
	)

	if subjectType == subjectTypeGroup {
		observedPermissions, found, err = getObservedGroupPermissions(c.client, subject, projectKey)
	} else {
		observedPermissions, found, err = getObservedUserPermissions(c.client, subject, projectKey)
	}

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObservePermissions)
	}

	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	permissions.Status.AtProvider.Permissions = observedPermissions

	former := permissions.Spec.ForProvider.DeepCopy()
	iam.LateInitializePermissions(&permissions.Spec.ForProvider, &permissions.Status.AtProvider)

	permissions.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        iam.IsPermissionsUpToDate(&permissions.Spec.ForProvider, &permissions.Status.AtProvider),
		ResourceLateInitialized: iam.IsPermissionsLateInitialized(former, &permissions.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// getObservedGroupPermissions retrieves the currently assigned permissions
// for a group, paginating through results until the group is found or
// exhausted.
// Returns (permissions, true, nil) when found, (nil, false, nil) when not
// found.
//
//nolint:dupl // Intentional structural similarity with getObservedUserPermissions; different API types prevent abstraction.
func getObservedGroupPermissions(permClient iam.PermissionsClient, groupName, projectKey string) ([]string, bool, error) { //nolint:gocritic // named results not helpful for pagination loops
	const maxPageSize = int64(100)

	var projectKeyPtr *string
	if projectKey != "" {
		projectKeyPtr = &projectKey
	}

	for page := int64(1); ; page++ {
		opts := iam.GeneratePermissionsGroupsOptions(groupName, projectKeyPtr, &sonar.PaginationArgs{Page: page, PageSize: maxPageSize})

		result, resp, err := permClient.Groups(opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, false, errors.Wrap(err, "cannot list group permissions")
		}

		for _, g := range result.Groups {
			if g.Name == groupName {
				return g.Permissions, true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return nil, false, nil
		}
	}
}

// getObservedUserPermissions retrieves the currently assigned permissions
// for a user, paginating through results until the user is found or
// exhausted.
// Returns (permissions, true, nil) when found, (nil, false, nil) when not
// found.
//
//nolint:dupl // Intentional structural similarity with getObservedGroupPermissions; different API types prevent abstraction.
func getObservedUserPermissions(permClient iam.PermissionsClient, login, projectKey string) ([]string, bool, error) { //nolint:gocritic // named results not helpful for pagination loops
	const maxPageSize = int64(100)

	var projectKeyPtr *string
	if projectKey != "" {
		projectKeyPtr = &projectKey
	}

	for page := int64(1); ; page++ {
		opts := iam.GeneratePermissionsUsersOptions(login, projectKeyPtr, &sonar.PaginationArgs{Page: page, PageSize: maxPageSize})

		result, resp, err := permClient.Users(opts) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, false, errors.Wrap(err, "cannot list user permissions")
		}

		for _, u := range result.Users {
			if u.Login == login {
				return u.Permissions, true, nil
			}
		}

		if result.Paging.Total <= result.Paging.PageIndex*result.Paging.PageSize {
			return nil, false, nil
		}
	}
}

// computePermissionsDiff returns (toAdd, toRemove) to converge from
// observed to desired.
func computePermissionsDiff(desired, observed []string) (toAdd, toRemove []string) {
	desiredSet := helpers.NewStringSetFromSlice(desired)
	observedSet := helpers.NewStringSetFromSlice(observed)

	for _, p := range desired {
		if _, ok := observedSet[p]; !ok {
			toAdd = append(toAdd, p)
		}
	}

	for _, p := range observed {
		if _, ok := desiredSet[p]; !ok {
			toRemove = append(toRemove, p)
		}
	}

	return toAdd, toRemove
}
