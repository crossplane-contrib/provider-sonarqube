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
	"sort"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	baseTemplate := func() *v1alpha1.PermissionsTemplate {
		return withExternalName(&v1alpha1.PermissionsTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "template-a"},
			Spec: v1alpha1.PermissionsTemplateSpec{
				ForProvider: v1alpha1.PermissionsTemplateParameters{
					Name:               "template-a",
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

	cases := map[string]struct {
		client  *fake.MockPermissionsTemplatesClient
		mg      resource.Managed
		wantErr string
	}{
		"WrongTypeReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      &notPermissionsTemplate{},
			wantErr: errNotPermissionsTemplate,
		},
		"MissingExternalNameReturnsError": {
			client:  &fake.MockPermissionsTemplatesClient{},
			mg:      newPermissionsTemplate("template-a"),
			wantErr: "external name is not set",
		},
		"BaseUpdateFailureReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				UpdateTemplateFn: func(opt *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("update failed")
				},
			},
			mg:      baseTemplate(),
			wantErr: "failed to reconcile PermissionsTemplate",
		},
		"DefaultSetFailureReturnsWrappedError": {
			client: &fake.MockPermissionsTemplatesClient{
				SetDefaultFn: func(opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error) {
					return nil, errors.New("default failed")
				},
			},
			mg: func() resource.Managed {
				template := baseTemplate()
				template.Spec.ForProvider.Default = ptr.To(true)

				return template
			}(),
			wantErr: "failed to reconcile PermissionsTemplate",
		},
		"SuccessfulUpdateReconcilesPermissions": {
			client: &fake.MockPermissionsTemplatesClient{
				UpdateTemplateFn: func(opt *sonar.PermissionsUpdateTemplateOptions) (*sonar.PermissionsUpdateTemplate, *http.Response, error) {
					return &sonar.PermissionsUpdateTemplate{}, mockHTTPResponse(), nil
				},
				SetDefaultFn: func(opt *sonar.PermissionsSetDefaultTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				AddGroupFn: func(opt *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				RemoveGroupFn: func(opt *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				AddUserFn: func(opt *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				RemoveUserFn: func(opt *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				AddProjectCreatorFn: func(opt *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
				RemoveProjectCreatorFn: func(opt *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			mg: func() resource.Managed {
				template := baseTemplate()
				template.Spec.ForProvider.Default = ptr.To(true)
				template.Spec.ForProvider.CreatorPermissions = stringSlice("admin")
				template.Spec.ForProvider.GroupPermissions = &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: stringSlice("admin")}}
				template.Spec.ForProvider.UserPermissions = &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: stringSlice("admin")}}
				template.Status.AtProvider.Default = false
				template.Status.AtProvider.CreatorPermissions = []string{"scan"}
				template.Status.AtProvider.GroupPermissions = []v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}}
				template.Status.AtProvider.UserPermissions = []v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}}

				return template
			}(),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			_, err := e.Update(context.Background(), tc.mg)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Update() unexpected error: %v", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Update() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestComputePermissionsDiff(t *testing.T) {
	t.Parallel()

	e := &external{}
	cases := map[string]struct {
		spec       []string
		obs        []string
		wantAdd    []string
		wantRemove []string
	}{
		"NoChanges":    {spec: []string{"scan"}, obs: []string{"scan"}},
		"AddOnly":      {spec: []string{"scan", "admin"}, obs: []string{"scan"}, wantAdd: []string{"admin"}},
		"RemoveOnly":   {spec: []string{"scan"}, obs: []string{"scan", "admin"}, wantRemove: []string{"admin"}},
		"AddAndRemove": {spec: []string{"scan", "issueadmin"}, obs: []string{"scan", "admin"}, wantAdd: []string{"issueadmin"}, wantRemove: []string{"admin"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotAdd, gotRemove := e.computePermissionsDiff(tc.spec, tc.obs)
			if diff := cmp.Diff(tc.wantAdd, gotAdd); diff != "" {
				t.Fatalf("computePermissionsDiff() add mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantRemove, gotRemove); diff != "" {
				t.Fatalf("computePermissionsDiff() remove mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyTemplatePermissions(t *testing.T) {
	t.Parallel()

	e := &external{}
	calls := make([]string, 0, 4)

	err := e.applyTemplatePermissions(
		[]string{"add-a", "add-b"},
		[]string{"remove-a"},
		func(permission string) error {
			calls = append(calls, "add:"+permission)

			return nil
		},
		func(permission string) error {
			calls = append(calls, "remove:"+permission)

			return nil
		},
	)
	if err != nil {
		t.Fatalf("applyTemplatePermissions() unexpected error: %v", err)
	}

	sort.Strings(calls)

	if diff := cmp.Diff([]string{"add:add-a", "add:add-b", "remove:remove-a"}, calls); diff != "" {
		t.Fatalf("applyTemplatePermissions() calls mismatch (-want +got):\n%s", diff)
	}

	err = e.applyTemplatePermissions(
		[]string{"add-a"},
		[]string{"remove-a"},
		func(permission string) error { return errors.New("add failed") },
		func(permission string) error { return errors.New("remove failed") },
	)
	if err == nil || !strings.Contains(err.Error(), "add failed") || !strings.Contains(err.Error(), "remove failed") {
		t.Fatalf("applyTemplatePermissions() aggregated error = %v", err)
	}
}

func TestBaseFieldsAndDefaultHelpers(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockPermissionsTemplatesClient{}}
	template := withExternalName(newPermissionsTemplate("template-a"), "template-id")

	template.Spec.ForProvider.Description = ptr.To("desc")
	template.Spec.ForProvider.ProjectKeyPattern = ptr.To("proj-.*")
	template.Spec.ForProvider.Default = ptr.To(true)
	template.Status.AtProvider.Name = "template-a"
	template.Status.AtProvider.Description = "desc"
	template.Status.AtProvider.ProjectKeyPattern = "proj-.*"
	template.Status.AtProvider.Default = true

	err := e.updateTemplateBaseFields("template-id", template)
	if err != nil {
		t.Fatalf("updateTemplateBaseFields() unexpected error: %v", err)
	}

	err = e.setTemplateAsDefaultIfNeeded("template-id", template)
	if err != nil {
		t.Fatalf("setTemplateAsDefaultIfNeeded() unexpected error: %v", err)
	}

	template.Spec.ForProvider.Default = ptr.To(false)
	template.Status.AtProvider.Default = false

	err = e.setTemplateAsDefaultIfNeeded("template-id", template)
	if err != nil {
		t.Fatalf("setTemplateAsDefaultIfNeeded() unexpected error when not requested: %v", err)
	}
}

func TestGroupUserCreatorReconciliation(t *testing.T) {
	t.Parallel()

	var groupAdds, groupRemoves, userAdds, userRemoves, creatorAdds, creatorRemoves []string

	e := &external{client: &fake.MockPermissionsTemplatesClient{
		AddGroupFn: func(opt *sonar.PermissionsAddGroupToTemplateOptions) (*http.Response, error) {
			groupAdds = append(groupAdds, opt.GroupName+":"+opt.Permission)

			return mockHTTPResponse(), nil
		},
		RemoveGroupFn: func(opt *sonar.PermissionsRemoveGroupFromTemplateOptions) (*http.Response, error) {
			groupRemoves = append(groupRemoves, opt.GroupName+":"+opt.Permission)

			return mockHTTPResponse(), nil
		},
		AddUserFn: func(opt *sonar.PermissionsAddUserToTemplateOptions) (*http.Response, error) {
			userAdds = append(userAdds, opt.Login+":"+opt.Permission)

			return mockHTTPResponse(), nil
		},
		RemoveUserFn: func(opt *sonar.PermissionsRemoveUserFromTemplateOptions) (*http.Response, error) {
			userRemoves = append(userRemoves, opt.Login+":"+opt.Permission)

			return mockHTTPResponse(), nil
		},
		AddProjectCreatorFn: func(opt *sonar.PermissionsAddProjectCreatorToTemplateOptions) (*http.Response, error) {
			creatorAdds = append(creatorAdds, opt.Permission)

			return mockHTTPResponse(), nil
		},
		RemoveProjectCreatorFn: func(opt *sonar.PermissionsRemoveProjectCreatorFromTemplateOptions) (*http.Response, error) {
			creatorRemoves = append(creatorRemoves, opt.Permission)

			return mockHTTPResponse(), nil
		},
	}}

	groupsSpec := &[]v1alpha1.PermissionsTemplateGroupParameters{{Name: "devs", Permissions: stringSlice("admin")}}
	groupsObs := &[]v1alpha1.PermissionsTemplateGroupObservation{{Name: "devs", Permissions: []string{"scan"}}}

	err := e.updatePermissionsTemplateGroups("template-id", groupsSpec, groupsObs)
	if err != nil {
		t.Fatalf("updatePermissionsTemplateGroups() unexpected error: %v", err)
	}

	usersSpec := &[]v1alpha1.PermissionsTemplateUserParameters{{Login: "alice", Permissions: stringSlice("admin")}}
	usersObs := &[]v1alpha1.PermissionsTemplateUserObservation{{Login: "alice", Permissions: []string{"scan"}}}

	err = e.updatePermissionsTemplateUsers("template-id", usersSpec, usersObs)
	if err != nil {
		t.Fatalf("updatePermissionsTemplateUsers() unexpected error: %v", err)
	}

	creatorSpec := &[]string{"admin"}
	creatorObs := []string{"scan"}

	err = e.updatePermissionsTemplateCreator("template-id", creatorSpec, creatorObs)
	if err != nil {
		t.Fatalf("updatePermissionsTemplateCreator() unexpected error: %v", err)
	}

	if diff := cmp.Diff([]string{"devs:admin"}, groupAdds); diff != "" {
		t.Fatalf("updatePermissionsTemplateGroups() add calls mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"devs:scan"}, groupRemoves); diff != "" {
		t.Fatalf("updatePermissionsTemplateGroups() remove calls mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"alice:admin"}, userAdds); diff != "" {
		t.Fatalf("updatePermissionsTemplateUsers() add calls mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"alice:scan"}, userRemoves); diff != "" {
		t.Fatalf("updatePermissionsTemplateUsers() remove calls mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"admin"}, creatorAdds); diff != "" {
		t.Fatalf("updatePermissionsTemplateCreator() add calls mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"scan"}, creatorRemoves); diff != "" {
		t.Fatalf("updatePermissionsTemplateCreator() remove calls mismatch (-want +got):\n%s", diff)
	}
}
