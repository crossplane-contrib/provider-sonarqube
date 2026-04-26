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

package user

import (
	"context"
	"fmt"
	"sync"

	stderrors "errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Update is responsible for updating the external resource to match the
// desired state of the managed resource.
func (c *external) Update(_ context.Context, managedResource resource.Managed) (managed.ExternalUpdate, error) {
	const updateErrorCapacity = 2

	result := managed.ExternalUpdate{}

	userResource, ok := managedResource.(*v1alpha1.User)
	if !ok {
		return result, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(userResource)
	if externalName == "" {
		return result, fmt.Errorf(errExternalNameNotSet, userResource.Name)
	}

	updateWaitGroup := sync.WaitGroup{}
	errorSlice := make([]error, 0, updateErrorCapacity)
	errorSliceMutex := sync.Mutex{}

	updateWaitGroup.Go(func() {
		err := c.updateUserFields(userResource, externalName)
		if err != nil {
			errorSliceMutex.Lock()

			errorSlice = append(errorSlice, err)
			errorSliceMutex.Unlock()
		}
	})

	updateWaitGroup.Go(func() {
		err := c.reconcileGroupMemberships(userResource, externalName)
		if err != nil {
			errorSliceMutex.Lock()

			errorSlice = append(errorSlice, err)
			errorSliceMutex.Unlock()
		}
	})
	updateWaitGroup.Wait()

	return result, stderrors.Join(errorSlice...)
}

// updateUserFields updates the mutable fields of the User resource.
// It does not handle password or group membership updates.
func (c *external) updateUserFields(userResource *v1alpha1.User, externalName string) error {
	_, resp, err := c.usersClient.Update(externalName, iam.GenerateUpdateUserOptions(&userResource.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return errors.Wrap(err, "cannot update User")
	}

	return nil
}

// reconcileGroupMemberships ensures that the User's group memberships in
// SonarQube match the desired state specified in the User resource.
// It calculates the necessary additions and removals of group memberships
// and performs the required API calls to reconcile the state.
func (c *external) reconcileGroupMemberships(userResource *v1alpha1.User, externalName string) error {
	if userResource.Spec.ForProvider.Groups == nil {
		return nil
	}

	desiredGroups := desiredGroupIDs(userResource.Spec.ForProvider.Groups)
	toAdd, toRemove := buildGroupsDiff(desiredGroups, userResource.Status.AtProvider.Groups)

	membershipsWaitGroup := sync.WaitGroup{}
	membershipsWaitGroup.Add(len(toAdd) + len(toRemove))
	errorSlice := make([]error, 0, len(toAdd)+len(toRemove))
	errorSliceMutex := sync.Mutex{}

	for _, groupID := range toAdd {
		go func(groupID string) {
			defer membershipsWaitGroup.Done()

			created, resp, err := c.groupsClient.CreateGroupMembership(ptr.To(iam.GenerateGroupCreateMembershipOptions(groupID, externalName))) //nolint:bodyclose // closed via helpers.CloseBody
			defer helpers.CloseBody(resp)

			if err != nil {
				errorSliceMutex.Lock()

				errorSlice = append(errorSlice, errors.Wrapf(err, "cannot add User to Group %s", groupID))
				errorSliceMutex.Unlock()

				return
			}

			if created == nil || created.Id == "" {
				errorSliceMutex.Lock()

				errorSlice = append(errorSlice, fmt.Errorf("cannot add User to Group %s: create group membership returned empty membership ID", groupID))
				errorSliceMutex.Unlock()
			}
		}(groupID)
	}

	for _, groupMembershipID := range toRemove {
		go func(groupMembershipID string) {
			defer membershipsWaitGroup.Done()

			resp, err := c.groupsClient.DeleteGroupMembership(groupMembershipID) //nolint:bodyclose // closed via helpers.CloseBody
			defer helpers.CloseBody(resp)

			if err != nil {
				errorSliceMutex.Lock()

				errorSlice = append(errorSlice, errors.Wrapf(err, "cannot remove user membership %s", groupMembershipID))
				errorSliceMutex.Unlock()
			}
		}(groupMembershipID)
	}

	membershipsWaitGroup.Wait()

	return stderrors.Join(errorSlice...)
}

// desiredGroupIDs extracts the desired group IDs from the UserGroupsParameters.
// It filters out any nil or empty group IDs,
// returning a slice of valid group IDs that the user should be a member of.
func desiredGroupIDs(groups *[]v1alpha1.UserGroupsParameters) []string {
	if groups == nil {
		return nil
	}

	out := make([]string, 0, len(*groups))
	for _, group := range *groups {
		if group.GroupId == nil || *group.GroupId == "" {
			continue
		}

		out = append(out, *group.GroupId)
	}

	return out
}

// buildGroupsDiff compares the desired and observed group memberships for a
// user and returns two slices: one containing the ids of groups that the user
// should be added to (present in desired but not in observed),
// and another containing the membership ids of groups that the user should be
// removed from (present in observed but not in desired).
// This function is used to determine the necessary changes to reconcile
// the user's group memberships with the desired state.
func buildGroupsDiff(desired []string, observed map[string]string) (toAdd []string, toRemove []string) {
	desiredSet := helpers.NewStringSetFromSlice(desired)

	for groupId := range desiredSet {
		if _, exists := observed[groupId]; !exists {
			toAdd = append(toAdd, groupId)
		}
	}

	for groupId, membershipId := range observed {
		if _, exists := desiredSet[groupId]; !exists {
			toRemove = append(toRemove, membershipId)
		}
	}

	return toAdd, toRemove
}
