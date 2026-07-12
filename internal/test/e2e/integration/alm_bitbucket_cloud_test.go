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

// TestALMBitbucketCloudCRUD creates an ALMBitbucketCloud resource, waits
// for Ready, then verifies SonarQube reports a matching Bitbucket Cloud
// ALM definition with the expected key, clientId and workspace.
func TestALMBitbucketCloudCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName         = "e2e-almbitbucketcloud-crud"
		almKey         = "e2e-almbitbucketcloud-crud"
		almClientID    = "e2e-fake-client-id"
		almWorkspace   = "e2e-fake-workspace"
		secretName     = "e2e-almbitbucketcloud-crud-secret"
		secretKey      = "clientSecret"
		connSecretName = "e2e-almbitbucketcloud-crud-connection"
	)

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-fake-bitbucket-cloud-client-secret"},
	}
	if err := f.Kube.Create(context.Background(), clientSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating client secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), clientSecret)
	})

	alm := &integrationv1alpha1.ALMBitbucketCloud{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.ALMBitbucketCloudSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.ALMBitbucketCloudParameters{
				Key:       almKey,
				ClientID:  almClientID,
				Workspace: almWorkspace,
				ClientSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
					Key:                  secretKey,
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, alm, 2*time.Minute)
	e2e.AssertReady(t, alm)
	e2e.AssertSynced(t, alm)
	e2e.AssertExternalName(t, alm, almKey)

	got, err := f.FindALMBitbucketCloudDefinitionByKey(context.Background(), almKey)
	if err != nil {
		t.Fatalf("fetching Bitbucket Cloud ALM definition: %v", err)
	}
	if got == nil {
		t.Fatalf("Bitbucket Cloud ALM definition %q not found in SonarQube", almKey)
	}
	if got.ClientID != almClientID {
		t.Errorf("ALM clientId = %q, want %q", got.ClientID, almClientID)
	}
	if got.Workspace != almWorkspace {
		t.Errorf("ALM workspace = %q, want %q", got.Workspace, almWorkspace)
	}
}
