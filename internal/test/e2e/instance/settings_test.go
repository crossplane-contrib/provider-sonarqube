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

// TestSettingsGlobalScalar sets a scalar global-scope setting and verifies
// SonarQube reports the same value on GET.
//
// Not run in parallel: concurrent global-scope Settings tests would write
// to the same SonarQube instance and race each other. One global settings
// test per run is the safe upper bound.
func TestSettingsGlobalScalar(t *testing.T) {
	f := e2e.New(t)
	const (
		crName       = "e2e-settings-global-scalar"
		settingKey   = "sonar.forceAuthentication"
		settingValue = "false"
	)

	settings := &instancev1alpha1.Settings{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.SettingsSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.SettingsParameters{
				Settings: map[string]instancev1alpha1.SettingParameters{
					settingKey: {Value: ptr.To(settingValue)},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, settings, 2*time.Minute)
	e2e.AssertReady(t, settings)
	e2e.AssertSynced(t, settings)

	got, err := f.FetchSettingValue(context.Background(), "", settingKey)
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
