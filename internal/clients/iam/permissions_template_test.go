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
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// permissionsTemplateID is the test permissions template ID.
	permissionsTemplateID = "template-id"
	// permissionScan is the scan permission for testing.
	permissionScan = "scan"
	// templateNameA is the name of test template A.
	templateNameA = "template-a"
	// templateDescription is the description of the test template.
	templateDescription = "desc"
	// projectKeyPattern is the project key pattern for testing.
	projectKeyPattern = "proj-.*"
)

// TestLateInitializePermissionsTemplate tests template late initialization.
func TestLateInitializePermissionsTemplate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PermissionsTemplateParameters
		observation *v1alpha1.PermissionsTemplateObservation
		want        *v1alpha1.PermissionsTemplateParameters
	}{
		"NilInputsDoNothing": {
			spec:        nil,
			observation: nil,
			want:        nil,
		},
		"InitializesMissingFieldsOnly": {
			spec: &v1alpha1.PermissionsTemplateParameters{Name: templateNameA},
			observation: &v1alpha1.PermissionsTemplateObservation{
				Description:        "generated",
				ProjectKeyPattern:  projectKeyPattern,
				Default:            false,
				CreatorPermissions: []string{"scan", "admin"},
			},
			want: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new("generated"),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(false),
				CreatorPermissions: &[]string{"scan", "admin"},
			},
		},
		"PreservesExistingValues": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new("custom"),
				ProjectKeyPattern:  new("existing"),
				Default:            new(true),
				CreatorPermissions: &[]string{"user"},
			},
			observation: &v1alpha1.PermissionsTemplateObservation{
				Description:        "generated",
				ProjectKeyPattern:  projectKeyPattern,
				Default:            false,
				CreatorPermissions: []string{"scan", "admin"},
			},
			want: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new("custom"),
				ProjectKeyPattern:  new("existing"),
				Default:            new(true),
				CreatorPermissions: &[]string{"user"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializePermissionsTemplate(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.want, tc.spec); diff != "" {
				t.Fatalf("LateInitializePermissionsTemplate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsPermissionsTemplateLateInitialized tests detecting late initialization.
func TestIsPermissionsTemplateLateInitialized(t *testing.T) {
	t.Parallel()

	currentDescription := "current"
	currentPattern := "pattern"
	currentPermissions := []string{"admin"}

	cases := map[string]struct {
		former  *v1alpha1.PermissionsTemplateParameters
		current *v1alpha1.PermissionsTemplateParameters
		want    bool
	}{
		"NilFormerReturnsFalse": {
			former:  nil,
			current: &v1alpha1.PermissionsTemplateParameters{},
			want:    false,
		},
		"NilCurrentReturnsFalse": {
			former:  &v1alpha1.PermissionsTemplateParameters{},
			current: nil,
			want:    false,
		},
		"SameValuesReturnsFalse": {
			former: &v1alpha1.PermissionsTemplateParameters{
				Description:        new(templateDescription),
				ProjectKeyPattern:  new("pattern"),
				CreatorPermissions: &[]string{"scan"},
			},
			current: &v1alpha1.PermissionsTemplateParameters{
				Description:        new(templateDescription),
				ProjectKeyPattern:  new("pattern"),
				CreatorPermissions: &[]string{"scan"},
			},
			want: false,
		},
		"DescriptionChangedReturnsTrue": {
			former:  &v1alpha1.PermissionsTemplateParameters{Description: new("former")},
			current: &v1alpha1.PermissionsTemplateParameters{Description: &currentDescription},
			want:    true,
		},
		"ProjectKeyPatternChangedReturnsTrue": {
			former:  &v1alpha1.PermissionsTemplateParameters{ProjectKeyPattern: new("former")},
			current: &v1alpha1.PermissionsTemplateParameters{ProjectKeyPattern: &currentPattern},
			want:    true,
		},
		"CreatorPermissionsAddedReturnsTrue": {
			former:  &v1alpha1.PermissionsTemplateParameters{},
			current: &v1alpha1.PermissionsTemplateParameters{CreatorPermissions: &currentPermissions},
			want:    true,
		},
		"CreatorPermissionsRemovedReturnsTrue": {
			former:  &v1alpha1.PermissionsTemplateParameters{CreatorPermissions: &currentPermissions},
			current: &v1alpha1.PermissionsTemplateParameters{},
			want:    true,
		},
		"CreatorPermissionsEqualReturnsFalse": {
			former:  &v1alpha1.PermissionsTemplateParameters{CreatorPermissions: &[]string{"admin", "scan"}},
			current: &v1alpha1.PermissionsTemplateParameters{CreatorPermissions: &[]string{"scan", "admin"}},
			want:    false,
		},
		"EmptyCreatorPermissionsIgnored": {
			former:  &v1alpha1.PermissionsTemplateParameters{},
			current: &v1alpha1.PermissionsTemplateParameters{CreatorPermissions: &[]string{}},
			want:    false,
		},
		"DefaultChangedReturnsTrue": {
			former:  &v1alpha1.PermissionsTemplateParameters{Default: new(false)},
			current: &v1alpha1.PermissionsTemplateParameters{Default: new(true)},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsPermissionsTemplateLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsPermissionsTemplateLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsPermissionsTemplateUpToDate tests template up-to-date status.
func TestIsPermissionsTemplateUpToDate(t *testing.T) {
	t.Parallel()

	creatorPermissions := []string{"scan"}
	groupPermissions := []v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &[]string{"scan"}}}
	userPermissions := []v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &[]string{"scan"}}}

	baseObservation := &v1alpha1.PermissionsTemplateObservation{
		ID:                 "template-1",
		Name:               templateNameA,
		Description:        templateDescription,
		ProjectKeyPattern:  projectKeyPattern,
		Default:            true,
		CreatorPermissions: []string{"scan"},
		GroupPermissions:   []v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}},
		UserPermissions:    []v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}},
	}

	cases := map[string]struct {
		spec        *v1alpha1.PermissionsTemplateParameters
		observation *v1alpha1.PermissionsTemplateObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: baseObservation,
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{Name: templateNameA},
			want: false,
		},
		"EverythingMatchesReturnsTrue": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new(templateDescription),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(true),
				CreatorPermissions: &creatorPermissions,
				GroupPermissions:   &groupPermissions,
				UserPermissions:    &userPermissions,
			},
			observation: baseObservation,
			want:        true,
		},
		"DefaultDiffReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new(templateDescription),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(false),
				CreatorPermissions: &creatorPermissions,
				GroupPermissions:   &groupPermissions,
				UserPermissions:    &userPermissions,
			},
			observation: baseObservation,
			want:        false,
		},
		"CreatorPermissionsDiffReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new(templateDescription),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(true),
				CreatorPermissions: &[]string{"admin"},
				GroupPermissions:   &groupPermissions,
				UserPermissions:    &userPermissions,
			},
			observation: baseObservation,
			want:        false,
		},
		"GroupPermissionsDiffReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new(templateDescription),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(true),
				CreatorPermissions: &creatorPermissions,
				GroupPermissions:   &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &[]string{"admin"}}},
				UserPermissions:    &userPermissions,
			},
			observation: baseObservation,
			want:        false,
		},
		"UserPermissionsDiffReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:               templateNameA,
				Description:        new(templateDescription),
				ProjectKeyPattern:  new(projectKeyPattern),
				Default:            new(true),
				CreatorPermissions: &creatorPermissions,
				GroupPermissions:   &groupPermissions,
				UserPermissions:    &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &[]string{"admin"}}},
			},
			observation: baseObservation,
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsPermissionsTemplateUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Fatalf("IsPermissionsTemplateUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestArePermissionsTemplateBaseFieldsUpToDate tests base fields up-to-date.
func TestArePermissionsTemplateBaseFieldsUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PermissionsTemplateParameters
		observation *v1alpha1.PermissionsTemplateObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: nil,
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec: &v1alpha1.PermissionsTemplateParameters{Name: templateNameA},
			want: false,
		},
		"MatchingValuesReturnsTrue": {
			spec: &v1alpha1.PermissionsTemplateParameters{
				Name:              templateNameA,
				Description:       new(templateDescription),
				ProjectKeyPattern: new(projectKeyPattern),
			},
			observation: &v1alpha1.PermissionsTemplateObservation{
				Name:              templateNameA,
				Description:       templateDescription,
				ProjectKeyPattern: projectKeyPattern,
			},
			want: true,
		},
		"NameDiffReturnsFalse": {
			spec:        &v1alpha1.PermissionsTemplateParameters{Name: templateNameA},
			observation: &v1alpha1.PermissionsTemplateObservation{Name: "template-b"},
			want:        false,
		},
		"DescriptionDiffReturnsFalse": {
			spec:        &v1alpha1.PermissionsTemplateParameters{Name: templateNameA, Description: new("spec")},
			observation: &v1alpha1.PermissionsTemplateObservation{Name: templateNameA, Description: "obs"},
			want:        false,
		},
		"PatternDiffReturnsFalse": {
			spec:        &v1alpha1.PermissionsTemplateParameters{Name: templateNameA, ProjectKeyPattern: new("spec")},
			observation: &v1alpha1.PermissionsTemplateObservation{Name: templateNameA, ProjectKeyPattern: "obs"},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ArePermissionsTemplateBaseFieldsUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Fatalf("ArePermissionsTemplateBaseFieldsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGeneratePermissionsTemplateSearchOptions tests generating search options.
func TestGeneratePermissionsTemplateSearchOptions(t *testing.T) {
	t.Parallel()

	withPagination := GeneratePermissionsTemplateSearchOptions(templateNameA, &sonar.PaginationArgs{Page: 2, PageSize: 50})
	if withPagination == nil {
		t.Fatal("GeneratePermissionsTemplateSearchOptions() expected non-nil options")
	}

	if withPagination.Query != templateNameA || withPagination.Page != 2 || withPagination.PageSize != 50 {
		t.Fatalf("GeneratePermissionsTemplateSearchOptions() got %+v, want query and pagination to match", withPagination)
	}

	withoutPagination := GeneratePermissionsTemplateSearchOptions("template-b", nil)
	if withoutPagination == nil {
		t.Fatal("GeneratePermissionsTemplateSearchOptions() expected non-nil options")
	}

	if withoutPagination.Query != "template-b" || withoutPagination.Page != 0 || withoutPagination.PageSize != 0 {
		t.Fatalf("GeneratePermissionsTemplateSearchOptions() got %+v, want zero pagination", withoutPagination)
	}
}

// TestGeneratePermissionsTemplateUpdateOptions tests generating update options.
func TestGeneratePermissionsTemplateUpdateOptions(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.PermissionsTemplateParameters{
		Name:              templateNameA,
		Description:       new(templateDescription),
		ProjectKeyPattern: new(projectKeyPattern),
	}

	got := GeneratePermissionsTemplateUpdateOptions(permissionsTemplateID, spec)
	if got == nil {
		t.Fatal("GeneratePermissionsTemplateUpdateOptions() expected non-nil options")
	}

	if got.ID != permissionsTemplateID || got.Name != templateNameA {
		t.Fatalf("GeneratePermissionsTemplateUpdateOptions() got %+v, want ID and Name to match", got)
	}

	if got.Description != templateDescription {
		t.Fatalf("GeneratePermissionsTemplateUpdateOptions() description = %q, want %q", got.Description, templateDescription)
	}

	if got.ProjectKeyPattern != projectKeyPattern {
		t.Fatalf("GeneratePermissionsTemplateUpdateOptions() projectKeyPattern = %q, want %q", got.ProjectKeyPattern, projectKeyPattern)
	}
}

// TestArePermissionsTemplateGroupsUpToDate tests group permissions up-to-date.
func TestArePermissionsTemplateGroupsUpToDate(t *testing.T) {
	t.Parallel()

	specPermissions := []string{"scan"}
	obsPermissions := []string{"scan"}

	cases := map[string]struct {
		spec        *[]v1alpha1.PermissionsTemplateGroupParameters
		observation *[]v1alpha1.PermissionsTemplateGroupObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec: &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &specPermissions}},
			want: false,
		},
		"MatchingGroupsDifferentOrderReturnsTrue": {
			spec:        &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &specPermissions}, {Name: "qa", Permissions: &[]string{}}},
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "qa", Permissions: []string{}}, {Name: "devs", Permissions: obsPermissions}},
			want:        true,
		},
		"EmptyObservedGroupIgnored": {
			spec:        &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{}}, {Name: "devs", Permissions: []string{"scan"}}},
			want:        true,
		},
		"MissingObservedGroupReturnsFalse": {
			spec:        &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "other", Permissions: []string{"scan"}}},
			want:        false,
		},
		"PermissionsDiffReturnsFalse": {
			spec:        &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &[]string{"admin"}}},
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}},
			want:        false,
		},
		"EmptyGroupNameMissingMatchFails": {
			spec:        &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ArePermissionsTemplateGroupsUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Fatalf("ArePermissionsTemplateGroupsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCreatePermissionsTemplateGroupMapping tests group permission mappings.
func TestCreatePermissionsTemplateGroupMapping(t *testing.T) {
	t.Parallel()

	specPermissions := []string{"scan"}
	spec := &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "", Permissions: &specPermissions}, {Name: "devs", Permissions: &specPermissions}, {Name: "qa", Permissions: &[]string{}}}
	observation := &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}, {Name: "qa", Permissions: []string{}}, {Name: "other", Permissions: []string{"admin"}}}

	got := CreatePermissionsTemplateGroupMapping(spec, observation)
	want := map[string][2]int{
		"devs":  {1, 0},
		"qa":    {2, 1},
		"other": {-1, 2},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("CreatePermissionsTemplateGroupMapping() mismatch (-want +got):\n%s", diff)
	}
}

// TestCreatePermissionsTemplateGroupMappingSkipsEmptyObservedEntries
// tests skipping empty groups.
func TestCreatePermissionsTemplateGroupMappingSkipsEmptyObservedEntries(t *testing.T) {
	t.Parallel()

	specPermissions := []string{"scan"}
	spec := &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &specPermissions}}
	observation := &[]v1alpha1.PermissionsTemplateGroupObservation{
		{Name: "", Permissions: []string{"scan"}},
		{Name: "ghost", Permissions: []string{}},
	}

	got := CreatePermissionsTemplateGroupMapping(spec, observation)
	want := map[string][2]int{"devs": {0, -1}}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("CreatePermissionsTemplateGroupMapping() mismatch (-want +got):\n%s", diff)
	}
}

// TestArePermissionsTemplateUsersUpToDate tests user permissions up-to-date.
func TestArePermissionsTemplateUsersUpToDate(t *testing.T) {
	t.Parallel()

	specPermissions := []string{"scan"}

	cases := map[string]struct {
		spec        *[]v1alpha1.PermissionsTemplateUserParameters
		observation *[]v1alpha1.PermissionsTemplateUserObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec: &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &specPermissions}},
			want: false,
		},
		"MatchingUsersReturnsTrue": {
			spec:        &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}},
			want:        true,
		},
		"EmptyPermissionsInSpecAndObservationMatch": {
			spec:        &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &[]string{}}},
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{}}},
			want:        true,
		},
		"EmptyObservedUserIgnored": {
			spec:        &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{}}, {Login: "alice", Permissions: []string{"scan"}}},
			want:        true,
		},
		"MissingUserReturnsFalse": {
			spec:        &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &specPermissions}},
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "bob", Permissions: []string{"scan"}}},
			want:        false,
		},
		"PermissionDiffReturnsFalse": {
			spec:        &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &[]string{"admin"}}},
			observation: &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ArePermissionsTemplateUsersUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Fatalf("ArePermissionsTemplateUsersUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCreatePermissionsTemplateUserMapping tests user mappings.
func TestCreatePermissionsTemplateUserMapping(t *testing.T) {
	t.Parallel()

	specPermissions := []string{"scan"}
	spec := &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &specPermissions}, {Login: "bob", Permissions: &[]string{}}}
	observation := &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}, {Login: "bob", Permissions: []string{}}, {Login: "charlie", Permissions: []string{}}}

	got := CreatePermissionsTemplateUserMapping(spec, observation)
	want := map[string][2]int{
		"alice": {0, 0},
		"bob":   {1, 1},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("CreatePermissionsTemplateUserMapping() mismatch (-want +got):\n%s", diff)
	}
}

// TestGeneratePermissionsTemplateOptions tests template options.
func TestGeneratePermissionsTemplateOptions(t *testing.T) {
	t.Parallel()

	creationSpec := &v1alpha1.PermissionsTemplateParameters{Name: templateNameA, Description: new(templateDescription), ProjectKeyPattern: new(projectKeyPattern)}

	gotCreate := GeneratePermissionsTemplateCreationOptions(creationSpec)
	if gotCreate.Name != templateNameA || gotCreate.Description != templateDescription || gotCreate.ProjectKeyPattern != projectKeyPattern {
		t.Fatalf("GeneratePermissionsTemplateCreationOptions() got %+v", gotCreate)
	}

	gotDelete := GeneratePermissionsTemplateDeleteOptions(permissionsTemplateID)
	if gotDelete.TemplateID != permissionsTemplateID {
		t.Fatalf("GeneratePermissionsTemplateDeleteOptions() TemplateID = %q, want %q", gotDelete.TemplateID, permissionsTemplateID)
	}

	gotDefault := GeneratePermissionsTemplateSetAsDefaultOptions(permissionsTemplateID)
	if gotDefault.TemplateID != permissionsTemplateID {
		t.Fatalf("GeneratePermissionsTemplateSetAsDefaultOptions() TemplateID = %q, want %q", gotDefault.TemplateID, permissionsTemplateID)
	}

	gotGroups := GeneratePermissionsTemplateGroupsSearchOptions(permissionsTemplateID, &sonar.PaginationArgs{Page: 3, PageSize: 25})
	if gotGroups.TemplateID != permissionsTemplateID || gotGroups.Page != 3 || gotGroups.PageSize != 25 {
		t.Fatalf("GeneratePermissionsTemplateGroupsSearchOptions() got %+v", gotGroups)
	}

	gotUsers := GeneratePermissionsTemplateUsersSearchOptions(permissionsTemplateID, &sonar.PaginationArgs{Page: 4, PageSize: 50})
	if gotUsers.TemplateID != permissionsTemplateID || gotUsers.Page != 4 || gotUsers.PageSize != 50 {
		t.Fatalf("GeneratePermissionsTemplateUsersSearchOptions() got %+v", gotUsers)
	}

	gotAddGroup := GeneratePermissionsTemplateAddGroupPermissionOptions(permissionsTemplateID, "devs", permissionScan)
	if gotAddGroup.TemplateID != permissionsTemplateID || gotAddGroup.GroupName != testGroupName || gotAddGroup.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateAddGroupPermissionOptions() got %+v", gotAddGroup)
	}

	gotRemoveGroup := GeneratePermissionsTemplateRemoveGroupPermissionOptions(permissionsTemplateID, "devs", permissionScan)
	if gotRemoveGroup.TemplateID != permissionsTemplateID || gotRemoveGroup.GroupName != "devs" || gotRemoveGroup.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateRemoveGroupPermissionOptions() got %+v", gotRemoveGroup)
	}

	gotAddUser := GeneratePermissionsTemplateAddUserPermissionOptions(permissionsTemplateID, "alice", permissionScan)
	if gotAddUser.TemplateID != permissionsTemplateID || gotAddUser.Login != testUserLogin || gotAddUser.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateAddUserPermissionOptions() got %+v", gotAddUser)
	}

	gotRemoveUser := GeneratePermissionsTemplateRemoveUserPermissionOptions(permissionsTemplateID, "alice", permissionScan)
	if gotRemoveUser.TemplateID != permissionsTemplateID || gotRemoveUser.Login != "alice" || gotRemoveUser.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateRemoveUserPermissionOptions() got %+v", gotRemoveUser)
	}

	gotAddCreator := GeneratePermissionsTemplateAddCreatorPermissionOptions(permissionsTemplateID, permissionScan)
	if gotAddCreator.TemplateID != permissionsTemplateID || gotAddCreator.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateAddCreatorPermissionOptions() got %+v", gotAddCreator)
	}

	gotRemoveCreator := GeneratePermissionsTemplateRemoveCreatorPermissionOptions(permissionsTemplateID, permissionScan)
	if gotRemoveCreator.TemplateID != permissionsTemplateID || gotRemoveCreator.Permission != permissionScan {
		t.Fatalf("GeneratePermissionsTemplateRemoveCreatorPermissionOptions() got %+v", gotRemoveCreator)
	}
}

// TestUpdatePermissionsTemplateObservation tests template observation updates.
func TestUpdatePermissionsTemplateObservation(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-03T10:30:00Z"
	updatedAt := "2026-04-03T11:45:00Z"
	template := &sonar.PermissionTemplate{
		ID:                "template-1",
		Name:              templateNameA,
		Description:       templateDescription,
		ProjectKeyPattern: projectKeyPattern,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		Permissions: []sonar.PermissionsTemplatePermission{
			{Key: "scan", WithProjectCreator: true},
			{Key: "admin", WithProjectCreator: false},
			{Key: "issueadmin", WithProjectCreator: true},
		},
	}

	obs := &v1alpha1.PermissionsTemplateObservation{}
	UpdatePermissionsTemplateObservation(obs, template, true)

	if obs.ID != "template-1" || obs.Name != templateNameA || obs.Description != templateDescription || obs.ProjectKeyPattern != projectKeyPattern || !obs.Default {
		t.Fatalf("UpdatePermissionsTemplateObservation() got %+v", obs)
	}

	if obs.CreatedAt == nil || obs.UpdatedAt == nil || !obs.CreatedAt.Time.Equal(time.Date(2026, time.April, 3, 10, 30, 0, 0, time.UTC)) || !obs.UpdatedAt.Time.Equal(time.Date(2026, time.April, 3, 11, 45, 0, 0, time.UTC)) {
		t.Fatalf("UpdatePermissionsTemplateObservation() timestamps got createdAt=%v updatedAt=%v", obs.CreatedAt, obs.UpdatedAt)
	}

	if diff := cmp.Diff([]string{"scan", "issueadmin"}, obs.CreatorPermissions); diff != "" {
		t.Fatalf("UpdatePermissionsTemplateObservation() creator permissions mismatch (-want +got):\n%s", diff)
	}
}

// TestGeneratePermissionsTemplateObservations tests template observations.
func TestGeneratePermissionsTemplateObservations(t *testing.T) {
	t.Parallel()

	t.Run("GroupObservations", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsTemplateGroupObservations(&[]sonar.PermissionsTemplateGroup{{Name: "devs", Permissions: []string{"scan"}}, {Name: "qa", Permissions: []string{}}})

		want := []v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}, {Name: "qa", Permissions: []string{}}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("GeneratePermissionsTemplateGroupObservations() mismatch (-want +got):\n%s", diff)
		}

		if got := GeneratePermissionsTemplateGroupObservations(nil); got != nil {
			t.Fatalf("GeneratePermissionsTemplateGroupObservations(nil) = %v, want nil", got)
		}
	})

	t.Run("UserObservations", func(t *testing.T) {
		t.Parallel()

		got := GeneratePermissionsTemplateUserObservations(&[]sonar.PermissionsTemplateUser{{Login: "alice", Permissions: []string{"scan"}}, {Login: "bob", Permissions: []string{}}})

		want := []v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}, {Login: "bob", Permissions: []string{}}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("GeneratePermissionsTemplateUserObservations() mismatch (-want +got):\n%s", diff)
		}

		if got := GeneratePermissionsTemplateUserObservations(nil); got != nil {
			t.Fatalf("GeneratePermissionsTemplateUserObservations(nil) = %v, want nil", got)
		}
	})

	t.Run("SingleObservations", func(t *testing.T) {
		t.Parallel()

		group := GeneratePermissionsTemplateGroupObservation(&sonar.PermissionsTemplateGroup{Name: "devs", Permissions: []string{"scan"}})
		if diff := cmp.Diff(&v1alpha1.PermissionsTemplateGroupObservation{Name: "devs", Permissions: []string{"scan"}}, group); diff != "" {
			t.Fatalf("GeneratePermissionsTemplateGroupObservation() mismatch (-want +got):\n%s", diff)
		}

		user := GeneratePermissionsTemplateUserObservation(&sonar.PermissionsTemplateUser{Login: "alice", Permissions: []string{"scan"}})
		if diff := cmp.Diff(&v1alpha1.PermissionsTemplateUserObservation{Login: "alice", Permissions: []string{"scan"}}, user); diff != "" {
			t.Fatalf("GeneratePermissionsTemplateUserObservation() mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestNewPermissionsTemplatesClient tests creating a new client.
func TestNewPermissionsTemplatesClient(t *testing.T) {
	t.Parallel()

	client := NewPermissionsTemplatesClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})
	if client == nil {
		t.Fatal("NewPermissionsTemplatesClient() expected non-nil client")
	}
}
