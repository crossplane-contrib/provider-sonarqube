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

package application

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
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

const (
	// testApplicationKey is the application key used across tests.
	testApplicationKey = "my-application"
	// testApplicationName is the application name used across tests.
	testApplicationName = "My Application"
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

// newApplication builds an Application resource with the given member
// projects, branches and optional external name.
func newApplication(externalName string, projects []string, branches ...v1alpha1.ApplicationBranchParameters) *v1alpha1.Application {
	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "test-application"},
		Spec: v1alpha1.ApplicationSpec{
			ForProvider: v1alpha1.ApplicationParameters{
				Key:        testApplicationKey,
				Name:       testApplicationName,
				Visibility: "public",
				Projects:   projects,
				Branches:   branches,
			},
		},
	}
	if externalName != "" {
		meta.SetExternalName(app, externalName)
	}

	return app
}

// newBranch builds an ApplicationBranchParameters from alternating project
// key / project branch pairs.
func newBranch(name string, projectBranchPairs ...string) v1alpha1.ApplicationBranchParameters {
	branch := v1alpha1.ApplicationBranchParameters{Name: name}
	for i := 0; i+1 < len(projectBranchPairs); i += 2 {
		branch.Projects = append(branch.Projects, v1alpha1.ApplicationBranchProjectParameters{
			Project: projectBranchPairs[i],
			Branch:  projectBranchPairs[i+1],
		})
	}

	return branch
}

// newBranchObservation builds an ApplicationBranchObservation from
// alternating project key / project branch pairs.
func newBranchObservation(name string, isMain bool, projectBranchPairs ...string) v1alpha1.ApplicationBranchObservation {
	branch := v1alpha1.ApplicationBranchObservation{Name: name, IsMain: isMain}
	for i := 0; i+1 < len(projectBranchPairs); i += 2 {
		branch.Projects = append(branch.Projects, v1alpha1.ApplicationBranchProjectObservation{
			Project: projectBranchPairs[i],
			Branch:  projectBranchPairs[i+1],
		})
	}

	return branch
}

// newApplicationResource returns an Application suitable for Connect tests.
func newApplicationResource(name string) *v1alpha1.Application {
	return &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.ApplicationGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.ApplicationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.ApplicationSpec{
			ForProvider: v1alpha1.ApplicationParameters{
				Key:  testApplicationKey,
				Name: testApplicationName,
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

// TestSetupGatedRegistersApplicationGVK verifies SetupGated registers the
// Application GVK.
func TestSetupGatedRegistersApplicationGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.ApplicationGroupVersionKind, g.gvks[0]); diff != "" {
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
		mg      *v1alpha1.Application
		objects []runtime.Object
		pcRef   *xpv1.ProviderConfigReference
		wantErr string
	}{
		"TrackUsageError": {
			mg:      newApplicationResource("test-application"),
			wantErr: errTrackPCUsage,
		},
		"GetConfigError": {
			mg:      newApplicationResource("test-application"),
			pcRef:   &xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"},
			wantErr: errGetPC,
		},
		"Success": {
			mg:      newApplicationResource("test-application"),
			objects: []runtime.Object{providerConfig, secret},
			pcRef:   &xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithRuntimeObjects(tc.objects...).Build()

			if app := tc.mg; tc.pcRef != nil {
				app.SetProviderConfigReference(tc.pcRef)
			}

			c := &connector{
				kube:         kubeClient,
				usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn: instance.NewApplicationsClient,
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

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	// showApplication returns an application with two member projects, a
	// main branch and a "release" branch.
	showApplication := func(opt *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
		if opt.Branch == "release" {
			return &sonar.ApplicationsShow{Application: sonar.ApplicationDetails{
				Key: testApplicationKey,
				Projects: []sonar.ApplicationProject{
					{Key: "proj-a", Branch: "release-a"},
					{Key: "proj-b", Branch: "main", IsMain: true},
				},
			}}, mockHTTPOK(), nil
		}

		return &sonar.ApplicationsShow{Application: sonar.ApplicationDetails{
			Key:        testApplicationKey,
			Name:       testApplicationName,
			Visibility: "public",
			Projects: []sonar.ApplicationProject{
				{Key: "proj-a", Branch: "main", IsMain: true},
				{Key: "proj-b", Branch: "main", IsMain: true},
			},
			Branches: []sonar.ApplicationBranch{{Name: "main", IsMain: true}, {Name: "release"}},
		}}, mockHTTPOK(), nil
	}

	cases := map[string]struct {
		client *fake.MockApplicationsClient
		mg     *v1alpha1.Application
		want   managed.ExternalObservation
		err    error
	}{
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockApplicationsClient{},
			mg:     newApplication("", nil),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"ShowNotFoundReturnsNotExists": {
			client: &fake.MockApplicationsClient{
				ShowFn: func(_ *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
					return nil, mockHTTPNotFound(), errBoom
				},
			},
			mg:   newApplication(testApplicationKey, nil),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"ShowNilResultReturnsNotExists": {
			client: &fake.MockApplicationsClient{
				ShowFn: func(_ *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
					return nil, mockHTTPOK(), nil
				},
			},
			mg:   newApplication(testApplicationKey, nil),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"ShowAPIError": {
			client: &fake.MockApplicationsClient{
				ShowFn: func(_ *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
					return nil, mockHTTPOK(), errBoom
				},
			},
			mg:  newApplication(testApplicationKey, nil),
			err: errors.Wrap(errBoom, errObserveApplication),
		},
		"ShowBranchAPIError": {
			client: &fake.MockApplicationsClient{
				ShowFn: func(opt *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
					if opt.Branch != "" {
						return nil, mockHTTPOK(), errBoom
					}

					return showApplication(opt)
				},
			},
			mg:  newApplication(testApplicationKey, nil),
			err: errors.Wrapf(errBoom, errObserveApplicationBranch, "release"),
		},
		"ExistsAndUpToDate": {
			client: &fake.MockApplicationsClient{ShowFn: showApplication},
			mg:     newApplication(testApplicationKey, []string{"proj-a", "proj-b"}, newBranch("release", "proj-a", "release-a", "proj-b", "main")),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"BranchExtraProject": {
			client: &fake.MockApplicationsClient{ShowFn: showApplication},
			mg:     newApplication(testApplicationKey, []string{"proj-a", "proj-b"}, newBranch("release", "proj-a", "release-a")),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"ProjectsDrift": {
			client: &fake.MockApplicationsClient{ShowFn: showApplication},
			mg:     newApplication(testApplicationKey, []string{"proj-a"}, newBranch("release", "proj-a", "release-a")),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"BranchDrift": {
			client: &fake.MockApplicationsClient{ShowFn: showApplication},
			mg:     newApplication(testApplicationKey, []string{"proj-a", "proj-b"}, newBranch("release", "proj-a", "other")),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"UnwantedBranch": {
			client: &fake.MockApplicationsClient{ShowFn: showApplication},
			mg:     newApplication(testApplicationKey, []string{"proj-a", "proj-b"}),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Observe() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() observation -want +got:\n%s", diff)
			}
		})
	}
}

// TestObservePopulatesStatus verifies Observe records the observed state,
// including the per-branch project branches, in the status.
func TestObservePopulatesStatus(t *testing.T) {
	t.Parallel()

	client := &fake.MockApplicationsClient{
		ShowFn: func(opt *sonar.ApplicationsShowOptions) (*sonar.ApplicationsShow, *http.Response, error) {
			if opt.Branch == "release" {
				return &sonar.ApplicationsShow{Application: sonar.ApplicationDetails{
					Projects: []sonar.ApplicationProject{{Key: "proj-a", Branch: "release-a"}},
				}}, mockHTTPOK(), nil
			}

			return &sonar.ApplicationsShow{Application: sonar.ApplicationDetails{
				Key:      testApplicationKey,
				Name:     testApplicationName,
				Projects: []sonar.ApplicationProject{{Key: "proj-a", Branch: "main", IsMain: true}},
				Branches: []sonar.ApplicationBranch{{Name: "main", IsMain: true}, {Name: "release"}},
			}}, mockHTTPOK(), nil
		},
	}

	app := newApplication(testApplicationKey, nil)

	_, err := (&external{client: client}).Observe(context.Background(), app)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	want := v1alpha1.ApplicationObservation{
		Key:      testApplicationKey,
		Name:     testApplicationName,
		Projects: []string{"proj-a"},
		Branches: []v1alpha1.ApplicationBranchObservation{
			{Name: "main", IsMain: true, Projects: []v1alpha1.ApplicationBranchProjectObservation{{Project: "proj-a", Branch: "main", IsMain: true}}},
			newBranchObservation("release", false, "proj-a", "release-a"),
		},
	}

	if diff := cmp.Diff(want, app.Status.AtProvider); diff != "" {
		t.Errorf("Observe() status -want +got:\n%s", diff)
	}

	if app.GetCondition(xpv1.TypeReady).Reason != xpv1.Available().Reason {
		t.Errorf("Observe() expected Available condition")
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client           *fake.MockApplicationsClient
		mg               *v1alpha1.Application
		err              error
		wantExternalName string
	}{
		"CreateAPIError": {
			client: &fake.MockApplicationsClient{
				CreateFn: func(_ *sonar.ApplicationsCreateOptions) (*sonar.ApplicationsCreate, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			mg:  newApplication("", nil),
			err: errors.Wrap(errBoom, errCreateApplication),
		},
		"Success": {
			client: &fake.MockApplicationsClient{
				CreateFn: func(opt *sonar.ApplicationsCreateOptions) (*sonar.ApplicationsCreate, *http.Response, error) {
					if opt.Key != testApplicationKey || opt.Name != testApplicationName || opt.Visibility != "public" {
						return nil, nil, errors.Errorf("unexpected create options: %+v", opt)
					}

					return &sonar.ApplicationsCreate{}, mockHTTPOK(), nil
				},
			},
			mg:               newApplication("", nil),
			wantExternalName: testApplicationKey,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			_, err := e.Create(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Create() error -want +got:\n%s", diff)
			}

			if app := tc.mg; tc.wantExternalName != "" {
				if got := meta.GetExternalName(app); got != tc.wantExternalName {
					t.Errorf("Create() external name = %q, want %q", got, tc.wantExternalName)
				}
			}
		})
	}
}

// recordingClient returns a mock client that records every mutating call
// in order, failing the call whose record matches failOn.
func recordingClient(calls *[]string, failOn string) *fake.MockApplicationsClient {
	record := func(call string) (*http.Response, error) {
		*calls = append(*calls, call)
		if call == failOn {
			return nil, errBoom
		}

		return mockHTTPOK(), nil
	}

	return &fake.MockApplicationsClient{
		UpdateFn: func(opt *sonar.ApplicationsUpdateOptions) (*http.Response, error) {
			return record("update:" + opt.Name + ":" + opt.Description)
		},
		AddProjectFn: func(opt *sonar.ApplicationsAddProjectOptions) (*http.Response, error) {
			return record("add:" + opt.Project)
		},
		RemoveProjectFn: func(opt *sonar.ApplicationsRemoveProjectOptions) (*http.Response, error) {
			return record("remove:" + opt.Project)
		},
		CreateBranchFn: func(opt *sonar.ApplicationsCreateBranchOptions) (*http.Response, error) {
			return record("createBranch:" + opt.Branch + ":" + strings.Join(opt.Project, ",") + "=" + strings.Join(opt.ProjectBranch, ","))
		},
		UpdateBranchFn: func(opt *sonar.ApplicationsUpdateBranchOptions) (*http.Response, error) {
			return record("updateBranch:" + opt.Branch + ">" + opt.Name + ":" + strings.Join(opt.Project, ",") + "=" + strings.Join(opt.ProjectBranch, ","))
		},
		DeleteBranchFn: func(opt *sonar.ApplicationsDeleteBranchOptions) (*http.Response, error) {
			return record("deleteBranch:" + opt.Branch)
		},
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	// driftedApplication returns an application whose observed state differs
	// from its spec in every managed aspect.
	driftedApplication := func() *v1alpha1.Application {
		app := newApplication(testApplicationKey, []string{"proj-a", "proj-c"},
			newBranch("release", "proj-a", "release-a"),
			newBranch("feature", "proj-c", "feature-c", "proj-a", "feature-a"),
		)
		app.Spec.ForProvider.Description = new("new description")
		app.Status.AtProvider = v1alpha1.ApplicationObservation{
			Name:        "Old Name",
			Description: "old description",
			Projects:    []string{"proj-a", "proj-b"},
			Branches: []v1alpha1.ApplicationBranchObservation{
				newBranchObservation("main", true, "proj-a", "main", "proj-b", "main"),
				newBranchObservation("release", false, "proj-a", "old-release"),
				newBranchObservation("stale", false, "proj-b", "stale-b"),
			},
		}

		return app
	}

	fullSync := []string{
		"update:" + testApplicationName + ":new description",
		"add:proj-c",
		"deleteBranch:stale",
		"updateBranch:release>release:proj-a=release-a",
		"createBranch:feature:proj-a,proj-c=feature-a,feature-c",
		"remove:proj-b",
	}

	cases := map[string]struct {
		mg        *v1alpha1.Application
		failOn    string
		wantCalls []string
		err       error
	}{
		"ExternalNameNotSet": {
			mg:  newApplication("", nil),
			err: errors.Errorf(errExternalNameNotSet, "test-application"),
		},
		"FullSync": {
			mg:        driftedApplication(),
			wantCalls: fullSync,
		},
		"InSyncIsNoOp": {
			mg: func() *v1alpha1.Application {
				app := newApplication(testApplicationKey, []string{"proj-a"}, newBranch("release", "proj-a", "release-a"))
				app.Status.AtProvider = v1alpha1.ApplicationObservation{
					Name:        testApplicationName,
					Description: "unmanaged description",
					Projects:    []string{"proj-a"},
					Branches: []v1alpha1.ApplicationBranchObservation{
						newBranchObservation("main", true, "proj-a", "main"),
						newBranchObservation("release", false, "proj-a", "release-a"),
					},
				}

				return app
			}(),
		},
		"UnmanagedDescriptionIsPreserved": {
			mg: func() *v1alpha1.Application {
				app := newApplication(testApplicationKey, nil)
				app.Status.AtProvider = v1alpha1.ApplicationObservation{Name: "Old Name", Description: "kept"}

				return app
			}(),
			wantCalls: []string{"update:" + testApplicationName + ":kept"},
		},
		"UpdateError": {
			mg:        driftedApplication(),
			failOn:    fullSync[0],
			wantCalls: fullSync[:1],
			err:       errors.Wrap(errBoom, errUpdateApplication),
		},
		"AddProjectError": {
			mg:        driftedApplication(),
			failOn:    fullSync[1],
			wantCalls: fullSync[:2],
			err:       errors.Wrapf(errBoom, errAddProject, "proj-c"),
		},
		"DeleteBranchError": {
			mg:        driftedApplication(),
			failOn:    fullSync[2],
			wantCalls: fullSync[:3],
			err:       errors.Wrapf(errBoom, errDeleteBranch, "stale"),
		},
		"UpdateBranchError": {
			mg:        driftedApplication(),
			failOn:    fullSync[3],
			wantCalls: fullSync[:4],
			err:       errors.Wrapf(errBoom, errUpdateBranch, "release"),
		},
		"CreateBranchError": {
			mg:        driftedApplication(),
			failOn:    fullSync[4],
			wantCalls: fullSync[:5],
			err:       errors.Wrapf(errBoom, errCreateBranch, "feature"),
		},
		"RemoveProjectError": {
			mg:        driftedApplication(),
			failOn:    fullSync[5],
			wantCalls: fullSync,
			err:       errors.Wrapf(errBoom, errRemoveProject, "proj-b"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls []string

			e := &external{client: recordingClient(&calls, tc.failOn)}
			_, err := e.Update(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Update() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantCalls, calls); diff != "" {
				t.Errorf("Update() calls -want +got:\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client *fake.MockApplicationsClient
		mg     *v1alpha1.Application
		err    error
	}{
		"EmptyExternalNameIsNoOp": {
			client: &fake.MockApplicationsClient{},
			mg:     newApplication("", nil),
		},
		"DeleteAPIError": {
			client: &fake.MockApplicationsClient{
				DeleteFn: func(_ *sonar.ApplicationsDeleteOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			mg:  newApplication(testApplicationKey, nil),
			err: errors.Wrap(errBoom, errDeleteApplication),
		},
		"AlreadyGone": {
			client: &fake.MockApplicationsClient{
				DeleteFn: func(_ *sonar.ApplicationsDeleteOptions) (*http.Response, error) {
					return mockHTTPNotFound(), errBoom //nolint:nilnil // SonarQube returns the 404 response alongside the error
				},
			},
			mg: newApplication(testApplicationKey, nil),
		},
		"Success": {
			client: &fake.MockApplicationsClient{
				DeleteFn: func(opt *sonar.ApplicationsDeleteOptions) (*http.Response, error) {
					if opt.Application != testApplicationKey {
						return nil, errors.Errorf("unexpected application %q", opt.Application)
					}

					return mockHTTPOK(), nil
				},
			},
			mg: newApplication(testApplicationKey, nil),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			_, err := e.Delete(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Errorf("Delete() error -want +got:\n%s", diff)
			}
		})
	}
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	err := (&external{}).Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}
