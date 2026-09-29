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
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestObserveAfterCreateFindsTemplate tests that the Observe following a
// Create finds the new template instead of the cached index read before
// it, so that the template is not created twice, and that K templates
// share one index scan.
func TestObserveAfterCreateFindsTemplate(t *testing.T) {
	t.Parallel()

	var templates []sonar.PermissionTemplate

	searchCalls, createCalls := 0, 0
	inner := &fake.MockPermissionsTemplatesClient{
		SearchFn: func(*sonar.PermissionsSearchTemplatesOptions) (*sonar.PermissionsSearchTemplates, *http.Response, error) {
			searchCalls++

			return &sonar.PermissionsSearchTemplates{PermissionTemplates: templates}, mockHTTPResponse(), nil
		},
		CreateFn: func(opt *sonar.PermissionsCreateTemplateOptions) (*sonar.PermissionsCreateTemplate, *http.Response, error) {
			createCalls++

			created := sonar.PermissionTemplate{ID: "id-" + opt.Name, Name: opt.Name}
			templates = append(templates, created)

			return &sonar.PermissionsCreateTemplate{PermissionTemplate: sonar.PermissionsTemplateBasic{ID: created.ID, Name: created.Name}}, mockHTTPResponse(), nil
		},
		TemplateGroupsFn: func(*sonar.PermissionsTemplateGroupsOptions) (*sonar.PermissionsTemplateGroups, *http.Response, error) {
			return &sonar.PermissionsTemplateGroups{Paging: singlePage}, mockHTTPResponse(), nil
		},
		TemplateUsersFn: func(*sonar.PermissionsTemplateUsersOptions) (*sonar.PermissionsTemplateUsers, *http.Response, error) {
			return &sonar.PermissionsTemplateUsers{Paging: singlePage}, mockHTTPResponse(), nil
		},
	}
	scoped := cachetest.NewScoped(t)
	e := &external{client: iam.NewCachedPermissionsTemplatesClient(inner, scoped), cache: scoped}

	// Crossplane defaults the external name to metadata.name before Create.
	mr := withExternalName(newPermissionsTemplate(templateNameA), templateNameA)

	observation, err := e.Observe(context.Background(), mr)
	if err != nil || observation.ResourceExists {
		t.Fatalf("Observe() before Create = %+v, %v, want not existing", observation, err)
	}

	_, err = e.Create(context.Background(), mr)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	observation, err = e.Observe(context.Background(), mr)
	if err != nil || !observation.ResourceExists {
		t.Fatalf("Observe() after Create = %+v, %v, want existing", observation, err)
	}

	for range 3 {
		other := withExternalName(newPermissionsTemplate("other"), "id-"+templateNameA)

		_, err = e.Observe(context.Background(), other)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}
	}

	if createCalls != 1 || searchCalls != 2 {
		t.Errorf("CreateTemplate called %d times and SearchTemplates %d times, want 1 and 2", createCalls, searchCalls)
	}
}
