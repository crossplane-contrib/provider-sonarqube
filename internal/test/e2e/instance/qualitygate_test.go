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

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestQualityGateCRUD creates a QualityGate with one condition, waits for
// Ready, verifies it exists in SonarQube with the expected metadata, and
// lets the t.Cleanup delete it.
func TestQualityGateCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName = "e2e-qualitygate-crud"
		qgName = "e2e-qualitygate-crud"
	)

	qg := &instancev1alpha1.QualityGate{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: instancev1alpha1.QualityGateSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityGateParameters{
				Name: qgName,
				Conditions: []instancev1alpha1.QualityGateConditionParameters{
					{Metric: "blocker_violations", Op: stringPtr("GT"), Error: "0"},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, qg, 2*time.Minute)
	e2e.AssertReady(t, qg)
	e2e.AssertSynced(t, qg)
	e2e.AssertExternalName(t, qg, qgName)

	got, err := f.FindQualityGate(qgName)
	if err != nil {
		t.Fatalf("listing quality gates: %v", err)
	}
	if got == nil {
		t.Fatalf("quality gate %q not found in SonarQube", qgName)
	}
	if got.IsBuiltIn {
		t.Errorf("quality gate %q is reported as built-in in SonarQube; want user-created", qgName)
	}
}

// TestQualityGateInvalidMetric creates a QualityGate whose condition metric
// is not accepted by SonarQube. The CR creation itself succeeds (the field
// passes CRD-level validation) but the controller must surface the error on
// the Synced condition instead of marking the resource Ready.
func TestQualityGateInvalidMetric(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName = "e2e-qualitygate-invalid"
		qgName = "e2e-qualitygate-invalid"
	)

	qg := &instancev1alpha1.QualityGate{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: instancev1alpha1.QualityGateSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityGateParameters{
				Name: qgName,
				Conditions: []instancev1alpha1.QualityGateConditionParameters{
					{Metric: "not_a_real_metric", Op: stringPtr("GT"), Error: "0"},
				},
			},
		},
	}

	f.Apply(t, qg)
	t.Cleanup(func() { f.Delete(t, qg); _ = f.WaitForDeletion(context.Background(), qg, e2e.DefaultDeleteTimeout) })

	// Wait a reasonable window for the controller to observe and fail.
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if err := f.Kube.Get(context.Background(), kubeKey(qg), qg); err != nil {
			t.Fatalf("get %s: %v", qg.Name, err)
		}
		if qg.GetCondition(xpv1.TypeSynced).Status == corev1.ConditionFalse {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("expected Synced=False for invalid QualityGate, got: %s", e2e.SummariseConditions(qg))
}
