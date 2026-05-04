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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"

	reference "github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	errors "github.com/pkg/errors"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences of this Webhook.
func (mg *Webhook) ResolveReferences(ctx context.Context, c client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(c, mg)

	var (
		rsp reference.NamespacedResolutionResponse
		err error
	)

	currentValue := reference.FromPtrValue(mg.Spec.ForProvider.ProjectKey)
	if mg.Spec.ForProvider.ProjectKeyRef != nil || mg.Spec.ForProvider.ProjectKeySelector != nil {
		currentValue = ""
	}

	rsp, err = resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
		CurrentValue: currentValue,
		Extract:      reference.ExternalName(),
		Namespace:    mg.GetNamespace(),
		Reference:    mg.Spec.ForProvider.ProjectKeyRef,
		Selector:     mg.Spec.ForProvider.ProjectKeySelector,
		To: reference.To{
			List:    &v1alpha1.ProjectList{},
			Managed: &v1alpha1.Project{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to resolve Webhook.Spec.ForProvider.ProjectKey")
	}

	mg.Spec.ForProvider.ProjectKey = reference.ToPtrValue(rsp.ResolvedValue)
	mg.Spec.ForProvider.ProjectKeyRef = rsp.ResolvedReference

	return nil
}
