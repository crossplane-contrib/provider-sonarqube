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

package devopspermissionmapping

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// errBoom is the error returned by failing fakes.
var errBoom = errors.New("boom")

// newMapping returns a resource managing role, with external name set to the
// role.
func newMapping(role string, permissions ...string) *v1alpha1.DevOpsPermissionMapping {
	mapping := &v1alpha1.DevOpsPermissionMapping{
		ObjectMeta: metav1.ObjectMeta{Name: "mapping", Namespace: "default"},
		Spec: v1alpha1.DevOpsPermissionMappingSpec{
			ForProvider: v1alpha1.DevOpsPermissionMappingParameters{
				Platform:    v1alpha1.DevOpsPlatformGitHub,
				Role:        role,
				Permissions: permissions,
			},
		},
	}
	meta.SetExternalName(mapping, role)

	return mapping
}

// deleted marks mapping as being deleted.
func deleted(mapping *v1alpha1.DevOpsPermissionMapping) *v1alpha1.DevOpsPermissionMapping {
	mapping.DeletionTimestamp = new(metav1.Now())

	return mapping
}

// searchReturning returns a client whose Search lists mappings.
func searchReturning(mappings ...sonar.DopTranslationPermissionMapping) *fake.MockDevOpsPermissionMappingsClient {
	return &fake.MockDevOpsPermissionMappingsClient{
		SearchFn: func(v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
			return &sonar.DopTranslationPermissionMappingsSearch{PermissionMappings: mappings}, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}
}

// TestObserve tests the observation of a mapping.
func TestObserve(t *testing.T) {
	t.Parallel()

	baseRole := sonar.DopTranslationPermissionMapping{Id: "1", Role: "maintain", BaseRole: true, Permissions: sonar.DopTranslationPermissions{Scan: true}}
	customRole := sonar.DopTranslationPermissionMapping{Id: "2", Role: "custom"}

	noExternalName := newMapping("custom")
	meta.SetExternalName(noExternalName, "")

	otherExternalName := newMapping("custom")
	meta.SetExternalName(otherExternalName, "maintain")

	cases := map[string]struct {
		client        *fake.MockDevOpsPermissionMappingsClient
		mapping       *v1alpha1.DevOpsPermissionMapping
		want          managed.ExternalObservation
		wantErr       bool
		wantAvailable bool
	}{
		"NoExternalName": {
			client:  searchReturning(customRole),
			mapping: noExternalName,
		},
		"ExternalNameMismatch": {
			client:  searchReturning(baseRole, customRole),
			mapping: otherExternalName,
			wantErr: true,
		},
		"SearchFails": {
			client: &fake.MockDevOpsPermissionMappingsClient{
				SearchFn: func(v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			mapping: newMapping("custom"),
			wantErr: true,
		},
		"Missing": {
			client:  searchReturning(baseRole),
			mapping: newMapping("custom"),
		},
		"UpToDate": {
			client:        searchReturning(baseRole),
			mapping:       newMapping("maintain", "scan"),
			want:          managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantAvailable: true,
		},
		"OutOfDate": {
			client:        searchReturning(baseRole),
			mapping:       newMapping("maintain", "admin"),
			want:          managed.ExternalObservation{ResourceExists: true},
			wantAvailable: true,
		},
		"DeletedBaseRoleIsReportedAbsent": {
			client:  searchReturning(baseRole),
			mapping: deleted(newMapping("maintain", "scan")),
		},
		"DeletedCustomRoleStillExists": {
			client:        searchReturning(customRole),
			mapping:       deleted(newMapping("custom")),
			want:          managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantAvailable: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			got, err := e.Observe(context.Background(), tc.mapping)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %t", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Observe() (-want +got):\n%s", diff)
			}

			available := tc.mapping.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
			if available != tc.wantAvailable {
				t.Errorf("Observe() available = %t, want %t", available, tc.wantAvailable)
			}
		})
	}
}

// TestCreate tests the creation of a mapping.
func TestCreate(t *testing.T) {
	t.Parallel()

	for name, createErr := range map[string]error{"Success": nil, "Failure": errBoom} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got *sonar.DopTranslationPermissionMappingCreateOptions

			e := &external{client: &fake.MockDevOpsPermissionMappingsClient{
				CreateFn: func(_ v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
					got = opt

					return nil, nil, createErr
				},
			}}
			mapping := newMapping("custom", "user")

			_, err := e.Create(context.Background(), mapping)
			if !errors.Is(err, createErr) {
				t.Fatalf("Create() error = %v, want %v", err, createErr)
			}

			want := &sonar.DopTranslationPermissionMappingCreateOptions{Role: "custom", Permissions: sonar.DopTranslationPermissions{User: true}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Create() options (-want +got):\n%s", diff)
			}

			if meta.GetExternalName(mapping) != "custom" {
				t.Errorf("Create() external name = %q, want %q", meta.GetExternalName(mapping), "custom")
			}
		})
	}
}

// TestUpdate tests the update of a mapping.
func TestUpdate(t *testing.T) {
	t.Parallel()

	for name, updateErr := range map[string]error{"Success": nil, "Failure": errBoom} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: &fake.MockDevOpsPermissionMappingsClient{
				UpdateFn: func(_ v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
					if role != "custom" || opt.Permissions.Admin == nil || *opt.Permissions.Admin {
						t.Errorf("Update(%q, %+v): want role custom with admin revoked", role, opt.Permissions)
					}

					return nil, nil, updateErr
				},
			}}

			_, err := e.Update(context.Background(), newMapping("custom", "user"))
			if !errors.Is(err, updateErr) {
				t.Errorf("Update() error = %v, want %v", err, updateErr)
			}
		})
	}
}

// TestDelete tests the deletion of a mapping.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		baseRole  bool
		status    int
		err       error
		wantCalls int
		wantErr   bool
	}{
		"CustomRole":   {status: http.StatusNoContent, wantCalls: 1},
		"AlreadyGone":  {status: http.StatusNotFound, err: errBoom, wantCalls: 1},
		"Failure":      {status: http.StatusInternalServerError, err: errBoom, wantCalls: 1, wantErr: true},
		"BaseRoleKept": {baseRole: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			e := &external{client: &fake.MockDevOpsPermissionMappingsClient{
				DeleteFn: func(v1alpha1.DevOpsPlatform, string) (*http.Response, error) {
					calls++

					return &http.Response{StatusCode: tc.status}, tc.err //nolint:nilnil // Mirrors the SDK, which returns both.
				},
			}}
			mapping := newMapping("custom")
			mapping.Status.AtProvider.BaseRole = tc.baseRole

			_, err := e.Delete(context.Background(), mapping)
			if (err != nil) != tc.wantErr {
				t.Errorf("Delete() error = %v, wantErr %t", err, tc.wantErr)
			}

			if calls != tc.wantCalls {
				t.Errorf("Delete() calls = %d, want %d", calls, tc.wantCalls)
			}

			err = e.Disconnect(context.Background())
			if err != nil {
				t.Errorf("Disconnect() error = %v", err)
			}
		})
	}
}

// TestConnect tests the construction of the external client.
func TestConnect(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, apisv1alpha1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() error = %v", err)
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("token")},
	}
	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://sonarqube/api",
			Token: &apisv1alpha1.ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: "credentials", Namespace: "default"},
						Key:             "token",
					},
				},
			},
		},
	}

	cases := map[string]struct {
		configName string
		wantErr    bool
	}{
		"Success":               {configName: "config"},
		"MissingProviderConfig": {configName: "absent", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(secret.DeepCopy(), providerConfig.DeepCopy()).Build()
			c := &connector{
				kube:  kube,
				usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn: func(common.Config) iam.DevOpsPermissionMappingsClient {
					return &fake.MockDevOpsPermissionMappingsClient{}
				},
			}
			mapping := newMapping("custom")
			mapping.UID = "uid"
			mapping.Spec.ProviderConfigReference = &xpv1.ProviderConfigReference{Kind: "ProviderConfig", Name: tc.configName}

			got, err := c.Connect(context.Background(), mapping)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Connect() error = %v, wantErr %t", err, tc.wantErr)
			}

			if (got != nil) == tc.wantErr {
				t.Errorf("Connect() client = %v, wantErr %t", got, tc.wantErr)
			}
		})
	}
}

// mockGate records the registrations of a controller.
type mockGate struct {
	gvks []schema.GroupVersionKind
}

// Register records the gated kinds.
func (m *mockGate) Register(_ func(), gvks ...schema.GroupVersionKind) {
	m.gvks = append(m.gvks, gvks...)
}

// Set is a no-op.
func (m *mockGate) Set(schema.GroupVersionKind, bool) bool {
	return true
}

// TestSetupGated tests that the controller waits for its CRD.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}

	options := helpers.Options{Options: controller.DefaultOptions()}
	options.Gate = gate

	err := SetupGated(nil, options)
	if err != nil {
		t.Fatalf("SetupGated() error = %v", err)
	}

	if diff := cmp.Diff([]schema.GroupVersionKind{v1alpha1.DevOpsPermissionMappingGroupVersionKind}, gate.gvks); diff != "" {
		t.Errorf("SetupGated() gated kinds (-want +got):\n%s", diff)
	}
}

// TestSetup tests that the controller registers with a manager. The manager is
// never started, so no API server is needed.
func TestSetup(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	mgr, err := ctrl.NewManager(&rest.Config{Host: "http://127.0.0.1:0"}, ctrl.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = Setup(mgr, helpers.Options{Options: controller.DefaultOptions()})
	if err != nil {
		t.Errorf("Setup() error = %v", err)
	}
}
