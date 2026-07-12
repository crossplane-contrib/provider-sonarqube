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

package permissionstemplate

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// notPermissionsTemplate is a type for testing non-PermissionsTemplate
// resources.
type notPermissionsTemplate struct {
	resource.Managed
}

// mockGate is a mock implementation of a gate for testing.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register registers a callback for a GroupVersionKind.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set sets whether a GroupVersionKind is enabled.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// newPermissionsTemplate creates a test PermissionsTemplate resource.
func newPermissionsTemplate(name string) *v1alpha1.PermissionsTemplate {
	return &v1alpha1.PermissionsTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.PermissionsTemplateSpec{
			ForProvider: v1alpha1.PermissionsTemplateParameters{Name: name},
		},
	}
}

// withExternalName sets the external name on a PermissionsTemplate.
func withExternalName(template *v1alpha1.PermissionsTemplate, externalName string) *v1alpha1.PermissionsTemplate {
	meta.SetExternalName(template, externalName)

	return template
}

// stringSlice creates a pointer to a string slice.
func stringSlice(values ...string) *[]string {
	copied := append([]string(nil), values...)

	return &copied
}

// TestSetupGatedRegistersPermissionsTemplateGVK tests SetupGated registers GVK.
func TestSetupGatedRegistersPermissionsTemplateGVK(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}
	opts := controller.DefaultOptions()
	opts.Gate = gate

	err := SetupGated(nil, opts)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if !gate.registered {
		t.Fatal("SetupGated() expected Gate.Register to be called")
	}

	if gate.callback == nil {
		t.Fatal("SetupGated() expected a non-nil callback")
	}

	if len(gate.gvks) != 1 {
		t.Fatalf("SetupGated() registered %d GVKs, want 1", len(gate.gvks))
	}

	if diff := cmp.Diff(v1alpha1.PermissionsTemplateGroupVersionKind, gate.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockPermissionsTemplatesClient
		mg      resource.Managed
		want    string
		wantErr string
	}{
		"WrongTypeReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      &notPermissionsTemplate{},
			wantErr: errNotPermissionsTemplate,
		},
		"CreateFailureReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				CreateFn: func(opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("create failed")
				},
			},
			mg:      newPermissionsTemplate("template-a"),
			wantErr: "failed to create PermissionsTemplate",
		},
		"SuccessfulCreateSetsExternalName": {
			client: &fake.MockPermissionsTemplatesClient{
				CreateFn: func(opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
					if opt.Name != "template-a" {
						t.Fatalf("CreateFn name = %q, want %q", opt.Name, "template-a")
					}

					return &sonar.PermissionsCreateTemplate{PermissionTemplate: sonar.PermissionsTemplateBasic{ID: "template-id", Name: "template-a"}}, mockHTTPResponse(), nil
				},
			},
			mg:   newPermissionsTemplate("template-a"),
			want: "template-id",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			_, err := e.Create(context.Background(), tc.mg)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Create() unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Create() error = %v, want substring %q", err, tc.wantErr)
			}

			if tc.want != "" {
				template, ok := tc.mg.(*v1alpha1.PermissionsTemplate)
				if !ok {
					t.Fatalf("Create() managed resource type = %T, want *v1alpha1.PermissionsTemplate", tc.mg)
				}

				if externalName := meta.GetExternalName(template); externalName != tc.want {
					t.Fatalf("Create() external name = %q, want %q", externalName, tc.want)
				}
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockPermissionsTemplatesClient
		mg      resource.Managed
		wantErr string
	}{
		"WrongTypeReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      &notPermissionsTemplate{},
			wantErr: errNotPermissionsTemplate,
		},
		"MissingExternalNameReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      newPermissionsTemplate("template-a"),
			wantErr: "external name is not set",
		},
		"DeleteFailureReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				DeleteFn: func(opt *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(), errors.New("delete failed")
				},
			},
			mg:      withExternalName(newPermissionsTemplate("template-a"), "template-id"),
			wantErr: "failed to delete PermissionsTemplate",
		},
		"SuccessfulDelete": {
			client: &fake.MockPermissionsTemplatesClient{
				DeleteFn: func(opt *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error) {
					if opt.TemplateID != "template-id" {
						t.Fatalf("DeleteFn TemplateID = %q, want %q", opt.TemplateID, "template-id")
					}

					return mockHTTPResponse(), nil
				},
			},
			mg: withExternalName(newPermissionsTemplate("template-a"), "template-id"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			_, err := e.Delete(context.Background(), tc.mg)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Delete() unexpected error: %v", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Delete() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
