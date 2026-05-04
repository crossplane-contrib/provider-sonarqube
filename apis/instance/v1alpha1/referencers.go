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

// ResolveReferences parses the references to other custom
// resources and resolves them to the actual values.
func (project *Project) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(readerClient, project)

	// Resolve Quality Gate Name.
	//
	// CurrentValue caching: the resolver treats a non-empty CurrentValue as a
	// cache hit and skips re-fetching the referenced resource.  We must only
	// pass the cached value when no ref/selector is set (i.e. the user has
	// provided the value directly).  When a ref or selector IS present we
	// always pass "" so the resolver re-reads the referenced resource's
	// crossplane.io/external-name annotation on every reconcile.  This is
	// necessary because Crossplane's NameAsExternalName initializer first sets
	// the annotation to the K8s metadata.name, and the controller later
	// overwrites it with the true SonarQube identifier once the resource has
	// been created; without re-resolution the stale K8s name would be cached
	// forever.
	currentGateName := ""
	if project.Spec.ForProvider.QualityGateNameRef == nil && project.Spec.ForProvider.QualityGateNameSelector == nil {
		currentGateName = ptr.Deref(project.Spec.ForProvider.QualityGateName, "")
	}

	response, err := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
		CurrentValue: currentGateName,
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

	// Use ptr.To to allocate an independent copy of the resolved string.
	// Storing &response.ResolvedValue would be a pointer into the local response
	// struct; any subsequent resolver.Resolve call reassigns that struct in-place,
	// silently overwriting every field pointer that was stored from a previous call.
	project.Spec.ForProvider.QualityGateName = new(response.ResolvedValue)
	project.Spec.ForProvider.QualityGateNameRef = response.ResolvedReference

	// Resolve Quality Profile Id for each language.
	// Same CurrentValue caching logic as above: bypass the cache when a ref or
	// selector is set so that the true external name is always read from the
	// referenced resource's annotation rather than from a potentially stale
	// cached value.
	for language, profile := range project.Spec.ForProvider.QualityProfiles {
		currentProfileID := ""
		if profile.IdRef == nil && profile.IdSelector == nil {
			currentProfileID = ptr.Deref(profile.Id, "")
		}

		profileResponse, profileErr := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: currentProfileID,
			Reference:    profile.IdRef,
			Selector:     profile.IdSelector,
			To: reference.To{
				List:    &QualityProfileList{},
				Managed: &QualityProfile{},
			},
			Extract: reference.ExternalName(),
		})
		if profileErr != nil {
			return errors.Wrap(profileErr, "spec.forProvider.qualityProfileId")
		}

		qualityProfile, ok := project.Spec.ForProvider.QualityProfiles[language]
		if !ok {
			return errors.Errorf("language %s not found in spec.forProvider.qualityProfiles", language)
		}

		// ptr.To allocates an independent copy per language, preventing each loop
		// iteration from overwriting pointers stored in previous iterations.
		qualityProfile.Id = new(profileResponse.ResolvedValue)
		qualityProfile.IdRef = profileResponse.ResolvedReference
		project.Spec.ForProvider.QualityProfiles[language] = qualityProfile
	}

	return nil
}

// ResolveReferences parses the references to other custom resources and
// resolves them to
// the actual values.
func (qualityProfile *QualityProfile) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPINamespacedResolver(readerClient, qualityProfile)

	// Resolve Rule for each profile rule.
	for ruleIdx, profileRule := range qualityProfile.Spec.ForProvider.Rules {
		currentRuleKey := ""
		if profileRule.RuleRef == nil && profileRule.RuleSelector == nil {
			currentRuleKey = ptr.Deref(profileRule.Rule, "")
		}

		ruleResponse, ruleErr := resolver.Resolve(ctx, reference.NamespacedResolutionRequest{
			CurrentValue: currentRuleKey,
			Reference:    profileRule.RuleRef,
			Selector:     profileRule.RuleSelector,
			To: reference.To{
				List:    &RuleList{},
				Managed: &Rule{},
			},
			Extract: reference.ExternalName(),
		})
		if ruleErr != nil {
			return errors.Wrap(ruleErr, "spec.forProvider.rules.rule")
		} else if ruleResponse.ResolvedValue == "" {
			return errors.Errorf("unable to resolve spec.forProvider.rules[%d]: resolved value is empty", ruleIdx)
		}

		rule := qualityProfile.Spec.ForProvider.Rules[ruleIdx]

		// ptr.To allocates an independent copy per rule, preventing each loop iteration from overwriting pointers stored in previous iterations.
		rule.Rule = new(ruleResponse.ResolvedValue)
		rule.RuleRef = ruleResponse.ResolvedReference
		qualityProfile.Spec.ForProvider.Rules[ruleIdx] = rule
	}

	return nil
}
