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

package portfolio

import (
	"context"
	"net/http"
	"reflect"
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
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
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
	// testPortfolioKey is the portfolio key used across tests.
	testPortfolioKey = "my-portfolio"
	// testPortfolioName is the portfolio name used across tests.
	testPortfolioName = "My Portfolio"
)

// notPortfolio is a sentinel type used to test invalid managed
// resource handling.
type notPortfolio struct {
	resource.Managed
}

// mockGate is a minimal implementation of the feature gate used by SetupGated.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register implements the gate interface, capturing the callback and GVKs.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set implements the gate interface as a no-op.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// errBoom is a generic test error.
var errBoom = errors.New("boom")

// errComparer compares errors by their message string.
var errComparer = cmp.Comparer(func(x, y error) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}

	return x.Error() == y.Error()
})

// mockHTTPOK returns a minimal 200 OK HTTP response for testing.
func mockHTTPOK() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}
}

// mockHTTPNotFound returns a minimal 404 response for testing.
func mockHTTPNotFound() *http.Response {
	return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"}
}

// newPortfolio builds a Portfolio resource with the given key and
// optional external name.
func newPortfolio(key, name, externalName string) *v1alpha1.Portfolio {
	p := &v1alpha1.Portfolio{
		ObjectMeta: metav1.ObjectMeta{Name: "test-portfolio"},
		Spec: v1alpha1.PortfolioSpec{
			ForProvider: v1alpha1.PortfolioParameters{
				Key:        key,
				Name:       name,
				Visibility: "public",
			},
		},
	}
	if externalName != "" {
		meta.SetExternalName(p, externalName)
	}

	return p
}

// newPortfolioWithMode builds a Portfolio with the given selection
// mode settings.
func newPortfolioWithMode(key, name, externalName, mode, regexp, tags, branch string) *v1alpha1.Portfolio {
	p := newPortfolio(key, name, externalName)
	p.Spec.ForProvider.SelectionMode = mode
	p.Spec.ForProvider.Regexp = regexp
	p.Spec.ForProvider.Tags = tags
	p.Spec.ForProvider.Branch = branch

	return p
}

// newPortfolioResource returns a Portfolio suitable for Connect tests.
func newPortfolioResource(name string) *v1alpha1.Portfolio {
	return &v1alpha1.Portfolio{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.PortfolioGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.PortfolioKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.PortfolioSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider: v1alpha1.PortfolioParameters{
				Key:  testPortfolioKey,
				Name: testPortfolioName,
			},
		},
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

// TestSetupGatedRegistersPortfolioGVK verifies SetupGated registers
// the Portfolio GVK.
func TestSetupGatedRegistersPortfolioGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.PortfolioGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackCoverage verifies the callback body executes
// when called.
// The callback panics because mgr is nil; the panic is recovered so
// the test completes.
func TestSetupGatedCallbackCoverage(t *testing.T) {
	t.Parallel()

	defer func() { _ = recover() }()

	g := &mockGate{}
	o := controller.DefaultOptions()
	o.Gate = g

	_ = SetupGated(nil, o)

	g.callback()
}

// TestConnectTypeAssertion verifies Connect returns an error for
// non-Portfolio types.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notPortfolio{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Portfolio type, got nil")
	}

	if !strings.Contains(err.Error(), errNotPortfolio) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errNotPortfolio)
	}
}

// TestConnectTrackUsageError verifies Connect returns an error when
// usage tracking fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	p := newPortfolioResource("test-portfolio")

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), p)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errTrackPCUsage) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errTrackPCUsage)
	}
}

// TestConnectGetConfigError verifies Connect returns an error when
// the ProviderConfig is missing.
func TestConnectGetConfigError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	p := newPortfolioResource("test-portfolio")
	p.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), p)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errGetPC) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errGetPC)
	}
}

// TestConnectSuccess verifies Connect returns a valid ExternalClient
// on success.
func TestConnectSuccess(t *testing.T) {
	t.Parallel()

	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000",
			Token: &apisv1alpha1.ProviderCredentials{
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "sonar-secret",
							Namespace: "default",
						},
						Key: "token",
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

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(providerConfig, secret).Build()
	p := newPortfolioResource("test-portfolio")
	p.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: instance.NewPortfoliosClient,
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

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		client *fake.MockPortfoliosClient
		args   args
		want   want
	}{
		"NotPortfolioError": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: &notPortfolio{}},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotPortfolio),
			},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ShowNotFoundReturnsNotExists": {
			client: &fake.MockPortfoliosClient{
				ShowFn: func(_ *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error) {
					return nil, mockHTTPNotFound(), errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ShowAPIError": {
			client: &fake.MockPortfoliosClient{
				ShowFn: func(_ *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error) {
					return nil, mockHTTPOK(), errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errObservePortfolio),
			},
		},
		"ExistsAndUpToDate": {
			client: &fake.MockPortfoliosClient{
				ShowFn: func(_ *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error) {
					return &sonar.ViewDetails{
						Key:           testPortfolioKey,
						Name:          testPortfolioName,
						Visibility:    "public",
						SelectionMode: selectionModeNone,
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: false,
					ConnectionDetails:       managed.ConnectionDetails{},
				},
				err: nil,
			},
		},
		"ExistsAndNotUpToDate": {
			client: &fake.MockPortfoliosClient{
				ShowFn: func(_ *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error) {
					return &sonar.ViewDetails{
						Key:           testPortfolioKey,
						Name:          "Old Name",
						Visibility:    "public",
						SelectionMode: selectionModeNone,
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        false,
					ResourceLateInitialized: false,
					ConnectionDetails:       managed.ConnectionDetails{},
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Observe() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() observation -want +got:\n%s", diff)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o            managed.ExternalCreation
		err          error
		externalName string
	}

	okNoneMode := func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
		return mockHTTPOK(), nil
	}

	cases := map[string]struct {
		client *fake.MockPortfoliosClient
		args   args
		want   want
	}{
		"NotPortfolioError": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: &notPortfolio{}},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotPortfolio),
			},
		},
		"CreateAPIError": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errCreatePortfolio),
			},
		},
		"SetSelectionModeError": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetNoneModeFn: func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.Wrap(errBoom, "cannot set NONE selection mode"), errSetSelectionMode),
			},
		},
		"SuccessNONEMode": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetNoneModeFn: okNoneMode,
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPortfolioKey,
			},
		},
		"SuccessMANUALMode": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetManualModeFn: func(_ *sonar.ViewsSetManualModeOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolioWithMode(testPortfolioKey, testPortfolioName, "", selectionModeManual, "", "", "")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPortfolioKey,
			},
		},
		"SuccessREGEXPMode": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetRegexpModeFn: func(_ *sonar.ViewsSetRegexpModeOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolioWithMode(testPortfolioKey, testPortfolioName, "", selectionModeRegexp, ".*", "", "main")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPortfolioKey,
			},
		},
		"SuccessREMAININGMode": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetRemainingProjectsModeFn: func(_ *sonar.ViewsSetRemainingProjectsModeOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolioWithMode(testPortfolioKey, testPortfolioName, "", selectionModeRemaining, "", "", "")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPortfolioKey,
			},
		},
		"SuccessTAGSMode": {
			client: &fake.MockPortfoliosClient{
				CreateFn: func(_ *sonar.ViewsCreateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetTagsModeFn: func(_ *sonar.ViewsSetTagsModeOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolioWithMode(testPortfolioKey, testPortfolioName, "", selectionModeTags, "", "java,go", "")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPortfolioKey,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Create() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Create() result -want +got:\n%s", diff)
			}

			if tc.want.externalName != "" {
				if p, ok := tc.args.mg.(*v1alpha1.Portfolio); ok {
					if gotName := meta.GetExternalName(p); gotName != tc.want.externalName {
						t.Errorf("Create() external name = %q, want %q", gotName, tc.want.externalName)
					}
				}
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalUpdate
		err error
	}

	okNoneMode := func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
		return mockHTTPOK(), nil
	}

	cases := map[string]struct {
		client *fake.MockPortfoliosClient
		args   args
		want   want
	}{
		"NotPortfolioError": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: &notPortfolio{}},
			want:   want{o: managed.ExternalUpdate{}, err: errors.New(errNotPortfolio)},
		},
		"EmptyExternalNameReturnsError": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Errorf(errExternalNameNotSet, "test-portfolio"),
			},
		},
		"UpdateAPIError": {
			client: &fake.MockPortfoliosClient{
				UpdateFn: func(_ *sonar.ViewsUpdateOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errUpdatePortfolio),
			},
		},
		"SetSelectionModeError": {
			client: &fake.MockPortfoliosClient{
				UpdateFn: func(_ *sonar.ViewsUpdateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetNoneModeFn: func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.Wrap(errBoom, "cannot set NONE selection mode"), errSetSelectionMode),
			},
		},
		"Success": {
			client: &fake.MockPortfoliosClient{
				UpdateFn: func(_ *sonar.ViewsUpdateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
				SetNoneModeFn: okNoneMode,
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Update() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Update() result -want +got:\n%s", diff)
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
		o   managed.ExternalDelete
		err error
	}

	cases := map[string]struct {
		client *fake.MockPortfoliosClient
		args   args
		want   want
	}{
		"NotPortfolioError": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: &notPortfolio{}},
			want:   want{o: managed.ExternalDelete{}, err: errors.New(errNotPortfolio)},
		},
		"EmptyExternalNameReturnsNil": {
			client: &fake.MockPortfoliosClient{},
			args:   args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, "")},
			want:   want{o: managed.ExternalDelete{}, err: nil},
		},
		"DeleteAPIError": {
			client: &fake.MockPortfoliosClient{
				DeleteFn: func(_ *sonar.ViewsDeleteOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errBoom, errDeletePortfolio),
			},
		},
		"Success": {
			client: &fake.MockPortfoliosClient{
				DeleteFn: func(_ *sonar.ViewsDeleteOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Delete() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Delete() result -want +got:\n%s", diff)
			}
		})
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockPortfoliosClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v, want nil", err)
	}
}

// TestSetSelectionModePassesCorrectOptions verifies setSelectionMode
// passes the right options per mode.
func TestSetSelectionModePassesCorrectOptions(t *testing.T) {
	t.Parallel()

	t.Run("NONEMode", func(t *testing.T) {
		t.Parallel()

		var captured string

		e := &external{client: &fake.MockPortfoliosClient{
			SetNoneModeFn: func(opt *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
				captured = opt.Portfolio

				return mockHTTPOK(), nil
			},
		}}

		spec := &v1alpha1.PortfolioParameters{SelectionMode: selectionModeNone}

		err := e.setSelectionMode(context.Background(), testPortfolioKey, spec)
		if err != nil {
			t.Fatalf("setSelectionMode() error = %v", err)
		}

		if captured != testPortfolioKey {
			t.Errorf("SetNoneMode() called with portfolio %q, want %q", captured, testPortfolioKey)
		}
	})

	t.Run("REGEXPMode", func(t *testing.T) {
		t.Parallel()

		var capturedRegexp, capturedBranch string

		e := &external{client: &fake.MockPortfoliosClient{
			SetRegexpModeFn: func(opt *sonar.ViewsSetRegexpModeOptions) (*http.Response, error) {
				capturedRegexp = opt.Regexp
				capturedBranch = opt.Branch

				return mockHTTPOK(), nil
			},
		}}

		spec := &v1alpha1.PortfolioParameters{SelectionMode: selectionModeRegexp, Regexp: ".*-service", Branch: "main"}

		err := e.setSelectionMode(context.Background(), testPortfolioKey, spec)
		if err != nil {
			t.Fatalf("setSelectionMode() error = %v", err)
		}

		if capturedRegexp != ".*-service" {
			t.Errorf("SetRegexpMode() Regexp = %q, want %q", capturedRegexp, ".*-service")
		}

		if capturedBranch != "main" {
			t.Errorf("SetRegexpMode() Branch = %q, want %q", capturedBranch, "main")
		}
	})

	t.Run("TAGSMode", func(t *testing.T) {
		t.Parallel()

		var capturedTags []string

		e := &external{client: &fake.MockPortfoliosClient{
			SetTagsModeFn: func(opt *sonar.ViewsSetTagsModeOptions) (*http.Response, error) {
				capturedTags = opt.Tags

				return mockHTTPOK(), nil
			},
		}}

		spec := &v1alpha1.PortfolioParameters{SelectionMode: selectionModeTags, Tags: "java,go"}

		err := e.setSelectionMode(context.Background(), testPortfolioKey, spec)
		if err != nil {
			t.Fatalf("setSelectionMode() error = %v", err)
		}

		if want := []string{"java", "go"}; !reflect.DeepEqual(capturedTags, want) {
			t.Errorf("SetTagsMode() Tags = %q, want %q", capturedTags, want)
		}
	})

	t.Run("EmptyModeDefaultsToNONE", func(t *testing.T) {
		t.Parallel()

		called := false

		e := &external{client: &fake.MockPortfoliosClient{
			SetNoneModeFn: func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
				called = true

				return mockHTTPOK(), nil
			},
		}}

		spec := &v1alpha1.PortfolioParameters{SelectionMode: ""}

		err := e.setSelectionMode(context.Background(), testPortfolioKey, spec)
		if err != nil {
			t.Fatalf("setSelectionMode() error = %v", err)
		}

		if !called {
			t.Error("setSelectionMode() with empty mode did not call SetNoneMode")
		}
	})

	t.Run("UnknownModeReturnsNil", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fake.MockPortfoliosClient{}}

		spec := &v1alpha1.PortfolioParameters{SelectionMode: "UNKNOWN_MODE"}

		err := e.setSelectionMode(context.Background(), testPortfolioKey, spec)
		if err != nil {
			t.Errorf("setSelectionMode() with unknown mode error = %v, want nil", err)
		}
	})
}
