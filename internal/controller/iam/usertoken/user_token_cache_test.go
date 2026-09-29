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

package usertoken

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestLifecycleWithCacheGeneratesOnce tests that, with the observe cache
// on, the Observe following a Generate finds the new token instead of the
// list cached before it, so that exactly one token is generated; that a
// renewal is observed with the expiration date of the new token; and that
// the Observe following a Revoke reports the token gone.
func TestLifecycleWithCacheGeneratesOnce(t *testing.T) {
	t.Parallel()

	var tokens []sonar.UserToken

	calls := map[string]int{}
	inner := &fake.MockUserTokensClient{
		SearchFn: func(*sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
			calls["Search"]++

			return &sonar.UserTokensSearch{UserTokens: slices.Clone(tokens)}, mockHTTPOK(), nil
		},
		GenerateFn: func(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
			calls["Generate"]++

			tokens = append(tokens, sonar.UserToken{Name: opt.Name, ExpirationDate: opt.ExpirationDate})

			return &sonar.UserTokensGenerate{Name: opt.Name, Token: "secret", ExpirationDate: opt.ExpirationDate}, mockHTTPOK(), nil
		},
		RevokeFn: func(opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
			calls["Revoke"]++

			tokens = slices.DeleteFunc(tokens, func(token sonar.UserToken) bool { return token.Name == opt.Name })

			return mockHTTPNoContent(), nil
		},
	}
	e := &external{client: iam.NewCachedUserTokensClient(inner, cachetest.NewScoped(t))}
	token := withRenewalDays(newToken(testTokenName, testTokenName), 30)

	observation, err := e.Observe(context.Background(), token)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() before Create = %+v, %v, want not existing", observation, err)
	}

	_, err = e.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for range 3 {
		observation, err = e.Observe(context.Background(), token)
		if err != nil || !observation.ResourceExists {
			t.Fatalf("Observe() after Create = %+v, %v, want existing", observation, err)
		}
	}

	if calls["Generate"] != 1 || calls["Search"] != 2 {
		t.Fatalf("calls after Create = %v, want 1 Generate and 2 Search", calls)
	}

	// Renew with an older expiration date: the next Observe must read the
	// new date from a fresh list.
	tokens[0].ExpirationDate = time.Now().Add(-time.Hour).Format(time.DateOnly)

	_, err = e.Update(context.Background(), token)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	observation, err = e.Observe(context.Background(), token)
	if err != nil || !observation.ResourceExists || !observation.ResourceUpToDate {
		t.Fatalf("Observe() after renewal = %+v, %v, want existing and up to date", observation, err)
	}

	_, err = e.Delete(context.Background(), token)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	observation, err = e.Observe(context.Background(), token)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() after Delete = %+v, %v, want not existing", observation, err)
	}

	if calls["Generate"] != 2 || calls["Revoke"] != 2 || calls["Search"] != 4 {
		t.Errorf("calls = %v, want 2 Generate, 2 Revoke and 4 Search", calls)
	}
}
