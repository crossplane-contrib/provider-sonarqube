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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// testGroupName is a test constant for group names.
	testGroupName = "devs"
	// testProjectKey is a test constant for project keys.
	testProjectKey = "my-project"
)

// permissionScan is defined in permissions_template_test.go as permissionScan

// TestPermissionsNewClient tests creating a new permissions client.
func TestPermissionsNewClient(t *testing.T) {
	t.Parallel()

	t.Run("WithToken", func(t *testing.T) {
		t.Parallel()

		client := NewPermissionsClient(common.Config{
			AuthType: common.PersonalAccessToken,
			Token:    "token",
			BaseURL:  "http://localhost:9000",
		})

		if client == nil {
			t.Fatal("NewPermissionsClient() expected non-nil client")
		}
	})

	t.Run("WithBasicAuth", func(t *testing.T) {
		t.Parallel()

		client := NewPermissionsClient(common.Config{
			AuthType:  common.BasicAuth,
			BasicAuth: &common.BasicAuthArgs{Username: "user", Password: "pass"},
			BaseURL:   "http://localhost:9000",
		})

		if client == nil {
			t.Fatal("NewPermissionsClient() expected non-nil client")
		}
	})
}

// TestPermissionsGenerateAddGroupOptions tests generating add group options.
func TestPermissionsGenerateAddGroupOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddGroupOptions(testGroupName, permissionScan, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
		}

		if got.GroupName != testGroupName || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsAddGroupOptions() got = %+v, want GroupName=%s Permission=%s", got, testGroupName, permissionScan)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsAddGroupOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddGroupOptions("", "", nil)
		if got == nil {
			t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
		}

		if got.GroupName != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsAddGroupOptions() got = %+v, want empty values", got)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddGroupOptions(testGroupName, permissionScan, new(testProjectKey))
		if got == nil {
			t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsAddGroupOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})
}

// TestPermissionsGenerateRemoveGroupOptions tests generating remove group
// options.
func TestPermissionsGenerateRemoveGroupOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveGroupOptions(testGroupName, permissionScan, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
		}

		if got.GroupName != testGroupName || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() got = %+v, want GroupName=%s Permission=%s", got, testGroupName, permissionScan)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveGroupOptions("", "", nil)
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
		}

		if got.GroupName != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() got = %+v, want empty values", got)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveGroupOptions(testGroupName, permissionScan, new(testProjectKey))
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})
}

// TestPermissionsGenerateGroupsOptions tests generating groups options.
func TestPermissionsGenerateGroupsOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithoutPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions(testGroupName, nil, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.Query != testGroupName {
			t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want %q", got.Query, testGroupName)
		}

		if got.Page != 0 || got.PageSize != 0 {
			t.Fatalf("GeneratePermissionsGroupsOptions() pagination unexpectedly set: page=%d pageSize=%d", got.Page, got.PageSize)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsGroupsOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions(testGroupName, nil, &sonar.PaginationArgs{Page: 2, PageSize: 100})
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.Query != testGroupName {
			t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want %q", got.Query, testGroupName)
		}

		if got.Page != 2 || got.PageSize != 100 {
			t.Fatalf("GeneratePermissionsGroupsOptions() pagination got page=%d pageSize=%d, want page=2 pageSize=100", got.Page, got.PageSize)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions(testGroupName, new(testProjectKey), nil)
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsGroupsOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})

	t.Run("EmptyGroupName", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions("", nil, &sonar.PaginationArgs{Page: 1, PageSize: 50})
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.Query != "" {
			t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want empty string", got.Query)
		}
	})
}

// TestPermissionsGenerateAddUserOptions tests generating add user options.
func TestPermissionsGenerateAddUserOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddUserOptions(testUserLogin, permissionScan, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsAddUserOptions() expected non-nil options")
		}

		if got.Login != testUserLogin || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsAddUserOptions() got = %+v, want Login=%s Permission=%s", got, testUserLogin, permissionScan)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsAddUserOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddUserOptions("", "", nil)
		if got == nil {
			t.Fatal("GeneratePermissionsAddUserOptions() expected non-nil options")
		}

		if got.Login != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsAddUserOptions() got = %+v, want empty values", got)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddUserOptions(testUserLogin, permissionScan, new(testProjectKey))
		if got == nil {
			t.Fatal("GeneratePermissionsAddUserOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsAddUserOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})
}

// TestPermissionsGenerateRemoveUserOptions tests generating remove user
// options.
func TestPermissionsGenerateRemoveUserOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveUserOptions(testUserLogin, permissionScan, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveUserOptions() expected non-nil options")
		}

		if got.Login != testUserLogin || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsRemoveUserOptions() got = %+v, want Login=%s Permission=%s", got, testUserLogin, permissionScan)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsRemoveUserOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveUserOptions("", "", nil)
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveUserOptions() expected non-nil options")
		}

		if got.Login != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsRemoveUserOptions() got = %+v, want empty values", got)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveUserOptions(testUserLogin, permissionScan, new(testProjectKey))
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveUserOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsRemoveUserOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})
}

// TestPermissionsGenerateUsersOptions tests generating users options.
func TestPermissionsGenerateUsersOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithoutPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsUsersOptions(testUserLogin, nil, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsUsersOptions() expected non-nil options")
		}

		if got.Query != testUserLogin {
			t.Fatalf("GeneratePermissionsUsersOptions() Query = %q, want %q", got.Query, testUserLogin)
		}

		if got.Page != 0 || got.PageSize != 0 {
			t.Fatalf("GeneratePermissionsUsersOptions() pagination unexpectedly set: page=%d pageSize=%d", got.Page, got.PageSize)
		}

		if got.ProjectKey != "" {
			t.Fatalf("GeneratePermissionsUsersOptions() unexpected ProjectKey %q", got.ProjectKey)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsUsersOptions(testUserLogin, nil, &sonar.PaginationArgs{Page: 2, PageSize: 100})
		if got == nil {
			t.Fatal("GeneratePermissionsUsersOptions() expected non-nil options")
		}

		if got.Page != 2 || got.PageSize != 100 {
			t.Fatalf("GeneratePermissionsUsersOptions() pagination got page=%d pageSize=%d, want page=2 pageSize=100", got.Page, got.PageSize)
		}
	})

	t.Run("WithProjectKey", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsUsersOptions(testUserLogin, new(testProjectKey), nil)
		if got == nil {
			t.Fatal("GeneratePermissionsUsersOptions() expected non-nil options")
		}

		if got.ProjectKey != testProjectKey {
			t.Fatalf("GeneratePermissionsUsersOptions() ProjectKey = %q, want %q", got.ProjectKey, testProjectKey)
		}
	})

	t.Run("EmptyLogin", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsUsersOptions("", nil, &sonar.PaginationArgs{Page: 1, PageSize: 50})
		if got == nil {
			t.Fatal("GeneratePermissionsUsersOptions() expected non-nil options")
		}

		if got.Query != "" {
			t.Fatalf("GeneratePermissionsUsersOptions() Query = %q, want empty string", got.Query)
		}
	})
}

// TestLateInitializePermissions tests that LateInitializePermissions is a
// no-op.
func TestLateInitializePermissions(t *testing.T) {
	t.Parallel()

	t.Run("DoesNotPanic", func(t *testing.T) {
		t.Parallel()

		devs := "devs"
		spec := &v1alpha1.PermissionsParameters{
			GroupName:   &devs,
			Permissions: []string{"scan"},
		}
		obs := &v1alpha1.PermissionsObservation{
			Permissions: []string{"admin"},
		}

		// Should not panic or modify the spec.
		LateInitializePermissions(spec, obs)

		if len(spec.Permissions) != 1 || spec.Permissions[0] != "scan" {
			t.Fatalf("LateInitializePermissions() modified spec unexpectedly: %+v", spec.Permissions)
		}
	})

	t.Run("NilInputsDoNotPanic", func(t *testing.T) {
		t.Parallel()

		LateInitializePermissions(nil, nil)
	})
}

// TestIsPermissionsLateInitialized tests that IsPermissionsLateInitialized
// always returns false.
func TestIsPermissionsLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.PermissionsParameters
		current *v1alpha1.PermissionsParameters
	}{
		"BothNil":    {former: nil, current: nil},
		"FormerNil":  {former: nil, current: &v1alpha1.PermissionsParameters{Permissions: []string{"scan"}}},
		"CurrentNil": {former: &v1alpha1.PermissionsParameters{Permissions: []string{"scan"}}, current: nil},
		"BothNonNil": {former: &v1alpha1.PermissionsParameters{Permissions: []string{"scan"}}, current: &v1alpha1.PermissionsParameters{Permissions: []string{"admin"}}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsPermissionsLateInitialized(tc.former, tc.current); got != false {
				t.Fatalf("IsPermissionsLateInitialized() = %v, want false", got)
			}
		})
	}
}

// TestIsPermissionsUpToDate tests checking if permissions spec is up to date.
func TestIsPermissionsUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.PermissionsParameters
		obs  *v1alpha1.PermissionsObservation
		want bool
	}{
		"NilSpec": {
			spec: nil,
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{"admin"}},
			want: true,
		},
		"NilObs": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{"admin"}},
			obs:  nil,
			want: false,
		},
		"MatchExact": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{"scan", "admin"}},
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{"admin", "scan"}},
			want: true,
		},
		"MatchEmpty": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{}},
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{}},
			want: true,
		},
		"ExtraObserved": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{"scan"}},
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{"scan", "admin"}},
			want: false,
		},
		"ExtraSpec": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{"scan", "admin"}},
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{"scan"}},
			want: false,
		},
		"Mismatch": {
			spec: &v1alpha1.PermissionsParameters{Permissions: []string{"admin"}},
			obs:  &v1alpha1.PermissionsObservation{Permissions: []string{"scan"}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsPermissionsUpToDate(tc.spec, tc.obs); got != tc.want {
				t.Fatalf("IsPermissionsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestPermissionsAreEqual tests checking if permissions are equal.
func TestPermissionsAreEqual(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec     *[]string
		observed []string
		want     bool
	}{
		"NilSpec": {
			spec:     nil,
			observed: []string{"admin"},
			want:     true,
		},
		"BothEmpty": {
			spec:     &[]string{},
			observed: []string{},
			want:     true,
		},
		"SameDifferentOrder": {
			spec:     &[]string{"scan", "admin"},
			observed: []string{"admin", "scan"},
			want:     true,
		},
		"DuplicateSpecEqualDeduped": {
			spec:     &[]string{"scan", "scan", "admin"},
			observed: []string{"admin", "scan"},
			want:     true,
		},
		"ObservedHasExtra": {
			spec:     &[]string{"scan"},
			observed: []string{"scan", "admin"},
			want:     false,
		},
		"SpecHasExtra": {
			spec:     &[]string{"scan", "admin"},
			observed: []string{"scan"},
			want:     false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ArePermissionsEqual(tc.spec, tc.observed); got != tc.want {
				t.Fatalf("ArePermissionsEqual() = %v, want %v", got, tc.want)
			}
		})
	}
}
