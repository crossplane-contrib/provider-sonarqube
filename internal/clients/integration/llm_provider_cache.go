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

package integration

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

const (
	// llmProviderDefinitionsCacheNamespace is the cache namespace of
	// api/v2/llm-connectivity/llm-provider-definitions.
	llmProviderDefinitionsCacheNamespace = "llm-connectivity/llm-provider-definitions"
	// llmProvidersCacheNamespace is the cache namespace of
	// api/v2/llm-connectivity/llm-providers.
	llmProvidersCacheNamespace = "llm-connectivity/llm-providers"
	// llmProviderMappingsCacheNamespace is the cache namespace of
	// api/v2/llm-connectivity/llm-provider-mappings, keyed by AI capability.
	llmProviderMappingsCacheNamespace = "llm-connectivity/llm-provider-mappings"
)

// NewCachedLLMConnectivityClient decorates client with scoped, or returns
// client as-is when scoped does not cache.
func NewCachedLLMConnectivityClient(client LLMConnectivityClient, scoped cache.Scoped) LLMConnectivityClient {
	if !scoped.Enabled() {
		return client
	}

	return &cachedLLMConnectivityClient{LLMConnectivityClient: client, cache: scoped}
}

// cachedLLMConnectivityClient is a LLMConnectivityClient that caches the
// provider definitions, the providers and the mappings, so that every
// LLMProvider of one connection shares one call of each per TTL.
//
// Provider writes invalidate the providers, and deleting a provider also
// invalidates the mappings, as SonarQube may drop the mappings selecting
// it. Mapping writes invalidate the mappings: a mapping write by one
// LLMProvider unmaps the capability from another one. The definitions only
// change with the SonarQube version and are never invalidated.
type cachedLLMConnectivityClient struct {
	LLMConnectivityClient

	// cache holds the responses of the embedded client's connection.
	cache cache.Scoped
}

// GetProviderDefinitions returns the cached provider definitions.
func (c *cachedLLMConnectivityClient) GetProviderDefinitions(ctx context.Context) (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error) {
	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(llmProviderDefinitionsCacheNamespace, ""), c.LLMConnectivityClient.GetProviderDefinitions)
}

// ListProviders returns the cached providers.
func (c *cachedLLMConnectivityClient) ListProviders(ctx context.Context) (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error) {
	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(llmProvidersCacheNamespace, ""), c.LLMConnectivityClient.ListProviders)
}

// GetProviderMappings returns the cached mappings of the requested AI
// capability (every capability when unset).
func (c *cachedLLMConnectivityClient) GetProviderMappings(ctx context.Context, opt *sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
	var capability string
	if opt != nil {
		capability = opt.AiCapability
	}

	return cache.FetchWithResponse(ctx, c.cache.Store, c.cache.Key(llmProviderMappingsCacheNamespace, capability),
		func(ctx context.Context) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
			return c.LLMConnectivityClient.GetProviderMappings(ctx, opt)
		})
}

// CreateProvider creates a provider, then invalidates the providers.
func (c *cachedLLMConnectivityClient) CreateProvider(ctx context.Context, opt *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	defer c.cache.Invalidate(llmProvidersCacheNamespace)

	return c.LLMConnectivityClient.CreateProvider(ctx, opt)
}

// UpdateProvider updates a provider, then invalidates the providers.
func (c *cachedLLMConnectivityClient) UpdateProvider(ctx context.Context, providerID string, opt *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	defer c.cache.Invalidate(llmProvidersCacheNamespace)

	return c.LLMConnectivityClient.UpdateProvider(ctx, providerID, opt)
}

// DeleteProvider deletes a provider, then invalidates the providers and the
// mappings.
func (c *cachedLLMConnectivityClient) DeleteProvider(ctx context.Context, providerID string) (*http.Response, error) {
	defer c.cache.Invalidate(llmProvidersCacheNamespace, llmProviderMappingsCacheNamespace)

	return c.LLMConnectivityClient.DeleteProvider(ctx, providerID)
}

// UpsertProviderMapping maps an AI capability, then invalidates the
// mappings.
func (c *cachedLLMConnectivityClient) UpsertProviderMapping(ctx context.Context, opt *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error) {
	defer c.cache.Invalidate(llmProviderMappingsCacheNamespace)

	return c.LLMConnectivityClient.UpsertProviderMapping(ctx, opt)
}
