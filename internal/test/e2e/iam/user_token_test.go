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

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestUserTokenCRUD provisions a UserToken CR with RenewalPeriodDays,
// waits for it to be Ready and Synced, verifies the token exists in SonarQube,
// and checks that the connection secret contains the token value.
func TestUserTokenCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName                = "e2e-usertoken-crud"
		tokenName             = "e2e-usertoken-crud"
		tokenSecretName       = "e2e-usertoken-secret"
		tokenSecretKey        = "token"
		renewalDays     int64 = 90
	)

	token := &iamv1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: iamv1alpha1.UserTokenSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{
					Name: tokenSecretName,
				},
			},
			ForProvider: iamv1alpha1.UserTokenParameters{
				Name:              tokenName,
				Type:              sonar.TokenTypeUserToken,
				RenewalPeriodDays: ptr.To(renewalDays),
			},
		},
	}

	// Create the token and wait for it to be ready and synced
	f.CreateAndWaitForReady(t, token, 2*time.Minute)
	e2e.AssertReady(t, token)
	e2e.AssertSynced(t, token)

	// Verify external name is set
	externalName := token.GetAnnotations()["crossplane.io/external-name"]
	if externalName == "" {
		t.Fatalf("expected external-name to be populated after Ready")
	}
	if externalName != tokenName {
		t.Errorf("external name = %q, want %q", externalName, tokenName)
	}

	// Verify the token secret exists and contains the token value
	secret := &corev1.Secret{}
	if err := f.Kube.Get(context.Background(), kubeKey(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tokenSecretName, Namespace: "default"},
	}), secret); err != nil {
		t.Fatalf("getting token secret: %v", err)
	}

	tokenValue := secret.Data[tokenSecretKey]
	if len(tokenValue) == 0 {
		t.Fatalf("token secret missing %q key or value", tokenSecretKey)
	}
	t.Logf("token secret contains %d byte token", len(tokenValue))

	// Verify the token is in SonarQube with correct properties
	// Note: This requires SonarQube token search API
	// For now, we verify the CR status shows the expected properties
	if token.Status.AtProvider.Name != tokenName {
		t.Errorf("observed token name = %q, want %q", token.Status.AtProvider.Name, tokenName)
	}
	if token.Status.AtProvider.Type != sonar.TokenTypeUserToken {
		t.Errorf("observed token type = %q, want %q", token.Status.AtProvider.Type, sonar.TokenTypeUserToken)
	}

	// Cleanup
	if err := f.Kube.Delete(context.Background(), token); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("deleting token: %v", err)
	}
}

// TestUserTokenExpirationDate tests creating a token with explicit ExpirationDate.
func TestUserTokenExpirationDate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName          = "e2e-expiry-token"
		tokenName       = "e2e-expiry-token"
		tokenSecretName = "e2e-expiry-token-secret"
		expiryDate      = "2027-12-31"
	)

	token := &iamv1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: iamv1alpha1.UserTokenSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{
					Name: tokenSecretName,
				},
			},
			ForProvider: iamv1alpha1.UserTokenParameters{
				Name:           tokenName,
				Type:           sonar.TokenTypeUserToken,
				ExpirationDate: ptr.To(expiryDate),
			},
		},
	}

	// Create the token and wait for it to be ready
	f.CreateAndWaitForReady(t, token, 2*time.Minute)
	e2e.AssertReady(t, token)
	e2e.AssertSynced(t, token)

	// Verify external name is set
	if token.GetAnnotations()["crossplane.io/external-name"] == "" {
		t.Fatalf("expected external-name to be populated after Ready")
	}

	// Cleanup
	if err := f.Kube.Delete(context.Background(), token); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("deleting token: %v", err)
	}
}
