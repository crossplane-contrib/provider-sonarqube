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

package group

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
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
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
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

type notGroup struct {
	resource.Managed
}

type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

func mockHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       http.NoBody,
	}
}

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
		t.Errorf("%s() error mismatch: want %q, got %q", method, wantErrSubstr, gotErr.Error())
	}
}

func newTestGroup(externalName string, spec v1alpha1.GroupParameters) *v1alpha1.Group {
	g := &v1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-group",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.GroupSpec{ForProvider: spec},
	}

	if externalName != "" {
		meta.SetExternalName(g, externalName)
	}

	return g
}

func TestObserve(t *testing.T) {
	t.Parallel()

	tDesc := "engineering"

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		client *fake.MockGroupsClient
		args   args
		want   want
	}{
		"NotGroupError": {
			client: &fake.MockGroupsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notGroup{},
			},
			want: want{errSubstr: errNotGroup},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockGroupsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("", v1alpha1.GroupParameters{Name: "devs"}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"FetchNotFoundReturnsNotExists": {
			client: &fake.MockGroupsClient{
				FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"FetchErrorReturnsWrappedError": {
			client: &fake.MockGroupsClient{
				FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"}),
			},
			want: want{errSubstr: errObserveGroup},
		},
		"SuccessfulObserveUpToDate": {
			client: &fake.MockGroupsClient{
				FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "group-1", Name: "devs", Description: "engineering"}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: newTestGroup("group-1", v1alpha1.GroupParameters{
					Name:        "devs",
					Description: &tDesc,
				}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveLateInitialization": {
			client: &fake.MockGroupsClient{
				FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "group-1", Name: "devs", Description: "engineering"}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs", Description: nil}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveNotUpToDate": {
			client: &fake.MockGroupsClient{
				FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "group-1", Name: "admins", Description: "engineering"}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs", Description: &tDesc}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			checkError(t, "Observe", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("ObserveWritesStatusObservation", func(t *testing.T) {
		t.Parallel()

		group := newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"})
		e := &external{client: &fake.MockGroupsClient{
			FetchGroupFn: func(_ string) (*sonar.Group, *http.Response, error) {
				return &sonar.Group{Id: "group-1", Name: "devs", Description: "engineering", Managed: true, Default: false}, mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Observe(context.Background(), group)
		if err != nil {
			t.Fatalf("Observe() unexpected error: %v", err)
		}

		if group.Status.AtProvider.ID != "group-1" || group.Status.AtProvider.Name != "devs" {
			t.Fatalf("Observe() status not updated correctly: %+v", group.Status.AtProvider)
		}
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	description := "engineering"

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation  managed.ExternalCreation
		errSubstr string
	}

	cases := map[string]struct {
		client *fake.MockGroupsClient
		args   args
		want   want
	}{
		"NotGroupError": {
			client: &fake.MockGroupsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notGroup{},
			},
			want: want{errSubstr: errNotGroup},
		},
		"CreateFails": {
			client: &fake.MockGroupsClient{
				CreateGroupFn: func(_ *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
					return nil, nil, errors.New("create error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("", v1alpha1.GroupParameters{Name: "devs", Description: &description}),
			},
			want: want{errSubstr: errCreateGroup},
		},
		"CreateReturnsNilGroup": {
			client: &fake.MockGroupsClient{
				CreateGroupFn: func(_ *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("", v1alpha1.GroupParameters{Name: "devs"}),
			},
			want: want{errSubstr: errNoGroupID},
		},
		"CreateReturnsEmptyID": {
			client: &fake.MockGroupsClient{
				CreateGroupFn: func(_ *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "", Name: "devs"}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("", v1alpha1.GroupParameters{Name: "devs"}),
			},
			want: want{errSubstr: errNoGroupID},
		},
		"SuccessfulCreate": {
			client: &fake.MockGroupsClient{
				CreateGroupFn: func(_ *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "group-1", Name: "devs"}, mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroup("", v1alpha1.GroupParameters{Name: "devs", Description: &description}),
			},
			want: want{creation: managed.ExternalCreation{}},
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

			group, ok := tc.args.mg.(*v1alpha1.Group)
			if ok && name == "SuccessfulCreate" {
				if gotName := meta.GetExternalName(group); gotName != "group-1" {
					t.Fatalf("Create() external name = %q, want %q", gotName, "group-1")
				}
			}
		})
	}

	t.Run("CreatePassesExpectedOptions", func(t *testing.T) {
		t.Parallel()

		desc := "engineering"
		group := newTestGroup("", v1alpha1.GroupParameters{Name: "devs", Description: &desc})

		called := false
		e := &external{client: &fake.MockGroupsClient{
			CreateGroupFn: func(opt *sonar.AuthorizationsCreateGroupOptions) (*sonar.Group, *http.Response, error) {
				called = true

				if opt == nil || opt.Name != "devs" || opt.Description != desc {
					t.Fatalf("Create() unexpected options: %+v", opt)
				}

				return &sonar.Group{Id: "group-1", Name: "devs"}, mockHTTPResponse(http.StatusCreated), nil
			},
		}}

		_, err := e.Create(context.Background(), group)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if !called {
			t.Fatal("Create() expected CreateGroup to be called")
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		update    managed.ExternalUpdate
		errSubstr string
	}

	cases := map[string]struct {
		client *fake.MockGroupsClient
		args   args
		want   want
	}{
		"NotGroupError": {
			client: &fake.MockGroupsClient{},
			args:   args{ctx: context.Background(), mg: &notGroup{}},
			want:   want{errSubstr: errNotGroup},
		},
		"EmptyExternalName": {
			client: &fake.MockGroupsClient{},
			args:   args{ctx: context.Background(), mg: newTestGroup("", v1alpha1.GroupParameters{Name: "devs"})},
			want:   want{errSubstr: "cannot update Group without external name"},
		},
		"UpdateFails": {
			client: &fake.MockGroupsClient{
				UpdateGroupFn: func(_ string, _ *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error) {
					return nil, nil, errors.New("update error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"})},
			want: want{errSubstr: errUpdateGroup},
		},
		"SuccessfulUpdate": {
			client: &fake.MockGroupsClient{
				UpdateGroupFn: func(_ string, _ *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error) {
					return &sonar.Group{Id: "group-1", Name: "devs"}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"})},
			want: want{update: managed.ExternalUpdate{}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("UpdatePassesExternalNameAndOptions", func(t *testing.T) {
		t.Parallel()

		desc := "engineering"
		group := newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs", Description: &desc})

		e := &external{client: &fake.MockGroupsClient{
			UpdateGroupFn: func(groupID string, opt *sonar.AuthorizationsUpdateGroupOptions) (*sonar.Group, *http.Response, error) {
				if groupID != "group-1" {
					t.Fatalf("Update() groupID = %q, want %q", groupID, "group-1")
				}

				if opt == nil || opt.Name != "devs" || opt.Description == nil || *opt.Description != desc {
					t.Fatalf("Update() unexpected options: %+v", opt)
				}

				return &sonar.Group{Id: "group-1"}, mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Update(context.Background(), group)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}
	})
}

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
		client *fake.MockGroupsClient
		args   args
		want   want
	}{
		"NotGroupError": {
			client: &fake.MockGroupsClient{},
			args:   args{ctx: context.Background(), mg: &notGroup{}},
			want:   want{errSubstr: errNotGroup},
		},
		"NoExternalNameReturnsSuccess": {
			client: &fake.MockGroupsClient{},
			args:   args{ctx: context.Background(), mg: newTestGroup("", v1alpha1.GroupParameters{Name: "devs"})},
			want:   want{deletion: managed.ExternalDelete{}},
		},
		"DeleteFails": {
			client: &fake.MockGroupsClient{
				DeleteGroupFn: func(_ string) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("delete error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"})},
			want: want{errSubstr: errDeleteGroup},
		},
		"SuccessfulDelete": {
			client: &fake.MockGroupsClient{
				DeleteGroupFn: func(_ string) (*http.Response, error) {
					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"})},
			want: want{deletion: managed.ExternalDelete{}},
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

	t.Run("DeleteUsesExternalName", func(t *testing.T) {
		t.Parallel()

		called := false
		e := &external{client: &fake.MockGroupsClient{
			DeleteGroupFn: func(groupID string) (*http.Response, error) {
				called = true

				if groupID != "group-1" {
					t.Fatalf("Delete() groupID = %q, want %q", groupID, "group-1")
				}

				return mockHTTPResponse(http.StatusNoContent), nil
			},
		}}

		_, err := e.Delete(context.Background(), newTestGroup("group-1", v1alpha1.GroupParameters{Name: "devs"}))
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if !called {
			t.Fatal("Delete() expected DeleteGroup to be called")
		}
	})
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockGroupsClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notGroup{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Group type, got nil")
	}

	if !strings.Contains(err.Error(), errNotGroup) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errNotGroup)
	}
}

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

	group := &v1alpha1.Group{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.GroupKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "default",
			UID:       types.UID("group-uid"),
		},
		Spec: v1alpha1.GroupSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider:         v1alpha1.GroupParameters{Name: "devs"},
		},
	}

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: func(_ common.Config) iam.GroupsClient { return &fake.MockGroupsClient{} },
	}

	_, err = c.Connect(context.Background(), group)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errTrackPCUsage) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errTrackPCUsage)
	}
}

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

	group := &v1alpha1.Group{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.GroupKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "default",
			UID:       types.UID("group-uid"),
		},
		Spec: v1alpha1.GroupSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider:         v1alpha1.GroupParameters{Name: "devs"},
		},
	}
	group.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: func(_ common.Config) iam.GroupsClient { return &fake.MockGroupsClient{} },
	}

	_, err = c.Connect(context.Background(), group)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errGetPC) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errGetPC)
	}
}

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

	group := &v1alpha1.Group{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.GroupKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "default",
			UID:       types.UID("group-uid"),
		},
		Spec: v1alpha1.GroupSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider:         v1alpha1.GroupParameters{Name: "devs"},
		},
	}
	group.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	var capturedConfig common.Config

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: func(config common.Config) iam.GroupsClient {
			capturedConfig = config

			return &fake.MockGroupsClient{}
		},
	}

	got, err := c.Connect(context.Background(), group)
	if err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("Connect() expected non-nil external client")
	}

	if capturedConfig.BaseURL != "http://localhost:9000" || capturedConfig.Token != "my-token" {
		t.Fatalf("Connect() config mismatch: %+v", capturedConfig)
	}

	if capturedConfig.AuthType != common.PersonalAccessToken {
		t.Fatalf("Connect() auth type = %q, want %q", capturedConfig.AuthType, common.PersonalAccessToken)
	}

	if _, ok := got.(*external); !ok {
		t.Fatalf("Connect() returned %T, want *external", got)
	}
}

func TestSetupGatedRegistersGroupGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.GroupGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}
