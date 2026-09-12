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

package newcodeperiod

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
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

const (
	// testUpdatedAt is a Unix epoch millisecond timestamp used in observations.
	testUpdatedAt int64 = 1700000000000
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// notNewCodePeriod is a type for testing non-NewCodePeriod resources.
type notNewCodePeriod struct {
	resource.Managed
}

// mockHTTPResponse creates a minimal mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
}

// errComparer compares error messages for testing.
func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Error() == b.Error()
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o          managed.ExternalObservation
		err        error
		atProvider *v1alpha1.NewCodePeriodObservation
	}

	cases := map[string]struct {
		client *fake.MockNewCodePeriodsClient
		args   args
		want   want
	}{
		"NotNewCodePeriodError": {
			client: &fake.MockNewCodePeriodsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notNewCodePeriod{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotNewCodePeriod),
			},
		},
		"DeletingReturnsNotExists": {
			client: &fake.MockNewCodePeriodsClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-new-code-period",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ShowCallFails": {
			client: &fake.MockNewCodePeriodsClient{
				ShowFn: func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return nil, nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-new-code-period",
					},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errors.New("api error"), "failed to get instance new code period"),
			},
		},
		"SuccessfulObserveUpToDate": {
			client: &fake.MockNewCodePeriodsClient{
				ShowFn: func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return &sonar.NewCodePeriodsShow{
						Type:      "NUMBER_OF_DAYS",
						Value:     "30",
						Inherited: false,
						UpdatedAt: testUpdatedAt,
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-new-code-period",
					},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
				atProvider: &v1alpha1.NewCodePeriodObservation{
					Type:      "NUMBER_OF_DAYS",
					Value:     "30",
					Inherited: false,
					UpdatedAt: testUpdatedAt,
				},
			},
		},
		"SuccessfulObserveOutOfDate": {
			client: &fake.MockNewCodePeriodsClient{
				ShowFn: func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return &sonar.NewCodePeriodsShow{
						Type:      "NUMBER_OF_DAYS",
						Value:     "60",
						Inherited: false,
						UpdatedAt: testUpdatedAt,
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-new-code-period",
					},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				err: nil,
				atProvider: &v1alpha1.NewCodePeriodObservation{
					Type:      "NUMBER_OF_DAYS",
					Value:     "60",
					Inherited: false,
					UpdatedAt: testUpdatedAt,
				},
			},
		},
		"SuccessfulObservePreviousVersionNilValue": {
			client: &fake.MockNewCodePeriodsClient{
				ShowFn: func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return &sonar.NewCodePeriodsShow{
						Type:      "PREVIOUS_VERSION",
						Value:     "",
						Inherited: false,
						UpdatedAt: testUpdatedAt,
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-new-code-period",
					},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type: "PREVIOUS_VERSION",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
				atProvider: &v1alpha1.NewCodePeriodObservation{
					Type:      "PREVIOUS_VERSION",
					Value:     "",
					Inherited: false,
					UpdatedAt: testUpdatedAt,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{newCodePeriodsClient: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Observe() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}

			if tc.want.atProvider != nil {
				ncp, ok := tc.args.mg.(*v1alpha1.NewCodePeriod)
				if !ok {
					t.Fatal("Observe() expected a NewCodePeriod managed resource")
				}

				if diff := cmp.Diff(*tc.want.atProvider, ncp.Status.AtProvider); diff != "" {
					t.Errorf("Observe() AtProvider mismatch (-want +got):\n%s", diff)
				}
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
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		client *fake.MockNewCodePeriodsClient
		args   args
		want   want
	}{
		"NotNewCodePeriodError": {
			client: &fake.MockNewCodePeriodsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notNewCodePeriod{},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotNewCodePeriod),
			},
		},
		"SetFails": {
			client: &fake.MockNewCodePeriodsClient{
				SetFn: func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
					return nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("api error"), "failed to set instance new code period"),
			},
		},
		"SuccessfulCreate": {
			client: &fake.MockNewCodePeriodsClient{
				SetFn: func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
					if opt.Project != "" {
						return nil, errors.New("expected project to be empty")
					}

					if opt.Branch != "" {
						return nil, errors.New("expected branch to be empty")
					}

					if opt.Type != "NUMBER_OF_DAYS" {
						return nil, errors.New("unexpected type: " + opt.Type)
					}

					if opt.Value != "30" {
						return nil, errors.New("unexpected value: " + opt.Value)
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{newCodePeriodsClient: tc.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Create() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
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

	cases := map[string]struct {
		client *fake.MockNewCodePeriodsClient
		args   args
		want   want
	}{
		"NotNewCodePeriodError": {
			client: &fake.MockNewCodePeriodsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notNewCodePeriod{},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errNotNewCodePeriod),
			},
		},
		"SetFails": {
			client: &fake.MockNewCodePeriodsClient{
				SetFn: func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
					return nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.New("api error"), "failed to update instance new code period"),
			},
		},
		"SuccessfulUpdate": {
			client: &fake.MockNewCodePeriodsClient{
				SetFn: func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
					if opt.Project != "" {
						return nil, errors.New("expected project to be empty")
					}

					if opt.Branch != "" {
						return nil, errors.New("expected branch to be empty")
					}

					if opt.Type != "NUMBER_OF_DAYS" {
						return nil, errors.New("unexpected type: " + opt.Type)
					}

					if opt.Value != "30" {
						return nil, errors.New("unexpected value: " + opt.Value)
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{newCodePeriodsClient: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Update() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
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
		client *fake.MockNewCodePeriodsClient
		args   args
		want   want
	}{
		"NotNewCodePeriodError": {
			client: &fake.MockNewCodePeriodsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notNewCodePeriod{},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.New(errNotNewCodePeriod),
			},
		},
		"UnsetFails": {
			client: &fake.MockNewCodePeriodsClient{
				UnsetFn: func(opt *sonar.NewCodePeriodsUnsetOptions) (*http.Response, error) {
					return nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("api error"), "failed to unset instance new code period"),
			},
		},
		"SuccessfulDelete": {
			client: &fake.MockNewCodePeriodsClient{
				UnsetFn: func(opt *sonar.NewCodePeriodsUnsetOptions) (*http.Response, error) {
					if opt.Project != "" {
						return nil, errors.New("expected project to be empty")
					}

					if opt.Branch != "" {
						return nil, errors.New("expected branch to be empty")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.NewCodePeriod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-new-code-period"},
					Spec: v1alpha1.NewCodePeriodSpec{
						ForProvider: v1alpha1.NewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: new("30"),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{newCodePeriodsClient: tc.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Delete() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	client := &fake.MockNewCodePeriodsClient{}
	e := &external{newCodePeriodsClient: client}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// mockGate is a mock implementation of the feature gate interface.
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

// newNewCodePeriodResource returns a NewCodePeriod ready for Connect tests.
func newNewCodePeriodResource(name string) *v1alpha1.NewCodePeriod {
	return &v1alpha1.NewCodePeriod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.NewCodePeriodGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.NewCodePeriodKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.NewCodePeriodSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{},
			ForProvider: v1alpha1.NewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: new("30"),
			},
		},
	}
}

// TestConnectTypeAssertion verifies Connect returns an error for
// non-NewCodePeriod types.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notNewCodePeriod{})
	if err == nil {
		t.Fatal("Connect() expected error for non-NewCodePeriod type, got nil")
	}

	if !strings.Contains(err.Error(), errNotNewCodePeriod) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errNotNewCodePeriod)
	}
}

// TestConnectTrackUsageError verifies Connect returns an error when
// usage tracking fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	ncp := newNewCodePeriodResource("test-new-code-period")

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), ncp)
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
	ncp := newNewCodePeriodResource("test-new-code-period")
	ncp.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), ncp)
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
	ncp := newNewCodePeriodResource("test-new-code-period")
	ncp.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: instance.NewNewCodePeriodsClient,
	}

	got, err := c.Connect(context.Background(), ncp)
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

// TestSetupGatedRegistersNewCodePeriodGVK verifies SetupGated registers
// the NewCodePeriod GVK.
func TestSetupGatedRegistersNewCodePeriodGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.NewCodePeriodGroupVersionKind, g.gvks[0]); diff != "" {
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
