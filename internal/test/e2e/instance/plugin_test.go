//go:build e2e && !enterprise

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
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestPluginInstall creates a Plugin CR, waits for Ready, and verifies
// the plugin is installed in SonarQube. The t.Cleanup uninstalls it.
//
// Community Edition only: SonarQube's commercial editions (Developer,
// Enterprise, Data Center) disable the marketplace plugin-install WS
// entirely ("This WS is unsupported in commercial edition. Please install
// plugin manually.") regardless of which plugin is requested, so this
// resource cannot be exercised against the enterprise instance.
func TestPluginInstall(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName    = "e2e-plugin-findbugs"
		pluginKey = "findbugs"
	)

	plugin := &instancev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.PluginSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.PluginParameters{
				Key: pluginKey,
			},
		},
	}

	f.CreateAndWaitForReady(t, plugin, 5*time.Minute)
	e2e.AssertReady(t, plugin)
	e2e.AssertSynced(t, plugin)
	e2e.AssertExternalName(t, plugin, pluginKey)

	installed, err := f.FindInstalledPlugin(pluginKey)
	if err != nil {
		t.Fatalf("searching installed plugins: %v", err)
	}

	if installed != nil {
		if installed.Key != pluginKey {
			t.Errorf("plugin key = %q, want %q", installed.Key, pluginKey)
		}
		return
	}

	// Plugins require a SonarQube restart to move from pending to installed;
	// accept pending-install as equivalent for e2e purposes.
	pending, err := f.FindPendingInstallPlugin(pluginKey)
	if err != nil {
		t.Fatalf("searching pending plugins: %v", err)
	}

	if pending == nil {
		t.Fatalf("plugin %q not found in SonarQube installed or pending-install plugins", pluginKey)
	}
}
