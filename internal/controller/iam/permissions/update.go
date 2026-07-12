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
	"sync"

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

// applyPermissions concurrently adds toAdd permissions and removes toRemove
// permissions for the subject described in spec.
// Either slice may be nil (no-op for that direction).
//
//nolint:gocyclo,cyclop,nestif,funlen // Fan-out logic for two subject types and two directions requires nesting.
func (c *external) applyPermissions(ctx context.Context, spec v1alpha1.PermissionsParameters, toAdd, toRemove []string) error {
	var projectKey *string
	if spec.ProjectKey != nil && *spec.ProjectKey != "" {
		projectKey = spec.ProjectKey
	}

	total := len(toAdd) + len(toRemove)
	if total == 0 {
		return nil
	}

	errChan := make(chan error, total)

	var waitGroup sync.WaitGroup

	if spec.GroupName != nil {
		groupName := *spec.GroupName

		for _, perm := range toAdd {
			waitGroup.Add(1)

			go func(permission string) {
				defer waitGroup.Done()

				resp, err := c.client.AddGroup(ctx, iam.GeneratePermissionsAddGroupOptions(groupName, permission, projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot add permission %s to group %s", permission, groupName)
				}
			}(perm)
		}

		for _, perm := range toRemove {
			waitGroup.Add(1)

			go func(permission string) {
				defer waitGroup.Done()

				resp, err := c.client.RemoveGroup(ctx, iam.GeneratePermissionsRemoveGroupOptions(groupName, permission, projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot remove permission %s from group %s", permission, groupName)
				}
			}(perm)
		}
	} else if spec.Login != nil {
		login := *spec.Login

		for _, perm := range toAdd {
			waitGroup.Add(1)

			go func(permission string) {
				defer waitGroup.Done()

				resp, err := c.client.AddUser(ctx, iam.GeneratePermissionsAddUserOptions(login, permission, projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot add permission %s to user %s", permission, login)
				}
			}(perm)
		}

		for _, perm := range toRemove {
			waitGroup.Add(1)

			go func(permission string) {
				defer waitGroup.Done()

				resp, err := c.client.RemoveUser(ctx, iam.GeneratePermissionsRemoveUserOptions(login, permission, projectKey)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					errChan <- errors.Wrapf(err, "cannot remove permission %s from user %s", permission, login)
				}
			}(perm)
		}
	}

	waitGroup.Wait()
	close(errChan)

	var errs []error

	for err := range errChan {
		errs = append(errs, err)
	}

	return stderrors.Join(errs...)
}
