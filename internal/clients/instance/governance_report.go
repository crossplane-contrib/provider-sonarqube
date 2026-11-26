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

package instance

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errGetReportStatus indicates the report status could not be fetched.
	errGetReportStatus = "cannot get governance report status"
	// errUpdateReportFrequency indicates the report frequency update failed.
	errUpdateReportFrequency = "cannot update governance report frequency"
	// errUpdateReportRecipients indicates the report recipients update failed.
	errUpdateReportRecipients = "cannot update governance report recipients"
)

// GovernanceReportsClient is the interface for interacting with the
// SonarQube Governance Reports API.
type GovernanceReportsClient interface {
	Status(ctx context.Context, opt *sonar.GovernanceReportsStatusOptions) (*sonar.GovernanceReportsStatus, *http.Response, error)
	UpdateFrequency(ctx context.Context, opt *sonar.GovernanceReportsUpdateFrequencyOptions) (*http.Response, error)
	UpdateRecipients(ctx context.Context, opt *sonar.GovernanceReportsUpdateRecipientsOptions) (*http.Response, error)
}

// NewGovernanceReportsClient creates a new GovernanceReportsClient using the
// provided SonarQube client configuration.
func NewGovernanceReportsClient(clientConfig common.Config) GovernanceReportsClient {
	return common.NewClient(clientConfig).GovernanceReports
}

// IsGovernanceReportManaged returns true when at least one report setting is
// set, meaning the report endpoints have to be called.
func IsGovernanceReportManaged(spec *v1alpha1.GovernanceReportParameters) bool {
	return spec != nil && (spec.Frequency != nil || spec.Recipients != nil)
}

// ObserveGovernanceReport returns the report configuration of the component
// with the given key. It returns nil, without calling SonarQube, when no report
// setting is managed.
func ObserveGovernanceReport(ctx context.Context, client GovernanceReportsClient, key string, spec *v1alpha1.GovernanceReportParameters) (*v1alpha1.GovernanceReportObservation, error) {
	if !IsGovernanceReportManaged(spec) {
		return nil, nil //nolint:nilnil // no observation for unmanaged settings
	}

	status, resp, err := client.Status(ctx, &sonar.GovernanceReportsStatusOptions{ComponentKey: key}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return nil, errors.Wrap(err, errGetReportStatus)
	}

	return GenerateGovernanceReportObservation(status), nil
}

// GenerateGovernanceReportObservation converts a report status to a
// GovernanceReportObservation.
func GenerateGovernanceReportObservation(status *sonar.GovernanceReportsStatus) *v1alpha1.GovernanceReportObservation {
	if status == nil {
		return nil
	}

	recipients := slices.Clone(status.ComponentRecipients)
	slices.Sort(recipients)

	return &v1alpha1.GovernanceReportObservation{
		Frequency:       status.ComponentFrequency,
		GlobalFrequency: status.GlobalFrequency,
		Recipients:      recipients,
	}
}

// IsGovernanceReportUpToDate returns true when every managed report setting
// matches the observed state.
func IsGovernanceReportUpToDate(spec *v1alpha1.GovernanceReportParameters, observation *v1alpha1.GovernanceReportObservation) bool {
	return isReportFrequencyUpToDate(spec, observation) && areReportRecipientsUpToDate(spec, observation)
}

// isReportFrequencyUpToDate returns true when the frequency is unmanaged or
// matches the observed one.
func isReportFrequencyUpToDate(spec *v1alpha1.GovernanceReportParameters, observation *v1alpha1.GovernanceReportObservation) bool {
	if spec == nil || spec.Frequency == nil {
		return true
	}

	return observation != nil && *spec.Frequency == observation.Frequency
}

// areReportRecipientsUpToDate returns true when the recipients are unmanaged
// or match the observed ones, ignoring order.
func areReportRecipientsUpToDate(spec *v1alpha1.GovernanceReportParameters, observation *v1alpha1.GovernanceReportObservation) bool {
	if spec == nil || spec.Recipients == nil {
		return true
	}

	return observation != nil && helpers.AreStringSlicesEqualDeDuped(*spec.Recipients, observation.Recipients)
}

// UpdateGovernanceReport applies the managed report settings that differ from
// the observed ones to the component with the given key.
func UpdateGovernanceReport(ctx context.Context, client GovernanceReportsClient, key string, spec *v1alpha1.GovernanceReportParameters, observation *v1alpha1.GovernanceReportObservation) error {
	if !isReportFrequencyUpToDate(spec, observation) {
		resp, err := client.UpdateFrequency(ctx, &sonar.GovernanceReportsUpdateFrequencyOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			ComponentKey: key,
			Frequency:    *spec.Frequency,
		})
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrap(err, errUpdateReportFrequency)
		}
	}

	if !areReportRecipientsUpToDate(spec, observation) {
		resp, err := client.UpdateRecipients(ctx, &sonar.GovernanceReportsUpdateRecipientsOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			ComponentKey: key,
			Recipients:   strings.Join(*spec.Recipients, ","),
		})
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrap(err, errUpdateReportRecipients)
		}
	}

	return nil
}
