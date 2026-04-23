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

//go:build e2e

package iam_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestUserCRUD provisions a password Secret, then creates a local User
// referencing it. Waits for Ready and verifies the user exists in
// SonarQube with the expected login, name and email.
func TestUserCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName     = "e2e-user-crud"
		userLogin  = "e2e-user-crud"
		userName   = "E2E User CRUD"
		userEmail  = "e2e-user-crud@example.com"
		secretName = "e2e-user-crud-password"
		secretKey  = "password"
	)

	pwSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-password-123!"},
	}
	if err := f.Kube.Create(context.Background(), pwSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating password secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), pwSecret)
	})

	user := &iamv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: iamv1alpha1.UserSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.UserParameters{
				Login: userLogin,
				Name:  userName,
				Email: stringPtr(userEmail),
				Local: boolPtr(true),
				PasswordSecretRef: &xpv1.SecretKeySelector{
					Key: secretKey,
					SecretReference: xpv1.SecretReference{
						Name:      secretName,
						Namespace: "default",
					},
				},
				// Anonymise on delete so re-creating with the same login
				// after a previous run leaves no observable trace.
				Anonymize: boolPtr(true),
			},
		},
	}

	f.CreateAndWaitForReady(t, user, 2*time.Minute)
	e2e.AssertReady(t, user)
	e2e.AssertSynced(t, user)

	id := user.GetAnnotations()["crossplane.io/external-name"]
	if id == "" {
		t.Fatalf("expected external-name to be populated after Ready")
	}

	got, err := f.FetchUser(id)
	if err != nil {
		t.Fatalf("fetching user: %v", err)
	}
	if got == nil {
		t.Fatalf("user %q (id=%s) not found in SonarQube", userLogin, id)
	}
	if got.Login != userLogin {
		t.Errorf("user login = %q, want %q", got.Login, userLogin)
	}
	if got.Name != userName {
		t.Errorf("user name = %q, want %q", got.Name, userName)
	}
	if got.Email != userEmail {
		t.Errorf("user email = %q, want %q", got.Email, userEmail)
	}
	if !got.Local {
		t.Errorf("user local = false, want true")
	}
}
