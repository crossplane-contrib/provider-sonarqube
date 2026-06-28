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

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestGeneratePortfolioObservation tests the
// GeneratePortfolioObservation function.
func TestGeneratePortfolioObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		details *sonar.ViewDetails
		want    v1alpha1.PortfolioObservation
	}{
		"NilDetails": {
			details: nil,
			want:    v1alpha1.PortfolioObservation{},
		},
		"EmptyDetails": {
			details: &sonar.ViewDetails{},
			want:    v1alpha1.PortfolioObservation{},
		},
		"FullDetails": {
			details: &sonar.ViewDetails{
				Key:           "my-portfolio",
				Name:          "My Portfolio",
				Description:   "A test portfolio",
				Qualifier:     "VW",
				Visibility:    "public",
				SelectionMode: "MANUAL",
			},
			want: v1alpha1.PortfolioObservation{
				Key:           "my-portfolio",
				Name:          "My Portfolio",
				Description:   "A test portfolio",
				Qualifier:     "VW",
				Visibility:    "public",
				SelectionMode: "MANUAL",
			},
		},
		"PartialDetails": {
			details: &sonar.ViewDetails{
				Key:  "partial",
				Name: "Partial Portfolio",
			},
			want: v1alpha1.PortfolioObservation{
				Key:  "partial",
				Name: "Partial Portfolio",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GeneratePortfolioObservation(tc.details)

			if got.Key != tc.want.Key {
				t.Errorf("GeneratePortfolioObservation() Key = %q, want %q", got.Key, tc.want.Key)
			}

			if got.Name != tc.want.Name {
				t.Errorf("GeneratePortfolioObservation() Name = %q, want %q", got.Name, tc.want.Name)
			}

			if got.Description != tc.want.Description {
				t.Errorf("GeneratePortfolioObservation() Description = %q, want %q", got.Description, tc.want.Description)
			}

			if got.Qualifier != tc.want.Qualifier {
				t.Errorf("GeneratePortfolioObservation() Qualifier = %q, want %q", got.Qualifier, tc.want.Qualifier)
			}

			if got.Visibility != tc.want.Visibility {
				t.Errorf("GeneratePortfolioObservation() Visibility = %q, want %q", got.Visibility, tc.want.Visibility)
			}

			if got.SelectionMode != tc.want.SelectionMode {
				t.Errorf("GeneratePortfolioObservation() SelectionMode = %q, want %q", got.SelectionMode, tc.want.SelectionMode)
			}
		})
	}
}

// TestLateInitializePortfolio tests the LateInitializePortfolio function.
func TestLateInitializePortfolio(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PortfolioParameters
		observation *v1alpha1.PortfolioObservation
	}{
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.PortfolioObservation{Name: "obs"},
		},
		"NilObservation": {
			spec:        &v1alpha1.PortfolioParameters{Name: "spec"},
			observation: nil,
		},
		"BothNil": {
			spec:        nil,
			observation: nil,
		},
		"NoOpWithValues": {
			spec:        &v1alpha1.PortfolioParameters{Key: "k", Name: "n"},
			observation: &v1alpha1.PortfolioObservation{Key: "k", Name: "n"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specCopy := tc.spec
			if tc.spec != nil {
				specVal := *tc.spec
				specCopy = &specVal
			}

			LateInitializePortfolio(tc.spec, tc.observation)

			if tc.spec != nil && *tc.spec != *specCopy {
				t.Errorf("LateInitializePortfolio() modified spec unexpectedly: got %+v, want %+v", *tc.spec, *specCopy)
			}
		})
	}
}

// TestIsPortfolioLateInitialized tests the IsPortfolioLateInitialized function.
func TestIsPortfolioLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.PortfolioParameters
		current *v1alpha1.PortfolioParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.PortfolioParameters{Name: "n"},
			want:    false,
		},
		"NilCurrent": {
			former:  &v1alpha1.PortfolioParameters{Name: "n"},
			current: nil,
			want:    false,
		},
		"BothNil": {
			former:  nil,
			current: nil,
			want:    false,
		},
		"EqualParameters": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			want:    false,
		},
		"DifferentName": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "old"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "new"},
			want:    true,
		},
		"DifferentSelectionMode": {
			former:  &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "NONE"},
			current: &v1alpha1.PortfolioParameters{Key: "k", Name: "n", SelectionMode: "MANUAL"},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPortfolioLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsPortfolioLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsPortfolioUpToDate tests the IsPortfolioUpToDate function.
func TestIsPortfolioUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PortfolioParameters
		observation *v1alpha1.PortfolioObservation
		want        bool
	}{
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.PortfolioObservation{Name: "n"},
			want:        true,
		},
		"NilObservation": {
			spec:        &v1alpha1.PortfolioParameters{Name: "n"},
			observation: nil,
			want:        false,
		},
		"UpToDateAllFieldsMatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				Description:   "desc",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				Description:   "desc",
				SelectionMode: "NONE",
			},
			want: true,
		},
		"SelectionModeNormalizedEmptyEqualsNONE": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "NONE",
			},
			want: true,
		},
		"SelectionModeNormalizedBothEmpty": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "",
			},
			want: true,
		},
		"NameMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "New Name",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "Old Name",
				SelectionMode: "NONE",
			},
			want: false,
		},
		"DescriptionMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				Description:   "new desc",
				SelectionMode: "NONE",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				Description:   "old desc",
				SelectionMode: "NONE",
			},
			want: false,
		},
		"SelectionModeMismatch": {
			spec: &v1alpha1.PortfolioParameters{
				Name:          "My Portfolio",
				SelectionMode: "MANUAL",
			},
			observation: &v1alpha1.PortfolioObservation{
				Name:          "My Portfolio",
				SelectionMode: "NONE",
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPortfolioUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsPortfolioUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNormalizeSelectionMode tests the normalizeSelectionMode function.
func TestNormalizeSelectionMode(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mode string
		want string
	}{
		"EmptyBecomesNONE":   {mode: "", want: "NONE"},
		"NONEUnchanged":      {mode: "NONE", want: "NONE"},
		"MANUALUnchanged":    {mode: "MANUAL", want: "MANUAL"},
		"REGEXPUnchanged":    {mode: "REGEXP", want: "REGEXP"},
		"REMAININGUnchanged": {mode: "REMAINING", want: "REMAINING"},
		"TAGSUnchanged":      {mode: "TAGS", want: "TAGS"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := normalizeSelectionMode(tc.mode)
			if got != tc.want {
				t.Errorf("normalizeSelectionMode(%q) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}
