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

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestQualityGateAiCodeAssurance creates a QualityGate qualified for AI Code
// Assurance, verifies it in SonarQube, then disqualifies it. AI Code
// Assurance requires a commercial edition, so this requires
// SONARQUBE_LICENSE_KEY; it skips otherwise.
func TestQualityGateAiCodeAssurance(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName = "e2e-qualitygate-aica"
		qgName = "e2e-qualitygate-aica"
	)

	qg := &instancev1alpha1.QualityGate{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.QualityGateSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityGateParameters{
				Name:            qgName,
				AiCodeAssurance: ptr.To(true),
			},
		},
	}

	f.CreateAndWaitForReady(t, qg, 2*time.Minute)
	e2e.AssertReady(t, qg)

	// The qualification is applied by the update following the creation.
	waitForQualityGateAiCodeAssurance(t, f, qgName, true)

	if err := f.Kube.Get(context.Background(), kubeKey(qg), qg); err != nil {
		t.Fatalf("re-fetching quality gate CR: %v", err)
	}
	qg.Spec.ForProvider.AiCodeAssurance = ptr.To(false)
	f.Update(t, qg)

	waitForQualityGateAiCodeAssurance(t, f, qgName, false)
}

// waitForQualityGateAiCodeAssurance polls SonarQube until the AI Code
// Assurance qualification of the quality gate matches want.
func waitForQualityGateAiCodeAssurance(t *testing.T, f *e2e.Framework, name string, want bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		got, err := f.FindQualityGate(context.Background(), name)
		if err != nil {
			t.Fatalf("listing quality gates: %v", err)
		}
		if got != nil && got.IsAiCodeSupported == want {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("quality gate %q AI Code Assurance did not become %t", name, want)
}
