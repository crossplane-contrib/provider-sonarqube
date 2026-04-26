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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	testGroupName = "devs"
)

// permissionScan is defined in permissions_template_test.go as permissionScan

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

func TestPermissionsGenerateAddGroupOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddGroupOptions(testGroupName, permissionScan)
		if got == nil {
			t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
		}

		if got.GroupName != testGroupName || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsAddGroupOptions() got = %+v, want GroupName=%s Permission=%s", got, testGroupName, permissionScan)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsAddGroupOptions("", "")
		if got == nil {
			t.Fatal("GeneratePermissionsAddGroupOptions() expected non-nil options")
		}

		if got.GroupName != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsAddGroupOptions() got = %+v, want empty values", got)
		}
	})
}

func TestPermissionsGenerateRemoveGroupOptions(t *testing.T) {
	t.Parallel()

	t.Run("RegularValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveGroupOptions(testGroupName, permissionScan)
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
		}

		if got.GroupName != testGroupName || got.Permission != permissionScan {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() got = %+v, want GroupName=%s Permission=%s", got, testGroupName, permissionScan)
		}
	})

	t.Run("EmptyValues", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsRemoveGroupOptions("", "")
		if got == nil {
			t.Fatal("GeneratePermissionsRemoveGroupOptions() expected non-nil options")
		}

		if got.GroupName != "" || got.Permission != "" {
			t.Fatalf("GeneratePermissionsRemoveGroupOptions() got = %+v, want empty values", got)
		}
	})
}

func TestPermissionsGenerateGroupsOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithoutPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions(testGroupName, nil)
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.Query != testGroupName {
			t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want %q", got.Query, testGroupName)
		}

		if got.Page != 0 || got.PageSize != 0 {
			t.Fatalf("GeneratePermissionsGroupsOptions() pagination unexpectedly set: page=%d pageSize=%d", got.Page, got.PageSize)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions(testGroupName, &sonar.PaginationArgs{Page: 2, PageSize: 100})
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

	t.Run("EmptyGroupName", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsGroupsOptions("", &sonar.PaginationArgs{Page: 1, PageSize: 50})
		if got == nil {
			t.Fatal("GeneratePermissionsGroupsOptions() expected non-nil options")
		}

		if got.Query != "" {
			t.Fatalf("GeneratePermissionsGroupsOptions() Query = %q, want empty string", got.Query)
		}
	})
}

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
