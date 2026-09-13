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

// TestNewCodePeriodCRUD creates an instance-wide NewCodePeriod, waits for
// Ready, and verifies SonarQube reports the same type and value.
func TestNewCodePeriodCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	ncp := &instancev1alpha1.NewCodePeriod{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-newcodeperiod-crud", Namespace: f.Namespace},
		Spec: instancev1alpha1.NewCodePeriodSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.NewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
		},
	}

	f.CreateAndWaitForReady(t, ncp, 2*time.Minute)
	e2e.AssertReady(t, ncp)
	e2e.AssertSynced(t, ncp)

	got, err := f.FetchInstanceNewCodePeriod(context.Background())
	if err != nil {
		t.Fatalf("fetching instance new code period: %v", err)
	}
	if got == nil {
		t.Fatal("instance new code period not found in SonarQube")
	}
	if got.Type != "NUMBER_OF_DAYS" {
		t.Errorf("instance new code period type = %q, want %q", got.Type, "NUMBER_OF_DAYS")
	}
	if got.Value != "30" {
		t.Errorf("instance new code period value = %q, want %q", got.Value, "30")
	}
}

// TestNewCodePeriodUpdate creates an instance-wide NewCodePeriod, then
// updates the value in place and verifies SonarQube reports the new value.
func TestNewCodePeriodUpdate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	ncp := &instancev1alpha1.NewCodePeriod{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-newcodeperiod-update", Namespace: f.Namespace},
		Spec: instancev1alpha1.NewCodePeriodSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.NewCodePeriodParameters{
				Type:  "NUMBER_OF_DAYS",
				Value: ptr.To("30"),
			},
		},
	}

	f.CreateAndWaitForReady(t, ncp, 2*time.Minute)

	if err := f.Kube.Get(context.Background(), kubeKey(ncp), ncp); err != nil {
		t.Fatalf("get %s: %v", ncp.Name, err)
	}
	ncp.Spec.ForProvider.Value = ptr.To("45")
	if err := f.Kube.Update(context.Background(), ncp); err != nil {
		t.Fatalf("updating %s/%s: %v", ncp.GetNamespace(), ncp.GetName(), err)
	}
	if err := f.WaitForReady(context.Background(), ncp, 2*time.Minute); err != nil {
		t.Fatalf("waiting for %s/%s to be Ready: %v\n  conditions: %s",
			ncp.GetNamespace(), ncp.GetName(), err, e2e.SummariseConditions(ncp))
	}

	got, err := f.FetchInstanceNewCodePeriod(context.Background())
	if err != nil {
		t.Fatalf("fetching instance new code period: %v", err)
	}
	if got == nil {
		t.Fatal("instance new code period not found in SonarQube")
	}
	if got.Value != "45" {
		t.Errorf("instance new code period value = %q, want %q", got.Value, "45")
	}
}
