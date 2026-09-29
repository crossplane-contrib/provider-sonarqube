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
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// stubTemplatesClient is a PermissionsTemplatesClient serving templates and
// their members from memory, and counting the calls of every method.
type stubTemplatesClient struct {
	// templates are served by SearchTemplates.
	templates []sonar.PermissionTemplate
	// defaultID is the ID of the default template.
	defaultID string
	// groups are served by TemplateGroups, whatever the template.
	groups []sonar.PermissionsTemplateGroup
	// users are served by TemplateUsers, whatever the template.
	users []sonar.PermissionsTemplateUser
	// ignorePages makes SearchTemplates return every template whatever the
	// requested page, like current SonarQube versions.
	ignorePages bool
	// readErr is returned by the reads when set.
	readErr error
	// calls counts the calls of every method.
	calls map[string]int
}

// newStubTemplatesClient returns a stubTemplatesClient serving templates.
func newStubTemplatesClient(templates ...sonar.PermissionTemplate) *stubTemplatesClient {
	return &stubTemplatesClient{templates: templates, calls: map[string]int{}}
}

// AddGroupToTemplate counts the call.
func (s *stubTemplatesClient) AddGroupToTemplate(context.Context, *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error) {
	return s.write("AddGroupToTemplate")
}

// AddProjectCreatorToTemplate counts the call.
func (s *stubTemplatesClient) AddProjectCreatorToTemplate(context.Context, *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error) {
	return s.write("AddProjectCreatorToTemplate")
}

// AddUserToTemplate counts the call.
func (s *stubTemplatesClient) AddUserToTemplate(context.Context, *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error) {
	return s.write("AddUserToTemplate")
}

// CreateTemplate counts the call and adds the template.
func (s *stubTemplatesClient) CreateTemplate(_ context.Context, opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
	s.calls["CreateTemplate"]++

	template := sonar.PermissionTemplate{ID: "id-" + opt.Name, Name: opt.Name}
	s.templates = append(s.templates, template)

	return &sonar.PermissionsCreateTemplate{PermissionTemplate: sonar.PermissionsTemplateBasic{ID: template.ID, Name: template.Name}}, okResponse(), nil
}

// DeleteTemplate counts the call.
func (s *stubTemplatesClient) DeleteTemplate(context.Context, *sonar.PermissionsDeleteTemplateOptions) (*http.Response, error) {
	return s.write("DeleteTemplate")
}

// RemoveGroupFromTemplate counts the call.
func (s *stubTemplatesClient) RemoveGroupFromTemplate(context.Context, *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error) {
	return s.write("RemoveGroupFromTemplate")
}

// RemoveProjectCreatorFromTemplate counts the call.
func (s *stubTemplatesClient) RemoveProjectCreatorFromTemplate(context.Context, *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error) {
	return s.write("RemoveProjectCreatorFromTemplate")
}

// RemoveUserFromTemplate counts the call.
func (s *stubTemplatesClient) RemoveUserFromTemplate(context.Context, *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error) {
	return s.write("RemoveUserFromTemplate")
}

// SearchTemplates counts the call and returns the requested page of
// templates, or every template when ignorePages is set.
func (s *stubTemplatesClient) SearchTemplates(_ context.Context, opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
	s.calls["SearchTemplates"]++

	if s.readErr != nil {
		return nil, nil, s.readErr
	}

	templates := s.templates
	if !s.ignorePages {
		templates, _ = page(s.templates, opt.PaginationArgs)
	}

	result := &sonar.PermissionsSearchTemplates{PermissionTemplates: templates}
	if s.defaultID != "" {
		result.DefaultTemplates = []sonar.PermissionsDefaultTemplate{{Qualifier: "TRK", TemplateID: s.defaultID}}
	}

	return result, okResponse(), nil
}

// SetDefaultTemplate counts the call and makes the template the default.
func (s *stubTemplatesClient) SetDefaultTemplate(_ context.Context, opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error) {
	s.defaultID = opt.TemplateID

	return s.write("SetDefaultTemplate")
}

// TemplateGroups counts the call and returns the requested page of groups.
func (s *stubTemplatesClient) TemplateGroups(_ context.Context, opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
	s.calls["TemplateGroups"]++

	if s.readErr != nil {
		return nil, nil, s.readErr
	}

	groups, paging := page(s.groups, opt.PaginationArgs)

	return &sonar.PermissionsTemplateGroups{Groups: groups, Paging: paging}, okResponse(), nil
}

// TemplateUsers counts the call and returns the requested page of users.
func (s *stubTemplatesClient) TemplateUsers(_ context.Context, opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
	s.calls["TemplateUsers"]++

	if s.readErr != nil {
		return nil, nil, s.readErr
	}

	users, paging := page(s.users, opt.PaginationArgs)

	return &sonar.PermissionsTemplateUsers{Users: users, Paging: paging}, okResponse(), nil
}

// UpdateTemplate counts the call.
func (s *stubTemplatesClient) UpdateTemplate(context.Context, *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error) {
	s.calls["UpdateTemplate"]++

	return &sonar.PermissionsUpdateTemplate{}, okResponse(), nil
}

// write counts a call to method.
func (s *stubTemplatesClient) write(method string) (*http.Response, error) {
	s.calls[method]++

	return okResponse(), nil
}

// manyTemplates returns count templates named template-<index>.
func manyTemplates(count int) []sonar.PermissionTemplate {
	templates := make([]sonar.PermissionTemplate, 0, count)
	for index := range count {
		templates = append(templates, sonar.PermissionTemplate{ID: fmt.Sprintf("id-%d", index), Name: fmt.Sprintf("template-%d", index)})
	}

	return templates
}

// TestPermissionTemplatesIndex tests building the index of permission
// templates.
func TestPermissionTemplatesIndex(t *testing.T) {
	t.Parallel()

	t.Run("MultiPageWithDefault", func(t *testing.T) {
		t.Parallel()

		client := newStubTemplatesClient(manyTemplates(int(permissionTemplatesPageSize) + 1)...)
		client.defaultID = "id-500"

		index, err := PermissionTemplatesIndex(context.Background(), client, cachetest.Disabled())
		if err != nil {
			t.Fatalf("PermissionTemplatesIndex() error = %v", err)
		}

		if got := client.calls["SearchTemplates"]; got != 2 {
			t.Errorf("SearchTemplates called %d times, want 2", got)
		}

		template, isDefault, found := index.FindByID("id-500")
		if !found || !isDefault || template.Name != "template-500" {
			t.Errorf("FindByID(id-500) = %+v, %v, %v, want the default template-500", template, isDefault, found)
		}

		template, isDefault, found = index.FindByName("template-3")
		if !found || isDefault || template.ID != "id-3" {
			t.Errorf("FindByName(template-3) = %+v, %v, %v, want the non-default id-3", template, isDefault, found)
		}

		if _, _, found = index.FindByID("absent"); found {
			t.Error("FindByID(absent) found a template")
		}

		if _, _, found = index.FindByName("absent"); found {
			t.Error("FindByName(absent) found a template")
		}
	})

	// A server returning every template whatever the page must not make
	// the scan loop forever.
	t.Run("ServerIgnoresPages", func(t *testing.T) {
		t.Parallel()

		client := newStubTemplatesClient(manyTemplates(int(permissionTemplatesPageSize) + 10)...)
		client.ignorePages = true

		index, err := PermissionTemplatesIndex(context.Background(), client, cachetest.Disabled())
		if err != nil {
			t.Fatalf("PermissionTemplatesIndex() error = %v", err)
		}

		if got := client.calls["SearchTemplates"]; got != 2 {
			t.Errorf("SearchTemplates called %d times, want 2", got)
		}

		if got := len(index.Templates); got != int(permissionTemplatesPageSize)+10 {
			t.Errorf("index holds %d templates, want %d", got, permissionTemplatesPageSize+10)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		t.Parallel()

		client := newStubTemplatesClient()
		client.readErr = errors.New("boom")

		_, err := PermissionTemplatesIndex(context.Background(), client, cachetest.Disabled())
		if err == nil || !strings.Contains(err.Error(), "cannot search permission templates") {
			t.Errorf("PermissionTemplatesIndex() error = %v, want a search error", err)
		}
	})
}

// TestPermissionTemplateMembers tests reading the paginated groups and
// users of a template.
func TestPermissionTemplateMembers(t *testing.T) {
	t.Parallel()

	client := newStubTemplatesClient()
	for index := range int(permissionTemplateMembersPageSize) + 1 {
		client.groups = append(client.groups, sonar.PermissionsTemplateGroup{Name: fmt.Sprintf("group-%d", index)})
	}

	client.users = []sonar.PermissionsTemplateUser{{Login: "alice", Permissions: []string{"scan"}}}

	groups, err := PermissionTemplateGroups(context.Background(), client, cachetest.Disabled(), "id")
	if err != nil {
		t.Fatalf("PermissionTemplateGroups() error = %v", err)
	}

	if len(groups) != len(client.groups) || client.calls["TemplateGroups"] != 2 {
		t.Errorf("PermissionTemplateGroups() = %d groups in %d calls, want %d in 2", len(groups), client.calls["TemplateGroups"], len(client.groups))
	}

	users, err := PermissionTemplateUsers(context.Background(), client, cachetest.Disabled(), "id")
	if err != nil {
		t.Fatalf("PermissionTemplateUsers() error = %v", err)
	}

	if diff := cmp.Diff(client.users, users); diff != "" {
		t.Errorf("PermissionTemplateUsers() mismatch (-want +got):\n%s", diff)
	}

	client.readErr = errors.New("boom")

	_, err = PermissionTemplateGroups(context.Background(), client, cachetest.Disabled(), "id")
	if err == nil {
		t.Error("PermissionTemplateGroups() error = nil, want the API error")
	}

	_, err = PermissionTemplateUsers(context.Background(), client, cachetest.Disabled(), "id")
	if err == nil {
		t.Error("PermissionTemplateUsers() error = nil, want the API error")
	}
}

// TestPermissionTemplatesCachedAcrossResources tests that K template
// lookups share one scan per TTL, and that members are cached per
// template.
func TestPermissionTemplatesCachedAcrossResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newStubTemplatesClient(manyTemplates(5)...)
	scoped := cachetest.NewScoped(t)

	for _, id := range []string{"id-0", "id-1", "id-2", "id-3", "id-4"} {
		index, err := PermissionTemplatesIndex(ctx, client, scoped)
		if err != nil {
			t.Fatalf("PermissionTemplatesIndex() error = %v", err)
		}

		if _, _, found := index.FindByID(id); !found {
			t.Errorf("FindByID(%q) found nothing", id)
		}

		for range 2 {
			_, err = PermissionTemplateGroups(ctx, client, scoped, id)
			if err != nil {
				t.Fatalf("PermissionTemplateGroups() error = %v", err)
			}

			_, err = PermissionTemplateUsers(ctx, client, scoped, id)
			if err != nil {
				t.Fatalf("PermissionTemplateUsers() error = %v", err)
			}
		}
	}

	want := map[string]int{"SearchTemplates": 1, "TemplateGroups": 5, "TemplateUsers": 5}
	if diff := cmp.Diff(want, client.calls); diff != "" {
		t.Errorf("calls mismatch (-want +got):\n%s", diff)
	}
}

// TestNewCachedPermissionsTemplatesClientDisabled tests that the raw client
// is used when the cache is disabled.
func TestNewCachedPermissionsTemplatesClientDisabled(t *testing.T) {
	t.Parallel()

	inner := newStubTemplatesClient()
	if got := NewCachedPermissionsTemplatesClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("NewCachedPermissionsTemplatesClient() with a noop store = %T, want the raw client", got)
	}

	if _, cached := NewPermissionsTemplatesClient(newTestConfig()).(*cachedPermissionsTemplatesClient); cached {
		t.Error("NewPermissionsTemplatesClient() returned a cached client while the cache is disabled")
	}
}

// TestCachedPermissionsTemplatesClientWritesInvalidate tests which cached
// template datasets every write invalidates.
func TestCachedPermissionsTemplatesClientWritesInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write func(PermissionsTemplatesClient) error
		// want are the reads reaching SonarQube after reading everything,
		// writing, and reading everything again.
		want map[string]int
	}{
		"CreateTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				_, resp, err := c.CreateTemplate(ctx, &sonar.PermissionsCreateTemplateOptions{Name: "new"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 1},
		},
		"UpdateTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				_, resp, err := c.UpdateTemplate(ctx, &sonar.PermissionsUpdateTemplateOptions{ID: "id-0"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 1},
		},
		"SetDefaultTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.SetDefaultTemplate(ctx, GeneratePermissionsTemplateSetAsDefaultOptions("id-0")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 1},
		},
		"DeleteTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.DeleteTemplate(ctx, GeneratePermissionsTemplateDeleteOptions("id-0")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 2},
		},
		"AddGroupToTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.AddGroupToTemplate(ctx, GeneratePermissionsTemplateAddGroupPermissionOptions("id-0", "devs", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 1},
		},
		"RemoveGroupFromTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.RemoveGroupFromTemplate(ctx, GeneratePermissionsTemplateRemoveGroupPermissionOptions("id-0", "devs", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 1},
		},
		"AddUserToTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.AddUserToTemplate(ctx, GeneratePermissionsTemplateAddUserPermissionOptions("id-0", "alice", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 2},
		},
		"RemoveUserFromTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.RemoveUserFromTemplate(ctx, GeneratePermissionsTemplateRemoveUserPermissionOptions("id-0", "alice", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 2},
		},
		"AddProjectCreatorToTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.AddProjectCreatorToTemplate(ctx, GeneratePermissionsTemplateAddCreatorPermissionOptions("id-0", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 2},
		},
		"RemoveProjectCreatorFromTemplate": {
			write: func(c PermissionsTemplatesClient) error {
				resp, err := c.RemoveProjectCreatorFromTemplate(ctx, GeneratePermissionsTemplateRemoveCreatorPermissionOptions("id-0", "scan")) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 2},
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inner := newStubTemplatesClient(manyTemplates(1)...)
			scoped := cachetest.NewScoped(t)
			client := NewCachedPermissionsTemplatesClient(inner, scoped)

			readAllTemplateData(t, client, scoped)

			err := tc.write(client)
			if err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}

			if got := inner.calls[name]; got != 1 {
				t.Errorf("inner %s called %d times, want 1", name, got)
			}

			readAllTemplateData(t, client, scoped)

			for method, want := range tc.want {
				if got := inner.calls[method]; got != want {
					t.Errorf("%s called %d times, want %d", method, got, want)
				}
			}
		})
	}
}

// TestCachedPermissionsTemplatesClientObservesOwnWrites tests that the
// next lookup after Create finds the new template, and that the default
// flag flips after SetDefaultTemplate.
func TestCachedPermissionsTemplatesClientObservesOwnWrites(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner := newStubTemplatesClient(manyTemplates(1)...)
	inner.defaultID = "id-0"
	scoped := cachetest.NewScoped(t)
	client := NewCachedPermissionsTemplatesClient(inner, scoped)

	index, err := PermissionTemplatesIndex(ctx, client, scoped)
	if err != nil {
		t.Fatalf("PermissionTemplatesIndex() error = %v", err)
	}

	if _, _, found := index.FindByName("new"); found {
		t.Fatal("FindByName(new) found a template before Create")
	}

	created, resp, err := client.CreateTemplate(ctx, &sonar.PermissionsCreateTemplateOptions{Name: "new"}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("CreateTemplate() error = %v", err)
	}

	index, err = PermissionTemplatesIndex(ctx, client, scoped)
	if err != nil {
		t.Fatalf("PermissionTemplatesIndex() error = %v", err)
	}

	if _, isDefault, found := index.FindByID(created.PermissionTemplate.ID); !found || isDefault {
		t.Fatalf("FindByID() after Create = default %v, found %v, want a non-default template", isDefault, found)
	}

	resp, err = client.SetDefaultTemplate(ctx, GeneratePermissionsTemplateSetAsDefaultOptions(created.PermissionTemplate.ID)) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("SetDefaultTemplate() error = %v", err)
	}

	index, err = PermissionTemplatesIndex(ctx, client, scoped)
	if err != nil {
		t.Fatalf("PermissionTemplatesIndex() error = %v", err)
	}

	if _, isDefault, _ := index.FindByID(created.PermissionTemplate.ID); !isDefault {
		t.Error("FindByID() after SetDefaultTemplate is not the default")
	}
}

// TestGeneratePermissionsTemplateObservationsCopyPermissions tests that the
// observations never share the permissions of a (cached) response.
func TestGeneratePermissionsTemplateObservationsCopyPermissions(t *testing.T) {
	t.Parallel()

	groups := []sonar.PermissionsTemplateGroup{{Name: "devs", Permissions: []string{"scan"}}}
	users := []sonar.PermissionsTemplateUser{{Login: "alice", Permissions: []string{"scan"}}}

	GeneratePermissionsTemplateGroupObservations(&groups)[0].Permissions[0] = "mutated"
	GeneratePermissionsTemplateUserObservations(&users)[0].Permissions[0] = "mutated"

	if groups[0].Permissions[0] != "scan" || users[0].Permissions[0] != "scan" {
		t.Errorf("mutating an observation changed the response: %+v, %+v", groups, users)
	}
}

// readAllTemplateData reads the template index and the members of template
// id-0 through client.
func readAllTemplateData(t *testing.T, client PermissionsTemplatesClient, scoped cache.Scoped) {
	t.Helper()

	ctx := context.Background()

	_, err := PermissionTemplatesIndex(ctx, client, scoped)
	if err != nil {
		t.Fatalf("PermissionTemplatesIndex() error = %v", err)
	}

	_, err = PermissionTemplateGroups(ctx, client, scoped, "id-0")
	if err != nil {
		t.Fatalf("PermissionTemplateGroups() error = %v", err)
	}

	_, err = PermissionTemplateUsers(ctx, client, scoped, "id-0")
	if err != nil {
		t.Fatalf("PermissionTemplateUsers() error = %v", err)
	}
}

// TestEntityWritesInvalidateTemplates tests that deleting a group or
// deactivating a user invalidates the template datasets mentioning them.
func TestEntityWritesInvalidateTemplates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writes := map[string]struct {
		write func(groups GroupsClient, users UsersClient) error
		want  map[string]int
	}{
		"DeleteGroup": {
			write: func(groups GroupsClient, _ UsersClient) error {
				resp, err := groups.DeleteGroup(ctx, "id") //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 2, "TemplateUsers": 1},
		},
		"DeactivateUser": {
			write: func(_ GroupsClient, users UsersClient) error {
				resp, err := users.Deactivate(ctx, &sonar.UsersDeactivateOptionsV2{Id: "id"}) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: map[string]int{"SearchTemplates": 2, "TemplateGroups": 1, "TemplateUsers": 2},
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			templates := newStubTemplatesClient(manyTemplates(1)...)
			scoped := cachetest.NewScoped(t)
			groups := NewCachedGroupsClient(&stubGroupsClient{calls: map[string]int{}}, scoped)
			users := NewCachedUsersClient(&stubUsersClient{calls: map[string]int{}}, scoped)

			readAllTemplateData(t, templates, scoped)

			err := tc.write(groups, users)
			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}

			readAllTemplateData(t, templates, scoped)

			for method, want := range tc.want {
				if got := templates.calls[method]; got != want {
					t.Errorf("%s called %d times, want %d", method, got, want)
				}
			}
		})
	}
}
