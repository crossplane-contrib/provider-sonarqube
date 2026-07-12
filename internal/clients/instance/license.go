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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// LicensesClient handles SonarQube License API operations.
type LicensesClient interface {
	Get(ctx context.Context) (*sonar.LicenseGet, *http.Response, error)
	Set(ctx context.Context, opt *sonar.LicenseSetOptions) (*http.Response, error)
	UnsetLicense(ctx context.Context) (*http.Response, error)
}

// NewLicensesClient creates a new LicensesClient from the given config.
func NewLicensesClient(clientConfig common.Config) LicensesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.Editions
}

// GenerateLicenseObservation builds a LicenseObservation from the SonarQube
// API response.
func GenerateLicenseObservation(license *sonar.License) v1alpha1.LicenseObservation {
	if license == nil {
		return v1alpha1.LicenseObservation{}
	}

	return v1alpha1.LicenseObservation{
		ContactEmail:           license.ContactEmail,
		ExpiresAt:              license.ExpiresAt,
		IsExpired:              license.IsExpired,
		IsOfficialDistribution: license.IsOfficialDistribution,
		IsSupported:            license.IsSupported,
		IsValidEdition:         license.IsValidEdition,
		IsValidServerId:        license.IsValidServerId,
		LOCsMax:                license.LOCsMax,
		LOCsRemaining:          license.LOCsRemaining,
		Organization:           license.Organization,
		ProductEdition:         license.ProductEdition,
		ServerId:               license.ServerId,
		Type:                   license.Type,
	}
}

// LateInitializeLicense fills empty spec fields from the observation.
// License has no late-initializable fields.
func LateInitializeLicense(spec *v1alpha1.LicenseParameters, observation *v1alpha1.LicenseObservation) {
	if spec == nil || observation == nil {
		return
	}
}

// IsLicenseLateInitialized returns true if the spec changed
// during late initialization.
func IsLicenseLateInitialized(former, current *v1alpha1.LicenseParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !cmp.Equal(former, current)
}

// IsLicenseUpToDate returns true if the desired license key matches the
// license key last saved to the connection secret.
func IsLicenseUpToDate(desiredLicenseKey, savedLicenseKey string) bool {
	return desiredLicenseKey == savedLicenseKey
}
