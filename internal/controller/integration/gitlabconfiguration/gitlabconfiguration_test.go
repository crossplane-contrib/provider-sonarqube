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

package gitlabconfiguration

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

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
	// testResourceName is the name of the test GitLabConfiguration.
	testResourceName = "gitlab"
	// testExternalName is the configuration ID assigned by SonarQube.
	testExternalName = "gitlab-configuration"
	// testApplicationID is the GitLab application ID used in tests.
	testApplicationID = "gitlab-app-id"
	// testGitLabURL is the GitLab URL used in tests.
	testGitLabURL = "https://gitlab.example.com"
	// testCredentialsSecret is the Secret holding the GitLab credentials.
	testCredentialsSecret = "gitlab-credentials"
	// testConnectionSecret is the connection Secret of the resource.
	testConnectionSecret = "gitlab-connection"
	// testApplicationSecret is the application secret used in tests.
	testApplicationSecret = "application-secret-value"
	// testProvisioningToken is the provisioning token used in tests.
	testProvisioningToken = "provisioning-token-value"
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

// secretKeySelector creates a LocalSecretKeySelector on the credentials
// Secret.
func secretKeySelector(key string) *xpv1.LocalSecretKeySelector {
	return &xpv1.LocalSecretKeySelector{
		LocalSecretReference: xpv1.LocalSecretReference{Name: testCredentialsSecret},
		Key:                  key,
	}
}

// credentialsSecret creates the Secret holding the GitLab credentials.
func credentialsSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCredentialsSecret, Namespace: testNamespace},
		Data: map[string][]byte{
			"secret": []byte(testApplicationSecret),
			"token":  []byte(testProvisioningToken),
			"empty":  []byte(""),
		},
	}
}

// connectionSecret creates the connection Secret with the given data.
func connectionSecret(data map[string]string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testConnectionSecret, Namespace: testNamespace},
		Data:       map[string][]byte{},
	}

	for key, value := range data {
		secret.Data[key] = []byte(value)
	}

	return secret
}

// configurationOption customizes a test GitLabConfiguration.
type configurationOption func(*v1alpha1.GitLabConfiguration)

// withProvisioningToken references the provisioning token.
func withProvisioningToken() configurationOption {
	return func(cr *v1alpha1.GitLabConfiguration) {
		cr.Spec.ForProvider.ProvisioningType = v1alpha1.GitLabProvisioningTypeAuto
		cr.Spec.ForProvider.ProvisioningTokenSecretRef = secretKeySelector("token")
	}
}

// withApplicationSecretKey references another key of the credentials Secret.
func withApplicationSecretKey(key string) configurationOption {
	return func(cr *v1alpha1.GitLabConfiguration) {
		cr.Spec.ForProvider.ApplicationSecretRef = secretKeySelector(key)
	}
}

// newTestConfiguration creates a GitLabConfiguration for use in tests.
func newTestConfiguration(externalName string, options ...configurationOption) *v1alpha1.GitLabConfiguration {
	cr := &v1alpha1.GitLabConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testResourceName,
			Namespace:   testNamespace,
			Annotations: map[string]string{},
			UID:         types.UID("gitlab-uid"),
		},
		Spec: v1alpha1.GitLabConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: testConnectionSecret},
			},
			ForProvider: v1alpha1.GitLabConfigurationParameters{
				Enabled:              new(true),
				ApplicationID:        testApplicationID,
				URL:                  testGitLabURL,
				ApplicationSecretRef: secretKeySelector("secret"),
				SynchronizeGroups:    new(true),
				AllowedGroups:        []string{"group-a"},
				AllowAllGroups:       new(false),
				ProvisioningType:     v1alpha1.GitLabProvisioningTypeJIT,
				AllowUsersToSignUp:   new(false),
			},
		},
	}

	for _, option := range options {
		option(cr)
	}

	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}

	return cr
}

// observedConfiguration returns the SonarQube configuration matching
// newTestConfiguration.
func observedConfiguration() *sonar.DopTranslationGitlabConfiguration {
	return &sonar.DopTranslationGitlabConfiguration{
		Id:                 testExternalName,
		Enabled:            true,
		ApplicationId:      testApplicationID,
		Url:                testGitLabURL,
		SynchronizeGroups:  true,
		AllowedGroups:      []string{"group-a"},
		ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
		AllowUsersToSignUp: false,
	}
}

// buildKubeClient creates a fake Kubernetes client with the given objects.
func buildKubeClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
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

// TestObserve tests observing the GitLab configuration.
func TestObserve(t *testing.T) {
	t.Parallel()

	getReturns := func(configuration *sonar.DopTranslationGitlabConfiguration) *fake.MockGitLabConfigurationsClient {
		return &fake.MockGitLabConfigurationsClient{
			GetGitlabConfigurationFn: func(id string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
				if id != testExternalName {
					return nil, mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				}

				return configuration, mockHTTPResponse(http.StatusOK), nil
			},
		}
	}

	savedSecret := connectionSecret(map[string]string{connectionDetailApplicationSecretKey: testApplicationSecret})

	cases := map[string]struct {
		objects       []client.Object
		client        *fake.MockGitLabConfigurationsClient
		mg            *v1alpha1.GitLabConfiguration
		want          managed.ExternalObservation
		wantErr       string
		wantCondition *xpv1.Condition
	}{
		"MissingExternalName": {
			client: &fake.MockGitLabConfigurationsClient{},
			mg:     newTestConfiguration(""),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"NotFound": {
			client: getReturns(observedConfiguration()),
			mg:     newTestConfiguration(testResourceName),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"GetError": {
			client: &fake.MockGitLabConfigurationsClient{
				GetGitlabConfigurationFn: func(_ string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:      newTestConfiguration(testExternalName),
			wantErr: errGetConfiguration,
		},
		"UpToDate": {
			objects:       []client.Object{credentialsSecret(), savedSecret},
			client:        getReturns(observedConfiguration()),
			mg:            newTestConfiguration(testExternalName),
			want:          managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}},
			wantCondition: new(xpv1.Available()),
		},
		"LateInitialized": {
			objects: []client.Object{credentialsSecret(), savedSecret},
			client:  getReturns(observedConfiguration()),
			mg: newTestConfiguration(testExternalName, func(cr *v1alpha1.GitLabConfiguration) {
				cr.Spec.ForProvider.AllowUsersToSignUp = nil
			}),
			want: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true, ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"SpecDrift": {
			objects: []client.Object{credentialsSecret(), savedSecret},
			client: getReturns(func() *sonar.DopTranslationGitlabConfiguration {
				configuration := observedConfiguration()
				configuration.ApplicationId = "other"

				return configuration
			}()),
			mg:   newTestConfiguration(testExternalName),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}},
		},
		"ApplicationSecretNeverSaved": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}},
		},
		"ApplicationSecretRotated": {
			objects: []client.Object{
				credentialsSecret(),
				connectionSecret(map[string]string{connectionDetailApplicationSecretKey: "old"}),
			},
			client: getReturns(observedConfiguration()),
			mg:     newTestConfiguration(testExternalName),
			want:   managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}},
		},
		"ProvisioningTokenRotated": {
			objects: []client.Object{
				credentialsSecret(),
				connectionSecret(map[string]string{
					connectionDetailApplicationSecretKey: testApplicationSecret,
					connectionDetailProvisioningTokenKey: "old",
				}),
			},
			client: getReturns(func() *sonar.DopTranslationGitlabConfiguration {
				configuration := observedConfiguration()
				configuration.ProvisioningType = v1alpha1.GitLabProvisioningTypeAuto
				configuration.IsProvisioningTokenSet = true

				return configuration
			}()),
			mg:   newTestConfiguration(testExternalName, withProvisioningToken()),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}},
		},
		"ErrorMessageMakesUnavailable": {
			objects: []client.Object{credentialsSecret(), savedSecret},
			client: getReturns(func() *sonar.DopTranslationGitlabConfiguration {
				configuration := observedConfiguration()
				configuration.ErrorMessage = new("GitLab is unreachable")

				return configuration
			}()),
			mg:            newTestConfiguration(testExternalName),
			want:          managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}},
			wantCondition: new(xpv1.Unavailable().WithMessage("GitLab is unreachable")),
		},
		"MissingApplicationSecret": {
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errGetApplicationSecret,
		},
		"EmptyApplicationSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName, withApplicationSecretKey("empty")),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errEmptyApplicationSecret,
		},
		"MissingProvisioningToken": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedConfiguration()),
			mg: newTestConfiguration(testExternalName, func(cr *v1alpha1.GitLabConfiguration) {
				cr.Spec.ForProvider.ProvisioningTokenSecretRef = secretKeySelector("missing")
			}),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errGetProvisioningToken,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}

			got, err := e.Observe(context.Background(), tc.mg)

			checkError(t, "Observe", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() -want, +got:\n%s", diff)
			}

			if tc.wantCondition != nil {
				gotCondition := tc.mg.GetCondition(tc.wantCondition.Type)
				if !gotCondition.Equal(*tc.wantCondition) {
					t.Errorf("Observe() condition = %+v, want %+v", gotCondition, *tc.wantCondition)
				}
			}
		})
	}
}

// TestObserveSavedCredentialsError tests that a failure to read the
// connection secret, other than its absence, is reported.
func TestObserveSavedCredentialsError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(credentialsSecret()).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if key.Name == testConnectionSecret {
				return errors.New("kube unavailable")
			}

			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()

	e := &external{kubeClient: kube, client: &fake.MockGitLabConfigurationsClient{
		GetGitlabConfigurationFn: func(_ string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
			return observedConfiguration(), mockHTTPResponse(http.StatusOK), nil
		},
	}}

	_, err := e.Observe(context.Background(), newTestConfiguration(testExternalName))

	checkError(t, "Observe", errGetSavedCredentials, err)
}

// TestCreate tests creating the GitLab configuration.
func TestCreate(t *testing.T) {
	t.Parallel()

	emptySearch := func() (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error) {
		return &sonar.DopTranslationGitlabConfigurationsSearch{}, mockHTTPResponse(http.StatusOK), nil
	}

	cases := map[string]struct {
		objects          []client.Object
		client           *fake.MockGitLabConfigurationsClient
		mg               *v1alpha1.GitLabConfiguration
		want             managed.ExternalCreation
		wantErr          string
		wantExternalName string
		wantOptions      *sonar.DopTranslationGitlabConfigurationCreateOptions
	}{
		"MissingApplicationSecret": {
			client:           &fake.MockGitLabConfigurationsClient{},
			mg:               newTestConfiguration(testResourceName),
			wantErr:          errGetApplicationSecret,
			wantExternalName: testResourceName,
		},
		"SearchError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: func() (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:               newTestConfiguration(testResourceName),
			wantErr:          errSearchConfigurations,
			wantExternalName: testResourceName,
		},
		"ConfigurationAlreadyExists": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: func() (*sonar.DopTranslationGitlabConfigurationsSearch, *http.Response, error) {
					return &sonar.DopTranslationGitlabConfigurationsSearch{
						GitlabConfigurations: []sonar.DopTranslationGitlabConfiguration{{Id: testExternalName}},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg:               newTestConfiguration(testResourceName),
			wantErr:          `already has a GitLab configuration (id "gitlab-configuration")`,
			wantExternalName: testResourceName,
		},
		"CreateError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: emptySearch,
				CreateGitlabConfigurationFn: func(_ *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("bad request")
				},
			},
			mg:               newTestConfiguration(testResourceName),
			wantErr:          errCreateConfiguration,
			wantExternalName: testResourceName,
		},
		"MissingID": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: emptySearch,
				CreateGitlabConfigurationFn: func(_ *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					return &sonar.DopTranslationGitlabConfiguration{}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg:               newTestConfiguration(testResourceName),
			wantErr:          errNoConfigurationID,
			wantExternalName: testResourceName,
		},
		"Success": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: emptySearch,
			},
			mg: newTestConfiguration(testResourceName),
			want: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailApplicationSecretKey: []byte(testApplicationSecret),
			}},
			wantExternalName: testExternalName,
			wantOptions: &sonar.DopTranslationGitlabConfigurationCreateOptions{
				Enabled:            true,
				ApplicationId:      testApplicationID,
				Url:                testGitLabURL,
				Secret:             testApplicationSecret,
				SynchronizeGroups:  true,
				AllowedGroups:      []string{"group-a"},
				AllowAllGroups:     new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeJIT,
				AllowUsersToSignUp: new(false),
			},
		},
		"SuccessWithProvisioningToken": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				SearchGitlabConfigurationsFn: emptySearch,
			},
			mg: newTestConfiguration(testResourceName, withProvisioningToken()),
			want: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailApplicationSecretKey: []byte(testApplicationSecret),
				connectionDetailProvisioningTokenKey: []byte(testProvisioningToken),
			}},
			wantExternalName: testExternalName,
			wantOptions: &sonar.DopTranslationGitlabConfigurationCreateOptions{
				Enabled:            true,
				ApplicationId:      testApplicationID,
				Url:                testGitLabURL,
				Secret:             testApplicationSecret,
				SynchronizeGroups:  true,
				AllowedGroups:      []string{"group-a"},
				AllowAllGroups:     new(false),
				ProvisioningType:   v1alpha1.GitLabProvisioningTypeAuto,
				ProvisioningToken:  testProvisioningToken,
				AllowUsersToSignUp: new(false),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotOptions *sonar.DopTranslationGitlabConfigurationCreateOptions

			if tc.client.CreateGitlabConfigurationFn == nil {
				tc.client.CreateGitlabConfigurationFn = func(opt *sonar.DopTranslationGitlabConfigurationCreateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					gotOptions = opt

					return &sonar.DopTranslationGitlabConfiguration{Id: testExternalName}, mockHTTPResponse(http.StatusOK), nil
				}
			}

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}

			got, err := e.Create(context.Background(), tc.mg)

			checkError(t, "Create", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Create() -want, +got:\n%s", diff)
			}

			if gotExternalName := meta.GetExternalName(tc.mg); gotExternalName != tc.wantExternalName {
				t.Errorf("Create() external name = %q, want %q", gotExternalName, tc.wantExternalName)
			}

			if diff := cmp.Diff(tc.wantOptions, gotOptions); diff != "" {
				t.Errorf("Create() options -want, +got:\n%s", diff)
			}
		})
	}
}

// TestUpdate tests updating the GitLab configuration.
func TestUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		objects   []client.Object
		client    *fake.MockGitLabConfigurationsClient
		mg        *v1alpha1.GitLabConfiguration
		want      managed.ExternalUpdate
		wantErr   string
		wantID    string
		wantToken string
	}{
		"MissingExternalName": {
			client:  &fake.MockGitLabConfigurationsClient{},
			mg:      newTestConfiguration(""),
			wantErr: "external name is not set",
		},
		"MissingProvisioningToken": {
			objects: []client.Object{credentialsSecret()},
			client:  &fake.MockGitLabConfigurationsClient{},
			mg: newTestConfiguration(testExternalName, func(cr *v1alpha1.GitLabConfiguration) {
				cr.Spec.ForProvider.ProvisioningTokenSecretRef = secretKeySelector("missing")
			}),
			wantErr: errGetProvisioningToken,
		},
		"UpdateError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockGitLabConfigurationsClient{
				UpdateGitlabConfigurationFn: func(_ string, _ *sonar.DopTranslationGitlabConfigurationUpdateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("bad request")
				},
			},
			mg:      newTestConfiguration(testExternalName),
			wantErr: errUpdateConfiguration,
		},
		"Success": {
			objects: []client.Object{credentialsSecret()},
			client:  &fake.MockGitLabConfigurationsClient{},
			mg:      newTestConfiguration(testExternalName, withProvisioningToken()),
			want: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailApplicationSecretKey: []byte(testApplicationSecret),
				connectionDetailProvisioningTokenKey: []byte(testProvisioningToken),
			}},
			wantID:    testExternalName,
			wantToken: testProvisioningToken,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				gotID      string
				gotOptions *sonar.DopTranslationGitlabConfigurationUpdateOptions
			)

			if tc.client.UpdateGitlabConfigurationFn == nil {
				tc.client.UpdateGitlabConfigurationFn = func(id string, opt *sonar.DopTranslationGitlabConfigurationUpdateOptions) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
					gotID, gotOptions = id, opt

					return observedConfiguration(), mockHTTPResponse(http.StatusOK), nil
				}
			}

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}

			got, err := e.Update(context.Background(), tc.mg)

			checkError(t, "Update", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Update() -want, +got:\n%s", diff)
			}

			if gotID != tc.wantID {
				t.Errorf("Update() id = %q, want %q", gotID, tc.wantID)
			}

			if tc.wantToken != "" && (gotOptions == nil || gotOptions.ProvisioningToken != tc.wantToken || gotOptions.Secret != testApplicationSecret) {
				t.Errorf("Update() options = %+v, want the application secret and provisioning token", gotOptions)
			}
		})
	}
}

// TestDelete tests deleting the GitLab configuration.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockGitLabConfigurationsClient
		mg      *v1alpha1.GitLabConfiguration
		wantErr string
	}{
		"MissingExternalName": {
			client: &fake.MockGitLabConfigurationsClient{},
			mg:     newTestConfiguration(""),
		},
		"Success": {
			client: &fake.MockGitLabConfigurationsClient{
				DeleteGitlabConfigurationFn: func(id string) (*http.Response, error) {
					if id != testExternalName {
						return nil, errors.Errorf("unexpected id %q", id)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			mg: newTestConfiguration(testExternalName),
		},
		"AlreadyDeleted": {
			client: &fake.MockGitLabConfigurationsClient{
				DeleteGitlabConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: SonarQube returns the response along with the error.
					return mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				},
			},
			mg: newTestConfiguration(testExternalName),
		},
		"DeleteError": {
			client: &fake.MockGitLabConfigurationsClient{
				DeleteGitlabConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: SonarQube returns the response along with the error.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:      newTestConfiguration(testExternalName),
			wantErr: errDeleteConfiguration,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(), client: tc.client}

			_, err := e.Delete(context.Background(), tc.mg)

			checkError(t, "Delete", tc.wantErr, err)

			if got := tc.mg.GetCondition(xpv1.TypeReady); got.Reason != xpv1.ReasonDeleting {
				t.Errorf("Delete() Ready reason = %q, want %q", got.Reason, xpv1.ReasonDeleting)
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
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnect tests building the external client.
func TestConnect(t *testing.T) {
	t.Parallel()

	newScheme := func(t *testing.T) *runtime.Scheme {
		t.Helper()

		scheme := runtime.NewScheme()

		for _, addToScheme := range []func(*runtime.Scheme) error{
			apisv1alpha1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme, corev1.AddToScheme,
		} {
			err := addToScheme(scheme)
			if err != nil {
				t.Fatalf("AddToScheme() unexpected error: %v", err)
			}
		}

		return scheme
	}

	t.Run("TrackError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), newTestConfiguration(""))

		checkError(t, "Connect", errTrackPCUsage, err)
	})

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		cr := newTestConfiguration("")
		cr.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

		_, err := c.Connect(context.Background(), cr)

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
		providerSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "prov-secret", Namespace: testNamespace},
			Data:       map[string][]byte{"token": []byte("test-token")},
		}

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(providerConfig, providerSecret).Build()

		cr := newTestConfiguration("")
		cr.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(_ common.Config) integration.GitLabConfigurationsClient {
				return &fake.MockGitLabConfigurationsClient{}
			},
		}

		got, err := c.Connect(context.Background(), cr)
		if err != nil {
			t.Fatalf("Connect() unexpected error: %v", err)
		}

		if got == nil {
			t.Fatal("Connect() returned nil client")
		}
	})
}

// TestSetupGatedRegistersGitLabConfigurationGVK verifies that SetupGated
// registers the GitLabConfiguration GVK with the gate.
func TestSetupGatedRegistersGitLabConfigurationGVK(t *testing.T) {
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

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.GitLabConfigurationGroupVersionKind {
		t.Fatalf("SetupGated() registered GVKs %v, want [%v]", g.gvks, v1alpha1.GitLabConfigurationGroupVersionKind)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}

// TestObserveDeletedWithoutSecrets checks that missing Secrets do not block
// deletion.
func TestObserveDeletedWithoutSecrets(t *testing.T) {
	t.Parallel()

	configuration := newTestConfiguration(testExternalName)
	now := metav1.Now()
	configuration.DeletionTimestamp = &now

	e := &external{kubeClient: buildKubeClient(), client: &fake.MockGitLabConfigurationsClient{
		GetGitlabConfigurationFn: func(_ string) (*sonar.DopTranslationGitlabConfiguration, *http.Response, error) {
			return observedConfiguration(), mockHTTPResponse(http.StatusOK), nil
		},
	}}

	got, err := e.Observe(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Observe(...): unexpected error: %v", err)
	}

	if !got.ResourceExists {
		t.Error("Observe(...): the configuration must still be reported as existing")
	}
}
