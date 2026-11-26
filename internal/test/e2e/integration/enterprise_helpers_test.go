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
	"os"
	"sync"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// EnvLicenseKey names the environment variable holding a real SonarQube
// Enterprise Edition license key. Tests that need one skip themselves via
// requireLicense when it is unset.
const EnvLicenseKey = "SONARQUBE_LICENSE_KEY"

// The License managed resource is shared with the instance e2e package,
// which runs in parallel in the same namespace: both packages create (or
// reuse) the same Secret and License.
const (
	licenseSecretName     = "e2e-license-key"
	licenseCRName         = "e2e-license"
	licenseConnSecretName = "e2e-license-conn"
)

// licenseOnce/licenseErr memoize applying the license once per test
// binary run.
var (
	licenseOnce sync.Once
	licenseErr  error
)

// requireLicense skips the calling test unless EnvLicenseKey is set, then
// ensures the shared License managed resource is applied and Ready.
func requireLicense(t *testing.T, f *e2e.Framework) {
	t.Helper()

	key := os.Getenv(EnvLicenseKey)
	if key == "" {
		t.Skipf("%s is not set; skipping test that requires a licensed SonarQube instance", EnvLicenseKey)
	}

	licenseOnce.Do(func() { licenseErr = applyLicense(f, key) })
	if licenseErr != nil {
		t.Fatalf("applying shared License for enterprise e2e tests: %v", licenseErr)
	}
}

// applyLicense creates the Secret and License managed resource applying key,
// tolerating their creation by the instance e2e package, and waits for the
// License to report Ready. It does not delete them: later tests still need a
// licensed instance.
func applyLicense(f *e2e.Framework, key string) error {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: licenseSecretName, Namespace: f.Namespace},
		StringData: map[string]string{"licenseKey": key},
	}
	if err := f.Kube.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	lic := &instancev1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: licenseCRName, Namespace: f.Namespace},
		Spec: instancev1alpha1.LicenseSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: licenseConnSecretName},
			},
			ForProvider: instancev1alpha1.LicenseParameters{
				LicenseKeySecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secret.Name},
					Key:                  "licenseKey",
				},
			},
		},
	}
	if err := f.Kube.Create(ctx, lic); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return f.WaitForReady(ctx, lic, 3*time.Minute)
}
