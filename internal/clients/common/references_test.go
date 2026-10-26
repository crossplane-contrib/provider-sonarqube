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

package common

import (
	"context"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
)

const (
	// tenantNamespace is the namespace of the referencing managed resource.
	tenantNamespace = "tenant-a"
	// victimNamespace is another tenant's namespace.
	victimNamespace = "tenant-b"
)

// newReferenceTestGroup returns a Group whose external name is its
// SonarQube ID.
func newReferenceTestGroup(namespace, name, groupID string) *iamv1alpha1.Group {
	group := &iamv1alpha1.Group{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    map[string]string{"team": "platform"},
	}}
	meta.SetExternalName(group, groupID)

	return group
}

// TestSameNamespaceReferenceResolver tests that references and selectors
// resolve only inside the managed resource's namespace.
func TestSameNamespaceReferenceResolver(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		groups      iamv1alpha1.UserGroupsParameters
		wantGroupID string
		errContains string
	}{
		"SameNamespaceRefResolves": {
			groups:      iamv1alpha1.UserGroupsParameters{GroupIdRef: &xpv1.NamespacedReference{Name: "own-group"}},
			wantGroupID: "own-group-id",
		},
		"ExplicitSameNamespaceRefResolves": {
			groups:      iamv1alpha1.UserGroupsParameters{GroupIdRef: &xpv1.NamespacedReference{Name: "own-group", Namespace: tenantNamespace}},
			wantGroupID: "own-group-id",
		},
		"CrossNamespaceRefRejected": {
			groups:      iamv1alpha1.UserGroupsParameters{GroupIdRef: &xpv1.NamespacedReference{Name: "victim-group", Namespace: victimNamespace}},
			errContains: "cross-namespace reference",
		},
		"SameNamespaceSelectorResolves": {
			groups:      iamv1alpha1.UserGroupsParameters{GroupIdSelector: &xpv1.NamespacedSelector{MatchLabels: map[string]string{"team": "platform"}}},
			wantGroupID: "own-group-id",
		},
		"CrossNamespaceSelectorRejected": {
			groups:      iamv1alpha1.UserGroupsParameters{GroupIdSelector: &xpv1.NamespacedSelector{MatchLabels: map[string]string{"team": "platform"}, Namespace: victimNamespace}},
			errContains: "cross-namespace selector",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			_ = iamv1alpha1.SchemeBuilder.AddToScheme(scheme)

			groups := []iamv1alpha1.UserGroupsParameters{tc.groups}
			user := &iamv1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: tenantNamespace},
				Spec:       iamv1alpha1.UserSpec{ForProvider: iamv1alpha1.UserParameters{Login: "alice", Name: "Alice", Groups: &groups}},
			}

			kubeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(
				user,
				newReferenceTestGroup(tenantNamespace, "own-group", "own-group-id"),
				newReferenceTestGroup(victimNamespace, "victim-group", "victim-group-id"),
			).WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if key.Namespace != tenantNamespace {
						t.Errorf("unexpected Get in namespace %q", key.Namespace)
					}

					return c.Get(ctx, key, obj, opts...)
				},
				List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					listOptions := &client.ListOptions{}
					listOptions.ApplyOptions(opts)

					if listOptions.Namespace != tenantNamespace {
						t.Errorf("unexpected List in namespace %q", listOptions.Namespace)
					}

					return c.List(ctx, list, opts...)
				},
				Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
					// The fake client does not support server-side apply for
					// typed objects; persisting the resolved value is
					// crossplane-runtime's concern, not this resolver's.
					return nil
				},
			}).Build()

			err := NewSameNamespaceReferenceResolver(kubeClient).ResolveReferences(context.Background(), user)
			if tc.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("ResolveReferences() error = %v, want error containing %q", err, tc.errContains)
				}

				if got := (*user.Spec.ForProvider.Groups)[0].GroupId; got != nil {
					t.Errorf("ResolveReferences() groupId = %q, want unset", *got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ResolveReferences() unexpected error: %v", err)
			}

			if got := (*user.Spec.ForProvider.Groups)[0].GroupId; got == nil || *got != tc.wantGroupID {
				t.Errorf("ResolveReferences() groupId = %v, want %q", got, tc.wantGroupID)
			}
		})
	}
}
