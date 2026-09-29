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

package iam

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// stubUsersClient is a UsersClient counting the user writes. Other methods
// panic through the nil embedded interface.
type stubUsersClient struct {
	UsersClient

	// calls counts the calls of every method.
	calls map[string]int
}

// Create counts the call.
func (s *stubUsersClient) Create(context.Context, *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	s.calls["Create"]++

	return &sonar.UserV2{}, okResponse(), nil
}

// Update counts the call.
func (s *stubUsersClient) Update(context.Context, string, *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	s.calls["Update"]++

	return &sonar.UserV2{}, okResponse(), nil
}

// Deactivate counts the call.
func (s *stubUsersClient) Deactivate(context.Context, *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
	s.calls["Deactivate"]++

	return okResponse(), nil
}

// stubProjectsClient is an instance.ProjectsClient counting the project
// writes. Other methods panic through the nil embedded interface.
type stubProjectsClient struct {
	instance.ProjectsClient

	// calls counts the calls of every method.
	calls map[string]int
}

// Create counts the call.
func (s *stubProjectsClient) Create(context.Context, *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error) {
	s.calls["Create"]++

	return &sonar.ProjectsCreate{}, okResponse(), nil
}

// Delete counts the call.
func (s *stubProjectsClient) Delete(context.Context, *sonar.ProjectsDeleteOptions) (*http.Response, error) {
	s.calls["Delete"]++

	return okResponse(), nil
}

// BulkDelete counts the call.
func (s *stubProjectsClient) BulkDelete(context.Context, *sonar.ProjectsBulkDeleteOptions) (*http.Response, error) {
	s.calls["BulkDelete"]++

	return okResponse(), nil
}

// UpdateKey counts the call.
func (s *stubProjectsClient) UpdateKey(context.Context, *sonar.ProjectsUpdateKeyOptions) (*http.Response, error) {
	s.calls["UpdateKey"]++

	return okResponse(), nil
}

// UpdateVisibility counts the call.
func (s *stubProjectsClient) UpdateVisibility(context.Context, *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error) {
	s.calls["UpdateVisibility"]++

	return okResponse(), nil
}

// TestNewCachedUsersClientDisabled tests that the raw client is used when
// the cache is disabled.
func TestNewCachedUsersClientDisabled(t *testing.T) {
	t.Parallel()

	inner := &stubUsersClient{calls: map[string]int{}}
	if got := NewCachedUsersClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedUsersClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewUsersClient(newTestConfig()).(*cachedUsersClient); cached {
		t.Error("NewUsersClient() returned a cached client while the cache is disabled")
	}
}

// TestEntityWritesInvalidatePermissions tests that user and project writes
// invalidate the cached permission searches mentioning users or projects,
// and only those.
func TestEntityWritesInvalidatePermissions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write func(users UsersClient, projects instance.ProjectsClient) error
		// wantGroups and wantUsers are the Groups and Users calls after
		// reading, writing and reading again.
		wantGroups int
		wantUsers  int
	}{
		"UserCreate": {
			write: func(users UsersClient, _ instance.ProjectsClient) error {
				_, resp, err := users.Create(ctx, &sonar.UsersCreateOptionsV2{Login: "alice"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2, wantUsers: 4,
		},
		"UserUpdate": {
			write: func(users UsersClient, _ instance.ProjectsClient) error {
				_, resp, err := users.Update(ctx, "id", &sonar.UsersUpdateOptionsV2{Login: "renamed"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2, wantUsers: 4,
		},
		"UserDeactivate": {
			write: func(users UsersClient, _ instance.ProjectsClient) error {
				resp, err := users.Deactivate(ctx, &sonar.UsersDeactivateOptionsV2{Id: "id"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2, wantUsers: 4,
		},
		"ProjectDelete": {
			write: func(_ UsersClient, projects instance.ProjectsClient) error {
				resp, err := projects.Delete(ctx, &sonar.ProjectsDeleteOptions{Project: testProjectKey}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 4, wantUsers: 4,
		},
		"ProjectBulkDelete": {
			write: func(_ UsersClient, projects instance.ProjectsClient) error {
				resp, err := projects.BulkDelete(ctx, &sonar.ProjectsBulkDeleteOptions{Projects: []string{testProjectKey}}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 4, wantUsers: 4,
		},
		"ProjectUpdateKey": {
			write: func(_ UsersClient, projects instance.ProjectsClient) error {
				resp, err := projects.UpdateKey(ctx, &sonar.ProjectsUpdateKeyOptions{From: testProjectKey, To: "renamed"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 4, wantUsers: 4,
		},
		"ProjectUpdateVisibility": {
			write: func(_ UsersClient, projects instance.ProjectsClient) error {
				resp, err := projects.UpdateVisibility(ctx, &sonar.ProjectsUpdateVisibilityOptions{Project: testProjectKey, Visibility: "private"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 4, wantUsers: 4,
		},
		"ProjectCreate": {
			write: func(_ UsersClient, projects instance.ProjectsClient) error {
				_, resp, err := projects.Create(ctx, &sonar.ProjectsCreateOptions{Project: "new"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 4, wantUsers: 4,
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			permissions := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan"}})
			permissions.users = []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan"}}}
			scoped := cachetest.NewScoped(t)
			users := NewCachedUsersClient(&stubUsersClient{calls: map[string]int{}}, scoped)
			projects := instance.NewCachedProjectsClient(&stubProjectsClient{calls: map[string]int{}}, scoped)

			readEntityPermissions(t, permissions, scoped)

			err := tc.write(users, projects)
			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}

			readEntityPermissions(t, permissions, scoped)

			if got := permissions.calls["Groups"]; got != tc.wantGroups {
				t.Errorf("Groups called %d times, want %d", got, tc.wantGroups)
			}

			if got := permissions.calls["Users"]; got != tc.wantUsers {
				t.Errorf("Users called %d times, want %d", got, tc.wantUsers)
			}
		})
	}
}

// readEntityPermissions reads the global permissions of the test group,
// the permissions of the test group on the test project (one scan), and
// the global and project permissions of a user.
func readEntityPermissions(t *testing.T, client PermissionsClient, scoped cache.Scoped) {
	t.Helper()

	ctx := context.Background()

	for _, projectKey := range []string{"", testProjectKey} {
		_, _, err := GroupPermissions(ctx, client, scoped, testGroupName, projectKey)
		if err != nil {
			t.Fatalf("GroupPermissions(%q) error = %v", projectKey, err)
		}

		_, _, err = UserPermissions(ctx, client, scoped, "alice", projectKey)
		if err != nil {
			t.Fatalf("UserPermissions(%q) error = %v", projectKey, err)
		}
	}
}
