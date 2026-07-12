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

// Package permissionstemplate handles permissions template
// observation and management.
package permissionstemplate

import (
	"context"
	"errors"
	"sync"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	pkgerrors "github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// permissionsTemplateNotFound is the error message when template is not found.
	permissionsTemplateNotFound = "PermissionsTemplate not found"
)

// errPermissionsTemplateNotFound is returned when a template is not found.
var errPermissionsTemplateNotFound = errors.New(permissionsTemplateNotFound)

// Observe observes the external resource and returns an ExternalObservation.
// It is used to determine if the resource exists, is up to date,
// and late initialize the spec if needed.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	permissionsTemplate, ok := mg.(*v1alpha1.PermissionsTemplate)
	if !ok {
		return managed.ExternalObservation{}, pkgerrors.New(errNotPermissionsTemplate)
	}

	externalName := meta.GetExternalName(permissionsTemplate)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	template, isDefault, err := c.observePermissionsTemplate(ctx, &externalName, nil)
	if err != nil {
		// If the PermissionsTemplate is not found, we consider that it doesn't exist and we don't return an error. Any other error is returned.
		if errors.Is(err, errPermissionsTemplateNotFound) {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}

		return managed.ExternalObservation{}, pkgerrors.Wrap(err, "failed to observe PermissionsTemplate")
	}

	permissionsTemplate.SetConditions(xpv1.Available())

	groupsObservations, usersObservations, err := c.observeTemplatePermissions(ctx, template.ID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	permissionsTemplate.Status.AtProvider.GroupPermissions = groupsObservations
	permissionsTemplate.Status.AtProvider.UserPermissions = usersObservations

	iam.UpdatePermissionsTemplateObservation(&permissionsTemplate.Status.AtProvider, &template, isDefault)

	former := permissionsTemplate.Spec.ForProvider.DeepCopy()
	iam.LateInitializePermissionsTemplate(&permissionsTemplate.Spec.ForProvider, &permissionsTemplate.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        iam.IsPermissionsTemplateUpToDate(&permissionsTemplate.Spec.ForProvider, &permissionsTemplate.Status.AtProvider),
		ResourceLateInitialized: iam.IsPermissionsTemplateLateInitialized(former, &permissionsTemplate.Spec.ForProvider),
	}, nil
}

// observeTemplatePermissions collects group and user permissions
// concurrently for a template.
func (c *external) observeTemplatePermissions(ctx context.Context, templateID string) ([]v1alpha1.PermissionsTemplateGroupObservation, []v1alpha1.PermissionsTemplateUserObservation, error) {
	var (
		groupsObservations []v1alpha1.PermissionsTemplateGroupObservation
		usersObservations  []v1alpha1.PermissionsTemplateUserObservation
		waitGroup          sync.WaitGroup
		resultMutex        sync.Mutex
		aggregatedErrors   []error
	)

	runTask := func(fetch func() error, wrapMessage string) {
		waitGroup.Go(func() {
			err := fetch()
			if err == nil {
				return
			}

			resultMutex.Lock()
			defer resultMutex.Unlock()

			aggregatedErrors = append(aggregatedErrors, pkgerrors.Wrap(err, wrapMessage))
		})
	}

	runTask(func() error {
		observations, err := c.observePermissionsTemplateGroups(ctx, templateID)
		if err != nil {
			return err
		}

		resultMutex.Lock()
		groupsObservations = observations
		resultMutex.Unlock()

		return nil
	}, "failed to observe PermissionsTemplate groups permissions")

	runTask(func() error {
		observations, err := c.observePermissionsTemplateUsers(ctx, templateID)
		if err != nil {
			return err
		}

		resultMutex.Lock()
		usersObservations = observations
		resultMutex.Unlock()

		return nil
	}, "failed to observe PermissionsTemplate users permissions")

	waitGroup.Wait()

	err := errors.Join(aggregatedErrors...)
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to observe PermissionsTemplate permissions")
	}

	return groupsObservations, usersObservations, nil
}

// observePermissionsTemplate resolves a template either by ID
// (preferred) or by name.
// ID lookup scans paginated templates without a query because SonarQube
// query filtering is name-oriented.
func (c *external) observePermissionsTemplate(ctx context.Context, templateID, templateName *string) (sonar.PermissionTemplate, bool, error) {
	if templateID == nil && templateName == nil {
		return sonar.PermissionTemplate{}, false, pkgerrors.New("either id or name must be provided to search for PermissionsTemplate")
	}

	if templateID != nil {
		return c.searchPermissionsTemplate(ctx, "", func(template sonar.PermissionTemplate) bool {
			return template.ID == *templateID
		})
	}

	return c.searchPermissionsTemplate(ctx, *templateName, func(template sonar.PermissionTemplate) bool {
		return template.Name == *templateName
	})
}

// searchPermissionsTemplate scans template pages using the provided query
// and returns the first matching template.
func (c *external) searchPermissionsTemplate(ctx context.Context, query string, match func(template sonar.PermissionTemplate) bool) (sonar.PermissionTemplate, bool, error) {
	const maxPageSize = int64(500)

	defaultTemplatesIds := make(map[string]struct{})

	for page := int64(1); ; page++ {
		templateSearchOptions := iam.GeneratePermissionsTemplateSearchOptions(query, &sonar.PaginationArgs{Page: page, PageSize: maxPageSize})
		templates, resp, err := c.client.SearchTemplates(ctx, templateSearchOptions) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return sonar.PermissionTemplate{}, false, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate")
		}

		for _, defaultTemplate := range templates.DefaultTemplates {
			defaultTemplatesIds[defaultTemplate.TemplateID] = struct{}{}
		}

		for _, template := range templates.PermissionTemplates {
			if !match(template) {
				continue
			}

			_, isDefault := defaultTemplatesIds[template.ID]

			return template, isDefault, nil
		}

		if len(templates.PermissionTemplates) == 0 || len(templates.PermissionTemplates) < int(maxPageSize) {
			return sonar.PermissionTemplate{}, false, errPermissionsTemplateNotFound
		}
	}
}

// getTemplateSearchString chooses the best search key for template lookup.
func getTemplateSearchString(templateID, templateName *string) (string, error) {
	if templateID == nil && templateName == nil {
		return "", pkgerrors.New("either id or name must be provided to search for PermissionsTemplate")
	}

	if templateID != nil {
		return *templateID, nil
	}

	return *templateName, nil
}

// findMatchingTemplate returns the first template matching the
// provided id or name.
func findMatchingTemplate(templates []sonar.PermissionTemplate, defaultTemplatesIDs map[string]struct{}, templateID, templateName *string) (sonar.PermissionTemplate, bool, bool) {
	for _, template := range templates {
		if (templateID != nil && template.ID != *templateID) || (templateName != nil && template.Name != *templateName) {
			continue
		}

		_, isDefault := defaultTemplatesIDs[template.ID]

		return template, isDefault, true
	}

	return sonar.PermissionTemplate{}, false, false
}

// observeTemplatePermissionsPage paginates through template permission
// endpoints until exhaustion.
func (c *external) observeTemplatePermissionsPage(fetchPage func(page int64) (int, error)) error {
	const maxPageSize = int64(100)

	for page := int64(1); ; page++ {
		count, err := fetchPage(page)
		if err != nil {
			return err
		}

		if count == 0 || count < int(maxPageSize) {
			break
		}
	}

	return nil
}

// observePermissionsTemplateGroups retrieves all group permissions
// associated with a PermissionsTemplate.
//
//nolint:dupl // User and group pagination handlers intentionally mirror each other with different API endpoints.
func (c *external) observePermissionsTemplateGroups(ctx context.Context, templateID string) ([]v1alpha1.PermissionsTemplateGroupObservation, error) {
	const maxPageSize = int64(100)

	groupsObservations := []v1alpha1.PermissionsTemplateGroupObservation{}

	err := c.observeTemplatePermissionsPage(func(page int64) (int, error) {
		options := iam.GeneratePermissionsTemplateGroupsSearchOptions(templateID, &sonar.PaginationArgs{Page: page, PageSize: maxPageSize})
		groups, resp, err := c.client.TemplateGroups(ctx, options) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return 0, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate groups")
		}

		groupsObservations = append(groupsObservations, iam.GeneratePermissionsTemplateGroupObservations(&groups.Groups)...)

		return len(groups.Groups), nil
	})
	if err != nil {
		return nil, err
	}

	return groupsObservations, nil
}

// observePermissionsTemplateUsers retrieves all user permissions
// associated with a PermissionsTemplate.
//
//nolint:dupl // This mirrors group pagination behavior with a different API endpoint and observation type.
func (c *external) observePermissionsTemplateUsers(ctx context.Context, templateID string) ([]v1alpha1.PermissionsTemplateUserObservation, error) {
	const maxPageSize = int64(100)

	usersObservations := []v1alpha1.PermissionsTemplateUserObservation{}

	err := c.observeTemplatePermissionsPage(func(page int64) (int, error) {
		options := iam.GeneratePermissionsTemplateUsersSearchOptions(templateID, &sonar.PaginationArgs{Page: page, PageSize: maxPageSize})
		users, resp, err := c.client.TemplateUsers(ctx, options) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return 0, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate users")
		}

		usersObservations = append(usersObservations, iam.GeneratePermissionsTemplateUserObservations(&users.Users)...)

		return len(users.Users), nil
	})
	if err != nil {
		return nil, err
	}

	return usersObservations, nil
}
