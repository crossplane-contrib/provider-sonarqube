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

package almazure

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
	// testExternalName is the test ALM Azure external name.
	testExternalName = "azure-main"
	// testRenamedAzureKey is a renamed Azure key for testing.
	testRenamedAzureKey = "azure-renamed"
	// testAzureURL is the test Azure URL.
	testAzureURL = "https://dev.azure.com/example"
	// azurePATValue is the test Azure personal access token value.
	azurePATValue = "azurePATValue"
)

// notALMAzure is a test type that is not an ALMAzure.
type notALMAzure struct {
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

// newTestALMAzure creates a test ALMAzure resource.
func newTestALMAzure(externalName string, tokenSelector *xpv1.LocalSecretKeySelector) *v1alpha1.ALMAzure {
	alm := &v1alpha1.ALMAzure{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-alm-azure",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("alm-azure-uid"),
		},
		Spec: v1alpha1.ALMAzureSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider: v1alpha1.ALMAzureParameters{
				ALMCommonParameters: v1alpha1.ALMCommonParameters{
					URL:                          testAzureURL,
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

// tokenSecret creates a test secret with a personal access token.
func tokenSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

// TestObserve tests observing ALMAzure resource state.
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
		settingsClient *fake.MockALMSettingsAzureClient
		args           args
		want           want
	}{
		"NotALMAzure": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: &notALMAzure{}},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: errNotALMAzure},
		},
		"MissingExternalNameReturnsNotExists": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure("", tokenRef)},
			want:           want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"MissingPATSelectorReturnsError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, nil)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPATReturnsError": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, tokenRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "personal access token is empty"},
		},
		"MissingSavedTokenSecretDoesNotFailObserve": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Azure: []sonar.AzureDefinition{{Key: testExternalName, URL: testAzureURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testAzureURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"MissingSavedTokenKeyDoesNotFailObserve": {
			objects: []runtime.Object{
				tokenSecret("pat-secret", "default", "token", azurePATValue),
				tokenSecret("connection-secret", "default", "other-key", "value"),
			},
			settingsClient: &fake.MockALMSettingsAzureClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Azure: []sonar.AzureDefinition{{Key: testExternalName, URL: testAzureURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testAzureURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveUpToDate": {
			objects: []runtime.Object{
				tokenSecret("pat-secret", "default", "token", azurePATValue),
				tokenSecret("connection-secret", "default", connectionDetailTokenKey, azurePATValue),
			},
			settingsClient: &fake.MockALMSettingsAzureClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Azure: []sonar.AzureDefinition{{Key: testExternalName, URL: testAzureURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testAzureURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveNotUpToDate": {
			objects: []runtime.Object{
				tokenSecret("pat-secret", "default", "token", azurePATValue),
				tokenSecret("connection-secret", "default", connectionDetailTokenKey, "different"),
			},
			settingsClient: &fake.MockALMSettingsAzureClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Azure: []sonar.AzureDefinition{{Key: testExternalName, URL: testAzureURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testAzureURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"NoConnectionSecretRefTriggersUpdate": {
			// When writeConnectionSecretToRef is not set (bypassing CRD validation), the
			// controller cannot compare stored vs current secret values and treats the
			// resource as out-of-date so the next Update re-writes the connection secret.
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Azure: []sonar.AzureDefinition{{Key: testExternalName, URL: testAzureURL}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				// No SetWriteConnectionSecretToReference → saved token is empty string.
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testAzureURL

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
				integrationsClient: &fake.MockALMIntegrationsAzureClient{},
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

// TestCreate tests creating an ALMAzure resource.
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
		settingsClient *fake.MockALMSettingsAzureClient
		args           args
		want           want
	}{
		"NotALMAzure": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: &notALMAzure{}},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: errNotALMAzure},
		},
		"PATReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure("", nil)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPAT": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure("", tokenRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "personal access token is empty"},
		},
		"CreateError": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				CreateAzureFn: func(_ *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api create failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMAzure("", tokenRef)},
			want: want{creation: managed.ExternalCreation{}, errSubstr: "cannot create ALMAzure resource in SonarQube API"},
		},
		"SuccessfulCreate": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				CreateAzureFn: func(opt *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.URL != testAzureURL || opt.PersonalAccessToken != azurePATValue {
						t.Fatalf("Create() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMAzure("", tokenRef)},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte(azurePATValue)}}},
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
				integrationsClient: &fake.MockALMIntegrationsAzureClient{},
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

			alm, ok := tc.args.mg.(*v1alpha1.ALMAzure)
			if ok && name == "SuccessfulCreate" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != "azure-main" {
					t.Fatalf("Create() external name = %q, want %q", gotExternalName, "azure-main")
				}
			}
		})
	}
}

// TestUpdate tests updating an ALMAzure resource.
func TestUpdate(t *testing.T) { //nolint:gocognit // table-driven test covers all update paths
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
		settingsClient *fake.MockALMSettingsAzureClient
		args           args
		want           want
		registerMG     bool // pre-register tc.args.mg with the fake kube client (needed for kubeClient.Update after key change)
	}{
		"NotALMAzure": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: &notALMAzure{}},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: errNotALMAzure},
		},
		"MissingExternalName": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure("", tokenRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "external name is not set"},
		},
		"PATReadError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, nil)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "cannot get personal access token from secret reference"},
		},
		"EmptyPAT": {
			objects:        []runtime.Object{tokenSecret("pat-secret", "default", "token", "")},
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, tokenRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "personal access token is empty"},
		},
		"UpdateError": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				UpdateAzureFn: func(_ *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api update failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, tokenRef)},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update ALMAzure resource in SonarQube API"},
		},
		"SuccessfulUpdateWithoutKeyChange": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				UpdateAzureFn: func(opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != "" || opt.URL != testAzureURL || opt.PersonalAccessToken != azurePATValue {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMAzure(testExternalName, tokenRef)},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte(azurePATValue)}}},
		},
		"SuccessfulUpdateWithKeyChange": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				UpdateAzureFn: func(opt *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != testRenamedAzureKey {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.Spec.ForProvider.Key = testRenamedAzureKey

				return alm
			}()},
			want:       want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailTokenKey: []byte(azurePATValue)}}},
			registerMG: true,
		},
		"UpdateKeyChangePersistError": {
			objects: []runtime.Object{tokenSecret("pat-secret", "default", "token", azurePATValue)},
			settingsClient: &fake.MockALMSettingsAzureClient{
				UpdateAzureFn: func(_ *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			// registerMG is false → ALMAzure not in fake client → kubeClient.Update returns not-found
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.Spec.ForProvider.Key = testRenamedAzureKey

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
				if alm, ok := tc.args.mg.(*v1alpha1.ALMAzure); ok {
					builder = builder.WithRuntimeObjects(alm)
				}
			}

			kubeClient := builder.Build()
			e := &external{
				kubeClient:         kubeClient,
				integrationsClient: &fake.MockALMIntegrationsAzureClient{},
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

			if alm, ok := tc.args.mg.(*v1alpha1.ALMAzure); ok && name == "SuccessfulUpdateWithKeyChange" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != testRenamedAzureKey {
					t.Fatalf("Update() external name = %q, want %q", gotExternalName, testRenamedAzureKey)
				}

				// Verify the external name was persisted to the Kubernetes API.
				persisted := &v1alpha1.ALMAzure{}

				getErr := kubeClient.Get(tc.args.ctx, types.NamespacedName{Name: alm.Name, Namespace: alm.Namespace}, persisted)
				if getErr != nil {
					t.Fatalf("Update() kubeClient.Get() error: %v", getErr)
				}

				if gotPersistedName := meta.GetExternalName(persisted); gotPersistedName != testRenamedAzureKey {
					t.Fatalf("Update() persisted external name = %q, want %q", gotPersistedName, testRenamedAzureKey)
				}
			}
		})
	}
}

// TestDelete tests deleting an ALMAzure resource.
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
		settingsClient *fake.MockALMSettingsAzureClient
		args           args
		want           want
		wantCall       bool
	}{
		"NotALMAzure": {
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: &notALMAzure{}},
			want:           want{deletion: managed.ExternalDelete{}, errSubstr: errNotALMAzure},
			wantCall:       false,
		},
		"MissingExternalName": {
			settingsClient: &fake.MockALMSettingsAzureClient{},
			args:           args{ctx: context.Background(), mg: newTestALMAzure("", nil)},
			want:           want{deletion: managed.ExternalDelete{}},
			wantCall:       false,
		},
		"DeleteError": {
			settingsClient: &fake.MockALMSettingsAzureClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api delete failed")
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMAzure("azure-main", nil)},
			want:     want{deletion: managed.ExternalDelete{}, errSubstr: "cannot delete ALMAzure resource in SonarQube API"},
			wantCall: true,
		},
		"SuccessfulDelete": {
			settingsClient: &fake.MockALMSettingsAzureClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args:     args{ctx: context.Background(), mg: newTestALMAzure("azure-main", nil)},
			want:     want{deletion: managed.ExternalDelete{}},
			wantCall: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{
				integrationsClient: &fake.MockALMIntegrationsAzureClient{},
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

			alm, ok := tc.args.mg.(*v1alpha1.ALMAzure)
			if ok {
				if cond := alm.GetCondition(xpv1.TypeReady); cond.Reason != xpv1.ReasonDeleting {
					t.Fatalf("Delete() expected deleting condition, got %s", cond.Reason)
				}
			}
		})
	}
}

// TestDisconnect tests disconnecting an ALMAzure resource.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{integrationsClient: &fake.MockALMIntegrationsAzureClient{}, settingsClient: &fake.MockALMSettingsAzureClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// TestGetSavedAPIToken tests retrieving the saved API token
// from an ALMAzure resource.
func TestGetSavedAPIToken(t *testing.T) {
	t.Parallel()

	tokenRef := &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"}

	cases := map[string]struct {
		objects   []runtime.Object
		alm       *v1alpha1.ALMAzure
		wantToken string
		wantError string
	}{
		"NoWriteSecretRef": {
			objects:   nil,
			alm:       newTestALMAzure(testExternalName, tokenRef),
			wantToken: "",
			wantError: "",
		},
		"WriteSecretNameEmpty": {
			objects: nil,
			alm: func() *v1alpha1.ALMAzure {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: ""})

				return alm
			}(),
			wantToken: "",
			wantError: "",
		},
		"MissingSecretReturnsEmptyToken": {
			objects: nil,
			alm: func() *v1alpha1.ALMAzure {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing"})

				return alm
			}(),
			wantToken: "",
			wantError: "",
		},
		"MissingSecretKeyReturnsEmptyToken": {
			objects: []runtime.Object{tokenSecret("connection-secret", "default", "other-key", "value")},
			alm: func() *v1alpha1.ALMAzure {
				alm := newTestALMAzure(testExternalName, tokenRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantToken: "",
			wantError: "",
		},
		"SuccessfulRead": {
			objects: []runtime.Object{tokenSecret("connection-secret", "default", connectionDetailTokenKey, "saved-token")},
			alm: func() *v1alpha1.ALMAzure {
				alm := newTestALMAzure("azure-main", tokenRef)
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

// TestConnect tests connecting to an ALMAzure resource.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TypeAssertionError", func(t *testing.T) {
		t.Parallel()

		c := &connector{}

		_, err := c.Connect(context.Background(), &notALMAzure{})
		if err == nil || !strings.Contains(err.Error(), errNotALMAzure) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errNotALMAzure)
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

		alm := newTestALMAzure("azure-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})

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

		alm := newTestALMAzure("azure-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})
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

		alm := newTestALMAzure("azure-main", &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "pat-secret"}, Key: "token"})
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

// TestSetupGatedRegistersALMAzureGVK tests that SetupGated
// registers the ALMAzure GVK.
func TestSetupGatedRegistersALMAzureGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.ALMAzureGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails tests that the
// SetupGated callback panics on setup failure.
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
