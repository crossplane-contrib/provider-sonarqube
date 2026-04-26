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

package permissionstemplate

import (
	"context"
	"errors"
	"fmt"
	"sync"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	pkgerrors "github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Update updates the PermissionsTemplate external resource using the
// SonarQube API, based on the desired state of the managed resource.
// It returns an ExternalUpdate which may include connection details
// to be stored in a secret.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	permissionsTemplate, ok := mg.(*v1alpha1.PermissionsTemplate)
	if !ok {
		return managed.ExternalUpdate{}, pkgerrors.New(errNotPermissionsTemplate)
	}

	externalName := meta.GetExternalName(permissionsTemplate)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf("external name is not set for PermissionsTemplate %s", permissionsTemplate.Name)
	}

	err := c.reconcilePermissionsTemplate(externalName, permissionsTemplate)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	permissionsTemplate.SetConditions(xpv1.ReconcileSuccess())

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// reconcilePermissionsTemplate applies all update operations needed
// to reach desired state.
func (c *external) reconcilePermissionsTemplate(templateID string, permissionsTemplate *v1alpha1.PermissionsTemplate) error {
	var (
		reconcileWaitGroup sync.WaitGroup
		aggregatedErrors   []error
		errorsMutex        sync.Mutex
	)

	reconcileTask := func(fn func() error, msg string) {
		reconcileWaitGroup.Go(func() {
			err := fn()
			if err != nil {
				errorsMutex.Lock()

				aggregatedErrors = append(aggregatedErrors, pkgerrors.Wrap(err, msg))

				errorsMutex.Unlock()
			}
		})
	}

	reconcileTask(func() error {
		return c.updateTemplateBaseFields(templateID, permissionsTemplate)
	}, "failed to update PermissionsTemplate")

	reconcileTask(func() error {
		return c.setTemplateAsDefaultIfNeeded(templateID, permissionsTemplate)
	}, "failed to set PermissionsTemplate as default")

	reconcileTask(func() error {
		return c.updatePermissionsTemplateUsers(templateID, permissionsTemplate.Spec.ForProvider.UserPermissions, &permissionsTemplate.Status.AtProvider.UserPermissions)
	}, "failed to update PermissionsTemplate users permissions")

	reconcileTask(func() error {
		return c.updatePermissionsTemplateGroups(templateID, permissionsTemplate.Spec.ForProvider.GroupPermissions, &permissionsTemplate.Status.AtProvider.GroupPermissions)
	}, "failed to update PermissionsTemplate groups permissions")

	reconcileTask(func() error {
		return c.updatePermissionsTemplateCreator(templateID, permissionsTemplate.Spec.ForProvider.CreatorPermissions, permissionsTemplate.Status.AtProvider.CreatorPermissions)
	}, "failed to update PermissionsTemplate creator permissions")

	reconcileWaitGroup.Wait()

	err := errors.Join(aggregatedErrors...)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to reconcile PermissionsTemplate")
	}

	return nil
}

// updateTemplateBaseFields updates mutable template metadata
// when drift is detected.
func (c *external) updateTemplateBaseFields(templateID string, permissionsTemplate *v1alpha1.PermissionsTemplate) error {
	// Avoid unnecessary API updates when the desired base fields already match the observed state.
	if iam.ArePermissionsTemplateBaseFieldsUpToDate(&permissionsTemplate.Spec.ForProvider, &permissionsTemplate.Status.AtProvider) {
		return nil
	}

	updateOptions := iam.GeneratePermissionsTemplateUpdateOptions(templateID, &permissionsTemplate.Spec.ForProvider)
	_, resp, err := c.client.UpdateTemplate(updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return err
	}

	return nil
}

// setTemplateAsDefaultIfNeeded marks the template as default
// only when requested and not already default.
func (c *external) setTemplateAsDefaultIfNeeded(templateID string, permissionsTemplate *v1alpha1.PermissionsTemplate) error {
	if permissionsTemplate.Spec.ForProvider.Default == nil || !*permissionsTemplate.Spec.ForProvider.Default || permissionsTemplate.Status.AtProvider.Default {
		return nil
	}

	defaultOptions := iam.GeneratePermissionsTemplateSetAsDefaultOptions(templateID)
	resp, err := c.client.SetDefaultTemplate(defaultOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return err
	}

	return nil
}

// applyTemplatePermissions applies add and remove permission
// operations concurrently.
func (c *external) applyTemplatePermissions(addPermissions, removePermissions []string, addFn, removeFn func(permission string) error) error {
	var (
		permissionsWaitGroup sync.WaitGroup
		aggregatedErrors     []error
		errorsMutex          sync.Mutex
	)

	for _, permission := range addPermissions {
		perm := permission

		permissionsWaitGroup.Go(func() {
			err := addFn(perm)
			if err != nil {
				errorsMutex.Lock()

				aggregatedErrors = append(aggregatedErrors, err)

				errorsMutex.Unlock()
			}
		})
	}

	for _, permission := range removePermissions {
		perm := permission

		permissionsWaitGroup.Go(func() {
			err := removeFn(perm)
			if err != nil {
				errorsMutex.Lock()

				aggregatedErrors = append(aggregatedErrors, err)

				errorsMutex.Unlock()
			}
		})
	}

	permissionsWaitGroup.Wait()

	return errors.Join(aggregatedErrors...)
}

// updatePermissionsTemplateGroups updates the groups permissions of a
// PermissionsTemplate based on the difference between the spec and the
// observation.
// It adds the permissions that are in the spec but not in the observation,
// and removes the permissions that are in the observation but not in the spec.
// It returns an error if any occurred during the update.
//
//nolint:dupl // Group and user reconciliation intentionally share the same control flow with different API endpoints.
func (c *external) updatePermissionsTemplateGroups(templateID string, specGroups *[]v1alpha1.PermissionsTemplateGroupParameters, observationGroups *[]v1alpha1.PermissionsTemplateGroupObservation) error {
	if specGroups == nil || observationGroups == nil {
		return nil
	}

	groupsMapping := iam.CreatePermissionsTemplateGroupMapping(specGroups, observationGroups)

	return c.updateMappedPermissions(
		groupsMapping,
		func(specIndex int) []string {
			if specIndex < 0 || (*specGroups)[specIndex].Permissions == nil {
				return nil
			}

			return *(*specGroups)[specIndex].Permissions
		},
		func(obsIndex int) []string {
			if obsIndex < 0 {
				return nil
			}

			return (*observationGroups)[obsIndex].Permissions
		},
		func(subjectID string, permission string) error {
			addOptions := iam.GeneratePermissionsTemplateAddGroupPermissionOptions(templateID, subjectID, permission)
			resp, err := c.client.AddGroupToTemplate(addOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to add permission %s to group %s for PermissionsTemplate with id %s", permission, subjectID, templateID)
			}

			return nil
		},
		func(subjectID string, permission string) error {
			removeOptions := iam.GeneratePermissionsTemplateRemoveGroupPermissionOptions(templateID, subjectID, permission)
			resp, err := c.client.RemoveGroupFromTemplate(removeOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to remove permission %s from group %s for PermissionsTemplate with id %s", permission, subjectID, templateID)
			}

			return nil
		},
	)
}

// updatePermissionsTemplateUsers updates users in a permissions template.
//
//nolint:dupl // Group and user reconciliation intentionally share the same control flow with different API endpoints.
func (c *external) updatePermissionsTemplateUsers(templateID string, specUsers *[]v1alpha1.PermissionsTemplateUserParameters, observationUsers *[]v1alpha1.PermissionsTemplateUserObservation) error {
	if specUsers == nil || observationUsers == nil {
		return nil
	}

	usersMapping := iam.CreatePermissionsTemplateUserMapping(specUsers, observationUsers)

	return c.updateMappedPermissions(
		usersMapping,
		func(specIndex int) []string {
			if specIndex < 0 || (*specUsers)[specIndex].Permissions == nil {
				return nil
			}

			return *(*specUsers)[specIndex].Permissions
		},
		func(obsIndex int) []string {
			if obsIndex < 0 {
				return nil
			}

			return (*observationUsers)[obsIndex].Permissions
		},
		func(subjectID string, permission string) error {
			addOptions := iam.GeneratePermissionsTemplateAddUserPermissionOptions(templateID, subjectID, permission)
			resp, err := c.client.AddUserToTemplate(addOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to add permission %s to user %s for PermissionsTemplate with id %s", permission, subjectID, templateID)
			}

			return nil
		},
		func(subjectID string, permission string) error {
			removeOptions := iam.GeneratePermissionsTemplateRemoveUserPermissionOptions(templateID, subjectID, permission)
			resp, err := c.client.RemoveUserFromTemplate(removeOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to remove permission %s from user %s for PermissionsTemplate with id %s", permission, subjectID, templateID)
			}

			return nil
		},
	)
}

// updateMappedPermissions reconciles permissions for mapped subjects
// using provided accessors.
func (c *external) updateMappedPermissions(
	mapping map[string][2]int,
	getSpecPermissions func(specIndex int) []string,
	getObservedPermissions func(obsIndex int) []string,
	addPermission func(subjectID string, permission string) error,
	removePermission func(subjectID string, permission string) error,
) error {
	return c.updatePermissionsTemplateSubject(mapping, getSpecPermissions, getObservedPermissions, addPermission, removePermission)
}

// updatePermissionsTemplateSubject reconciles each subject's
// permissions and aggregates failures.
func (c *external) updatePermissionsTemplateSubject(
	mapping map[string][2]int,
	getSpecPermissions func(specIndex int) []string,
	getObservedPermissions func(obsIndex int) []string,
	addPermission func(subjectID string, permission string) error,
	removePermission func(subjectID string, permission string) error,
) error {
	aggregatedErrors := make([]error, 0)

	for subjectID, indexes := range mapping {
		specPermissions := getSpecPermissions(indexes[0])
		observedPermissions := getObservedPermissions(indexes[1])

		permissionsToAdd, permissionsToRemove := c.computePermissionsDiff(specPermissions, observedPermissions)
		if len(permissionsToAdd) == 0 && len(permissionsToRemove) == 0 {
			continue
		}

		err := c.applyTemplatePermissions(
			permissionsToAdd,
			permissionsToRemove,
			func(permission string) error {
				return addPermission(subjectID, permission)
			},
			func(permission string) error {
				return removePermission(subjectID, permission)
			},
		)
		if err != nil {
			aggregatedErrors = append(aggregatedErrors, err)
		}
	}

	return errors.Join(aggregatedErrors...)
}

// computePermissionsDiff computes the permissions to add and remove for a user
// based on the spec and observation.
// It returns two slices,
// one with the permissions to add and one with the permissions to remove.
func (c *external) computePermissionsDiff(specPermissions, obsPermissions []string) (permissionsToAdd, permissionsToRemove []string) {
	specPerms := helpers.NewStringSetFromSlice(specPermissions)
	obsPerms := helpers.NewStringSetFromSlice(obsPermissions)

	for _, perm := range specPermissions {
		if _, exists := obsPerms[perm]; !exists {
			permissionsToAdd = append(permissionsToAdd, perm)
		}
	}

	for _, perm := range obsPermissions {
		if _, exists := specPerms[perm]; !exists {
			permissionsToRemove = append(permissionsToRemove, perm)
		}
	}

	return permissionsToAdd, permissionsToRemove
}

// updatePermissionsTemplateCreator updates the creator permissions of a
// PermissionsTemplate based on the difference between the spec and
// the observation. It adds the permissions that are in the spec but not in the
// observation, and removes the permissions that are in the observation but not
// in the spec. It returns an error if any occurred during the update.
func (c *external) updatePermissionsTemplateCreator(templateID string, spec *[]string, observation []string) error {
	if spec == nil {
		return nil
	}

	permissionsToAdd, permissionsToRemove := c.computePermissionsDiff(*spec, observation)

	return c.applyTemplatePermissions(
		permissionsToAdd,
		permissionsToRemove,
		func(permission string) error {
			addOptions := iam.GeneratePermissionsTemplateAddCreatorPermissionOptions(templateID, permission)
			resp, err := c.client.AddProjectCreatorToTemplate(addOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to add creator permission %s to PermissionsTemplate with id %s", permission, templateID)
			}

			return nil
		},
		func(permission string) error {
			removeOptions := iam.GeneratePermissionsTemplateRemoveCreatorPermissionOptions(templateID, permission)
			resp, err := c.client.RemoveProjectCreatorFromTemplate(removeOptions) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				return pkgerrors.Wrapf(err, "failed to remove creator permission %s from PermissionsTemplate with id %s", permission, templateID)
			}

			return nil
		},
	)
}
