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

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// ResolveReferences resolves references to other managed resources for
// Webhook. References and selectors resolve in the Webhook's own namespace.
func (webhook *Webhook) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, webhook)

	response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(webhook.Spec.ForProvider.ProjectKey),
		Extract:      reference.ExternalName(),
		Namespace:    webhook.GetNamespace(),
		Reference:    webhook.Spec.ForProvider.ProjectKeyRef,
		Selector:     webhook.Spec.ForProvider.ProjectKeySelector,
		To: reference.To{
			List:    &instancev1alpha1.ProjectList{},
			Managed: &instancev1alpha1.Project{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.projectKey")
	}

	webhook.Spec.ForProvider.ProjectKey = reference.ToPtrValue(response.ResolvedValue)
	webhook.Spec.ForProvider.ProjectKeyRef = response.ResolvedReference

	return nil
}
