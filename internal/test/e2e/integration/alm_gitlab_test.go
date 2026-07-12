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

package integration_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestALMGitLabCRUD creates an ALMGitLab resource, waits for Ready, then
// verifies SonarQube reports a matching GitLab ALM definition with the
// expected key and URL.
func TestALMGitLabCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName         = "e2e-almgitlab-crud"
		almKey         = "e2e-almgitlab-crud"
		almURL         = "https://gitlab.example.com"
		secretName     = "e2e-almgitlab-crud-token"
		secretKey      = "token"
		connSecretName = "e2e-almgitlab-crud-connection"
	)

	patSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-fake-gitlab-pat"},
	}
	if err := f.Kube.Create(context.Background(), patSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating PAT secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), patSecret)
	})

	alm := &integrationv1alpha1.ALMGitLab{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.ALMGitLabSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.ALMGitLabParameters{
				ALMCommonParameters: integrationv1alpha1.ALMCommonParameters{
					Key: almKey,
					URL: almURL,
					PersonalAccessTokenSecretRef: &xpv1.LocalSecretKeySelector{
						LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
						Key:                  secretKey,
					},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, alm, 2*time.Minute)
	e2e.AssertReady(t, alm)
	e2e.AssertSynced(t, alm)
	e2e.AssertExternalName(t, alm, almKey)

	got, err := f.FindALMGitLabDefinitionByKey(almKey)
	if err != nil {
		t.Fatalf("fetching GitLab ALM definition: %v", err)
	}
	if got == nil {
		t.Fatalf("GitLab ALM definition %q not found in SonarQube", almKey)
	}
	if got.URL != almURL {
		t.Errorf("ALM URL = %q, want %q", got.URL, almURL)
	}
}
