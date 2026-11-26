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

package slackintegration

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
	// testResourceName is the name of the test SlackIntegration.
	testResourceName = "test-slack"
	// testExternalName is the configuration ID assigned by SonarQube.
	testExternalName = "slack-configuration"
	// testClientID is the client ID of the Slack app.
	testClientID = "client-id"
	// testCredentialsSecret is the Secret holding the Slack credentials.
	testCredentialsSecret = "slack-credentials"
	// testConnectionSecret is the connection Secret of the resource.
	testConnectionSecret = "slack-connection"
	// testClientSecret is the client secret used in tests.
	testClientSecret = "client-secret-value"
	// testSigningSecret is the signing secret used in tests.
	testSigningSecret = "signing-secret-value"
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

// credentialsSecret creates the Secret holding the Slack credentials.
func credentialsSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCredentialsSecret, Namespace: testNamespace},
		Data: map[string][]byte{
			"client":  []byte(testClientSecret),
			"signing": []byte(testSigningSecret),
			"empty":   []byte(""),
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

// savedCredentials returns the connection data matching the credentials
// Secret.
func savedCredentials() map[string]string {
	return map[string]string{
		connectionDetailClientSecretKey:  testClientSecret,
		connectionDetailSigningSecretKey: testSigningSecret,
	}
}

// appliedConnectionDetails returns the connection details published after the
// credentials are applied.
func appliedConnectionDetails() managed.ConnectionDetails {
	return managed.ConnectionDetails{
		connectionDetailClientSecretKey:  []byte(testClientSecret),
		connectionDetailSigningSecretKey: []byte(testSigningSecret),
	}
}

// slackOption customizes a test SlackIntegration.
type slackOption func(*v1alpha1.SlackIntegration)

// withClientSecretKey references another key of the credentials Secret.
func withClientSecretKey(key string) slackOption {
	return func(cr *v1alpha1.SlackIntegration) {
		cr.Spec.ForProvider.ClientSecretRef = secretKeySelector(key)
	}
}

// withSigningSecretKey references another key of the credentials Secret.
func withSigningSecretKey(key string) slackOption {
	return func(cr *v1alpha1.SlackIntegration) {
		cr.Spec.ForProvider.SigningSecretRef = secretKeySelector(key)
	}
}

// withDeletionTimestamp marks the resource as being deleted.
func withDeletionTimestamp() slackOption {
	return func(cr *v1alpha1.SlackIntegration) {
		cr.SetDeletionTimestamp(&metav1.Time{Time: time.Unix(1, 0)})
	}
}

// newTestSlack creates a SlackIntegration for use in tests.
func newTestSlack(externalName string, options ...slackOption) *v1alpha1.SlackIntegration {
	cr := &v1alpha1.SlackIntegration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testResourceName,
			Namespace:   testNamespace,
			Annotations: map[string]string{},
			UID:         types.UID("slack-uid"),
		},
		Spec: v1alpha1.SlackIntegrationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: testConnectionSecret},
			},
			ForProvider: v1alpha1.SlackIntegrationParameters{
				ClientID:         testClientID,
				ClientSecretRef:  secretKeySelector("client"),
				SigningSecretRef: secretKeySelector("signing"),
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

// listReturning returns a list function answering with the given
// configurations.
func listReturning(configurations ...sonar.IntegrationsIntegrationConfiguration) func(*sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error) {
	return func(opt *sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error) {
		if opt.IntegrationType != sonar.IntegrationTypeSlack {
			return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("unexpected integration type")
		}

		return &sonar.IntegrationsIntegrationConfigurationSearch{IntegrationConfigurations: configurations}, mockHTTPResponse(http.StatusOK), nil
	}
}

// listFailing returns a list function failing with the given status code.
func listFailing(statusCode int) func(*sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error) {
	return func(_ *sonar.IntegrationsListIntegrationConfigurationsOptions) (*sonar.IntegrationsIntegrationConfigurationSearch, *http.Response, error) {
		return nil, mockHTTPResponse(statusCode), errors.New("boom")
	}
}

// observedConfiguration returns the SonarQube configuration matching
// newTestSlack.
func observedConfiguration() sonar.IntegrationsIntegrationConfiguration {
	return sonar.IntegrationsIntegrationConfiguration{
		Id:              testExternalName,
		IntegrationType: sonar.IntegrationTypeSlack,
		ClientId:        testClientID,
		AppId:           "app-id",
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

// TestObserve tests observing the Slack configuration.
//
//nolint:bodyclose // the mock responses have no body to close
func TestObserve(t *testing.T) {
	t.Parallel()

	found := &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listReturning(observedConfiguration())}
	savedSecret := connectionSecret(savedCredentials())
	upToDate := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}}
	outdated := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}}

	cases := map[string]struct {
		objects       []client.Object
		client        *fake.MockSlackIntegrationsClient
		mg            *v1alpha1.SlackIntegration
		want          managed.ExternalObservation
		wantErr       string
		wantCondition *xpv1.Condition
	}{
		"MissingExternalName": {
			client: &fake.MockSlackIntegrationsClient{},
			mg:     newTestSlack(""),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"NotListed": {
			client: found,
			mg:     newTestSlack("other-id"),
			want:   managed.ExternalObservation{ResourceExists: false},
		},
		"ListError": {
			client:  &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listFailing(http.StatusInternalServerError)},
			mg:      newTestSlack(testExternalName),
			wantErr: errListConfigurations,
		},
		"ListNotFoundIsAnError": {
			client:  &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listFailing(http.StatusNotFound)},
			mg:      newTestSlack(testExternalName),
			wantErr: errListConfigurations,
		},
		"UpToDate": {
			objects:       []client.Object{credentialsSecret(), savedSecret},
			client:        found,
			mg:            newTestSlack(testExternalName),
			want:          upToDate,
			wantCondition: new(xpv1.Available()),
		},
		"ClientIDDrift": {
			objects: []client.Object{credentialsSecret(), savedSecret},
			client: &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listReturning(
				sonar.IntegrationsIntegrationConfiguration{Id: testExternalName, ClientId: "other"},
			)},
			mg:   newTestSlack(testExternalName),
			want: outdated,
		},
		"CredentialsNeverSaved": {
			objects: []client.Object{credentialsSecret()},
			client:  found,
			mg:      newTestSlack(testExternalName),
			want:    outdated,
		},
		"ClientSecretRotated": {
			objects: []client.Object{credentialsSecret(), connectionSecret(map[string]string{
				connectionDetailClientSecretKey:  "old",
				connectionDetailSigningSecretKey: testSigningSecret,
			})},
			client: found,
			mg:     newTestSlack(testExternalName),
			want:   outdated,
		},
		"SigningSecretRotated": {
			objects: []client.Object{credentialsSecret(), connectionSecret(map[string]string{
				connectionDetailClientSecretKey:  testClientSecret,
				connectionDetailSigningSecretKey: "old",
			})},
			client: found,
			mg:     newTestSlack(testExternalName),
			want:   outdated,
		},
		"DeletedWithoutCredentials": {
			client: found,
			mg:     newTestSlack(testExternalName, withDeletionTimestamp()),
			want:   managed.ExternalObservation{ResourceExists: true},
		},
		"MissingClientSecret": {
			client:  found,
			mg:      newTestSlack(testExternalName),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errGetClientSecret,
		},
		"EmptyClientSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  found,
			mg:      newTestSlack(testExternalName, withClientSecretKey("empty")),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errEmptyClientSecret,
		},
		"MissingSigningSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  found,
			mg:      newTestSlack(testExternalName, withSigningSecretKey("missing")),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errGetSigningSecret,
		},
		"EmptySigningSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  found,
			mg:      newTestSlack(testExternalName, withSigningSecretKey("empty")),
			want:    managed.ExternalObservation{ResourceExists: true},
			wantErr: errEmptySigningSecret,
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
//
//nolint:bodyclose // the mock responses have no body to close
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

	e := &external{kubeClient: kube, client: &fake.MockSlackIntegrationsClient{
		ListIntegrationConfigurationsFn: listReturning(observedConfiguration()),
	}}

	_, err := e.Observe(context.Background(), newTestSlack(testExternalName))

	checkError(t, "Observe", errGetSavedCredentials, err)
}

// TestCreate tests creating the Slack configuration.
//
//nolint:bodyclose // the mock responses have no body to close
func TestCreate(t *testing.T) {
	t.Parallel()

	createReturning := func(id string) func(*sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
		return func(opt *sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
			if opt.ClientId != testClientID || opt.ClientSecret != testClientSecret || opt.SigningSecret != testSigningSecret {
				return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("unexpected options")
			}

			return &sonar.IntegrationsIntegrationConfiguration{Id: id}, mockHTTPResponse(http.StatusCreated), nil
		}
	}

	cases := map[string]struct {
		objects          []client.Object
		client           *fake.MockSlackIntegrationsClient
		want             managed.ExternalCreation
		wantErr          string
		wantExternalName string
	}{
		"MissingClientSecret": {
			client:  &fake.MockSlackIntegrationsClient{},
			wantErr: errGetClientSecret,
		},
		"ListError": {
			objects: []client.Object{credentialsSecret()},
			client:  &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listFailing(http.StatusInternalServerError)},
			wantErr: errListConfigurations,
		},
		"ConfigurationAlreadyExists": {
			objects: []client.Object{credentialsSecret()},
			client:  &fake.MockSlackIntegrationsClient{ListIntegrationConfigurationsFn: listReturning(observedConfiguration())},
			wantErr: `already has a Slack configuration (id "slack-configuration")`,
		},
		"CreateError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockSlackIntegrationsClient{
				ListIntegrationConfigurationsFn: listReturning(),
				CreateIntegrationConfigurationFn: func(_ *sonar.IntegrationsIntegrationConfigurationCreateOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("bad request")
				},
			},
			wantErr: errCreateConfiguration,
		},
		"MissingID": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockSlackIntegrationsClient{
				ListIntegrationConfigurationsFn:  listReturning(),
				CreateIntegrationConfigurationFn: createReturning(""),
			},
			wantErr: errNoConfigurationID,
		},
		"Success": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockSlackIntegrationsClient{
				ListIntegrationConfigurationsFn:  listReturning(),
				CreateIntegrationConfigurationFn: createReturning(testExternalName),
			},
			want:             managed.ExternalCreation{ConnectionDetails: appliedConnectionDetails()},
			wantExternalName: testExternalName,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}
			cr := newTestSlack("")

			got, err := e.Create(context.Background(), cr)

			checkError(t, "Create", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Create() -want, +got:\n%s", diff)
			}

			if externalName := meta.GetExternalName(cr); externalName != tc.wantExternalName {
				t.Errorf("Create() external name = %q, want %q", externalName, tc.wantExternalName)
			}
		})
	}
}

// TestUpdate tests updating the Slack configuration.
func TestUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		objects []client.Object
		client  *fake.MockSlackIntegrationsClient
		mg      *v1alpha1.SlackIntegration
		want    managed.ExternalUpdate
		wantErr string
	}{
		"MissingExternalName": {
			client:  &fake.MockSlackIntegrationsClient{},
			mg:      newTestSlack(""),
			wantErr: "external name is not set",
		},
		"MissingSigningSecret": {
			objects: []client.Object{credentialsSecret()},
			client:  &fake.MockSlackIntegrationsClient{},
			mg:      newTestSlack(testExternalName, withSigningSecretKey("missing")),
			wantErr: errGetSigningSecret,
		},
		"UpdateError": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockSlackIntegrationsClient{
				UpdateIntegrationConfigurationFn: func(_ string, _ *sonar.IntegrationsIntegrationConfigurationPatchOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("bad request")
				},
			},
			mg:      newTestSlack(testExternalName),
			wantErr: errUpdateConfiguration,
		},
		"Success": {
			objects: []client.Object{credentialsSecret()},
			client: &fake.MockSlackIntegrationsClient{
				UpdateIntegrationConfigurationFn: func(id string, opt *sonar.IntegrationsIntegrationConfigurationPatchOptions) (*sonar.IntegrationsIntegrationConfiguration, *http.Response, error) {
					if id != testExternalName || opt.ClientId != testClientID || opt.ClientSecret != testClientSecret || opt.SigningSecret != testSigningSecret {
						return nil, mockHTTPResponse(http.StatusBadRequest), errors.New("unexpected request")
					}

					return new(observedConfiguration()), mockHTTPResponse(http.StatusOK), nil
				},
			},
			mg:   newTestSlack(testExternalName),
			want: managed.ExternalUpdate{ConnectionDetails: appliedConnectionDetails()},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: buildKubeClient(tc.objects...), client: tc.client}

			got, err := e.Update(context.Background(), tc.mg)

			checkError(t, "Update", tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Update() -want, +got:\n%s", diff)
			}
		})
	}
}

// TestDelete tests deleting the Slack configuration.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *fake.MockSlackIntegrationsClient
		mg      *v1alpha1.SlackIntegration
		wantErr string
	}{
		"MissingExternalName": {
			client: &fake.MockSlackIntegrationsClient{},
			mg:     newTestSlack(""),
		},
		"Success": {
			client: &fake.MockSlackIntegrationsClient{
				DeleteIntegrationConfigurationFn: func(id string) (*http.Response, error) {
					if id != testExternalName {
						return nil, errors.Errorf("unexpected id %q", id)
					}

					return mockHTTPResponse(http.StatusNoContent), nil
				},
			},
			mg: newTestSlack(testExternalName),
		},
		"AlreadyDeleted": {
			client: &fake.MockSlackIntegrationsClient{
				DeleteIntegrationConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: SonarQube returns the response along with the error.
					return mockHTTPResponse(http.StatusNotFound), errors.New("not found")
				},
			},
			mg: newTestSlack(testExternalName),
		},
		"DeleteError": {
			client: &fake.MockSlackIntegrationsClient{
				DeleteIntegrationConfigurationFn: func(_ string) (*http.Response, error) {
					//nolint:nilnil // Intentional: SonarQube returns the response along with the error.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
				},
			},
			mg:      newTestSlack(testExternalName),
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

		_, err := c.Connect(context.Background(), newTestSlack(""))

		checkError(t, "Connect", errTrackPCUsage, err)
	})

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		cr := newTestSlack("")
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

		cr := newTestSlack("")
		cr.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:  kube,
			usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(_ common.Config) integration.SlackIntegrationsClient {
				return &fake.MockSlackIntegrationsClient{}
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

// TestSetupGatedRegistersSlackIntegrationGVK verifies that SetupGated
// registers the SlackIntegration GVK with the gate.
func TestSetupGatedRegistersSlackIntegrationGVK(t *testing.T) {
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

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.SlackIntegrationGroupVersionKind {
		t.Fatalf("SetupGated() registered GVKs %v, want [%v]", g.gvks, v1alpha1.SlackIntegrationGroupVersionKind)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}
