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

package almbitbucketcloud

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
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
	// testExternalName is the test ALM Bitbucket Cloud external name.
	testExternalName = "bitbucket-cloud-main"
	// testRenamedBitbucketCloudKey is a renamed Bitbucket Cloud key for testing.
	testRenamedBitbucketCloudKey = "bitbucket-cloud-renamed"
	// testClientID is the test Bitbucket Cloud OAuth consumer client ID.
	testClientID = "test-client-id"
	// testWorkspace is the test Bitbucket Cloud workspace.
	testWorkspace = "test-workspace"
	// bitbucketCloudClientSecretValue is the test OAuth client secret
	// value.
	bitbucketCloudClientSecretValue = "bitbucketCloudClientSecretValue"
)

// notALMBitbucketCloud is a test type that is not an ALMBitbucketCloud.
type notALMBitbucketCloud struct {
	resource.Managed
}

// mockGate is a mock implementation of the feature gate interface.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register registers a callback with the mock gate.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set sets a feature gate status in the mock.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// mockHTTPResponse creates a mock HTTP response with the given status code.
func mockHTTPResponse(statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode, Body: http.NoBody}
}

// checkError checks if an error matches expectations.
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

// newTestALMBitbucketCloud creates a test ALMBitbucketCloud resource.
func newTestALMBitbucketCloud(externalName string, clientSecretSelector *xpv1.LocalSecretKeySelector) *v1alpha1.ALMBitbucketCloud {
	alm := &v1alpha1.ALMBitbucketCloud{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-alm-bitbucket-cloud",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("alm-bitbucket-cloud-uid"),
		},
		Spec: v1alpha1.ALMBitbucketCloudSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider: v1alpha1.ALMBitbucketCloudParameters{
				Key:             testExternalName,
				ClientID:        testClientID,
				ClientSecretRef: clientSecretSelector,
				Workspace:       testWorkspace,
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(alm, externalName)
	}

	return alm
}

// clientSecretSecret creates a test secret with a client secret.
func clientSecretSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

// TestObserve tests observing ALMBitbucketCloud resource state.
func TestObserve(t *testing.T) {
	t.Parallel()

	clientSecretRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"}

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
		settingsClient *fake.MockALMSettingsBitbucketCloudClient
		args           args
		want           want
	}{
		"NotALMBitbucketCloud": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: &notALMBitbucketCloud{}},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: errNotALMBitbucketCloud},
		},
		"MissingExternalNameReturnsNotExists": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", clientSecretRef)},
			want:           want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"MissingClientSecretSelectorReturnsError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, nil)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"EmptyClientSecretReturnsError": {
			objects:        []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "client secret is empty"},
		},
		"MissingSavedSecretDoesNotFailObserve": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{BitbucketCloud: []sonar.BitbucketCloudDefinition{{Key: testExternalName, ClientID: testClientID, Workspace: testWorkspace}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.ClientID = testClientID
				alm.Status.AtProvider.Workspace = testWorkspace

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"MissingSavedSecretKeyDoesNotFailObserve": {
			objects: []runtime.Object{
				clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue),
				clientSecretSecret("connection-secret", "default", "other-key", "value"),
			},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{BitbucketCloud: []sonar.BitbucketCloudDefinition{{Key: testExternalName, ClientID: testClientID, Workspace: testWorkspace}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.ClientID = testClientID
				alm.Status.AtProvider.Workspace = testWorkspace

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveUpToDate": {
			objects: []runtime.Object{
				clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue),
				clientSecretSecret("connection-secret", "default", connectionDetailClientSecretKey, bitbucketCloudClientSecretValue),
			},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{BitbucketCloud: []sonar.BitbucketCloudDefinition{{Key: testExternalName, ClientID: testClientID, Workspace: testWorkspace}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.ClientID = testClientID
				alm.Status.AtProvider.Workspace = testWorkspace

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveNotUpToDate": {
			objects: []runtime.Object{
				clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue),
				clientSecretSecret("connection-secret", "default", connectionDetailClientSecretKey, "different"),
			},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{BitbucketCloud: []sonar.BitbucketCloudDefinition{{Key: testExternalName, ClientID: testClientID, Workspace: testWorkspace}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.ClientID = testClientID
				alm.Status.AtProvider.Workspace = testWorkspace

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"ListDefinitionsError": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api list failed")
			}},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want: want{observation: managed.ExternalObservation{}, errSubstr: "cannot list ALM settings definitions from SonarQube API"},
		},
		"DefinitionNotFoundReturnsNotExists": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
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
				integrationsClient: &fake.MockALMIntegrationsBitbucketCloudClient{},
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

// TestCreate tests creating an ALMBitbucketCloud resource.
func TestCreate(t *testing.T) {
	t.Parallel()

	clientSecretRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"}

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
		settingsClient *fake.MockALMSettingsBitbucketCloudClient
		args           args
		want           want
	}{
		"NotALMBitbucketCloud": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: &notALMBitbucketCloud{}},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: errNotALMBitbucketCloud},
		},
		"ClientSecretReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", nil)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"EmptyClientSecret": {
			objects:        []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", clientSecretRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "client secret is empty"},
		},
		"CreateError": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				CreateBitbucketCloudFn: func(_ *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api create failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", clientSecretRef)},
			want: want{creation: managed.ExternalCreation{}, errSubstr: "cannot create ALMBitbucketCloud resource in SonarQube API"},
		},
		"SuccessfulCreate": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				CreateBitbucketCloudFn: func(opt *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.ClientID != testClientID || opt.Workspace != testWorkspace || opt.ClientSecret != bitbucketCloudClientSecretValue {
						t.Fatalf("Create() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", clientSecretRef)},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{connectionDetailClientSecretKey: []byte(bitbucketCloudClientSecretValue)}}},
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
				integrationsClient: &fake.MockALMIntegrationsBitbucketCloudClient{},
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

			alm, ok := tc.args.mg.(*v1alpha1.ALMBitbucketCloud)
			if ok && name == "SuccessfulCreate" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != testExternalName {
					t.Fatalf("Create() external name = %q, want %q", gotExternalName, testExternalName)
				}
			}
		})
	}
}

// TestUpdate tests updating an ALMBitbucketCloud resource.
func TestUpdate(t *testing.T) { //nolint:gocognit // table-driven test covers all update paths
	t.Parallel()

	clientSecretRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"}

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
		settingsClient *fake.MockALMSettingsBitbucketCloudClient
		args           args
		want           want
		registerMG     bool // pre-register tc.args.mg with the fake kube client (needed for kubeClient.Update after key change)
	}{
		"NotALMBitbucketCloud": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: &notALMBitbucketCloud{}},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: errNotALMBitbucketCloud},
		},
		"MissingExternalName": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", clientSecretRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "external name is not set"},
		},
		"ClientSecretReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, nil)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"EmptyClientSecret": {
			objects:        []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "client secret is empty"},
		},
		"UpdateError": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				UpdateBitbucketCloudFn: func(_ *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api update failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update ALMBitbucketCloud resource in SonarQube API"},
		},
		"SuccessfulUpdateWithoutKeyChange": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				UpdateBitbucketCloudFn: func(opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != "" || opt.ClientID != testClientID || opt.Workspace != testWorkspace || opt.ClientSecret != bitbucketCloudClientSecretValue {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, clientSecretRef)},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailClientSecretKey: []byte(bitbucketCloudClientSecretValue)}}},
		},
		"SuccessfulUpdateWithKeyChange": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				UpdateBitbucketCloudFn: func(opt *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != testRenamedBitbucketCloudKey {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.Spec.ForProvider.Key = testRenamedBitbucketCloudKey

				return alm
			}()},
			want:       want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailClientSecretKey: []byte(bitbucketCloudClientSecretValue)}}},
			registerMG: true,
		},
		"UpdateKeyChangePersistError": {
			objects: []runtime.Object{clientSecretSecret("client-secret", "default", "clientSecret", bitbucketCloudClientSecretValue)},
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				UpdateBitbucketCloudFn: func(_ *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			// registerMG is false → ALMBitbucketCloud not in fake client → kubeClient.Update returns not-found
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.Spec.ForProvider.Key = testRenamedBitbucketCloudKey

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update external name annotation after key change"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()

			addCorev1Err := corev1.SchemeBuilder.AddToScheme(scheme)
			if addCorev1Err != nil {
				t.Fatalf("AddToScheme(corev1) unexpected error: %v", addCorev1Err)
			}

			addV1alpha1Err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
			if addV1alpha1Err != nil {
				t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", addV1alpha1Err)
			}

			builder := fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tc.objects...)

			if tc.registerMG {
				if alm, ok := tc.args.mg.(*v1alpha1.ALMBitbucketCloud); ok {
					builder = builder.WithRuntimeObjects(alm)
				}
			}

			kubeClient := builder.Build()
			e := &external{
				kubeClient:         kubeClient,
				integrationsClient: &fake.MockALMIntegrationsBitbucketCloudClient{},
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

			if alm, ok := tc.args.mg.(*v1alpha1.ALMBitbucketCloud); ok && name == "SuccessfulUpdateWithKeyChange" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != testRenamedBitbucketCloudKey {
					t.Fatalf("Update() external name = %q, want %q", gotExternalName, testRenamedBitbucketCloudKey)
				}

				// Verify the external name was persisted to the Kubernetes API.
				persisted := &v1alpha1.ALMBitbucketCloud{}

				getErr := kubeClient.Get(tc.args.ctx, types.NamespacedName{Name: alm.Name, Namespace: alm.Namespace}, persisted)
				if getErr != nil {
					t.Fatalf("Update() kubeClient.Get() error: %v", getErr)
				}

				if gotPersistedName := meta.GetExternalName(persisted); gotPersistedName != testRenamedBitbucketCloudKey {
					t.Fatalf("Update() persisted external name = %q, want %q", gotPersistedName, testRenamedBitbucketCloudKey)
				}
			}
		})
	}
}

// TestDelete tests deleting an ALMBitbucketCloud resource.
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
		settingsClient *fake.MockALMSettingsBitbucketCloudClient
		args           args
		want           want
		wantCall       bool
	}{
		"NotALMBitbucketCloud": {
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: &notALMBitbucketCloud{}},
			want:           want{deletion: managed.ExternalDelete{}, errSubstr: errNotALMBitbucketCloud},
			wantCall:       false,
		},
		"MissingExternalName": {
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{},
			args:           args{ctx: context.Background(), mg: newTestALMBitbucketCloud("", nil)},
			want:           want{deletion: managed.ExternalDelete{}},
			wantCall:       false,
		},
		"DeleteError": {
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api delete failed")
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, nil)},
			want:     want{deletion: managed.ExternalDelete{}, errSubstr: "cannot delete ALMBitbucketCloud resource in SonarQube API"},
			wantCall: true,
		},
		"SuccessfulDelete": {
			settingsClient: &fake.MockALMSettingsBitbucketCloudClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMBitbucketCloud(testExternalName, nil)},
			want:     want{deletion: managed.ExternalDelete{}},
			wantCall: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{
				integrationsClient: &fake.MockALMIntegrationsBitbucketCloudClient{},
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

			alm, ok := tc.args.mg.(*v1alpha1.ALMBitbucketCloud)
			if ok {
				if cond := alm.GetCondition(xpv1.TypeReady); cond.Reason != xpv1.ReasonDeleting {
					t.Fatalf("Delete() expected deleting condition, got %s", cond.Reason)
				}
			}
		})
	}
}

// TestDisconnect tests disconnecting an ALMBitbucketCloud resource.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{integrationsClient: &fake.MockALMIntegrationsBitbucketCloudClient{}, settingsClient: &fake.MockALMSettingsBitbucketCloudClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// TestGetSavedClientSecret tests retrieving the saved client secret from an
// ALMBitbucketCloud resource.
func TestGetSavedClientSecret(t *testing.T) {
	t.Parallel()

	clientSecretRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"}

	cases := map[string]struct {
		objects   []runtime.Object
		alm       *v1alpha1.ALMBitbucketCloud
		wantValue string
		wantError string
	}{
		"NoWriteSecretRef": {
			objects:   nil,
			alm:       newTestALMBitbucketCloud(testExternalName, clientSecretRef),
			wantValue: "",
			wantError: "",
		},
		"WriteSecretNameEmpty": {
			objects: nil,
			alm: func() *v1alpha1.ALMBitbucketCloud {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: ""})

				return alm
			}(),
			wantValue: "",
			wantError: "",
		},
		"MissingSecretReturnsEmptyValue": {
			objects: nil,
			alm: func() *v1alpha1.ALMBitbucketCloud {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing"})

				return alm
			}(),
			wantValue: "",
			wantError: "",
		},
		"MissingSecretKeyReturnsEmptyValue": {
			objects: []runtime.Object{clientSecretSecret("connection-secret", "default", "other-key", "value")},
			alm: func() *v1alpha1.ALMBitbucketCloud {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantValue: "",
			wantError: "",
		},
		"SuccessfulRead": {
			objects: []runtime.Object{clientSecretSecret("connection-secret", "default", connectionDetailClientSecretKey, "saved-secret")},
			alm: func() *v1alpha1.ALMBitbucketCloud {
				alm := newTestALMBitbucketCloud(testExternalName, clientSecretRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantValue: "saved-secret",
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

			got, err := e.getSavedClientSecret(context.Background(), tc.alm)
			checkError(t, "getSavedClientSecret", tc.wantError, err)

			if tc.wantError != "" {
				return
			}

			if got != tc.wantValue {
				t.Fatalf("getSavedClientSecret() = %q, want %q", got, tc.wantValue)
			}
		})
	}
}

// TestConnect tests connecting to an ALMBitbucketCloud resource.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TypeAssertionError", func(t *testing.T) {
		t.Parallel()

		c := &connector{}

		_, err := c.Connect(context.Background(), &notALMBitbucketCloud{})
		if err == nil || !strings.Contains(err.Error(), errNotALMBitbucketCloud) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errNotALMBitbucketCloud)
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

		alm := newTestALMBitbucketCloud("bitbucket-cloud-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"})

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

		alm := newTestALMBitbucketCloud("bitbucket-cloud-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"})
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

		secret := clientSecretSecret("provider-secret", "default", "token", "provider-token")

		kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, secret).Build()

		alm := newTestALMBitbucketCloud("bitbucket-cloud-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "client-secret"}, Key: "clientSecret"})
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

// TestSetupGatedRegistersALMBitbucketCloudGVK tests that SetupGated
// registers the ALMBitbucketCloud GVK.
func TestSetupGatedRegistersALMBitbucketCloudGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.ALMBitbucketCloudGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails tests that the SetupGated
// callback panics on setup failure.
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
