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
	"errors"
	"strings"
	"testing"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestJiraProjectBindingResolveReferences tests reference resolution for
// JiraProjectBinding.
func TestJiraProjectBindingResolveReferences(t *testing.T) {
	t.Parallel()

	// resolving answers Gets with the external name of the requested kind,
	// failing for the kind named by failing.
	resolving := func(failing string) client.Reader {
		return &test.MockClient{
			MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
				if key.Namespace != testNamespace || key.Name == failing {
					return errors.New("boom")
				}

				switch typed := obj.(type) {
				case *instancev1alpha1.Project:
					setExtName(typed, "project-key")
				case *JiraOrganizationBinding:
					setExtName(typed, "binding-id")
				}

				return nil
			},
		}
	}

	cases := map[string]struct {
		failing   string
		wantErr   string
		wantKey   string
		wantOrgID string
	}{
		"Resolved":                {wantKey: "project-key", wantOrgID: "binding-id"},
		"ProjectNotResolved":      {failing: "my-project", wantErr: "spec.forProvider.projectKey"},
		"OrganizationNotResolved": {failing: "my-organization", wantErr: "spec.forProvider.jiraOrganizationBindingId"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			binding := &JiraProjectBinding{}
			binding.SetNamespace(testNamespace)
			binding.Spec.ForProvider.ProjectKeyRef = &xpv1.Reference{Name: "my-project"}
			binding.Spec.ForProvider.JiraOrganizationBindingIDRef = &xpv1.Reference{Name: "my-organization"}

			err := binding.ResolveReferences(context.Background(), resolving(tc.failing))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ResolveReferences() error = %v, want substring %q", err, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("ResolveReferences() unexpected error: %v", err)
			}

			if got := ptr.Deref(binding.Spec.ForProvider.ProjectKey, ""); got != tc.wantKey {
				t.Errorf("projectKey = %q, want %q", got, tc.wantKey)
			}

			if got := ptr.Deref(binding.Spec.ForProvider.JiraOrganizationBindingID, ""); got != tc.wantOrgID {
				t.Errorf("jiraOrganizationBindingId = %q, want %q", got, tc.wantOrgID)
			}
		})
	}
}
