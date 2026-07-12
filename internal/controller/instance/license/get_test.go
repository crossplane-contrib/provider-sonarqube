// Copyright 2026 The Crossplane Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package license

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	licenseKeyRef := secretRef("license-key", "license")

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		objects []runtime.Object
		client  *fake.MockLicensesClient
		args    args
		want    want
	}{
		"NotLicense": {
			objects: []runtime.Object{},
			client:  &fake.MockLicensesClient{},
			args:    args{ctx: context.Background(), mg: &notLicense{}},
			want:    want{observation: managed.ExternalObservation{}, errSubstr: errNotLicense},
		},
		"GetAPIError": {
			objects: []runtime.Object{},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return nil, nil, errors.New("boom")
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, licenseKeyRef)},
			want: want{observation: managed.ExternalObservation{}, errSubstr: errGetLicense},
		},
		"NoLicenseApplied": {
			objects: []runtime.Object{},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return &sonar.LicenseGet{}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense("", licenseKeyRef)},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"MissingLicenseSourceReturnsError": {
			objects: []runtime.Object{},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return &sonar.LicenseGet{License: sonar.License{ProductEdition: "enterprise"}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, nil)},
			want: want{observation: managed.ExternalObservation{}, errSubstr: "licenseKeySecretRef or endpoint must be set"},
		},
		"MissingSavedLicenseKeyDoesNotFailObserve": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return &sonar.LicenseGet{License: sonar.License{ProductEdition: "enterprise"}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				l := newTestLicense(externalLicenseName, licenseKeyRef)
				l.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "missing-connection-secret"})

				return l
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false}},
		},
		"UpToDate": {
			objects: []runtime.Object{
				testSecret("license-key", "default", "license", testLicenseKey),
				testSecret("connection-secret", "default", connectionDetailLicenseKeyKey, testLicenseKey),
			},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return &sonar.LicenseGet{License: sonar.License{ProductEdition: "enterprise"}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				l := newTestLicense(externalLicenseName, licenseKeyRef)
				l.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return l
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false}},
		},
		"KeyChanged": {
			objects: []runtime.Object{
				testSecret("license-key", "default", "license", "new-"+testLicenseKey),
				testSecret("connection-secret", "default", connectionDetailLicenseKeyKey, testLicenseKey),
			},
			client: &fake.MockLicensesClient{
				GetFn: func() (*sonar.LicenseGet, *http.Response, error) {
					return &sonar.LicenseGet{License: sonar.License{ProductEdition: "enterprise"}}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: func() resource.Managed {
				l := newTestLicense(externalLicenseName, licenseKeyRef)
				l.SetWriteConnectionSecretToReference(&xpv1.LocalSecretReference{Name: "connection-secret"})

				return l
			}()},
			want: want{observation: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: false}},
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
			e := &external{kube: kubeClient, client: tc.client}

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

// endpointServer starts a test HTTP server that checks the received request
// with check (if non-nil), then responds with statusCode and body.
func endpointServer(t *testing.T, statusCode int, body string, check func(*testing.T, *http.Request)) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(t, r)
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// TestResolveLicenseKey tests all branches of resolveLicenseKey, including
// license keys sourced from a secret, an unauthenticated endpoint, and an
// endpoint using HTTP Basic or bearer-token authentication.
func TestResolveLicenseKey(t *testing.T) {
	t.Parallel()

	licenseKeyRef := secretRef("license-key", "license")

	type want struct {
		licenseKey string
		errSubstr  string
	}

	cases := map[string]struct {
		objects []runtime.Object
		license *v1alpha1.License
		want    want
	}{
		"SecretRef": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			license: newTestLicense(externalLicenseName, licenseKeyRef),
			want:    want{licenseKey: testLicenseKey},
		},
		"SecretRefMissing": {
			objects: []runtime.Object{},
			license: newTestLicense(externalLicenseName, licenseKeyRef),
			want:    want{errSubstr: "cannot get license key from secret"},
		},
		"NeitherSourceSet": {
			objects: []runtime.Object{},
			license: newTestLicense(externalLicenseName, nil),
			want:    want{errSubstr: "licenseKeySecretRef or endpoint must be set"},
		},
		"EndpointUnauthenticated": {
			objects: []runtime.Object{},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL: endpointServer(t, http.StatusOK, "  "+testLicenseKey+"  ", func(t *testing.T, r *http.Request) {
					t.Helper()

					if got := r.Header.Get("Authorization"); got != "" {
						t.Errorf("Authorization header = %q, want empty", got)
					}
				}).URL,
			}),
			want: want{licenseKey: testLicenseKey},
		},
		"EndpointBasicAuth": {
			objects: []runtime.Object{testSecret("endpoint-password", "default", "password", "s3cret")},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL: endpointServer(t, http.StatusOK, testLicenseKey, func(t *testing.T, r *http.Request) {
					t.Helper()

					user, pass, ok := r.BasicAuth()
					if !ok || user != "alice" || pass != "s3cret" {
						t.Errorf("BasicAuth() = (%q, %q, %v), want (\"alice\", \"s3cret\", true)", user, pass, ok)
					}
				}).URL,
				BasicAuthUsername:          new("alice"),
				BasicAuthPasswordSecretRef: secretRef("endpoint-password", "password"),
			}),
			want: want{licenseKey: testLicenseKey},
		},
		"EndpointBasicAuthPasswordSecretMissing": {
			objects: []runtime.Object{},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL:                        "http://example.invalid",
				BasicAuthUsername:          new("alice"),
				BasicAuthPasswordSecretRef: secretRef("endpoint-password", "password"),
			}),
			want: want{errSubstr: "cannot get basic auth password from secret"},
		},
		"EndpointBearerToken": {
			objects: []runtime.Object{testSecret("endpoint-token", "default", "token", "tok123")},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL: endpointServer(t, http.StatusOK, testLicenseKey, func(t *testing.T, r *http.Request) {
					t.Helper()

					if got := r.Header.Get("Authorization"); got != "Bearer tok123" {
						t.Errorf("Authorization header = %q, want %q", got, "Bearer tok123")
					}
				}).URL,
				BearerTokenSecretRef: secretRef("endpoint-token", "token"),
			}),
			want: want{licenseKey: testLicenseKey},
		},
		"EndpointBearerTokenSecretMissing": {
			objects: []runtime.Object{},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL:                  "http://example.invalid",
				BearerTokenSecretRef: secretRef("endpoint-token", "token"),
			}),
			want: want{errSubstr: "cannot get bearer token from secret"},
		},
		"EndpointFetchErrorNoSavedKeyReturnsError": {
			objects: []runtime.Object{},
			license: newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
				URL: endpointServer(t, http.StatusInternalServerError, "", nil).URL,
			}),
			want: want{errSubstr: "cannot get license key from endpoint"},
		},
		"EndpointFetchErrorFallsBackToSavedKey": {
			objects: []runtime.Object{testSecret("example-license-conn", "default", connectionDetailLicenseKeyKey, testLicenseKey)},
			license: func() *v1alpha1.License {
				l := newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
					URL: endpointServer(t, http.StatusInternalServerError, "", nil).URL,
				})
				l.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{Name: "example-license-conn"}

				return l
			}(),
			want: want{licenseKey: testLicenseKey},
		},
		"EndpointFetchErrorAndSavedKeySecretMissingReturnsEndpointError": {
			objects: []runtime.Object{},
			license: func() *v1alpha1.License {
				l := newTestLicenseWithEndpoint(externalLicenseName, &v1alpha1.LicenseEndpoint{
					URL: endpointServer(t, http.StatusInternalServerError, "", nil).URL,
				})
				l.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{Name: "missing-conn-secret"}

				return l
			}(),
			want: want{errSubstr: "cannot get license key from endpoint"},
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
			e := &external{kube: kubeClient}

			got, err := e.resolveLicenseKey(context.Background(), tc.license)
			checkError(t, "resolveLicenseKey", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if got == nil || *got != tc.want.licenseKey {
				t.Fatalf("resolveLicenseKey() = %v, want %q", got, tc.want.licenseKey)
			}
		})
	}
}
