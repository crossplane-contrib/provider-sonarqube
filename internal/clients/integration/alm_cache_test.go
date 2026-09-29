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
	"errors"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// The cached client must stand in for every per-ALM settings client.
var (
	_ ALMSettingsClient               = (*cachedALMSettingsClient)(nil)
	_ ALMSettingsAzureClient          = (*cachedALMSettingsClient)(nil)
	_ ALMSettingsBitbucketClient      = (*cachedALMSettingsClient)(nil)
	_ ALMSettingsBitbucketCloudClient = (*cachedALMSettingsClient)(nil)
	_ ALMSettingsGitHubClient         = (*cachedALMSettingsClient)(nil)
	_ ALMSettingsGitLabClient         = (*cachedALMSettingsClient)(nil)
)

// countingALMSettingsClient is an almSettingsAPI counting the calls of the
// methods the tests exercise. Other methods panic through the nil embedded
// interface.
type countingALMSettingsClient struct {
	almSettingsAPI

	calls    map[string]int
	writeErr error
}

// newCountingALMSettingsClient returns a countingALMSettingsClient with no
// call.
func newCountingALMSettingsClient() *countingALMSettingsClient {
	return &countingALMSettingsClient{calls: map[string]int{}}
}

// ListDefinitions counts the call and returns one GitHub definition.
func (c *countingALMSettingsClient) ListDefinitions(context.Context) (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
	c.calls["ListDefinitions"]++

	return &sonar.AlmSettingsListDefinitions{Github: []sonar.GithubDefinition{{Key: "github"}}},
		&http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// Delete counts the call and returns writeErr.
func (c *countingALMSettingsClient) Delete(context.Context, *sonar.AlmSettingsDeleteOptions) (*http.Response, error) {
	return c.write("Delete")
}

// DeleteBinding counts the call and returns writeErr.
func (c *countingALMSettingsClient) DeleteBinding(context.Context, *sonar.AlmSettingsDeleteBindingOptions) (*http.Response, error) {
	return c.write("DeleteBinding")
}

// SetGithubBinding counts the call and returns writeErr.
func (c *countingALMSettingsClient) SetGithubBinding(context.Context, *sonar.AlmSettingsSetGithubBindingOptions) (*http.Response, error) {
	return c.write("SetGithubBinding")
}

// CreateAzure counts the call and returns writeErr.
func (c *countingALMSettingsClient) CreateAzure(context.Context, *sonar.AlmSettingsCreateAzureOptions) (*http.Response, error) {
	return c.write("CreateAzure")
}

// UpdateAzure counts the call and returns writeErr.
func (c *countingALMSettingsClient) UpdateAzure(context.Context, *sonar.AlmSettingsUpdateAzureOptions) (*http.Response, error) {
	return c.write("UpdateAzure")
}

// CreateBitbucket counts the call and returns writeErr.
func (c *countingALMSettingsClient) CreateBitbucket(context.Context, *sonar.AlmSettingsCreateBitbucketOptions) (*http.Response, error) {
	return c.write("CreateBitbucket")
}

// UpdateBitbucket counts the call and returns writeErr.
func (c *countingALMSettingsClient) UpdateBitbucket(context.Context, *sonar.AlmSettingsUpdateBitbucketOptions) (*http.Response, error) {
	return c.write("UpdateBitbucket")
}

// CreateBitbucketCloud counts the call and returns writeErr.
func (c *countingALMSettingsClient) CreateBitbucketCloud(context.Context, *sonar.AlmSettingsCreateBitbucketCloudOptions) (*http.Response, error) {
	return c.write("CreateBitbucketCloud")
}

// UpdateBitbucketCloud counts the call and returns writeErr.
func (c *countingALMSettingsClient) UpdateBitbucketCloud(context.Context, *sonar.AlmSettingsUpdateBitbucketCloudOptions) (*http.Response, error) {
	return c.write("UpdateBitbucketCloud")
}

// CreateGithub counts the call and returns writeErr.
func (c *countingALMSettingsClient) CreateGithub(context.Context, *sonar.AlmSettingsCreateGithubOptions) (*http.Response, error) {
	return c.write("CreateGithub")
}

// UpdateGithub counts the call and returns writeErr.
func (c *countingALMSettingsClient) UpdateGithub(context.Context, *sonar.AlmSettingsUpdateGithubOptions) (*http.Response, error) {
	return c.write("UpdateGithub")
}

// CreateGitlab counts the call and returns writeErr.
func (c *countingALMSettingsClient) CreateGitlab(context.Context, *sonar.AlmSettingsCreateGitlabOptions) (*http.Response, error) {
	return c.write("CreateGitlab")
}

// UpdateGitlab counts the call and returns writeErr.
func (c *countingALMSettingsClient) UpdateGitlab(context.Context, *sonar.AlmSettingsUpdateGitlabOptions) (*http.Response, error) {
	return c.write("UpdateGitlab")
}

// write counts a call to method and returns writeErr.
func (c *countingALMSettingsClient) write(method string) (*http.Response, error) {
	c.calls[method]++

	if c.writeErr != nil {
		return nil, c.writeErr
	}

	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
}

// listDefinitions calls ListDefinitions on client and fails the test on
// error.
func listDefinitions(t *testing.T, client ALMSettingsClient) *sonar.AlmSettingsListDefinitions {
	t.Helper()

	definitions, resp, err := client.ListDefinitions(context.Background()) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || resp == nil || definitions == nil {
		t.Fatalf("ListDefinitions() = %v, %v, %v", definitions, resp, err)
	}

	return definitions
}

// assertListDefinitionsCalls checks that ListDefinitions reached the inner
// client want times.
func assertListDefinitionsCalls(t *testing.T, inner *countingALMSettingsClient, want int) {
	t.Helper()

	if got := inner.calls["ListDefinitions"]; got != want {
		t.Errorf("inner ListDefinitions called %d times, want %d", got, want)
	}
}

// TestNewCachedALMSettingsClientDisabled tests that the raw client is used
// when the cache is disabled.
func TestNewCachedALMSettingsClientDisabled(t *testing.T) {
	t.Parallel()

	inner := newCountingALMSettingsClient()

	if got := newCachedALMSettingsClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("newCachedALMSettingsClient() with a noop store = %T, want the raw client", got)
	}

	// The flag is off by default, so the constructors return the raw SDK
	// client.
	config := common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: "http://localhost:9000"}
	if _, cached := NewALMSettingsGitHubClient(config).(*cachedALMSettingsClient); cached {
		t.Error("NewALMSettingsGitHubClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedALMSettingsClientSharesDefinitionsAcrossKinds tests that the
// clients of every ALM kind on one connection share one list_definitions
// call.
func TestCachedALMSettingsClientSharesDefinitionsAcrossKinds(t *testing.T) {
	t.Parallel()

	inner := newCountingALMSettingsClient()
	scoped := cachetest.NewScoped(t)

	clients := []ALMSettingsClient{
		ALMSettingsGitHubClient(newCachedALMSettingsClient(inner, scoped)),
		ALMSettingsGitLabClient(newCachedALMSettingsClient(inner, scoped)),
		ALMSettingsAzureClient(newCachedALMSettingsClient(inner, scoped)),
		ALMSettingsBitbucketClient(newCachedALMSettingsClient(inner, scoped)),
		ALMSettingsBitbucketCloudClient(newCachedALMSettingsClient(inner, scoped)),
	}

	for _, client := range clients {
		definitions := listDefinitions(t, client)

		if FindGitHubALMDefinitionByKey(&definitions.Github, "github") == nil {
			t.Errorf("ListDefinitions() = %+v, want the github definition", definitions)
		}
	}

	assertListDefinitionsCalls(t, inner, 1)
}

// TestCachedALMSettingsClientWritesInvalidate tests that every write to an
// ALM setting, failed or not, invalidates the definitions.
func TestCachedALMSettingsClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]func(almSettingsAPI) (*http.Response, error){
		"Delete": func(c almSettingsAPI) (*http.Response, error) {
			return c.Delete(ctx, &sonar.AlmSettingsDeleteOptions{Key: "k"})
		},
		"CreateAzure": func(c almSettingsAPI) (*http.Response, error) {
			return c.CreateAzure(ctx, &sonar.AlmSettingsCreateAzureOptions{Key: "k"})
		},
		"UpdateAzure": func(c almSettingsAPI) (*http.Response, error) {
			return c.UpdateAzure(ctx, &sonar.AlmSettingsUpdateAzureOptions{Key: "k"})
		},
		"CreateBitbucket": func(c almSettingsAPI) (*http.Response, error) {
			return c.CreateBitbucket(ctx, &sonar.AlmSettingsCreateBitbucketOptions{Key: "k"})
		},
		"UpdateBitbucket": func(c almSettingsAPI) (*http.Response, error) {
			return c.UpdateBitbucket(ctx, &sonar.AlmSettingsUpdateBitbucketOptions{Key: "k"})
		},
		"CreateBitbucketCloud": func(c almSettingsAPI) (*http.Response, error) {
			return c.CreateBitbucketCloud(ctx, &sonar.AlmSettingsCreateBitbucketCloudOptions{Key: "k"})
		},
		"UpdateBitbucketCloud": func(c almSettingsAPI) (*http.Response, error) {
			return c.UpdateBitbucketCloud(ctx, &sonar.AlmSettingsUpdateBitbucketCloudOptions{Key: "k"})
		},
		"CreateGithub": func(c almSettingsAPI) (*http.Response, error) {
			return c.CreateGithub(ctx, &sonar.AlmSettingsCreateGithubOptions{Key: "k"})
		},
		"UpdateGithub": func(c almSettingsAPI) (*http.Response, error) {
			return c.UpdateGithub(ctx, &sonar.AlmSettingsUpdateGithubOptions{Key: "k"})
		},
		"CreateGitlab": func(c almSettingsAPI) (*http.Response, error) {
			return c.CreateGitlab(ctx, &sonar.AlmSettingsCreateGitlabOptions{Key: "k"})
		},
		"UpdateGitlab": func(c almSettingsAPI) (*http.Response, error) {
			return c.UpdateGitlab(ctx, &sonar.AlmSettingsUpdateGitlabOptions{Key: "k"})
		},
	}

	for name, write := range writes {
		for _, writeErr := range []error{nil, errors.New("partial failure")} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				inner := newCountingALMSettingsClient()
				inner.writeErr = writeErr
				client := newCachedALMSettingsClient(inner, cachetest.NewScoped(t))

				listDefinitions(t, client)

				resp, err := write(client) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if !errors.Is(err, writeErr) {
					t.Fatalf("%s() error = %v, want %v", name, err, writeErr)
				}

				if got := inner.calls[name]; got != 1 {
					t.Errorf("inner %s called %d times, want 1", name, got)
				}

				listDefinitions(t, client)
				assertListDefinitionsCalls(t, inner, 2)
			})
		}
	}
}

// TestCachedALMSettingsClientBindingWritesKeepDefinitions tests that project
// binding writes, which do not change the definitions, keep them cached.
func TestCachedALMSettingsClientBindingWritesKeepDefinitions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner := newCountingALMSettingsClient()
	client := newCachedALMSettingsClient(inner, cachetest.NewScoped(t))

	listDefinitions(t, client)

	resp, err := client.SetGithubBinding(ctx, &sonar.AlmSettingsSetGithubBindingOptions{Project: "p"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("SetGithubBinding() error = %v", err)
	}

	resp, err = client.DeleteBinding(ctx, &sonar.AlmSettingsDeleteBindingOptions{Project: "p"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("DeleteBinding() error = %v", err)
	}

	listDefinitions(t, client)
	assertListDefinitionsCalls(t, inner, 1)
}

// TestCachedALMSettingsClientScopesAreIsolated tests that connections with
// different scopes share no cached definitions.
func TestCachedALMSettingsClientScopesAreIsolated(t *testing.T) {
	t.Parallel()

	store := cachetest.NewStore(t)
	inner := newCountingALMSettingsClient()
	first := newCachedALMSettingsClient(inner, cache.NewScoped(store, t.Name()+"/first"))
	second := newCachedALMSettingsClient(inner, cache.NewScoped(store, t.Name()+"/second"))

	listDefinitions(t, first)
	listDefinitions(t, second)
	assertListDefinitionsCalls(t, inner, 2)

	// A write through one scope does not invalidate the other.
	resp, err := first.Delete(context.Background(), &sonar.AlmSettingsDeleteOptions{Key: "k"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	listDefinitions(t, second)
	assertListDefinitionsCalls(t, inner, 2)
}
