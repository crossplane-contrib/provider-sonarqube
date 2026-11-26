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

package iam_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// envLicenseKey names the environment variable holding a SonarQube Enterprise
// Edition license key, see the instance e2e package.
const envLicenseKey = "SONARQUBE_LICENSE_KEY"

// newLicensedFramework skips the test unless a license key is set, then applies
// it to SonarQube. The key is set through the API rather than through a License
// resource, which the instance e2e package owns.
func newLicensedFramework(t *testing.T) *e2e.Framework {
	t.Helper()

	key := os.Getenv(envLicenseKey)
	if key == "" {
		t.Skipf("%s is not set; skipping test that requires a licensed SonarQube instance", envLicenseKey)
	}

	f := e2e.New(t)

	resp, err := f.Sonar.Editions.Set(context.Background(), &sonar.LicenseSetOptions{License: key})
	defer helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("setting the SonarQube license: %v", helpers.RedactURLError(err))
	}

	return f
}

// newE2EMapping returns a DevOpsPermissionMapping managed resource.
func newE2EMapping(f *e2e.Framework, name string, platform iamv1alpha1.DevOpsPlatform, role string, permissions []string) *iamv1alpha1.DevOpsPermissionMapping {
	return &iamv1alpha1.DevOpsPermissionMapping{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace},
		Spec: iamv1alpha1.DevOpsPermissionMappingSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.DevOpsPermissionMappingParameters{
				Platform:    platform,
				Role:        role,
				Permissions: permissions,
			},
		},
	}
}

// waitForMappingPermissions waits until SonarQube grants exactly the wanted
// permissions to the role.
func waitForMappingPermissions(t *testing.T, f *e2e.Framework, platform, role string, want sonar.DopTranslationPermissions) {
	t.Helper()

	var got *sonar.DopTranslationPermissionMapping
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var err error
		got, err = f.FindDevOpsPermissionMapping(ctx, platform, role)
		if err != nil || got == nil {
			return false, err
		}
		return got.Permissions == want, nil
	}); err != nil {
		t.Fatalf("%s role %q mapping = %+v, want permissions %+v: %v", platform, role, got, want, err)
	}
}

// TestDevOpsPermissionMappingCustomRole creates the mapping of a custom GitHub
// role whose name needs URL escaping, updates it, then deletes it.
func TestDevOpsPermissionMappingCustomRole(t *testing.T) {
	t.Parallel()

	f := newLicensedFramework(t)
	const role = "E2E Custom Role"

	mapping := newE2EMapping(f, "e2e-mapping-custom", iamv1alpha1.DevOpsPlatformGitHub, role, []string{"user", "codeviewer"})

	f.CreateAndWaitForReady(t, mapping, 2*time.Minute)
	e2e.AssertSynced(t, mapping)
	e2e.AssertExternalName(t, mapping, role)
	waitForMappingPermissions(t, f, "github", role, sonar.DopTranslationPermissions{User: true, CodeViewer: true})

	if err := f.Kube.Get(context.Background(), kubeKey(mapping), mapping); err != nil {
		t.Fatalf("re-fetching mapping CR: %v", err)
	}
	if mapping.Status.AtProvider.BaseRole {
		t.Errorf("AtProvider.BaseRole = true, want false for a custom role")
	}
	mapping.Spec.ForProvider.Permissions = []string{"scan"}
	f.Update(t, mapping)
	waitForMappingPermissions(t, f, "github", role, sonar.DopTranslationPermissions{Scan: true})

	f.Delete(t, mapping)
	if err := f.WaitForDeletion(context.Background(), mapping, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for mapping deletion: %v", err)
	}
	got, err := f.FindDevOpsPermissionMapping(context.Background(), "github", role)
	if err != nil {
		t.Fatalf("fetching mappings: %v", err)
	}
	if got != nil {
		t.Errorf("mapping of custom role %q still exists after deletion: %+v", role, got)
	}
}

// TestDevOpsPermissionMappingBaseRole manages the mapping of a built-in GitLab
// role, and verifies that deleting the resource keeps its permissions.
func TestDevOpsPermissionMappingBaseRole(t *testing.T) {
	t.Parallel()

	f := newLicensedFramework(t)
	const role = "reporter"
	permissions := []string{"user", "codeviewer", "issueadmin"}
	want := sonar.DopTranslationPermissions{User: true, CodeViewer: true, IssueAdmin: true}

	mapping := newE2EMapping(f, "e2e-mapping-base", iamv1alpha1.DevOpsPlatformGitLab, role, slices.Clone(permissions))

	f.CreateAndWaitForReady(t, mapping, 2*time.Minute)
	e2e.AssertSynced(t, mapping)
	if !mapping.Status.AtProvider.BaseRole {
		t.Errorf("AtProvider.BaseRole = false, want true for role %q", role)
	}
	waitForMappingPermissions(t, f, "gitlab", role, want)

	f.Delete(t, mapping)
	if err := f.WaitForDeletion(context.Background(), mapping, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for mapping deletion: %v", err)
	}
	waitForMappingPermissions(t, f, "gitlab", role, want)
}
