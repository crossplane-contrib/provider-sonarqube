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

// ResolveReferences resolves references to other managed resources for Project.
// Resolution respects the ref/selector Policy: with the default IfNotPresent
// policy a resolved value is cached and not re-fetched on every reconcile;
// set policy.resolve=Always on a reference to force re-resolution each time.
func (project *Project) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(readerClient, project)

	response, err := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
		CurrentValue: reference.FromPtrValue(project.Spec.ForProvider.QualityGateName),
		Extract:      reference.ExternalName(),
		Namespace:    project.GetNamespace(),
		Reference:    project.Spec.ForProvider.QualityGateNameRef,
		Selector:     project.Spec.ForProvider.QualityGateNameSelector,
		To: reference.To{
			List:    &QualityGateList{},
			Managed: &QualityGate{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.qualityGateName")
	}

	project.Spec.ForProvider.QualityGateName = reference.ToPtrValue(response.ResolvedValue)
	project.Spec.ForProvider.QualityGateNameRef = response.ResolvedReference

	for language, profile := range project.Spec.ForProvider.QualityProfiles {
		profileResponse, profileErr := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: reference.FromPtrValue(profile.Id),
			Extract:      reference.ExternalName(),
			Namespace:    project.GetNamespace(),
			Reference:    profile.IdRef,
			Selector:     profile.IdSelector,
			To: reference.To{
				List:    &QualityProfileList{},
				Managed: &QualityProfile{},
			},
		})
		if profileErr != nil {
			return errors.Wrap(profileErr, "spec.forProvider.qualityProfileId")
		}

		qualityProfile := project.Spec.ForProvider.QualityProfiles[language]
		qualityProfile.Id = reference.ToPtrValue(profileResponse.ResolvedValue)
		qualityProfile.IdRef = profileResponse.ResolvedReference
		project.Spec.ForProvider.QualityProfiles[language] = qualityProfile
	}

	return nil
}
