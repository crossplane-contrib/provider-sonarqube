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
	"os"
	"sync"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// EnvLicenseKey names the environment variable holding a real SonarQube
// Enterprise Edition license key. It is optional: a valid license is a paid
// artifact that not every environment has, so tests that need one skip
// themselves via requireLicense rather than fail when it is unset.
const EnvLicenseKey = "SONARQUBE_LICENSE_KEY"

const (
	licenseSecretName     = "e2e-license-key"
	licenseCRName         = "e2e-license"
	licenseConnSecretName = "e2e-license-conn"
)

// licenseOnce/licenseErr memoize the outcome of applying the license once
// per test binary run - license state lives in the SonarQube instance
// itself, not per-test, so every test that needs a licensed instance shares
// the same underlying License managed resource instead of re-applying it.
var (
	licenseOnce sync.Once
	licenseErr  error
)

// requireLicense skips the calling test unless EnvLicenseKey is set, then
// ensures the shared License managed resource has been applied and is
// Ready, failing the test if that setup fails.
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

// applyLicense creates (or reuses) the Secret and License managed resource
// that apply key to the SonarQube instance, and waits for the License to
// report Ready. It deliberately does not register a t.Cleanup deletion:
// the License must stay applied for the lifetime of the test binary so
// later tests (e.g. Portfolio) still see a licensed instance.
func applyLicense(f *e2e.Framework, key string) error {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: licenseSecretName, Namespace: f.Namespace},
		StringData: map[string]string{"licenseKey": key},
	}
	if err := f.Kube.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	lic := &instancev1alpha1.License{ObjectMeta: metav1.ObjectMeta{Name: licenseCRName, Namespace: f.Namespace}}
	err := f.Kube.Get(ctx, client.ObjectKeyFromObject(lic), lic)
	if apierrors.IsNotFound(err) {
		lic = &instancev1alpha1.License{
			ObjectMeta: metav1.ObjectMeta{Name: licenseCRName, Namespace: f.Namespace},
			Spec: instancev1alpha1.LicenseSpec{
				ManagedResourceSpec: xpv1.ManagedResourceSpec{
					ProviderConfigReference: &xpv1.ProviderConfigReference{
						Kind: "ClusterProviderConfig",
						Name: f.ProviderConfigName,
					},
					WriteConnectionSecretToReference: &xpv1.LocalSecretReference{
						Name: licenseConnSecretName,
					},
				},
				ForProvider: instancev1alpha1.LicenseParameters{
					LicenseKeySecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: secret.Name, Namespace: secret.Namespace},
						Key:             "licenseKey",
					},
				},
			},
		}
		err = f.Kube.Create(ctx, lic)
	}
	if err != nil {
		return err
	}

	if err := f.WaitForReady(ctx, lic, 3*time.Minute); err != nil {
		return err
	}
	return nil
}
