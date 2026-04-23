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

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const defaultMembershipPageSize = 500

// Observe is responsible for observing the external resource and determining if it needs to be created or updated.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	userResource, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(userResource)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, resp, err := c.usersClient.Fetch(externalName) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, "cannot observe User")
	}

	// Consider deactivated users as non-existent, as SonarQube does not provide a way to reactivate them and they cannot be updated. They can only be deleted and recreated if needed.
	if user == nil || !user.Active {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	userResource.Status.AtProvider = iam.GenerateUserObservation(user)

	userResource.SetConditions(xpv1.Available())

	groupMemberships, err := c.getUserGroupsObservation(externalName)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, err
	}

	userResource.Status.AtProvider.Groups = groupMemberships

	former := userResource.Spec.ForProvider.DeepCopy()
	iam.LateInitializeUser(&userResource.Spec.ForProvider, &userResource.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        iam.IsUserUpToDate(&userResource.Spec.ForProvider, &userResource.Status.AtProvider),
		ResourceLateInitialized: iam.IsUserLateInitialized(former, &userResource.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// passwordFromSecret retrieves the password for a local user from the Kubernetes secret specified in the UserParameters. If the PasswordSecretRef is not set and the user is local, it returns an error, as a password is required for creating a local user. If the user is not local, it returns an empty password without error, as a password is not required for non-local users.
//
//nolint:nilnil // Returning nil password for non-local users is expected.
func (c *external) passwordFromSecret(ctx context.Context, userResource *v1alpha1.User) (*string, error) {
	if !ptr.Deref(userResource.Spec.ForProvider.Local, false) {
		return nil, nil
	}

	if userResource.Spec.ForProvider.PasswordSecretRef == nil {
		return nil, errors.New("passwordSecretRef must be set for local user creation")
	}

	password, err := common.GetTokenValueFromSecret(ctx, c.kube, userResource, userResource.Spec.ForProvider.PasswordSecretRef)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get password from secret")
	}

	return password, nil
}

// getUserGroupsObservation retrieves the groups that the user belongs to from the SonarQube API. It handles pagination to ensure all groups are retrieved, and returns a map of group IDs to membership IDs for efficient lookup.
func (c *external) getUserGroupsObservation(userID string) (map[string]string, error) {
	var allGroups []sonar.GroupMembership

	pagination := &sonar.PaginationParamsV2{
		PageIndex: 1,
		PageSize:  defaultMembershipPageSize,
	}

	for {
		options := iam.GenerateGroupSearchMembershipsOptions(nil, &userID, pagination)

		foundMembership, resp, err := c.groupsClient.SearchGroupMemberships(&options) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return nil, errors.Wrap(err, "cannot fetch user groups")
		}

		allGroups = append(allGroups, foundMembership.GroupMemberships...)

		if pagination.PageIndex*pagination.PageSize >= foundMembership.Page.Total {
			break
		}

		pagination.PageIndex++
	}

	return iam.GenerateGroupMembershipObservation(&allGroups), nil
}
