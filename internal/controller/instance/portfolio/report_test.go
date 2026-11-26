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

package portfolio

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// newReportExternal returns an external client for an existing portfolio
// matching newReportResource, using the given report client.
func newReportExternal(reports *fake.MockGovernanceReportsClient) *external {
	return &external{
		client: &fake.MockPortfoliosClient{
			ShowFn: func(_ *sonar.ViewsShowOptions) (*sonar.ViewDetails, *http.Response, error) {
				return &sonar.ViewDetails{Key: testPortfolioKey, Name: testPortfolioName, SelectionMode: selectionModeNone}, mockHTTPOK(), nil
			},
			UpdateFn: func(_ *sonar.ViewsUpdateOptions) (*http.Response, error) {
				return mockHTTPOK(), nil
			},
			SetNoneModeFn: func(_ *sonar.ViewsSetNoneModeOptions) (*http.Response, error) {
				return mockHTTPOK(), nil
			},
		},
		reports: reports,
	}
}

// newReportResource returns an existing Portfolio with the given report
// settings.
func newReportResource(report *v1alpha1.GovernanceReportParameters) *v1alpha1.Portfolio {
	portfolio := newPortfolio(testPortfolioKey, testPortfolioName, testPortfolioKey)
	portfolio.Spec.ForProvider.Report = report

	return portfolio
}

// TestReportReconciliation verifies a drifted report frequency is detected by
// Observe and fixed by Update.
func TestReportReconciliation(t *testing.T) {
	t.Parallel()

	frequency := "monthly"
	e := newReportExternal(&fake.MockGovernanceReportsClient{
		StatusFn: func(_ *sonar.GovernanceReportsStatusOptions) (*sonar.GovernanceReportsStatus, *http.Response, error) {
			return &sonar.GovernanceReportsStatus{ComponentFrequency: frequency}, mockHTTPOK(), nil
		},
		UpdateFrequencyFn: func(opt *sonar.GovernanceReportsUpdateFrequencyOptions) (*http.Response, error) {
			frequency = opt.Frequency

			return mockHTTPOK(), nil
		},
	})
	resource := newReportResource(&v1alpha1.GovernanceReportParameters{Frequency: new("daily")})

	for _, wantUpToDate := range []bool{false, true} {
		observation, err := e.Observe(context.Background(), resource)
		if err != nil {
			t.Fatalf("Observe() unexpected error: %v", err)
		}

		if observation.ResourceUpToDate != wantUpToDate {
			t.Fatalf("Observe() up to date = %v, want %v", observation.ResourceUpToDate, wantUpToDate)
		}

		_, err = e.Update(context.Background(), resource)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}
	}

	if frequency != "daily" {
		t.Errorf("frequency = %q, want daily", frequency)
	}
}

// TestReportUnmanaged verifies the report endpoints are not called when no
// setting is managed: every method of the empty mock returns an error.
func TestReportUnmanaged(t *testing.T) {
	t.Parallel()

	e := newReportExternal(&fake.MockGovernanceReportsClient{})
	resource := newReportResource(&v1alpha1.GovernanceReportParameters{})

	observation, err := e.Observe(context.Background(), resource)
	if err != nil || !observation.ResourceUpToDate {
		t.Fatalf("Observe() = %+v, %v, want up to date", observation, err)
	}

	_, err = e.Update(context.Background(), resource)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}
}

// TestReportErrors verifies report API errors are returned.
func TestReportErrors(t *testing.T) {
	t.Parallel()

	e := newReportExternal(&fake.MockGovernanceReportsClient{
		StatusFn: func(_ *sonar.GovernanceReportsStatusOptions) (*sonar.GovernanceReportsStatus, *http.Response, error) {
			return nil, nil, errBoom
		},
		UpdateFrequencyFn: func(_ *sonar.GovernanceReportsUpdateFrequencyOptions) (*http.Response, error) {
			return nil, errBoom
		},
	})
	resource := newReportResource(&v1alpha1.GovernanceReportParameters{Frequency: new("daily")})

	_, err := e.Observe(context.Background(), resource)
	if !errors.Is(err, errBoom) {
		t.Errorf("Observe() error = %v, want %v", err, errBoom)
	}

	_, err = e.Update(context.Background(), resource)
	if !errors.Is(err, errBoom) {
		t.Errorf("Update() error = %v, want %v", err, errBoom)
	}
}
