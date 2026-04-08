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

package almgitlab

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

const (
	testExternalName = "gitlab-main"
	testGitLabURL    = "https://gitlab.example.com"
)

type notALMGitLab struct {
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
	return &http.Response{StatusCode: statusCode, Body: http.NoBody}
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

func newTestALMGitLab(externalName string, tokenSelector *xpv1.LocalSecretKeySelector) *v1alpha1.ALMGitLab {
	alm := &v1alpha1.ALMGitLab{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-alm-gitlab",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("alm-gitlab-uid"),
		},
		Spec: v1alpha1.ALMGitLabSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider: v1alpha1.ALMGitLabParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{
					URL:                          testGitLabURL,
					Key:                          testExternalName,
					PersonalAccessTokenSecretRef: tokenSelector,
				},
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(alm, externalName)
	}

	return alm
}

func tokenSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

func TestObserve(t *testing.T) {
	t.Parallel()

	tokenRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		objects        []runtime.Object
		settingsClient *fake.MockALMSettingsGitLabClient
		args           args
		want           want
	}{
		"NotALMGitLab": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitLab{}},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: errNotALMGitLab},
		},
		"MissingExternalNameReturnsNotExists": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab("", tokenRef)},
			want:           want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"MissingPATSelectorReturnsError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, nil)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPATReturnsError": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, tokenRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "personal access token is empty"},
		},
		"MissingSavedTokenSecretDoesNotFailObserve": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Gitlab: []sonar.GitlabDefinition{{Key: testExternalName, URL: testGitLabURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitLabURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveUpToDate": {
			objects: []runtime.Object{
				tokenSecret("pat-secret", "default", "token", "pat-value"),
				tokenSecret("connection-secret", "default", connectionDetailTokenKey, "pat-value"),
			},
			settingsClient: &fake.MockALMSettingsGitLabClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Gitlab: []sonar.GitlabDefinition{{Key: testExternalName, URL: testGitLabURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitLabURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveNotUpToDate": {
			objects: []runtime.Object{
				tokenSecret("pat-secret", "default", "token", "pat-value"),
				tokenSecret("connection-secret", "default", connectionDetailTokenKey, "different"),
			},
			settingsClient: &fake.MockALMSettingsGitLabClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Gitlab: []sonar.GitlabDefinition{{Key: testExternalName, URL: testGitLabURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitLabURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()

			err := corev1.SchemeBuilder.AddToScheme(scheme)
			if err != nil {
				t.Fatalf("AddToScheme() unexpected error: %v", err)
			}

			kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tc.objects...).Build()
			e := &external{
				kubeClient:         kubeClient,
				integrationsClient: &fake.MockALMIntegrationsGitLabClient{},
				settingsClient:     tc.settingsClient,
			}

			got, err := e.Observe(tc.args.ctx, tc.args.mg)
			checkError(t, "Observe", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Fatalf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	tokenRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation  managed.ExternalCreation
		errSubstr string
	}

	cases := map[string]struct {
		objects        []runtime.Object
		settingsClient *fake.MockALMSettingsGitLabClient
		args           args
		want           want
	}{
		"NotALMGitLab": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitLab{}},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: errNotALMGitLab},
		},
		"PATReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab("", nil)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPAT": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab("", tokenRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "personal access token is empty"},
		},
		"CreateError": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{
				CreateGitlabFn: func(_ *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api create failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitLab("", tokenRef)},
			want: want{creation: managed.ExternalCreation{}, errSubstr: "cannot create ALMGitLab resource in SonarQube API"},
		},
		"SuccessfulCreate": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{
				CreateGitlabFn: func(opt *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.URL != testGitLabURL || opt.PersonalAccessToken != "pat-value" {
						t.Fatalf("Create() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitLab("", tokenRef)},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte("pat-value")}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()

			err := corev1.SchemeBuilder.AddToScheme(scheme)
			if err != nil {
				t.Fatalf("AddToScheme() unexpected error: %v", err)
			}

			kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tc.objects...).Build()
			e := &external{
				kubeClient:         kubeClient,
				integrationsClient: &fake.MockALMIntegrationsGitLabClient{},
				settingsClient:     tc.settingsClient,
			}

			got, err := e.Create(tc.args.ctx, tc.args.mg)
			checkError(t, "Create", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Fatalf("Create() mismatch (-want +got):\n%s", diff)
			}

			alm, ok := tc.args.mg.(*v1alpha1.ALMGitLab)
			if ok && name == "SuccessfulCreate" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != "gitlab-main" {
					t.Fatalf("Create() external name = %q, want %q", gotExternalName, "gitlab-main")
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tokenRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		update    managed.ExternalUpdate
		errSubstr string
	}

	cases := map[string]struct {
		objects        []runtime.Object
		settingsClient *fake.MockALMSettingsGitLabClient
		args           args
		want           want
	}{
		"NotALMGitLab": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitLab{}},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: errNotALMGitLab},
		},
		"MissingExternalName": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab("", tokenRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "external name is not set"},
		},
		"PATReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, nil)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPAT": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, tokenRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "personal access token is empty"},
		},
		"UpdateError": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{
				UpdateGitlabFn: func(_ *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api update failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, tokenRef)},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update ALMGitLab resource in SonarQube API"},
		},
		"SuccessfulUpdateWithoutKeyChange": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{
				UpdateGitlabFn: func(opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != "" || opt.URL != testGitLabURL || opt.PersonalAccessToken != "pat-value" {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitLab(testExternalName, tokenRef)},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte("pat-value")}}},
		},
		"SuccessfulUpdateWithKeyChange": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", "pat-value")},
			settingsClient: &fake.MockALMSettingsGitLabClient{
				UpdateGitlabFn: func(opt *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != "gitlab-renamed" {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.Spec.ForProvider.Key = "gitlab-renamed"

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte("pat-value")}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()

			err := corev1.SchemeBuilder.AddToScheme(scheme)
			if err != nil {
				t.Fatalf("AddToScheme() unexpected error: %v", err)
			}

			kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tc.objects...).Build()
			e := &external{
				kubeClient:         kubeClient,
				integrationsClient: &fake.MockALMIntegrationsGitLabClient{},
				settingsClient:     tc.settingsClient,
			}

			got, err := e.Update(tc.args.ctx, tc.args.mg)
			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Fatalf("Update() mismatch (-want +got):\n%s", diff)
			}

			alm, ok := tc.args.mg.(*v1alpha1.ALMGitLab)
			if ok && name == "SuccessfulUpdateWithKeyChange" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != "gitlab-renamed" {
					t.Fatalf("Update() external name = %q, want %q", gotExternalName, "gitlab-renamed")
				}
			}
		})
	}
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
		settingsClient *fake.MockALMSettingsGitLabClient
		args           args
		want           want
		wantCall       bool
	}{
		"NotALMGitLab": {
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitLab{}},
			want:           want{deletion: managed.ExternalDelete{}, errSubstr: errNotALMGitLab},
			wantCall:       false,
		},
		"MissingExternalName": {
			settingsClient: &fake.MockALMSettingsGitLabClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitLab("", nil)},
			want:           want{deletion: managed.ExternalDelete{}},
			wantCall:       false,
		},
		"DeleteError": {
			settingsClient: &fake.MockALMSettingsGitLabClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != "gitlab-main" {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api delete failed")
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMGitLab("gitlab-main", nil)},
			want:     want{deletion: managed.ExternalDelete{}, errSubstr: "cannot delete ALMGitLab resource in SonarQube API"},
			wantCall: true,
		},
		"SuccessfulDelete": {
			settingsClient: &fake.MockALMSettingsGitLabClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != "gitlab-main" {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMGitLab("gitlab-main", nil)},
			want:     want{deletion: managed.ExternalDelete{}},
			wantCall: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{
				integrationsClient: &fake.MockALMIntegrationsGitLabClient{},
				settingsClient:     tc.settingsClient,
			}

			got, err := e.Delete(tc.args.ctx, tc.args.mg)
			checkError(t, "Delete", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.deletion, got); diff != "" {
				t.Fatalf("Delete() mismatch (-want +got):\n%s", diff)
			}

			alm, ok := tc.args.mg.(*v1alpha1.ALMGitLab)
			if ok {
				if cond := alm.GetCondition(xpv1.TypeReady); cond.Reason != xpv1.ReasonDeleting {
					t.Fatalf("Delete() expected deleting condition, got %s", cond.Reason)
				}
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{integrationsClient: &fake.MockALMIntegrationsGitLabClient{}, settingsClient: &fake.MockALMSettingsGitLabClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

func TestGetSavedAPIToken(t *testing.T) {
	t.Parallel()

	tokenRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"}

	cases := map[string]struct {
		objects   []runtime.Object
		alm       *v1alpha1.ALMGitLab
		wantToken string
		wantError string
	}{
		"NoWriteSecretRef": {
			objects:   nil,
			alm:       newTestALMGitLab(testExternalName, tokenRef),
			wantToken: "",
			wantError: "",
		},
		"WriteSecretNameEmpty": {
			objects: nil,
			alm: func() *v1alpha1.ALMGitLab {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: ""})

				return alm
			}(),
			wantToken: "",
			wantError: "",
		},
		"MissingSecretReturnsEmptyToken": {
			objects: nil,
			alm: func() *v1alpha1.ALMGitLab {
				alm := newTestALMGitLab(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing"})

				return alm
			}(),
			wantToken: "",
			wantError: "",
		},
		"SuccessfulRead": {
			objects: []runtime.Object{tokenSecret("connection-secret", "default", connectionDetailTokenKey, "saved-token")},
			alm: func() *v1alpha1.ALMGitLab {
				alm := newTestALMGitLab("gitlab-main", tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantToken: "saved-token",
			wantError: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()

			err := corev1.SchemeBuilder.AddToScheme(scheme)
			if err != nil {
				t.Fatalf("AddToScheme() unexpected error: %v", err)
			}

			kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tc.objects...).Build()
			e := &external{kubeClient: kubeClient}

			got, err := e.getSavedAPIToken(context.Background(), tc.alm)
			checkError(t, "getSavedAPIToken", tc.wantError, err)

			if tc.wantError != "" {
				return
			}

			if got != tc.wantToken {
				t.Fatalf("getSavedAPIToken() = %q, want %q", got, tc.wantToken)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TypeAssertionError", func(t *testing.T) {
		t.Parallel()

		c := &connector{}

		_, err := c.Connect(context.Background(), &notALMGitLab{})
		if err == nil || !strings.Contains(err.Error(), errNotALMGitLab) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errNotALMGitLab)
		}
	})

	t.Run("TrackUsageError", func(t *testing.T) {
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

		alm := newTestALMGitLab("gitlab-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})

		c := &connector{
			kube:  kubeClient,
			usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err = c.Connect(context.Background(), alm)
		if err == nil || !strings.Contains(err.Error(), errTrackPCUsage) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errTrackPCUsage)
		}
	})

	t.Run("GetConfigError", func(t *testing.T) {
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

		alm := newTestALMGitLab("gitlab-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})
		alm.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kubeClient,
			usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err = c.Connect(context.Background(), alm)
		if err == nil || !strings.Contains(err.Error(), errGetPC) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errGetPC)
		}
	})

	t.Run("Success", func(t *testing.T) {
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
						SecretRef: &xpv1.SecretKeySelector{SecretReference: xpv1.SecretReference{Name: "provider-secret", Namespace: "default"}, Key: "token"},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
		}

		secret := tokenSecret("provider-secret", "default", "token", "provider-token")

		kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, secret).Build()

		alm := newTestALMGitLab("gitlab-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})
		alm.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kubeClient,
			usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		}

		got, err := c.Connect(context.Background(), alm)
		if err != nil {
			t.Fatalf("Connect() unexpected error: %v", err)
		}

		if got == nil {
			t.Fatal("Connect() expected non-nil external client")
		}

		if _, ok := got.(*external); !ok {
			t.Fatalf("Connect() returned %T, want *external", got)
		}
	})
}

func TestSetupGatedRegistersALMGitLabGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.ALMGitLabGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

func TestSetupGatedCallbackPanicsWhenSetupFails(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := controller.DefaultOptions()
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if g.callback == nil {
		t.Fatal("SetupGated() expected callback to be registered")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}
