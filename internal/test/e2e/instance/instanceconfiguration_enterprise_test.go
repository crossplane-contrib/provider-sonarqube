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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestInstanceConfigurationEnterprise manages the Enterprise Edition options
// of an InstanceConfiguration. It changes the Compute Engine worker count and
// declares the current SCIM and SCA enablement: toggling either would turn
// users and groups read-only or need an SCA license, and disturb the tests
// running in parallel. It manages none of the options of
// TestInstanceConfigurationCommunity, so the two resources do not conflict.
// The original worker count is restored once the resource is gone. Requires
// SONARQUBE_LICENSE_KEY.
func TestInstanceConfigurationEnterprise(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)
	ctx := context.Background()

	workerCount, err := f.FetchComputeEngineWorkerCount(ctx)
	if err != nil {
		t.Fatalf("fetching compute engine worker count: %v", err)
	}
	if !workerCount.CanSetWorkerCount {
		t.Skip("SonarQube reports the compute engine worker count cannot be set on this instance")
	}
	originalCount := workerCount.Value
	wantCount := int64(2)
	if originalCount == wantCount {
		wantCount = 1
	}

	scimEnabled, err := f.FetchSCIMEnabled(ctx)
	if err != nil {
		t.Fatalf("fetching SCIM status: %v", err)
	}
	scaEnabled, err := f.FetchSCAEnabled(ctx)
	if err != nil {
		t.Fatalf("fetching SCA enablement: %v", err)
	}

	// Registered before the resource is created so that it runs after the
	// resource deletion registered by CreateAndWaitForReady.
	t.Cleanup(func() {
		if restoreErr := f.SetComputeEngineWorkerCount(context.Background(), originalCount); restoreErr != nil {
			t.Logf("cleanup: restoring compute engine worker count to %d: %v", originalCount, restoreErr)
		}
	})

	ic := &instancev1alpha1.InstanceConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-instanceconfiguration-enterprise", Namespace: f.Namespace},
		Spec: instancev1alpha1.InstanceConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.InstanceConfigurationParameters{
				ComputeEngineWorkerCount: ptr.To(wantCount),
				SCIMEnabled:              ptr.To(scimEnabled),
				SCAEnabled:               ptr.To(scaEnabled),
			},
		},
	}

	f.CreateAndWaitForReady(t, ic, 2*time.Minute)
	e2e.AssertSynced(t, ic)

	// The first reconcile may report Ready before the update is applied;
	// poll SonarQube directly until the worker count matches.
	var lastCount int64
	if pollErr := wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, fetchErr := f.FetchComputeEngineWorkerCount(ctx)
		if fetchErr != nil {
			return false, fetchErr
		}
		lastCount = got.Value
		return got.Value == wantCount, nil
	}); pollErr != nil {
		t.Fatalf("compute engine worker count did not reach %d: last=%d: %v", wantCount, lastCount, pollErr)
	}

	if gotSCIM, scimErr := f.FetchSCIMEnabled(ctx); scimErr != nil || gotSCIM != scimEnabled {
		t.Errorf("SCIM enabled = %v (err %v), want %v", gotSCIM, scimErr, scimEnabled)
	}
	if gotSCA, scaErr := f.FetchSCAEnabled(ctx); scaErr != nil || gotSCA != scaEnabled {
		t.Errorf("SCA enabled = %v (err %v), want %v", gotSCA, scaErr, scaEnabled)
	}

	f.Delete(t, ic)
	if err = f.WaitForDeletion(ctx, ic, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for deletion of %s: %v", ic.Name, err)
	}

	got, err := f.FetchComputeEngineWorkerCount(ctx)
	if err != nil {
		t.Fatalf("fetching compute engine worker count after deletion: %v", err)
	}
	if got.Value != wantCount {
		t.Errorf("compute engine worker count after deletion = %d, want %d left in place", got.Value, wantCount)
	}
}
