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

// newTestAICodeFixConfiguration creates a minimal AICodeFixConfiguration
// enabling AI CodeFix for some projects.
func newTestAICodeFixConfiguration() *AICodeFixConfiguration {
	return &AICodeFixConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "ai-codefix", Namespace: testNamespace},
		Spec: AICodeFixConfigurationSpec{
			ForProvider: AICodeFixConfigurationParameters{Enablement: AICodeFixEnabledForSomeProjects},
		},
	}
}

// TestAICodeFixConfigurationResolveReferences tests reference resolution for
// AICodeFixConfiguration.
func TestAICodeFixConfigurationResolveReferences(t *testing.T) {
	t.Parallel()

	const projectExternalName = "my-sonarqube-project"

	cases := map[string]struct {
		reason       string
		refs         []xpv1.Reference
		keys         []string
		client       client.Reader
		wantKeys     []string
		errSubstring string
	}{
		"NoRefs_KeysPreserved": {
			reason:   "Keys set without references are kept as-is.",
			keys:     []string{"a", "b"},
			client:   test.NewMockClient(),
			wantKeys: []string{"a", "b"},
		},
		"Refs_ResolveInOwnNamespace": {
			reason: "References resolve to the Projects' external names, in the resource's own namespace.",
			refs:   []xpv1.Reference{{Name: testProjectK8sName}},
			client: &test.MockClient{
				MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
					if key.Namespace != testNamespace {
						return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
					}

					setExtName(obj, projectExternalName)

					return nil
				},
			},
			wantKeys: []string{projectExternalName},
		},
		"Refs_GetError": {
			reason:       "A failure to get a referenced Project is reported with the field path.",
			refs:         []xpv1.Reference{{Name: testProjectK8sName}},
			client:       &test.MockClient{MockGet: test.NewMockGetFn(errors.New("boom"))},
			errSubstring: "spec.forProvider.enabledProjectKeys",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			configuration := newTestAICodeFixConfiguration()
			configuration.Spec.ForProvider.EnabledProjectKeys = tc.keys
			configuration.Spec.ForProvider.EnabledProjectKeyRefs = tc.refs

			err := configuration.ResolveReferences(context.Background(), tc.client)

			if tc.errSubstring != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSubstring) {
					t.Fatalf("%s\nResolveReferences() error = %v, want substring %q", tc.reason, err, tc.errSubstring)
				}

				return
			}

			if err != nil {
				t.Fatalf("%s\nResolveReferences() unexpected error: %v", tc.reason, err)
			}

			if !slices.Equal(configuration.Spec.ForProvider.EnabledProjectKeys, tc.wantKeys) {
				t.Errorf("%s\nEnabledProjectKeys = %v, want %v", tc.reason, configuration.Spec.ForProvider.EnabledProjectKeys, tc.wantKeys)
			}
		})
	}
}
