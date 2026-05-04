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

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// testNamespace is the default namespace used in tests.
const testNamespace = "default"

// testProjectK8sName is the reusable K8s object name for test Projects.
const testProjectK8sName = "example-project"

// newTestWebhook creates a minimal Webhook resource for testing.
func newTestWebhook() *Webhook {
	return &Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: testNamespace,
		},
		Spec: WebhookSpec{
			ForProvider: WebhookParameters{
				Name: "My Webhook",
				URL:  "https://example.com/hook",
			},
		},
	}
}

// setExtName sets the external-name annotation on a Kubernetes object.
func setExtName(obj metav1.Object, name string) {
	meta.SetExternalName(obj, name)
}

// TestWebhookResolveReferences tests reference resolution for Webhook.
func TestWebhookResolveReferences(t *testing.T) {
	t.Parallel()

	const projectExternalName = "my-sonarqube-project"

	notFoundProject := kerrors.NewNotFound(
		schema.GroupResource{Group: "instance.sonarqube.crossplane.io", Resource: "projects"},
		"wrong-project-name",
	)

	cases := map[string]struct {
		reason    string
		webhook   *Webhook
		client    client.Reader
		wantKey   string
		errSubstr string
	}{
		"NoRefsNoSelectors_NoOp": {
			reason:  "A webhook with no refs or selectors resolves without error.",
			webhook: newTestWebhook(),
			client:  test.NewMockClient(),
		},
		"ProjectKeyAlreadySet_NoRef_Preserved": {
			reason: "When ProjectKey is set but no ref/selector exists, it is preserved.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKey = ptr.To(projectExternalName)

				return w
			}(),
			client:  test.NewMockClient(),
			wantKey: projectExternalName,
		},
		"ProjectKeyRef_ResolvesToExternalName": {
			reason: "ProjectKeyRef.Name is the K8s object name; ExternalName is resolved.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: testProjectK8sName}

				return w
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, projectExternalName)

					return nil
				}),
			},
			wantKey: projectExternalName,
		},
		"ProjectKeyRef_WithExplicitNamespace": {
			reason: "An explicit namespace in ProjectKeyRef is respected.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{
					Name:      testProjectK8sName,
					Namespace: testNamespace,
				}

				return w
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, projectExternalName)

					return nil
				}),
			},
			wantKey: projectExternalName,
		},
		// Regression: using the SonarQube project key instead of the K8s
		// metadata.name causes a not-found error from the API server.
		"ProjectKeyRef_WrongName_ReturnsError": {
			reason: "Using a non-existing K8s name in ProjectKeyRef returns an error.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: "sonarqube-key-not-k8s-name"}

				return w
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(notFoundProject),
			},
			errSubstr: "failed to resolve Webhook.Spec.ForProvider.ProjectKey",
		},
		"ProjectKeySelector_ResolvesToExternalName": {
			reason: "A selector lists matching Projects and resolves the ExternalName.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeySelector = &xpv1.NamespacedSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}

				return w
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					projectList, ok := obj.(*instancev1alpha1.ProjectList)
					if !ok {
						return nil
					}

					p := instancev1alpha1.Project{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testProjectK8sName,
							Namespace: testNamespace,
						},
					}
					setExtName(&p, projectExternalName)
					projectList.Items = []instancev1alpha1.Project{p}

					return nil
				}),
			},
			wantKey: projectExternalName,
		},
		// Regression: Crossplane's NameAsExternalName initializer defaults
		// crossplane.io/external-name to the K8s metadata.name before the
		// Project controller runs. If the stale K8s name is passed as
		// CurrentValue, the resolver treats it as a cache hit and skips
		// re-resolution. Fix: pass CurrentValue="" when a ref is present.
		"ProjectKeyRef_StaleCachedK8sName_IsOverwrittenByAnnotation": {
			reason: "A stale ProjectKey must be overwritten when a ref is present.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				// Stale cached value set by the NameAsExternalName initializer.
				w.Spec.ForProvider.ProjectKey = new(testProjectK8sName)
				// Ref still points to the same resource.
				w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: testProjectK8sName}

				return w
			}(),
			client: &test.MockClient{
				// The Project now has its real annotation: the SonarQube key.
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, projectExternalName)

					return nil
				}),
			},
			wantKey: projectExternalName,
		},
		"ProjectKeySelector_EmptyList_ReturnsError": {
			reason: "A selector that matches no Projects returns an error.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeySelector = &xpv1.NamespacedSelector{
					MatchLabels: map[string]string{"env": "nonexistent"},
				}

				return w
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					projectList, ok := obj.(*instancev1alpha1.ProjectList)
					if !ok {
						return nil
					}

					projectList.Items = []instancev1alpha1.Project{}

					return nil
				}),
			},
			errSubstr: "failed to resolve Webhook.Spec.ForProvider.ProjectKey",
		},
		"ProjectKeySelector_ListAPIError_ReturnsError": {
			reason: "A List API failure during selector resolution propagates as an error.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeySelector = &xpv1.NamespacedSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}

				return w
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(errors.New("list API failure")),
			},
			errSubstr: "failed to resolve Webhook.Spec.ForProvider.ProjectKey",
		},
		"ProjectKeyRef_GetAPIError_ReturnsError": {
			reason: "A Get API failure during ref resolution propagates as an error.",
			webhook: func() *Webhook {
				w := newTestWebhook()
				w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: testProjectK8sName}

				return w
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(errors.New("get API failure")),
			},
			errSubstr: "failed to resolve Webhook.Spec.ForProvider.ProjectKey",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.webhook.ResolveReferences(context.Background(), tc.client)

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

			if tc.wantKey != "" {
				got := ptr.Deref(tc.webhook.Spec.ForProvider.ProjectKey, "")
				if got != tc.wantKey {
					t.Errorf("[%s] (%s) ProjectKey: want %q, got %q", name, tc.reason, tc.wantKey, got)
				}
			}
		})
	}
}

// TestWebhookResolveReferences_RefTakesPrecedenceOverSelector verifies that
// when both a ref and a selector are set, the resolver uses the ref only.
func TestWebhookResolveReferences_RefTakesPrecedenceOverSelector(t *testing.T) {
	t.Parallel()

	const (
		refExtName      = "project-from-ref"
		selectorExtName = "project-from-selector"
	)

	w := newTestWebhook()
	w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: testProjectK8sName}
	w.Spec.ForProvider.ProjectKeySelector = &xpv1.NamespacedSelector{
		MatchLabels: map[string]string{"env": "prod"},
	}

	listCalled := false
	c := &test.MockClient{
		MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
			setExtName(obj, refExtName)

			return nil
		}),
		MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
			listCalled = true

			projectList, ok := obj.(*instancev1alpha1.ProjectList)
			if !ok {
				return nil
			}

			p := instancev1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "other-project", Namespace: testNamespace},
			}
			setExtName(&p, selectorExtName)
			projectList.Items = []instancev1alpha1.Project{p}

			return nil
		}),
	}

	err := w.ResolveReferences(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ptr.Deref(w.Spec.ForProvider.ProjectKey, "")
	if got != refExtName {
		t.Errorf("ProjectKey: want %q (from ref), got %q — ref did not take precedence over selector", refExtName, got)
	}

	if listCalled {
		t.Error("List was called but should not be when Ref is set")
	}
}

// TestWebhookResolveReferences_ProjectKeyRefIsUpdated verifies that after
// resolution the ProjectKeyRef is updated with the resolved reference.
func TestWebhookResolveReferences_ProjectKeyRefIsUpdated(t *testing.T) {
	t.Parallel()

	w := newTestWebhook()
	w.Spec.ForProvider.ProjectKeyRef = &xpv1.NamespacedReference{Name: testProjectK8sName}

	c := &test.MockClient{
		MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
			setExtName(obj, "resolved-key")

			return nil
		}),
	}

	err := w.ResolveReferences(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Spec.ForProvider.ProjectKeyRef == nil {
		t.Fatal("ProjectKeyRef is nil after resolution, expected it to be set")
	}

	if w.Spec.ForProvider.ProjectKeyRef.Name != testProjectK8sName {
		t.Errorf("ProjectKeyRef.Name: want %q, got %q", testProjectK8sName, w.Spec.ForProvider.ProjectKeyRef.Name)
	}
}
