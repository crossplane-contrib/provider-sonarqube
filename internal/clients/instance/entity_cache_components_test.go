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

package instance

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// TestNewCachedComponentsClients tests that the raw application and
// portfolio clients are used when the cache is disabled, and that deleting
// an application or a portfolio invalidates the namespaces depending on
// projects.
func TestNewCachedComponentsClients(t *testing.T) {
	t.Parallel()

	var (
		applications ApplicationsClient = &cachedApplicationsClient{}
		portfolios   PortfoliosClient   = &cachedPortfoliosClient{}
	)

	if got := NewCachedApplicationsClient(applications, cachetest.Disabled()); got != applications {
		t.Errorf("NewCachedApplicationsClient() with a noop store = %T, want the raw client", got)
	}

	if got := NewCachedPortfoliosClient(portfolios, cachetest.Disabled()); got != portfolios {
		t.Errorf("NewCachedPortfoliosClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewApplicationsClient(newTestConfig()).(*cachedApplicationsClient); cached {
		t.Error("NewApplicationsClient() returned a cached client while the cache is disabled")
	}

	if _, cached := NewPortfoliosClient(newTestConfig()).(*cachedPortfoliosClient); cached {
		t.Error("NewPortfoliosClient() returned a cached client while the cache is disabled")
	}

	deletes := map[string]func(scoped cache.Scoped) (*http.Response, error){
		"Application": func(scoped cache.Scoped) (*http.Response, error) {
			return NewCachedApplicationsClient(&deletingApplicationsClient{}, scoped).Delete(context.Background(), &sonar.ApplicationsDeleteOptions{Application: "app"})
		},
		"Portfolio": func(scoped cache.Scoped) (*http.Response, error) {
			return NewCachedPortfoliosClient(&deletingPortfoliosClient{}, scoped).Delete(context.Background(), &sonar.ViewsDeleteOptions{Key: "portfolio"})
		},
	}

	for name, deleteComponent := range deletes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scoped := cachetest.NewScoped(t)
			key := scoped.Key(projectsEntityTestNamespace, "")

			if !scoped.Store.Set(key, "value", 0) {
				t.Fatal("Set() was rejected")
			}

			resp, err := deleteComponent(scoped) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}

			if _, _, found := scoped.Store.Get(key); found {
				t.Error("Delete() kept an entry depending on projects")
			}
		})
	}
}

// deletingApplicationsClient is an ApplicationsClient accepting deletions.
// Other methods panic through the nil embedded interface.
type deletingApplicationsClient struct {
	ApplicationsClient
}

// Delete succeeds.
func (*deletingApplicationsClient) Delete(context.Context, *sonar.ApplicationsDeleteOptions) (*http.Response, error) {
	return okResponse(), nil
}

// deletingPortfoliosClient is a PortfoliosClient accepting deletions.
// Other methods panic through the nil embedded interface.
type deletingPortfoliosClient struct {
	PortfoliosClient
}

// Delete succeeds.
func (*deletingPortfoliosClient) Delete(context.Context, *sonar.ViewsDeleteOptions) (*http.Response, error) {
	return okResponse(), nil
}
