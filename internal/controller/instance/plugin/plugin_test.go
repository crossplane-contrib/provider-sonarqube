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

package plugin

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
	// testPluginKey is the plugin key used across tests.
	testPluginKey = "findbugs"
)

// notPlugin is a sentinel type used to test invalid managed resource handling.
type notPlugin struct {
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

// mockHTTPOK returns a mock 200 OK HTTP response for testing.
func mockHTTPOK() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}
}

// newPlugin builds a Plugin resource with the given key and optional
// external name.
func newPlugin(key, externalName string) *v1alpha1.Plugin {
	p := &v1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin"},
		Spec: v1alpha1.PluginSpec{
			ForProvider: v1alpha1.PluginParameters{Key: key},
		},
	}
	if externalName != "" {
		meta.SetExternalName(p, externalName)
	}

	return p
}

// errBoom is a generic test error.
var errBoom = errors.New("boom")

// errComparer compares errors by their message.
var errComparer = cmp.Comparer(func(x, y error) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}

	return x.Error() == y.Error()
})

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
		client *fake.MockPluginsClient
		args   args
		want   want
	}{
		"NotPluginError": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: &notPlugin{}},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotPlugin),
			},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"InstalledAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errObservePlugin),
			},
		},
		"PluginNotInInstalledNorPending": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: "other-plugin"}},
					}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"PluginPendingInstall": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{Plugins: []sonar.PluginInstalled{}}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return &sonar.PluginsPending{
						Installing: []sonar.PluginPending{{Key: testPluginKey, Name: "FindBugs"}},
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
				err: nil,
			},
		},
		"PendingAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{Plugins: []sonar.PluginInstalled{}}, mockHTTPOK(), nil
				},
				PendingFn: func() (*sonar.PluginsPending, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errObservePlugin),
			},
		},
		"UpdatesAPIError": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: true,
				},
				err: errors.Wrap(errBoom, "cannot get plugin updates from SonarQube"),
			},
		},
		"PluginAtLatestVersion": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{Plugins: []sonar.PluginWithUpdates{}}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
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
		"PluginHasUpdateAutoUpdateEnabled": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Plugin {
					p := newPlugin(testPluginKey, testPluginKey)
					autoUpdate := true
					p.Spec.ForProvider.AutoUpdate = &autoUpdate

					return p
				}(),
			},
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
		"PluginHasUpdateAutoUpdateDisabled": {
			client: &fake.MockPluginsClient{
				InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
					return &sonar.PluginsInstalled{
						Plugins: []sonar.PluginInstalled{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
				UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
					return &sonar.PluginsUpdates{
						Plugins: []sonar.PluginWithUpdates{{Key: testPluginKey}},
					}, mockHTTPOK(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Plugin {
					p := newPlugin(testPluginKey, testPluginKey)
					autoUpdate := false
					p.Spec.ForProvider.AutoUpdate = &autoUpdate

					return p
				}(),
			},
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

// TestObservePopulatesAtProvider verifies Observe writes all
// observation fields.
func TestObservePopulatesAtProvider(t *testing.T) {
	t.Parallel()

	mg := newPlugin(testPluginKey, testPluginKey)
	e := &external{client: &fake.MockPluginsClient{
		InstalledFn: func(_ *sonar.PluginsInstalledOptions) (*sonar.PluginsInstalled, *http.Response, error) {
			return &sonar.PluginsInstalled{
				Plugins: []sonar.PluginInstalled{
					{
						Key:         testPluginKey,
						Name:        "FindBugs",
						Version:     "4.0.0",
						Description: "Find bugs",
						Filename:    "findbugs.jar",
						License:     "LGPL",
					},
				},
			}, mockHTTPOK(), nil
		},
		UpdatesFn: func() (*sonar.PluginsUpdates, *http.Response, error) {
			return &sonar.PluginsUpdates{Plugins: []sonar.PluginWithUpdates{}}, mockHTTPOK(), nil
		},
	}}

	_, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	obs := mg.Status.AtProvider
	if obs.Name != "FindBugs" {
		t.Errorf("AtProvider.Name = %q, want %q", obs.Name, "FindBugs")
	}

	if obs.Version != "4.0.0" {
		t.Errorf("AtProvider.Version = %q, want %q", obs.Version, "4.0.0")
	}

	if !obs.IsLatest {
		t.Errorf("AtProvider.IsLatest = false, want true (no updates available)")
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

	cases := map[string]struct {
		client *fake.MockPluginsClient
		args   args
		want   want
	}{
		"NotPluginError": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: &notPlugin{}},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotPlugin),
			},
		},
		"InstallAPIError": {
			client: &fake.MockPluginsClient{
				InstallFn: func(_ *sonar.PluginsInstallOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errInstallPlugin),
			},
		},
		"SuccessSetsExternalNameToKey": {
			client: &fake.MockPluginsClient{
				InstallFn: func(opt *sonar.PluginsInstallOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want: want{
				o:            managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
				err:          nil,
				externalName: testPluginKey,
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
				if p, ok := tc.args.mg.(*v1alpha1.Plugin); ok {
					if got := meta.GetExternalName(p); got != tc.want.externalName {
						t.Errorf("Create() external name = %q, want %q", got, tc.want.externalName)
					}
				}
			}
		})
	}
}

// TestCreatePassesCorrectKey verifies Install is called with the plugin key.
func TestCreatePassesCorrectKey(t *testing.T) {
	t.Parallel()

	var capturedKey string

	e := &external{client: &fake.MockPluginsClient{
		InstallFn: func(opt *sonar.PluginsInstallOptions) (*http.Response, error) {
			capturedKey = opt.Key

			return mockHTTPOK(), nil
		},
	}}

	_, err := e.Create(context.Background(), newPlugin(testPluginKey, ""))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if capturedKey != testPluginKey {
		t.Errorf("Install() called with key %q, want %q", capturedKey, testPluginKey)
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
		client *fake.MockPluginsClient
		args   args
		want   want
	}{
		"NotPluginError": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: &notPlugin{}},
			want:   want{o: managed.ExternalUpdate{}, err: errors.New(errNotPlugin)},
		},
		"EmptyExternalNameReturnsError": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New("cannot update SonarQube Plugin without external name"),
			},
		},
		"UpdateAPIError": {
			client: &fake.MockPluginsClient{
				UpdateFn: func(_ *sonar.PluginsUpdateOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, "cannot update SonarQube Plugin"),
			},
		},
		"Success": {
			client: &fake.MockPluginsClient{
				UpdateFn: func(_ *sonar.PluginsUpdateOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
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

// TestUpdatePassesExternalNameAsKey verifies Update is called with
// the external name.
func TestUpdatePassesExternalNameAsKey(t *testing.T) {
	t.Parallel()

	var capturedKey string

	e := &external{client: &fake.MockPluginsClient{
		UpdateFn: func(opt *sonar.PluginsUpdateOptions) (*http.Response, error) {
			capturedKey = opt.Key

			return mockHTTPOK(), nil
		},
	}}

	_, err := e.Update(context.Background(), newPlugin(testPluginKey, testPluginKey))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if capturedKey != testPluginKey {
		t.Errorf("Update() called with key %q, want %q", capturedKey, testPluginKey)
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
		client *fake.MockPluginsClient
		args   args
		want   want
	}{
		"NotPluginError": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: &notPlugin{}},
			want:   want{o: managed.ExternalDelete{}, err: errors.New(errNotPlugin)},
		},
		"EmptyExternalNameIsNoOp": {
			client: &fake.MockPluginsClient{},
			args:   args{ctx: context.Background(), mg: newPlugin(testPluginKey, "")},
			want:   want{o: managed.ExternalDelete{}, err: nil},
		},
		"UninstallAPIError": {
			client: &fake.MockPluginsClient{
				UninstallFn: func(_ *sonar.PluginsUninstallOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{o: managed.ExternalDelete{}, err: errors.Wrap(errBoom, errUninstallPlugin)},
		},
		"Success": {
			client: &fake.MockPluginsClient{
				UninstallFn: func(_ *sonar.PluginsUninstallOptions) (*http.Response, error) {
					return mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newPlugin(testPluginKey, testPluginKey)},
			want: want{o: managed.ExternalDelete{}, err: nil},
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

// TestDeletePassesExternalNameAsKey verifies Uninstall is called
// with the external name.
func TestDeletePassesExternalNameAsKey(t *testing.T) {
	t.Parallel()

	var capturedKey string

	e := &external{client: &fake.MockPluginsClient{
		UninstallFn: func(opt *sonar.PluginsUninstallOptions) (*http.Response, error) {
			capturedKey = opt.Key

			return mockHTTPOK(), nil
		},
	}}

	_, err := e.Delete(context.Background(), newPlugin(testPluginKey, testPluginKey))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if capturedKey != testPluginKey {
		t.Errorf("Delete() called with key %q, want %q", capturedKey, testPluginKey)
	}
}

// TestDisconnect verifies Disconnect is a no-op returning nil.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockPluginsClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v, want nil", err)
	}
}

// TestSetupGatedRegistersPluginGVK verifies SetupGated registers
// the Plugin GVK.
func TestSetupGatedRegistersPluginGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.PluginGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
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

// newPluginResource returns a Plugin ready for Connect tests.
func newPluginResource(name string) *v1alpha1.Plugin {
	return &v1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.PluginGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.PluginKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.PluginSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{},
			ForProvider:         v1alpha1.PluginParameters{Key: testPluginKey},
		},
	}
}

// TestConnectTypeAssertion verifies Connect returns an error for
// non-Plugin types.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notPlugin{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Plugin type, got nil")
	}

	if !strings.Contains(err.Error(), errNotPlugin) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errNotPlugin)
	}
}

// TestConnectTrackUsageError verifies Connect returns an error when
// usage tracking fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	plug := newPluginResource("test-plugin")

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), plug)
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
	plug := newPluginResource("test-plugin")
	plug.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), plug)
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
	plug := newPluginResource("test-plugin")
	plug.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: instance.NewPluginsClient,
	}

	got, err := c.Connect(context.Background(), plug)
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
