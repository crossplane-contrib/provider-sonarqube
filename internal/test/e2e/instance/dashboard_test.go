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
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// newE2EDashboard returns a Dashboard managed resource with the given target
// parameters.
func newE2EDashboard(f *e2e.Framework, name string, params instancev1alpha1.DashboardParameters) *instancev1alpha1.Dashboard {
	params.Name = name
	params.Layout = `{"widgets":[]}`

	return &instancev1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace},
		Spec: instancev1alpha1.DashboardSpec{
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

// TestDashboardCRUD creates a project dashboard and a portfolio dashboard,
// renames the former, then deletes it and verifies it is gone from SonarQube.
// Custom dashboards are an Enterprise Edition feature, so this requires
// SONARQUBE_LICENSE_KEY; it skips otherwise.
func TestDashboardCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		projectKey   = "e2e-dashboard-project"
		portfolioKey = "e2e-dashboard-portfolio"
		renamed      = "e2e-dashboard-renamed"
	)

	f.CreateAndWaitForReady(t, newE2EApplicationProject(f, projectKey), 2*time.Minute)
	f.CreateAndWaitForReady(t, &instancev1alpha1.Portfolio{
		ObjectMeta: metav1.ObjectMeta{Name: portfolioKey, Namespace: f.Namespace},
		Spec: instancev1alpha1.PortfolioSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.PortfolioParameters{Key: portfolioKey, Name: portfolioKey},
		},
	}, 2*time.Minute)

	projectDashboard := newE2EDashboard(f, "e2e-dashboard-for-project", instancev1alpha1.DashboardParameters{
		ProjectKeyRef: &xpv1.Reference{Name: projectKey},
	})
	portfolioDashboard := newE2EDashboard(f, "e2e-dashboard-for-portfolio", instancev1alpha1.DashboardParameters{
		PortfolioKeyRef: &xpv1.Reference{Name: portfolioKey},
	})

	for _, dashboard := range []*instancev1alpha1.Dashboard{projectDashboard, portfolioDashboard} {
		f.CreateAndWaitForReady(t, dashboard, 2*time.Minute)
		e2e.AssertReady(t, dashboard)
		e2e.AssertSynced(t, dashboard)

		got, err := f.FetchDashboard(context.Background(), meta.GetExternalName(dashboard))
		if err != nil {
			t.Fatalf("fetching dashboard: %v", err)
		}
		if got == nil || got.Name != dashboard.Spec.ForProvider.Name {
			t.Fatalf("dashboard %q not found in SonarQube, got %+v", dashboard.Name, got)
		}
	}

	id := meta.GetExternalName(projectDashboard)

	if err := f.Kube.Get(context.Background(), kubeKey(projectDashboard), projectDashboard); err != nil {
		t.Fatalf("re-fetching dashboard CR: %v", err)
	}
	projectDashboard.Spec.ForProvider.Name = renamed
	f.Update(t, projectDashboard)

	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, err := f.FetchDashboard(ctx, id)
		return got != nil && got.Name == renamed, err
	}); err != nil {
		t.Fatalf("dashboard %s was not renamed: %v", id, err)
	}

	f.Delete(t, projectDashboard)
	if err := f.WaitForDeletion(context.Background(), projectDashboard, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for the dashboard CR deletion: %v", err)
	}

	got, err := f.FetchDashboard(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching deleted dashboard: %v", err)
	}
	if got != nil {
		t.Errorf("dashboard %s still exists in SonarQube", id)
	}
}
