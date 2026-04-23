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

// TestSettingsGlobalScalar sets a scalar global-scope setting and verifies
// SonarQube reports the same value on GET.
//
// Not run in parallel: concurrent global-scope Settings tests would write
// to the same SonarQube instance and race each other. One global settings
// test per run is the safe upper bound.
//
// NOTE: this test asserts on Synced=True only — the Settings controller
// currently does not call SetConditions(xpv1.Available()), so Ready stays
// Unknown forever. Once that controller bug is fixed, switch the wait to
// CreateAndWaitForReady and add e2e.AssertReady. See provider source at
// internal/controller/instance/settings/settings.go (only Creating() and
// Deleting() are set today; Available() is missing).
func TestSettingsGlobalScalar(t *testing.T) {
	f := e2e.New(t)
	const (
		crName       = "e2e-settings-global-scalar"
		settingKey   = "sonar.forceAuthentication"
		settingValue = "false"
	)

	settings := &instancev1alpha1.Settings{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: instancev1alpha1.SettingsSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.SettingsParameters{
				Settings: map[string]instancev1alpha1.SettingParameters{
					settingKey: {Value: stringPtr(settingValue)},
				},
			},
		},
	}

	// Settings never reaches Ready=True today (see TODO above), so create
	// the resource and poll for Synced=True instead of using
	// CreateAndWaitForReady.
	f.Apply(t, settings)
	t.Cleanup(func() {
		f.Delete(t, settings)
		_ = f.WaitForDeletion(context.Background(), settings, e2e.DefaultDeleteTimeout)
	})

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if err := f.Kube.Get(context.Background(), kubeKey(settings), settings); err != nil {
			t.Fatalf("get %s: %v", settings.Name, err)
		}
		if settings.GetCondition(xpv1.TypeSynced).Status == corev1.ConditionTrue {
			break
		}
		time.Sleep(2 * time.Second)
	}
	e2e.AssertSynced(t, settings)

	got, err := f.FetchSettingValue("", settingKey)
	if err != nil {
		t.Fatalf("fetching setting %q: %v", settingKey, err)
	}
	if got == nil {
		t.Fatalf("setting %q not found in SonarQube", settingKey)
	}
	if got.Value != settingValue {
		t.Errorf("setting %q value = %q, want %q", settingKey, got.Value, settingValue)
	}
}
