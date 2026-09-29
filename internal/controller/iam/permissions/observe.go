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

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
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
		observedPermissions, found, err = iam.GroupPermissions(ctx, c.client, c.cache, subject, projectKey)
	} else {
		observedPermissions, found, err = iam.UserPermissions(ctx, c.client, c.cache, subject, projectKey)
	}

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObservePermissions)
	}

	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Deleting: Check that the resource has no permissions left; mark the
	// external resource as non-existent so the managed reconciler can
	// remove the finalizer and allow the CR to be deleted.
	if !permissions.DeletionTimestamp.IsZero() && len(observedPermissions) == 0 {
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

// computePermissionsDiff returns (toAdd, toRemove) to converge from
// observed to desired.
func computePermissionsDiff(desired, observed []string) (toAdd, toRemove []string) {
	return helpers.StringSetDifference(desired, observed), helpers.StringSetDifference(observed, desired)
}
