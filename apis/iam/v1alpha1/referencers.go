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
	resource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// ResolveReferences resolves references to other managed resources for User.
// References and selectors resolve in the User's own namespace.
// Resolution respects the ref/selector Policy: with the default IfNotPresent
// policy a resolved value is cached and not re-fetched on every reconcile;
// set policy.resolve=Always on a reference to force re-resolution each time.
func (user *User) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, user)

	if user.Spec.ForProvider.Groups == nil {
		return nil
	}

	for groupIdx, userGroup := range *user.Spec.ForProvider.Groups {
		groupResponse, groupErr := resolver.Resolve(ctx, reference.ResolutionRequest{
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
			return errors.Wrapf(groupErr, "spec.forProvider.groups[%d].groupId", groupIdx)
		}

		userGroup.GroupId = reference.ToPtrValue(groupResponse.ResolvedValue)
		userGroup.GroupIdRef = groupResponse.ResolvedReference
		(*user.Spec.ForProvider.Groups)[groupIdx] = userGroup
	}

	return nil
}

// ResolveReferences resolves references to other managed resources for
// Permissions. References and selectors resolve in the Permissions' own
// namespace.
func (permissions *Permissions) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, permissions)

	err := resolveGroupName(ctx, resolver, permissions.GetNamespace(), &permissions.Spec.ForProvider.GroupName, &permissions.Spec.ForProvider.GroupNameRef, permissions.Spec.ForProvider.GroupNameSelector)
	if err != nil {
		return err
	}

	err = resolveUserLogin(ctx, resolver, permissions.GetNamespace(), &permissions.Spec.ForProvider.Login, &permissions.Spec.ForProvider.LoginRef, permissions.Spec.ForProvider.LoginSelector)
	if err != nil {
		return err
	}

	return resolveStringPtr(ctx, resolver, permissions.GetNamespace(), "spec.forProvider.projectKey",
		reference.To{List: &instancev1alpha1.ProjectList{}, Managed: &instancev1alpha1.Project{}}, reference.ExternalName(),
		&permissions.Spec.ForProvider.ProjectKey, &permissions.Spec.ForProvider.ProjectKeyRef, permissions.Spec.ForProvider.ProjectKeySelector)
}

// ResolveReferences resolves references to other managed resources for
// QualityGateUsergroupAssociation. References and selectors resolve in the
// association's own namespace.
func (association *QualityGateUsergroupAssociation) ResolveReferences(ctx context.Context, readerClient client.Reader) error { //nolint:dupl // Same shape as the Quality Profile association, but bound to distinct API types.
	resolver := reference.NewAPIResolver(readerClient, association)

	gateResponse, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: association.Spec.ForProvider.GateName,
		Extract:      reference.ExternalName(),
		Namespace:    association.GetNamespace(),
		Reference:    association.Spec.ForProvider.GateNameRef,
		Selector:     association.Spec.ForProvider.GateNameSelector,
		To: reference.To{
			List:    &instancev1alpha1.QualityGateList{},
			Managed: &instancev1alpha1.QualityGate{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.gateName")
	}

	association.Spec.ForProvider.GateName = gateResponse.ResolvedValue
	association.Spec.ForProvider.GateNameRef = gateResponse.ResolvedReference

	err = resolveGroupName(ctx, resolver, association.GetNamespace(), &association.Spec.ForProvider.GroupName, &association.Spec.ForProvider.GroupNameRef, association.Spec.ForProvider.GroupNameSelector)
	if err != nil {
		return err
	}

	return resolveUserLogin(ctx, resolver, association.GetNamespace(), &association.Spec.ForProvider.Login, &association.Spec.ForProvider.LoginRef, association.Spec.ForProvider.LoginSelector)
}

// ResolveReferences resolves references to other managed resources for
// QualityProfileUsergroupAssociation. References and selectors resolve in the
// association's own namespace.
func (association *QualityProfileUsergroupAssociation) ResolveReferences(ctx context.Context, readerClient client.Reader) error { //nolint:dupl // Same shape as the Quality Gate association, but bound to distinct API types.
	resolver := reference.NewAPIResolver(readerClient, association)

	profileResponse, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: association.Spec.ForProvider.QualityProfile,
		Extract:      instancev1alpha1.QualityProfileName(),
		Namespace:    association.GetNamespace(),
		Reference:    association.Spec.ForProvider.QualityProfileRef,
		Selector:     association.Spec.ForProvider.QualityProfileSelector,
		To: reference.To{
			List:    &instancev1alpha1.QualityProfileList{},
			Managed: &instancev1alpha1.QualityProfile{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.qualityProfile")
	}

	association.Spec.ForProvider.QualityProfile = profileResponse.ResolvedValue
	association.Spec.ForProvider.QualityProfileRef = profileResponse.ResolvedReference

	err = resolveGroupName(ctx, resolver, association.GetNamespace(), &association.Spec.ForProvider.GroupName, &association.Spec.ForProvider.GroupNameRef, association.Spec.ForProvider.GroupNameSelector)
	if err != nil {
		return err
	}

	return resolveUserLogin(ctx, resolver, association.GetNamespace(), &association.Spec.ForProvider.Login, &association.Spec.ForProvider.LoginRef, association.Spec.ForProvider.LoginSelector)
}

// ResolveReferences resolves references to other managed resources for
// UserToken. References and selectors resolve in the UserToken's own
// namespace.
func (userToken *UserToken) ResolveReferences(ctx context.Context, readerClient client.Reader) error {
	resolver := reference.NewAPIResolver(readerClient, userToken)

	return resolveUserLogin(ctx, resolver, userToken.GetNamespace(), &userToken.Spec.ForProvider.Login, &userToken.Spec.ForProvider.LoginRef, userToken.Spec.ForProvider.LoginSelector)
}

// resolveGroupName resolves a group name from a Group reference or selector in
// the given namespace, and stores the resolved name and reference.
func resolveGroupName(ctx context.Context, resolver *reference.APIResolver, namespace string, groupName **string, groupNameRef **xpv1.Reference, groupNameSelector *xpv1.Selector) error {
	return resolveStringPtr(ctx, resolver, namespace, "spec.forProvider.groupName", reference.To{List: &GroupList{}, Managed: &Group{}}, GroupName(),
		groupName, groupNameRef, groupNameSelector)
}

// resolveUserLogin resolves a user login from a User reference or selector in
// the given namespace, and stores the resolved login and reference.
func resolveUserLogin(ctx context.Context, resolver *reference.APIResolver, namespace string, login **string, loginRef **xpv1.Reference, loginSelector *xpv1.Selector) error {
	return resolveStringPtr(ctx, resolver, namespace, "spec.forProvider.login", reference.To{List: &UserList{}, Managed: &User{}}, UserLogin(),
		login, loginRef, loginSelector)
}

// resolveStringPtr resolves a reference or selector for a *string field in the
// given namespace, and stores the resolved value and reference. fieldPath
// names the field in resolution errors.
func resolveStringPtr(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, fieldPath string,
	target reference.To,
	extract reference.ExtractValueFn,
	value **string,
	ref **xpv1.Reference,
	selector *xpv1.Selector,
) error {
	response, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(*value),
		Extract:      extract,
		Namespace:    namespace,
		Reference:    *ref,
		Selector:     selector,
		To:           target,
	})
	if err != nil {
		return errors.Wrap(err, fieldPath)
	}

	*value = reference.ToPtrValue(response.ResolvedValue)
	*ref = response.ResolvedReference

	return nil
}

// UserLogin extracts the value of the login from the referenced User resource.
func UserLogin() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		cr, ok := mg.(*User)
		if !ok {
			return ""
		}

		return cr.Spec.ForProvider.Login
	}
}

// GroupName extracts the name of the group from the referenced Group resource.
// Note: the Group external name is the SonarQube-assigned ID, not the
// human-readable name, so a custom extractor is required.
func GroupName() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		cr, ok := mg.(*Group)
		if !ok {
			return ""
		}

		return cr.Spec.ForProvider.Name
	}
}
