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
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestObserve tests observing Permissions resource state.
//
//nolint:maintidx // Comprehensive observe-path coverage requires a larger test body.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		client *fake.MockPermissionsClient
		args   args
		want   want
	}{
		"NotPermissionsType": {
			client: &fake.MockPermissionsClient{},
			args:   args{ctx: context.Background(), mg: &notPermissions{}},
			want:   want{errSubstr: errNotPermissions},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("", "devs", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"InvalidExternalNameNoColon": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("no-colon", "devs", []string{"scan"}),
			},
			// Crossplane defaults the external name to metadata.name before
			// Create runs; an unparseable name must not error - it means the
			// resource does not exist yet.
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"InvalidExternalNameUnknownType": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("robot:r2d2", "devs", []string{"scan"}),
			},
			// Same rationale: an unrecognised subject type triggers Create.
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"GroupNotFoundReturnsNotExists": {
			client: &fake.MockPermissionsClient{
				GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "other", Permissions: []string{"scan"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("group:devs", "devs", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"GroupAPIErrorWrapped": {
			client: &fake.MockPermissionsClient{
				GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("group:devs", "devs", []string{"scan"}),
			},
			want: want{errSubstr: errObservePermissions},
		},
		"UserNotFoundReturnsNotExists": {
			client: &fake.MockPermissionsClient{
				UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
					return &sonar.PermissionsUsers{
						Users:  []sonar.PermissionUser{{Login: "bob", Permissions: []string{"scan"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserPermissions("user:alice", "alice", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"UserAPIErrorWrapped": {
			client: &fake.MockPermissionsClient{
				UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
					return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserPermissions("user:alice", "alice", []string{"scan"}),
			},
			want: want{errSubstr: errObservePermissions},
		},
		"GroupUpToDate": {
			client: &fake.MockPermissionsClient{
				GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"scan"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("group:devs", "devs", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true,
				ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"GroupNotUpToDate": {
			client: &fake.MockPermissionsClient{
				GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"admin"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("group:devs", "devs", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: false,
				ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"UserUpToDate": {
			client: &fake.MockPermissionsClient{
				UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
					return &sonar.PermissionsUsers{
						Users:  []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserPermissions("user:alice", "alice", []string{"scan"}),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true,
				ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
		"GroupWithProjectKey": {
			client: &fake.MockPermissionsClient{
				GroupsFn: func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
					if opt.ProjectKey != "proj-1" {
						return nil, nil, errors.New("unexpected project key")
					}

					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"user"}}},
						Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissionsWithProject("group:devs:proj-1", "devs", "proj-1", []string{"user"}),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists: true, ResourceUpToDate: true,
				ResourceLateInitialized: false, ConnectionDetails: managed.ConnectionDetails{},
			}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			checkError(t, "Observe", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("ObserveSetsAtProvider", func(t *testing.T) {
		t.Parallel()

		p := newTestGroupPermissions("group:devs", "devs", []string{"scan", "admin"})
		e := &external{client: &fake.MockPermissionsClient{
			GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"scan", "admin"}}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Observe(context.Background(), p)
		if err != nil {
			t.Fatalf("Observe() unexpected error: %v", err)
		}

		if diff := cmp.Diff(sorted([]string{"scan", "admin"}), sorted(p.Status.AtProvider.Permissions)); diff != "" {
			t.Fatalf("Observe() AtProvider.Permissions mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("ObservePaginatesUntilGroupFound", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		p := newTestGroupPermissions("group:devs", "devs", []string{"provisioning"})
		e := &external{client: &fake.MockPermissionsClient{
			GroupsFn: func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				callCount++

				if opt.Page == 1 {
					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "other", Permissions: []string{"admin"}}},
						Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				}

				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"provisioning"}}},
					Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 2, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			},
		}}

		obs, err := e.Observe(context.Background(), p)
		if err != nil {
			t.Fatalf("Observe() unexpected error: %v", err)
		}

		if !obs.ResourceExists {
			t.Fatal("Observe() expected ResourceExists=true after pagination")
		}

		if callCount != 2 {
			t.Fatalf("Observe() expected 2 API calls for pagination, got %d", callCount)
		}
	})
}

// TestGetObservedGroupPermissions tests the group permissions lookup helper.
func TestGetObservedGroupPermissions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client          iam.PermissionsClient
		groupName       string
		projectKey      string
		wantPermissions []string
		wantFound       bool
		wantErrSubstr   string
	}{
		"GroupFoundWithPermissions": {
			client: &fake.MockPermissionsClient{GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"scan", "admin"}}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			groupName:       "devs",
			wantPermissions: []string{"scan", "admin"},
			wantFound:       true,
		},
		"GroupNotFound": {
			client: &fake.MockPermissionsClient{GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "other"}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			groupName: "devs",
			wantFound: false,
		},
		"APIError": {
			client: &fake.MockPermissionsClient{GroupsFn: func(_ *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
			}},
			groupName:     "devs",
			wantErrSubstr: "cannot list group permissions",
		},
		"PaginationFindsOnSecondPage": {
			client: &fake.MockPermissionsClient{GroupsFn: func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				if opt.Page == 1 {
					return &sonar.PermissionsGroups{
						Groups: []sonar.PermissionGroup{{Name: "admins", Permissions: []string{"admin"}}},
						Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				}

				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"scan"}}},
					Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 2, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			groupName:       "devs",
			wantPermissions: []string{"scan"},
			wantFound:       true,
		},
		"ProjectKeyPassedToAPI": {
			client: &fake.MockPermissionsClient{GroupsFn: func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				if opt.ProjectKey != "proj-1" {
					return nil, nil, errors.New("unexpected project key")
				}

				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{{Name: "devs", Permissions: []string{"user"}}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			groupName:       "devs",
			projectKey:      "proj-1",
			wantPermissions: []string{"user"},
			wantFound:       true,
		},
		// SonarQube ignores the q (name) filter when projectKey is set and
		// returns empty permissions; the helper must omit q in that case.
		"ProjectKeyOmitsQueryFilter": {
			client: &fake.MockPermissionsClient{GroupsFn: func(opt *sonar.PermissionsGroupsOptions) (*sonar.PermissionsGroups, *http.Response, error) {
				if opt.Query != "" {
					return nil, nil, errors.New("query filter must not be sent with projectKey")
				}

				return &sonar.PermissionsGroups{
					Groups: []sonar.PermissionGroup{
						{Name: "other", Permissions: []string{"admin"}},
						{Name: "devs", Permissions: []string{"codeviewer", "user"}},
					},
					Paging: sonar.PermissionsPaging{Total: 2, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			groupName:       "devs",
			projectKey:      "proj-1",
			wantPermissions: []string{"codeviewer", "user"},
			wantFound:       true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotPerms, gotFound, err := getObservedGroupPermissions(tc.client, tc.groupName, tc.projectKey)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("getObservedGroupPermissions() expected error containing %q, got nil", tc.wantErrSubstr)
				}

				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("getObservedGroupPermissions() error = %q, want containing %q", err.Error(), tc.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("getObservedGroupPermissions() unexpected error: %v", err)
			}

			if gotFound != tc.wantFound {
				t.Fatalf("getObservedGroupPermissions() found = %v, want %v", gotFound, tc.wantFound)
			}

			if diff := cmp.Diff(sorted(tc.wantPermissions), sorted(gotPerms)); diff != "" {
				t.Fatalf("getObservedGroupPermissions() permissions mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGetObservedUserPermissions tests the user permissions lookup helper.
func TestGetObservedUserPermissions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client          iam.PermissionsClient
		login           string
		projectKey      string
		wantPermissions []string
		wantFound       bool
		wantErrSubstr   string
	}{
		"UserFoundWithPermissions": {
			client: &fake.MockPermissionsClient{UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
				return &sonar.PermissionsUsers{
					Users:  []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan", "admin"}}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			login:           "alice",
			wantPermissions: []string{"scan", "admin"},
			wantFound:       true,
		},
		"UserNotFound": {
			client: &fake.MockPermissionsClient{UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
				return &sonar.PermissionsUsers{
					Users:  []sonar.PermissionUser{{Login: "bob"}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			login:     "alice",
			wantFound: false,
		},
		"APIError": {
			client: &fake.MockPermissionsClient{UsersFn: func(_ *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
				return nil, mockHTTPResponse(http.StatusInternalServerError), errors.New("boom")
			}},
			login:         "alice",
			wantErrSubstr: "cannot list user permissions",
		},
		"PaginationFindsOnSecondPage": {
			client: &fake.MockPermissionsClient{UsersFn: func(opt *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
				if opt.Page == 1 {
					return &sonar.PermissionsUsers{
						Users:  []sonar.PermissionUser{{Login: "bob", Permissions: []string{"admin"}}},
						Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(http.StatusOK), nil
				}

				return &sonar.PermissionsUsers{
					Users:  []sonar.PermissionUser{{Login: "alice", Permissions: []string{"scan"}}},
					Paging: sonar.PermissionsPaging{Total: 150, PageIndex: 2, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			login:           "alice",
			wantPermissions: []string{"scan"},
			wantFound:       true,
		},
		"ProjectKeyPassedToAPI": {
			client: &fake.MockPermissionsClient{UsersFn: func(opt *sonar.PermissionsUsersOptions) (*sonar.PermissionsUsers, *http.Response, error) {
				if opt.ProjectKey != "proj-1" {
					return nil, nil, errors.New("unexpected project key")
				}

				return &sonar.PermissionsUsers{
					Users:  []sonar.PermissionUser{{Login: "alice", Permissions: []string{"codeviewer"}}},
					Paging: sonar.PermissionsPaging{Total: 1, PageIndex: 1, PageSize: 100},
				}, mockHTTPResponse(http.StatusOK), nil
			}},
			login:           "alice",
			projectKey:      "proj-1",
			wantPermissions: []string{"codeviewer"},
			wantFound:       true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotPerms, gotFound, err := getObservedUserPermissions(tc.client, tc.login, tc.projectKey)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("getObservedUserPermissions() expected error containing %q, got nil", tc.wantErrSubstr)
				}

				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("getObservedUserPermissions() error = %q, want containing %q", err.Error(), tc.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("getObservedUserPermissions() unexpected error: %v", err)
			}

			if gotFound != tc.wantFound {
				t.Fatalf("getObservedUserPermissions() found = %v, want %v", gotFound, tc.wantFound)
			}

			if diff := cmp.Diff(sorted(tc.wantPermissions), sorted(gotPerms)); diff != "" {
				t.Fatalf("getObservedUserPermissions() permissions mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestComputePermissionsDiff tests computing permission deltas.
func TestComputePermissionsDiff(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		desired    []string
		observed   []string
		wantAdd    []string
		wantRemove []string
	}{
		"IdenticalReturnsNoChanges": {
			desired:    []string{"admin", "scan"},
			observed:   []string{"scan", "admin"},
			wantAdd:    nil,
			wantRemove: nil,
		},
		"AddsMissingPermissions": {
			desired:    []string{"admin", "scan", "provisioning"},
			observed:   []string{"admin", "scan"},
			wantAdd:    []string{"provisioning"},
			wantRemove: nil,
		},
		"RemovesExtraPermissions": {
			desired:    []string{"admin"},
			observed:   []string{"admin", "scan"},
			wantAdd:    nil,
			wantRemove: []string{"scan"},
		},
		"AddsAndRemoves": {
			desired:    []string{"admin", "provisioning"},
			observed:   []string{"admin", "scan"},
			wantAdd:    []string{"provisioning"},
			wantRemove: []string{"scan"},
		},
		"EmptyDesiredRemovesAll": {
			desired:    []string{},
			observed:   []string{"admin", "scan"},
			wantAdd:    nil,
			wantRemove: []string{"admin", "scan"},
		},
		"EmptyObservedAddsAll": {
			desired:    []string{"admin", "scan"},
			observed:   []string{},
			wantAdd:    []string{"admin", "scan"},
			wantRemove: nil,
		},
		"BothEmptyNoChanges": {
			desired:    []string{},
			observed:   []string{},
			wantAdd:    nil,
			wantRemove: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotAdd, gotRemove := computePermissionsDiff(tc.desired, tc.observed)

			if diff := cmp.Diff(sorted(tc.wantAdd), sorted(gotAdd)); diff != "" {
				t.Fatalf("computePermissionsDiff() add mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(sorted(tc.wantRemove), sorted(gotRemove)); diff != "" {
				t.Fatalf("computePermissionsDiff() remove mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
