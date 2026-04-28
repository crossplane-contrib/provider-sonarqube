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

package webhook

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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

const (
	// testExternalName is the external name used in tests.
	testExternalName = "webhook-key-abc123"
	// testWebhookName is the webhook display name used in tests.
	testWebhookName = "my-webhook"
	// testWebhookURL is the webhook target URL used in tests.
	testWebhookURL = "https://ci.example.com/hook"
	// testSecretValue is the HMAC secret value used in tests.
	testSecretValue = "supersecrethmacvalue123"
)

// notAWebhook is a test type that is not a Webhook.
type notAWebhook struct{ resource.Managed }

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

// Set implements the gate interface (no-op for tests).
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// mockHTTPResponse creates a mock HTTP response with the given status code.
func mockHTTPResponse(statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode, Body: http.NoBody}
}

// webhookSecretRef creates a test LocalSecretKeySelector.
func webhookSecretRef(name, key string) *xpv1.LocalSecretKeySelector {
	return &xpv1.LocalSecretKeySelector{
		LocalSecretReference: xpv1.LocalSecretReference{Name: name},
		Key:                  key,
	}
}

// testKubeSecret creates a Kubernetes Secret with a single key-value pair.
func testKubeSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

// newTestWebhook creates a Webhook resource for use in tests.
func newTestWebhook(externalName string, secretKeyRef *xpv1.LocalSecretKeySelector) *v1alpha1.Webhook {
	webhookResource := &v1alpha1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-webhook",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("webhook-uid"),
		},
		Spec: v1alpha1.WebhookSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider: v1alpha1.WebhookParameters{
				Name:      testWebhookName,
				URL:       testWebhookURL,
				SecretRef: secretKeyRef,
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(webhookResource, externalName)
	}

	return webhookResource
}

// buildKubeClient creates a fake Kubernetes client with the given objects.
func buildKubeClient(objects []runtime.Object) *fakekube.ClientBuilder {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	builder := fakekube.NewClientBuilder().WithScheme(scheme)
	builder = builder.WithRuntimeObjects(objects...)

	return builder
}

// checkError verifies that an error matches (or does not match) expectations.
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
		t.Errorf("%s() error = %q, want substring %q", method, gotErr.Error(), wantErrSubstr)
	}
}

// TestObserve tests observing Webhook resource state.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		objects       []runtime.Object
		webhookClient *fake.MockWebhooksClient
		args          args
		want          want
	}{
		"NotAWebhook": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: &notAWebhook{}},
			want:          want{observation: managed.ExternalObservation{}, errSubstr: errNotWebhook},
		},
		"MissingExternalNameReturnsNotExists": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want:          want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"ListErrorReturnsError": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{observation: managed.ExternalObservation{}, errSubstr: errListWebhooks},
		},
		"WebhookNotInListReturnsNotExists": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"WebhookExistsAndUpToDate": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"WebhookExistsNameChanged": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: "old-name", URL: testWebhookURL},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"SecretRefSetButNoSecretOnSonar": {
			objects: []runtime.Object{testKubeSecret("hook-secret", "default", "value", testSecretValue)},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL, HasSecret: false},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, webhookSecretRef("hook-secret", "value"))},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"ReadSecretErrorReturnsError": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, webhookSecretRef("missing-secret", "value"))},
			want: want{observation: managed.ExternalObservation{}, errSubstr: errGetSecret},
		},
		"SavedSecretMissingTreatedAsEmpty": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL, HasSecret: false},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				wh := newTestWebhook(testExternalName, nil)
				wh.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-conn-secret"})

				return wh
			}()},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"SavedSecretMatchesCurrentSecret": {
			objects: []runtime.Object{
				testKubeSecret("hook-secret", "default", "value", testSecretValue),
				testKubeSecret("conn-secret", "default", connectionDetailSecretKey, testSecretValue),
			},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL, HasSecret: true},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				wh := newTestWebhook(testExternalName, webhookSecretRef("hook-secret", "value"))
				wh.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "conn-secret"})

				return wh
			}()},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"SecretRotatedNotUpToDate": {
			objects: []runtime.Object{
				testKubeSecret("hook-secret", "default", "value", "new-secret"),
				testKubeSecret("conn-secret", "default", connectionDetailSecretKey, "old-secret"),
			},
			webhookClient: &fake.MockWebhooksClient{
				ListFn: func(_ *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
					return &sonar.WebhooksList{Webhooks: []sonar.Webhook{
						{Key: testExternalName, Name: testWebhookName, URL: testWebhookURL, HasSecret: true},
					}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				wh := newTestWebhook(testExternalName, webhookSecretRef("hook-secret", "value"))
				wh.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "conn-secret"})

				return wh
			}()},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := buildKubeClient(tc.objects).Build()
			externalClient := &external{kubeClient: kube, client: tc.webhookClient}

			got, err := externalClient.Observe(tc.args.ctx, tc.args.mg)

			checkError(t, "Observe", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if got.ResourceExists != tc.want.observation.ResourceExists {
				t.Errorf("Observe().ResourceExists = %v, want %v", got.ResourceExists, tc.want.observation.ResourceExists)
			}

			if got.ResourceUpToDate != tc.want.observation.ResourceUpToDate {
				t.Errorf("Observe().ResourceUpToDate = %v, want %v", got.ResourceUpToDate, tc.want.observation.ResourceUpToDate)
			}
		})
	}
}

// TestCreate tests creating Webhook resources.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		errSubstr string
	}

	cases := map[string]struct {
		objects       []runtime.Object
		webhookClient *fake.MockWebhooksClient
		args          args
		want          want
	}{
		"NotAWebhook": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: &notAWebhook{}},
			want:          want{errSubstr: errNotWebhook},
		},
		"MissingSecretReturnsError": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: newTestWebhook("", webhookSecretRef("missing-secret", "value"))},
			want:          want{errSubstr: errGetSecret},
		},
		"CreateAPIErrorReturnsError": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				CreateFn: func(_ *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want: want{errSubstr: errCreateWebhook},
		},
		"SuccessfulCreate": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				CreateFn: func(_ *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
					return &sonar.WebhooksCreate{Webhook: sonar.Webhook{Key: testExternalName}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want: want{},
		},
		"SuccessfulCreateWithSecret": {
			objects: []runtime.Object{testKubeSecret("hook-secret", "default", "value", testSecretValue)},
			webhookClient: &fake.MockWebhooksClient{
				CreateFn: func(opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
					if opt.Secret != testSecretValue {
						return nil, nil, errors.Errorf("unexpected secret: got %q want %q", opt.Secret, testSecretValue)
					}

					return &sonar.WebhooksCreate{Webhook: sonar.Webhook{Key: testExternalName}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook("", webhookSecretRef("hook-secret", "value"))},
			want: want{},
		},
		"CreateAPIReturnsEmptyWebhookKey": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				CreateFn: func(_ *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
					return &sonar.WebhooksCreate{}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want: want{errSubstr: "API response missing webhook key"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := buildKubeClient(tc.objects).Build()
			externalClient := &external{kubeClient: kube, client: tc.webhookClient}

			_, err := externalClient.Create(tc.args.ctx, tc.args.mg)

			checkError(t, "Create", tc.want.errSubstr, err)
		})
	}
}

// TestUpdate tests updating Webhook resources.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		errSubstr string
	}

	cases := map[string]struct {
		objects       []runtime.Object
		webhookClient *fake.MockWebhooksClient
		args          args
		want          want
	}{
		"NotAWebhook": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: &notAWebhook{}},
			want:          want{errSubstr: errNotWebhook},
		},
		"MissingExternalNameReturnsError": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want:          want{errSubstr: "external name is not set"},
		},
		"UpdateAPIErrorReturnsError": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				UpdateFn: func(_ *sonar.WebhooksUpdateOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{errSubstr: errUpdateWebhook},
		},
		"SuccessfulUpdate": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				UpdateFn: func(_ *sonar.WebhooksUpdateOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{},
		},
		"UpdateReadSecretErrorReturnsError": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: newTestWebhook(testExternalName, webhookSecretRef("missing-secret", "value"))},
			want:          want{errSubstr: errGetSecret},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := buildKubeClient(tc.objects).Build()
			externalClient := &external{kubeClient: kube, client: tc.webhookClient}

			_, err := externalClient.Update(tc.args.ctx, tc.args.mg)

			checkError(t, "Update", tc.want.errSubstr, err)
		})
	}
}

// TestDelete tests deleting Webhook resources.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		errSubstr string
	}

	cases := map[string]struct {
		objects       []runtime.Object
		webhookClient *fake.MockWebhooksClient
		args          args
		want          want
	}{
		"NotAWebhook": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: &notAWebhook{}},
			want:          want{errSubstr: errNotWebhook},
		},
		"MissingExternalNameIsNoop": {
			objects:       []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{},
			args:          args{ctx: context.Background(), mg: newTestWebhook("", nil)},
			want:          want{},
		},
		"DeleteAPIErrorReturnsError": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				DeleteFn: func(_ *sonar.WebhooksDeleteOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{errSubstr: errDeleteWebhook},
		},
		"SuccessfulDelete": {
			objects: []runtime.Object{},
			webhookClient: &fake.MockWebhooksClient{
				DeleteFn: func(_ *sonar.WebhooksDeleteOptions) (*http.Response, error) {
					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestWebhook(testExternalName, nil)},
			want: want{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := buildKubeClient(tc.objects).Build()
			externalClient := &external{kubeClient: kube, client: tc.webhookClient}

			_, err := externalClient.Delete(tc.args.ctx, tc.args.mg)

			checkError(t, "Delete", tc.want.errSubstr, err)
		})
	}
}

// TestDisconnect verifies that Disconnect is a no-op and returns no error.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnect tests the connector's Connect method.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("NotAWebhook", func(t *testing.T) {
		t.Parallel()

		c := &connector{}
		_, err := c.Connect(context.Background(), &notAWebhook{})

		checkError(t, "Connect", errNotWebhook, err)
	})

	t.Run("TrackUsageError", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()

		addErr := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(apisv1alpha1) unexpected error: %v", addErr)
		}

		addErr = v1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", addErr)
		}

		kube := fakekube.NewClientBuilder().WithScheme(scheme).Build()
		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err := c.Connect(context.Background(), newTestWebhook("", nil))

		checkError(t, "Connect", errTrackPCUsage, err)
	})

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()

		addErr := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(apisv1alpha1) unexpected error: %v", addErr)
		}

		addErr = v1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", addErr)
		}

		kube := fakekube.NewClientBuilder().WithScheme(scheme).Build()

		wh := newTestWebhook("", nil)
		wh.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err := c.Connect(context.Background(), wh)

		checkError(t, "Connect", errGetPC, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()

		addErr := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(apisv1alpha1) unexpected error: %v", addErr)
		}

		addErr = v1alpha1.SchemeBuilder.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", addErr)
		}

		addErr = corev1.AddToScheme(scheme)
		if addErr != nil {
			t.Fatalf("AddToScheme(corev1) unexpected error: %v", addErr)
		}

		providerConfig := &apisv1alpha1.ProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
			Spec: apisv1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &apisv1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{Name: "prov-secret", Namespace: "default"},
							Key:             "token",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
		}

		providerSecret := testKubeSecret("prov-secret", "default", "token", "test-token")

		kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, providerSecret).Build()

		wh := newTestWebhook("", nil)
		wh.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(cfg common.Config) integration.WebhooksClient {
				return &fake.MockWebhooksClient{}
			},
		}

		got, err := c.Connect(context.Background(), wh)
		if err != nil {
			t.Fatalf("Connect() unexpected error: %v", err)
		}

		if got == nil {
			t.Fatal("Connect() returned nil client")
		}
	})
}

// TestSetupGatedRegistersWebhookGVK verifies that SetupGated registers the
// Webhook GVK with the gate and returns nil.
func TestSetupGatedRegistersWebhookGVK(t *testing.T) {
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

	if g.gvks[0] != v1alpha1.WebhookGroupVersionKind {
		t.Fatalf("SetupGated() GVK = %v, want %v", g.gvks[0], v1alpha1.WebhookGroupVersionKind)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails verifies that the registered
// callback panics when Setup fails (e.g. nil manager).
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
