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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestLateInitializeGroup tests the LateInitializeGroup function.
func TestLateInitializeGroup(t *testing.T) {
	t.Parallel()

	t.Run("NilSpecNoPanic", func(t *testing.T) {
		t.Parallel()

		obs := &v1alpha1.GroupObservation{Description: "from-api"}
		LateInitializeGroup(nil, obs)
	})

	t.Run("NilObservationNoPanic", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		LateInitializeGroup(spec, nil)

		if spec.Description != nil {
			t.Fatalf("LateInitializeGroup() expected nil description, got %q", *spec.Description)
		}
	})

	t.Run("DescriptionInitializedWhenMissing", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Description: "engineering group"}

		LateInitializeGroup(spec, obs)

		if spec.Description == nil || *spec.Description != "engineering group" {
			t.Fatalf("LateInitializeGroup() description = %v, want %q", spec.Description, "engineering group")
		}
	})

	t.Run("DescriptionNotOverwrittenWhenPresent", func(t *testing.T) {
		t.Parallel()

		current := "custom"
		spec := &v1alpha1.GroupParameters{Name: "devs", Description: &current}
		obs := &v1alpha1.GroupObservation{Description: "engineering group"}

		LateInitializeGroup(spec, obs)

		if spec.Description == nil || *spec.Description != "custom" {
			t.Fatalf("LateInitializeGroup() description = %v, want %q", spec.Description, "custom")
		}
	})

	t.Run("EmptyObservationDescriptionInitialized", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Description: ""}

		LateInitializeGroup(spec, obs)

		if spec.Description == nil || *spec.Description != "" {
			t.Fatalf("LateInitializeGroup() description = %v, want empty string pointer", spec.Description)
		}
	})

	t.Run("PermissionsInitializedWhenMissing", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Permissions: []string{"read", "write"}}

		LateInitializeGroup(spec, obs)

		if spec.Permissions == nil || len(*spec.Permissions) != 2 {
			t.Fatalf("LateInitializeGroup() permissions = %v, want [\"read\", \"write\"]", spec.Permissions)
		}
	})

	t.Run("EmptyPermissionsInitialized", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Permissions: []string{}}

		LateInitializeGroup(spec, obs)

		if spec.Permissions == nil || len(*spec.Permissions) != 0 {
			t.Fatalf("LateInitializeGroup() permissions = %v, want empty slice pointer", spec.Permissions)
		}
	})

	t.Run("PermissionsNotOverwrittenWhenPresent", func(t *testing.T) {
		t.Parallel()

		existing := []string{"write"}
		spec := &v1alpha1.GroupParameters{Name: "devs", Permissions: &existing}
		obs := &v1alpha1.GroupObservation{Permissions: []string{"read", "write"}}

		LateInitializeGroup(spec, obs)

		if spec.Permissions == nil || len(*spec.Permissions) != 1 || (*spec.Permissions)[0] != "write" {
			t.Fatalf("LateInitializeGroup() permissions = %v, want [\"write\"]", spec.Permissions)
		}
	})
}

// TestIsGroupLateInitialized tests the IsGroupLateInitialized function.
func TestIsGroupLateInitialized(t *testing.T) {
	t.Parallel()

	d1 := "d1"
	d2 := "d2"

	cases := map[string]struct {
		former  *v1alpha1.GroupParameters
		current *v1alpha1.GroupParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.GroupParameters{Name: "devs"},
			want:    false,
		},
		"NilCurrent": {
			former:  &v1alpha1.GroupParameters{Name: "devs"},
			current: nil,
			want:    false,
		},
		"NoChanges": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			want:    false,
		},
		"NameChanged": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "admins", Description: &d1},
			want:    true,
		},
		"DescriptionChanged": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "devs", Description: &d2},
			want:    true,
		},

		"PermissionsAdded": {
			former:  &v1alpha1.GroupParameters{Name: "devs"},
			current: &v1alpha1.GroupParameters{Name: "devs", Permissions: &[]string{"read", "write"}},
			want:    true,
		},
		"EmptyDescriptionNotMeaningful": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: nil},
			current: &v1alpha1.GroupParameters{Name: "devs", Description: ptr.To("")},
			want:    false,
		},
		"EmptyPermissionsNotMeaningful": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Permissions: nil},
			current: &v1alpha1.GroupParameters{Name: "devs", Permissions: ptr.To([]string{})},
			want:    false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGroupLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsGroupLateInitialized() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestNewPermissionsClient tests the NewPermissionsClient function.
func TestNewPermissionsClient(t *testing.T) {
	t.Parallel()

	client := NewPermissionsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})

	if client == nil {
		t.Fatal("NewPermissionsClient() expected non-nil client")
	}
}

// TestGeneratePermissionsAddGroupOptions tests the GeneratePermissionsAddGroupOptions function.
func TestGeneratePermissionsAddGroupOptions(t *testing.T) {
	t.Parallel()

	got := GeneratePermissionsAddGroupOptions("devs", "read")
	if got == nil {
		t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
	}

	if got.GroupName != "devs" {
		t.Fatalf("GeneratePermissionsAddGroupOptions() GroupName = %q, want %q", got.GroupName, "devs")
	}

	if got.Permission != "read" {
		t.Fatalf("GeneratePermissionsAddGroupOptions() Permission = %q, want %q", got.Permission, "read")
	}
}

// TestGeneratePermissionsRemoveGroupOptions tests the GeneratePermissionsRemoveGroupOptions function.
func TestGeneratePermissionsRemoveGroupOptions(t *testing.T) {
	t.Parallel()

	got := GeneratePermissionsRemoveGroupOptions("devs", "write")
	if got == nil {
		t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
	}

	if got.GroupName != "devs" {
		t.Fatalf("GeneratePermissionsRemoveGroupOptions() GroupName = %q, want %q", got.GroupName, "devs")
	}

	if got.Permission != "write" {
		t.Fatalf("GeneratePermissionsRemoveGroupOptions() Permission = %q, want %q", got.Permission, "write")
	}
}

// TestGeneratePermissionsGroupsOptions tests the GeneratePermissionsGroupsOptions function.
func TestGeneratePermissionsGroupsOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		groupName  string
		pagination *sonar.PaginationArgs
		wantQuery  string
	}{
		"WithoutPagination": {
			groupName:  "devs",
			pagination: nil,
			wantQuery:  "devs",
		},
		"WithPagination": {
			groupName: "devs",
			pagination: &sonar.PaginationArgs{
				Page:     2,
				PageSize: 50,
			},
			wantQuery: "devs",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GeneratePermissionsGroupsOptions(tc.groupName, tc.pagination)
			if got == nil {
				t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
			}

			if got.Query != tc.wantQuery {
				t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want %q", got.Query, tc.wantQuery)
			}

			if tc.pagination != nil {
				if got.Page != tc.pagination.Page {
					t.Fatalf("GeneratePermissionsGroupsOptions() Page = %d, want %d", got.Page, tc.pagination.Page)
				}

				if got.PageSize != tc.pagination.PageSize {
					t.Fatalf("GeneratePermissionsGroupsOptions() PageSize = %d, want %d", got.PageSize, tc.pagination.PageSize)
				}
			}
		})
	}
}

func TestGenerateGroupCreateMembershipOptions(t *testing.T) {
	t.Parallel()

	got := GenerateGroupCreateMembershipOptions("group-1", "user-1")
	if got.GroupId != "group-1" || got.UserId != "user-1" {
		t.Fatalf("GenerateGroupCreateMembershipOptions() = %+v", got)
	}
}

func TestGenerateGroupSearchMembershipsOptions(t *testing.T) {
	t.Parallel()

	groupID := "group-1"
	userID := "user-1"
	pagination := &sonar.PaginationParamsV2{PageIndex: 2, PageSize: 50}

	got := GenerateGroupSearchMembershipsOptions(&groupID, &userID, pagination)
	if got.GroupId != groupID || got.UserId != userID {
		t.Fatalf("GenerateGroupSearchMembershipsOptions() ids mismatch: %+v", got)
	}

	if got.PageIndex != 2 || got.PageSize != 50 {
		t.Fatalf("GenerateGroupSearchMembershipsOptions() pagination mismatch: %+v", got.PaginationParamsV2)
	}
}

func TestGenerateGroupMembershipObservation(t *testing.T) {
	t.Parallel()

	if got := GenerateGroupMembershipObservation(nil); got != nil {
		t.Fatalf("GenerateGroupMembershipObservation(nil) = %v, want nil", got)
	}

	memberships := []sonar.GroupMembership{{GroupId: "g1", Id: "m1"}, {GroupId: "g2", Id: "m2"}}
	if diff := cmp.Diff(map[string]string{"g1": "m1", "g2": "m2"}, GenerateGroupMembershipObservation(&memberships)); diff != "" {
		t.Fatalf("GenerateGroupMembershipObservation() mismatch (-want +got):\n%s", diff)
	}
}

// TestIsGroupUpToDate tests the IsGroupUpToDate function.
func TestIsGroupUpToDate(t *testing.T) {
	t.Parallel()

	desc := "engineering"
	perms := []string{"read", "write"}

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
		obs  *v1alpha1.GroupObservation
		want bool
	}{
		"NilSpec": {
			spec: nil,
			obs:  &v1alpha1.GroupObservation{Name: "devs"},
			want: true,
		},
		"NilObservation": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
			obs:  nil,
			want: false,
		},
		"UpToDate": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "engineering"},
			want: true,
		},
		"NameMismatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "admins", Description: "engineering"},
			want: false,
		},
		"DescriptionMismatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "different"},
			want: false,
		},
		"NilDescriptionIsComparable": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: nil},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "any"},
			want: true,
		},
		"PermissionsMatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Permissions: &perms},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Permissions: perms},
			want: true,
		},
		"PermissionsMismatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Permissions: &[]string{"read"}},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Permissions: perms},
			want: false,
		},
		"NilPermissionsIsUpToDate": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Permissions: nil},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Permissions: perms},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGroupUpToDate(tc.spec, tc.obs); got != tc.want {
				t.Fatalf("IsGroupUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestGenerateCreateGroupOptions(t *testing.T) {
	t.Parallel()

	description := "engineering"
	permissions := []string{"read", "write"}

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
		want string
	}{
		"NilSpec": {
			spec: nil,
		},
		"WithoutDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
			want: "devs",
		},
		"WithDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &description},
			want: "devs",
		},
		"WithPermissions": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Permissions: &permissions},
			want: "devs",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateCreateGroupOptions(tc.spec)
			if tc.spec == nil {
				if got != nil {
					t.Fatal("GenerateCreateGroupOptions() expected nil, got non-nil")
				}

				return
			}

			if got == nil {
				t.Fatal("GenerateCreateGroupOptions() got nil, want non-nil")
			}

			if got.Name != tc.want {
				t.Fatalf("GenerateCreateGroupOptions() name = %q, want %q", got.Name, tc.want)
			}

			wantDescription := ""
			if tc.spec.Description != nil {
				wantDescription = *tc.spec.Description
			}

			if got.Description != wantDescription {
				t.Fatalf("GenerateCreateGroupOptions() description mismatch: got %q, want %q", got.Description, wantDescription)
			}
		})
	}
}

func TestGenerateUpdateGroupOptions(t *testing.T) {
	t.Parallel()

	description := "engineering"
	permissions := []string{"read", "write"}

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
	}{
		"NilSpec": {
			spec: nil,
		},
		"WithoutDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
		},
		"WithDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &description},
		},
		"WithPermissions": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Permissions: &permissions},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUpdateGroupOptions(tc.spec)
			if tc.spec == nil {
				if got != nil {
					t.Fatal("GenerateUpdateGroupOptions() expected nil, got non-nil")
				}

				return
			}

			if got == nil {
				t.Fatal("GenerateUpdateGroupOptions() got nil, want non-nil")
			}

			if got.Name != tc.spec.Name {
				t.Fatalf("GenerateUpdateGroupOptions() name = %q, want %q", got.Name, tc.spec.Name)
			}

			if !cmp.Equal(got.Description, tc.spec.Description) {
				t.Fatalf("GenerateUpdateGroupOptions() description mismatch: got %v, want %v", got.Description, tc.spec.Description)
			}
		})
	}
}

func TestGenerateGroupObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input *sonar.Group
		want  v1alpha1.GroupObservation
	}{
		"NilGroup": {
			input: nil,
			want:  v1alpha1.GroupObservation{},
		},
		"PopulatedGroup": {
			input: &sonar.Group{
				Id:          "group-1",
				Name:        "devs",
				Description: "engineering",
				Managed:     true,
				Default:     false,
			},
			want: v1alpha1.GroupObservation{
				ID:          "group-1",
				Name:        "devs",
				Description: "engineering",
				Managed:     true,
				Default:     false,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGroupObservation(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GenerateGroupObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewGroupsClient(t *testing.T) {
	t.Parallel()

	client := NewGroupsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})

	if client == nil {
		t.Fatal("NewGroupsClient() expected non-nil client")
	}
}

// TestArePermissionsEqual tests the ArePermissionsEqual function.
func TestArePermissionsEqual(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec     *[]string
		observed []string
		want     bool
	}{
		"NilSpec": {
			spec:     nil,
			observed: []string{"read", "write"},
			want:     true,
		},
		"EmptySpec": {
			spec:     &[]string{},
			observed: []string{},
			want:     true,
		},
		"EmptyObserved": {
			spec:     &[]string{"read"},
			observed: []string{},
			want:     false,
		},
		"MatchingPermissions": {
			spec:     &[]string{"read", "write"},
			observed: []string{"read", "write"},
			want:     true,
		},
		"MatchingPermissionsUnordered": {
			spec:     &[]string{"read", "write"},
			observed: []string{"write", "read"},
			want:     true,
		},
		"MatchingPermissionsWithDuplicates": {
			spec:     &[]string{"read", "read", "write"},
			observed: []string{"read", "write"},
			want:     true,
		},
		"MismatchingPermissions": {
			spec:     &[]string{"read"},
			observed: []string{"write"},
			want:     false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ArePermissionsEqual(tc.spec, tc.observed); got != tc.want {
				t.Fatalf("ArePermissionsEqual() = %t, want %t", got, tc.want)
			}
		})
	}
}
