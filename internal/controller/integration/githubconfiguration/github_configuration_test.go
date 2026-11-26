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

package githubconfiguration

import (
	"context"
	"net"
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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testNamespace is the namespace of the test resources.
	testNamespace = "default"
	// testConfigurationID is the GitHub configuration ID used in tests.
	testConfigurationID = "github-configuration"
	// testAppID is the GitHub App ID used in tests.
	testAppID = "123456"
	// testClientID is the GitHub App client ID used in tests.
	testClientID = "Iv1.abc123"
	// testClientSecret is the GitHub App client secret used in tests.
	testClientSecret = "supersecretclientvalue123"
	// testPrivateKey is the GitHub App private key used in tests.
	testPrivateKey = "supersecretprivatekey456"
	// testAPIURL is the GitHub API URL used in tests.
	testAPIURL = "https://api.github.com/"
	// testWebURL is the GitHub web URL used in tests.
	testWebURL = "https://github.com/"
	// testClientSecretName is the name of the client secret Secret.
	testClientSecretName = "github-client-secret"
	// testPrivateKeyName is the name of the private key Secret.
	testPrivateKeyName = "github-private-key"
	// testConnectionSecretName is the name of the connection Secret.
	testConnectionSecretName = "github-connection"
	// testSecretKey is the key of the referenced Secrets.
	testSecretKey = "value"
)

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

// testKubeSecret creates a Kubernetes Secret with the given data.
func testKubeSecret(name string, data map[string]string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Data:       map[string][]byte{},
	}

	for key, value := range data {
		secret.Data[key] = []byte(value)
	}

	return secret
}

// testCredentialSecrets returns the Secrets referenced by the test
// GitHubConfiguration.
func testCredentialSecrets() []runtime.Object {
	return []runtime.Object{
		testKubeSecret(testClientSecretName, map[string]string{testSecretKey: testClientSecret}),
		testKubeSecret(testPrivateKeyName, map[string]string{testSecretKey: testPrivateKey}),
	}
}

// testSavedConnectionSecret returns a connection Secret holding the given
// client secret, and the test client ID and private key.
func testSavedConnectionSecret(savedClientSecret string) *corev1.Secret {
	return testKubeSecret(testConnectionSecretName, map[string]string{
		connectionDetailClientIDKey:     testClientID,
		connectionDetailClientSecretKey: savedClientSecret,
		connectionDetailPrivateKeyKey:   testPrivateKey,
	})
}

// newTestGitHubConfiguration creates a GitHubConfiguration resource for use
// in tests.
func newTestGitHubConfiguration(externalName string) *v1alpha1.GitHubConfiguration {
	configuration := &v1alpha1.GitHubConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-github-configuration",
			Namespace:   testNamespace,
			Annotations: map[string]string{},
			UID:         types.UID("github-configuration-uid"),
		},
		Spec: v1alpha1.GitHubConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: testConnectionSecretName},
			},
			ForProvider: v1alpha1.GitHubConfigurationParameters{
				Enabled:       new(true),
				ApplicationID: testAppID,
				ClientID:      testClientID,
				ClientSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: testClientSecretName},
					Key:                  testSecretKey,
				},
				PrivateKeyRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: testPrivateKeyName},
					Key:                  testSecretKey,
				},
				APIURL:                          testAPIURL,
				WebURL:                          testWebURL,
				AllowedOrganizations:            []string{"org-a"},
				ProvisioningType:                v1alpha1.GitHubProvisioningTypeJIT,
				SynchronizeGroups:               new(false),
				AllowUsersToSignUp:              new(true),
				ProjectVisibility:               new(true),
				UserConsentRequiredAfterUpgrade: new(false),
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(configuration, externalName)
	}

	return configuration
}

// testSonarConfiguration returns the SonarQube GitHub configuration matching
// newTestGitHubConfiguration.
func testSonarConfiguration() sonar.DopTranslationGithubConfiguration {
	return sonar.DopTranslationGithubConfiguration{
		Id:            testConfigurationID,
		Enabled:       true,
		ApplicationId: testAppID,
		ApiUrl:        testAPIURL,
		WebUrl:        testWebURL,
		AllowedOrganizations: []string{
			"org-a",
		},
		ProvisioningType:   v1alpha1.GitHubProvisioningTypeJIT,
		AllowUsersToSignUp: true,
		ProjectVisibility:  true,
	}
}

// searchClient returns a client whose search returns the given
// configurations.
func searchClient(configurations ...sonar.DopTranslationGithubConfiguration) *fake.MockGitHubConfigurationsClient {
	return &fake.MockGitHubConfigurationsClient{
		SearchGithubConfigurationsFn: func() (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error) {
			return &sonar.DopTranslationGithubConfigurationsSearch{GithubConfigurations: configurations}, mockHTTPResponse(http.StatusOK), nil
		},
	}
}

// buildKubeClient creates a fake Kubernetes client with the given objects.
func buildKubeClient(objects []runtime.Object) *fakekube.ClientBuilder {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return fakekube.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...)
}

// checkError verifies that an error matches (or does not match) expectations.
func checkError(t *testing.T, method, wantErrSubstr string, gotErr error) {
	t.Helper()

	switch {
	case wantErrSubstr == "" && gotErr != nil:
		t.Errorf("%s() unexpected error: %v", method, gotErr)
	case wantErrSubstr != "" && gotErr == nil:
		t.Errorf("%s() expected error containing %q, got nil", method, wantErrSubstr)
	case wantErrSubstr != "" && !strings.Contains(gotErr.Error(), wantErrSubstr):
		t.Errorf("%s() error = %q, want substring %q", method, gotErr.Error(), wantErrSubstr)
	}
}

// TestObserve tests observing the GitHub configuration.
func TestObserve(t *testing.T) {
	t.Parallel()

	withSecrets := func(extra ...runtime.Object) []runtime.Object {
		return append(testCredentialSecrets(), extra...)
	}

	cases := map[string]struct {
		objects          []runtime.Object
		client           *fake.MockGitHubConfigurationsClient
		mg               *v1alpha1.GitHubConfiguration
		wantObservation  managed.ExternalObservation
		wantErrSubstr    string
		wantExternalName string
	}{
		"SearchErrorReturnsError": {
			objects: withSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				SearchGithubConfigurationsFn: func() (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			mg:               newTestGitHubConfiguration(testConfigurationID),
			wantErrSubstr:    errSearchConfigurations,
			wantExternalName: testConfigurationID,
		},
		"NilSearchReturnsNotExists": {
			objects: withSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				SearchGithubConfigurationsFn: func() (*sonar.DopTranslationGithubConfigurationsSearch, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg:              newTestGitHubConfiguration(""),
			wantObservation: managed.ExternalObservation{ResourceExists: false},
		},
		"NoConfigurationReturnsNotExists": {
			objects:         withSecrets(),
			client:          searchClient(),
			mg:              newTestGitHubConfiguration(""),
			wantObservation: managed.ExternalObservation{ResourceExists: false},
		},
		"OtherConfigurationIDReturnsNotExists": {
			objects:          withSecrets(),
			client:           searchClient(testSonarConfiguration()),
			mg:               newTestGitHubConfiguration("other-id"),
			wantObservation:  managed.ExternalObservation{ResourceExists: false},
			wantExternalName: "other-id",
		},
		"ExistsAndUpToDate": {
			objects: withSecrets(testSavedConnectionSecret(testClientSecret)),
			client:  searchClient(testSonarConfiguration()),
			mg:      newTestGitHubConfiguration(testConfigurationID),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"AdoptsExistingConfiguration": {
			objects: withSecrets(testSavedConnectionSecret(testClientSecret)),
			client:  searchClient(testSonarConfiguration()),
			mg:      newTestGitHubConfiguration(""),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"FieldDriftNotUpToDate": {
			objects: withSecrets(testSavedConnectionSecret(testClientSecret)),
			client: searchClient(func() sonar.DopTranslationGithubConfiguration {
				configuration := testSonarConfiguration()
				configuration.AllowedOrganizations = []string{"org-b"}

				return configuration
			}()),
			mg: newTestGitHubConfiguration(testConfigurationID),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"ClientSecretRotatedNotUpToDate": {
			objects: withSecrets(testSavedConnectionSecret("old-client-secret")),
			client:  searchClient(testSonarConfiguration()),
			mg:      newTestGitHubConfiguration(testConfigurationID),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"MissingConnectionSecretNotUpToDate": {
			objects: withSecrets(),
			client:  searchClient(testSonarConfiguration()),
			mg:      newTestGitHubConfiguration(testConfigurationID),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"NoConnectionSecretRefNotUpToDate": {
			objects: withSecrets(),
			client:  searchClient(testSonarConfiguration()),
			mg: func() *v1alpha1.GitHubConfiguration {
				configuration := newTestGitHubConfiguration(testConfigurationID)
				configuration.SetWriteConnectionSecretToReference(nil)

				return configuration
			}(),
			wantObservation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{},
			},
			wantExternalName: testConfigurationID,
		},
		"MissingClientSecretReturnsError": {
			objects:          []runtime.Object{testKubeSecret(testPrivateKeyName, map[string]string{testSecretKey: testPrivateKey})},
			client:           searchClient(testSonarConfiguration()),
			mg:               newTestGitHubConfiguration(testConfigurationID),
			wantErrSubstr:    "cannot get client secret from secret reference",
			wantExternalName: testConfigurationID,
		},
		"EmptyPrivateKeyReturnsError": {
			objects: []runtime.Object{
				testKubeSecret(testClientSecretName, map[string]string{testSecretKey: testClientSecret}),
				testKubeSecret(testPrivateKeyName, map[string]string{testSecretKey: ""}),
			},
			client:           searchClient(testSonarConfiguration()),
			mg:               newTestGitHubConfiguration(testConfigurationID),
			wantErrSubstr:    "private key is empty",
			wantExternalName: testConfigurationID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects).Build(), client: tc.client}

			got, err := e.Observe(context.Background(), tc.mg)
			checkError(t, "Observe", tc.wantErrSubstr, err)

			if tc.wantErrSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.wantObservation, got); diff != "" {
				t.Errorf("Observe() -want, +got:\n%s", diff)
			}

			if externalName := meta.GetExternalName(tc.mg); externalName != tc.wantExternalName {
				t.Errorf("Observe() external name = %q, want %q", externalName, tc.wantExternalName)
			}
		})
	}
}

// TestObserveLateInitializes tests that Observe fills the unset optional
// fields from SonarQube and reports the observed state.
func TestObserveLateInitializes(t *testing.T) {
	t.Parallel()

	sonarConfiguration := testSonarConfiguration()
	sonarConfiguration.ErrorMessage = new("invalid GitHub App")

	configuration := newTestGitHubConfiguration(testConfigurationID)
	configuration.Spec.ForProvider.SynchronizeGroups = nil
	configuration.Spec.ForProvider.AllowedOrganizations = nil

	e := &external{
		kubeClient: buildKubeClient(append(testCredentialSecrets(), testSavedConnectionSecret(testClientSecret))).Build(),
		client:     searchClient(sonarConfiguration),
	}

	got, err := e.Observe(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !got.ResourceLateInitialized || !got.ResourceUpToDate {
		t.Errorf("Observe() = %+v, want late-initialized and up to date", got)
	}

	if !cmp.Equal(configuration.Spec.ForProvider.AllowedOrganizations, []string{"org-a"}) {
		t.Errorf("Observe() AllowedOrganizations = %v, want [org-a]", configuration.Spec.ForProvider.AllowedOrganizations)
	}

	if configuration.Spec.ForProvider.SynchronizeGroups == nil || *configuration.Spec.ForProvider.SynchronizeGroups {
		t.Errorf("Observe() SynchronizeGroups = %v, want false", configuration.Spec.ForProvider.SynchronizeGroups)
	}

	if configuration.Status.AtProvider.ErrorMessage != "invalid GitHub App" {
		t.Errorf("Observe() ErrorMessage = %q, want %q", configuration.Status.AtProvider.ErrorMessage, "invalid GitHub App")
	}

	if condition := configuration.GetCondition(xpv1.TypeReady); condition.Reason != xpv1.ReasonAvailable {
		t.Errorf("Observe() Ready reason = %q, want %q", condition.Reason, xpv1.ReasonAvailable)
	}
}

// TestCreate tests creating the GitHub configuration.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		objects          []runtime.Object
		client           *fake.MockGitHubConfigurationsClient
		wantCreation     managed.ExternalCreation
		wantErrSubstr    string
		wantExternalName string
	}{
		"Success": {
			objects: testCredentialSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				CreateGithubConfigurationFn: func(opt *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
					if opt.ClientId != testClientID || opt.ClientSecret != testClientSecret || opt.PrivateKey != testPrivateKey {
						return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("unexpected credentials")
					}

					return &sonar.DopTranslationGithubConfiguration{Id: testConfigurationID}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			wantCreation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientIDKey:     []byte(testClientID),
				connectionDetailClientSecretKey: []byte(testClientSecret),
				connectionDetailPrivateKeyKey:   []byte(testPrivateKey),
			}},
			wantExternalName: testConfigurationID,
		},
		"MissingSecretReturnsError": {
			objects:       []runtime.Object{},
			client:        &fake.MockGitHubConfigurationsClient{},
			wantErrSubstr: "cannot get client secret from secret reference",
		},
		"APIErrorReturnsError": {
			objects: testCredentialSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				CreateGithubConfigurationFn: func(_ *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("GitHub configuration already exists")
				},
			},
			wantErrSubstr: errCreateConfiguration,
		},
		"MissingIDReturnsError": {
			objects: testCredentialSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				CreateGithubConfigurationFn: func(_ *sonar.DopTranslationGithubConfigurationCreateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
					return &sonar.DopTranslationGithubConfiguration{}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			wantErrSubstr: errNoConfigurationID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			configuration := newTestGitHubConfiguration("")
			e := &external{kubeClient: buildKubeClient(tc.objects).Build(), client: tc.client}

			got, err := e.Create(context.Background(), configuration)
			checkError(t, "Create", tc.wantErrSubstr, err)

			if tc.wantErrSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.wantCreation, got); diff != "" {
				t.Errorf("Create() -want, +got:\n%s", diff)
			}

			if externalName := meta.GetExternalName(configuration); externalName != tc.wantExternalName {
				t.Errorf("Create() external name = %q, want %q", externalName, tc.wantExternalName)
			}
		})
	}
}

// TestUpdate tests updating the GitHub configuration.
func TestUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		objects       []runtime.Object
		client        *fake.MockGitHubConfigurationsClient
		mg            *v1alpha1.GitHubConfiguration
		wantUpdate    managed.ExternalUpdate
		wantErrSubstr string
	}{
		"MissingExternalNameReturnsError": {
			objects:       testCredentialSecrets(),
			client:        &fake.MockGitHubConfigurationsClient{},
			mg:            newTestGitHubConfiguration(""),
			wantErrSubstr: "external name is not set",
		},
		"MissingSecretReturnsError": {
			objects:       []runtime.Object{testKubeSecret(testClientSecretName, map[string]string{testSecretKey: testClientSecret})},
			client:        &fake.MockGitHubConfigurationsClient{},
			mg:            newTestGitHubConfiguration(testConfigurationID),
			wantErrSubstr: "cannot get private key from secret reference",
		},
		"APIErrorReturnsError": {
			objects: testCredentialSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				UpdateGithubConfigurationFn: func(_ string, _ *sonar.DopTranslationGithubConfigurationUpdateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("api error")
				},
			},
			mg:            newTestGitHubConfiguration(testConfigurationID),
			wantErrSubstr: errUpdateConfiguration,
		},
		"Success": {
			objects: testCredentialSecrets(),
			client: &fake.MockGitHubConfigurationsClient{
				UpdateGithubConfigurationFn: func(id string, opt *sonar.DopTranslationGithubConfigurationUpdateOptions) (*sonar.DopTranslationGithubConfiguration, *http.Response, error) {
					if id != testConfigurationID {
						return nil, mockHTTPResponse(http.StatusNotFound), errors.Errorf("unexpected id %q", id)
					}

					if opt.PrivateKey != testPrivateKey || opt.AllowedOrganizations == nil || !cmp.Equal(opt.AllowedOrganizations.Value, []string{"org-a"}) {
						return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("unexpected options")
					}

					return &sonar.DopTranslationGithubConfiguration{Id: id}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg: newTestGitHubConfiguration(testConfigurationID),
			wantUpdate: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailClientIDKey:     []byte(testClientID),
				connectionDetailClientSecretKey: []byte(testClientSecret),
				connectionDetailPrivateKeyKey:   []byte(testPrivateKey),
			}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects).Build(), client: tc.client}

			got, err := e.Update(context.Background(), tc.mg)
			checkError(t, "Update", tc.wantErrSubstr, err)

			if tc.wantErrSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.wantUpdate, got); diff != "" {
				t.Errorf("Update() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestDelete tests deleting the GitHub configuration.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client        *fake.MockGitHubConfigurationsClient
		externalName  string
		wantErrSubstr string
	}{
		"MissingExternalNameIsNoOp": {
			client:       &fake.MockGitHubConfigurationsClient{},
			externalName: "",
		},
		"Success": {
			client: &fake.MockGitHubConfigurationsClient{
				DeleteGithubConfigurationFn: func(id string) (*http.Response, error) {
					if id != testConfigurationID {
						//nolint:nilnil // Intentional: simulating an HTTP failure.
						return mockHTTPResponse(http.StatusNotFound), errors.Errorf("unexpected id %q", id)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			externalName: testConfigurationID,
		},
		"NotFoundIsSuccess": {
			client: &fake.MockGitHubConfigurationsClient{
				DeleteGithubConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating an HTTP failure.
					return mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				},
			},
			externalName: testConfigurationID,
		},
		"APIErrorReturnsError": {
			client: &fake.MockGitHubConfigurationsClient{
				DeleteGithubConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating an HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			externalName:  testConfigurationID,
			wantErrSubstr: errDeleteConfiguration,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			configuration := newTestGitHubConfiguration(tc.externalName)
			e := &external{kubeClient: buildKubeClient(nil).Build(), client: tc.client}

			_, err := e.Delete(context.Background(), configuration)
			checkError(t, "Delete", tc.wantErrSubstr, err)

			if condition := configuration.GetCondition(xpv1.TypeReady); condition.Reason != xpv1.ReasonDeleting {
				t.Errorf("Delete() Ready reason = %q, want %q", condition.Reason, xpv1.ReasonDeleting)
			}
		})
	}
}

// TestDisconnect tests that Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// newTestScheme returns a scheme with the provider and core types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	for _, addToScheme := range []func(*runtime.Scheme) error{
		apisv1alpha1.SchemeBuilder.AddToScheme,
		v1alpha1.SchemeBuilder.AddToScheme,
		corev1.AddToScheme,
	} {
		err := addToScheme(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() unexpected error: %v", err)
		}
	}

	return scheme
}

// TestConnect tests connecting to SonarQube.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TrackUsageError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err := c.Connect(context.Background(), newTestGitHubConfiguration(""))

		checkError(t, "Connect", errTrackPCUsage, err)
	})

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

		configuration := newTestGitHubConfiguration("")
		configuration.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
		}

		_, err := c.Connect(context.Background(), configuration)

		checkError(t, "Connect", errGetPC, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		providerConfig := &apisv1alpha1.ProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: testNamespace},
			Spec: apisv1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &apisv1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{Name: "prov-secret", Namespace: testNamespace},
							Key:             "token",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
		}

		providerSecret := testKubeSecret("prov-secret", map[string]string{"token": "test-token"})

		kube := fakekube.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(providerConfig, providerSecret).Build()

		configuration := newTestGitHubConfiguration("")
		configuration.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(_ common.Config) integration.GitHubConfigurationsClient {
				return &fake.MockGitHubConfigurationsClient{}
			},
		}

		got, err := c.Connect(context.Background(), configuration)
		if err != nil {
			t.Fatalf("Connect() unexpected error: %v", err)
		}

		if got == nil {
			t.Fatal("Connect() returned nil client")
		}
	})
}

// TestSetupGatedRegistersGitHubConfigurationGVK tests that SetupGated
// registers the GitHubConfiguration GVK.
func TestSetupGatedRegistersGitHubConfigurationGVK(t *testing.T) {
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

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.GitHubConfigurationGroupVersionKind {
		t.Fatalf("SetupGated() GVKs = %v, want [%v]", g.gvks, v1alpha1.GitHubConfigurationGroupVersionKind)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails tests that the gated callback
// panics when the controller setup fails.
func TestSetupGatedCallbackPanicsWhenSetupFails(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}

// TestTransportErrorDoesNotLeakSecrets verifies that a transport error while
// sending the GitHub App credentials does not put them in the returned
// error, which crossplane-runtime publishes to conditions, events and logs.
func TestTransportErrorDoesNotLeakSecrets(t *testing.T) {
	t.Parallel()

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() unexpected error: %v", err)
	}

	unreachableURL := "http://" + listener.Addr().String() + "/api"
	_ = listener.Close()

	configurationsClient := integration.NewGitHubConfigurationsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "dummy-token",
		BaseURL:  unreachableURL,
	})
	e := &external{kubeClient: buildKubeClient(testCredentialSecrets()).Build(), client: configurationsClient}

	_, createErr := e.Create(context.Background(), newTestGitHubConfiguration(""))
	_, updateErr := e.Update(context.Background(), newTestGitHubConfiguration(testConfigurationID))

	for operation, gotErr := range map[string]error{"Create": createErr, "Update": updateErr} {
		if gotErr == nil {
			t.Fatalf("%s() expected a transport error, got nil", operation)
		}

		for _, secret := range []string{testClientSecret, testPrivateKey} {
			if strings.Contains(gotErr.Error(), secret) {
				t.Errorf("%s() error leaks a GitHub App credential: %q", operation, gotErr.Error())
			}
		}
	}
}

// TestObserveDeletedWithoutSecrets checks that missing Secrets do not block
// deletion.
func TestObserveDeletedWithoutSecrets(t *testing.T) {
	t.Parallel()

	configuration := newTestGitHubConfiguration(testConfigurationID)
	now := metav1.Now()
	configuration.DeletionTimestamp = &now

	e := &external{kubeClient: buildKubeClient(nil).Build(), client: searchClient(testSonarConfiguration())}

	got, err := e.Observe(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Observe(...): unexpected error: %v", err)
	}

	if !got.ResourceExists {
		t.Error("Observe(...): the configuration must still be reported as existing")
	}
}
