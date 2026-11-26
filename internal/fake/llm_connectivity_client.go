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

package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockLLMConnectivityClient is a mock implementation of the
// LLMConnectivityClient interface.
type MockLLMConnectivityClient struct {
	GetProviderDefinitionsFn func() (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error)
	ListProvidersFn          func() (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error)
	GetProviderMappingsFn    func(opt *sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error)
	CreateProviderFn         func(opt *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error)
	UpdateProviderFn         func(providerID string, opt *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error)
	DeleteProviderFn         func(providerID string) (*http.Response, error)
	UpsertProviderMappingFn  func(opt *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error)
}

// Ensure MockLLMConnectivityClient implements LLMConnectivityClient.
var _ integration.LLMConnectivityClient = &MockLLMConnectivityClient{}

// GetProviderDefinitions implements
// LLMConnectivityClient.GetProviderDefinitions.
func (m *MockLLMConnectivityClient) GetProviderDefinitions(_ context.Context) (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error) {
	if m.GetProviderDefinitionsFn != nil {
		return m.GetProviderDefinitionsFn()
	}

	return nil, nil, errNotImplemented
}

// ListProviders implements LLMConnectivityClient.ListProviders.
func (m *MockLLMConnectivityClient) ListProviders(_ context.Context) (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error) {
	if m.ListProvidersFn != nil {
		return m.ListProvidersFn()
	}

	return nil, nil, errNotImplemented
}

// GetProviderMappings implements LLMConnectivityClient.GetProviderMappings.
func (m *MockLLMConnectivityClient) GetProviderMappings(_ context.Context, opt *sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
	if m.GetProviderMappingsFn != nil {
		return m.GetProviderMappingsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// CreateProvider implements LLMConnectivityClient.CreateProvider.
func (m *MockLLMConnectivityClient) CreateProvider(_ context.Context, opt *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	if m.CreateProviderFn != nil {
		return m.CreateProviderFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateProvider implements LLMConnectivityClient.UpdateProvider.
func (m *MockLLMConnectivityClient) UpdateProvider(_ context.Context, providerID string, opt *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	if m.UpdateProviderFn != nil {
		return m.UpdateProviderFn(providerID, opt)
	}

	return nil, nil, errNotImplemented
}

// DeleteProvider implements LLMConnectivityClient.DeleteProvider.
func (m *MockLLMConnectivityClient) DeleteProvider(_ context.Context, providerID string) (*http.Response, error) {
	if m.DeleteProviderFn != nil {
		return m.DeleteProviderFn(providerID)
	}

	return nil, errNotImplemented
}

// UpsertProviderMapping implements
// LLMConnectivityClient.UpsertProviderMapping.
func (m *MockLLMConnectivityClient) UpsertProviderMapping(_ context.Context, opt *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error) {
	if m.UpsertProviderMappingFn != nil {
		return m.UpsertProviderMappingFn(opt)
	}

	return nil, nil, errNotImplemented
}
