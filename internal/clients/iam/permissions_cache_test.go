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
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// stubPermissionsClient is a PermissionsClient serving group and user
// permissions from in-memory lists, split into pages of the requested
// size, and counting the calls of every method.
type stubPermissionsClient struct {
	// groups are served by Groups, whatever the query.
	groups []sonar.PermissionGroup
	// users are served by Users, whatever the query.
	users []sonar.PermissionUser
	// readErr is returned by Groups and Users when set.
	readErr error
	// calls counts the calls of every method.
	calls map[string]int
	// groupsOptions records the options of every Groups call.
	groupsOptions []sonar.PermissionsGroupsOptions
	// usersOptions records the options of every Users call.
	usersOptions []sonar.PermissionsUsersOptions
}

// newStubPermissionsClient returns a stubPermissionsClient serving groups.
func newStubPermissionsClient(groups ...sonar.PermissionGroup) *stubPermissionsClient {
	return &stubPermissionsClient{groups: groups, calls: map[string]int{}}
}

// AddGroup counts the call.
func (s *stubPermissionsClient) AddGroup(context.Context, *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
	s.calls["AddGroup"]++

	return okResponse(), nil
}

// Groups counts the call and returns the requested page of groups.
func (s *stubPermissionsClient) Groups(_ context.Context, opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
	s.calls["Groups"]++
	s.groupsOptions = append(s.groupsOptions, *opt)

	if s.readErr != nil {
		return nil, nil, s.readErr
	}

	items, paging := page(s.groups, opt.PaginationArgs)

	return &sonar.PermissionsGroups{Groups: items, Paging: paging}, okResponse(), nil
}

// RemoveGroup counts the call.
func (s *stubPermissionsClient) RemoveGroup(context.Context, *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
	s.calls["RemoveGroup"]++

	return okResponse(), nil
}

// AddUser counts the call.
func (s *stubPermissionsClient) AddUser(context.Context, *sonar.PermissionsAddUserOptions) (*http.Response, error) {
	s.calls["AddUser"]++

	return okResponse(), nil
}

// Users counts the call and returns the requested page of users.
func (s *stubPermissionsClient) Users(_ context.Context, opt *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
	s.calls["Users"]++
	s.usersOptions = append(s.usersOptions, *opt)

	if s.readErr != nil {
		return nil, nil, s.readErr
	}

	items, paging := page(s.users, opt.PaginationArgs)

	return &sonar.PermissionsUsers{Users: items, Paging: paging}, okResponse(), nil
}

// RemoveUser counts the call.
func (s *stubPermissionsClient) RemoveUser(context.Context, *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
	s.calls["RemoveUser"]++

	return okResponse(), nil
}

// stubGroupsClient is a GroupsClient counting the group writes. Other
// methods panic through the nil embedded interface.
type stubGroupsClient struct {
	GroupsClient

	// calls counts the calls of every method.
	calls map[string]int
}

// CreateGroup counts the call.
func (s *stubGroupsClient) CreateGroup(context.Context, *sonar.AuthorizationsCreateGroupOptions) (*sonar.AuthorizationsGroup, *http.Response, error) {
	s.calls["CreateGroup"]++

	return &sonar.AuthorizationsGroup{}, okResponse(), nil
}

// UpdateGroup counts the call.
func (s *stubGroupsClient) UpdateGroup(context.Context, string, *sonar.AuthorizationsUpdateGroupOptions) (*sonar.AuthorizationsGroup, *http.Response, error) {
	s.calls["UpdateGroup"]++

	return &sonar.AuthorizationsGroup{}, okResponse(), nil
}

// DeleteGroup counts the call.
func (s *stubGroupsClient) DeleteGroup(context.Context, string) (*http.Response, error) {
	s.calls["DeleteGroup"]++

	return okResponse(), nil
}

// CreateGroupMembership counts the call.
func (s *stubGroupsClient) CreateGroupMembership(context.Context, *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.AuthorizationsGroupMembership, *http.Response, error) {
	s.calls["CreateGroupMembership"]++

	return &sonar.AuthorizationsGroupMembership{}, okResponse(), nil
}

// page returns the requested page of items and its paging.
func page[T any](items []T, args sonar.PaginationArgs) ([]T, sonar.Paging) {
	start := min(int((args.Page-1)*args.PageSize), len(items))
	end := min(start+int(args.PageSize), len(items))

	return items[start:end], sonar.Paging{PageIndex: args.Page, PageSize: args.PageSize, Total: int64(len(items))}
}

// newTestConfig returns the config of a local SonarQube.
func newTestConfig() common.Config {
	return common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: "http://localhost:9000"}
}

// okResponse returns a fresh 200 response.
func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// manyGroups returns count groups named group-<index>, each with the scan
// permission, followed by extra.
func manyGroups(count int, extra ...sonar.PermissionGroup) []sonar.PermissionGroup {
	groups := make([]sonar.PermissionGroup, 0, count+len(extra))
	for index := range count {
		groups = append(groups, sonar.PermissionGroup{Name: fmt.Sprintf("group-%d", index), Permissions: []string{"scan"}})
	}

	return append(groups, extra...)
}

// TestGroupPermissionsIndex tests building the index of group permissions.
func TestGroupPermissionsIndex(t *testing.T) {
	t.Parallel()

	devs := sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"user", "codeviewer"}}

	t.Run("MultiPageWithGroupOnLastPage", func(t *testing.T) {
		t.Parallel()

		client := newStubPermissionsClient(manyGroups(2*int(permissionsPageSize), devs)...)

		index, err := GroupPermissionsIndex(context.Background(), client, cachetest.Disabled(), testProjectKey)
		if err != nil {
			t.Fatalf("GroupPermissionsIndex() error = %v", err)
		}

		if got := client.calls["Groups"]; got != 3 {
			t.Errorf("Groups called %d times, want 3", got)
		}

		if diff := cmp.Diff(devs.Permissions, index[testGroupName]); diff != "" {
			t.Errorf("index[%q] mismatch (-want +got):\n%s", testGroupName, diff)
		}

		if _, found := index["absent"]; found {
			t.Error("index contains an absent group")
		}

		// The index is built from the unfiltered, project-scoped list.
		for _, opt := range client.groupsOptions {
			if opt.Query != "" || opt.ProjectKey != testProjectKey || opt.PageSize != permissionsPageSize {
				t.Errorf("Groups options = %+v, want no query, project %q and page size %d", opt, testProjectKey, permissionsPageSize)
			}
		}
	})

	t.Run("ZeroPageSize", func(t *testing.T) {
		t.Parallel()

		client := &zeroPagingPermissionsClient{stubPermissionsClient: newStubPermissionsClient(devs)}

		_, err := GroupPermissionsIndex(context.Background(), client, cachetest.Disabled(), testProjectKey)
		if err == nil || !strings.Contains(err.Error(), "zero PageSize") {
			t.Errorf("GroupPermissionsIndex() error = %v, want a zero PageSize error", err)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		t.Parallel()

		client := newStubPermissionsClient()
		client.readErr = errors.New("boom")

		_, err := GroupPermissionsIndex(context.Background(), client, cachetest.Disabled(), testProjectKey)
		if err == nil || !strings.Contains(err.Error(), "cannot list group permissions") {
			t.Errorf("GroupPermissionsIndex() error = %v, want a list error", err)
		}
	})
}

// zeroPagingPermissionsClient is a stubPermissionsClient whose Groups
// responses carry a zero page size.
type zeroPagingPermissionsClient struct {
	*stubPermissionsClient
}

// Groups returns the groups of the stub with a zero page size.
func (z *zeroPagingPermissionsClient) Groups(ctx context.Context, opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
	result, resp, err := z.stubPermissionsClient.Groups(ctx, opt)
	if result != nil {
		result.Paging.PageSize = 0
	}

	return result, resp, err
}

// TestGroupPermissions tests looking up the permissions of one group.
func TestGroupPermissions(t *testing.T) {
	t.Parallel()

	devs := sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan", "admin"}}

	cases := map[string]struct {
		groups          []sonar.PermissionGroup
		readErr         error
		projectKey      string
		wantPermissions []string
		wantFound       bool
		wantQuery       string
		wantErrSubstr   string
	}{
		"GlobalFound": {
			groups:          []sonar.PermissionGroup{{Name: "other"}, devs},
			wantPermissions: devs.Permissions,
			wantFound:       true,
			wantQuery:       testGroupName,
		},
		"GlobalFoundOnSecondPage": {
			groups:          manyGroups(int(permissionsPageSize), devs),
			wantPermissions: devs.Permissions,
			wantFound:       true,
			wantQuery:       testGroupName,
		},
		"GlobalNotFound": {
			groups:    []sonar.PermissionGroup{{Name: "other"}},
			wantQuery: testGroupName,
		},
		"GlobalAPIError": {
			readErr:       errors.New("boom"),
			wantErrSubstr: "cannot search group permissions",
		},
		// SonarQube drops project-scoped results when q is combined with
		// projectKey: the project path must read the unfiltered index.
		"ProjectFoundWithoutQuery": {
			groups:          []sonar.PermissionGroup{{Name: "other", Permissions: []string{"admin"}}, devs},
			projectKey:      testProjectKey,
			wantPermissions: devs.Permissions,
			wantFound:       true,
		},
		"ProjectNotFound": {
			groups:     []sonar.PermissionGroup{{Name: "other", Permissions: []string{"admin"}}},
			projectKey: testProjectKey,
		},
		"ProjectAPIError": {
			readErr:       errors.New("boom"),
			projectKey:    testProjectKey,
			wantErrSubstr: "cannot list group permissions",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newStubPermissionsClient(tc.groups...)
			client.readErr = tc.readErr

			permissions, found, err := GroupPermissions(context.Background(), client, cachetest.Disabled(), testGroupName, tc.projectKey)

			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("GroupPermissions() error = %v, want containing %q", err, tc.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("GroupPermissions() error = %v", err)
			}

			if found != tc.wantFound {
				t.Errorf("GroupPermissions() found = %v, want %v", found, tc.wantFound)
			}

			if diff := cmp.Diff(tc.wantPermissions, permissions); diff != "" {
				t.Errorf("GroupPermissions() mismatch (-want +got):\n%s", diff)
			}

			for _, opt := range client.groupsOptions {
				if opt.Query != tc.wantQuery || opt.ProjectKey != tc.projectKey {
					t.Errorf("Groups options = %+v, want query %q and project %q", opt, tc.wantQuery, tc.projectKey)
				}
			}
		})
	}
}

// TestUserPermissions tests looking up the permissions of one user.
func TestUserPermissions(t *testing.T) {
	t.Parallel()

	alice := sonar.PermissionUser{Login: "alice", Permissions: []string{"scan", "admin"}}

	cases := map[string]struct {
		users           []sonar.PermissionUser
		readErr         error
		projectKey      string
		wantPermissions []string
		wantFound       bool
		wantErrSubstr   string
	}{
		"GlobalFound": {
			users:           []sonar.PermissionUser{{Login: "bob"}, alice},
			wantPermissions: alice.Permissions,
			wantFound:       true,
		},
		"ProjectFound": {
			users:           []sonar.PermissionUser{alice},
			projectKey:      testProjectKey,
			wantPermissions: alice.Permissions,
			wantFound:       true,
		},
		"FoundOnSecondPage": {
			users:           append(slices.Repeat([]sonar.PermissionUser{{Login: "bob"}}, int(permissionsPageSize)), alice),
			wantPermissions: alice.Permissions,
			wantFound:       true,
		},
		"NotFound": {
			users: []sonar.PermissionUser{{Login: "bob"}},
		},
		"APIError": {
			readErr:       errors.New("boom"),
			wantErrSubstr: "cannot search user permissions",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newStubPermissionsClient()
			client.users = tc.users
			client.readErr = tc.readErr

			permissions, found, err := UserPermissions(context.Background(), client, cachetest.Disabled(), "alice", tc.projectKey)

			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("UserPermissions() error = %v, want containing %q", err, tc.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("UserPermissions() error = %v", err)
			}

			if found != tc.wantFound {
				t.Errorf("UserPermissions() found = %v, want %v", found, tc.wantFound)
			}

			if diff := cmp.Diff(tc.wantPermissions, permissions); diff != "" {
				t.Errorf("UserPermissions() mismatch (-want +got):\n%s", diff)
			}

			// Users are searched by login, including on a project.
			for _, opt := range client.usersOptions {
				if opt.Query != "alice" || opt.ProjectKey != tc.projectKey {
					t.Errorf("Users options = %+v, want query %q and project %q", opt, "alice", tc.projectKey)
				}
			}
		})
	}
}

// TestGroupPermissionsIndexSharedByProjectLookups tests that lookups of
// many groups on one project share one paginated scan per TTL, and that
// projects are cached separately.
func TestGroupPermissionsIndexSharedByProjectLookups(t *testing.T) {
	t.Parallel()

	client := newStubPermissionsClient(manyGroups(int(permissionsPageSize), sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"user"}})...)
	scoped := cachetest.NewScoped(t)

	for _, group := range []string{testGroupName, "group-0", "group-1", "absent"} {
		_, _, err := GroupPermissions(context.Background(), client, scoped, group, testProjectKey)
		if err != nil {
			t.Fatalf("GroupPermissions(%q) error = %v", group, err)
		}
	}

	// One scan of the two pages, whatever the number of lookups.
	if got := client.calls["Groups"]; got != 2 {
		t.Errorf("Groups called %d times, want 2", got)
	}

	_, _, err := GroupPermissions(context.Background(), client, scoped, testGroupName, "other-project")
	if err != nil {
		t.Fatalf("GroupPermissions() error = %v", err)
	}

	if got := client.calls["Groups"]; got != 4 {
		t.Errorf("Groups called %d times after a lookup on another project, want 4", got)
	}
}

// TestPermissionsSearchesCachedPerQuery tests that global group lookups and
// user lookups are cached per name.
func TestPermissionsSearchesCachedPerQuery(t *testing.T) {
	t.Parallel()

	client := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan"}})
	client.users = []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan"}}}
	scoped := cachetest.NewScoped(t)
	ctx := context.Background()

	for range 3 {
		_, _, err := GroupPermissions(ctx, client, scoped, testGroupName, "")
		if err != nil {
			t.Fatalf("GroupPermissions() error = %v", err)
		}

		_, _, err = UserPermissions(ctx, client, scoped, "alice", "")
		if err != nil {
			t.Fatalf("UserPermissions() error = %v", err)
		}

		_, _, err = UserPermissions(ctx, client, scoped, "alice", testProjectKey)
		if err != nil {
			t.Fatalf("UserPermissions() error = %v", err)
		}
	}

	if got := client.calls["Groups"]; got != 1 {
		t.Errorf("Groups called %d times, want 1", got)
	}

	// Global and project user permissions are distinct entries.
	if got := client.calls["Users"]; got != 2 {
		t.Errorf("Users called %d times, want 2", got)
	}

	_, _, err := GroupPermissions(ctx, client, scoped, "other", "")
	if err != nil {
		t.Fatalf("GroupPermissions() error = %v", err)
	}

	if got := client.calls["Groups"]; got != 2 {
		t.Errorf("Groups called %d times after a lookup of another group, want 2", got)
	}
}

// TestGroupPermissionsReturnsCopy tests that mutating returned permissions
// does not corrupt the cache.
func TestGroupPermissionsReturnsCopy(t *testing.T) {
	t.Parallel()

	client := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan"}})
	scoped := cachetest.NewScoped(t)

	for _, projectKey := range []string{"", testProjectKey} {
		first, _, err := GroupPermissions(context.Background(), client, scoped, testGroupName, projectKey)
		if err != nil {
			t.Fatalf("GroupPermissions() error = %v", err)
		}

		first[0] = "mutated"

		second, _, err := GroupPermissions(context.Background(), client, scoped, testGroupName, projectKey)
		if err != nil {
			t.Fatalf("GroupPermissions() error = %v", err)
		}

		if diff := cmp.Diff([]string{"scan"}, second); diff != "" {
			t.Errorf("GroupPermissions(%q) after mutation mismatch (-want +got):\n%s", projectKey, diff)
		}
	}
}

// TestNewCachedPermissionsClientsDisabled tests that the raw clients are
// used when the cache is disabled.
func TestNewCachedPermissionsClientsDisabled(t *testing.T) {
	t.Parallel()

	permissions := newStubPermissionsClient()
	if got := NewCachedPermissionsClient(permissions, cachetest.Disabled()); got != permissions {
		t.Errorf("NewCachedPermissionsClient() with a noop store = %T, want the raw client", got)
	}

	groups := &stubGroupsClient{calls: map[string]int{}}
	if got := NewCachedGroupsClient(groups, cachetest.Disabled()); got != groups {
		t.Errorf("NewCachedGroupsClient() with a noop store = %T, want the raw client", got)
	}

	// The flag is off by default, so the constructors return the raw SDK
	// clients.
	if _, cached := NewPermissionsClient(newTestConfig()).(*cachedPermissionsClient); cached {
		t.Error("NewPermissionsClient() returned a cached client while the cache is disabled")
	}

	if _, cached := NewGroupsClient(newTestConfig()).(*cachedGroupsClient); cached {
		t.Error("NewGroupsClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedPermissionsClientWritesInvalidate tests that permission writes
// invalidate the searches of their subject type only.
func TestCachedPermissionsClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write      func(PermissionsClient) error
		wantGroups int
		wantUsers  int
	}{
		"AddGroup": {
			write: func(c PermissionsClient) error {
				resp, err := c.AddGroup(ctx, GeneratePermissionsAddGroupOptions(testGroupName, "scan", nil)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2, wantUsers: 1,
		},
		"RemoveGroup": {
			write: func(c PermissionsClient) error {
				resp, err := c.RemoveGroup(ctx, GeneratePermissionsRemoveGroupOptions(testGroupName, "scan", nil)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2, wantUsers: 1,
		},
		"AddUser": {
			write: func(c PermissionsClient) error {
				resp, err := c.AddUser(ctx, GeneratePermissionsAddUserOptions("alice", "scan", nil)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 1, wantUsers: 2,
		},
		"RemoveUser": {
			write: func(c PermissionsClient) error {
				resp, err := c.RemoveUser(ctx, GeneratePermissionsRemoveUserOptions("alice", "scan", nil)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 1, wantUsers: 2,
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inner := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan"}})
			inner.users = []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan"}}}
			scoped := cachetest.NewScoped(t)
			client := NewCachedPermissionsClient(inner, scoped)

			readAllPermissions(t, client, scoped)

			err := tc.write(client)
			if err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}

			if got := inner.calls[name]; got != 1 {
				t.Errorf("inner %s called %d times, want 1", name, got)
			}

			readAllPermissions(t, client, scoped)

			if got := inner.calls["Groups"]; got != tc.wantGroups {
				t.Errorf("Groups called %d times, want %d", got, tc.wantGroups)
			}

			if got := inner.calls["Users"]; got != tc.wantUsers {
				t.Errorf("Users called %d times, want %d", got, tc.wantUsers)
			}
		})
	}
}

// TestCachedPermissionsClientRemoveGroupReportsNotFound tests the deletion
// path: once the last permission of a group is removed, the next lookup
// reports the group as not found instead of serving the stale index.
func TestCachedPermissionsClientRemoveGroupReportsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"user"}})
	scoped := cachetest.NewScoped(t)
	client := NewCachedPermissionsClient(inner, scoped)

	_, found, err := GroupPermissions(ctx, client, scoped, testGroupName, testProjectKey)
	if err != nil || !found {
		t.Fatalf("GroupPermissions() = %v, %v, want found", found, err)
	}

	// SonarQube only lists groups having at least one permission.
	inner.groups = nil

	resp, err := client.RemoveGroup(ctx, GeneratePermissionsRemoveGroupOptions(testGroupName, "user", new(testProjectKey))) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("RemoveGroup() error = %v", err)
	}

	_, found, err = GroupPermissions(ctx, client, scoped, testGroupName, testProjectKey)
	if err != nil || found {
		t.Errorf("GroupPermissions() after RemoveGroup = %v, %v, want not found", found, err)
	}
}

// TestCachedGroupsClientWritesInvalidate tests that creating, renaming or
// deleting a group invalidates the group permissions, and that membership
// writes do not.
func TestCachedGroupsClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write      func(GroupsClient) error
		wantGroups int
	}{
		"CreateGroup": {
			write: func(c GroupsClient) error {
				_, resp, err := c.CreateGroup(ctx, &sonar.AuthorizationsCreateGroupOptions{Name: testGroupName}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2,
		},
		"UpdateGroup": {
			write: func(c GroupsClient) error {
				_, resp, err := c.UpdateGroup(ctx, "id", &sonar.AuthorizationsUpdateGroupOptions{Name: "renamed"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2,
		},
		"DeleteGroup": {
			write: func(c GroupsClient) error {
				resp, err := c.DeleteGroup(ctx, "id") //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 2,
		},
		"CreateGroupMembership": {
			write: func(c GroupsClient) error {
				_, resp, err := c.CreateGroupMembership(ctx, &sonar.AuthorizationsCreateGroupMembershipOptions{GroupId: "id", UserId: "user"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			wantGroups: 1,
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			permissions := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"scan"}})
			scoped := cachetest.NewScoped(t)
			groups := &stubGroupsClient{calls: map[string]int{}}
			client := NewCachedGroupsClient(groups, scoped)

			readGroupPermissions(t, permissions, scoped)

			err := tc.write(client)
			if err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}

			if got := groups.calls[name]; got != 1 {
				t.Errorf("inner %s called %d times, want 1", name, got)
			}

			readGroupPermissions(t, permissions, scoped)

			if got := permissions.calls["Groups"]; got != tc.wantGroups {
				t.Errorf("Groups called %d times, want %d", got, tc.wantGroups)
			}
		})
	}
}

// TestCachedGroupsClientRenameReportsOldNameNotFound tests that after a
// group rename, a lookup of its old name reports it as not found.
func TestCachedGroupsClientRenameReportsOldNameNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	permissions := newStubPermissionsClient(sonar.PermissionGroup{Name: testGroupName, Permissions: []string{"user"}})
	scoped := cachetest.NewScoped(t)
	groups := NewCachedGroupsClient(&stubGroupsClient{calls: map[string]int{}}, scoped)

	_, found, err := GroupPermissions(ctx, permissions, scoped, testGroupName, testProjectKey)
	if err != nil || !found {
		t.Fatalf("GroupPermissions() = %v, %v, want found", found, err)
	}

	permissions.groups = []sonar.PermissionGroup{{Name: "renamed", Permissions: []string{"user"}}}

	_, resp, err := groups.UpdateGroup(ctx, "id", &sonar.AuthorizationsUpdateGroupOptions{Name: "renamed"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("UpdateGroup() error = %v", err)
	}

	_, found, err = GroupPermissions(ctx, permissions, scoped, testGroupName, testProjectKey)
	if err != nil || found {
		t.Errorf("GroupPermissions() of the old name = %v, %v, want not found", found, err)
	}
}

// readAllPermissions looks up the global permissions of the test group and
// of a user through client.
func readAllPermissions(t *testing.T, client PermissionsClient, scoped cache.Scoped) {
	t.Helper()

	readGroupPermissions(t, client, scoped)

	_, _, err := UserPermissions(context.Background(), client, scoped, "alice", "")
	if err != nil {
		t.Fatalf("UserPermissions() error = %v", err)
	}
}

// readGroupPermissions looks up the global permissions of the test group
// through client.
func readGroupPermissions(t *testing.T, client PermissionsClient, scoped cache.Scoped) {
	t.Helper()

	_, _, err := GroupPermissions(context.Background(), client, scoped, testGroupName, "")
	if err != nil {
		t.Fatalf("GroupPermissions() error = %v", err)
	}
}
