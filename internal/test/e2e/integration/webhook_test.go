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
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestWebhookGlobalCRUD creates a global Webhook resource, waits for Ready,
// then verifies SonarQube reports a matching webhook with the expected name
// and URL.
func TestWebhookGlobalCRUD(t *testing.T) {
	t.Parallel()

	framework := e2e.New(t)

	const (
		crName      = "e2e-webhook-global-crud"
		webhookName = "e2e-global-webhook"
		webhookURL  = "https://ci.example.com/e2e-hook"
	)

	webhook := &integrationv1alpha1.Webhook{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: integrationv1alpha1.WebhookSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: framework.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.WebhookParameters{
				Name: webhookName,
				URL:  webhookURL,
			},
		},
	}

	framework.CreateAndWaitForReady(t, webhook, 2*time.Minute)
	e2e.AssertReady(t, webhook)
	e2e.AssertSynced(t, webhook)

	// External name is the SonarQube-assigned webhook key; verify it is set.
	externalName := meta.GetExternalName(webhook)
	if externalName == "" {
		t.Fatal("external name (SonarQube webhook key) is empty after resource became Ready")
	}

	// Verify the webhook exists in SonarQube.
	got, err := framework.FindGlobalWebhookByKey(externalName)
	if err != nil {
		t.Fatalf("fetching webhook from SonarQube: %v", err)
	}
	if got == nil {
		t.Fatalf("webhook %q not found in SonarQube", externalName)
	}

	if got.Name != webhookName {
		t.Errorf("webhook Name = %q, want %q", got.Name, webhookName)
	}

	if got.URL != webhookURL {
		t.Errorf("webhook URL = %q, want %q", got.URL, webhookURL)
	}
}
