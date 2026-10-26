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
	"fmt"
	"strings"
	"testing"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// Values returned by ownNamespaceClient for each referenced kind.
const (
	resolvedGroupName      = "resolved-group"
	resolvedLogin          = "resolved-login"
	resolvedProjectKey     = "resolved-project-key"
	resolvedGateName       = "resolved-gate"
	resolvedProfileName    = "resolved-profile"
	selectedObjectName     = "selected-object"
	missingReferencedName  = "missing"
	otherTenantNamespace   = "other-tenant"
	otherTenantGroupName   = "other-tenant-group"
	resolvesInOwnNamespace = "resolves in the referencing resource's namespace"
)

// ownNamespaceClient returns a mock client that only serves objects from
// testNamespace. Get returns NotFound for any other namespace or for the name
// missingReferencedName, and List fails unless it is scoped to testNamespace,
// so a resolver that reads outside its own namespace fails the test.
func ownNamespaceClient() *test.MockClient {
	return &test.MockClient{
		MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
			if key.Namespace != testNamespace || key.Name == missingReferencedName {
				return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
			}

			fillReferenced(obj)

			return nil
		},
		MockList: func(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
			listOpts := &client.ListOptions{}
			listOpts.ApplyOptions(opts)

			if listOpts.Namespace != testNamespace {
				return fmt.Errorf("list scoped to namespace %q, want %q", listOpts.Namespace, testNamespace)
			}

			return fillReferencedList(list)
		},
	}
}

// fillReferenced sets the value each extractor reads on a referenced object.
func fillReferenced(obj client.Object) {
	switch referenced := obj.(type) {
	case *Group:
		referenced.Spec.ForProvider.Name = resolvedGroupName
	case *User:
		referenced.Spec.ForProvider.Login = resolvedLogin
	case *instancev1alpha1.Project:
		setExtName(referenced, resolvedProjectKey)
	case *instancev1alpha1.QualityGate:
		setExtName(referenced, resolvedGateName)
	case *instancev1alpha1.QualityProfile:
		referenced.Spec.ForProvider.Name = resolvedProfileName
	}
}

// fillReferencedList returns one selected object of the listed kind.
func fillReferencedList(list client.ObjectList) error {
	objectMeta := metav1.ObjectMeta{Name: selectedObjectName, Namespace: testNamespace}

	switch referencedList := list.(type) {
	case *GroupList:
		group := Group{ObjectMeta: objectMeta}
		fillReferenced(&group)
		referencedList.Items = []Group{group}
	case *UserList:
		user := User{ObjectMeta: objectMeta}
		fillReferenced(&user)
		referencedList.Items = []User{user}
	case *instancev1alpha1.ProjectList:
		project := instancev1alpha1.Project{ObjectMeta: objectMeta}
		fillReferenced(&project)
		referencedList.Items = []instancev1alpha1.Project{project}
	case *instancev1alpha1.QualityGateList:
		gate := instancev1alpha1.QualityGate{ObjectMeta: objectMeta}
		fillReferenced(&gate)
		referencedList.Items = []instancev1alpha1.QualityGate{gate}
	case *instancev1alpha1.QualityProfileList:
		profile := instancev1alpha1.QualityProfile{ObjectMeta: objectMeta}
		fillReferenced(&profile)
		referencedList.Items = []instancev1alpha1.QualityProfile{profile}
	default:
		return fmt.Errorf("unexpected list type %T", list)
	}

	return nil
}

// selector returns a label selector for the resolver tests.
func selector() *xpv1.Selector {
	return &xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}
}

// testObjectMeta returns the metadata of a referencing resource.
func testObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: "referencing", Namespace: testNamespace}
}

// checkResolveErr reports whether err matches errSubstr, and returns true
// when the caller should stop checking resolved values.
func checkResolveErr(t *testing.T, err error, errSubstr string) bool {
	t.Helper()

	if errSubstr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return false
	}

	if err == nil || !strings.Contains(err.Error(), errSubstr) {
		t.Fatalf("expected error containing %q, got %v", errSubstr, err)
	}

	return true
}

// TestPermissionsResolveReferences tests reference resolution for Permissions.
func TestPermissionsResolveReferences(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason        string
		params        PermissionsParameters
		errSubstr     string
		wantGroupName string
		wantLogin     string
		wantProject   string
		wantRefName   string
	}{
		"GroupNameRefAndProjectKeySelector": {
			reason:        "A group reference and a project selector " + resolvesInOwnNamespace + ".",
			params:        PermissionsParameters{GroupNameRef: &xpv1.Reference{Name: "group"}, ProjectKeySelector: selector()},
			wantGroupName: resolvedGroupName,
			wantProject:   resolvedProjectKey,
			wantRefName:   selectedObjectName,
		},
		"LoginSelectorAndProjectKeyRef": {
			reason:      "A login selector and a project reference " + resolvesInOwnNamespace + ".",
			params:      PermissionsParameters{LoginSelector: selector(), ProjectKeyRef: &xpv1.Reference{Name: "project"}},
			wantLogin:   resolvedLogin,
			wantProject: resolvedProjectKey,
			wantRefName: "project",
		},
		"MissingLoginRef": {
			reason:    "A missing user reference reports the login field path.",
			params:    PermissionsParameters{LoginRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.login",
		},
		"MissingProjectKeyRef": {
			reason:    "A missing project reference reports the projectKey field path.",
			params:    PermissionsParameters{ProjectKeyRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.projectKey",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			permissions := &Permissions{ObjectMeta: testObjectMeta(), Spec: PermissionsSpec{ForProvider: tc.params}}

			err := permissions.ResolveReferences(context.Background(), ownNamespaceClient())
			if checkResolveErr(t, err, tc.errSubstr) {
				return
			}

			forProvider := permissions.Spec.ForProvider
			if got := ptr.Deref(forProvider.GroupName, ""); got != tc.wantGroupName {
				t.Errorf("%s: GroupName: want %q, got %q", tc.reason, tc.wantGroupName, got)
			}

			if got := ptr.Deref(forProvider.Login, ""); got != tc.wantLogin {
				t.Errorf("%s: Login: want %q, got %q", tc.reason, tc.wantLogin, got)
			}

			if got := ptr.Deref(forProvider.ProjectKey, ""); got != tc.wantProject {
				t.Errorf("%s: ProjectKey: want %q, got %q", tc.reason, tc.wantProject, got)
			}

			if forProvider.ProjectKeyRef == nil || forProvider.ProjectKeyRef.Name != tc.wantRefName {
				t.Errorf("%s: ProjectKeyRef: want name %q, got %+v", tc.reason, tc.wantRefName, forProvider.ProjectKeyRef)
			}
		})
	}
}

// TestQualityGateUsergroupAssociationResolveReferences tests reference
// resolution for QualityGateUsergroupAssociation.
func TestQualityGateUsergroupAssociationResolveReferences(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason        string
		params        QualityGateUsergroupAssociationParameters
		errSubstr     string
		wantGate      string
		wantGroupName string
		wantLogin     string
	}{
		"GateNameRefAndGroupNameSelector": {
			reason:        "A gate reference and a group selector " + resolvesInOwnNamespace + ".",
			params:        QualityGateUsergroupAssociationParameters{GateNameRef: &xpv1.Reference{Name: "gate"}, GroupNameSelector: selector()},
			wantGate:      resolvedGateName,
			wantGroupName: resolvedGroupName,
		},
		"GateNameSelectorAndLoginRef": {
			reason:    "A gate selector and a login reference " + resolvesInOwnNamespace + ".",
			params:    QualityGateUsergroupAssociationParameters{GateNameSelector: selector(), LoginRef: &xpv1.Reference{Name: "user"}},
			wantGate:  resolvedGateName,
			wantLogin: resolvedLogin,
		},
		"MissingGateNameRef": {
			reason:    "A missing gate reference reports the gateName field path.",
			params:    QualityGateUsergroupAssociationParameters{GateNameRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.gateName",
		},
		"MissingGroupNameRef": {
			reason:    "A missing group reference reports the groupName field path.",
			params:    QualityGateUsergroupAssociationParameters{GateName: "gate", GroupNameRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.groupName",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			association := &QualityGateUsergroupAssociation{
				ObjectMeta: testObjectMeta(),
				Spec:       QualityGateUsergroupAssociationSpec{ForProvider: tc.params},
			}

			err := association.ResolveReferences(context.Background(), ownNamespaceClient())
			if checkResolveErr(t, err, tc.errSubstr) {
				return
			}

			forProvider := association.Spec.ForProvider
			if forProvider.GateName != tc.wantGate {
				t.Errorf("%s: GateName: want %q, got %q", tc.reason, tc.wantGate, forProvider.GateName)
			}

			if got := ptr.Deref(forProvider.GroupName, ""); got != tc.wantGroupName {
				t.Errorf("%s: GroupName: want %q, got %q", tc.reason, tc.wantGroupName, got)
			}

			if got := ptr.Deref(forProvider.Login, ""); got != tc.wantLogin {
				t.Errorf("%s: Login: want %q, got %q", tc.reason, tc.wantLogin, got)
			}
		})
	}
}

// TestQualityProfileUsergroupAssociationResolveReferences tests reference
// resolution for QualityProfileUsergroupAssociation.
func TestQualityProfileUsergroupAssociationResolveReferences(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason        string
		params        QualityProfileUsergroupAssociationParameters
		errSubstr     string
		wantProfile   string
		wantGroupName string
		wantLogin     string
	}{
		"QualityProfileRefAndLoginSelector": {
			reason:      "A profile reference and a login selector " + resolvesInOwnNamespace + ".",
			params:      QualityProfileUsergroupAssociationParameters{QualityProfileRef: &xpv1.Reference{Name: "profile"}, LoginSelector: selector()},
			wantProfile: resolvedProfileName,
			wantLogin:   resolvedLogin,
		},
		"QualityProfileSelectorAndGroupNameRef": {
			reason:        "A profile selector and a group reference " + resolvesInOwnNamespace + ".",
			params:        QualityProfileUsergroupAssociationParameters{QualityProfileSelector: selector(), GroupNameRef: &xpv1.Reference{Name: "group"}},
			wantProfile:   resolvedProfileName,
			wantGroupName: resolvedGroupName,
		},
		"MissingQualityProfileRef": {
			reason:    "A missing profile reference reports the qualityProfile field path.",
			params:    QualityProfileUsergroupAssociationParameters{QualityProfileRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.qualityProfile",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			association := &QualityProfileUsergroupAssociation{
				ObjectMeta: testObjectMeta(),
				Spec:       QualityProfileUsergroupAssociationSpec{ForProvider: tc.params},
			}

			err := association.ResolveReferences(context.Background(), ownNamespaceClient())
			if checkResolveErr(t, err, tc.errSubstr) {
				return
			}

			forProvider := association.Spec.ForProvider
			if forProvider.QualityProfile != tc.wantProfile {
				t.Errorf("%s: QualityProfile: want %q, got %q", tc.reason, tc.wantProfile, forProvider.QualityProfile)
			}

			if got := ptr.Deref(forProvider.GroupName, ""); got != tc.wantGroupName {
				t.Errorf("%s: GroupName: want %q, got %q", tc.reason, tc.wantGroupName, got)
			}

			if got := ptr.Deref(forProvider.Login, ""); got != tc.wantLogin {
				t.Errorf("%s: Login: want %q, got %q", tc.reason, tc.wantLogin, got)
			}
		})
	}
}

// TestUserTokenResolveReferences tests reference resolution for UserToken.
func TestUserTokenResolveReferences(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason    string
		params    UserTokenParameters
		errSubstr string
		wantLogin string
	}{
		"LoginRef": {
			reason:    "A login reference " + resolvesInOwnNamespace + ".",
			params:    UserTokenParameters{LoginRef: &xpv1.Reference{Name: "user"}},
			wantLogin: resolvedLogin,
		},
		"LoginSelector": {
			reason:    "A login selector " + resolvesInOwnNamespace + ".",
			params:    UserTokenParameters{LoginSelector: selector()},
			wantLogin: resolvedLogin,
		},
		"MissingLoginRef": {
			reason:    "A missing user reference reports the login field path.",
			params:    UserTokenParameters{LoginRef: &xpv1.Reference{Name: missingReferencedName}},
			errSubstr: "spec.forProvider.login",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			userToken := &UserToken{ObjectMeta: testObjectMeta(), Spec: UserTokenSpec{ForProvider: tc.params}}

			err := userToken.ResolveReferences(context.Background(), ownNamespaceClient())
			if checkResolveErr(t, err, tc.errSubstr) {
				return
			}

			if got := ptr.Deref(userToken.Spec.ForProvider.Login, ""); got != tc.wantLogin {
				t.Errorf("%s: Login: want %q, got %q", tc.reason, tc.wantLogin, got)
			}
		})
	}
}

// TestUserResolveReferences_SelectorIgnoresOtherNamespaces verifies that a
// User's group selector only lists Groups in the User's own namespace.
func TestUserResolveReferences_SelectorIgnoresOtherNamespaces(t *testing.T) {
	t.Parallel()

	user := newTestUser(testNamespace)
	groups := []UserGroupsParameters{{GroupIdSelector: selector()}}
	user.Spec.ForProvider.Groups = &groups

	c := &test.MockClient{
		MockList: func(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
			listOpts := &client.ListOptions{}
			listOpts.ApplyOptions(opts)

			groupList, ok := list.(*GroupList)
			if !ok {
				return fmt.Errorf("unexpected list type %T", list)
			}

			// Behave like the API server: an unscoped list returns every
			// namespace, a scoped list only the matching one.
			if listOpts.Namespace == "" || listOpts.Namespace == otherTenantNamespace {
				group := Group{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Namespace: otherTenantNamespace}}
				setExtName(&group, otherTenantGroupName)
				groupList.Items = []Group{group}
			}

			return nil
		},
	}

	err := user.ResolveReferences(context.Background(), c)
	if err == nil {
		t.Fatalf("expected no match in namespace %q, got GroupId %q", testNamespace, ptr.Deref(groups[0].GroupId, ""))
	}

	if got := ptr.Deref((*user.Spec.ForProvider.Groups)[0].GroupId, ""); got == otherTenantGroupName {
		t.Fatalf("resolved a Group from namespace %q", otherTenantNamespace)
	}
}
