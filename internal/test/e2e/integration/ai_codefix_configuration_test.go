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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestAICodeFixConfigurationDisabled creates an AICodeFixConfiguration
// disabling AI CodeFix, waits for Ready, and verifies SonarQube reports AI
// CodeFix as disabled. Enabling it needs an LLM provider mapped to
// AI_CODEFIX, which needs real LLM credentials, so it is not covered here.
// Requires SONARQUBE_LICENSE_KEY; skips otherwise.
func TestAICodeFixConfigurationDisabled(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	configuration := &integrationv1alpha1.AICodeFixConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-aicodefix", Namespace: f.Namespace},
		Spec: integrationv1alpha1.AICodeFixConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: integrationv1alpha1.AICodeFixConfigurationParameters{
				Enablement: integrationv1alpha1.AICodeFixDisabled,
			},
		},
	}

	f.CreateAndWaitForReady(t, configuration, 2*time.Minute)
	e2e.AssertReady(t, configuration)
	e2e.AssertSynced(t, configuration)

	got, err := f.FetchAICodeFixEnablement(context.Background())
	if err != nil {
		t.Fatalf("fetching AI CodeFix enablement: %v", err)
	}
	if got.Enablement != integrationv1alpha1.AICodeFixDisabled {
		t.Errorf("AI CodeFix enablement = %q, want %q", got.Enablement, integrationv1alpha1.AICodeFixDisabled)
	}
}
