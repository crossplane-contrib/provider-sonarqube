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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestInstanceConfigurationCommunity manages the Community Edition options of
// an InstanceConfiguration. It keeps the current Clean Code policy mode, as
// switching it re-indexes every issue and would disturb the tests running
// in parallel, flips the default project visibility, then checks that
// deleting the resource leaves the visibility in place. The original
// visibility is restored once the resource is gone.
func TestInstanceConfigurationCommunity(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	mode, err := f.FetchCleanCodePolicyMode(ctx)
	if err != nil {
		t.Fatalf("fetching clean code policy mode: %v", err)
	}

	originalVisibility := "public"
	setting, err := f.FetchSettingValue(ctx, "", instance.DefaultProjectVisibilitySettingKey)
	if err != nil {
		t.Fatalf("fetching default project visibility: %v", err)
	}
	if setting != nil && setting.Value != "" {
		originalVisibility = setting.Value
	}
	flippedVisibility := "private"
	if originalVisibility == "private" {
		flippedVisibility = "public"
	}

	// Registered before the resource is created so that it runs after the
	// resource deletion registered by CreateAndWaitForReady.
	t.Cleanup(func() {
		if restoreErr := f.SetDefaultProjectVisibility(context.Background(), originalVisibility); restoreErr != nil {
			t.Logf("cleanup: restoring default project visibility to %q: %v", originalVisibility, restoreErr)
		}
	})

	ic := &instancev1alpha1.InstanceConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-instanceconfiguration-community", Namespace: f.Namespace},
		Spec: instancev1alpha1.InstanceConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.InstanceConfigurationParameters{
				CleanCodePolicyMode:      ptr.To(mode),
				DefaultProjectVisibility: ptr.To(originalVisibility),
			},
		},
	}

	f.CreateAndWaitForReady(t, ic, 2*time.Minute)
	e2e.AssertReady(t, ic)
	e2e.AssertSynced(t, ic)

	if got := ptr.Deref(ic.Status.AtProvider.CleanCodePolicyMode, ""); got != mode {
		t.Errorf("atProvider.cleanCodePolicyMode = %q, want %q", got, mode)
	}
	if got := ptr.Deref(ic.Status.AtProvider.DefaultProjectVisibility, ""); got != originalVisibility {
		t.Errorf("atProvider.defaultProjectVisibility = %q, want %q", got, originalVisibility)
	}
	if ic.Status.AtProvider.ComputeEngineWorkerCount != nil || ic.Status.AtProvider.SCIMEnabled != nil || ic.Status.AtProvider.SCAEnabled != nil {
		t.Errorf("atProvider reports options left out of the spec: %+v", ic.Status.AtProvider)
	}

	if err = f.Kube.Get(ctx, kubeKey(ic), ic); err != nil {
		t.Fatalf("get %s: %v", ic.Name, err)
	}
	ic.Spec.ForProvider.DefaultProjectVisibility = ptr.To(flippedVisibility)
	f.Update(t, ic)

	// WaitForReady returns immediately when conditions are already True from
	// the previous reconcile; poll SonarQube directly until the controller
	// has applied the spec change.
	var lastVisibility string
	if pollErr := wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, false, func(ctx context.Context) (bool, error) {
		value, fetchErr := f.FetchSettingValue(ctx, "", instance.DefaultProjectVisibilitySettingKey)
		if fetchErr != nil {
			return false, fetchErr
		}
		if value == nil {
			return false, nil
		}
		lastVisibility = value.Value
		return value.Value == flippedVisibility, nil
	}); pollErr != nil {
		t.Fatalf("default project visibility did not reach %q: last=%q: %v", flippedVisibility, lastVisibility, pollErr)
	}

	f.Delete(t, ic)
	if err = f.WaitForDeletion(ctx, ic, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for deletion of %s: %v", ic.Name, err)
	}

	value, err := f.FetchSettingValue(ctx, "", instance.DefaultProjectVisibilitySettingKey)
	if err != nil {
		t.Fatalf("fetching default project visibility after deletion: %v", err)
	}
	if value == nil || value.Value != flippedVisibility {
		t.Errorf("default project visibility after deletion = %+v, want %q left in place", value, flippedVisibility)
	}

	gotMode, err := f.FetchCleanCodePolicyMode(ctx)
	if err != nil {
		t.Fatalf("fetching clean code policy mode after deletion: %v", err)
	}
	if gotMode != mode {
		t.Errorf("clean code policy mode after deletion = %q, want %q left in place", gotMode, mode)
	}
}
