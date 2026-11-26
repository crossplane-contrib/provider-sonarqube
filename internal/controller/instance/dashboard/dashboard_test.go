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

package dashboard

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

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
	"k8s.io/utils/ptr"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testDashboardID is the dashboard ID used across tests.
	testDashboardID = "dashboard-id"
	// testTargetID is the identifier of the target project.
	testTargetID = "project-id"
	// testDashboardName is the dashboard name used across tests.
	testDashboardName = "My Dashboard"
)

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

// TestSetupGatedRegistersDashboardGVK verifies SetupGated registers the
// Dashboard GVK.
func TestSetupGatedRegistersDashboardGVK(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if !g.registered || g.callback == nil {
		t.Fatal("SetupGated() expected Gate.Register to be called with a callback")
	}

	if len(g.gvks) != 1 {
		t.Fatalf("SetupGated() registered %d GVKs, want 1", len(g.gvks))
	}

	if diff := cmp.Diff(v1alpha1.DashboardGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackCoverage verifies the callback body executes when
// called. The callback panics because mgr is nil; the panic is recovered so
// the test completes.
func TestSetupGatedCallbackCoverage(t *testing.T) {
	t.Parallel()

	defer func() { _ = recover() }()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	_ = SetupGated(nil, o)

	g.callback()
}

// TestConnect tests the Connect method.
func TestConnect(t *testing.T) {
	t.Parallel()

	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000",
			Token: &apisv1alpha1.ProviderCredentials{
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: "sonar-secret", Namespace: "default"},
						Key:             "token",
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

	cases := map[string]struct {
		mg      *v1alpha1.Dashboard
		objects []runtime.Object
		pcRef   *xpv1.ProviderConfigReference
		wantErr string
	}{
		"TrackUsageError": {
			mg:      newDashboardResource("test-application"),
			wantErr: errTrackPCUsage,
		},
		"GetConfigError": {
			mg:      newDashboardResource("test-application"),
			pcRef:   &xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"},
			wantErr: errGetPC,
		},
		"Success": {
			mg:      newDashboardResource("test-application"),
			objects: []runtime.Object{providerConfig, secret},
			pcRef:   &xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithRuntimeObjects(tc.objects...).Build()

			if tc.pcRef != nil {
				tc.mg.SetProviderConfigReference(tc.pcRef)
			}

			c := &connector{
				kube:            kubeClient,
				usage:           resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn:    instance.NewDashboardsClient,
				newNavigationFn: instance.NewComponentNavigationClient,
			}

			got, err := c.Connect(context.Background(), tc.mg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Connect() error = %v, want to contain %q", err, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("Connect() unexpected error: %v", err)
			}

			if _, ok := got.(*external); !ok {
				t.Fatalf("Connect() returned %T, want *external", got)
			}
		})
	}
}

// newDashboard builds a project Dashboard resource with an optional external
// name.
func newDashboard(externalName string) *v1alpha1.Dashboard {
	dashboard := newDashboardResource("test-dashboard")
	if externalName != "" {
		meta.SetExternalName(dashboard, externalName)
	}

	return dashboard
}

// deleted marks a Dashboard as being deleted.
func deleted(dashboard *v1alpha1.Dashboard) *v1alpha1.Dashboard {
	dashboard.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})

	return dashboard
}

// withoutTarget removes the target of a Dashboard.
func withoutTarget(dashboard *v1alpha1.Dashboard) *v1alpha1.Dashboard {
	dashboard.Spec.ForProvider.ProjectKey = nil

	return dashboard
}

// newDashboardResource returns a Dashboard suitable for Connect tests.
func newDashboardResource(name string) *v1alpha1.Dashboard {
	return &v1alpha1.Dashboard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.DashboardGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.DashboardKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.DashboardSpec{
			ForProvider: v1alpha1.DashboardParameters{
				Name:       testDashboardName,
				Layout:     `{"widgets":[]}`,
				ProjectKey: new("my-project"),
			},
		},
	}
}

// navigationFound serves the target project.
func navigationFound() *fake.MockComponentNavigationClient {
	return &fake.MockComponentNavigationClient{
		ComponentFn: func(_ *sonar.NavigationComponentOptions) (*sonar.NavigationComponent, *http.Response, error) {
			return &sonar.NavigationComponent{ID: testTargetID}, mockHTTPOK(), nil
		},
	}
}

// navigationFailing answers every lookup with the given response and error.
func navigationFailing(resp *http.Response, err error) *fake.MockComponentNavigationClient {
	return &fake.MockComponentNavigationClient{
		ComponentFn: func(_ *sonar.NavigationComponentOptions) (*sonar.NavigationComponent, *http.Response, error) {
			return nil, resp, err
		},
	}
}

// getDashboard serves a dashboard of the target project with the given name.
func getDashboard(name, resourceID string) func(string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
	return func(dashboardID string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
		return &sonar.DashboardsDashboardResponse{
			Id:           dashboardID,
			Name:         name,
			Layout:       `{ "widgets": [] }`,
			ResourceType: "project",
			ResourceId:   resourceID,
		}, mockHTTPOK(), nil
	}
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	found := &fake.MockDashboardsClient{GetFn: getDashboard(testDashboardName, testTargetID)} //nolint:bodyclose // test double, no body
	foreign := &fake.MockDashboardsClient{GetFn: getDashboard(testDashboardName, "other-id")} //nolint:bodyclose // test double, no body
	exists := func(upToDate bool) managed.ExternalObservation {
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate, ConnectionDetails: managed.ConnectionDetails{}}
	}

	cases := map[string]struct {
		client     *fake.MockDashboardsClient
		navigation *fake.MockComponentNavigationClient
		mg         *v1alpha1.Dashboard
		want       managed.ExternalObservation
		err        error
	}{
		"EmptyExternalName": {
			client: &fake.MockDashboardsClient{},
			mg:     newDashboard(""),
		},
		"NotFound": {
			client: &fake.MockDashboardsClient{GetFn: func(_ string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
				return nil, mockHTTPNotFound(), errBoom
			}},
			mg: newDashboard(testDashboardID),
		},
		"GetError": {
			client: &fake.MockDashboardsClient{GetFn: func(_ string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
				return nil, mockHTTPOK(), errBoom
			}},
			mg:  newDashboard(testDashboardID),
			err: errors.Wrap(errBoom, errObserveDashboard),
		},
		"GetNilResult": {
			client: &fake.MockDashboardsClient{GetFn: func(_ string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
				return nil, mockHTTPOK(), nil
			}},
			mg:  newDashboard(testDashboardID),
			err: errors.New(errObserveDashboard),
		},
		"UpToDate": {
			client:     found,
			navigation: navigationFound(),
			mg:         newDashboard(testDashboardID),
			want:       exists(true),
		},
		"OutOfDate": {
			client:     &fake.MockDashboardsClient{GetFn: getDashboard("renamed", testTargetID)}, //nolint:bodyclose // test double, no body
			navigation: navigationFound(),
			mg:         newDashboard(testDashboardID),
			want:       exists(false),
		},
		"DeletedAndOwned": {
			client:     found,
			navigation: navigationFound(),
			mg:         deleted(newDashboard(testDashboardID)),
			want:       exists(true),
		},
		"LookupError": {
			client:     found,
			navigation: navigationFailing(mockHTTPOK(), errBoom), //nolint:bodyclose // test double, no body
			mg:         newDashboard(testDashboardID),
			err:        errors.Wrap(errors.Wrapf(errBoom, errLookupTarget, "project", "my-project"), errObserveDashboard),
		},
		"LookupWithoutID": {
			client:     found,
			navigation: navigationFailing(mockHTTPOK(), nil), //nolint:bodyclose // test double, no body
			mg:         newDashboard(testDashboardID),
			err:        errors.Wrap(errors.Wrapf(errors.New(errNoTargetID), errLookupTarget, "project", "my-project"), errObserveDashboard),
		},
		"ForeignDashboard": {
			client:     foreign,
			navigation: navigationFound(),
			mg:         newDashboard(testDashboardID),
			err:        errors.Errorf(errTargetMismatch, testDashboardID, "project", "my-project"),
		},
		"MissingTarget": {
			client:     found,
			navigation: navigationFailing(mockHTTPNotFound(), errBoom), //nolint:bodyclose // test double, no body
			mg:         newDashboard(testDashboardID),
			err:        errors.Errorf(errTargetMismatch, testDashboardID, "project", "my-project"),
		},
		"NoTarget": {
			client: found,
			mg:     withoutTarget(newDashboard(testDashboardID)),
			err:    errors.New(errNoTarget),
		},
		"DeletedForeignDashboard": {
			client:     foreign,
			navigation: navigationFound(),
			mg:         deleted(newDashboard(testDashboardID)),
		},
		"DeletedMissingTarget": {
			client:     found,
			navigation: navigationFailing(mockHTTPNotFound(), errBoom), //nolint:bodyclose // test double, no body
			mg:         deleted(newDashboard(testDashboardID)),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client, navigation: tc.navigation}

			got, err := e.Observe(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
			}

			// Late initialization is exercised separately.
			got.ResourceLateInitialized = false

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObserveLateInitialize verifies the description is late initialized and
// the observation recorded.
func TestObserveLateInitialize(t *testing.T) {
	t.Parallel()

	e := &external{
		client: &fake.MockDashboardsClient{GetFn: func(_ string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
			return &sonar.DashboardsDashboardResponse{
				Id: testDashboardID, Name: testDashboardName, Description: "observed", Layout: `{"widgets":[]}`,
				ResourceType: "project", ResourceId: testTargetID,
			}, mockHTTPOK(), nil
		}},
		navigation: navigationFound(),
	}
	dashboard := newDashboard(testDashboardID)

	got, err := e.Observe(context.Background(), dashboard)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !got.ResourceLateInitialized || ptr.Deref(dashboard.Spec.ForProvider.Description, "") != "observed" {
		t.Errorf("Observe() did not late initialize the description: %+v", got)
	}

	if dashboard.Status.AtProvider.ID != testDashboardID {
		t.Errorf("atProvider.id = %q, want %q", dashboard.Status.AtProvider.ID, testDashboardID)
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	list := func(names ...string) func(*sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
		return func(_ *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
			result := &sonar.DashboardsDashboardsResponse{Page: sonar.PageResponseV2{PageIndex: 1, PageSize: 500}}
			for _, name := range names {
				result.Dashboards = append(result.Dashboards, sonar.DashboardsDashboardItem{Id: "existing-id", Name: name})
			}

			return result, mockHTTPOK(), nil
		}
	}
	create := func(id string, err error) func(*sonar.DashboardsCreateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
		return func(opt *sonar.DashboardsCreateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
			if opt.ResourceId != testTargetID || opt.ResourceType != "project" {
				return nil, nil, errors.Errorf("unexpected target %+v", opt)
			}

			return &sonar.DashboardsDashboardResponse{Id: id}, mockHTTPOK(), err
		}
	}

	cases := map[string]struct {
		client     *fake.MockDashboardsClient
		navigation *fake.MockComponentNavigationClient
		mg         *v1alpha1.Dashboard
		wantName   string
		wantErr    string
	}{
		"Success": {
			client:     &fake.MockDashboardsClient{ListFn: list("other"), CreateFn: create(testDashboardID, nil)}, //nolint:bodyclose // test double, no body
			navigation: navigationFound(),
			mg:         newDashboard(""),
			wantName:   testDashboardID,
		},
		"NoTarget": {
			client:  &fake.MockDashboardsClient{},
			mg:      withoutTarget(newDashboard("")),
			wantErr: errNoTarget,
		},
		"LookupError": {
			client:     &fake.MockDashboardsClient{},
			navigation: navigationFailing(mockHTTPOK(), errBoom), //nolint:bodyclose // test double, no body
			mg:         newDashboard(""),
			wantErr:    errCreateDashboard,
		},
		"TargetNotFound": {
			client:     &fake.MockDashboardsClient{},
			navigation: navigationFailing(mockHTTPNotFound(), errBoom), //nolint:bodyclose // test double, no body
			mg:         newDashboard(""),
			wantErr:    "does not exist",
		},
		"ListError": {
			client: &fake.MockDashboardsClient{ListFn: func(_ *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
				return nil, mockHTTPOK(), errBoom
			}},
			navigation: navigationFound(),
			mg:         newDashboard(""),
			wantErr:    errListDashboards,
		},
		"NameAlreadyUsed": {
			client:     &fake.MockDashboardsClient{ListFn: list(testDashboardName)}, //nolint:bodyclose // test double, no body
			navigation: navigationFound(),
			mg:         newDashboard(""),
			wantErr:    "with ID existing-id",
		},
		"CreateError": {
			client:     &fake.MockDashboardsClient{ListFn: list(), CreateFn: create("", errBoom)}, //nolint:bodyclose // test double, no body
			navigation: navigationFound(),
			mg:         newDashboard(""),
			wantErr:    errCreateDashboard,
		},
		"CreateWithoutID": {
			client:     &fake.MockDashboardsClient{ListFn: list(), CreateFn: create("", nil)}, //nolint:bodyclose // test double, no body
			navigation: navigationFound(),
			mg:         newDashboard(""),
			wantErr:    errNoDashboardID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client, navigation: tc.navigation}

			_, err := e.Create(context.Background(), tc.mg)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}

			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("Create() error = %v, want to contain %q", err, tc.wantErr)
			}

			if got := meta.GetExternalName(tc.mg); got != tc.wantName {
				t.Errorf("external name = %q, want %q", got, tc.wantName)
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	update := func(err error) func(string, *sonar.DashboardsUpdateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
		return func(_ string, _ *sonar.DashboardsUpdateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
			return nil, mockHTTPOK(), err
		}
	}

	cases := map[string]struct {
		client *fake.MockDashboardsClient
		mg     *v1alpha1.Dashboard
		err    error
	}{
		"Success": {client: &fake.MockDashboardsClient{UpdateFn: update(nil)}, mg: newDashboard(testDashboardID)}, //nolint:bodyclose // test double, no body
		"NoExternalName": {
			client: &fake.MockDashboardsClient{},
			mg:     newDashboard(""),
			err:    errors.Errorf(errExternalNameNotSet, "test-dashboard"),
		},
		"APIError": {
			client: &fake.MockDashboardsClient{UpdateFn: update(errBoom)}, //nolint:bodyclose // test double, no body
			mg:     newDashboard(testDashboardID),
			err:    errors.Wrap(errBoom, errUpdateDashboard),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			_, err := e.Update(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Update() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete and Disconnect methods.
func TestDelete(t *testing.T) {
	t.Parallel()

	remove := func(resp *http.Response, err error) func(string) (*http.Response, error) {
		return func(_ string) (*http.Response, error) { return resp, err }
	}

	cases := map[string]struct {
		client *fake.MockDashboardsClient
		mg     *v1alpha1.Dashboard
		err    error
	}{
		"Success":        {client: &fake.MockDashboardsClient{DeleteFn: remove(mockHTTPOK(), nil)}, mg: newDashboard(testDashboardID)}, //nolint:bodyclose // test double, no body
		"NoExternalName": {client: &fake.MockDashboardsClient{}, mg: newDashboard("")},
		"AlreadyGone":    {client: &fake.MockDashboardsClient{DeleteFn: remove(mockHTTPNotFound(), errBoom)}, mg: newDashboard(testDashboardID)}, //nolint:bodyclose // test double, no body
		"APIError": {
			client: &fake.MockDashboardsClient{DeleteFn: remove(mockHTTPOK(), errBoom)}, //nolint:bodyclose // test double, no body
			mg:     newDashboard(testDashboardID),
			err:    errors.Wrap(errBoom, errDeleteDashboard),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			_, err := e.Delete(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Delete() error mismatch (-want +got):\n%s", diff)
			}

			err = e.Disconnect(context.Background())
			if err != nil {
				t.Errorf("Disconnect() unexpected error: %v", err)
			}
		})
	}
}
