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

package integration_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestALMGitHubCRUD creates an ALMGitHub resource, waits for Ready, then
// verifies SonarQube reports a matching GitHub ALM definition with the
// expected key, URL, AppID and ClientID.
func TestALMGitHubCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName          = "e2e-almgithub-crud"
		almKey          = "e2e-almgithub-crud"
		almURL          = "https://api.github.com"
		almAppID        = "123456"
		almClientID     = "Iv1.e2etest123456"
		clientSecretRef = "e2e-almgithub-crud-client-secret"
		privateKeyRef   = "e2e-almgithub-crud-private-key"
		connSecretName  = "e2e-almgithub-crud-connection"
	)

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: clientSecretRef, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"clientSecret": "e2e-fake-client-secret"},
	}
	privateKey := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: privateKeyRef, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"privateKey": "e2e-fake-private-key"},
	}

	for _, s := range []*corev1.Secret{clientSecret, privateKey} {
		if err := f.Kube.Create(context.Background(), s); err != nil && !apierrors.IsAlreadyExists(err) {
			t.Fatalf("creating secret %s: %v", s.Name, err)
		}
		s := s
		t.Cleanup(func() {
			_ = f.Kube.Delete(context.Background(), s)
		})
	}

	alm := &integrationv1alpha1.ALMGitHub{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: integrationv1alpha1.ALMGitHubSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.ALMGitHubParameters{
				Key:      almKey,
				URL:      almURL,
				AppID:    almAppID,
				ClientID: almClientID,
				ClientSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: clientSecretRef},
					Key:                  "clientSecret",
				},
				PrivateKeyRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: privateKeyRef},
					Key:                  "privateKey",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, alm, 2*time.Minute)
	e2e.AssertReady(t, alm)
	e2e.AssertSynced(t, alm)
	e2e.AssertExternalName(t, alm, almKey)

	got, err := f.FindALMGitHubDefinitionByKey(almKey)
	if err != nil {
		t.Fatalf("fetching GitHub ALM definition: %v", err)
	}
	if got == nil {
		t.Fatalf("GitHub ALM definition %q not found in SonarQube", almKey)
	}
	if got.URL != almURL {
		t.Errorf("ALM URL = %q, want %q", got.URL, almURL)
	}
	if got.AppID != almAppID {
		t.Errorf("ALM AppID = %q, want %q", got.AppID, almAppID)
	}
	if got.ClientID != almClientID {
		t.Errorf("ALM ClientID = %q, want %q", got.ClientID, almClientID)
	}
}
