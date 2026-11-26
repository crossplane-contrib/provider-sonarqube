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
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// scaOnce/scaErr memoize the outcome of enabling SCA once per test binary
// run, since SCA enablement is an instance-wide setting.
var (
	scaOnce sync.Once
	scaErr  error
)

// requireSCAEnabled skips the calling test unless SCA can be enabled on the
// licensed SonarQube instance. SCA is enabled through the API directly: no
// managed resource manages its enablement yet. It is left enabled for the
// rest of the test binary run, like the shared License.
func requireSCAEnabled(t *testing.T, f *e2e.Framework) {
	t.Helper()

	requireLicense(t, f)

	scaOnce.Do(func() { scaErr = enableSCA(f) })
	if scaErr != nil {
		t.Skipf("SCA cannot be enabled on this SonarQube instance (the license may not include it); skipping SCA test: %v", scaErr)
	}
}

// enableSCA enables SCA on the SonarQube instance, then checks that the
// license profiles API answers.
func enableSCA(f *e2e.Framework) error {
	ctx := context.Background()

	enablement, resp, err := f.Sonar.V2.Sca.GetEnablement(ctx)
	helpers.CloseBody(resp)
	if err != nil {
		return fmt.Errorf("reading SCA enablement: %w", err)
	}

	if !enablement.Enablement {
		_, resp, err = f.Sonar.V2.Sca.SetEnablement(ctx, &sonar.ScaSetEnablementOptions{Enablement: true})
		helpers.CloseBody(resp)
		if err != nil {
			return fmt.Errorf("enabling SCA: %w", err)
		}
	}

	_, resp, err = f.Sonar.V2.Sca.ListLicenseProfiles(ctx, &sonar.ScaLicenseProfileListOptions{})
	helpers.CloseBody(resp)
	if err != nil {
		return fmt.Errorf("listing SCA license profiles: %w", err)
	}

	return nil
}

// waitForSCALicenseProfile polls SonarQube until the license profile with
// the given identifier satisfies check.
func waitForSCALicenseProfile(t *testing.T, f *e2e.Framework, id, what string, check func(*sonar.ScaLicenseProfileDetails, []string) bool) {
	t.Helper()

	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		details, err := f.FetchSCALicenseProfile(ctx, id)
		if err != nil || details == nil {
			return false, err
		}
		projects, err := f.SCALicenseProfileProjects(ctx, id)
		if err != nil {
			return false, err
		}
		return check(details, projects), nil
	}); err != nil {
		t.Fatalf("SCA license profile %q: waiting for %s: %v", id, what, err)
	}
}

// scaCategoryPolicy returns the policy of the given category in details.
func scaCategoryPolicy(details *sonar.ScaLicenseProfileDetails, category string) string {
	for _, c := range details.Categories {
		if c.Key == category {
			return c.Policy
		}
	}
	return ""
}

// scaLicensePolicy returns the policy of the given license in details.
func scaLicensePolicy(details *sonar.ScaLicenseProfileDetails, spdxLicenseID string) string {
	for _, l := range details.Licenses {
		if l.SpdxLicenseId == spdxLicenseID {
			return l.Policy
		}
	}
	return ""
}

// oppositeSCAPolicy returns the other license policy.
func oppositeSCAPolicy(policy string) string {
	if policy == sonar.ScaLicensePolicyAllow {
		return sonar.ScaLicensePolicyDeny
	}
	return sonar.ScaLicensePolicyAllow
}

// TestSCALicenseProfileCRUD creates a Project and a SCALicenseProfile
// assigned to it, verifies the profile, its category policy and its
// assignment in SonarQube, then renames the profile, flips the category
// policy, sets a license policy and unassigns the project, and finally
// deletes the profile. SCA license profiles are an Enterprise Edition
// feature that needs SCA, so this requires SONARQUBE_LICENSE_KEY and a
// license that allows enabling SCA; it skips otherwise.
func TestSCALicenseProfileCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireSCAEnabled(t, f)

	const (
		crName      = "e2e-sca-license-profile-crud"
		profileName = "E2E SCA License Profile CRUD"
		renamed     = "E2E SCA License Profile CRUD renamed"
		projectKey  = "e2e-sca-license-profile-crud"
		category    = sonar.ScaLicenseCategoryCopyleftStrong
	)

	f.CreateAndWaitForReady(t, newE2EApplicationProject(f, projectKey), 2*time.Minute)

	profile := &instancev1alpha1.SCALicenseProfile{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.SCALicenseProfileSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.SCALicenseProfileParameters{
				Name:        profileName,
				Default:     new(false),
				Categories:  []instancev1alpha1.SCALicenseProfileCategoryPolicy{{Category: category, Policy: sonar.ScaLicensePolicyDeny}},
				ProjectRefs: []xpv1.Reference{{Name: projectKey}},
			},
		},
	}

	f.CreateAndWaitForReady(t, profile, 2*time.Minute)
	e2e.AssertReady(t, profile)
	e2e.AssertSynced(t, profile)

	id := profile.Status.AtProvider.ID
	if id == "" {
		t.Fatal("SCALicenseProfile status.atProvider.id is empty")
	}
	e2e.AssertExternalName(t, profile, id)

	waitForSCALicenseProfile(t, f, id, "the created profile", func(details *sonar.ScaLicenseProfileDetails, projects []string) bool {
		return details.Profile.Name == profileName &&
			scaCategoryPolicy(details, category) == sonar.ScaLicensePolicyDeny &&
			slices.Equal(projects, []string{projectKey})
	})

	details, err := f.FetchSCALicenseProfile(context.Background(), id)
	if err != nil || details == nil {
		t.Fatalf("fetching SCA license profile %q: %v", id, err)
	}
	if len(details.Licenses) == 0 {
		t.Fatalf("SCA license profile %q has no license policy", id)
	}
	license := details.Licenses[0].SpdxLicenseId
	licensePolicy := oppositeSCAPolicy(details.Licenses[0].Policy)

	// Rename, flip the category, set a license policy and unassign the project.
	if err := f.Kube.Get(context.Background(), kubeKey(profile), profile); err != nil {
		t.Fatalf("re-fetching SCALicenseProfile CR: %v", err)
	}
	profile.Spec.ForProvider.Name = renamed
	profile.Spec.ForProvider.Categories = []instancev1alpha1.SCALicenseProfileCategoryPolicy{{Category: category, Policy: sonar.ScaLicensePolicyAllow}}
	profile.Spec.ForProvider.Licenses = []instancev1alpha1.SCALicenseProfileLicensePolicy{{SPDXLicenseID: license, Policy: licensePolicy}}
	profile.Spec.ForProvider.Projects = nil
	profile.Spec.ForProvider.ProjectRefs = nil
	f.Update(t, profile)

	waitForSCALicenseProfile(t, f, id, "the updated profile", func(details *sonar.ScaLicenseProfileDetails, projects []string) bool {
		return details.Profile.Name == renamed &&
			scaCategoryPolicy(details, category) == sonar.ScaLicensePolicyAllow &&
			scaLicensePolicy(details, license) == licensePolicy &&
			len(projects) == 0
	})

	f.Delete(t, profile)
	if err := f.WaitForDeletion(context.Background(), profile, 2*time.Minute); err != nil {
		t.Fatalf("waiting for SCALicenseProfile deletion: %v", err)
	}
	if got, err := f.FetchSCALicenseProfile(context.Background(), id); err != nil || got != nil {
		t.Errorf("SCA license profile %q after deletion = %v, %v; want not found", id, got, err)
	}
}
