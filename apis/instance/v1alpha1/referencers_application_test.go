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
	"slices"
	"strings"
	"testing"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// TestApplicationResolveReferences tests reference resolution for
// Application. The mock client only serves Projects from testNamespace, so a
// resolver that reads from any other namespace fails.
func TestApplicationResolveReferences(t *testing.T) {
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

			projectList, ok := list.(*ProjectList)
			if !ok {
				return fmt.Errorf("unexpected list type %T", list)
			}

			for _, name := range []string{"selected-b", "selected-a"} {
				project := Project{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}}
				setExtName(&project, name+"-key")
				projectList.Items = append(projectList.Items, project)
			}

			return nil
		},
	}

	cases := map[string]struct {
		reason       string
		params       ApplicationParameters
		errSubstr    string
		wantProjects []string
	}{
		"ProjectRefs": {
			reason:       "Project references resolve in the Application's namespace.",
			params:       ApplicationParameters{ProjectRefs: []xpv1.Reference{{Name: "first"}, {Name: "second"}}},
			wantProjects: []string{"first-key", "second-key"},
		},
		"ProjectSelector": {
			reason:       "A project selector lists Projects in the Application's namespace, sorted by key.",
			params:       ApplicationParameters{ProjectSelector: &xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}},
			wantProjects: []string{"selected-a-key", "selected-b-key"},
		},
		"MissingProjectRef": {
			reason:    "A missing project reference reports the projects field path.",
			params:    ApplicationParameters{ProjectRefs: []xpv1.Reference{{Name: "missing"}}},
			errSubstr: "spec.forProvider.projects",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			application := &Application{
				ObjectMeta: metav1.ObjectMeta{Name: "test-application", Namespace: testNamespace},
				Spec:       ApplicationSpec{ForProvider: tc.params},
			}

			err := application.ResolveReferences(context.Background(), ownNamespaceClient)
			if tc.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("%s: expected error containing %q, got %v", tc.reason, tc.errSubstr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.reason, err)
			}

			if got := application.Spec.ForProvider.Projects; !slices.Equal(got, tc.wantProjects) {
				t.Errorf("%s: Projects: want %v, got %v", tc.reason, tc.wantProjects, got)
			}
		})
	}
}
