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

// waitForSlackClientID polls SonarQube until the Slack configuration with the
// given ID has the wanted client ID, failing the test after two minutes.
func waitForSlackClientID(t *testing.T, f *e2e.Framework, id, want string) {
	t.Helper()

	var last *sonar.IntegrationsIntegrationConfiguration
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, err := f.FindSlackIntegrationByID(ctx, id)
		if err != nil {
			return false, err
		}
		last = got
		return got != nil && got.ClientId == want, nil
	}); err != nil {
		t.Fatalf("waiting for Slack configuration %q to have client id %q: %v; last observed: %+v", id, want, err, last)
	}
}

// TestSlackIntegrationLifecycle creates the Slack configuration, verifies
// SonarQube reports it, updates it from the spec, reverts a change made
// directly in SonarQube, and deletes it.
//
// SonarQube holds a single Slack configuration per instance, so the whole
// lifecycle runs in one test.
func TestSlackIntegrationLifecycle(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName          = "e2e-slack-integration"
		secretName      = "e2e-slack-integration-credentials"
		connSecretName  = "e2e-slack-integration-connection"
		clientID        = "1234567890.1111111111"
		updatedClientID = "1234567890.2222222222"
		driftedClientID = "1234567890.3333333333"
	)

	credentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"clientSecret":  "e2e-fake-slack-client-secret",
			"signingSecret": "e2e-fake-slack-signing-secret",
		},
	}
	if err := f.Kube.Create(context.Background(), credentials); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating Slack credentials secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), credentials)
	})

	slack := &integrationv1alpha1.SlackIntegration{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.SlackIntegrationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.SlackIntegrationParameters{
				ClientID: clientID,
				ClientSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
					Key:                  "clientSecret",
				},
				SigningSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
					Key:                  "signingSecret",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, slack, 2*time.Minute)
	e2e.AssertReady(t, slack)
	e2e.AssertSynced(t, slack)

	id := meta.GetExternalName(slack)
	if id == "" || id == crName {
		t.Fatalf("external name = %q, want the SonarQube-assigned configuration id", id)
	}

	waitForSlackClientID(t, f, id, clientID)

	// Update the spec and wait for SonarQube to reflect it.
	if err := f.Kube.Get(context.Background(), kubeKey(slack), slack); err != nil {
		t.Fatalf("re-fetching SlackIntegration: %v", err)
	}
	slack.Spec.ForProvider.ClientID = updatedClientID
	f.Update(t, slack)

	waitForSlackClientID(t, f, id, updatedClientID)

	// Change the client id directly in SonarQube to simulate drift.
	_, resp, err := f.Sonar.V2.Integrations.UpdateIntegrationConfiguration(context.Background(), id, &sonar.IntegrationsIntegrationConfigurationPatchOptions{
		ClientId: driftedClientID,
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("updating Slack configuration in SonarQube: %v", err)
	}

	// Touch an annotation to trigger a reconcile rather than waiting for
	// the provider poll interval.
	if err := f.Kube.Get(context.Background(), kubeKey(slack), slack); err != nil {
		t.Fatalf("re-fetching SlackIntegration: %v", err)
	}
	annotations := slack.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	slack.SetAnnotations(annotations)
	f.Update(t, slack)

	waitForSlackClientID(t, f, id, updatedClientID)

	// Delete the resource and verify SonarQube no longer has the
	// configuration.
	f.Delete(t, slack)
	if err := f.WaitForDeletion(context.Background(), slack, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for SlackIntegration deletion: %v", err)
	}

	got, err := f.FindSlackIntegrationByID(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching Slack configuration after deletion: %v", err)
	}
	if got != nil {
		t.Errorf("Slack configuration %q still exists after deletion: %+v", id, got)
	}
}
