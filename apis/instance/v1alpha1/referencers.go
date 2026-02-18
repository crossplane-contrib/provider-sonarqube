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
)

// ResolveReferences parses the references to other custom resources and resolves them to
// the actual values.
func (project *Project) ResolveReferences(ctx context.Context, client client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(client, project)

	// Resolve Quality Gate Name
	response, err := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
		CurrentValue: ptr.Deref(project.Spec.ForProvider.QualityGateName, ""),
		Reference:    project.Spec.ForProvider.QualityGateNameRef,
		Selector:     project.Spec.ForProvider.QualityGateNameSelector,
		To: reference.To{
			List:    &QualityGateList{},
			Managed: &QualityGate{},
		},
		Extract: reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.qualityGateName")
	}

	project.Spec.ForProvider.QualityGateName = &response.ResolvedValue
	project.Spec.ForProvider.QualityGateNameRef = response.ResolvedReference

	// Resolve Quality Profile Id for each language
	for language, profile := range project.Spec.ForProvider.QualityProfiles {
		response, err = resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: ptr.Deref(profile.Id, ""),
			Reference:    profile.IdRef,
			Selector:     profile.IdSelector,
			To: reference.To{
				List:    &QualityProfileList{},
				Managed: &QualityProfile{},
			},
			Extract: reference.ExternalName(),
		})
		if err != nil {
			return errors.Wrap(err, "spec.forProvider.qualityProfileId")
		}

		qualityProfile, ok := project.Spec.ForProvider.QualityProfiles[language]
		if !ok {
			return errors.Errorf("language %s not found in spec.forProvider.qualityProfiles", language)
		}

		qualityProfile.Id = &response.ResolvedValue
		qualityProfile.IdRef = response.ResolvedReference
		project.Spec.ForProvider.QualityProfiles[language] = qualityProfile
	}

	return nil
}
