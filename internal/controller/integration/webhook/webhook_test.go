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
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
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
