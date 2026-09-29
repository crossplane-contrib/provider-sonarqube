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

package webhook

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestLifecycleWithCacheCreatesOnce tests that, with the observe cache on,
// the Observe following a Create finds the new webhook instead of the list
// cached before it, so that the webhook is created exactly once, and that
// the Observe following a Delete reports it gone.
func TestLifecycleWithCacheCreatesOnce(t *testing.T) {
	t.Parallel()

	var webhooks []sonar.WebhooksDefinition

	calls := map[string]int{}
	inner := &fake.MockWebhooksClient{
		ListFn: func(*sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
			calls["List"]++

			return &sonar.WebhooksList{Webhooks: slices.Clone(webhooks)}, mockHTTPResponse(http.StatusOK), nil
		},
		CreateFn: func(opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
			calls["Create"]++

			created := sonar.WebhooksDefinition{Key: "hook-key", Name: opt.Name, URL: opt.URL}
			webhooks = append(webhooks, created)

			return &sonar.WebhooksCreate{Webhook: created}, mockHTTPResponse(http.StatusOK), nil
		},
		DeleteFn: func(opt *sonar.WebhooksDeleteOptions) (*http.Response, error) {
			calls["Delete"]++

			webhooks = slices.DeleteFunc(webhooks, func(webhook sonar.WebhooksDefinition) bool { return webhook.Key == opt.Webhook })

			return mockHTTPResponse(http.StatusNoContent), nil
		},
	}
	e := &external{
		kubeClient: buildKubeClient(nil).Build(),
		client:     integration.NewCachedWebhooksClient(inner, cachetest.NewScoped(t)),
	}

	// Crossplane defaults the external name to metadata.name before Create.
	webhook := newTestWebhook("test-webhook", nil)

	observation, err := e.Observe(context.Background(), webhook)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() before Create = %+v, %v, want not existing", observation, err)
	}

	_, err = e.Create(context.Background(), webhook)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for range 3 {
		observation, err = e.Observe(context.Background(), webhook)
		if err != nil || !observation.ResourceExists {
			t.Fatalf("Observe() after Create = %+v, %v, want existing", observation, err)
		}
	}

	_, err = e.Delete(context.Background(), webhook)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	observation, err = e.Observe(context.Background(), webhook)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() after Delete = %+v, %v, want not existing", observation, err)
	}

	// One List before Create, one after it (then served from the cache),
	// and one after Delete.
	if calls["Create"] != 1 || calls["List"] != 3 {
		t.Errorf("calls = %v, want 1 Create and 3 List", calls)
	}
}
