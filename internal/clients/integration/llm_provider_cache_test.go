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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
)

// countingLLMConnectivityClient is a LLMConnectivityClient counting its
// calls.
type countingLLMConnectivityClient struct {
	calls map[string]int
}

// okResponse returns an empty 200 response.
func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// GetProviderDefinitions counts the call.
func (c *countingLLMConnectivityClient) GetProviderDefinitions(context.Context) (*sonar.LlmConnectivityLlmProviderDefinitionsResponse, *http.Response, error) {
	c.calls["GetProviderDefinitions"]++

	return &sonar.LlmConnectivityLlmProviderDefinitionsResponse{}, okResponse(), nil
}

// ListProviders counts the call.
func (c *countingLLMConnectivityClient) ListProviders(context.Context) (*sonar.LlmConnectivityLlmProvidersResponse, *http.Response, error) {
	c.calls["ListProviders"]++

	return &sonar.LlmConnectivityLlmProvidersResponse{}, okResponse(), nil
}

// GetProviderMappings counts the call.
func (c *countingLLMConnectivityClient) GetProviderMappings(context.Context, *sonar.LlmConnectivityGetProviderMappingsOptions) (*sonar.LlmConnectivityProviderMappingsResponse, *http.Response, error) {
	c.calls["GetProviderMappings"]++

	return &sonar.LlmConnectivityProviderMappingsResponse{}, okResponse(), nil
}

// CreateProvider counts the call.
func (c *countingLLMConnectivityClient) CreateProvider(context.Context, *sonar.LlmConnectivityCreateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	c.calls["CreateProvider"]++

	return &sonar.LlmConnectivityLlmProviderResponse{}, okResponse(), nil
}

// UpdateProvider counts the call.
func (c *countingLLMConnectivityClient) UpdateProvider(context.Context, string, *sonar.LlmConnectivityUpdateProviderOptions) (*sonar.LlmConnectivityLlmProviderResponse, *http.Response, error) {
	c.calls["UpdateProvider"]++

	return &sonar.LlmConnectivityLlmProviderResponse{}, okResponse(), nil
}

// DeleteProvider counts the call.
func (c *countingLLMConnectivityClient) DeleteProvider(context.Context, string) (*http.Response, error) {
	c.calls["DeleteProvider"]++

	return okResponse(), nil
}

// UpsertProviderMapping counts the call.
func (c *countingLLMConnectivityClient) UpsertProviderMapping(context.Context, *sonar.LlmConnectivityUpsertProviderMappingOptions) (*sonar.LlmConnectivityProviderMappingResponse, *http.Response, error) {
	c.calls["UpsertProviderMapping"]++

	return &sonar.LlmConnectivityProviderMappingResponse{}, okResponse(), nil
}

// readAll reads every cached dataset once.
func readAll(t *testing.T, client LLMConnectivityClient) {
	t.Helper()

	ctx := context.Background()

	_, _, err := client.GetProviderDefinitions(ctx) //nolint:bodyclose // fake or test server responses
	if err != nil {
		t.Fatalf("GetProviderDefinitions() error = %v", err)
	}

	_, _, err = client.ListProviders(ctx) //nolint:bodyclose // fake or test server responses
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}

	_, _, err = client.GetProviderMappings(ctx, &sonar.LlmConnectivityGetProviderMappingsOptions{}) //nolint:bodyclose // fake or test server responses
	if err != nil {
		t.Fatalf("GetProviderMappings() error = %v", err)
	}
}

// TestNewCachedLLMConnectivityClientDisabled tests that a disabled cache
// returns the client as-is.
func TestNewCachedLLMConnectivityClientDisabled(t *testing.T) {
	t.Parallel()

	inner := &countingLLMConnectivityClient{calls: map[string]int{}}

	if got := NewCachedLLMConnectivityClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedLLMConnectivityClient(disabled) = %T, want the inner client", got)
	}
}

// TestCachedLLMConnectivityClientInvalidation tests that reads are cached
// and that each write invalidates the datasets it changes.
func TestCachedLLMConnectivityClientInvalidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := map[string]struct {
		write func(client LLMConnectivityClient) error
		// want is the number of calls of each read after reading twice,
		// writing, and reading again.
		want map[string]int
	}{
		"CreateProvider": {
			write: func(client LLMConnectivityClient) error {
				_, _, err := client.CreateProvider(ctx, &sonar.LlmConnectivityCreateProviderOptions{}) //nolint:bodyclose // fake or test server responses

				return err
			},
			want: map[string]int{"GetProviderDefinitions": 1, "ListProviders": 2, "GetProviderMappings": 1},
		},
		"UpdateProvider": {
			write: func(client LLMConnectivityClient) error {
				_, _, err := client.UpdateProvider(ctx, "id", &sonar.LlmConnectivityUpdateProviderOptions{}) //nolint:bodyclose // fake or test server responses

				return err
			},
			want: map[string]int{"GetProviderDefinitions": 1, "ListProviders": 2, "GetProviderMappings": 1},
		},
		"DeleteProvider": {
			write: func(client LLMConnectivityClient) error {
				_, err := client.DeleteProvider(ctx, "id") //nolint:bodyclose // fake or test server responses

				return err
			},
			want: map[string]int{"GetProviderDefinitions": 1, "ListProviders": 2, "GetProviderMappings": 2},
		},
		"UpsertProviderMapping": {
			write: func(client LLMConnectivityClient) error {
				_, _, err := client.UpsertProviderMapping(ctx, &sonar.LlmConnectivityUpsertProviderMappingOptions{}) //nolint:bodyclose // fake or test server responses

				return err
			},
			want: map[string]int{"GetProviderDefinitions": 1, "ListProviders": 1, "GetProviderMappings": 2},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inner := &countingLLMConnectivityClient{calls: map[string]int{}}
			client := NewCachedLLMConnectivityClient(inner, cachetest.NewScoped(t))

			readAll(t, client)
			readAll(t, client)

			err := tc.write(client)
			if err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}

			readAll(t, client)

			for method, want := range tc.want {
				if inner.calls[method] != want {
					t.Errorf("%s calls = %d, want %d (all calls: %v)", method, inner.calls[method], want, inner.calls)
				}
			}

			if inner.calls[name] != 1 {
				t.Errorf("%s calls = %d, want 1", name, inner.calls[name])
			}
		})
	}
}
