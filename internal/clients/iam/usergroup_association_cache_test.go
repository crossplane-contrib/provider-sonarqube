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

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// stubQualityGatesClient is a QualityGateUsergroupAssociationClient keeping
// the selected groups and users of every gate in memory, and counting the
// calls of every method. Gates absent from groups do not exist.
type stubQualityGatesClient struct {
	// groups are the selected group names, by gate.
	groups map[string][]string
	// users are the selected logins, by gate.
	users map[string][]string
	// readErr is returned by the searches when set.
	readErr error
	// calls counts the calls of every method.
	calls map[string]int
	// options records the options of every search.
	options []any
}

// newStubQualityGatesClient returns a stubQualityGatesClient with one gate
// without any selection.
func newStubQualityGatesClient(gateName string) *stubQualityGatesClient {
	return &stubQualityGatesClient{
		groups: map[string][]string{gateName: nil},
		users:  map[string][]string{gateName: nil},
		calls:  map[string]int{},
	}
}

// AddGroup counts the call and selects the group.
func (s *stubQualityGatesClient) AddGroup(_ context.Context, opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error) {
	s.calls["AddGroup"]++
	s.groups[opt.GateName] = append(s.groups[opt.GateName], opt.GroupName)

	return okResponse(), nil
}

// AddUser counts the call and selects the user.
func (s *stubQualityGatesClient) AddUser(_ context.Context, opt *sonar.QualitygatesAddUserOptions) (*http.Response, error) {
	s.calls["AddUser"]++
	s.users[opt.GateName] = append(s.users[opt.GateName], opt.Login)

	return okResponse(), nil
}

// RemoveGroup counts the call and deselects the group.
func (s *stubQualityGatesClient) RemoveGroup(_ context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error) {
	s.calls["RemoveGroup"]++
	s.groups[opt.GateName] = slices.DeleteFunc(s.groups[opt.GateName], func(name string) bool { return name == opt.GroupName })

	return okResponse(), nil
}

// RemoveUser counts the call and deselects the user.
func (s *stubQualityGatesClient) RemoveUser(_ context.Context, opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error) {
	s.calls["RemoveUser"]++
	s.users[opt.GateName] = slices.DeleteFunc(s.users[opt.GateName], func(login string) bool { return login == opt.Login })

	return okResponse(), nil
}

// SearchGroups counts the call and returns the requested page of selected
// groups, or a 404 when the gate does not exist.
func (s *stubQualityGatesClient) SearchGroups(_ context.Context, opt *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
	s.calls["SearchGroups"]++
	s.options = append(s.options, *opt)

	names, exists := s.groups[opt.GateName]

	err := s.searchError(exists)
	if err != nil {
		return nil, searchResponse(err), err
	}

	groups := make([]sonar.QualityGateGroup, 0, len(names))
	for _, name := range names {
		groups = append(groups, sonar.QualityGateGroup{Name: name, Selected: true})
	}

	items, paging := page(groups, opt.PaginationArgs)

	return &sonar.QualitygatesSearchGroups{Groups: items, Paging: paging}, okResponse(), nil
}

// SearchUsers counts the call and returns the requested page of selected
// users, or a 404 when the gate does not exist.
func (s *stubQualityGatesClient) SearchUsers(_ context.Context, opt *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error) {
	s.calls["SearchUsers"]++
	s.options = append(s.options, *opt)

	logins, exists := s.users[opt.GateName]

	err := s.searchError(exists)
	if err != nil {
		return nil, searchResponse(err), err
	}

	users := make([]sonar.QualityGateUser, 0, len(logins))
	for _, login := range logins {
		users = append(users, sonar.QualityGateUser{Login: login, Selected: true})
	}

	items, paging := page(users, opt.PaginationArgs)

	return &sonar.QualitygatesSearchUsers{Users: items, Paging: paging}, okResponse(), nil
}

// searchError returns the error of a search: readErr, or errGateNotFound
// when the gate does not exist.
func (s *stubQualityGatesClient) searchError(exists bool) error {
	if s.readErr != nil {
		return s.readErr
	}

	if !exists {
		return errGateNotFound
	}

	return nil
}

// searchResponse returns the response of a search failing with err: a 404
// for errGateNotFound.
func searchResponse(err error) *http.Response {
	if errors.Is(err, errGateNotFound) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}
	}

	return nil
}

// errGateNotFound is the error of a search on a missing gate.
var errGateNotFound = errors.New("not found")

// stubQualityProfilesClient is a QualityProfileUsergroupAssociationClient
// counting the calls of every method and recording the search options.
// Every profile has one selected group and one selected user.
type stubQualityProfilesClient struct {
	// calls counts the calls of every method.
	calls map[string]int
	// options records the options of every search.
	options []any
}

// AddGroup counts the call.
func (s *stubQualityProfilesClient) AddGroup(context.Context, *sonar.QualityprofilesAddGroupOptions) (*http.Response, error) {
	s.calls["AddGroup"]++

	return okResponse(), nil
}

// AddUser counts the call.
func (s *stubQualityProfilesClient) AddUser(context.Context, *sonar.QualityprofilesAddUserOptions) (*http.Response, error) {
	s.calls["AddUser"]++

	return okResponse(), nil
}

// RemoveGroup counts the call.
func (s *stubQualityProfilesClient) RemoveGroup(context.Context, *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error) {
	s.calls["RemoveGroup"]++

	return okResponse(), nil
}

// RemoveUser counts the call.
func (s *stubQualityProfilesClient) RemoveUser(context.Context, *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error) {
	s.calls["RemoveUser"]++

	return okResponse(), nil
}

// SearchGroups counts the call and returns one selected group.
func (s *stubQualityProfilesClient) SearchGroups(_ context.Context, opt *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
	s.calls["SearchGroups"]++
	s.options = append(s.options, *opt)

	return &sonar.QualityprofilesSearchGroups{
		Groups: []sonar.QualityprofilesProfileGroup{{Name: testGroupName, Selected: true}},
		Paging: sonar.Paging{PageIndex: 1, PageSize: opt.PageSize, Total: 1},
	}, okResponse(), nil
}

// SearchUsers counts the call and returns one selected user.
func (s *stubQualityProfilesClient) SearchUsers(_ context.Context, opt *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error) {
	s.calls["SearchUsers"]++
	s.options = append(s.options, *opt)

	return &sonar.QualityprofilesSearchUsers{
		Users:  []sonar.QualityprofilesProfileUser{{Login: "alice", Selected: true}},
		Paging: sonar.Paging{PageIndex: 1, PageSize: opt.PageSize, Total: 1},
	}, okResponse(), nil
}

// testGateName is the Quality Gate of the association cache tests.
const testGateName = "my-gate"

// TestQualityGateSelectedIndexes tests reading the selected groups and
// users of a Quality Gate over several pages, without name filter.
func TestQualityGateSelectedIndexes(t *testing.T) {
	t.Parallel()

	client := newStubQualityGatesClient(testGateName)
	for index := range int(selectedPageSize) + 1 {
		client.groups[testGateName] = append(client.groups[testGateName], fmt.Sprintf("group-%d", index))
	}

	client.users[testGateName] = []string{"alice"}

	groups, found, err := QualityGateSelectedGroups(context.Background(), client, cachetest.Disabled(), testGateName)
	if err != nil || !found {
		t.Fatalf("QualityGateSelectedGroups() = %v, %v, want found", found, err)
	}

	if len(groups) != int(selectedPageSize)+1 || client.calls["SearchGroups"] != 2 {
		t.Errorf("QualityGateSelectedGroups() = %d groups in %d calls, want %d in 2", len(groups), client.calls["SearchGroups"], selectedPageSize+1)
	}

	users, found, err := QualityGateSelectedUsers(context.Background(), client, cachetest.Disabled(), testGateName)
	if err != nil || !found {
		t.Fatalf("QualityGateSelectedUsers() = %v, %v, want found", found, err)
	}

	if user, selected := users["alice"]; !selected || user.Login != "alice" {
		t.Errorf("QualityGateSelectedUsers() = %+v, want alice", users)
	}

	for _, options := range client.options {
		switch opt := options.(type) {
		case sonar.QualitygatesSearchGroupsOptions:
			if opt.Query != "" || opt.Selected != sonar.SelectionFilterSelected || opt.GateName != testGateName {
				t.Errorf("SearchGroups options = %+v, want the selected groups of %q without query", opt, testGateName)
			}
		case sonar.QualitygatesSearchUsersOptions:
			if opt.Query != "" || opt.Selected != sonar.SelectionFilterSelected || opt.GateName != testGateName {
				t.Errorf("SearchUsers options = %+v, want the selected users of %q without query", opt, testGateName)
			}
		}
	}
}

// TestQualityGateSelectedIndexesMissingGate tests that a missing gate is
// reported as not found, without error, and is not cached.
func TestQualityGateSelectedIndexesMissingGate(t *testing.T) {
	t.Parallel()

	client := newStubQualityGatesClient("other-gate")
	scoped := cachetest.NewScoped(t)

	for range 2 {
		_, found, err := QualityGateSelectedGroups(context.Background(), client, scoped, testGateName)
		if err != nil || found {
			t.Errorf("QualityGateSelectedGroups() = %v, %v, want not found", found, err)
		}

		_, found, err = QualityGateSelectedUsers(context.Background(), client, scoped, testGateName)
		if err != nil || found {
			t.Errorf("QualityGateSelectedUsers() = %v, %v, want not found", found, err)
		}
	}

	if client.calls["SearchGroups"] != 2 || client.calls["SearchUsers"] != 2 {
		t.Errorf("searches called %v, want 2 of each: a missing gate must not be cached", client.calls)
	}

	client.readErr = errors.New("boom")

	_, _, err := QualityGateSelectedGroups(context.Background(), client, scoped, testGateName)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("QualityGateSelectedGroups() error = %v, want the API error", err)
	}
}

// TestQualityGateSelectedIndexesShared tests that K associations on one
// gate share one fetch per TTL, groups and users separately.
func TestQualityGateSelectedIndexesShared(t *testing.T) {
	t.Parallel()

	client := newStubQualityGatesClient(testGateName)
	client.groups[testGateName] = []string{"group-0", "group-1", "group-2"}
	client.users[testGateName] = []string{"alice", "bob"}
	scoped := cachetest.NewScoped(t)

	for range 5 {
		_, _, err := QualityGateSelectedGroups(context.Background(), client, scoped, testGateName)
		if err != nil {
			t.Fatalf("QualityGateSelectedGroups() error = %v", err)
		}

		_, _, err = QualityGateSelectedUsers(context.Background(), client, scoped, testGateName)
		if err != nil {
			t.Fatalf("QualityGateSelectedUsers() error = %v", err)
		}
	}

	if client.calls["SearchGroups"] != 1 || client.calls["SearchUsers"] != 1 {
		t.Errorf("searches called %v, want 1 of each", client.calls)
	}
}

// TestCachedQualityGateUsergroupAssociationClientWrites tests that the
// association writes invalidate the selections of their subject type, so
// that the next lookup sees them.
func TestCachedQualityGateUsergroupAssociationClientWrites(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner := newStubQualityGatesClient(testGateName)
	scoped := cachetest.NewScoped(t)
	client := NewCachedQualityGateUsergroupAssociationClient(inner, scoped)

	selected := func() (groupSelected, userSelected bool) {
		t.Helper()

		groups, _, err := QualityGateSelectedGroups(ctx, client, scoped, testGateName)
		if err != nil {
			t.Fatalf("QualityGateSelectedGroups() error = %v", err)
		}

		users, _, err := QualityGateSelectedUsers(ctx, client, scoped, testGateName)
		if err != nil {
			t.Fatalf("QualityGateSelectedUsers() error = %v", err)
		}

		_, groupSelected = groups[testGroupName]
		_, userSelected = users["alice"]

		return groupSelected, userSelected
	}

	steps := []struct {
		name      string
		write     func() (*http.Response, error)
		wantGroup bool
		wantUser  bool
	}{
		{"AddGroup", func() (*http.Response, error) {
			return client.AddGroup(ctx, GenerateQualityGateAddGroupOptions(testGateName, testGroupName))
		}, true, false},
		{"AddUser", func() (*http.Response, error) {
			return client.AddUser(ctx, GenerateQualityGateAddUserOptions(testGateName, "alice"))
		}, true, true},
		{"RemoveGroup", func() (*http.Response, error) {
			return client.RemoveGroup(ctx, GenerateQualityGateRemoveGroupOptions(testGateName, testGroupName))
		}, false, true},
		{"RemoveUser", func() (*http.Response, error) {
			return client.RemoveUser(ctx, GenerateQualityGateRemoveUserOptions(testGateName, "alice"))
		}, false, false},
	}

	selected()

	for _, step := range steps {
		resp, err := step.write() //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			t.Fatalf("%s() error = %v", step.name, err)
		}

		if groupSelected, userSelected := selected(); groupSelected != step.wantGroup || userSelected != step.wantUser {
			t.Errorf("after %s: group selected %v, user selected %v, want %v and %v", step.name, groupSelected, userSelected, step.wantGroup, step.wantUser)
		}
	}

	// Every write refetched only the selections of its subject type.
	if inner.calls["SearchGroups"] != 3 || inner.calls["SearchUsers"] != 3 {
		t.Errorf("searches called %v, want 3 of each", inner.calls)
	}
}

// TestQualityProfileSelectedIndexes tests that the Quality Profile
// selections are cached per (language, profile), read without name filter,
// and invalidated by the writes of their subject type.
func TestQualityProfileSelectedIndexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner := &stubQualityProfilesClient{calls: map[string]int{}}
	scoped := cachetest.NewScoped(t)
	client := NewCachedQualityProfileUsergroupAssociationClient(inner, scoped)

	read := func(language, profile string) {
		t.Helper()

		groups, found, err := QualityProfileSelectedGroups(ctx, client, scoped, language, profile)
		if _, selected := groups[testGroupName]; err != nil || !found || !selected {
			t.Fatalf("QualityProfileSelectedGroups(%q, %q) = %v, %v, %v", language, profile, groups, found, err)
		}

		users, found, err := QualityProfileSelectedUsers(ctx, client, scoped, language, profile)
		if _, selected := users["alice"]; err != nil || !found || !selected {
			t.Fatalf("QualityProfileSelectedUsers(%q, %q) = %v, %v, %v", language, profile, users, found, err)
		}
	}

	read("go", "Sonar way")
	read("go", "Sonar way")
	read("java", "Sonar way")

	if inner.calls["SearchGroups"] != 2 || inner.calls["SearchUsers"] != 2 {
		t.Errorf("searches called %v, want 2 of each (one per profile)", inner.calls)
	}

	for _, write := range []func() (*http.Response, error){
		func() (*http.Response, error) {
			return client.AddGroup(ctx, GenerateQualityProfileAddGroupOptions("go", "Sonar way", testGroupName))
		},
		func() (*http.Response, error) {
			return client.RemoveUser(ctx, GenerateQualityProfileRemoveUserOptions("go", "Sonar way", "alice"))
		},
		func() (*http.Response, error) {
			return client.AddUser(ctx, GenerateQualityProfileAddUserOptions("go", "Sonar way", "alice"))
		},
		func() (*http.Response, error) {
			return client.RemoveGroup(ctx, GenerateQualityProfileRemoveGroupOptions("go", "Sonar way", testGroupName))
		},
	} {
		resp, err := write() //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			t.Fatalf("write error = %v", err)
		}

		read("go", "Sonar way")
	}

	// Each write refetched the selections of its subject type, for every
	// profile: 2 group writes and 2 user writes.
	if inner.calls["SearchGroups"] != 4 || inner.calls["SearchUsers"] != 4 {
		t.Errorf("searches called %v, want 4 of each", inner.calls)
	}

	for _, options := range inner.options {
		switch opt := options.(type) {
		case sonar.QualityprofilesSearchGroupsOptions:
			if opt.Query != "" || opt.Selected != sonar.SelectionFilterSelected {
				t.Errorf("SearchGroups options = %+v, want the selected groups without query", opt)
			}
		case sonar.QualityprofilesSearchUsersOptions:
			if opt.Query != "" || opt.Selected != sonar.SelectionFilterSelected {
				t.Errorf("SearchUsers options = %+v, want the selected users without query", opt)
			}
		}
	}
}

// TestNewCachedUsergroupAssociationClientsDisabled tests that the raw
// clients are used when the cache is disabled.
func TestNewCachedUsergroupAssociationClientsDisabled(t *testing.T) {
	t.Parallel()

	gates := newStubQualityGatesClient(testGateName)
	if got := NewCachedQualityGateUsergroupAssociationClient(gates, cachetest.Disabled()); got != gates {
		t.Errorf("NewCachedQualityGateUsergroupAssociationClient() with a noop store = %T, want the raw client", got)
	}

	profiles := &stubQualityProfilesClient{calls: map[string]int{}}
	if got := NewCachedQualityProfileUsergroupAssociationClient(profiles, cachetest.Disabled()); got != profiles {
		t.Errorf("NewCachedQualityProfileUsergroupAssociationClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewQualityGateUsergroupAssociationClient(newTestConfig()).(*cachedQualityGateUsergroupAssociationClient); cached {
		t.Error("NewQualityGateUsergroupAssociationClient() returned a cached client while the cache is disabled")
	}

	if _, cached := NewQualityProfileUsergroupAssociationClient(newTestConfig()).(*cachedQualityProfileUsergroupAssociationClient); cached {
		t.Error("NewQualityProfileUsergroupAssociationClient() returned a cached client while the cache is disabled")
	}
}

// stubInstanceQualityGatesClient is an instance.QualityGatesClient counting
// deletions and renames. Other methods panic through the nil embedded
// interface.
type stubInstanceQualityGatesClient struct {
	instance.QualityGatesClient
}

// Delete succeeds.
func (*stubInstanceQualityGatesClient) Delete(context.Context, *sonar.QualitygatesDeleteOptions) (*http.Response, error) {
	return okResponse(), nil
}

// Rename succeeds.
func (*stubInstanceQualityGatesClient) Rename(context.Context, *sonar.QualitygatesRenameOptions) (*http.Response, error) {
	return okResponse(), nil
}

// stubInstanceQualityProfilesClient is an instance.QualityProfilesClient
// accepting deletions. Other methods panic through the nil embedded
// interface.
type stubInstanceQualityProfilesClient struct {
	instance.QualityProfilesClient
}

// Delete succeeds.
func (*stubInstanceQualityProfilesClient) Delete(context.Context, *sonar.QualityprofilesDeleteOptions) (*http.Response, error) {
	return okResponse(), nil
}

// TestEntityWritesInvalidateSelections tests that deleting or renaming a
// Quality Gate, deleting a Quality Profile, deleting a group or
// deactivating a user invalidates the selections mentioning them, whichever
// controller performs the write.
func TestEntityWritesInvalidateSelections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write func(scoped cache.Scoped) (*http.Response, error)
		// want are the searches after reading, writing and reading again.
		want map[string]int
	}{
		"QualityGateDelete": {
			write: func(scoped cache.Scoped) (*http.Response, error) {
				return instance.NewCachedQualityGatesClient(&stubInstanceQualityGatesClient{}, scoped).
					Delete(ctx, &sonar.QualitygatesDeleteOptions{Name: testGateName})
			},
			want: map[string]int{"gate/SearchGroups": 2, "gate/SearchUsers": 2, "profile/SearchGroups": 1, "profile/SearchUsers": 1},
		},
		"QualityGateRename": {
			write: func(scoped cache.Scoped) (*http.Response, error) {
				return instance.NewCachedQualityGatesClient(&stubInstanceQualityGatesClient{}, scoped).
					Rename(ctx, &sonar.QualitygatesRenameOptions{CurrentName: testGateName, Name: "renamed"})
			},
			want: map[string]int{"gate/SearchGroups": 2, "gate/SearchUsers": 2, "profile/SearchGroups": 1, "profile/SearchUsers": 1},
		},
		"QualityProfileDelete": {
			write: func(scoped cache.Scoped) (*http.Response, error) {
				return instance.NewCachedQualityProfilesClient(&stubInstanceQualityProfilesClient{}, scoped).
					Delete(ctx, &sonar.QualityprofilesDeleteOptions{Language: "go", QualityProfile: "Sonar way"})
			},
			want: map[string]int{"gate/SearchGroups": 1, "gate/SearchUsers": 1, "profile/SearchGroups": 2, "profile/SearchUsers": 2},
		},
		"GroupDelete": {
			write: func(scoped cache.Scoped) (*http.Response, error) {
				return NewCachedGroupsClient(&stubGroupsClient{calls: map[string]int{}}, scoped).DeleteGroup(ctx, "id")
			},
			want: map[string]int{"gate/SearchGroups": 2, "gate/SearchUsers": 1, "profile/SearchGroups": 2, "profile/SearchUsers": 1},
		},
		"UserDeactivate": {
			write: func(scoped cache.Scoped) (*http.Response, error) {
				return NewCachedUsersClient(&stubUsersClient{calls: map[string]int{}}, scoped).
					Deactivate(ctx, &sonar.UsersDeactivateOptionsV2{Id: "id"})
			},
			want: map[string]int{"gate/SearchGroups": 1, "gate/SearchUsers": 2, "profile/SearchGroups": 1, "profile/SearchUsers": 2},
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gates := newStubQualityGatesClient(testGateName)
			profiles := &stubQualityProfilesClient{calls: map[string]int{}}
			scoped := cachetest.NewScoped(t)

			readAllSelections(t, gates, profiles, scoped)

			resp, err := tc.write(scoped) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}

			readAllSelections(t, gates, profiles, scoped)

			got := map[string]int{
				"gate/SearchGroups":    gates.calls["SearchGroups"],
				"gate/SearchUsers":     gates.calls["SearchUsers"],
				"profile/SearchGroups": profiles.calls["SearchGroups"],
				"profile/SearchUsers":  profiles.calls["SearchUsers"],
			}
			for search, want := range tc.want {
				if got[search] != want {
					t.Errorf("%s called %d times, want %d", search, got[search], want)
				}
			}
		})
	}
}

// readAllSelections reads the selected groups and users of the test gate
// and of a Quality Profile.
func readAllSelections(t *testing.T, gates QualityGateUsergroupAssociationClient, profiles QualityProfileUsergroupAssociationClient, scoped cache.Scoped) {
	t.Helper()

	ctx := context.Background()

	_, _, groupsErr := QualityGateSelectedGroups(ctx, gates, scoped, testGateName)
	_, _, usersErr := QualityGateSelectedUsers(ctx, gates, scoped, testGateName)
	_, _, profileGroupsErr := QualityProfileSelectedGroups(ctx, profiles, scoped, "go", "Sonar way")
	_, _, profileUsersErr := QualityProfileSelectedUsers(ctx, profiles, scoped, "go", "Sonar way")

	err := errors.Join(groupsErr, usersErr, profileGroupsErr, profileUsersErr)
	if err != nil {
		t.Fatalf("reading selections: %v", err)
	}
}
