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
// (preferred) or by name from the index of every template, and reports
// whether it is a default template. It returns
// errPermissionsTemplateNotFound when no template matches.
func (c *external) observePermissionsTemplate(ctx context.Context, templateID, templateName *string) (sonar.PermissionTemplate, bool, error) {
	if templateID == nil && templateName == nil {
		return sonar.PermissionTemplate{}, false, pkgerrors.New("either id or name must be provided to search for PermissionsTemplate")
	}

	index, err := iam.PermissionTemplatesIndex(ctx, c.client, c.cache)
	if err != nil {
		return sonar.PermissionTemplate{}, false, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate")
	}

	var (
		template  sonar.PermissionTemplate
		isDefault bool
		found     bool
	)

	if templateID != nil {
		template, isDefault, found = index.FindByID(*templateID)
	} else {
		template, isDefault, found = index.FindByName(*templateName)
	}

	if !found {
		return sonar.PermissionTemplate{}, false, errPermissionsTemplateNotFound
	}

	return template, isDefault, nil
}

// observePermissionsTemplateGroups retrieves all group permissions
// associated with a PermissionsTemplate.
func (c *external) observePermissionsTemplateGroups(ctx context.Context, templateID string) ([]v1alpha1.PermissionsTemplateGroupObservation, error) {
	groups, err := iam.PermissionTemplateGroups(ctx, c.client, c.cache, templateID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate groups")
	}

	return iam.GeneratePermissionsTemplateGroupObservations(&groups), nil
}

// observePermissionsTemplateUsers retrieves all user permissions
// associated with a PermissionsTemplate.
func (c *external) observePermissionsTemplateUsers(ctx context.Context, templateID string) ([]v1alpha1.PermissionsTemplateUserObservation, error) {
	users, err := iam.PermissionTemplateUsers(ctx, c.client, c.cache, templateID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search for PermissionsTemplate users")
	}

	return iam.GeneratePermissionsTemplateUserObservations(&users), nil
}
