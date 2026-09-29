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

package permissions

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// cacheTestProjectKey is the project of the project-scoped permissions of
// the cache tests.
const cacheTestProjectKey = "my-project"

// serveGroups returns a GroupsFn serving the requested page of *groups,
// and counting its calls in calls.
func serveGroups(groups *[]sonar.PermissionGroup, calls *int) func(*sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
	return func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
		*calls++

		start := min(int((opt.Page-1)*opt.PageSize), len(*groups))
		end := min(start+int(opt.PageSize), len(*groups))

		return &sonar.PermissionsGroups{
			Groups: (*groups)[start:end],
			Paging: sonar.Paging{PageIndex: opt.Page, PageSize: opt.PageSize, Total: int64(len(*groups))},
		}, mockHTTPResponse(http.StatusOK), nil
	}
}

// TestObserveProjectPermissionsShareOneScan tests that K project-scoped
// Permissions resources on one project share one paginated scan per TTL.
func TestObserveProjectPermissionsShareOneScan(t *testing.T) {
	t.Parallel()

	const resources = 5

	groups := make([]sonar.PermissionGroup, 0, 150)
	for index := range 150 {
		groups = append(groups, sonar.PermissionGroup{Name: fmt.Sprintf("group-%d", index), Permissions: []string{"user"}})
	}

	calls := 0
	e := &external{
		client: &fake.MockPermissionsClient{GroupsFn: serveGroups(&groups, &calls)}, //nolint:bodyclose // a fake GroupsFn, not a response
		cache:  cachetest.NewScoped(t),
	}

	for index := range resources {
		// Spread the groups over both pages.
		name := fmt.Sprintf("group-%d", index*30)
		p := newTestGroupPermissionsWithProject("group:"+name+":"+cacheTestProjectKey, name, cacheTestProjectKey, []string{"user"})

		observation, err := e.Observe(context.Background(), p)
		if err != nil {
			t.Fatalf("Observe(%q) error = %v", name, err)
		}

		if !observation.ResourceExists || !observation.ResourceUpToDate {
			t.Errorf("Observe(%q) = %+v, want existing and up to date", name, observation)
		}
	}

	if calls != 2 {
		t.Errorf("Groups called %d times for %d resources, want 2 (one scan of two pages)", calls, resources)
	}
}

// TestObserveAfterWritesUsesFreshPermissions tests that the permission
// writes of Create and Delete invalidate the cached permissions, so that
// the next Observe sees them: Create is not repeated, and a deleted
// resource without permissions left lets its finalizer be removed.
func TestObserveAfterWritesUsesFreshPermissions(t *testing.T) {
	t.Parallel()

	var groups []sonar.PermissionGroup

	calls := 0
	scoped := cachetest.NewScoped(t)
	inner := &fake.MockPermissionsClient{
		GroupsFn: serveGroups(&groups, &calls), //nolint:bodyclose // a fake GroupsFn, not a response
		AddGroupFn: func(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
			groups = []sonar.PermissionGroup{{Name: opt.GroupName, Permissions: []string{opt.Permission}}}

			return mockHTTPResponse(http.StatusNoContent), nil
		},
		RemoveGroupFn: func(opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
			groups = slices.DeleteFunc(groups, func(group sonar.PermissionGroup) bool { return group.Name == opt.GroupName })

			return mockHTTPResponse(http.StatusNoContent), nil
		},
	}
	e := &external{client: iam.NewCachedPermissionsClient(inner, scoped), cache: scoped}
	p := newTestGroupPermissionsWithProject("group:devs:"+cacheTestProjectKey, "devs", cacheTestProjectKey, []string{"user"})

	observation, err := e.Observe(context.Background(), p)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() before Create = %+v, %v, want not existing", observation, err)
	}

	_, err = e.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	observation, err = e.Observe(context.Background(), p)
	if err != nil || !observation.ResourceExists || !observation.ResourceUpToDate {
		t.Fatalf("Observe() after Create = %+v, %v, want existing and up to date", observation, err)
	}

	_, err = e.Delete(context.Background(), p)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	p.DeletionTimestamp = new(metav1.Now())

	observation, err = e.Observe(context.Background(), p)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() after Delete = %+v, %v, want not existing", observation, err)
	}

	if calls != 3 {
		t.Errorf("Groups called %d times, want 3 (one per Observe after a write)", calls)
	}
}
