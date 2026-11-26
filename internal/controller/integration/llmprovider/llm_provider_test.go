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

package llmprovider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testProviderID is the SonarQube provider ID used in tests.
	testProviderID = "llm-1"
	// testNamespace is the namespace of the test resources.
	testNamespace = "default"
	// testAPIKey is the API key stored in the test secret.
	testAPIKey = "sk-test"
	// testConnSecret is the connection secret name.
	testConnSecret = "llm-conn"
)

// mockGate is a mock implementation of the feature gate interface.
type mockGate struct {
	callback func()
	gvks     []schema.GroupVersionKind
}

// Register records the callback and GVKs.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set implements the gate interface (no-op for tests).
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// okResponse returns an empty 200 response.
func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// newTestLLMProvider returns an OpenAI LLMProvider mapped to AI_CODEFIX.
func newTestLLMProvider(externalName string) *v1alpha1.LLMProvider {
	llmProvider := &v1alpha1.LLMProvider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-llm", Namespace: testNamespace, Annotations: map[string]string{}, UID: types.UID("llm-uid")},
		Spec: v1alpha1.LLMProviderSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: testConnSecret},
			},
			ForProvider: v1alpha1.LLMProviderParameters{
				Provider:      "OPENAI",
				Label:         "OpenAI",
				Configuration: map[string]string{"endpoint": "https://api.openai.com"},
				ConfigurationSecretRefs: map[string]xpv1.LocalSecretKeySelector{
					"apiKey": {LocalSecretReference: xpv1.LocalSecretReference{Name: "openai"}, Key: "apiKey"},
				},
				HTTPHeaders: []v1alpha1.LLMProviderHTTPHeader{},
				ModelMappings: []v1alpha1.LLMProviderModelMapping{
					{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"},
				},
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(llmProvider, externalName)
	}

	return llmProvider
}

// testDefinitions returns the OpenAI provider definition.
func testDefinitions() *sonar.LlmConnectivityLlmProviderDefinitionsResponse {
	return &sonar.LlmConnectivityLlmProviderDefinitionsResponse{ProviderDefinitions: []sonar.LlmConnectivityLlmProviderDefinitionResponse{{
		Provider: "OPENAI",
		Fields: []sonar.LlmConnectivityLlmProviderFieldResponse{
			{Key: "apiKey", Type: "STRING", Required: true, Secret: true},
			{Key: "endpoint", Type: "STRING", Required: true},
		},
	}}}
}

// testProviders returns the providers list holding the up-to-date provider.
func testProviders() *sonar.LlmConnectivityLlmProvidersResponse {
	return &sonar.LlmConnectivityLlmProvidersResponse{Providers: []sonar.LlmConnectivityLlmProviderResponse{{
		Id:            testProviderID,
		Provider:      "OPENAI",
		Label:         "OpenAI",
		Configuration: map[string]any{"endpoint": "https://api.openai.com", "apiKey": "****"},
	}}}
}

// testMappings returns the mappings, AI_CODEFIX mapped to providerID.
func testMappings(providerID string) *sonar.LlmConnectivityProviderMappingsResponse {
	return &sonar.LlmConnectivityProviderMappingsResponse{ProviderMappings: []sonar.LlmConnectivityProviderMappingResponse{
		{AiCapability: "AI_CODEFIX", LlmProviderId: providerID, ModelIdentifier: "gpt-4o"},
		{AiCapability: "HUNTER_AGENT"},
	}}
}

// newMockClient returns a client answering the reads with an up-to-date
// provider.
func newMockClient() *fake.MockLLMConnectivityClient {
	return &fake.MockLLMConnectivityClient{
		GetProviderDefinitionsFn: func() (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error) {
			return testDefinitions(), okResponse(), nil
		},
		ListProvidersFn: func() (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error) {
			return testProviders(), okResponse(), nil
		},
		GetProviderMappingsFn: func(*sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
			return testMappings(testProviderID), okResponse(), nil
		},
	}
}

// secret returns a Secret in the test namespace.
func secret(name string, data map[string]string) *corev1.Secret {
	bytes := map[string][]byte{}
	for key, value := range data {
		bytes[key] = []byte(value)
	}

	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}, Data: bytes}
}

// newKube returns a fake Kubernetes client holding objects.
func newKube(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// defaultKube returns a client holding the API key and the matching
// connection secret.
func defaultKube() client.Client {
	return newKube(
		secret("openai", map[string]string{"apiKey": testAPIKey}),
		secret(testConnSecret, map[string]string{"configuration.apiKey": testAPIKey}),
	)
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

// TestObserve tests observing LLMProvider resources.
func TestObserve(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		kube         client.Client
		client       func() *fake.MockLLMConnectivityClient
		mg           func() *v1alpha1.LLMProvider
		wantExists   bool
		wantUpToDate bool
		wantErr      string
	}{
		"NoExternalName": {
			kube:   defaultKube(),
			client: func() *fake.MockLLMConnectivityClient { return &fake.MockLLMConnectivityClient{} },
			mg:     func() *v1alpha1.LLMProvider { return newTestLLMProvider("") },
		},
		"ListError": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.ListProvidersFn = func() (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error) {
					return nil, nil, errors.New("boom")
				}

				return mock
			},
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantErr: errListProviders,
		},
		"NotFound": {
			kube:   defaultKube(),
			client: newMockClient,
			mg:     func() *v1alpha1.LLMProvider { return newTestLLMProvider("test-llm") },
		},
		"UpToDate": {
			kube:         defaultKube(),
			client:       newMockClient,
			mg:           func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantExists:   true,
			wantUpToDate: true,
		},
		"SecretRotated": {
			kube: newKube(
				secret("openai", map[string]string{"apiKey": "sk-new"}),
				secret(testConnSecret, map[string]string{"configuration.apiKey": testAPIKey}),
			),
			client:     newMockClient,
			mg:         func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantExists: true,
		},
		"NoConnectionSecret": {
			kube:       newKube(secret("openai", map[string]string{"apiKey": testAPIKey})),
			client:     newMockClient,
			mg:         func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantExists: true,
		},
		"MappingElsewhere": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.GetProviderMappingsFn = func(*sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
					return testMappings("other"), okResponse(), nil
				}

				return mock
			},
			mg:         func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantExists: true,
		},
		"DeletingSkipsSecrets": {
			kube:   newKube(),
			client: newMockClient,
			mg: func() *v1alpha1.LLMProvider {
				mg := newTestLLMProvider(testProviderID)
				mg.SetDeletionTimestamp(new(metav1.Now()))

				return mg
			},
			wantExists: true,
		},
		"MissingSecret": {
			kube:    newKube(),
			client:  newMockClient,
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantErr: `cannot get configuration "apiKey" from secret reference`,
		},
		"EmptySecret": {
			kube:    newKube(secret("openai", map[string]string{"apiKey": ""})),
			client:  newMockClient,
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantErr: "secret value is empty",
		},
		"DefinitionsError": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.GetProviderDefinitionsFn = nil

				return mock
			},
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantErr: errGetDefinitions,
		},
		"MappingsError": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.GetProviderMappingsFn = nil

				return mock
			},
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider(testProviderID) },
			wantErr: errGetMappings,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: tc.kube, client: tc.client()}
			mg := tc.mg()

			got, err := e.Observe(context.Background(), mg)
			checkError(t, "Observe", tc.wantErr, err)

			if got.ResourceExists != tc.wantExists || got.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("Observe() = exists %v, upToDate %v, want %v, %v", got.ResourceExists, got.ResourceUpToDate, tc.wantExists, tc.wantUpToDate)
			}

			if tc.wantUpToDate && mg.Status.AtProvider.Configuration["apiKey"] != "" {
				t.Errorf("Observe() reported secret configuration: %v", mg.Status.AtProvider.Configuration)
			}
		})
	}
}

// TestCreate tests creating LLMProvider resources.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		var sent *sonar.LlmConnectivityCreateProviderOptions

		mock := newMockClient()
		mock.CreateProviderFn = func(opt *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
			sent = opt

			return &sonar.LlmConnectivityLlmProviderResponse{Id: testProviderID}, okResponse(), nil
		}

		mg := newTestLLMProvider("test-llm")
		e := &external{kubeClient: defaultKube(), client: mock}

		got, err := e.Create(context.Background(), mg)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if meta.GetExternalName(mg) != testProviderID {
			t.Errorf("external name = %q, want %q", meta.GetExternalName(mg), testProviderID)
		}

		if sent == nil || sent.Configuration["apiKey"] != testAPIKey || sent.Provider != "OPENAI" {
			t.Errorf("CreateProvider() options = %+v", sent)
		}

		if string(got.ConnectionDetails["configuration.apiKey"]) != testAPIKey {
			t.Errorf("ConnectionDetails = %v, want the API key", got.ConnectionDetails)
		}
	})

	errorCases := map[string]struct {
		kube    client.Client
		client  func() *fake.MockLLMConnectivityClient
		mg      func() *v1alpha1.LLMProvider
		wantErr string
	}{
		"MissingSecret": {
			kube:    newKube(),
			client:  newMockClient,
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider("") },
			wantErr: "cannot get configuration",
		},
		"DefinitionsError": {
			kube:    defaultKube(),
			client:  func() *fake.MockLLMConnectivityClient { return &fake.MockLLMConnectivityClient{} },
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider("") },
			wantErr: errGetDefinitions,
		},
		"InvalidSpec": {
			kube:   defaultKube(),
			client: newMockClient,
			mg: func() *v1alpha1.LLMProvider {
				mg := newTestLLMProvider("")
				mg.Spec.ForProvider.Configuration["unknown"] = "x"

				return mg
			},
			wantErr: `unknown configuration key "unknown"`,
		},
		"APIError": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.CreateProviderFn = func(*sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
					return nil, nil, errors.New("connection test failed")
				}

				return mock
			},
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider("") },
			wantErr: errCreateProvider,
		},
		"MissingID": {
			kube: defaultKube(),
			client: func() *fake.MockLLMConnectivityClient {
				mock := newMockClient()
				mock.CreateProviderFn = func(*sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
					return &sonar.LlmConnectivityLlmProviderResponse{}, okResponse(), nil
				}

				return mock
			},
			mg:      func() *v1alpha1.LLMProvider { return newTestLLMProvider("") },
			wantErr: "missing LLM provider id",
		},
	}

	for name, tc := range errorCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{kubeClient: tc.kube, client: tc.client()}

			_, err := e.Create(context.Background(), tc.mg())
			checkError(t, "Create", tc.wantErr, err)
		})
	}
}

// observed returns an LLMProvider whose status was filled by Observe with
// the given mock.
func observed(t *testing.T, kube client.Client, mock *fake.MockLLMConnectivityClient) *v1alpha1.LLMProvider {
	t.Helper()

	mg := newTestLLMProvider(testProviderID)

	_, err := (&external{kubeClient: kube, client: mock}).Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	return mg
}

// TestUpdate tests updating LLMProvider resources.
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("NoExternalName", func(t *testing.T) {
		t.Parallel()

		e := &external{kubeClient: defaultKube(), client: newMockClient()}

		_, err := e.Update(context.Background(), newTestLLMProvider(""))
		checkError(t, "Update", "external name is not set", err)
	})

	t.Run("ProviderTypeChanged", func(t *testing.T) {
		t.Parallel()

		mg := newTestLLMProvider(testProviderID)
		mg.Status.AtProvider.Provider = "ANTHROPIC"

		_, err := (&external{kubeClient: defaultKube(), client: newMockClient()}).Update(context.Background(), mg)
		checkError(t, "Update", "the type cannot be changed", err)
	})

	t.Run("MappingOnly", func(t *testing.T) {
		t.Parallel()

		calls := map[string]int{}
		mock := newMockClient()
		mock.GetProviderMappingsFn = func(*sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
			return testMappings("other"), okResponse(), nil
		}
		mock.UpdateProviderFn = func(string, *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
			calls["UpdateProvider"]++

			return nil, okResponse(), nil
		}
		mock.UpsertProviderMappingFn = func(opt *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error) {
			calls["UpsertProviderMapping"]++

			if opt.LlmProviderId != testProviderID || opt.AiCapability != "AI_CODEFIX" || opt.ModelIdentifier != "gpt-4o" {
				t.Errorf("UpsertProviderMapping() options = %+v", opt)
			}

			return nil, okResponse(), nil
		}

		kube := defaultKube()
		mg := observed(t, kube, mock)

		_, err := (&external{kubeClient: kube, client: mock}).Update(context.Background(), mg)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if calls["UpdateProvider"] != 0 || calls["UpsertProviderMapping"] != 1 {
			t.Errorf("calls = %v, want only one UpsertProviderMapping", calls)
		}
	})

	t.Run("ConfigurationDrift", func(t *testing.T) {
		t.Parallel()

		var sent *sonar.LlmConnectivityUpdateProviderOptions

		mock := newMockClient()
		mock.UpdateProviderFn = func(providerID string, opt *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
			if providerID != testProviderID {
				t.Errorf("UpdateProvider() id = %q", providerID)
			}

			sent = opt

			return nil, okResponse(), nil
		}

		kube := defaultKube()
		mg := observed(t, kube, mock)
		mg.Spec.ForProvider.Label = "Renamed"

		got, err := (&external{kubeClient: kube, client: mock}).Update(context.Background(), mg)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if sent == nil || ptr.Deref(sent.Label, "") != "Renamed" || sent.Configuration["apiKey"] != testAPIKey {
			t.Errorf("UpdateProvider() options = %+v", sent)
		}

		if string(got.ConnectionDetails["configuration.apiKey"]) != testAPIKey {
			t.Errorf("ConnectionDetails = %v, want the API key", got.ConnectionDetails)
		}
	})

	errorCases := map[string]struct {
		kube    client.Client
		mutate  func(mock *fake.MockLLMConnectivityClient, mg *v1alpha1.LLMProvider)
		wantErr string
	}{
		"MissingSecret": {
			kube:    newKube(),
			mutate:  func(*fake.MockLLMConnectivityClient, *v1alpha1.LLMProvider) {},
			wantErr: "cannot get configuration",
		},
		"InvalidSpec": {
			kube: defaultKube(),
			mutate: func(_ *fake.MockLLMConnectivityClient, mg *v1alpha1.LLMProvider) {
				mg.Spec.ForProvider.Configuration["unknown"] = "x"
			},
			wantErr: "unknown configuration key",
		},
		"DefinitionsError": {
			kube: defaultKube(),
			mutate: func(mock *fake.MockLLMConnectivityClient, mg *v1alpha1.LLMProvider) {
				mock.GetProviderDefinitionsFn = nil
				mg.Spec.ForProvider.Label = "Renamed"
			},
			wantErr: errGetDefinitions,
		},
		"UpdateError": {
			kube: defaultKube(),
			mutate: func(mock *fake.MockLLMConnectivityClient, mg *v1alpha1.LLMProvider) {
				mg.Spec.ForProvider.Label = "Renamed"
			},
			wantErr: errUpdateProvider,
		},
		"UpsertError": {
			kube: defaultKube(),
			mutate: func(_ *fake.MockLLMConnectivityClient, mg *v1alpha1.LLMProvider) {
				mg.Spec.ForProvider.ModelMappings[0].ModelIdentifier = "gpt-5"
			},
			wantErr: "cannot map AI capability AI_CODEFIX",
		},
	}

	for name, tc := range errorCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := newMockClient()
			mg := observed(t, defaultKube(), mock)
			tc.mutate(mock, mg)

			_, err := (&external{kubeClient: tc.kube, client: mock}).Update(context.Background(), mg)
			checkError(t, "Update", tc.wantErr, err)
		})
	}
}

// TestDelete tests deleting LLMProvider resources.
func TestDelete(t *testing.T) {
	t.Parallel()

	unmapped := func() *fake.MockLLMConnectivityClient {
		mock := newMockClient()
		mock.GetProviderMappingsFn = func(*sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
			return testMappings("other"), okResponse(), nil
		}

		return mock
	}

	cases := map[string]struct {
		client  func() *fake.MockLLMConnectivityClient
		mg      *v1alpha1.LLMProvider
		wantErr string
	}{
		"NoExternalName": {
			client: func() *fake.MockLLMConnectivityClient { return &fake.MockLLMConnectivityClient{} },
			mg:     newTestLLMProvider(""),
		},
		"MappingsError": {
			client:  func() *fake.MockLLMConnectivityClient { return &fake.MockLLMConnectivityClient{} },
			mg:      newTestLLMProvider(testProviderID),
			wantErr: errGetMappings,
		},
		"StillMapped": {
			client:  newMockClient,
			mg:      newTestLLMProvider(testProviderID),
			wantErr: "before deleting it: AI_CODEFIX",
		},
		"Success": {
			client: func() *fake.MockLLMConnectivityClient {
				mock := unmapped()
				mock.DeleteProviderFn = func(string) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
				}

				return mock
			},
			mg: newTestLLMProvider(testProviderID),
		},
		"AlreadyGone": {
			client: func() *fake.MockLLMConnectivityClient {
				mock := unmapped()
				mock.DeleteProviderFn = func(string) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, errors.New("not found") //nolint:nilnil // the SDK returns the response along with the error
				}

				return mock
			},
			mg: newTestLLMProvider(testProviderID),
		},
		"APIError": {
			client: func() *fake.MockLLMConnectivityClient {
				mock := unmapped()
				mock.DeleteProviderFn = func(string) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusBadRequest, Body: http.NoBody}, errors.New("bad") //nolint:nilnil // the SDK returns the response along with the error
				}

				return mock
			},
			mg:      newTestLLMProvider(testProviderID),
			wantErr: errDeleteProvider,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := (&external{kubeClient: newKube(), client: tc.client()}).Delete(context.Background(), tc.mg)
			checkError(t, "Delete", tc.wantErr, err)
		})
	}
}

// TestDisconnect tests that Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	err := (&external{}).Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}
}

// TestGetSavedSecretsHeaders tests reading the saved secret headers.
func TestGetSavedSecretsHeaders(t *testing.T) {
	t.Parallel()

	mg := newTestLLMProvider(testProviderID)
	mg.Spec.ForProvider.HTTPHeaders = []v1alpha1.LLMProviderHTTPHeader{
		{Name: "X-Plain", Value: new("v")},
		{Name: "Authorization", ValueSecretRef: &xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "proxy"}, Key: "auth"}},
	}

	kube := newKube(
		secret("openai", map[string]string{"apiKey": testAPIKey}),
		secret("proxy", map[string]string{"auth": "Bearer t"}),
		secret(testConnSecret, map[string]string{"httpHeader.Authorization": "Bearer old"}),
	)
	e := &external{kubeClient: kube}

	current, err := e.readSecrets(context.Background(), mg)
	if err != nil || current.HTTPHeaders["Authorization"] != "Bearer t" || len(current.HTTPHeaders) != 1 {
		t.Errorf("readSecrets() = %+v, %v", current, err)
	}

	saved, err := e.getSavedSecrets(context.Background(), mg)
	if err != nil || saved.HTTPHeaders["Authorization"] != "Bearer old" || saved.Configuration["apiKey"] != "" {
		t.Errorf("getSavedSecrets() = %+v, %v", saved, err)
	}

	mg.Spec.ForProvider.HTTPHeaders[1].ValueSecretRef.Name = "missing"

	_, err = e.readSecrets(context.Background(), mg)
	checkError(t, "readSecrets", `cannot get HTTP header "Authorization"`, err)

	mg.Spec.WriteConnectionSecretToReference = nil

	saved, err = e.getSavedSecrets(context.Background(), mg)
	if err != nil || len(saved.Configuration) != 0 {
		t.Errorf("getSavedSecrets(no ref) = %+v, %v", saved, err)
	}
}

// newScheme returns a scheme with the provider and integration types.
func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{apisv1alpha1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme, corev1.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() error = %v", err)
		}
	}

	return scheme
}

// TestConnect tests connecting to SonarQube.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		mg := newTestLLMProvider("")
		mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing", Kind: "ProviderConfig"})

		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), mg)
		checkError(t, "Connect", errGetPC, err)
	})

	t.Run("TrackUsageError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), newTestLLMProvider(""))
		checkError(t, "Connect", errTrackPCUsage, err)
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

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(providerConfig, secret("prov-secret", map[string]string{"token": "t"})).Build()
		mg := newTestLLMProvider("")
		mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:         kube,
			usage:        resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(common.Config) integration.LLMConnectivityClient { return &fake.MockLLMConnectivityClient{} },
		}

		got, err := c.Connect(context.Background(), mg)
		if err != nil || got == nil {
			t.Fatalf("Connect() = %v, %v", got, err)
		}
	})
}

// TestSetupGated tests that SetupGated registers the LLMProvider GVK and
// that the callback panics when setup fails.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() error = %v", err)
	}

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.LLMProviderGroupVersionKind {
		t.Fatalf("SetupGated() GVKs = %v", g.gvks)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}
