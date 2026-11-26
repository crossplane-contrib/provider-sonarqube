//go:build e2e && enterprise

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
	"os"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// EnvOpenAIAPIKey names the environment variable holding an OpenAI API key.
// SonarQube tests the connection before saving an LLM provider, so the test
// needs a real key and skips itself when it is unset.
const EnvOpenAIAPIKey = "SONARQUBE_E2E_OPENAI_API_KEY"

// TestLLMProviderCRUD creates an OpenAI LLMProvider, waits for Ready, then
// verifies SonarQube reports a provider with the expected type and label
// under the external name. It maps no AI capability, since SonarQube cannot
// unmap one and would then refuse to delete the provider. Requires
// SONARQUBE_LICENSE_KEY and SONARQUBE_E2E_OPENAI_API_KEY; skips otherwise.
func TestLLMProviderCRUD(t *testing.T) {
	t.Parallel()

	apiKey := os.Getenv(EnvOpenAIAPIKey)
	if apiKey == "" {
		t.Skipf("%s is not set; skipping test that requires LLM credentials", EnvOpenAIAPIKey)
	}

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName         = "e2e-llmprovider-crud"
		label          = "e2e OpenAI"
		apiKeySecret   = "e2e-llmprovider-crud-api-key"
		connSecretName = "e2e-llmprovider-crud-connection"
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: apiKeySecret, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"apiKey": apiKey},
	}
	if err := f.Kube.Create(context.Background(), secret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating secret %s: %v", secret.Name, err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), secret)
	})

	llmProvider := &integrationv1alpha1.LLMProvider{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.LLMProviderSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.LLMProviderParameters{
				Provider: "OPENAI",
				Label:    label,
				ConfigurationSecretRefs: map[string]xpv1.LocalSecretKeySelector{
					"apiKey": {LocalSecretReference: xpv1.LocalSecretReference{Name: apiKeySecret}, Key: "apiKey"},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, llmProvider, 2*time.Minute)
	e2e.AssertReady(t, llmProvider)
	e2e.AssertSynced(t, llmProvider)

	id := meta.GetExternalName(llmProvider)
	got, err := f.FindLLMProviderByID(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching LLM provider %q: %v", id, err)
	}
	if got == nil {
		t.Fatalf("LLM provider %q not found in SonarQube", id)
	}
	if got.Provider != "OPENAI" {
		t.Errorf("LLM provider type = %q, want OPENAI", got.Provider)
	}
	if got.Label != label {
		t.Errorf("LLM provider label = %q, want %q", got.Label, label)
	}
}
