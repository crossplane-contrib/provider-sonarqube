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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// waitForGitHubConfiguration polls SonarQube until its GitHub configuration
// satisfies check, failing the test after two minutes.
func waitForGitHubConfiguration(t *testing.T, f *e2e.Framework, desc string, check func(*sonar.DopTranslationGithubConfiguration) bool) {
	t.Helper()

	var got *sonar.DopTranslationGithubConfiguration
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var fetchErr error
		got, fetchErr = f.FetchGitHubConfiguration(ctx)
		if fetchErr != nil {
			return false, fetchErr
		}
		return check(got), nil
	}); err != nil {
		t.Fatalf("%s: GitHub configuration = %+v: %v", desc, got, err)
	}
}

// TestGitHubConfigurationCRUD creates a GitHubConfiguration, verifies
// SonarQube reports it, updates it from the spec, reverts a change made
// directly in SonarQube, then deletes it.
//
// SonarQube supports a single GitHub configuration per instance, so this is
// the only test managing one. The configuration stays disabled: an enabled
// one makes SonarQube validate the fake GitHub App against GitHub.
func TestGitHubConfigurationCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName           = "e2e-githubconfiguration-crud"
		appID            = "123456"
		clientID         = "Iv1.e2etest123456"
		apiURL           = "https://api.github.com/"
		webURL           = "https://github.com/"
		clientSecretName = "e2e-githubconfiguration-crud-client-secret"
		privateKeyName   = "e2e-githubconfiguration-crud-private-key"
		connSecretName   = "e2e-githubconfiguration-crud-connection"
	)

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: clientSecretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"clientSecret": "e2e-fake-client-secret"},
	}
	privateKey := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: privateKeyName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"privateKey": "e2e-fake-private-key"},
	}

	for _, s := range []*corev1.Secret{clientSecret, privateKey} {
		if err := f.Kube.Create(context.Background(), s); err != nil && !apierrors.IsAlreadyExists(err) {
			t.Fatalf("creating secret %s: %v", s.Name, err)
		}
		t.Cleanup(func() {
			_ = f.Kube.Delete(context.Background(), s)
		})
	}

	configuration := &integrationv1alpha1.GitHubConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.GitHubConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.GitHubConfigurationParameters{
				Enabled:       new(false),
				ApplicationID: appID,
				ClientID:      clientID,
				ClientSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: clientSecretName},
					Key:                  "clientSecret",
				},
				PrivateKeyRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: privateKeyName},
					Key:                  "privateKey",
				},
				APIURL:               apiURL,
				WebURL:               webURL,
				AllowedOrganizations: []string{"e2e-org-a"},
				ProvisioningType:     integrationv1alpha1.GitHubProvisioningTypeJIT,
				SynchronizeGroups:    new(false),
			},
		},
	}

	f.CreateAndWaitForReady(t, configuration, 2*time.Minute)
	e2e.AssertReady(t, configuration)
	e2e.AssertSynced(t, configuration)

	externalName := meta.GetExternalName(configuration)
	if externalName == "" {
		t.Fatal("external name (SonarQube GitHub configuration ID) is empty after resource became Ready")
	}

	got, err := f.FetchGitHubConfiguration(context.Background())
	if err != nil {
		t.Fatalf("fetching GitHub configuration: %v", err)
	}
	if got == nil {
		t.Fatal("GitHub configuration not found in SonarQube")
	}
	if got.Id != externalName {
		t.Errorf("GitHub configuration ID = %q, want %q", got.Id, externalName)
	}
	if got.Enabled {
		t.Error("GitHub configuration Enabled = true, want false")
	}
	if got.ApplicationId != appID {
		t.Errorf("GitHub configuration ApplicationId = %q, want %q", got.ApplicationId, appID)
	}
	if got.ApiUrl != apiURL || got.WebUrl != webURL {
		t.Errorf("GitHub configuration URLs = %q, %q, want %q, %q", got.ApiUrl, got.WebUrl, apiURL, webURL)
	}
	if got.ProvisioningType != integrationv1alpha1.GitHubProvisioningTypeJIT {
		t.Errorf("GitHub configuration ProvisioningType = %q, want %q", got.ProvisioningType, integrationv1alpha1.GitHubProvisioningTypeJIT)
	}
	if !helpers.AreStringSlicesEqual(got.AllowedOrganizations, []string{"e2e-org-a"}) {
		t.Errorf("GitHub configuration AllowedOrganizations = %v, want [e2e-org-a]", got.AllowedOrganizations)
	}

	// Update the spec: SonarQube must follow.
	updatedOrganizations := []string{"e2e-org-a", "e2e-org-b"}
	if err = f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching GitHubConfiguration CR: %v", err)
	}
	configuration.Spec.ForProvider.AllowedOrganizations = updatedOrganizations
	configuration.Spec.ForProvider.SynchronizeGroups = new(true)
	f.Update(t, configuration)

	waitForGitHubConfiguration(t, f, "after spec update", func(c *sonar.DopTranslationGithubConfiguration) bool {
		return c != nil && c.SynchronizeGroups && helpers.AreStringSlicesEqual(c.AllowedOrganizations, updatedOrganizations)
	})

	// Change the organizations directly in SonarQube to simulate drift.
	_, resp, err := f.Sonar.V2.DopTranslation.UpdateGithubConfiguration(context.Background(), externalName, &sonar.DopTranslationGithubConfigurationUpdateOptions{
		AllowedOrganizations: &sonar.UpdateFieldListStringV2{Value: []string{"e2e-org-drift"}, Defined: true},
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("updating GitHub configuration in SonarQube: %v", err)
	}

	// Touch an annotation to trigger a reconcile rather than waiting for
	// the provider poll interval.
	if err = f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching GitHubConfiguration CR: %v", err)
	}
	annotations := configuration.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	configuration.SetAnnotations(annotations)
	f.Update(t, configuration)

	waitForGitHubConfiguration(t, f, "after drift", func(c *sonar.DopTranslationGithubConfiguration) bool {
		return c != nil && helpers.AreStringSlicesEqual(c.AllowedOrganizations, updatedOrganizations)
	})

	// Delete the resource: the GitHub configuration must be removed.
	f.Delete(t, configuration)
	if err = f.WaitForDeletion(context.Background(), configuration, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for deletion of GitHubConfiguration CR: %v", err)
	}

	got, err = f.FetchGitHubConfiguration(context.Background())
	if err != nil {
		t.Fatalf("fetching GitHub configuration after deletion: %v", err)
	}
	if got != nil {
		t.Errorf("GitHub configuration %q still exists in SonarQube after deletion", got.Id)
	}
}
