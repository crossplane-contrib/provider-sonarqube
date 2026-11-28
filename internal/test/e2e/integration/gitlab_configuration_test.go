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
	"slices"
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

// waitForGitLabConfiguration polls SonarQube until the GitLab configuration
// with the given ID satisfies check, failing the test after two minutes.
func waitForGitLabConfiguration(t *testing.T, f *e2e.Framework, id, desc string, check func(*sonar.DopTranslationGitlabConfiguration) bool) {
	t.Helper()

	var last *sonar.DopTranslationGitlabConfiguration
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, err := f.FetchGitLabConfiguration(ctx, id)
		if err != nil {
			return false, err
		}
		last = got
		return check(got), nil
	}); err != nil {
		t.Fatalf("waiting for GitLab configuration %q (%s): %v; last observed: %+v", id, desc, err, last)
	}
}

// TestGitLabConfigurationLifecycle creates the GitLab authentication
// configuration, verifies SonarQube reports it, updates it from the spec,
// reverts a change made directly in SonarQube, and deletes it.
//
// SonarQube holds a single GitLab configuration per instance, so the whole
// lifecycle runs in one test. The configuration is disabled: SonarQube
// validates an enabled configuration against GitLab when it is read.
func TestGitLabConfigurationLifecycle(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName         = "e2e-gitlab-configuration"
		secretName     = "e2e-gitlab-configuration-credentials"
		secretKey      = "secret"
		connSecretName = "e2e-gitlab-configuration-connection"
		applicationID  = "e2e-gitlab-application-id"
		gitlabURL      = "https://gitlab.example.com"
	)

	credentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-fake-gitlab-application-secret"},
	}
	if err := f.Kube.Create(context.Background(), credentials); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating application secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), credentials)
	})

	configuration := &integrationv1alpha1.GitLabConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.GitLabConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.GitLabConfigurationParameters{
				Enabled:       new(false),
				ApplicationID: applicationID,
				URL:           gitlabURL,
				ApplicationSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
					Key:                  secretKey,
				},
				SynchronizeGroups:  new(true),
				AllowedGroups:      []string{"e2e-group-a"},
				ProvisioningType:   integrationv1alpha1.GitLabProvisioningTypeJIT,
				AllowUsersToSignUp: new(false),
			},
		},
	}

	f.CreateAndWaitForReady(t, configuration, 2*time.Minute)
	e2e.AssertReady(t, configuration)
	e2e.AssertSynced(t, configuration)

	id := meta.GetExternalName(configuration)
	if id == "" || id == crName {
		t.Fatalf("external name = %q, want the SonarQube-assigned configuration id", id)
	}

	got, err := f.FetchGitLabConfiguration(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching GitLab configuration: %v", err)
	}
	if got == nil {
		t.Fatalf("GitLab configuration %q not found in SonarQube", id)
	}
	if got.Enabled || got.ApplicationId != applicationID || got.Url != gitlabURL || !got.SynchronizeGroups ||
		got.AllowUsersToSignUp || got.ProvisioningType != integrationv1alpha1.GitLabProvisioningTypeJIT ||
		!slices.Equal(got.AllowedGroups, []string{"e2e-group-a"}) {
		t.Errorf("GitLab configuration = %+v, want it to match the spec", got)
	}

	// Update the spec and wait for SonarQube to reflect it.
	if err := f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching GitLabConfiguration: %v", err)
	}
	configuration.Spec.ForProvider.AllowedGroups = []string{"e2e-group-a", "e2e-group-b"}
	configuration.Spec.ForProvider.AllowUsersToSignUp = new(true)
	f.Update(t, configuration)

	waitForGitLabConfiguration(t, f, id, "after spec update", func(got *sonar.DopTranslationGitlabConfiguration) bool {
		return got != nil && got.AllowUsersToSignUp &&
			helpers.AreStringSlicesEqual(got.AllowedGroups, []string{"e2e-group-a", "e2e-group-b"})
	})

	// Change the application id directly in SonarQube to simulate drift.
	_, resp, err := f.Sonar.V2.DopTranslation.UpdateGitlabConfiguration(context.Background(), id, &sonar.DopTranslationGitlabConfigurationUpdateOptions{
		ApplicationId: "e2e-drifted-application-id",
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("updating GitLab configuration in SonarQube: %v", err)
	}

	// Touch an annotation to trigger a reconcile rather than waiting for
	// the provider poll interval.
	if err := f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching GitLabConfiguration: %v", err)
	}
	annotations := configuration.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	configuration.SetAnnotations(annotations)
	f.Update(t, configuration)

	waitForGitLabConfiguration(t, f, id, "after drift", func(got *sonar.DopTranslationGitlabConfiguration) bool {
		return got != nil && got.ApplicationId == applicationID
	})

	// Delete the resource and verify SonarQube no longer has a configuration.
	f.Delete(t, configuration)
	if err := f.WaitForDeletion(context.Background(), configuration, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for GitLabConfiguration deletion: %v", err)
	}

	got, err = f.FetchGitLabConfiguration(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching GitLab configuration after deletion: %v", err)
	}
	if got != nil {
		t.Errorf("GitLab configuration %q still exists after deletion: %+v", id, got)
	}
}
