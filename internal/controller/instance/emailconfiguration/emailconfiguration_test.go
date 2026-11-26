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

package emailconfiguration

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testNamespace is the namespace of the test resources.
	testNamespace = "default"
	// testResourceName is the name of the test EmailConfiguration.
	testResourceName = "email"
	// testExternalName is the configuration ID assigned by SonarQube.
	testExternalName = "email-configuration-id"
	// testCredentialsSecret is the Secret holding the SMTP credentials.
	testCredentialsSecret = "email-credentials"
	// testConnectionSecret is the connection Secret of the resource.
	testConnectionSecret = "email-connection"
	// testPassword is the SMTP password used in tests.
	testPassword = "password-value"
	// testClientID is the OAuth client ID used in tests.
	testClientID = "client-id-value"
	// testClientSecret is the OAuth client secret used in tests.
	testClientSecret = "client-secret-value"
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

// credentialsSecret creates the Secret holding the SMTP credentials.
func credentialsSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCredentialsSecret, Namespace: testNamespace},
		Data: map[string][]byte{
			"password":     []byte(testPassword),
			"clientId":     []byte(testClientID),
			"clientSecret": []byte(testClientSecret),
			"empty":        []byte(""),
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

// configurationOption customizes a test EmailConfiguration.
type configurationOption func(*v1alpha1.EmailConfiguration)

// withOAuth selects OAUTH authentication.
func withOAuth() configurationOption {
	return func(cr *v1alpha1.EmailConfiguration) {
		forProvider := &cr.Spec.ForProvider
		forProvider.AuthMethod = v1alpha1.EmailAuthMethodOAuth
		forProvider.BasicPasswordSecretRef = nil
		forProvider.OAuthAuthenticationHost = new("https://login.example.com")
		forProvider.OAuthTenant = new("tenant")
		forProvider.OAuthScope = new("scope")
		forProvider.OAuthClientIDSecretRef = secretKeySelector("clientId")
		forProvider.OAuthClientSecretSecretRef = secretKeySelector("clientSecret")
	}
}

// newTestConfiguration creates an EmailConfiguration using BASIC
// authentication.
func newTestConfiguration(externalName string, options ...configurationOption) *v1alpha1.EmailConfiguration {
	cr := &v1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testResourceName,
			Namespace:   testNamespace,
			Annotations: map[string]string{},
			UID:         types.UID("email-uid"),
		},
		Spec: v1alpha1.EmailConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: testConnectionSecret},
			},
			ForProvider: v1alpha1.EmailConfigurationParameters{
				Host:                   "smtp.example.com",
				Port:                   587,
				SecurityProtocol:       "STARTTLS",
				FromAddress:            "sonarqube@example.com",
				FromName:               "SonarQube",
				SubjectPrefix:          "[SonarQube]",
				AuthMethod:             v1alpha1.EmailAuthMethodBasic,
				Username:               "user",
				BasicPasswordSecretRef: secretKeySelector("password"),
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
func observedConfiguration() *sonar.SystemEmailConfiguration {
	return &sonar.SystemEmailConfiguration{
		Id:                 testExternalName,
		Host:               "smtp.example.com",
		Port:               "587",
		SecurityProtocol:   "STARTTLS",
		FromAddress:        "sonarqube@example.com",
		FromName:           "SonarQube",
		SubjectPrefix:      "[SonarQube]",
		AuthMethod:         v1alpha1.EmailAuthMethodBasic,
		Username:           new("user"),
		IsBasicPasswordSet: true,
	}
}

// observedOAuthConfiguration returns the SonarQube configuration matching
// newTestConfiguration with OAUTH authentication.
func observedOAuthConfiguration() *sonar.SystemEmailConfiguration {
	configuration := observedConfiguration()
	configuration.AuthMethod = v1alpha1.EmailAuthMethodOAuth
	configuration.OauthAuthenticationHost = new("https://login.example.com")
	configuration.OauthTenant = new("tenant")
	configuration.OauthScope = new("scope")
	configuration.IsOauthClientIdSet = true
	configuration.IsOauthClientSecretSet = true

	return configuration
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

// TestObserve tests observing the email configuration.
func TestObserve(t *testing.T) {
	t.Parallel()

	getReturns := func(configuration *sonar.SystemEmailConfiguration) *fake.MockEmailConfigurationsClient {
		return &fake.MockEmailConfigurationsClient{
			GetEmailConfigurationFn: func(id string) (*sonar.SystemEmailConfiguration, *http.Response, error) {
				if id != testExternalName {
					return nil, mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				}

				return configuration, mockHTTPResponse(http.StatusOK), nil
			},
		}
	}

	savedPassword := connectionSecret(map[string]string{connectionDetailBasicPasswordKey: testPassword})
	savedOAuth := connectionSecret(map[string]string{
		connectionDetailOAuthClientIDKey:     testClientID,
		connectionDetailOAuthClientSecretKey: testClientSecret,
	})
	upToDate := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}}
	outOfDate := managed.ExternalObservation{ResourceExists: true, ConnectionDetails: managed.ConnectionDetails{}}
	unread := managed.ExternalObservation{ResourceExists: true}

	cases := map[string]struct {
		objects       []client.Object
		client        *fake.MockEmailConfigurationsClient
		mg            *v1alpha1.EmailConfiguration
		want          managed.ExternalObservation
		wantErr       string
		wantCondition *xpv1.Condition
	}{
		"MissingExternalName": {
			client: &fake.MockEmailConfigurationsClient{},
			mg:     newTestConfiguration(""),
		},
		"NotFound": {
			client: getReturns(observedConfiguration()),
			mg:     newTestConfiguration("unknown"),
		},
		"GetError": {
			client: &fake.MockEmailConfigurationsClient{
				GetEmailConfigurationFn: func(_ string) (*sonar.SystemEmailConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:      newTestConfiguration(testExternalName),
			wantErr: errGetConfiguration,
		},
		"BasicUpToDate": {
			objects:       []client.Object{credentialsSecret(), savedPassword},
			client:        getReturns(observedConfiguration()),
			mg:            newTestConfiguration(testExternalName),
			want:          upToDate,
			wantCondition: new(xpv1.Available()),
		},
		"OAuthUpToDate": {
			objects: []client.Object{credentialsSecret(), savedOAuth},
			client:  getReturns(observedOAuthConfiguration()),
			mg:      newTestConfiguration(testExternalName, withOAuth()),
			want:    upToDate,
		},
		"LateInitialized": {
			objects: []client.Object{credentialsSecret(), savedOAuth},
			client:  getReturns(observedOAuthConfiguration()),
			mg: newTestConfiguration(testExternalName, withOAuth(), func(cr *v1alpha1.EmailConfiguration) {
				cr.Spec.ForProvider.OAuthScope = nil
			}),
			want: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true, ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		"SpecDrift": {
			objects: []client.Object{credentialsSecret(), savedOAuth},
			client:  getReturns(observedOAuthConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    outOfDate,
		},
		"PasswordNeverSaved": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    outOfDate,
		},
		"PasswordRotated": {
			objects: []client.Object{credentialsSecret(), connectionSecret(map[string]string{connectionDetailBasicPasswordKey: "old"})},
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    outOfDate,
		},
		"ClientSecretRotated": {
			objects: []client.Object{credentialsSecret(), connectionSecret(map[string]string{
				connectionDetailOAuthClientIDKey:     testClientID,
				connectionDetailOAuthClientSecretKey: "old",
			})},
			client: getReturns(observedOAuthConfiguration()),
			mg:     newTestConfiguration(testExternalName, withOAuth()),
			want:   outOfDate,
		},
		"MissingPassword": {
			client:  getReturns(observedConfiguration()),
			mg:      newTestConfiguration(testExternalName),
			want:    unread,
			wantErr: errGetBasicPassword,
		},
		"EmptyPassword": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedConfiguration()),
			mg: newTestConfiguration(testExternalName, func(cr *v1alpha1.EmailConfiguration) {
				cr.Spec.ForProvider.BasicPasswordSecretRef = secretKeySelector("empty")
			}),
			want:    unread,
			wantErr: errEmptyCredential,
		},
		"MissingClientID": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedOAuthConfiguration()),
			mg: newTestConfiguration(testExternalName, withOAuth(), func(cr *v1alpha1.EmailConfiguration) {
				cr.Spec.ForProvider.OAuthClientIDSecretRef = secretKeySelector("missing")
			}),
			want:    unread,
			wantErr: errGetOAuthClientID,
		},
		"MissingClientSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  getReturns(observedOAuthConfiguration()),
			mg: newTestConfiguration(testExternalName, withOAuth(), func(cr *v1alpha1.EmailConfiguration) {
				cr.Spec.ForProvider.OAuthClientSecretSecretRef = nil
			}),
			want:    unread,
			wantErr: errGetOAuthClientSecret,
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

	e := &external{kubeClient: kube, client: &fake.MockEmailConfigurationsClient{
		GetEmailConfigurationFn: func(_ string) (*sonar.SystemEmailConfiguration, *http.Response, error) {
			return observedConfiguration(), mockHTTPResponse(http.StatusOK), nil
		},
	}}

	_, err := e.Observe(context.Background(), newTestConfiguration(testExternalName))

	checkError(t, "Observe", errGetSavedCredentials, err)
}

// TestCreate tests creating the email configuration.
func TestCreate(t *testing.T) {
	t.Parallel()

	emptySearch := func() (*sonar.SystemEmailConfigurationsSearch, *http.Response, error) {
		return &sonar.SystemEmailConfigurationsSearch{}, mockHTTPResponse(http.StatusOK), nil
	}

	createReturns := func(id string, err error) func(*sonar.SystemEmailConfigurationCreateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error) {
		return func(_ *sonar.SystemEmailConfigurationCreateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error) {
			return &sonar.SystemEmailConfiguration{Id: id}, mockHTTPResponse(http.StatusOK), err
		}
	}

	cases := map[string]struct {
		objects          []client.Object
		client           *fake.MockEmailConfigurationsClient
		mg               *v1alpha1.EmailConfiguration
		want             managed.ExternalCreation
		wantErr          string
		wantExternalName string
	}{
		"MissingPassword": {
			client:  &fake.MockEmailConfigurationsClient{},
			mg:      newTestConfiguration(""),
			wantErr: errGetBasicPassword,
		},
		"SearchError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: func() (*sonar.SystemEmailConfigurationsSearch, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:      newTestConfiguration(""),
			wantErr: errSearchConfigurations,
		},
		"ConfigurationAlreadyExists": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: func() (*sonar.SystemEmailConfigurationsSearch, *http.Response, error) {
					return &sonar.SystemEmailConfigurationsSearch{
						EmailConfigurations: []sonar.SystemEmailConfiguration{{Id: testExternalName}},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg:      newTestConfiguration(""),
			wantErr: `already has an email configuration (id "email-configuration-id")`,
		},
		"CreateError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: emptySearch,
				CreateEmailConfigurationFn:  createReturns("", errors.New("bad request")), //nolint:bodyclose // mock response without a body
			},
			mg:      newTestConfiguration(""),
			wantErr: errCreateConfiguration,
		},
		"MissingID": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: emptySearch,
				CreateEmailConfigurationFn:  createReturns("", nil), //nolint:bodyclose // mock response without a body
			},
			mg:      newTestConfiguration(""),
			wantErr: errNoConfigurationID,
		},
		"BasicSuccess": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: emptySearch,
				CreateEmailConfigurationFn:  createReturns(testExternalName, nil), //nolint:bodyclose // mock response without a body
			},
			mg: newTestConfiguration(""),
			want: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailBasicPasswordKey: []byte(testPassword),
			}},
			wantExternalName: testExternalName,
		},
		"OAuthSuccess": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockEmailConfigurationsClient{
				SearchEmailConfigurationsFn: emptySearch,
				CreateEmailConfigurationFn:  createReturns(testExternalName, nil), //nolint:bodyclose // mock response without a body
			},
			mg: newTestConfiguration("", withOAuth()),
			want: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailOAuthClientIDKey:     []byte(testClientID),
				connectionDetailOAuthClientSecretKey: []byte(testClientSecret),
			}},
			wantExternalName: testExternalName,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}

			got, err := e.Create(context.Background(), tc.mg)

			checkError(t, "Create", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Create() -want, +got:\n%s", diff)
			}

			if gotExternalName := meta.GetExternalName(tc.mg); gotExternalName != tc.wantExternalName {
				t.Errorf("Create() external name = %q, want %q", gotExternalName, tc.wantExternalName)
			}
		})
	}
}

// TestUpdate tests updating the email configuration.
func TestUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		objects []client.Object
		err     error
		mg      *v1alpha1.EmailConfiguration
		want    managed.ExternalUpdate
		wantErr string
	}{
		"MissingExternalName": {
			mg:      newTestConfiguration(""),
			wantErr: "external name is not set",
		},
		"MissingPassword": {
			mg:      newTestConfiguration(testExternalName),
			wantErr: errGetBasicPassword,
		},
		"UpdateError": {
			objects: []client.Object{credentialsSecret()},
			err:     errors.New("bad request"),
			mg:      newTestConfiguration(testExternalName),
			wantErr: errUpdateConfiguration,
		},
		"Success": {
			objects: []client.Object{credentialsSecret()},
			mg:      newTestConfiguration(testExternalName),
			want: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{
				connectionDetailBasicPasswordKey: []byte(testPassword),
			}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := &fake.MockEmailConfigurationsClient{
				UpdateEmailConfigurationFn: func(id string, opt *sonar.SystemEmailConfigurationUpdateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error) {
					if id != testExternalName || opt.BasicPassword != testPassword {
						return nil, mockHTTPResponse(http.StatusBadRequest), errors.Errorf("unexpected update of %q", id)
					}

					return observedConfiguration(), mockHTTPResponse(http.StatusOK), tc.err
				},
			}

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: mock}

			got, err := e.Update(context.Background(), tc.mg)

			checkError(t, "Update", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Update() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestDelete tests deleting the email configuration.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockEmailConfigurationsClient
		mg      *v1alpha1.EmailConfiguration
		wantErr string
	}{
		"MissingExternalName": {
			client: &fake.MockEmailConfigurationsClient{},
			mg:     newTestConfiguration(""),
		},
		"Success": {
			client: &fake.MockEmailConfigurationsClient{
				DeleteEmailConfigurationFn: func(id string) (*http.Response, error) {
					if id != testExternalName {
						return nil, errors.Errorf("unexpected id %q", id)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			mg: newTestConfiguration(testExternalName),
		},
		"AlreadyDeleted": {
			client: &fake.MockEmailConfigurationsClient{
				DeleteEmailConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: SonarQube returns the response along with the error.
					return mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				},
			},
			mg: newTestConfiguration(testExternalName),
		},
		"DeleteError": {
			client: &fake.MockEmailConfigurationsClient{
				DeleteEmailConfigurationFn: func(_ string) (*http.Response, error) {
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
			newServiceFn: func(_ common.Config) instance.EmailConfigurationsClient {
				return &fake.MockEmailConfigurationsClient{}
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

// TestSetupGatedRegistersEmailConfigurationGVK verifies that SetupGated
// registers the EmailConfiguration GVK with the gate.
func TestSetupGatedRegistersEmailConfigurationGVK(t *testing.T) {
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

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.EmailConfigurationGroupVersionKind {
		t.Fatalf("SetupGated() registered GVKs %v, want [%v]", g.gvks, v1alpha1.EmailConfigurationGroupVersionKind)
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

	e := &external{kubeClient: buildKubeClient(), client: &fake.MockEmailConfigurationsClient{
		GetEmailConfigurationFn: func(_ string) (*sonar.SystemEmailConfiguration, *http.Response, error) {
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
