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
	"strings"
	"testing"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
)

func newTestUser(namespace string) *User {
	return &User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: namespace,
		},
		Spec: UserSpec{
			ForProvider: UserParameters{
				Login: "test-login",
				Name:  "Test User",
			},
		},
	}
}

func setExtName(obj metav1.Object, name string) {
	meta.SetExternalName(obj, name)
}

const testNamespace = "default"

func TestUserResolveReferences(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason     string
		user       *User
		client     client.Reader
		wantGroups map[int]string
		errSubstr  string
	}{
		"NoGroups_NoOp": {
			reason: "A user with no groups resolves with no error.",
			user:   newTestUser(testNamespace),
			client: test.NewMockClient(),
		},
		"GroupIdAlreadySet_NoRefOrSelector_Preserved": {
			reason: "When GroupId is already set and no ref/selector exists, it is preserved.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupId: ptr.To("devs")}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client:     test.NewMockClient(),
			wantGroups: map[int]string{0: "devs"},
		},
		"GroupIdRef_ResolvesToExternalName": {
			reason: "GroupIdRef.Name is the Kubernetes object name and resolves to the external name.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupIdRef: &xpv1.NamespacedReference{Name: "example-group"}}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "devs")

					return nil
				}),
			},
			wantGroups: map[int]string{0: "devs"},
		},
		"GroupIdRef_WithExplicitNamespace": {
			reason: "An explicit namespace in GroupIdRef is respected.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupIdRef: &xpv1.NamespacedReference{Name: "example-group", Namespace: testNamespace}}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "ops")

					return nil
				}),
			},
			wantGroups: map[int]string{0: "ops"},
		},
		"GroupIdRef_WrongName_ReturnsError": {
			reason: "Using a non-existing Kubernetes group object name returns not found.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupIdRef: &xpv1.NamespacedReference{Name: "sonarqube-group-key-instead-of-k8s-name"}}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{Group: APIGroup, Resource: "groups"}, "wrong-group-name")),
			},
			errSubstr: "spec.forProvider.groups.groupId",
		},
		"GroupIdSelector_ResolvesToExternalName": {
			reason: "A selector lists matching Groups and resolves the external name.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupIdSelector: &xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					groupList, ok := obj.(*GroupList)
					if !ok {
						return nil
					}

					group := Group{ObjectMeta: metav1.ObjectMeta{Name: "platform-group", Namespace: testNamespace}}
					setExtName(&group, "platform")
					groupList.Items = []Group{group}

					return nil
				}),
			},
			wantGroups: map[int]string{0: "platform"},
		},
		"GroupId_StaleCachedK8sName_IsOverwrittenByAnnotation": {
			reason: "A stale cached GroupId (K8s name) must be overwritten when GroupIdRef is set.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{{GroupId: ptr.To("example-group"), GroupIdRef: &xpv1.NamespacedReference{Name: "example-group"}}}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "devs")

					return nil
				}),
			},
			wantGroups: map[int]string{0: "devs"},
		},
		"MultipleGroups_MixedInputs_AllResolvedIndependently": {
			reason: "Direct, ref and selector based groups are each resolved and retained independently.",
			user: func() *User {
				u := newTestUser(testNamespace)
				groups := []UserGroupsParameters{
					{GroupId: ptr.To("devs")},
					{GroupIdRef: &xpv1.NamespacedReference{Name: "group-ref-ops"}},
					{GroupIdSelector: &xpv1.Selector{MatchLabels: map[string]string{"team": "sec"}}},
				}
				u.Spec.ForProvider.Groups = &groups

				return u
			}(),
			client: &test.MockClient{
				MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
					if key.Name == "group-ref-ops" {
						setExtName(obj, "ops")

						return nil
					}

					return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
				},
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					groupList, ok := obj.(*GroupList)
					if !ok {
						return nil
					}

					group := Group{ObjectMeta: metav1.ObjectMeta{Name: "selector-group", Namespace: testNamespace}}
					setExtName(&group, "sec")
					groupList.Items = []Group{group}

					return nil
				}),
			},
			wantGroups: map[int]string{0: "devs", 1: "ops", 2: "sec"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.user.ResolveReferences(context.Background(), tc.client)

			if tc.errSubstr != "" {
				if err == nil {
					t.Errorf("[%s] (%s) expected error containing %q, got nil", name, tc.reason, tc.errSubstr)

					return
				}

				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("[%s] (%s) error %q does not contain %q", name, tc.reason, err.Error(), tc.errSubstr)
				}

				return
			}

			if err != nil {
				t.Errorf("[%s] (%s) unexpected error: %v", name, tc.reason, err)

				return
			}

			if tc.user.Spec.ForProvider.Groups == nil {
				return
			}

			groups := *tc.user.Spec.ForProvider.Groups
			for idx, wantGroup := range tc.wantGroups {
				if idx >= len(groups) {
					t.Errorf("[%s] (%s) expected group index %d but only %d groups are present", name, tc.reason, idx, len(groups))

					continue
				}

				if got := ptr.Deref(groups[idx].GroupId, ""); got != wantGroup {
					t.Errorf("[%s] (%s) Groups[%d].GroupId: want %q, got %q", name, tc.reason, idx, wantGroup, got)
				}
			}
		})
	}
}

func TestUserResolveReferences_StopsOnFirstError(t *testing.T) {
	t.Parallel()

	u := newTestUser(testNamespace)
	groups := []UserGroupsParameters{
		{GroupIdRef: &xpv1.NamespacedReference{Name: "missing-group"}},
		{GroupIdRef: &xpv1.NamespacedReference{Name: "second-group"}},
	}
	u.Spec.ForProvider.Groups = &groups

	callCount := 0
	c := &test.MockClient{
		MockGet: func(_ context.Context, _ client.ObjectKey, _ client.Object) error {
			callCount++

			return kerrors.NewNotFound(schema.GroupResource{Group: APIGroup, Resource: "groups"}, "missing-group")
		},
	}

	err := u.ResolveReferences(context.Background(), c)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	if !strings.Contains(err.Error(), "spec.forProvider.groups.groupId") {
		t.Fatalf("expected wrapped error path, got %q", err.Error())
	}

	if callCount != 1 {
		t.Errorf("expected resolver to stop on first failing group, got %d calls", callCount)
	}
}
