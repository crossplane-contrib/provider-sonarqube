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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences parses the references to other custom resources and
// resolves them to the actual values.
func (user *User) ResolveReferences(ctx context.Context, client client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(client, user)

	if user.Spec.ForProvider.Groups == nil {
		return nil
	}

	for groupIdx, userGroup := range *user.Spec.ForProvider.Groups {
		currentGroupID := ""
		if userGroup.GroupIdRef == nil && userGroup.GroupIdSelector == nil {
			currentGroupID = ptr.Deref(userGroup.GroupId, "")
		}

		var groupSelector *xpv1.NamespacedSelector
		if userGroup.GroupIdSelector != nil {
			groupSelector = &xpv1.NamespacedSelector{
				MatchLabels:        userGroup.GroupIdSelector.MatchLabels,
				MatchControllerRef: userGroup.GroupIdSelector.MatchControllerRef,
				Policy:             userGroup.GroupIdSelector.Policy,
				Namespace:          user.GetNamespace(),
			}
		}

		groupResponse, groupErr := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: currentGroupID,
			Reference:    userGroup.GroupIdRef,
			Selector:     groupSelector,
			To: reference.To{
				List:    &GroupList{},
				Managed: &Group{},
			},
			Extract: reference.ExternalName(),
		})
		if groupErr != nil {
			return errors.Wrap(groupErr, "spec.forProvider.groups.groupId")
		}

		if (userGroup.GroupIdRef != nil || userGroup.GroupIdSelector != nil) && groupResponse.ResolvedValue == "" {
			return errors.Errorf("unable to resolve spec.forProvider.groups[%d]: resolved value is empty", groupIdx)
		}

		group := userGroup
		group.GroupId = ptr.To(groupResponse.ResolvedValue)
		group.GroupIdRef = groupResponse.ResolvedReference
		(*user.Spec.ForProvider.Groups)[groupIdx] = group
	}

	return nil
}
