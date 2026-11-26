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

	"github.com/google/go-cmp/cmp"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// TestPortfolioResolveReferences tests reference resolution for Portfolio.
// The mock client only serves resources from testNamespace, so a resolver
// that reads from any other namespace fails.
func TestPortfolioResolveReferences(t *testing.T) {
	t.Parallel()

	ownNamespaceClient := &test.MockClient{
		MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
			if key.Namespace != testNamespace || key.Name == "missing" {
				return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
			}

			setExtName(obj, key.Name+"-key")

			return nil
		},
		MockList: func(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
			listOpts := &client.ListOptions{}
			listOpts.ApplyOptions(opts)

			if listOpts.Namespace != testNamespace {
				return fmt.Errorf("list scoped to namespace %q, want %q", listOpts.Namespace, testNamespace)
			}

			portfolioList, ok := list.(*PortfolioList)
			if !ok {
				return fmt.Errorf("unexpected list type %T", list)
			}

			for _, name := range []string{"selected-b", "selected-a"} {
				portfolio := Portfolio{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}}
				setExtName(&portfolio, name+"-key")
				portfolioList.Items = append(portfolioList.Items, portfolio)
			}

			return nil
		},
	}

	cases := map[string]struct {
		reason    string
		params    PortfolioParameters
		errSubstr string
		want      PortfolioParameters
	}{
		"NoReferences": {
			reason: "Unset members stay unmanaged.",
			params: PortfolioParameters{},
			want:   PortfolioParameters{},
		},
		"References": {
			reason: "Parent, member and referenced portfolio references resolve in the Portfolio's namespace.",
			params: PortfolioParameters{
				ParentRef:        &xpv1.Reference{Name: "parent"},
				Projects:         &[]PortfolioProjectParameters{{KeyRef: &xpv1.Reference{Name: "project"}, Branches: []string{"dev"}}},
				Applications:     &[]PortfolioApplicationParameters{{KeyRef: &xpv1.Reference{Name: "app"}}},
				SubPortfolioRefs: []xpv1.Reference{{Name: "first"}, {Name: "second"}},
			},
			want: PortfolioParameters{
				Parent:           new("parent-key"),
				ParentRef:        &xpv1.Reference{Name: "parent"},
				Projects:         &[]PortfolioProjectParameters{{Key: new("project-key"), KeyRef: &xpv1.Reference{Name: "project"}, Branches: []string{"dev"}}},
				Applications:     &[]PortfolioApplicationParameters{{Key: new("app-key"), KeyRef: &xpv1.Reference{Name: "app"}}},
				SubPortfolios:    &[]string{"first-key", "second-key"},
				SubPortfolioRefs: []xpv1.Reference{{Name: "first"}, {Name: "second"}},
			},
		},
		"SubPortfolioSelector": {
			reason: "A sub-portfolio selector lists Portfolios in the Portfolio's namespace, sorted by key.",
			params: PortfolioParameters{SubPortfolioSelector: &xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}},
			want: PortfolioParameters{
				SubPortfolios:        &[]string{"selected-a-key", "selected-b-key"},
				SubPortfolioRefs:     []xpv1.Reference{{Name: "selected-a"}, {Name: "selected-b"}},
				SubPortfolioSelector: &xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}},
			},
		},
		"MissingParentRef": {
			reason:    "A missing parent reference reports the parent field path.",
			params:    PortfolioParameters{ParentRef: &xpv1.Reference{Name: "missing"}},
			errSubstr: "spec.forProvider.parent",
		},
		"MissingProjectRef": {
			reason:    "A missing project reference reports the project field path.",
			params:    PortfolioParameters{Projects: &[]PortfolioProjectParameters{{KeyRef: &xpv1.Reference{Name: "missing"}}}},
			errSubstr: "spec.forProvider.projects[0].key",
		},
		"MissingApplicationRef": {
			reason:    "A missing application reference reports the application field path.",
			params:    PortfolioParameters{Applications: &[]PortfolioApplicationParameters{{KeyRef: &xpv1.Reference{Name: "missing"}}}},
			errSubstr: "spec.forProvider.applications[0].key",
		},
		"MissingSubPortfolioRef": {
			reason:    "A missing sub-portfolio reference reports the subPortfolios field path.",
			params:    PortfolioParameters{SubPortfolioRefs: []xpv1.Reference{{Name: "missing"}}},
			errSubstr: "spec.forProvider.subPortfolios",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			portfolio := &Portfolio{
				ObjectMeta: metav1.ObjectMeta{Name: "test-portfolio", Namespace: testNamespace},
				Spec:       PortfolioSpec{ForProvider: tc.params},
			}

			err := portfolio.ResolveReferences(context.Background(), ownNamespaceClient)
			if tc.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("%s: expected error containing %q, got %v", tc.reason, tc.errSubstr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want, portfolio.Spec.ForProvider); diff != "" {
				t.Errorf("%s: -want +got:\n%s", tc.reason, diff)
			}
		})
	}
}
