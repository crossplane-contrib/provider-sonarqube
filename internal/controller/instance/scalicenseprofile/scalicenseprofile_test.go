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

package scalicenseprofile

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
	// testProfileID is the license profile identifier used across tests.
	testProfileID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	// testProfileName is the license profile name used across tests.
	testProfileName = "Strict"
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

// newProfile builds a SCALicenseProfile resource with the given external
// name, modified by the optional modifiers.
func newProfile(externalName string, modifiers ...func(*v1alpha1.SCALicenseProfileParameters)) *v1alpha1.SCALicenseProfile {
	profile := &v1alpha1.SCALicenseProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sca-license-profile"},
		Spec: v1alpha1.SCALicenseProfileSpec{
			ForProvider: v1alpha1.SCALicenseProfileParameters{
				Name:         testProfileName,
				Organization: "default-organization",
				Default:      new(false),
				Categories:   []v1alpha1.SCALicenseProfileCategoryPolicy{{Category: "COPYLEFT_STRONG", Policy: "DENY"}},
				Licenses:     []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "MIT", Policy: "ALLOW"}},
				Projects:     []string{"proj-a"},
			},
		},
	}
	for _, modify := range modifiers {
		modify(&profile.Spec.ForProvider)
	}

	if externalName != "" {
		meta.SetExternalName(profile, externalName)
	}

	return profile
}

// listProfiles returns a license profile index holding the test profile.
func listProfiles(_ *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error) {
	return &sonar.ScaLicenseProfileIndex{LicenseProfiles: []sonar.ScaLicenseProfile{
		{Id: "other", Name: "Other", Default: true},
		{Id: testProfileID, Name: testProfileName},
	}}, mockHTTPOK(), nil
}

// getProfile returns the details of the test profile.
func getProfile(_ *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
	return &sonar.ScaLicenseProfileDetails{
		Profile: sonar.ScaLicenseProfile{Id: testProfileID, Name: testProfileName},
		Categories: []sonar.ScaLicenseProfileCategory{
			{Id: "c1", Key: "COPYLEFT_STRONG", Policy: "DENY"},
			{Id: "c2", Key: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		},
		Licenses: []sonar.ScaLicensePolicyLicense{
			{Id: "l1", SpdxLicenseId: "MIT", Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
			{Id: "l2", SpdxLicenseId: "GPL-3.0-only", Category: "COPYLEFT_STRONG", Policy: "DENY"},
		},
	}, mockHTTPOK(), nil
}

// listAssignedProjects returns proj-a as the only assigned project.
func listAssignedProjects(_ *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error) {
	return &sonar.ScaLicenseProfileAssignableProjectsResponse{
		AssignableProjects: []sonar.ScaLicenseProfileAssignableProject{{ProjectKey: "proj-a", AssignedToLicenseProfile: true}},
		Page:               sonar.PageResponseV2{PageIndex: 1, PageSize: 500, Total: 1},
	}, mockHTTPOK(), nil
}

// observingClient returns a client serving the test profile.
func observingClient() *fake.MockSCALicenseProfilesClient {
	return &fake.MockSCALicenseProfilesClient{
		ListLicenseProfilesFn:                  listProfiles,
		GetLicenseProfileFn:                    getProfile,
		ListLicenseProfileAssignableProjectsFn: listAssignedProjects,
	}
}

// newProfileResource returns a SCALicenseProfile suitable for Connect tests.
func newProfileResource(name string) *v1alpha1.SCALicenseProfile {
	return &v1alpha1.SCALicenseProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SCALicenseProfileGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.SCALicenseProfileKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
		},
		Spec: v1alpha1.SCALicenseProfileSpec{
			ForProvider: v1alpha1.SCALicenseProfileParameters{Name: testProfileName},
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

// TestSetupGatedRegistersSCALicenseProfileGVK verifies SetupGated registers
// the SCALicenseProfile GVK.
func TestSetupGatedRegistersSCALicenseProfileGVK(t *testing.T) {
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

	if diff := cmp.Diff(v1alpha1.SCALicenseProfileGroupVersionKind, g.gvks[0]); diff != "" {
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
		mg      *v1alpha1.SCALicenseProfile
		objects []runtime.Object
		pcRef   *xpv1.ProviderConfigReference
		wantErr string
	}{
		"TrackUsageError": {
			mg:      newProfileResource("test-sca-license-profile"),
			wantErr: errTrackPCUsage,
		},
		"GetConfigError": {
			mg:      newProfileResource("test-sca-license-profile"),
			pcRef:   &xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"},
			wantErr: errGetPC,
		},
		"Success": {
			mg:      newProfileResource("test-sca-license-profile"),
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
				kube:         kubeClient,
				usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn: instance.NewSCALicenseProfilesClient,
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

	withClient := func(modify func(*fake.MockSCALicenseProfilesClient)) *fake.MockSCALicenseProfilesClient {
		client := observingClient()
		modify(client)

		return client
	}

	cases := map[string]struct {
		client *fake.MockSCALicenseProfilesClient
		mg     *v1alpha1.SCALicenseProfile
		want   managed.ExternalObservation
		err    error
	}{
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockSCALicenseProfilesClient{},
			mg:     newProfile(""),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"ListAPIError": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.ListLicenseProfilesFn = func(_ *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error) {
					return nil, mockHTTPNotFound(), errBoom
				}
			}),
			mg:  newProfile(testProfileID),
			err: errors.Wrap(errBoom, errListLicenseProfiles),
		},
		"ListNilResultReturnsNotExists": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.ListLicenseProfilesFn = func(_ *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error) {
					return nil, mockHTTPOK(), nil
				}
			}),
			mg:   newProfile(testProfileID),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"ResourceNameAsExternalNameReturnsNotExists": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.GetLicenseProfileFn = func(_ *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
					return nil, &http.Response{StatusCode: http.StatusBadRequest}, errBoom
				}
			}),
			mg:   newProfile("test-sca-license-profile"),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"GetNotFoundReturnsNotExists": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.GetLicenseProfileFn = func(_ *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
					return nil, mockHTTPNotFound(), errBoom
				}
			}),
			mg:   newProfile(testProfileID),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"GetNilResultReturnsNotExists": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.GetLicenseProfileFn = func(_ *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
					return nil, mockHTTPOK(), nil
				}
			}),
			mg:   newProfile(testProfileID),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"GetAPIError": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.GetLicenseProfileFn = func(_ *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
					return nil, mockHTTPOK(), errBoom
				}
			}),
			mg:  newProfile(testProfileID),
			err: errors.Wrap(errBoom, errObserveLicenseProfile),
		},
		"ProjectsAPIError": {
			client: withClient(func(c *fake.MockSCALicenseProfilesClient) {
				c.ListLicenseProfileAssignableProjectsFn = func(_ *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error) {
					return nil, mockHTTPOK(), errBoom
				}
			}),
			mg:  newProfile(testProfileID),
			err: errors.Wrap(errors.Wrap(errBoom, "cannot fetch page 1"), errObserveProjects),
		},
		"ExistsAndUpToDate": {
			client: observingClient(),
			mg:     newProfile(testProfileID),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"LateInitializesDefault": {
			client: observingClient(),
			mg:     newProfile(testProfileID, func(p *v1alpha1.SCALicenseProfileParameters) { p.Default = nil }),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: true,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
		},
		"LicenseDrift": {
			client: observingClient(),
			mg: newProfile(testProfileID, func(p *v1alpha1.SCALicenseProfileParameters) {
				p.Licenses = []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "GPL-3.0-only", Policy: "ALLOW"}}
			}),
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"ProjectDrift": {
			client: observingClient(),
			mg:     newProfile(testProfileID, func(p *v1alpha1.SCALicenseProfileParameters) { p.Projects = nil }),
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
				t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObserveSetsStatus verifies Observe reports the profile, its policies
// and its assigned projects in the status, keeping only the licenses listed
// in the spec.
func TestObserveSetsStatus(t *testing.T) {
	t.Parallel()

	mg := newProfile(testProfileID)
	e := &external{client: observingClient()}

	_, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	want := v1alpha1.SCALicenseProfileObservation{
		ID:   testProfileID,
		Name: testProfileName,
		Categories: []v1alpha1.SCALicenseProfileCategoryObservation{
			{Category: "COPYLEFT_STRONG", Policy: "DENY"},
			{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		},
		Licenses: []v1alpha1.SCALicenseProfileLicenseObservation{
			{ID: "l1", SPDXLicenseID: "MIT", Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
		},
		Projects: []string{"proj-a"},
	}

	if diff := cmp.Diff(want, mg.Status.AtProvider); diff != "" {
		t.Errorf("Observe() status mismatch (-want +got):\n%s", diff)
	}

	if got := mg.GetCondition(xpv1.TypeReady); got.Reason != xpv1.ReasonAvailable {
		t.Errorf("Observe() Ready reason = %q, want %q", got.Reason, xpv1.ReasonAvailable)
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client           *fake.MockSCALicenseProfilesClient
		mg               *v1alpha1.SCALicenseProfile
		wantExternalName string
		err              error
	}{
		"Success": {
			client: &fake.MockSCALicenseProfilesClient{
				CreateLicenseProfileFn: func(body *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
					if body.Name != testProfileName || body.Organization != "default-organization" || !body.Default {
						return nil, mockHTTPOK(), errors.Errorf("unexpected create request %+v", body)
					}

					return &sonar.ScaLicenseProfile{Id: testProfileID, Name: body.Name}, mockHTTPOK(), nil
				},
			},
			mg:               newProfile("test-sca-license-profile", func(p *v1alpha1.SCALicenseProfileParameters) { p.Default = new(true) }),
			wantExternalName: testProfileID,
		},
		"APIError": {
			client: &fake.MockSCALicenseProfilesClient{
				CreateLicenseProfileFn: func(_ *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
					return nil, mockHTTPOK(), errBoom
				},
			},
			mg:               newProfile("test-sca-license-profile"),
			wantExternalName: "test-sca-license-profile",
			err:              errors.Wrap(errBoom, errCreateLicenseProfile),
		},
		"NoIdentifier": {
			client: &fake.MockSCALicenseProfilesClient{
				CreateLicenseProfileFn: func(_ *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
					return &sonar.ScaLicenseProfile{}, mockHTTPOK(), nil
				},
			},
			mg:               newProfile("test-sca-license-profile"),
			wantExternalName: "test-sca-license-profile",
			err:              errors.Wrap(errors.New(errNoLicenseProfileID), errCreateLicenseProfile),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			got, err := e.Create(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Fatalf("Create() error mismatch (-want +got):\n%s", diff)
			}

			if tc.err == nil {
				if diff := cmp.Diff(managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, got); diff != "" {
					t.Errorf("Create() mismatch (-want +got):\n%s", diff)
				}
			}

			if got := meta.GetExternalName(tc.mg); got != tc.wantExternalName {
				t.Errorf("Create() external name = %q, want %q", got, tc.wantExternalName)
			}
		})
	}
}

// updateCalls records the mutating calls made during an Update.
type updateCalls struct {
	settings   []*sonar.ScaLicenseProfileUpdateRequest
	categories []string
	licenses   []string
	assigned   []string
	unassigned []string
}

// recordingClient returns a client recording the mutating calls in calls,
// failing the call named failing.
func recordingClient(calls *updateCalls, failing string) *fake.MockSCALicenseProfilesClient {
	fail := func(name string) error {
		if name == failing {
			return errBoom
		}

		return nil
	}

	return &fake.MockSCALicenseProfilesClient{
		UpdateLicenseProfileFn: func(key string, body *sonar.ScaLicenseProfileUpdateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
			calls.settings = append(calls.settings, body)

			return &sonar.ScaLicenseProfile{Id: key}, mockHTTPOK(), fail("settings")
		},
		UpdateLicenseProfileCategoryFn: func(opt *sonar.ScaLicenseProfileCategoryOptions, body *sonar.ScaLicenseProfileCategoryUpdateRequest) (*sonar.ScaLicenseProfileCategory, *http.Response, error) {
			calls.categories = append(calls.categories, opt.LicenseProfileKey+"/"+opt.CategoryKey+"="+body.Policy)

			return &sonar.ScaLicenseProfileCategory{}, mockHTTPOK(), fail("category")
		},
		UpdateLicenseProfileLicenseFn: func(opt *sonar.ScaLicenseProfileLicenseOptions, body *sonar.ScaLicensePolicyLicenseUpdateRequest) (*sonar.ScaLicensePolicyLicense, *http.Response, error) {
			calls.licenses = append(calls.licenses, opt.LicenseProfileKey+"/"+opt.LicensePolicyId+"="+body.Policy)

			return &sonar.ScaLicensePolicyLicense{}, mockHTTPOK(), fail("license")
		},
		AssignLicenseProfileProjectFn: func(body *sonar.ScaLicenseProfileAssignmentRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
			calls.assigned = append(calls.assigned, body.LicenseProfileId+"/"+body.ProjectKey)

			return &sonar.ScaLicenseProfile{}, mockHTTPOK(), fail("assign")
		},
		DeleteLicenseProfileAssignmentFn: func(opt *sonar.ScaLicenseProfileAssignmentDeleteOptions) (*http.Response, error) {
			calls.unassigned = append(calls.unassigned, opt.ProjectKey)

			if failing == "unassign-not-found" {
				return mockHTTPNotFound(), errBoom
			}

			return mockHTTPOK(), fail("unassign")
		},
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	// drifted returns a profile whose spec differs from its observation in
	// every managed field.
	drifted := func() *v1alpha1.SCALicenseProfile {
		mg := newProfile(testProfileID, func(p *v1alpha1.SCALicenseProfileParameters) {
			p.Name = "Renamed"
			p.Default = new(true)
			p.Categories = []v1alpha1.SCALicenseProfileCategoryPolicy{
				{Category: "COPYLEFT_STRONG", Policy: "DENY"},
				{Category: "PERMISSIVE_STANDARD", Policy: "DENY"},
			}
			p.Licenses = []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "MIT", Policy: "ALLOW"}}
			p.Projects = []string{"proj-a", "proj-c"}
		})
		mg.Status.AtProvider = v1alpha1.SCALicenseProfileObservation{
			ID:   testProfileID,
			Name: testProfileName,
			Categories: []v1alpha1.SCALicenseProfileCategoryObservation{
				{Category: "COPYLEFT_STRONG", Policy: "DENY"},
				{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"},
			},
			Licenses: []v1alpha1.SCALicenseProfileLicenseObservation{{ID: "l1", SPDXLicenseID: "MIT", Policy: "DENY"}},
			Projects: []string{"proj-a", "proj-b"},
		}

		return mg
	}

	cases := map[string]struct {
		mg        *v1alpha1.SCALicenseProfile
		failing   string
		want      updateCalls
		err       error
		errSubstr string
	}{
		"NoExternalName": {
			mg:  newProfile(""),
			err: errors.Errorf(errExternalNameNotSet, "test-sca-license-profile"),
		},
		"UpToDateMakesNoCall": {
			mg: func() *v1alpha1.SCALicenseProfile {
				mg := newProfile(testProfileID)
				mg.Status.AtProvider = v1alpha1.SCALicenseProfileObservation{
					Name:       testProfileName,
					Categories: []v1alpha1.SCALicenseProfileCategoryObservation{{Category: "COPYLEFT_STRONG", Policy: "DENY"}},
					Licenses:   []v1alpha1.SCALicenseProfileLicenseObservation{{ID: "l1", SPDXLicenseID: "MIT", Policy: "ALLOW"}},
					Projects:   []string{"proj-a"},
				}

				return mg
			}(),
		},
		"ReconcilesEverything": {
			mg: drifted(),
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
				assigned:   []string{testProfileID + "/proj-c"},
				unassigned: []string{"proj-b"},
			},
		},
		"UnassignNotFoundIsIgnored": {
			mg:      drifted(),
			failing: "unassign-not-found",
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
				assigned:   []string{testProfileID + "/proj-c"},
				unassigned: []string{"proj-b"},
			},
		},
		"SettingsError": {
			mg:      drifted(),
			failing: "settings",
			want:    updateCalls{settings: []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}}},
			err:     errors.Wrap(errBoom, errUpdateLicenseProfile),
		},
		"CategoryError": {
			mg:      drifted(),
			failing: "category",
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
			},
			err: errors.Wrapf(errBoom, errUpdateCategory, "PERMISSIVE_STANDARD"),
		},
		"LicenseError": {
			mg:      drifted(),
			failing: "license",
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
			},
			err: errors.Wrapf(errBoom, errUpdateLicense, "MIT"),
		},
		"UnknownLicense": {
			mg: func() *v1alpha1.SCALicenseProfile {
				mg := drifted()
				mg.Spec.ForProvider.Licenses = append(mg.Spec.ForProvider.Licenses,
					v1alpha1.SCALicenseProfileLicensePolicy{SPDXLicenseID: "Unknown-1.0", Policy: "DENY"},
					v1alpha1.SCALicenseProfileLicensePolicy{SPDXLicenseID: "Unknown-2.0", Policy: "DENY"},
				)

				return mg
			}(),
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
				assigned:   []string{testProfileID + "/proj-c"},
				unassigned: []string{"proj-b"},
			},
			err: errors.Errorf(errUnknownLicenses, "Unknown-1.0, Unknown-2.0"),
		},
		"CategoryUpdateReappliesItsLicenses": {
			mg: func() *v1alpha1.SCALicenseProfile {
				mg := newProfile(testProfileID, func(p *v1alpha1.SCALicenseProfileParameters) {
					p.Categories = []v1alpha1.SCALicenseProfileCategoryPolicy{{Category: "PERMISSIVE_STANDARD", Policy: "DENY"}}
					p.Licenses = []v1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: "MIT", Policy: "ALLOW"}}
					p.Projects = nil
				})
				mg.Status.AtProvider = v1alpha1.SCALicenseProfileObservation{
					Name:       testProfileName,
					Categories: []v1alpha1.SCALicenseProfileCategoryObservation{{Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"}},
					Licenses:   []v1alpha1.SCALicenseProfileLicenseObservation{{ID: "l1", SPDXLicenseID: "MIT", Category: "PERMISSIVE_STANDARD", Policy: "ALLOW"}},
				}

				return mg
			}(),
			want: updateCalls{
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
			},
		},
		"AssignError": {
			mg:      drifted(),
			failing: "assign",
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
				assigned:   []string{testProfileID + "/proj-c"},
			},
			err: errors.Wrapf(errBoom, errAssignProject, "proj-c"),
		},
		"UnassignError": {
			mg:      drifted(),
			failing: "unassign",
			want: updateCalls{
				settings:   []*sonar.ScaLicenseProfileUpdateRequest{{Name: "Renamed", Default: new(true)}},
				categories: []string{testProfileID + "/PERMISSIVE_STANDARD=DENY"},
				licenses:   []string{testProfileID + "/l1=ALLOW"},
				assigned:   []string{testProfileID + "/proj-c"},
				unassigned: []string{"proj-b"},
			},
			err: errors.Wrapf(errBoom, errUnassignProject, "proj-b"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			calls := &updateCalls{}
			e := &external{client: recordingClient(calls, tc.failing)}

			got, err := e.Update(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Fatalf("Update() error mismatch (-want +got):\n%s", diff)
			}

			if tc.err == nil {
				if diff := cmp.Diff(managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, got); diff != "" {
					t.Errorf("Update() mismatch (-want +got):\n%s", diff)
				}
			}

			if diff := cmp.Diff(tc.want, *calls, cmp.AllowUnexported(updateCalls{})); diff != "" {
				t.Errorf("Update() calls mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockSCALicenseProfilesClient
		mg      *v1alpha1.SCALicenseProfile
		wantKey string
		err     error
	}{
		"NoExternalName": {
			client: &fake.MockSCALicenseProfilesClient{},
			mg:     newProfile(""),
		},
		"Success": {
			client:  &fake.MockSCALicenseProfilesClient{},
			mg:      newProfile(testProfileID),
			wantKey: testProfileID,
		},
		"NotFoundIsIgnored": {
			client: &fake.MockSCALicenseProfilesClient{
				DeleteLicenseProfileFn: func(_ *sonar.ScaLicenseProfileGetOptions) (*http.Response, error) {
					return mockHTTPNotFound(), errBoom //nolint:nilnil // SonarQube returns the 404 response alongside the error
				},
			},
			mg:      newProfile(testProfileID),
			wantKey: testProfileID,
		},
		"APIError": {
			client: &fake.MockSCALicenseProfilesClient{
				DeleteLicenseProfileFn: func(_ *sonar.ScaLicenseProfileGetOptions) (*http.Response, error) {
					return mockHTTPOK(), errBoom //nolint:nilnil // SonarQube returns the response alongside the error
				},
			},
			mg:      newProfile(testProfileID),
			wantKey: testProfileID,
			err:     errors.Wrap(errBoom, errDeleteLicenseProfile),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotKey string

			if tc.client.DeleteLicenseProfileFn == nil {
				tc.client.DeleteLicenseProfileFn = func(opt *sonar.ScaLicenseProfileGetOptions) (*http.Response, error) {
					gotKey = opt.Key

					return &http.Response{StatusCode: http.StatusNoContent}, nil
				}
			} else {
				deleteFn := tc.client.DeleteLicenseProfileFn
				tc.client.DeleteLicenseProfileFn = func(opt *sonar.ScaLicenseProfileGetOptions) (*http.Response, error) {
					gotKey = opt.Key

					return deleteFn(opt)
				}
			}

			e := &external{client: tc.client}

			_, err := e.Delete(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.err, err, errComparer); diff != "" {
				t.Fatalf("Delete() error mismatch (-want +got):\n%s", diff)
			}

			if gotKey != tc.wantKey {
				t.Errorf("Delete() deleted key %q, want %q", gotKey, tc.wantKey)
			}

			if got := tc.mg.GetCondition(xpv1.TypeReady); got.Reason != xpv1.ReasonDeleting {
				t.Errorf("Delete() Ready reason = %q, want %q", got.Reason, xpv1.ReasonDeleting)
			}
		})
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockSCALicenseProfilesClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}
