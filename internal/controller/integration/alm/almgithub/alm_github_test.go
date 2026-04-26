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

package almgithub

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
	testExternalName     = "github-main"
	testRenamedGitHubKey = "github-renamed"
	testGitHubURL        = "https://api.github.com"
	testAppID            = "123456"
	testClientID         = "Iv1.abc123"
	githubCSValue        = "githubCSValue"
	githubWHValue        = "githubWHValue"
	privateKeyValue      = "pk-value"
)

type notALMGitHub struct {
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

func newTestALMGitHub(externalName string, clientSecretRef, privateKeyRef *xpv1.LocalSecretKeySelector) *v1alpha1.ALMGitHub {
	alm := &v1alpha1.ALMGitHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-alm-github",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("alm-github-uid"),
		},
		Spec: v1alpha1.ALMGitHubSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider: v1alpha1.ALMGitHubParameters{
				URL:             testGitHubURL,
				Key:             testExternalName,
				AppID:           testAppID,
				ClientID:        testClientID,
				ClientSecretRef: clientSecretRef,
				PrivateKeyRef:   privateKeyRef,
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(alm, externalName)
	}

	return alm
}

func secretRef(name, key string) *xpv1.LocalSecretKeySelector {
	return &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: name}, Key: key}
}

func testSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

func multiKeySecret(name, namespace string, data map[string]string) *corev1.Secret {
	bdata := make(map[string][]byte, len(data))
	for k, v := range data {
		bdata[k] = []byte(v)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       bdata,
	}
}

//nolint:maintidx // test function covering many Observe paths
func TestObserve(t *testing.T) {
	t.Parallel()

	clientSecretRef := secretRef("client-secret", "clientSecret")
	privateKeyRef := secretRef("private-key", "privateKey")

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
		settingsClient *fake.MockALMSettingsGitHubClient
		args           args
		want           want
	}{
		"NotALMGitHub": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitHub{}},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: errNotALMGitHub},
		},
		"MissingExternalNameReturnsNotExists": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want:           want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"MissingClientSecretRefReturnsError": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, nil, privateKeyRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"EmptyClientSecretReturnsError": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "client secret is empty"},
		},
		"MissingPrivateKeyRefReturnsError": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", githubCSValue)},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, nil)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "cannot get private key from secret reference"},
		},
		"EmptyPrivateKeyReturnsError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", ""),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want:           want{observation: managed.ExternalObservation{}, errSubstr: "private key is empty"},
		},
		"MissingSavedSecretsDoesNotFailObserve": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: testExternalName, URL: testGitHubURL, AppID: testAppID, ClientID: testClientID}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitHubURL

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"DefinitionNotFound": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"SuccessfulObserveUpToDate": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
				multiKeySecret("connection-secret", "default", map[string]string{
					connectionDetailClientSecretKey:  githubCSValue,
					connectionDetailPrivateKeyKey:    privateKeyValue,
					connectionDetailWebhookSecretKey: "",
				}),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: testExternalName, URL: testGitHubURL, AppID: testAppID, ClientID: testClientID}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitHubURL
				alm.Status.AtProvider.AppID = testAppID
				alm.Status.AtProvider.ClientID = testClientID

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveNotUpToDate": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", "new-githubCSValue"),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
				multiKeySecret("connection-secret", "default", map[string]string{
					connectionDetailClientSecretKey: "old-githubCSValue",
					connectionDetailPrivateKeyKey:   privateKeyValue,
				}),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: testExternalName, URL: testGitHubURL, AppID: testAppID, ClientID: testClientID}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitHubURL
				alm.Status.AtProvider.AppID = testAppID
				alm.Status.AtProvider.ClientID = testClientID

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"WebhookSecretErrorReturnsError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("missing-webhook-secret", "webhookSecret")

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{}, errSubstr: "cannot get webhook secret from secret reference"},
		},
		"ListDefinitionsErrorReturnsError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api list failed")
			}},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want: want{observation: managed.ExternalObservation{}, errSubstr: "cannot list ALM settings definitions from SonarQube API"},
		},
		"NoConnectionSecretRefTriggersUpdate": {
			// When writeConnectionSecretToRef is not set (bypassing CRD validation), the
			// controller cannot compare stored vs current secret values and treats the
			// resource as out-of-date so the next Update re-writes the connection secret.
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: testExternalName, URL: testGitHubURL, AppID: testAppID, ClientID: testClientID}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				// No SetWriteConnectionSecretToReference → saved secrets are empty strings.
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitHubURL
				alm.Status.AtProvider.AppID = testAppID
				alm.Status.AtProvider.ClientID = testClientID

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"SuccessfulObserveWithWebhookSecret": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
				testSecret("webhook-secret", "default", "webhookSecret", githubWHValue),
				multiKeySecret("connection-secret", "default", map[string]string{
					connectionDetailClientSecretKey:  githubCSValue,
					connectionDetailPrivateKeyKey:    privateKeyValue,
					connectionDetailWebhookSecretKey: githubWHValue,
				}),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
				return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: testExternalName, URL: testGitHubURL, AppID: testAppID, ClientID: testClientID}}}, mockHTTPResponse(http.StatusOK), nil
			}},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("webhook-secret", "webhookSecret")
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})
				alm.Status.AtProvider.Key = testExternalName
				alm.Status.AtProvider.URL = testGitHubURL
				alm.Status.AtProvider.AppID = testAppID
				alm.Status.AtProvider.ClientID = testClientID

				return alm
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{}}},
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
				kubeClient:     kubeClient,
				settingsClient: tc.settingsClient,
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

	clientSecretRef := secretRef("client-secret", "clientSecret")
	privateKeyRef := secretRef("private-key", "privateKey")

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
		settingsClient *fake.MockALMSettingsGitHubClient
		args           args
		want           want
	}{
		"NotALMGitHub": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitHub{}},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: errNotALMGitHub},
		},
		"MissingClientSecretRef": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", nil, privateKeyRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"EmptyClientSecret": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "client secret is empty"},
		},
		"MissingPrivateKeyRef": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", githubCSValue)},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, nil)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "cannot get private key from secret reference"},
		},
		"EmptyPrivateKey": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", ""),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want:           want{creation: managed.ExternalCreation{}, errSubstr: "private key is empty"},
		},
		"CreateError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				CreateGithubFn: func(_ *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api create failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want: want{creation: managed.ExternalCreation{}, errSubstr: "cannot create ALMGitHub resource in SonarQube API"},
		},
		"WebhookSecretError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub("", clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("missing-webhook-secret", "webhookSecret")

				return alm
			}()},
			want: want{creation: managed.ExternalCreation{}, errSubstr: "cannot get webhook secret from secret reference"},
		},
		"SuccessfulCreate": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				CreateGithubFn: func(opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.URL != testGitHubURL || opt.AppID != testAppID || opt.ClientID != testClientID || opt.ClientSecret != githubCSValue || opt.PrivateKey != privateKeyValue {
						t.Fatalf("Create() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientSecretKey:  []byte(githubCSValue),
				connectionDetailPrivateKeyKey:    []byte(privateKeyValue),
				connectionDetailWebhookSecretKey: []byte(""),
			}}},
		},
		"SuccessfulCreateWithWebhookSecret": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
				testSecret("webhook-secret", "default", "webhookSecret", githubWHValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				CreateGithubFn: func(opt *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
					if opt == nil || opt.WebhookSecret != githubWHValue {
						t.Fatalf("Create() unexpected webhook secret: %+v", opt)
					}

					return mockHTTPResponse(http.StatusCreated), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub("", clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("webhook-secret", "webhookSecret")

				return alm
			}()},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientSecretKey:  []byte(githubCSValue),
				connectionDetailPrivateKeyKey:    []byte(privateKeyValue),
				connectionDetailWebhookSecretKey: []byte(githubWHValue),
			}}},
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
				kubeClient:     kubeClient,
				settingsClient: tc.settingsClient,
			}

			got, err := e.Create(tc.args.ctx, tc.args.mg)
			checkError(t, "Create", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Fatalf("Create() mismatch (-want +got):\n%s", diff)
			}

			alm, ok := tc.args.mg.(*v1alpha1.ALMGitHub)
			if ok && name == "SuccessfulCreate" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != testExternalName {
					t.Fatalf("Create() external name = %q, want %q", gotExternalName, testExternalName)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) { //nolint:maintidx,gocognit // table-driven test covers all update paths
	t.Parallel()

	clientSecretRef := secretRef("client-secret", "clientSecret")
	privateKeyRef := secretRef("private-key", "privateKey")

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
		settingsClient *fake.MockALMSettingsGitHubClient
		args           args
		want           want
		registerMG     bool // pre-register tc.args.mg with the fake kube client (needed for kubeClient.Update after key change)
	}{
		"NotALMGitHub": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitHub{}},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: errNotALMGitHub},
		},
		"MissingExternalName": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", clientSecretRef, privateKeyRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "external name is not set"},
		},
		"MissingClientSecretRef": {
			objects:        []runtime.Object{},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, nil, privateKeyRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "cannot get client secret from secret reference"},
		},
		"MissingPrivateKeyRef": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", githubCSValue)},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, nil)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "cannot get private key from secret reference"},
		},
		"EmptyClientSecret": {
			objects:        []runtime.Object{testSecret("client-secret", "default", "clientSecret", "")},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "client secret is empty"},
		},
		"EmptyPrivateKey": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", ""),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want:           want{update: managed.ExternalUpdate{}, errSubstr: "private key is empty"},
		},
		"WebhookSecretError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("missing-webhook-secret", "webhookSecret")

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot get webhook secret from secret reference"},
		},
		"UpdateError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				UpdateGithubFn: func(_ *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api update failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update ALMGitHub resource in SonarQube API"},
		},
		"SuccessfulUpdateWithoutKeyChange": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				UpdateGithubFn: func(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != "" || opt.URL != testGitHubURL || opt.AppID != testAppID || opt.ClientID != testClientID || opt.ClientSecret != githubCSValue || opt.PrivateKey != privateKeyValue {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientSecretKey:  []byte(githubCSValue),
				connectionDetailPrivateKeyKey:    []byte(privateKeyValue),
				connectionDetailWebhookSecretKey: []byte(""),
			}}},
		},
		"SuccessfulUpdateWithKeyChange": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				UpdateGithubFn: func(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName || opt.NewKey != testRenamedGitHubKey {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.Key = testRenamedGitHubKey

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientSecretKey:  []byte(githubCSValue),
				connectionDetailPrivateKeyKey:    []byte(privateKeyValue),
				connectionDetailWebhookSecretKey: []byte(""),
			}}},
			registerMG: true,
		},
		"UpdateKeyChangePersistError": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				UpdateGithubFn: func(_ *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			// registerMG is false → ALMGitHub not in fake client → kubeClient.Update returns not-found
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.Key = testRenamedGitHubKey

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{}, errSubstr: "cannot update external name annotation after key change"},
		},
		"SuccessfulUpdateWithWebhookSecret": {
			objects: []runtime.Object{
				testSecret("client-secret", "default", "clientSecret", githubCSValue),
				testSecret("private-key", "default", "privateKey", privateKeyValue),
				testSecret("webhook-secret", "default", "webhookSecret", githubWHValue),
			},
			settingsClient: &fake.MockALMSettingsGitHubClient{
				UpdateGithubFn: func(opt *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
					if opt == nil || opt.WebhookSecret != githubWHValue {
						t.Fatalf("Update() unexpected webhook secret: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.Spec.ForProvider.WebhookSecretRef = secretRef("webhook-secret", "webhookSecret")

				return alm
			}()},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientSecretKey:  []byte(githubCSValue),
				connectionDetailPrivateKeyKey:    []byte(privateKeyValue),
				connectionDetailWebhookSecretKey: []byte(githubWHValue),
			}}},
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
				if alm, ok := tc.args.mg.(*v1alpha1.ALMGitHub); ok {
					builder = builder.WithRuntimeObjects(alm)
				}
			}

			kubeClient := builder.Build()
			e := &external{
				kubeClient:     kubeClient,
				settingsClient: tc.settingsClient,
			}

			got, err := e.Update(tc.args.ctx, tc.args.mg)
			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Fatalf("Update() mismatch (-want +got):\n%s", diff)
			}

			if alm, ok := tc.args.mg.(*v1alpha1.ALMGitHub); ok && name == "SuccessfulUpdateWithKeyChange" {
				if gotExternalName := meta.GetExternalName(alm); gotExternalName != testRenamedGitHubKey {
					t.Fatalf("Update() external name = %q, want %q", gotExternalName, testRenamedGitHubKey)
				}

				// Verify the external name was persisted to the Kubernetes API.
				persisted := &v1alpha1.ALMGitHub{}

				getErr := kubeClient.Get(tc.args.ctx, types.NamespacedName{Name: alm.Name, Namespace: alm.Namespace}, persisted)
				if getErr != nil {
					t.Fatalf("Update() kubeClient.Get() error: %v", getErr)
				}

				if gotPersistedName := meta.GetExternalName(persisted); gotPersistedName != testRenamedGitHubKey {
					t.Fatalf("Update() persisted external name = %q, want %q", gotPersistedName, testRenamedGitHubKey)
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
		settingsClient *fake.MockALMSettingsGitHubClient
		args           args
		want           want
	}{
		"NotALMGitHub": {
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: &notALMGitHub{}},
			want:           want{deletion: managed.ExternalDelete{}, errSubstr: errNotALMGitHub},
		},
		"MissingExternalName": {
			settingsClient: &fake.MockALMSettingsGitHubClient{},
			args:           args{ctx: context.Background(), mg: newTestALMGitHub("", nil, nil)},
			want:           want{deletion: managed.ExternalDelete{}},
		},
		"DeleteError": {
			settingsClient: &fake.MockALMSettingsGitHubClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api delete failed")
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, nil, nil)},
			want: want{deletion: managed.ExternalDelete{}, errSubstr: "cannot delete ALMGitHub resource in SonarQube API"},
		},
		"SuccessfulDelete": {
			settingsClient: &fake.MockALMSettingsGitHubClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
					if opt == nil || opt.Key != testExternalName {
						t.Fatalf("Delete() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestALMGitHub(testExternalName, nil, nil)},
			want: want{deletion: managed.ExternalDelete{}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{
				settingsClient: tc.settingsClient,
			}

			got, err := e.Delete(tc.args.ctx, tc.args.mg)
			checkError(t, "Delete", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.deletion, got); diff != "" {
				t.Fatalf("Delete() mismatch (-want +got):\n%s", diff)
			}

			alm, ok := tc.args.mg.(*v1alpha1.ALMGitHub)
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

	e := &external{settingsClient: &fake.MockALMSettingsGitHubClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

func TestGetSavedSecrets(t *testing.T) {
	t.Parallel()

	clientSecretRef := secretRef("client-secret", "clientSecret")
	privateKeyRef := secretRef("private-key", "privateKey")

	cases := map[string]struct {
		objects           []runtime.Object
		alm               *v1alpha1.ALMGitHub
		wantClientSecret  string
		wantPrivateKey    string
		wantWebhookSecret string
		wantError         string
	}{
		"NoWriteSecretRef": {
			objects:           nil,
			alm:               newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef),
			wantClientSecret:  "",
			wantPrivateKey:    "",
			wantWebhookSecret: "",
		},
		"WriteSecretNameEmpty": {
			objects: nil,
			alm: func() *v1alpha1.ALMGitHub {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: ""})

				return alm
			}(),
			wantClientSecret:  "",
			wantPrivateKey:    "",
			wantWebhookSecret: "",
		},
		"MissingSecretReturnsEmpty": {
			objects: nil,
			alm: func() *v1alpha1.ALMGitHub {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing"})

				return alm
			}(),
			wantClientSecret:  "",
			wantPrivateKey:    "",
			wantWebhookSecret: "",
		},
		"MissingKeysReturnEmpty": {
			objects: []runtime.Object{testSecret("connection-secret", "default", "other-key", "value")},
			alm: func() *v1alpha1.ALMGitHub {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantClientSecret:  "",
			wantPrivateKey:    "",
			wantWebhookSecret: "",
		},
		"SuccessfulRead": {
			objects: []runtime.Object{multiKeySecret("connection-secret", "default", map[string]string{
				connectionDetailClientSecretKey:  "saved-cs",
				connectionDetailPrivateKeyKey:    "saved-pk",
				connectionDetailWebhookSecretKey: "saved-wh",
			})},
			alm: func() *v1alpha1.ALMGitHub {
				alm := newTestALMGitHub(testExternalName, clientSecretRef, privateKeyRef)
				alm.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return alm
			}(),
			wantClientSecret:  "saved-cs",
			wantPrivateKey:    "saved-pk",
			wantWebhookSecret: "saved-wh",
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

			gotCS, gotPK, gotWH, err := e.getSavedSecrets(context.Background(), tc.alm)
			checkError(t, "getSavedSecrets", tc.wantError, err)

			if tc.wantError != "" {
				return
			}

			if gotCS != tc.wantClientSecret {
				t.Fatalf("getSavedSecrets() clientSecret = %q, want %q", gotCS, tc.wantClientSecret)
			}

			if gotPK != tc.wantPrivateKey {
				t.Fatalf("getSavedSecrets() privateKey = %q, want %q", gotPK, tc.wantPrivateKey)
			}

			if gotWH != tc.wantWebhookSecret {
				t.Fatalf("getSavedSecrets() webhookSecret = %q, want %q", gotWH, tc.wantWebhookSecret)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TypeAssertionError", func(t *testing.T) {
		t.Parallel()

		c := &connector{}

		_, err := c.Connect(context.Background(), &notALMGitHub{})
		if err == nil || !strings.Contains(err.Error(), errNotALMGitHub) {
			t.Fatalf("Connect() error = %v, want to contain %q", err, errNotALMGitHub)
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

		alm := newTestALMGitHub("github-main", secretRef("client-secret", "clientSecret"), secretRef("private-key", "privateKey"))

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

		alm := newTestALMGitHub("github-main", secretRef("client-secret", "clientSecret"), secretRef("private-key", "privateKey"))
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

		secret := testSecret("provider-secret", "default", "token", "provider-token")

		kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, secret).Build()

		alm := newTestALMGitHub("github-main", secretRef("client-secret", "clientSecret"), secretRef("private-key", "privateKey"))
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

func TestSetupGatedRegistersALMGitHubGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.ALMGitHubGroupVersionKind, g.gvks[0]); diff != "" {
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
