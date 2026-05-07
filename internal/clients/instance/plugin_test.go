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
	"time"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestGeneratePluginObservation tests generating a plugin observation.
func TestGeneratePluginObservation(t *testing.T) {
	t.Parallel()

	// testTimestampMs is an epoch timestamp in milliseconds.
	testTimestampMs := int64(1_700_000_000_000) // 2023-11-14 ~22:13 UTC
	testMetaTime := metav1.NewTime(time.Unix(testTimestampMs/1000, 0).UTC())

	cases := map[string]struct {
		installed *sonar.PluginInstalled
		want      v1alpha1.PluginObservation
	}{
		"NilReturnsEmpty": {
			installed: nil,
			want:      v1alpha1.PluginObservation{},
		},
		"AllFieldsMapped": {
			installed: &sonar.PluginInstalled{
				Key:                  "findbugs",
				Name:                 "FindBugs",
				Version:              "4.0.0",
				Description:          "Find bugs in Java code",
				Filename:             "findbugs-4.0.0.jar",
				EditionBundled:       true,
				Hash:                 "abc123",
				HomepageURL:          "https://findbugs.example.com",
				ImplementationBuild:  "build-42",
				IssueTrackerURL:      "https://issues.example.com",
				License:              "LGPL",
				OrganizationName:     "SpotBugs",
				OrganizationURL:      "https://spotbugs.example.com",
				RequiredForLanguages: []string{"java", "kotlin"},
				SonarLintSupported:   true,
				UpdatedAt:            testTimestampMs,
			},
			want: v1alpha1.PluginObservation{
				Key:                  "findbugs",
				Name:                 "FindBugs",
				Version:              "4.0.0",
				Description:          "Find bugs in Java code",
				Filename:             "findbugs-4.0.0.jar",
				EditionBundled:       true,
				Hash:                 "abc123",
				HomepageURL:          "https://findbugs.example.com",
				ImplementationBuild:  "build-42",
				IssueTrackerURL:      "https://issues.example.com",
				License:              "LGPL",
				OrganizationName:     "SpotBugs",
				OrganizationURL:      "https://spotbugs.example.com",
				RequiredForLanguages: []string{"java", "kotlin"},
				SonarLintSupported:   true,
				UpdatedAt:            &testMetaTime,
			},
		},
		"ZeroTimestampProducesNilTime": {
			installed: &sonar.PluginInstalled{
				Key:       "checkstyle",
				Version:   "10.12.0",
				UpdatedAt: 0,
			},
			want: v1alpha1.PluginObservation{
				Key:     "checkstyle",
				Version: "10.12.0",
			},
		},
		"FalseBooleansPreserved": {
			installed: &sonar.PluginInstalled{
				Key:                "pmd",
				EditionBundled:     false,
				SonarLintSupported: false,
			},
			want: v1alpha1.PluginObservation{
				Key: "pmd",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GeneratePluginObservation(tc.installed)
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("GeneratePluginObservation() -want +got:\n%s", diff)
			}
		})
	}
}

// TestFindInstalledPlugin tests finding an installed plugin by key.
func TestFindInstalledPlugin(t *testing.T) {
	t.Parallel()

	twoPlugins := []sonar.PluginInstalled{
		{Key: "findbugs", Name: "FindBugs"},
		{Key: "checkstyle", Name: "Checkstyle"},
	}

	cases := map[string]struct {
		plugins []sonar.PluginInstalled
		key     string
		wantNil bool
		wantKey string
	}{
		"EmptyListReturnsNil": {
			plugins: []sonar.PluginInstalled{},
			key:     "findbugs",
			wantNil: true,
		},
		"KeyAbsentReturnsNil": {
			plugins: twoPlugins,
			key:     "pmd",
			wantNil: true,
		},
		"FirstKeyFound": {
			plugins: twoPlugins,
			key:     "findbugs",
			wantKey: "findbugs",
		},
		"SecondKeyFound": {
			plugins: twoPlugins,
			key:     "checkstyle",
			wantKey: "checkstyle",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindInstalledPlugin(tc.plugins, tc.key)
			if tc.wantNil {
				if got != nil {
					t.Errorf("FindInstalledPlugin() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("FindInstalledPlugin() = nil, want non-nil")
			}

			if got.Key != tc.wantKey {
				t.Errorf("FindInstalledPlugin() key = %q, want %q", got.Key, tc.wantKey)
			}
		})
	}
}

// TestIsPluginUpdatable tests checking if a plugin has available updates.
func TestIsPluginUpdatable(t *testing.T) {
	t.Parallel()

	twoUpdatable := []sonar.PluginWithUpdates{
		{Key: "findbugs"},
		{Key: "checkstyle"},
	}

	cases := map[string]struct {
		plugins *[]sonar.PluginWithUpdates
		key     string
		want    bool
	}{
		"NilPluginsReturnsFalse": {
			plugins: nil,
			key:     "findbugs",
			want:    false,
		},
		"EmptyListReturnsFalse": {
			plugins: &[]sonar.PluginWithUpdates{},
			key:     "findbugs",
			want:    false,
		},
		"KeyAbsentReturnsFalse": {
			plugins: &twoUpdatable,
			key:     "pmd",
			want:    false,
		},
		"FirstKeyFound": {
			plugins: &twoUpdatable,
			key:     "findbugs",
			want:    true,
		},
		"SecondKeyFound": {
			plugins: &twoUpdatable,
			key:     "checkstyle",
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPluginUpdatable(tc.plugins, tc.key)
			if got != tc.want {
				t.Errorf("IsPluginUpdatable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestLateInitializePlugin tests that LateInitializePlugin is a no-op.
func TestLateInitializePlugin(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PluginParameters
		observation *v1alpha1.PluginObservation
		wantSpec    *v1alpha1.PluginParameters
	}{
		"NilSpecIsNoOp": {
			spec:        nil,
			observation: &v1alpha1.PluginObservation{Name: "FindBugs"},
			wantSpec:    nil,
		},
		"NilObservationIsNoOp": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs"},
			observation: nil,
			wantSpec:    &v1alpha1.PluginParameters{Key: "findbugs"},
		},
		"NoFieldsModified": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(false)},
			observation: &v1alpha1.PluginObservation{Name: "FindBugs", Version: "4.0.0"},
			wantSpec:    &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(false)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializePlugin(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantSpec, tc.spec); diff != "" {
				t.Errorf("LateInitializePlugin() spec -want +got:\n%s", diff)
			}
		})
	}
}

// TestIsPluginLateInitialized tests checking for late-initialized fields.
func TestIsPluginLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.PluginParameters
		current *v1alpha1.PluginParameters
		want    bool
	}{
		"BothNilReturnsTrue": {
			former:  nil,
			current: nil,
			want:    true,
		},
		"FormerNilReturnsTrue": {
			former:  nil,
			current: &v1alpha1.PluginParameters{Key: "findbugs"},
			want:    true,
		},
		"CurrentNilReturnsTrue": {
			former:  &v1alpha1.PluginParameters{Key: "findbugs"},
			current: nil,
			want:    true,
		},
		"EqualSpecsReturnTrue": {
			former:  &v1alpha1.PluginParameters{Key: "findbugs"},
			current: &v1alpha1.PluginParameters{Key: "findbugs"},
			want:    true,
		},
		"EqualSpecsWithAutoUpdateReturnTrue": {
			former:  &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(true)},
			current: &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(true)},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPluginLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsPluginLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsPluginUpToDate tests checking if a plugin spec is up to date.
func TestIsPluginUpToDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.PluginParameters
		observation *v1alpha1.PluginObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: &v1alpha1.PluginObservation{IsLatest: false},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs"},
			observation: nil,
			want:        false,
		},
		"IsLatestTrueReturnsTrue": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs"},
			observation: &v1alpha1.PluginObservation{IsLatest: true},
			want:        true,
		},
		"IsLatestFalseReturnsFalse": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs"},
			observation: &v1alpha1.PluginObservation{IsLatest: false},
			want:        false,
		},
		"AutoUpdateFalseAlwaysReturnsTrue": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(false)},
			observation: &v1alpha1.PluginObservation{IsLatest: false},
			want:        true,
		},
		"AutoUpdateTrueFollowsIsLatest": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: new(true)},
			observation: &v1alpha1.PluginObservation{IsLatest: false},
			want:        false,
		},
		"AutoUpdateNilFollowsIsLatest": {
			spec:        &v1alpha1.PluginParameters{Key: "findbugs", AutoUpdate: nil},
			observation: &v1alpha1.PluginObservation{IsLatest: false},
			want:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsPluginUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsPluginUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
