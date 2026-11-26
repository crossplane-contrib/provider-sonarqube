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
	"slices"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

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

	got, err := f.FindPortfolioByKey(context.Background(), pfKey)
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

// newE2EPortfolio returns a Portfolio managed resource with the given key.
func newE2EPortfolio(f *e2e.Framework, key string, params instancev1alpha1.PortfolioParameters) *instancev1alpha1.Portfolio {
	params.Key = key
	params.Name = key

	return &instancev1alpha1.Portfolio{
		ObjectMeta: metav1.ObjectMeta{Name: key, Namespace: f.Namespace},
		Spec: instancev1alpha1.PortfolioSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: params,
		},
	}
}

// portfolioMembers are the members of a portfolio as returned by
// SonarQube, each sorted.
type portfolioMembers struct {
	Projects      []string
	Applications  []string
	References    []string
	SubPortfolios []string
}

// observePortfolioMembers extracts the members of a portfolio.
func observePortfolioMembers(details *sonar.ViewDetails) portfolioMembers {
	var members portfolioMembers
	for _, project := range details.SelectedProjects {
		members.Projects = append(members.Projects, project.ProjectKey)
	}
	for _, subView := range details.SubViews {
		switch subView.Qualifier {
		case "APP":
			members.Applications = append(members.Applications, subView.OriginalKey)
		case "VW":
			members.References = append(members.References, subView.OriginalKey)
		case "SVW":
			members.SubPortfolios = append(members.SubPortfolios, subView.Key)
		}
	}
	slices.Sort(members.Projects)
	slices.Sort(members.Applications)
	slices.Sort(members.References)
	slices.Sort(members.SubPortfolios)
	return members
}

// waitForPortfolioMembers polls SonarQube until the portfolio members match
// want.
func waitForPortfolioMembers(t *testing.T, f *e2e.Framework, key string, want portfolioMembers) {
	t.Helper()

	var got portfolioMembers
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		details, err := f.FindPortfolioByKey(ctx, key)
		if err != nil || details == nil {
			return false, err
		}
		got = observePortfolioMembers(details)
		return slices.Equal(got.Projects, want.Projects) &&
			slices.Equal(got.Applications, want.Applications) &&
			slices.Equal(got.References, want.References) &&
			slices.Equal(got.SubPortfolios, want.SubPortfolios), nil
	}); err != nil {
		t.Fatalf("portfolio %q members = %+v, want %+v: %v", key, got, want, err)
	}
}

// TestPortfolioMembers creates a MANUAL Portfolio selecting a Project,
// adding an Application and referencing another Portfolio, with a
// sub-portfolio created under it through its parent reference. It verifies
// the hierarchy in SonarQube, then drops the project and the referenced
// portfolio from the spec and verifies they are removed while the
// sub-portfolio is kept. Requires SONARQUBE_LICENSE_KEY; skips otherwise.
//
// Project and application branches are not exercised: they require analyzed
// branches, which the e2e environment does not produce.
func TestPortfolioMembers(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		projectKey     = "e2e-portfolio-members-project"
		applicationKey = "e2e-portfolio-members-app"
		referencedKey  = "e2e-portfolio-members-ref"
		rootKey        = "e2e-portfolio-members-root"
		childKey       = "e2e-portfolio-members-child"
	)

	f.CreateAndWaitForReady(t, newE2EApplicationProject(f, projectKey), 2*time.Minute)

	application := &instancev1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: applicationKey, Namespace: f.Namespace},
		Spec: instancev1alpha1.ApplicationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ApplicationParameters{
				Key:        applicationKey,
				Name:       applicationKey,
				Visibility: "public",
				Projects:   []string{projectKey},
			},
		},
	}
	f.CreateAndWaitForReady(t, application, 2*time.Minute)

	f.CreateAndWaitForReady(t, newE2EPortfolio(f, referencedKey, instancev1alpha1.PortfolioParameters{Visibility: "public"}), 2*time.Minute)

	root := newE2EPortfolio(f, rootKey, instancev1alpha1.PortfolioParameters{
		Visibility:       "public",
		SelectionMode:    "MANUAL",
		Projects:         &[]instancev1alpha1.PortfolioProjectParameters{{KeyRef: &xpv1.Reference{Name: projectKey}}},
		Applications:     &[]instancev1alpha1.PortfolioApplicationParameters{{KeyRef: &xpv1.Reference{Name: applicationKey}}},
		SubPortfolioRefs: []xpv1.Reference{{Name: referencedKey}},
	})
	f.CreateAndWaitForReady(t, root, 2*time.Minute)
	e2e.AssertSynced(t, root)

	child := newE2EPortfolio(f, childKey, instancev1alpha1.PortfolioParameters{
		ParentRef: &xpv1.Reference{Name: rootKey},
	})
	f.CreateAndWaitForReady(t, child, 2*time.Minute)
	e2e.AssertSynced(t, child)

	waitForPortfolioMembers(t, f, rootKey, portfolioMembers{
		Projects:      []string{projectKey},
		Applications:  []string{applicationKey},
		References:    []string{referencedKey},
		SubPortfolios: []string{childKey},
	})

	// Drop the project and the referenced portfolio.
	if err := f.Kube.Get(context.Background(), kubeKey(root), root); err != nil {
		t.Fatalf("re-fetching portfolio CR: %v", err)
	}
	root.Spec.ForProvider.Projects = &[]instancev1alpha1.PortfolioProjectParameters{}
	root.Spec.ForProvider.SubPortfolios = &[]string{}
	root.Spec.ForProvider.SubPortfolioRefs = nil
	f.Update(t, root)

	waitForPortfolioMembers(t, f, rootKey, portfolioMembers{
		Applications:  []string{applicationKey},
		SubPortfolios: []string{childKey},
	})
}
