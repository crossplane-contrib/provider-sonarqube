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
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences resolves references to other managed resources for Project.
// References and selectors resolve in the Project's own namespace.
// Resolution respects the ref/selector Policy: with the default IfNotPresent
// policy a resolved value is cached and not re-fetched on every reconcile;
// set policy.resolve=Always on a reference to force re-resolution each time.
func (project *Project) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, project)

	response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
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
		profileResponse, profileErr := resolver.Resolve(ctx, reference.ResolutionRequest{
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
			return errors.Wrapf(profileErr, "spec.forProvider.qualityProfiles[%s].id", language)
		}

		qualityProfile := project.Spec.ForProvider.QualityProfiles[language]
		qualityProfile.Id = reference.ToPtrValue(profileResponse.ResolvedValue)
		qualityProfile.IdRef = profileResponse.ResolvedReference
		project.Spec.ForProvider.QualityProfiles[language] = qualityProfile
	}

	return nil
}

// ResolveReferences resolves references to other managed resources for
// Application. References and selectors resolve in the Application's own
// namespace.
func (application *Application) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, application)

	response, err := resolver.ResolveMultiple(ctx, reference.MultiResolutionRequest{
		CurrentValues: application.Spec.ForProvider.Projects,
		Extract:       reference.ExternalName(),
		Namespace:     application.GetNamespace(),
		References:    application.Spec.ForProvider.ProjectRefs,
		Selector:      application.Spec.ForProvider.ProjectSelector,
		To: reference.To{
			List:    &ProjectList{},
			Managed: &Project{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.projects")
	}

	application.Spec.ForProvider.Projects = response.ResolvedValues
	application.Spec.ForProvider.ProjectRefs = response.ResolvedReferences

	return nil
}

// ResolveReferences resolves references to other managed resources for
// QualityProfile. References and selectors resolve in the QualityProfile's
// own namespace.
func (qualityProfile *QualityProfile) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, qualityProfile)

	for ruleIdx := range qualityProfile.Spec.ForProvider.Rules {
		rule := &qualityProfile.Spec.ForProvider.Rules[ruleIdx]

		response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: reference.FromPtrValue(rule.Rule),
			Extract:      reference.ExternalName(),
			Namespace:    qualityProfile.GetNamespace(),
			Reference:    rule.RuleRef,
			Selector:     rule.RuleSelector,
			To: reference.To{
				List:    &RuleList{},
				Managed: &Rule{},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "spec.forProvider.rules[%d].rule", ruleIdx)
		}

		rule.Rule = reference.ToPtrValue(response.ResolvedValue)
		rule.RuleRef = response.ResolvedReference
	}

	return nil
}

// QualityProfileName extracts the display name of a referenced
// QualityProfile. Association APIs identify a profile by display name,
// not by the profile key stored as the external name.
func QualityProfileName() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		qp, ok := mg.(*QualityProfile)
		if !ok {
			return ""
		}

		return qp.Spec.ForProvider.Name
	}
}
