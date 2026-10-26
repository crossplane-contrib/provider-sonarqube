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
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Update reconciles the delta between the desired and observed permissions.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	permissions, ok := mg.(*v1alpha1.Permissions)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPermissions)
	}

	externalName := meta.GetExternalName(permissions)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, permissions.Name)
	}

	toAdd, toRemove := computePermissionsDiff(permissions.Spec.ForProvider.Permissions, permissions.Status.AtProvider.Permissions)

	err := c.applyPermissions(ctx, permissions.Spec.ForProvider, toAdd, toRemove)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePermissions)
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// permissionCall sends one SonarQube request for a single permission.
type permissionCall func(permission string) (*http.Response, error)

// permissionCalls returns the subject described in spec and the calls that add
// and remove one of its permissions. ok is false when spec names no subject.
func (c *external) permissionCalls(ctx context.Context, spec v1alpha1.PermissionsParameters) (subject string, add, remove permissionCall, ok bool) {
	var projectKey *string
	if spec.ProjectKey != nil && *spec.ProjectKey != "" {
		projectKey = spec.ProjectKey
	}

	switch {
	case spec.GroupName != nil:
		groupName := *spec.GroupName
		add = func(permission string) (*http.Response, error) {
			return c.client.AddGroup(ctx, iam.GeneratePermissionsAddGroupOptions(groupName, permission, projectKey))
		}
		remove = func(permission string) (*http.Response, error) {
			return c.client.RemoveGroup(ctx, iam.GeneratePermissionsRemoveGroupOptions(groupName, permission, projectKey))
		}

		return "group " + groupName, add, remove, true
	case spec.Login != nil:
		login := *spec.Login
		add = func(permission string) (*http.Response, error) {
			return c.client.AddUser(ctx, iam.GeneratePermissionsAddUserOptions(login, permission, projectKey))
		}
		remove = func(permission string) (*http.Response, error) {
			return c.client.RemoveUser(ctx, iam.GeneratePermissionsRemoveUserOptions(login, permission, projectKey))
		}

		return "user " + login, add, remove, true
	default:
		return "", nil, nil, false
	}
}

// applyPermissions concurrently adds toAdd permissions and removes toRemove
// permissions for the subject described in spec.
// Either slice may be nil (no-op for that direction).
func (c *external) applyPermissions(ctx context.Context, spec v1alpha1.PermissionsParameters, toAdd, toRemove []string) error {
	subject, add, remove, ok := c.permissionCalls(ctx, spec)
	if !ok {
		return nil
	}

	total := len(toAdd) + len(toRemove)
	if total == 0 {
		return nil
	}

	errChan := make(chan error, total)

	// Bound the number of concurrent SonarQube requests: the permissions list
	// comes from the spec, so its length is controlled by the tenant.
	requestGroup := helpers.NewRequestGroup()

	schedule := func(permissions []string, call permissionCall, errFormat string) {
		for _, permission := range permissions {
			requestGroup.Go(func() error {
				resp, err := call(permission) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, errFormat, permission, subject)
				}

				return nil
			})
		}
	}

	schedule(toAdd, add, "cannot add permission %s to %s")
	schedule(toRemove, remove, "cannot remove permission %s from %s")

	// Errors are reported through errChan.
	_ = requestGroup.Wait()

	close(errChan)

	var errs []error

	for err := range errChan {
		errs = append(errs, err)
	}

	return stderrors.Join(errs...)
}
