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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// ResolveReferences resolves ProjectKey from a Project managed resource.
// If neither ProjectKeyRef nor ProjectKeySelector is set the direct
// ProjectKey value is used as-is and no resolution is performed.
func (webhook *Webhook) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	if webhook.Spec.ForProvider.ProjectKeyRef == nil && webhook.Spec.ForProvider.ProjectKeySelector == nil {
		return nil
	}

	resolver := reference.NewAPINamespacedResolver(readerClient, webhook)

	// Always pass CurrentValue="" when a ref/selector is set so the resolver
	// re-reads the referenced resource's external-name annotation on every
	// reconcile (prevents stale-cache issues during resource creation).
	response, err := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
		CurrentValue: "",
		Reference:    webhook.Spec.ForProvider.ProjectKeyRef,
		Selector:     webhook.Spec.ForProvider.ProjectKeySelector,
		To: reference.To{
			List:    &instancev1alpha1.ProjectList{},
			Managed: &instancev1alpha1.Project{},
		},
		Extract: reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.projectKey")
	}

	webhook.Spec.ForProvider.ProjectKey = ptr.To(response.ResolvedValue)
	webhook.Spec.ForProvider.ProjectKeyRef = response.ResolvedReference

	return nil
}
