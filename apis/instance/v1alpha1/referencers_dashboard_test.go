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

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// TestDashboardResolveReferences tests reference resolution for Dashboard.
// The mock client only serves objects from testNamespace, so a resolver that
// reads from any other namespace fails.
func TestDashboardResolveReferences(t *testing.T) {
	t.Parallel()

	ownNamespaceClient := &test.MockClient{
		MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
			if key.Namespace != testNamespace || key.Name == "missing" {
				return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
			}

			setExtName(obj, key.Name+"-key")

			return nil
		},
	}

	cases := map[string]struct {
		reason        string
		params        DashboardParameters
		errSubstr     string
		wantProject   string
		wantPortfolio string
	}{
		"ProjectRef": {
			reason:      "A project reference resolves to the Project external name.",
			params:      DashboardParameters{ProjectKeyRef: &xpv1.Reference{Name: "project"}},
			wantProject: "project-key",
		},
		"PortfolioRef": {
			reason:        "A portfolio reference resolves to the Portfolio external name.",
			params:        DashboardParameters{PortfolioKeyRef: &xpv1.Reference{Name: "portfolio"}},
			wantPortfolio: "portfolio-key",
		},
		"MissingProjectRef": {
			reason:    "A missing project reference reports the projectKey field path.",
			params:    DashboardParameters{ProjectKeyRef: &xpv1.Reference{Name: "missing"}},
			errSubstr: "spec.forProvider.projectKey",
		},
		"MissingPortfolioRef": {
			reason:    "A missing portfolio reference reports the portfolioKey field path.",
			params:    DashboardParameters{PortfolioKeyRef: &xpv1.Reference{Name: "missing"}},
			errSubstr: "spec.forProvider.portfolioKey",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dashboard := &Dashboard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dashboard", Namespace: testNamespace},
				Spec:       DashboardSpec{ForProvider: tc.params},
			}

			err := dashboard.ResolveReferences(context.Background(), ownNamespaceClient)
			if tc.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("%s: expected error containing %q, got %v", tc.reason, tc.errSubstr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.reason, err)
			}

			if got := ptr.Deref(dashboard.Spec.ForProvider.ProjectKey, ""); got != tc.wantProject {
				t.Errorf("%s: ProjectKey: want %q, got %q", tc.reason, tc.wantProject, got)
			}

			if got := ptr.Deref(dashboard.Spec.ForProvider.PortfolioKey, ""); got != tc.wantPortfolio {
				t.Errorf("%s: PortfolioKey: want %q, got %q", tc.reason, tc.wantPortfolio, got)
			}
		})
	}
}
