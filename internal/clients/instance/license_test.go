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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestGenerateLicenseObservation tests all branches of
// GenerateLicenseObservation.
func TestGenerateLicenseObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		license *sonar.License
		want    v1alpha1.LicenseObservation
	}{
		"NilReturnsEmpty": {
			license: nil,
			want:    v1alpha1.LicenseObservation{},
		},
		"AllFieldsMapped": {
			license: &sonar.License{
				ContactEmail:           "admin@acme.com",
				ExpiresAt:              "2027-01-01",
				IsExpired:              false,
				IsOfficialDistribution: true,
				IsSupported:            true,
				IsValidEdition:         true,
				IsValidServerId:        true,
				LOCsMax:                1000000,
				LOCsRemaining:          500000,
				Organization:           "Acme Corp",
				ProductEdition:         "enterprise",
				ServerId:               "ABC123",
				Type:                   "PRODUCTION",
			},
			want: v1alpha1.LicenseObservation{
				ContactEmail:           "admin@acme.com",
				ExpiresAt:              "2027-01-01",
				IsExpired:              false,
				IsOfficialDistribution: true,
				IsSupported:            true,
				IsValidEdition:         true,
				IsValidServerId:        true,
				LOCsMax:                1000000,
				LOCsRemaining:          500000,
				Organization:           "Acme Corp",
				ProductEdition:         "enterprise",
				ServerId:               "ABC123",
				Type:                   "PRODUCTION",
			},
		},
		"ZeroLicense": {
			license: &sonar.License{},
			want:    v1alpha1.LicenseObservation{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateLicenseObservation(tc.license)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateLicenseObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLateInitializeLicense tests all branches of LateInitializeLicense.
func TestLateInitializeLicense(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.LicenseParameters{LicenseKeySecretRef: &xpv1.SecretKeySelector{Key: "license"}}
	specCopy := spec.DeepCopy()
	obs := &v1alpha1.LicenseObservation{ProductEdition: "enterprise"}

	cases := map[string]struct {
		spec        *v1alpha1.LicenseParameters
		observation *v1alpha1.LicenseObservation
	}{
		"NilSpec":        {spec: nil, observation: obs},
		"NilObservation": {spec: spec, observation: nil},
		"BothNil":        {spec: nil, observation: nil},
		"NoOp":           {spec: spec, observation: obs},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			LateInitializeLicense(tc.spec, tc.observation)
		})
	}

	// Verify spec is unchanged after no-op call.
	if diff := cmp.Diff(specCopy, spec); diff != "" {
		t.Errorf("LateInitializeLicense() mutated spec (-want +got):\n%s", diff)
	}
}

// TestIsLicenseLateInitialized tests all branches of IsLicenseLateInitialized.
func TestIsLicenseLateInitialized(t *testing.T) {
	t.Parallel()

	ref := &xpv1.SecretKeySelector{Key: "license"}
	otherRef := &xpv1.SecretKeySelector{Key: "other"}
	cases := map[string]struct {
		former  *v1alpha1.LicenseParameters
		current *v1alpha1.LicenseParameters
		want    bool
	}{
		"NilFormer":  {former: nil, current: &v1alpha1.LicenseParameters{LicenseKeySecretRef: ref}, want: false},
		"NilCurrent": {former: &v1alpha1.LicenseParameters{LicenseKeySecretRef: ref}, current: nil, want: false},
		"BothNil":    {former: nil, current: nil, want: false},
		"Equal":      {former: &v1alpha1.LicenseParameters{LicenseKeySecretRef: ref}, current: &v1alpha1.LicenseParameters{LicenseKeySecretRef: ref}, want: false},
		"Different":  {former: &v1alpha1.LicenseParameters{LicenseKeySecretRef: ref}, current: &v1alpha1.LicenseParameters{LicenseKeySecretRef: otherRef}, want: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsLicenseLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsLicenseLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsLicenseUpToDate tests all branches of IsLicenseUpToDate.
func TestIsLicenseUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		desiredLicenseKey string
		savedLicenseKey   string
		want              bool
	}{
		"UpToDate": {
			desiredLicenseKey: "key-123",
			savedLicenseKey:   "key-123",
			want:              true,
		},
		"KeyChanged": {
			desiredLicenseKey: "new-key",
			savedLicenseKey:   "old-key",
			want:              false,
		},
		"NeverApplied": {
			desiredLicenseKey: "key-123",
			savedLicenseKey:   "",
			want:              false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsLicenseUpToDate(tc.desiredLicenseKey, tc.savedLicenseKey)
			if got != tc.want {
				t.Errorf("IsLicenseUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
