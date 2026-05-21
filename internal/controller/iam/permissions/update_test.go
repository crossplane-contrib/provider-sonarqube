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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestUpdate tests updating a Permissions resource.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		update    managed.ExternalUpdate
		errSubstr string
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
		"EmptyExternalName": {
			client: &fake.MockPermissionsClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupPermissions("", "devs", []string{"scan"}),
			},
			want: want{errSubstr: "external name is not set"},
		},
		"NoDiff": {
			client: &fake.MockPermissionsClient{},
			args: func() args {
				p := newTestGroupPermissions("group:devs", "devs", []string{"scan"})
				p.Status.AtProvider.Permissions = []string{"scan"}

				return args{ctx: context.Background(), mg: p}
			}(),
			want: want{update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}},
		},
		"AddFails": {
			client: &fake.MockPermissionsClient{
				AddGroupFn: func(_ *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("add error")
				},
			},
			args: func() args {
				p := newTestGroupPermissions("group:devs", "devs", []string{"scan", "admin"})
				p.Status.AtProvider.Permissions = []string{"scan"}

				return args{ctx: context.Background(), mg: p}
			}(),
			want: want{errSubstr: errUpdatePermissions},
		},
		"RemoveFails": {
			client: &fake.MockPermissionsClient{
				RemoveGroupFn: func(_ *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
					//nolint:nilnil // Intentional: simulating partial HTTP failure.
					return mockHTTPResponse(http.StatusInternalServerError), errors.New("remove error")
				},
			},
			args: func() args {
				p := newTestGroupPermissions("group:devs", "devs", []string{"scan"})
				p.Status.AtProvider.Permissions = []string{"scan", "admin"}

				return args{ctx: context.Background(), mg: p}
			}(),
			want: want{errSubstr: errUpdatePermissions},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("UpdateAddsAndRemovesCorrectPermissions", func(t *testing.T) {
		t.Parallel()

		added := make([]string, 0)
		removed := make([]string, 0)

		p := newTestGroupPermissions("group:devs", "devs", []string{"scan", "provisioning"})
		p.Status.AtProvider.Permissions = []string{"scan", "admin"}

		e := &external{client: &fake.MockPermissionsClient{
			AddGroupFn: func(opt *sonar.PermissionsAddGroupOptions) (*http.Response, error) {
				added = append(added, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
			RemoveGroupFn: func(opt *sonar.PermissionsRemoveGroupOptions) (*http.Response, error) {
				removed = append(removed, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Update(context.Background(), p)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"provisioning"}, sorted(added)); diff != "" {
			t.Fatalf("Update() permissions added mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]string{"admin"}, sorted(removed)); diff != "" {
			t.Fatalf("Update() permissions removed mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UpdateUserAddsAndRemoves", func(t *testing.T) {
		t.Parallel()

		added := make([]string, 0)
		removed := make([]string, 0)

		p := newTestUserPermissions("user:alice", "alice", []string{"scan", "provisioning"})
		p.Status.AtProvider.Permissions = []string{"scan", "admin"}

		e := &external{client: &fake.MockPermissionsClient{
			AddUserFn: func(opt *sonar.PermissionsAddUserOptions) (*http.Response, error) {
				added = append(added, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
			RemoveUserFn: func(opt *sonar.PermissionsRemoveUserOptions) (*http.Response, error) {
				removed = append(removed, opt.Permission)

				return mockHTTPResponse(http.StatusOK), nil
			},
		}}

		_, err := e.Update(context.Background(), p)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"provisioning"}, sorted(added)); diff != "" {
			t.Fatalf("Update() permissions added mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]string{"admin"}, sorted(removed)); diff != "" {
			t.Fatalf("Update() permissions removed mismatch (-want +got):\n%s", diff)
		}
	})
}
