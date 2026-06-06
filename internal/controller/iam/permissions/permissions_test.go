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

package permissions

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// notPermissions is a test type that is not a Permissions resource.
type notPermissions struct {
	resource.Managed
}

// mockGate is a mock implementation of the feature gate interface.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register records the callback and GVKs.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set is a no-op for the mock gate.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// mockHTTPResponse creates a mock HTTP response with the given status code.
func mockHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       http.NoBody,
	}
}

// checkError asserts the error matches the expected substring.
func checkError(t *testing.T, method, wantErrSubstr string, gotErr error) {
	t.Helper()

	if wantErrSubstr == "" && gotErr == nil {
		return
	}

	if wantErrSubstr == "" && gotErr != nil {
		t.Errorf("%s() unexpected error: %v", method, gotErr)

		return
	}

	if wantErrSubstr != "" && gotErr == nil {
		t.Errorf("%s() expected error containing %q, got nil", method, wantErrSubstr)

		return
	}

	if !strings.Contains(gotErr.Error(), wantErrSubstr) {
		t.Errorf("%s() error = %q, want containing %q", method, gotErr.Error(), wantErrSubstr)
	}
}

// newTestGroupPermissions creates a test Permissions resource for a group.
func newTestGroupPermissions(externalName, groupName string, permissions []string) *v1alpha1.Permissions {
	p := &v1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-permissions",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.PermissionsSpec{
			ForProvider: v1alpha1.PermissionsParameters{
				GroupName:   new(groupName),
				Permissions: permissions,
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(p, externalName)
	}

	return p
}

// newTestUserPermissions creates a test Permissions resource for a user.
func newTestUserPermissions(externalName, login string, permissions []string) *v1alpha1.Permissions {
	p := &v1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-permissions",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.PermissionsSpec{
			ForProvider: v1alpha1.PermissionsParameters{
				Login:       new(login),
				Permissions: permissions,
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(p, externalName)
	}

	return p
}

// newTestGroupPermissionsWithProject creates a test Permissions resource for
// a group scoped to a specific project.
func newTestGroupPermissionsWithProject(externalName, groupName, projectKey string, permissions []string) *v1alpha1.Permissions {
	p := newTestGroupPermissions(externalName, groupName, permissions)
	p.Spec.ForProvider.ProjectKey = new(projectKey)

	return p
}

// sorted returns a sorted copy of the input slice for deterministic
// comparison.
func sorted(in []string) []string {
	if in == nil {
		return nil
	}

	out := append([]string(nil), in...)
	sort.Strings(out)

	return out
}

// TestCreate tests creating a Permissions resource.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation  managed.ExternalCreation
		errSubstr string
	}

	cases := map[string]struct {
		client *fake.MockPermissionsClient
		args   args
		want   want
	}{
		"NotPermissionsType": {
			client: &fake.MockPermissionsClient{},
			args:   args{ctx: context.Background(), mg: &notPermissions{}},
			want:   want{errSubstr: errNotPermissions},
		},
		"GroupAddFails": {
			client: &fake.MockPermissionsClient{
				AddGroupFn: func(_ *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("add error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("", "devs", []string{"scan"}),
			},
			want: want{errSubstr: errCreatePermissions},
		},
		"UserAddFails": {
			client: &fake.MockPermissionsClient{
				AddUserFn: func(_ *sonar.PermissionsAddUserOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("add user error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserPermissions("", "alice", []string{"scan"}),
			},
			want: want{errSubstr: errCreatePermissions},
		},
		"GroupSuccessNoPermissions": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("", "devs", []string{}),
			},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			checkError(t, "Create", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("GroupSuccessExternalNameSet", func(t *testing.T) {
		t.Parallel()

		p := newTestGroupPermissions("", "devs", []string{"scan"})
		e := &external{client: &fake.MockPermissionsClient{
			AddGroupFn: func(_ *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if gotName := meta.GetExternalName(p); gotName != "group:devs" {
			t.Fatalf("Create() external name = %q, want %q", gotName, "group:devs")
		}
	})

	t.Run("UserSuccessExternalNameSet", func(t *testing.T) {
		t.Parallel()

		p := newTestUserPermissions("", "alice", []string{"scan"})
		e := &external{client: &fake.MockPermissionsClient{
			AddUserFn: func(_ *sonar.PermissionsAddUserOptions) (*http.Response, error) {
				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if gotName := meta.GetExternalName(p); gotName != "user:alice" {
			t.Fatalf("Create() external name = %q, want %q", gotName, "user:alice")
		}
	})

	t.Run("GroupWithProjectKeyExternalName", func(t *testing.T) {
		t.Parallel()

		p := newTestGroupPermissionsWithProject("", "devs", "my-project", []string{"user"})
		e := &external{client: &fake.MockPermissionsClient{
			AddGroupFn: func(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
				if opt.ProjectKey != "my-project" {
					t.Fatalf("Create() AddGroup unexpected projectKey %q", opt.ProjectKey)
				}

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if gotName := meta.GetExternalName(p); gotName != "group:devs:my-project" {
			t.Fatalf("Create() external name = %q, want %q", gotName, "group:devs:my-project")
		}
	})

	t.Run("MultiplePermissionsAllAdded", func(t *testing.T) {
		t.Parallel()

		added := make([]string, 0)
		p := newTestGroupPermissions("", "devs", []string{"scan", "admin", "provisioning"})
		e := &external{client: &fake.MockPermissionsClient{
			AddGroupFn: func(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
				added = append(added, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if diff := cmp.Diff(sorted([]string{"scan", "admin", "provisioning"}), sorted(added)); diff != "" {
			t.Fatalf("Create() permissions added mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestDelete tests deleting a Permissions resource.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		deletion  managed.ExternalDelete
		errSubstr string
	}

	cases := map[string]struct {
		client *fake.MockPermissionsClient
		args   args
		want   want
	}{
		"NotPermissionsType": {
			client: &fake.MockPermissionsClient{},
			args:   args{ctx: context.Background(), mg: &notPermissions{}},
			want:   want{errSubstr: errNotPermissions},
		},
		"EmptyExternalNameSuccess": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("", "devs", []string{"scan"}),
			},
			want: want{deletion: managed.ExternalDelete{}},
		},
		"EmptyObservedPermissionsSuccess": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("group:devs", "devs", []string{"scan"}),
			},
			want: want{deletion: managed.ExternalDelete{}},
		},
		"GroupRemoveFails": {
			client: &fake.MockPermissionsClient{
				RemoveGroupFn: func(_ *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("remove error")
				},
			},
			args: func() args {
				p := newTestGroupPermissions("group:devs", "devs", []string{"scan"})
				p.Status.AtProvider.Permissions = []string{"scan"}

				return args{ctx: context.Background(), mg: p}
			}(),
			want: want{errSubstr: errDeletePermissions},
		},
		"UserRemoveFails": {
			client: &fake.MockPermissionsClient{
				RemoveUserFn: func(_ *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("remove user error")
				},
			},
			args: func() args {
				p := newTestUserPermissions("user:alice", "alice", []string{"scan"})
				p.Status.AtProvider.Permissions = []string{"scan"}

				return args{ctx: context.Background(), mg: p}
			}(),
			want: want{errSubstr: errDeletePermissions},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			checkError(t, "Delete", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.deletion, got); diff != "" {
				t.Errorf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("GroupDeleteRemovesAllObservedPermissions", func(t *testing.T) {
		t.Parallel()

		removed := make([]string, 0)
		p := newTestGroupPermissions("group:devs", "devs", []string{"scan"})
		p.Status.AtProvider.Permissions = []string{"scan", "admin", "provisioning"}

		e := &external{client: &fake.MockPermissionsClient{
			RemoveGroupFn: func(opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
				removed = append(removed, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Delete(context.Background(), p)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if diff := cmp.Diff(sorted([]string{"scan", "admin", "provisioning"}), sorted(removed)); diff != "" {
			t.Fatalf("Delete() permissions removed mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UserDeleteRemovesAllObservedPermissions", func(t *testing.T) {
		t.Parallel()

		removed := make([]string, 0)
		p := newTestUserPermissions("user:alice", "alice", []string{"scan"})
		p.Status.AtProvider.Permissions = []string{"scan", "admin"}

		e := &external{client: &fake.MockPermissionsClient{
			RemoveUserFn: func(opt *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
				removed = append(removed, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Delete(context.Background(), p)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if diff := cmp.Diff(sorted([]string{"scan", "admin"}), sorted(removed)); diff != "" {
			t.Fatalf("Delete() permissions removed mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestDisconnect tests disconnecting a Permissions resource.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockPermissionsClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnectTypeAssertion tests Connect with invalid resource type.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notPermissions{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Permissions type, got nil")
	}

	if !strings.Contains(err.Error(), errNotPermissions) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errNotPermissions)
	}
}

// TestConnectTrackUsageError tests Connect when tracking usage fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).Build()

	p := &v1alpha1.Permissions{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.PermissionsGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.PermissionsKind},
		ObjectMeta: metav1.ObjectMeta{Name: "test-perms", Namespace: "default", UID: types.UID("perms-uid")},
		Spec: v1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider:         v1alpha1.PermissionsParameters{GroupName: new("devs"), Permissions: []string{"scan"}},
		},
	}

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: iam.NewPermissionsClient,
	}

	_, err = c.Connect(context.Background(), p)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errTrackPCUsage) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errTrackPCUsage)
	}
}

// TestConnectGetConfigError tests Connect when getting config fails.
func TestConnectGetConfigError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).Build()

	p := &v1alpha1.Permissions{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.PermissionsGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.PermissionsKind},
		ObjectMeta: metav1.ObjectMeta{Name: "test-perms", Namespace: "default", UID: types.UID("perms-uid")},
		Spec: v1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider:         v1alpha1.PermissionsParameters{GroupName: new("devs"), Permissions: []string{"scan"}},
		},
	}
	p.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: iam.NewPermissionsClient,
	}

	_, err = c.Connect(context.Background(), p)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errGetPC) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errGetPC)
	}
}

// TestConnectSuccess tests successful Connect.
func TestConnectSuccess(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000",
			Token: &apisv1alpha1.ProviderCredentials{
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: "sonar-secret", Namespace: "default"},
						Key:             "token",
					},
				},
				Source: xpv1.CredentialsSourceSecret,
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sonar-secret", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("my-token")},
	}

	kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, secret).Build()

	p := &v1alpha1.Permissions{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.PermissionsGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.PermissionsKind},
		ObjectMeta: metav1.ObjectMeta{Name: "test-perms", Namespace: "default", UID: types.UID("perms-uid")},
		Spec: v1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider:         v1alpha1.PermissionsParameters{GroupName: new("devs"), Permissions: []string{"scan"}},
		},
	}
	p.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: iam.NewPermissionsClient,
	}

	got, err := c.Connect(context.Background(), p)
	if err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("Connect() expected non-nil external client")
	}

	if _, ok := got.(*external); !ok {
		t.Fatalf("Connect() returned %T, want *external", got)
	}
}

// TestSetupGatedRegistersPermissionsGVK tests SetupGated registers the
// correct GVK.
func TestSetupGatedRegistersPermissionsGVK(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := controller.DefaultOptions()
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if !g.registered {
		t.Fatal("SetupGated() expected Gate.Register to be called")
	}

	if g.callback == nil {
		t.Fatal("SetupGated() expected a non-nil callback")
	}

	if len(g.gvks) != 1 {
		t.Fatalf("SetupGated() registered %d GVKs, want 1", len(g.gvks))
	}

	if diff := cmp.Diff(v1alpha1.PermissionsGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestParseExternalName tests parsing external names.
func TestParseExternalName(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input          string
		wantSubjType   string
		wantSubject    string
		wantProjectKey string
		wantErrSubstr  string
	}{
		"ValidGroupNoProject": {
			input:        "group:devs",
			wantSubjType: subjectTypeGroup,
			wantSubject:  "devs",
		},
		"ValidUserNoProject": {
			input:        "user:alice",
			wantSubjType: subjectTypeUser,
			wantSubject:  "alice",
		},
		"ValidGroupWithProject": {
			input:          "group:devs:my-project",
			wantSubjType:   subjectTypeGroup,
			wantSubject:    "devs",
			wantProjectKey: "my-project",
		},
		"ValidUserWithProject": {
			input:          "user:alice:my-project",
			wantSubjType:   subjectTypeUser,
			wantSubject:    "alice",
			wantProjectKey: "my-project",
		},
		"InvalidFormatNoColon": {
			input:         "invalid",
			wantErrSubstr: "invalid external name format",
		},
		"UnknownSubjectType": {
			input:         "robot:r2d2",
			wantErrSubstr: "unknown subject type",
		},
		"ProjectKeyWithColons": {
			input:          "group:devs:proj:extra",
			wantSubjType:   subjectTypeGroup,
			wantSubject:    "devs",
			wantProjectKey: "proj:extra",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotType, gotSubject, gotProjectKey, err := parseExternalName(tc.input)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("parseExternalName(%q) expected error containing %q, got nil", tc.input, tc.wantErrSubstr)
				}

				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("parseExternalName(%q) error = %q, want containing %q", tc.input, err.Error(), tc.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseExternalName(%q) unexpected error: %v", tc.input, err)
			}

			if gotType != tc.wantSubjType {
				t.Errorf("parseExternalName(%q) subjType = %q, want %q", tc.input, gotType, tc.wantSubjType)
			}

			if gotSubject != tc.wantSubject {
				t.Errorf("parseExternalName(%q) subject = %q, want %q", tc.input, gotSubject, tc.wantSubject)
			}

			if gotProjectKey != tc.wantProjectKey {
				t.Errorf("parseExternalName(%q) projectKey = %q, want %q", tc.input, gotProjectKey, tc.wantProjectKey)
			}
		})
	}
}

// TestBuildExternalName tests building external names from spec.
func TestBuildExternalName(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.PermissionsParameters
		want string
	}{
		"GroupOnly": {
			spec: &v1alpha1.PermissionsParameters{GroupName: new("devs"), Permissions: []string{"scan"}},
			want: "group:devs",
		},
		"UserOnly": {
			spec: &v1alpha1.PermissionsParameters{Login: new("alice"), Permissions: []string{"scan"}},
			want: "user:alice",
		},
		"GroupWithProject": {
			spec: &v1alpha1.PermissionsParameters{GroupName: new("devs"), ProjectKey: new("proj-1"), Permissions: []string{"user"}},
			want: "group:devs:proj-1",
		},
		"UserWithProject": {
			spec: &v1alpha1.PermissionsParameters{Login: new("alice"), ProjectKey: new("proj-1"), Permissions: []string{"user"}},
			want: "user:alice:proj-1",
		},
		"EmptyProjectKeyNotAppended": {
			spec: &v1alpha1.PermissionsParameters{GroupName: new("devs"), ProjectKey: new(""), Permissions: []string{"scan"}},
			want: "group:devs",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := buildExternalName(tc.spec)
			if got != tc.want {
				t.Fatalf("buildExternalName() = %q, want %q", got, tc.want)
			}
		})
	}
}
