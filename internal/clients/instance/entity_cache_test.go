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

// projectsEntityTestNamespace is a namespace depending on projects, only
// used by the tests of this package.
const projectsEntityTestNamespace = "test/instance/projects"

// init declares the dependencies of the test namespace.
func init() {
	cache.DependOn(projectsEntityTestNamespace, cache.EntityProject)
}

// countingProjectsClient is a ProjectsClient counting the project writes.
// Other methods panic through the nil embedded interface.
type countingProjectsClient struct {
	ProjectsClient

	// calls counts the calls of every method.
	calls map[string]int
}

// Delete counts the call.
func (c *countingProjectsClient) Delete(context.Context, *sonar.ProjectsDeleteOptions) (*http.Response, error) {
	c.calls["Delete"]++

	return okResponse(), nil
}

// TestNewCachedProjectsClient tests that the raw client is used when the
// cache is disabled, and that project writes invalidate the namespaces
// depending on projects.
func TestNewCachedProjectsClient(t *testing.T) {
	t.Parallel()

	inner := &countingProjectsClient{calls: map[string]int{}}
	if got := NewCachedProjectsClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedProjectsClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewProjectsClient(newTestConfig()).(*cachedProjectsClient); cached {
		t.Error("NewProjectsClient() returned a cached client while the cache is disabled")
	}

	scoped := cachetest.NewScoped(t)
	key := scoped.Key(projectsEntityTestNamespace, "")

	if !scoped.Store.Set(key, "value", 0) {
		t.Fatal("Set() was rejected")
	}

	resp, err := NewCachedProjectsClient(inner, scoped).Delete(context.Background(), &sonar.ProjectsDeleteOptions{Project: "p"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || inner.calls["Delete"] != 1 {
		t.Fatalf("Delete() = %v, calls %v", err, inner.calls)
	}

	if _, _, found := scoped.Store.Get(key); found {
		t.Error("Delete() kept an entry depending on projects")
	}
}
