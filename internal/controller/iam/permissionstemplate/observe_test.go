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

package permissionstemplate

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// permissionsTemplateTestID is a test permissions template ID.
const permissionsTemplateTestID = "template-id"

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	baseTemplate := func() *v1alpha1.PermissionsTemplate {
		return withExternalName(&v1alpha1.PermissionsTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: templateNameA},
			Spec: v1alpha1.PermissionsTemplateSpec{
				ForProvider: v1alpha1.PermissionsTemplateParameters{
					Name:               templateNameA,
					Description:        ptr.To("desc"),
					ProjectKeyPattern:  ptr.To("proj-.*"),
					Default:            ptr.To(false),
					CreatorPermissions: stringSlice("scan"),
					GroupPermissions: &[]v1alpha1.PermissionsTemplateGroupParameters{{
						Name:        "devs",
						Permissions: stringSlice("scan"),
					}},
					UserPermissions: &[]v1alpha1.PermissionsTemplateUserParameters{{
						Login:       "alice",
						Permissions: stringSlice("scan"),
					}},
				},
			},
		}, "template-id")
	}

	lateInitTemplate := func() *v1alpha1.PermissionsTemplate {
		return withExternalName(&v1alpha1.PermissionsTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "template-b"},
			Spec: v1alpha1.PermissionsTemplateSpec{
				ForProvider: v1alpha1.PermissionsTemplateParameters{Name: "template-b", Default: ptr.To(false)},
			},
		}, "template-id-b")
	}

	cases := map[string]struct {
		client  *fake.MockPermissionsTemplatesClient
		mg      resource.Managed
		want    managed.ExternalObservation
		wantErr string
	}{
		"WrongTypeReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      &notPermissionsTemplate{},
			want:    managed.ExternalObservation{},
			wantErr: errNotPermissionsTemplate,
		},
		"MissingExternalNameReturnsNotExists": {
			client: &fake.MockPermissionsTemplatesClient{},
			mg: &v1alpha1.PermissionsTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: templateNameA},
			},
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"SearchErrorReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("search failed")
				},
			},
			mg:      withExternalName(newPermissionsTemplate(templateNameA), templateNameA),
			want:    managed.ExternalObservation{},
			wantErr: "failed to observe PermissionsTemplate",
		},
		"NotFoundReturnsNotExists": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return &sonar.PermissionsSearchTemplates{PermissionTemplates: []sonar.PermissionTemplate{}}, mockHTTPResponse(), nil
				},
			},
			mg:   withExternalName(newPermissionsTemplate(templateNameA), templateNameA),
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"SuccessfulObserveWithoutLateInit": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return &sonar.PermissionsSearchTemplates{
						PermissionTemplates: []sonar.PermissionTemplate{{
							ID:                "template-id",
							Name:              templateNameA,
							Description:       "desc",
							ProjectKeyPattern: "proj-.*",
							Permissions:       []sonar.TemplatePermission{{Key: "scan", WithProjectCreator: true}},
						}},
					}, mockHTTPResponse(), nil
				},
				TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
					return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{{Name: "devs", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
				},
				TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
					return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{{Login: "alice", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
				},
			},
			mg:   baseTemplate(),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false},
		},
		"ObservedEmptyPermissionSubjectsAreIgnored": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return &sonar.PermissionsSearchTemplates{
						PermissionTemplates: []sonar.PermissionTemplate{{
							ID:                "template-id-empty",
							Name:              "template-empty",
							Description:       "desc",
							ProjectKeyPattern: "proj-.*",
							Permissions:       []sonar.TemplatePermission{{Key: "scan", WithProjectCreator: true}},
						}},
					}, mockHTTPResponse(), nil
				},
				TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
					return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{{Name: "devs", Permissions: []string{}}, {Name: "devs", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
				},
				TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
					return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{{Login: "alice", Permissions: []string{}}, {Login: "alice", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
				},
			},
			mg: func() resource.Managed {
				template := withExternalName(newPermissionsTemplate("template-empty"), "template-id-empty")
				template.Spec.ForProvider.Description = ptr.To("desc")
				template.Spec.ForProvider.ProjectKeyPattern = ptr.To("proj-.*")
				template.Spec.ForProvider.Default = ptr.To(false)
				template.Spec.ForProvider.CreatorPermissions = stringSlice("scan")
				template.Spec.ForProvider.GroupPermissions = &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: &[]string{"scan"}}}
				template.Spec.ForProvider.UserPermissions = &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: &[]string{"scan"}}}

				return template
			}(),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false},
		},
		"SuccessfulObserveWithLateInit": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return &sonar.PermissionsSearchTemplates{
						PermissionTemplates: []sonar.PermissionTemplate{{
							ID:                "template-id-b",
							Name:              "template-b",
							Description:       "generated-desc",
							ProjectKeyPattern: "generated-pattern",
							Permissions:       []sonar.TemplatePermission{{Key: "scan", WithProjectCreator: true}, {Key: "admin", WithProjectCreator: true}},
						}},
					}, mockHTTPResponse(), nil
				},
				TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
					return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{}}, mockHTTPResponse(), nil
				},
				TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
					return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{}}, mockHTTPResponse(), nil
				},
			},
			mg:   lateInitTemplate(),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
		},
		"GroupsOrUsersErrorReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
					return &sonar.PermissionsSearchTemplates{
						PermissionTemplates: []sonar.PermissionTemplate{{ID: "template-id", Name: templateNameA}},
					}, mockHTTPResponse(), nil
				},
				TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("groups failed")
				},
				TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
					return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{}}, mockHTTPResponse(), nil
				},
			},
			mg:      withExternalName(newPermissionsTemplate(templateNameA), "template-id"),
			want:    managed.ExternalObservation{},
			wantErr: "failed to observe PermissionsTemplate permissions",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(context.Background(), tc.mg)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Observe() unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Observe() error = %v, want substring %q", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObserveLateInitializesDefault tests Observe late initializes default.
func TestObserveLateInitializesDefault(t *testing.T) {
	t.Parallel()

	template := withExternalName(&v1alpha1.PermissionsTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "template-default"},
		Spec: v1alpha1.PermissionsTemplateSpec{
			ForProvider: v1alpha1.PermissionsTemplateParameters{Name: "template-default"},
		},
	}, "template-default-id")

	e := &external{client: &fake.MockPermissionsTemplatesClient{
		SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
			return &sonar.PermissionsSearchTemplates{
				PermissionTemplates: []sonar.PermissionTemplate{{
					ID:                "template-default-id",
					Name:              "template-default",
					Description:       "desc",
					ProjectKeyPattern: "proj-.*",
					Permissions:       []sonar.TemplatePermission{{Key: "scan", WithProjectCreator: true}},
				}},
			}, mockHTTPResponse(), nil
		},
		TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
			return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{}}, mockHTTPResponse(), nil
		},
		TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
			return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{}}, mockHTTPResponse(), nil
		},
	}}

	got, err := e.Observe(context.Background(), template)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !got.ResourceExists || !got.ResourceLateInitialized || !got.ResourceUpToDate {
		t.Fatalf("Observe() got %+v, want exists, late init, and up-to-date", got)
	}

	if template.Spec.ForProvider.Default == nil || *template.Spec.ForProvider.Default {
		t.Fatalf("Observe() default = %v, want false after late init", template.Spec.ForProvider.Default)
	}
}

// TestGetTemplateSearchString tests the getTemplateSearchString function.
func TestGetTemplateSearchString(t *testing.T) {
	t.Parallel()

	err := func() error {
		_, err := getTemplateSearchString(nil, nil)

		return err
	}()
	if err == nil {
		t.Fatal("getTemplateSearchString(nil, nil) expected error")
	}

	got, err := getTemplateSearchString(ptr.To(permissionsTemplateTestID), ptr.To("template-name"))
	if err != nil || got != permissionsTemplateTestID {
		t.Fatalf("getTemplateSearchString() got %q, err=%v", got, err)
	}

	got, err = getTemplateSearchString(nil, ptr.To("template-name"))
	if err != nil || got != "template-name" {
		t.Fatalf("getTemplateSearchString() got %q, err=%v", got, err)
	}
}

// TestObserveReturnsExistsWhenExternalNameIsTemplateID tests Observe
// returns exists when external name is template ID.
func TestObserveReturnsExistsWhenExternalNameIsTemplateID(t *testing.T) {
	t.Parallel()

	template := withExternalName(newPermissionsTemplate(templateNameA), permissionsTemplateTestID)
	searchCalls := 0

	e := &external{client: &fake.MockPermissionsTemplatesClient{
		SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
			searchCalls++

			if opt.Query != "" {
				t.Fatalf("SearchFn query = %q, want empty query", opt.Query)
			}

			return &sonar.PermissionsSearchTemplates{PermissionTemplates: []sonar.PermissionTemplate{{ID: permissionsTemplateTestID, Name: templateNameA}}}, mockHTTPResponse(), nil
		},
		TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
			return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{}}, mockHTTPResponse(), nil
		},
		TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
			return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{}}, mockHTTPResponse(), nil
		},
	}}

	got, err := e.Observe(context.Background(), template)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !got.ResourceExists {
		t.Fatal("Observe() ResourceExists = false, want true")
	}

	if searchCalls == 0 {
		t.Fatal("SearchFn was not called")
	}
}

// TestFindMatchingTemplate tests the findMatchingTemplate function.
func TestFindMatchingTemplate(t *testing.T) {
	t.Parallel()

	templates := []sonar.PermissionTemplate{{ID: permissionsTemplateTestID, Name: templateNameA}, {ID: "template-2", Name: "template-b"}}
	defaultTemplates := map[string]struct{}{permissionsTemplateTestID: {}}

	got, isDefault, found := findMatchingTemplate(templates, defaultTemplates, ptr.To(permissionsTemplateTestID), nil)
	if !found || !isDefault || got.ID != permissionsTemplateTestID {
		t.Fatalf("findMatchingTemplate() got=%+v isDefault=%v found=%v", got, isDefault, found)
	}

	_, _, found = findMatchingTemplate(templates, defaultTemplates, ptr.To("missing"), nil)
	if found {
		t.Fatal("findMatchingTemplate() expected no match")
	}
}

// TestObserveTemplatePermissionsPage tests observeTemplatePermissionsPage.
func TestObserveTemplatePermissionsPage(t *testing.T) {
	t.Parallel()

	calls := make([]int64, 0, 2)
	e := &external{}

	err := e.observeTemplatePermissionsPage(func(page int64) (int, error) {
		calls = append(calls, page)
		if page == 1 {
			return 500, nil
		}

		return 1, nil
	})
	if err != nil {
		t.Fatalf("observeTemplatePermissionsPage() unexpected error: %v", err)
	}

	if diff := cmp.Diff([]int64{1, 2}, calls); diff != "" {
		t.Fatalf("observeTemplatePermissionsPage() calls mismatch (-want +got):\n%s", diff)
	}

	expectedErr := errors.New("page failed")

	err = e.observeTemplatePermissionsPage(func(page int64) (int, error) {
		return 0, expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("observeTemplatePermissionsPage() error = %v, want %v", err, expectedErr)
	}
}

// TestObservePermissionsTemplate tests observing a PermissionsTemplate.
func TestObservePermissionsTemplate(t *testing.T) {
	t.Parallel()

	idLookupClient := &fake.MockPermissionsTemplatesClient{
		SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
			if opt.Query != "" {
				t.Fatalf("SearchFn query = %q, want empty query", opt.Query)
			}

			if opt.Page == 1 {
				return &sonar.PermissionsSearchTemplates{PermissionTemplates: []sonar.PermissionTemplate{{ID: permissionsTemplateTestID, Name: templateNameA}}, DefaultTemplates: []sonar.DefaultTemplate{{TemplateID: permissionsTemplateTestID}}}, mockHTTPResponse(), nil
			}

			return &sonar.PermissionsSearchTemplates{PermissionTemplates: []sonar.PermissionTemplate{}}, mockHTTPResponse(), nil
		},
	}

	e := &external{client: idLookupClient}

	got, isDefault, err := e.observePermissionsTemplate(ptr.To(permissionsTemplateTestID), nil)
	if err != nil || got.ID != permissionsTemplateTestID || !isDefault {
		t.Fatalf("observePermissionsTemplate() got=%+v isDefault=%v err=%v", got, isDefault, err)
	}

	nameLookupClient := &fake.MockPermissionsTemplatesClient{
		SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
			if opt.Query != templateNameA {
				t.Fatalf("SearchFn query = %q, want %q", opt.Query, templateNameA)
			}

			return &sonar.PermissionsSearchTemplates{PermissionTemplates: []sonar.PermissionTemplate{{ID: permissionsTemplateTestID, Name: templateNameA}}, DefaultTemplates: []sonar.DefaultTemplate{{TemplateID: permissionsTemplateTestID}}}, mockHTTPResponse(), nil
		},
	}

	e = &external{client: nameLookupClient}

	got, isDefault, err = e.observePermissionsTemplate(nil, ptr.To(templateNameA))
	if err != nil || got.ID != permissionsTemplateTestID || !isDefault {
		t.Fatalf("observePermissionsTemplate() name-only got=%+v isDefault=%v err=%v", got, isDefault, err)
	}

	_, _, err = e.observePermissionsTemplate(nil, nil)
	if err == nil {
		t.Fatal("observePermissionsTemplate(nil, nil) expected error")
	}

	errExternalClient := &external{client: &fake.MockPermissionsTemplatesClient{SearchFn: func(opt *sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
		return nil, mockHTTPResponse(), errors.New("search failed")
	}}}

	_, _, err = errExternalClient.observePermissionsTemplate(ptr.To(permissionsTemplateTestID), nil)
	if err == nil || !strings.Contains(err.Error(), "failed to search for PermissionsTemplate") {
		t.Fatalf("observePermissionsTemplate() error = %v", err)
	}
}

// TestObserveTemplatePermissionPagination tests permission pagination.
func TestObserveTemplatePermissionPagination(t *testing.T) {
	t.Parallel()

	groupsCalls := 0
	groupsClient := &fake.MockPermissionsTemplatesClient{
		TemplateGroupsFn: func(opt *sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
			groupsCalls++

			if opt.Page == 1 {
				return &sonar.PermissionsTemplateGroups{Groups: make([]sonar.TemplateGroup, 500)}, mockHTTPResponse(), nil
			}

			return &sonar.PermissionsTemplateGroups{Groups: []sonar.TemplateGroup{{Name: "devs", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
		},
	}

	groups, err := (&external{client: groupsClient}).observePermissionsTemplateGroups(permissionsTemplateTestID)
	if err != nil {
		t.Fatalf("observePermissionsTemplateGroups() unexpected error: %v", err)
	}

	if groupsCalls != 2 {
		t.Fatalf("observePermissionsTemplateGroups() calls = %d, want 2", groupsCalls)
	}

	if len(groups) != 501 {
		t.Fatalf("observePermissionsTemplateGroups() len = %d, want 501", len(groups))
	}

	usersCalls := 0
	usersClient := &fake.MockPermissionsTemplatesClient{
		TemplateUsersFn: func(opt *sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
			usersCalls++

			if opt.Page == 1 {
				return &sonar.PermissionsTemplateUsers{Users: make([]sonar.TemplateUser, 500)}, mockHTTPResponse(), nil
			}

			return &sonar.PermissionsTemplateUsers{Users: []sonar.TemplateUser{{Login: "alice", Permissions: []string{"scan"}}}}, mockHTTPResponse(), nil
		},
	}

	users, err := (&external{client: usersClient}).observePermissionsTemplateUsers(permissionsTemplateTestID)
	if err != nil {
		t.Fatalf("observePermissionsTemplateUsers() unexpected error: %v", err)
	}

	if usersCalls != 2 {
		t.Fatalf("observePermissionsTemplateUsers() calls = %d, want 2", usersCalls)
	}

	if len(users) != 501 {
		t.Fatalf("observePermissionsTemplateUsers() len = %d, want 501", len(users))
	}
}
