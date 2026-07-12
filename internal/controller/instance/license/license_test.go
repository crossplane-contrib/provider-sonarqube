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

package license

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

const (
	// testLicenseKey is the desired license key referenced by LicenseKeySecretRef.
	testLicenseKey = "test-license-key-abc123"
)

// notLicense is a fake Managed resource that is not a License.
type notLicense struct{ resource.Managed }

// GetObjectKind returns an empty ObjectKind.
func (n *notLicense) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }

// DeepCopyObject returns nil.
func (n *notLicense) DeepCopyObject() runtime.Object { return nil }

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

// secretRef builds a SecretKeySelector for the "default" namespace.
func secretRef(name, key string) *xpv1.SecretKeySelector {
	return &xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Name: name, Namespace: "default"},
		Key:             key,
	}
}

// testSecret creates a test secret with a single key-value pair.
func testSecret(name, namespace, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{key: []byte(value)},
	}
}

// newTestLicense creates a test License resource.
func newTestLicense(externalName string, licenseKeyRef *xpv1.SecretKeySelector) *v1alpha1.License {
	l := &v1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-license",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("test-license-uid"),
		},
		Spec: v1alpha1.LicenseSpec{
			ForProvider: v1alpha1.LicenseParameters{LicenseKeySecretRef: licenseKeyRef},
		},
	}
	if externalName != "" {
		meta.SetExternalName(l, externalName)
	}

	return l
}

// newTestLicenseWithEndpoint creates a test License resource sourcing its
// license key from endpoint.
func newTestLicenseWithEndpoint(externalName string, endpoint *v1alpha1.LicenseEndpoint) *v1alpha1.License {
	l := &v1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-license",
			Namespace:   "default",
			Annotations: map[string]string{},
			UID:         types.UID("test-license-uid"),
		},
		Spec: v1alpha1.LicenseSpec{
			ForProvider: v1alpha1.LicenseParameters{Endpoint: endpoint},
		},
	}
	if externalName != "" {
		meta.SetExternalName(l, externalName)
	}

	return l
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	licenseKeyRef := secretRef("license-key", "license")

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation  managed.ExternalCreation
		errSubstr string
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
			want:    want{creation: managed.ExternalCreation{}, errSubstr: errNotLicense},
		},
		"MissingLicenseSource": {
			objects: []runtime.Object{},
			client:  &fake.MockLicensesClient{},
			args:    args{ctx: context.Background(), mg: newTestLicense("", nil)},
			want:    want{creation: managed.ExternalCreation{}, errSubstr: "licenseKeySecretRef or endpoint must be set"},
		},
		"SetAPIError": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			client: &fake.MockLicensesClient{
				SetFn: func(_ *sonar.LicenseSetOptions) (*http.Response, error) {
					return nil, errors.New("boom")
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense("", licenseKeyRef)},
			want: want{creation: managed.ExternalCreation{}, errSubstr: errSetLicense},
		},
		"Success": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			client: &fake.MockLicensesClient{
				SetFn: func(opt *sonar.LicenseSetOptions) (*http.Response, error) {
					if opt == nil || opt.License != testLicenseKey {
						t.Fatalf("Create() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense("", licenseKeyRef)},
			want: want{creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailLicenseKeyKey: []byte(testLicenseKey),
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
			e := &external{kube: kubeClient, client: tc.client}

			got, err := e.Create(tc.args.ctx, tc.args.mg)
			checkError(t, "Create", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Fatalf("Create() mismatch (-want +got):\n%s", diff)
			}

			if l, ok := tc.args.mg.(*v1alpha1.License); ok && name == "Success" {
				if gotExternalName := meta.GetExternalName(l); gotExternalName != externalLicenseName {
					t.Fatalf("Create() external name = %q, want %q", gotExternalName, externalLicenseName)
				}
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	licenseKeyRef := secretRef("license-key", "license")

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		update    managed.ExternalUpdate
		errSubstr string
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
			want:    want{update: managed.ExternalUpdate{}, errSubstr: errNotLicense},
		},
		"MissingLicenseSource": {
			objects: []runtime.Object{},
			client:  &fake.MockLicensesClient{},
			args:    args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, nil)},
			want:    want{update: managed.ExternalUpdate{}, errSubstr: "licenseKeySecretRef or endpoint must be set"},
		},
		"SetAPIError": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			client: &fake.MockLicensesClient{
				SetFn: func(_ *sonar.LicenseSetOptions) (*http.Response, error) {
					return nil, errors.New("boom")
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, licenseKeyRef)},
			want: want{update: managed.ExternalUpdate{}, errSubstr: errSetLicense},
		},
		"Success": {
			objects: []runtime.Object{testSecret("license-key", "default", "license", testLicenseKey)},
			client: &fake.MockLicensesClient{
				SetFn: func(opt *sonar.LicenseSetOptions) (*http.Response, error) {
					if opt == nil || opt.License != testLicenseKey {
						t.Fatalf("Update() unexpected options: %+v", opt)
					}

					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, licenseKeyRef)},
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailLicenseKeyKey: []byte(testLicenseKey),
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
			e := &external{kube: kubeClient, client: tc.client}

			got, err := e.Update(tc.args.ctx, tc.args.mg)
			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Fatalf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o         managed.ExternalDelete
		errSubstr string
	}

	cases := map[string]struct {
		client *fake.MockLicensesClient
		args   args
		want   want
	}{
		"NotLicense": {
			client: &fake.MockLicensesClient{},
			args:   args{ctx: context.Background(), mg: &notLicense{}},
			want:   want{o: managed.ExternalDelete{}, errSubstr: errNotLicense},
		},
		"UnsetAPIError": {
			client: &fake.MockLicensesClient{
				UnsetLicenseFn: func() (*http.Response, error) {
					return nil, errors.New("boom")
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, nil)},
			want: want{o: managed.ExternalDelete{}, errSubstr: errUnsetLicense},
		},
		"Success": {
			client: &fake.MockLicensesClient{
				UnsetLicenseFn: func() (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{ctx: context.Background(), mg: newTestLicense(externalLicenseName, nil)},
			want: want{o: managed.ExternalDelete{}},
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

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Fatalf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDisconnect verifies Disconnect returns nil.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockLicensesClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v, want nil", err)
	}
}

// newScheme creates a runtime.Scheme with all required types registered.
func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(apisv1alpha1) unexpected error: %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", err)
	}

	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) unexpected error: %v", err)
	}

	return scheme
}

// TestConnectNotLicenseError verifies Connect fails on non-License input.
func TestConnectNotLicenseError(t *testing.T) {
	t.Parallel()

	c := &connector{}
	_, err := c.Connect(context.Background(), &notLicense{})
	checkError(t, "Connect", errNotLicense, err)
}

// TestConnectTrackUsageError verifies Connect returns an error when usage
// tracking fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	lic := newTestLicense(externalLicenseName, nil)

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), lic)
	checkError(t, "Connect", errTrackPCUsage, err)
}

// TestConnectGetConfigError verifies Connect returns an error when the
// ProviderConfig is missing.
func TestConnectGetConfigError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	lic := newTestLicense(externalLicenseName, nil)
	lic.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), lic)
	checkError(t, "Connect", errGetPC, err)
}

// TestConnectSuccess verifies Connect returns a valid ExternalClient on
// success.
func TestConnectSuccess(t *testing.T) {
	t.Parallel()

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

	kubeClient := fakekube.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(providerConfig, secret).
		Build()

	lic := newTestLicense(externalLicenseName, nil)
	lic.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: instance.NewLicensesClient,
	}

	ext, err := c.Connect(context.Background(), lic)
	if err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	if ext == nil {
		t.Fatal("Connect() expected non-nil ExternalClient")
	}
}

// TestSetupGatedRegistersLicenseGVK verifies SetupGated registers the
// License GVK.
func TestSetupGatedRegistersLicenseGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.LicenseGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails tests that the SetupGated
// callback panics when the underlying Setup call fails.
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
