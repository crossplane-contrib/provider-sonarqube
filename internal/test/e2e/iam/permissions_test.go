//go:build e2e

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

package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestPermissionsCRUDGroup creates a Group CR and a Permissions CR that
// assigns two global permissions to it. It waits for both to be Ready, then
// verifies the SonarQube API reports the expected permission set.
func TestPermissionsCRUDGroup(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		groupCRName = "e2e-perms-group-crud"
		groupName   = "e2e-perms-group-crud"
		permsCRName = "e2e-perms-group-crud-perms"
	)
	wantPerms := []string{"scan", "provisioning"}

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: groupCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.GroupParameters{
				Name: groupName,
			},
		},
	}

	f.CreateAndWaitForReady(t, group, 2*time.Minute)
	e2e.AssertReady(t, group)

	perms := &iamv1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{Name: permsCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PermissionsParameters{
				GroupName:   ptr.To(groupName),
				Permissions: wantPerms,
			},
		},
	}

	f.CreateAndWaitForReady(t, perms, 2*time.Minute)
	e2e.AssertReady(t, perms)
	e2e.AssertSynced(t, perms)

	gotPerms, err := f.GroupPermissions(groupName)
	if err != nil {
		t.Fatalf("fetching group permissions: %v", err)
	}
	if !equalStringSets(gotPerms, wantPerms) {
		t.Errorf("group permissions = %v, want %v (order ignored)", gotPerms, wantPerms)
	}
}

// TestPermissionsCRUDUser creates a User CR and a Permissions CR that
// assigns a global permission to it. It waits for both to be Ready, then
// verifies the SonarQube API reports the expected permission.
func TestPermissionsCRUDUser(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		userCRName  = "e2e-perms-user-crud"
		userLogin   = "e2e-perms-user-crud"
		userName    = "E2E Perms User CRUD"
		secretName  = "e2e-perms-user-pwd"
		secretKey   = "password"
		permsCRName = "e2e-perms-user-crud-perms"
	)
	wantPerms := []string{"scan"}

	pwSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-perms-pw-123!"},
	}
	if err := f.Kube.Create(context.Background(), pwSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating password secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), pwSecret)
	})

	user := &iamv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: userCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.UserSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.UserParameters{
				Login:     userLogin,
				Name:      userName,
				Local:     ptr.To(true),
				Anonymize: ptr.To(true),
				PasswordSecretRef: &xpv1.SecretKeySelector{
					Key: secretKey,
					SecretReference: xpv1.SecretReference{
						Name:      secretName,
						Namespace: f.Namespace,
					},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, user, 2*time.Minute)
	e2e.AssertReady(t, user)

	perms := &iamv1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{Name: permsCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PermissionsParameters{
				Login:       ptr.To(userLogin),
				Permissions: wantPerms,
			},
		},
	}

	f.CreateAndWaitForReady(t, perms, 2*time.Minute)
	e2e.AssertReady(t, perms)
	e2e.AssertSynced(t, perms)

	gotPerms, err := f.UserPermissions(userLogin)
	if err != nil {
		t.Fatalf("fetching user permissions: %v", err)
	}
	if !equalStringSets(gotPerms, wantPerms) {
		t.Errorf("user permissions = %v, want %v (order ignored)", gotPerms, wantPerms)
	}
}

// TestPermissionsUpdateGroup creates a Permissions CR with a single
// permission, then updates it to two permissions and verifies the change
// is reflected in SonarQube.
func TestPermissionsUpdateGroup(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		groupCRName = "e2e-perms-update-group"
		groupName   = "e2e-perms-update-group"
		permsCRName = "e2e-perms-update-perms"
	)
	initialPerms := []string{"scan"}
	updatedPerms := []string{"scan", "provisioning"}

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: groupCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.GroupParameters{
				Name: groupName,
			},
		},
	}

	f.CreateAndWaitForReady(t, group, 2*time.Minute)
	e2e.AssertReady(t, group)

	perms := &iamv1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{Name: permsCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PermissionsParameters{
				GroupName:   ptr.To(groupName),
				Permissions: initialPerms,
			},
		},
	}

	f.CreateAndWaitForReady(t, perms, 2*time.Minute)
	e2e.AssertReady(t, perms)
	e2e.AssertSynced(t, perms)

	// Re-fetch to obtain the latest resourceVersion before updating.
	if err := f.Kube.Get(context.Background(), client.ObjectKeyFromObject(perms), perms); err != nil {
		t.Fatalf("re-fetching permissions CR: %v", err)
	}
	perms.Spec.ForProvider.Permissions = updatedPerms
	f.Update(t, perms)

	// WaitForReady returns immediately when conditions are already True from
	// the previous reconcile; poll GroupPermissions directly until the
	// controller has applied the spec change.
	var gotPerms []string
	pollCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if pollErr := wait.PollUntilContextTimeout(pollCtx, 2*time.Second, 2*time.Minute, false, func(_ context.Context) (bool, error) {
		var fetchErr error
		gotPerms, fetchErr = f.GroupPermissions(groupName)
		if fetchErr != nil {
			return false, fetchErr
		}
		return equalStringSets(gotPerms, updatedPerms), nil
	}); pollErr != nil {
		t.Fatalf("group permissions did not update within timeout: last=%v want=%v: %v", gotPerms, updatedPerms, pollErr)
	}
}

// TestPermissionsProjectScoped creates a Project CR, a Group CR, and a
// Permissions CR scoped to that project. It waits for all three to be Ready,
// then verifies SonarQube reports the expected project-level permissions.
func TestPermissionsProjectScoped(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		projectCRName = "e2e-perms-project"
		projectKey    = "e2e-perms-project"
		groupCRName   = "e2e-perms-project-group"
		groupName     = "e2e-perms-project-group"
		permsCRName   = "e2e-perms-project-perms"
	)
	wantPerms := []string{"user", "codeviewer"}

	project := &instancev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: projectCRName, Namespace: f.Namespace},
		Spec: instancev1alpha1.ProjectSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ProjectParameters{
				Key:        projectKey,
				Name:       projectKey,
				Visibility: ptr.To("private"),
			},
		},
	}

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: groupCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.GroupParameters{
				Name: groupName,
			},
		},
	}

	f.CreateAndWaitForReady(t, project, 2*time.Minute)
	e2e.AssertReady(t, project)
	f.CreateAndWaitForReady(t, group, 2*time.Minute)
	e2e.AssertReady(t, group)

	perms := &iamv1alpha1.Permissions{
		ObjectMeta: metav1.ObjectMeta{Name: permsCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.PermissionsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PermissionsParameters{
				GroupName:   ptr.To(groupName),
				ProjectKey:  ptr.To(projectKey),
				Permissions: wantPerms,
			},
		},
	}

	f.CreateAndWaitForReady(t, perms, 2*time.Minute)
	e2e.AssertReady(t, perms)
	e2e.AssertSynced(t, perms)

	// Verify project-scoped permissions via the SonarQube API.
	res, resp, err := f.Sonar.Permissions.Groups(&sonar.PermissionsGroupsOptions{
		Query:      groupName,
		ProjectKey: projectKey,
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("querying project-scoped group permissions: %v", err)
	}

	var gotPerms []string
	for i := range res.Groups {
		if res.Groups[i].Name == groupName {
			gotPerms = make([]string, len(res.Groups[i].Permissions))
			copy(gotPerms, res.Groups[i].Permissions)
			break
		}
	}
	if !equalStringSets(gotPerms, wantPerms) {
		t.Errorf("project-scoped group permissions = %v, want %v (order ignored)", gotPerms, wantPerms)
	}
}
