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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// waitForReportRecipients polls SonarQube until the report recipients of the
// component match want, ignoring order.
func waitForReportRecipients(t *testing.T, f *e2e.Framework, key string, want []string) {
	t.Helper()

	var got []string
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		status, _, err := f.Sonar.GovernanceReports.Status(ctx, &sonar.GovernanceReportsStatusOptions{ComponentKey: key}) //nolint:bodyclose // closed by the client
		if err != nil {
			return false, err
		}
		got = slices.Sorted(slices.Values(status.ComponentRecipients))
		return slices.Equal(got, want), nil
	}); err != nil {
		t.Fatalf("component %q report recipients = %v, want %v: %v", key, got, want, err)
	}
}

// waitForPortfolioReport polls the Portfolio until its observed report matches
// want.
func waitForPortfolioReport(t *testing.T, f *e2e.Framework, portfolio *instancev1alpha1.Portfolio, frequency string, recipients []string) {
	t.Helper()

	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		// Reset the status so that fields omitted by the server are not kept.
		portfolio.Status = instancev1alpha1.PortfolioStatus{}
		if err := f.Kube.Get(ctx, kubeKey(portfolio), portfolio); err != nil {
			return false, err
		}
		report := portfolio.Status.AtProvider.Report
		return report != nil && report.Frequency == frequency && equality.Semantic.DeepEqual(report.Recipients, recipients), nil
	}); err != nil {
		t.Fatalf("portfolio report = %+v, want frequency %q and recipients %v: %v", portfolio.Status.AtProvider.Report, frequency, recipients, err)
	}
}

// TestPortfolioGovernanceReport creates a Portfolio with report settings,
// verifies they are applied and read back, then changes the frequency and
// clears the recipients. It requires SONARQUBE_LICENSE_KEY.
func TestPortfolioGovernanceReport(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		key       = "e2e-portfolio-report"
		recipient = "e2e-reports@example.com"
	)

	portfolio := &instancev1alpha1.Portfolio{
		ObjectMeta: metav1.ObjectMeta{Name: key, Namespace: f.Namespace},
		Spec: instancev1alpha1.PortfolioSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.PortfolioParameters{
				Key:        key,
				Name:       "E2E Portfolio Report",
				Visibility: "public",
				Report: &instancev1alpha1.GovernanceReportParameters{
					Frequency:  new("weekly"),
					Recipients: &[]string{recipient},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, portfolio, 2*time.Minute)
	waitForReportRecipients(t, f, key, []string{recipient})
	waitForPortfolioReport(t, f, portfolio, "weekly", []string{recipient})

	portfolio.Spec.ForProvider.Report = &instancev1alpha1.GovernanceReportParameters{
		Frequency:  new("daily"),
		Recipients: &[]string{},
	}
	f.Update(t, portfolio)

	waitForReportRecipients(t, f, key, nil)
	waitForPortfolioReport(t, f, portfolio, "daily", nil)
}

// TestApplicationGovernanceReport creates an Application with a report
// frequency and verifies it is applied and read back. Recipients are not
// exercised: SonarQube only documents them for portfolios. It requires
// SONARQUBE_LICENSE_KEY.
func TestApplicationGovernanceReport(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const key = "e2e-application-report"

	application := &instancev1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: key, Namespace: f.Namespace},
		Spec: instancev1alpha1.ApplicationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ApplicationParameters{
				Key:        key,
				Name:       "E2E Application Report",
				Visibility: "public",
				Report:     &instancev1alpha1.GovernanceReportParameters{Frequency: new("weekly")},
			},
		},
	}

	f.CreateAndWaitForReady(t, application, 2*time.Minute)

	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := f.Kube.Get(ctx, kubeKey(application), application); err != nil {
			return false, err
		}
		report := application.Status.AtProvider.Report
		return report != nil && report.Frequency == "weekly", nil
	}); err != nil {
		t.Fatalf("application report = %+v, want frequency weekly: %v", application.Status.AtProvider.Report, err)
	}
}
