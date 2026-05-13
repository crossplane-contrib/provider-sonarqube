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

package v1alpha1

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences resolves references to other managed resources for User.
// Resolution respects the ref/selector Policy: with the default IfNotPresent
// policy a resolved value is cached and not re-fetched on every reconcile;
// set policy.resolve=Always on a reference to force re-resolution each time.
func (user *User) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(readerClient, user)

	if user.Spec.ForProvider.Groups == nil {
		return nil
	}

	for groupIdx, userGroup := range *user.Spec.ForProvider.Groups {
		groupResponse, groupErr := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: reference.FromPtrValue(userGroup.GroupId),
			Extract:      reference.ExternalName(),
			Namespace:    user.GetNamespace(),
			Reference:    userGroup.GroupIdRef,
			Selector:     userGroup.GroupIdSelector,
			To: reference.To{
				List:    &GroupList{},
				Managed: &Group{},
			},
		})
		if groupErr != nil {
			return errors.Wrap(groupErr, "spec.forProvider.groups.groupId")
		}

		userGroup.GroupId = reference.ToPtrValue(groupResponse.ResolvedValue)
		userGroup.GroupIdRef = groupResponse.ResolvedReference
		(*user.Spec.ForProvider.Groups)[groupIdx] = userGroup
	}

	return nil
}
