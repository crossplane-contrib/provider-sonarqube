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
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/utils/ptr"
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
// Portfolio: the parent portfolio, the selected projects and applications,
// and the referenced portfolios. References and selectors resolve in the
// Portfolio's own namespace.
func (portfolio *Portfolio) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, portfolio)
	params := &portfolio.Spec.ForProvider
	namespace := portfolio.GetNamespace()

	err := resolveKey(ctx, resolver, namespace, &params.Parent, &params.ParentRef, params.ParentSelector,
		reference.To{List: &PortfolioList{}, Managed: &Portfolio{}})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.parent")
	}

	for projectIdx := range ptr.Deref(params.Projects, nil) {
		project := &(*params.Projects)[projectIdx]

		err = resolveKey(ctx, resolver, namespace, &project.Key, &project.KeyRef, project.KeySelector,
			reference.To{List: &ProjectList{}, Managed: &Project{}})
		if err != nil {
			return errors.Wrapf(err, "spec.forProvider.projects[%d].key", projectIdx)
		}
	}

	for applicationIdx := range ptr.Deref(params.Applications, nil) {
		application := &(*params.Applications)[applicationIdx]

		err = resolveKey(ctx, resolver, namespace, &application.Key, &application.KeyRef, application.KeySelector,
			reference.To{List: &ApplicationList{}, Managed: &Application{}})
		if err != nil {
			return errors.Wrapf(err, "spec.forProvider.applications[%d].key", applicationIdx)
		}
	}

	return errors.Wrap(portfolio.resolveSubPortfolios(ctx, resolver), "spec.forProvider.subPortfolios")
}

// resolveKey resolves the external name of the managed resource referenced
// or selected into key, recording the resolved reference.
func resolveKey(ctx context.Context, resolver *reference.APIResolver, namespace string, key **string, ref **xpv1.Reference, selector *xpv1.Selector, target reference.To) error {
	response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(*key),
		Extract:      reference.ExternalName(),
		Namespace:    namespace,
		Reference:    *ref,
		Selector:     selector,
		To:           target,
	})
	if err != nil {
		return err
	}

	*key = reference.ToPtrValue(response.ResolvedValue)
	*ref = response.ResolvedReference

	return nil
}

// resolveSubPortfolios resolves the referenced portfolios of the Portfolio.
// They are not managed when no value, reference or selector is set.
func (portfolio *Portfolio) resolveSubPortfolios(ctx context.Context, resolver *reference.APIResolver) error {
	params := &portfolio.Spec.ForProvider
	if params.SubPortfolios == nil && len(params.SubPortfolioRefs) == 0 && params.SubPortfolioSelector == nil {
		return nil
	}

	response, err := resolver.ResolveMultiple(ctx, reference.MultiResolutionRequest{
		CurrentValues: ptr.Deref(params.SubPortfolios, nil),
		Extract:       reference.ExternalName(),
		Namespace:     portfolio.GetNamespace(),
		References:    params.SubPortfolioRefs,
		Selector:      params.SubPortfolioSelector,
		To: reference.To{
			List:    &PortfolioList{},
			Managed: &Portfolio{},
		},
	})
	if err != nil {
		return err
	}

	// A nil slice would be serialized as null, leaving the referenced
	// portfolios unmanaged instead of removing them all.
	subPortfolios := response.ResolvedValues
	if subPortfolios == nil {
		subPortfolios = []string{}
	}

	params.SubPortfolios = &subPortfolios
	params.SubPortfolioRefs = response.ResolvedReferences

	return nil
}

// ResolveReferences resolves references to other managed resources for
// QualityProfile. References and selectors resolve in the QualityProfile's
// own namespace.
func (qualityProfile *QualityProfile) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, qualityProfile)

	err := qualityProfile.resolveParent(ctx, resolver)
	if err != nil {
		return err
	}

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

// resolveParent resolves the reference or selector to the parent
// QualityProfile into its display name.
func (qualityProfile *QualityProfile) resolveParent(ctx context.Context, resolver *reference.APIResolver) error {
	params := &qualityProfile.Spec.ForProvider

	// Without a reference there is nothing to resolve, and writing the result
	// back would turn an empty parent (no parent) into an unset one (unmanaged).
	if params.ParentRef == nil && params.ParentSelector == nil {
		return nil
	}

	response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(params.Parent),
		Extract:      QualityProfileName(),
		Namespace:    qualityProfile.GetNamespace(),
		Reference:    params.ParentRef,
		Selector:     params.ParentSelector,
		To: reference.To{
			List:    &QualityProfileList{},
			Managed: &QualityProfile{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.parent")
	}

	// A selector may match the profile itself, which cannot be its own parent.
	if response.ResolvedReference != nil && response.ResolvedReference.Name == qualityProfile.GetName() {
		return errors.New("spec.forProvider.parent: a QualityProfile cannot inherit from itself")
	}

	params.Parent = reference.ToPtrValue(response.ResolvedValue)
	params.ParentRef = response.ResolvedReference

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
