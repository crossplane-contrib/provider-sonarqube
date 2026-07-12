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
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestPortfolioCRUD creates a Portfolio, waits for Ready, and verifies it
// is queryable via SonarQube's Views.Show with the matching name and
// visibility. Portfolios are an Enterprise Edition feature that only
// function on a licensed instance, so this requires SONARQUBE_LICENSE_KEY;
// it skips otherwise.
func TestPortfolioCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName = "e2e-portfolio-crud"
		pfKey  = "e2e-portfolio-crud"
		pfName = "E2E Portfolio CRUD"
	)

	portfolio := &instancev1alpha1.Portfolio{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.PortfolioSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.PortfolioParameters{
				Key:         pfKey,
				Name:        pfName,
				Description: "created by the enterprise e2e suite",
				Visibility:  "public",
			},
		},
	}

	f.CreateAndWaitForReady(t, portfolio, 2*time.Minute)
	e2e.AssertReady(t, portfolio)
	e2e.AssertSynced(t, portfolio)
	e2e.AssertExternalName(t, portfolio, pfKey)

	got, err := f.FindPortfolioByKey(pfKey)
	if err != nil {
		t.Fatalf("fetching portfolio: %v", err)
	}
	if got == nil {
		t.Fatalf("portfolio %q not found in SonarQube", pfKey)
	}
	if got.Name != pfName {
		t.Errorf("portfolio name = %q, want %q", got.Name, pfName)
	}
	if got.Visibility != "public" {
		t.Errorf("portfolio visibility = %q, want %q", got.Visibility, "public")
	}
}
