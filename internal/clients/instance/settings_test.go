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
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

func TestGenerateSettingSetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params    v1alpha1.SettingParameters
		component *string
		want      *sonar.SettingsSetOption
	}{
		"BasicSetOptionWithValueOnly": {
			params: v1alpha1.SettingParameters{
				Key:   "sonar.core.serverBaseURL",
				Value: ptr.To("https://sonarqube.example.com"),
			},
			component: nil,
			want: &sonar.SettingsSetOption{
				Key:   "sonar.core.serverBaseURL",
				Value: "https://sonarqube.example.com",
			},
		},
		"SetOptionWithComponent": {
			params: v1alpha1.SettingParameters{
				Key:   "sonar.coverage.jacoco.xmlReportPaths",
				Value: ptr.To("target/site/jacoco/jacoco.xml"),
			},
			component: ptr.To("my-project-key"),
			want: &sonar.SettingsSetOption{
				Key:       "sonar.coverage.jacoco.xmlReportPaths",
				Value:     "target/site/jacoco/jacoco.xml",
				Component: "my-project-key",
			},
		},
		"SetOptionWithValues": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.exclusions",
				Values: ptr.To([]string{"**/*.test.js", "**/*.spec.js"}),
			},
			component: nil,
			want: &sonar.SettingsSetOption{
				Key:    "sonar.exclusions",
				Values: []string{"**/*.test.js", "**/*.spec.js"},
			},
		},
		"SetOptionWithFieldValues": {
			params: v1alpha1.SettingParameters{
				Key: "sonar.issue.enforce.multicriteria",
				FieldValues: ptr.To(map[string]string{
					"1.ruleKey":         "squid:S1134",
					"1.resourceKey":     "**/*.java",
					"1.enforceProperty": "severity",
					"1.enforceValue":    "CRITICAL",
				}),
			},
			component: nil,
			want: &sonar.SettingsSetOption{
				Key: "sonar.issue.enforce.multicriteria",
				FieldValues: sonar.JSONEncodedMap{
					"1.ruleKey":         "squid:S1134",
					"1.resourceKey":     "**/*.java",
					"1.enforceProperty": "severity",
					"1.enforceValue":    "CRITICAL",
				},
			},
		},
		"SetOptionWithEmptyValues": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.test.key",
				Values: ptr.To([]string{}),
			},
			component: nil,
			want: &sonar.SettingsSetOption{
				Key: "sonar.test.key",
			},
		},
		"SetOptionWithEmptyFieldValues": {
			params: v1alpha1.SettingParameters{
				Key:         "sonar.test.key",
				FieldValues: ptr.To(map[string]string{}),
			},
			component: nil,
			want: &sonar.SettingsSetOption{
				Key: "sonar.test.key",
			},
		},
		"SetOptionWithAllFields": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.complex.setting",
				Value:  ptr.To("base-value"),
				Values: ptr.To([]string{"value1", "value2"}),
				FieldValues: ptr.To(map[string]string{
					"field1": "val1",
					"field2": "val2",
				}),
			},
			component: ptr.To("project-key"),
			want: &sonar.SettingsSetOption{
				Key:       "sonar.complex.setting",
				Value:     "base-value",
				Values:    []string{"value1", "value2"},
				Component: "project-key",
				FieldValues: sonar.JSONEncodedMap{
					"field1": "val1",
					"field2": "val2",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingSetOptions(tc.params, tc.component)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSettingSetOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateSettingsValuesOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params *v1alpha1.SettingsParameters
		want   *sonar.SettingsValuesOption
	}{
		"BasicValuesOption": {
			params: &v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
				},
			},
			want: &sonar.SettingsValuesOption{
				Keys: []string{"sonar.core.serverBaseURL"},
			},
		},
		"ValuesOptionWithComponent": {
			params: &v1alpha1.SettingsParameters{
				Component: ptr.To("my-project-key"),
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.coverage.jacoco.xmlReportPaths": {
						Key:   "sonar.coverage.jacoco.xmlReportPaths",
						Value: ptr.To("target/site/jacoco/jacoco.xml"),
					},
				},
			},
			want: &sonar.SettingsValuesOption{
				Keys:      []string{"sonar.coverage.jacoco.xmlReportPaths"},
				Component: "my-project-key",
			},
		},
		"ValuesOptionWithMultipleSettings": {
			params: &v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
					"sonar.exclusions": {
						Key:    "sonar.exclusions",
						Values: ptr.To([]string{"**/*.test.js"}),
					},
					"sonar.coverage.exclusions": {
						Key:    "sonar.coverage.exclusions",
						Values: ptr.To([]string{"**/*.spec.js"}),
					},
				},
			},
			want: &sonar.SettingsValuesOption{
				Keys: []string{"sonar.core.serverBaseURL", "sonar.exclusions", "sonar.coverage.exclusions"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingsValuesOptions(tc.params)
			// Note: The order of keys in a map is not guaranteed, so we need to sort them before comparing
			if len(got.Keys) != len(tc.want.Keys) {
				t.Errorf("GenerateSettingsValuesOptions() keys length mismatch: got %d, want %d", len(got.Keys), len(tc.want.Keys))
			}
			// Check if all expected keys are present
			expectedKeys := make(map[string]bool)
			for _, k := range tc.want.Keys {
				expectedKeys[k] = true
			}

			for _, k := range got.Keys {
				if !expectedKeys[k] {
					t.Errorf("GenerateSettingsValuesOptions() unexpected key: %s", k)
				}

				delete(expectedKeys, k)
			}

			if len(expectedKeys) > 0 {
				t.Errorf("GenerateSettingsValuesOptions() missing keys: %v", expectedKeys)
			}
			// Check component
			if diff := cmp.Diff(tc.want.Component, got.Component); diff != "" {
				t.Errorf("GenerateSettingsValuesOptions() component mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateSettingsResetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params v1alpha1.SettingsParameters
		want   *sonar.SettingsResetOption
	}{
		"BasicResetOption": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
				},
			},
			want: &sonar.SettingsResetOption{
				Keys: []string{"sonar.core.serverBaseURL"},
			},
		},
		"ResetOptionWithComponent": {
			params: v1alpha1.SettingsParameters{
				Component: ptr.To("my-project-key"),
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.coverage.jacoco.xmlReportPaths": {
						Key:   "sonar.coverage.jacoco.xmlReportPaths",
						Value: ptr.To("target/site/jacoco/jacoco.xml"),
					},
				},
			},
			want: &sonar.SettingsResetOption{
				Keys:      []string{"sonar.coverage.jacoco.xmlReportPaths"},
				Component: "my-project-key",
			},
		},
		"ResetOptionWithMultipleSettings": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
					"sonar.exclusions": {
						Key:    "sonar.exclusions",
						Values: ptr.To([]string{"**/*.test.js"}),
					},
				},
			},
			want: &sonar.SettingsResetOption{
				Keys: []string{"sonar.core.serverBaseURL", "sonar.exclusions"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingsResetOptions(tc.params)
			// Check keys length
			if len(got.Keys) != len(tc.want.Keys) {
				t.Errorf("GenerateSettingsResetOptions() keys length mismatch: got %d, want %d", len(got.Keys), len(tc.want.Keys))
			}
			// Check if all expected keys are present
			expectedKeys := make(map[string]bool)
			for _, k := range tc.want.Keys {
				expectedKeys[k] = true
			}

			for _, k := range got.Keys {
				if !expectedKeys[k] {
					t.Errorf("GenerateSettingsResetOptions() unexpected key: %s", k)
				}

				delete(expectedKeys, k)
			}

			if len(expectedKeys) > 0 {
				t.Errorf("GenerateSettingsResetOptions() missing keys: %v", expectedKeys)
			}
			// Check component
			if diff := cmp.Diff(tc.want.Component, got.Component); diff != "" {
				t.Errorf("GenerateSettingsResetOptions() component mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateSettingsResetOptionsFromList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		keys      []string
		component *string
		want      *sonar.SettingsResetOption
	}{
		"BasicResetFromList": {
			keys:      []string{"sonar.core.serverBaseURL"},
			component: nil,
			want: &sonar.SettingsResetOption{
				Keys: []string{"sonar.core.serverBaseURL"},
			},
		},
		"ResetFromListWithComponent": {
			keys:      []string{"sonar.coverage.jacoco.xmlReportPaths"},
			component: ptr.To("my-project-key"),
			want: &sonar.SettingsResetOption{
				Keys:      []string{"sonar.coverage.jacoco.xmlReportPaths"},
				Component: "my-project-key",
			},
		},
		"ResetFromListWithMultipleKeys": {
			keys:      []string{"sonar.core.serverBaseURL", "sonar.exclusions", "sonar.coverage.exclusions"},
			component: ptr.To("another-project"),
			want: &sonar.SettingsResetOption{
				Keys:      []string{"sonar.core.serverBaseURL", "sonar.exclusions", "sonar.coverage.exclusions"},
				Component: "another-project",
			},
		},
		"ResetFromListWithEmptyKeys": {
			keys:      []string{},
			component: nil,
			want: &sonar.SettingsResetOption{
				Keys: []string{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingsResetOptionsFromList(tc.keys, tc.component)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSettingsResetOptionsFromList() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateSettingObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		observed *sonar.SettingValue
		want     v1alpha1.SettingObservation
	}{
		"ObservationWithValueOnly": {
			observed: &sonar.SettingValue{
				Key:   "sonar.core.serverBaseURL",
				Value: "https://sonarqube.example.com",
			},
			want: v1alpha1.SettingObservation{
				Value:       "https://sonarqube.example.com",
				Values:      nil,
				FieldValues: map[string]string{},
			},
		},
		"ObservationWithValues": {
			observed: &sonar.SettingValue{
				Key:    "sonar.exclusions",
				Values: []string{"**/*.test.js", "**/*.spec.js"},
			},
			want: v1alpha1.SettingObservation{
				Value:       "",
				Values:      []string{"**/*.test.js", "**/*.spec.js"},
				FieldValues: map[string]string{},
			},
		},
		"ObservationWithFieldValues": {
			observed: &sonar.SettingValue{
				Key: "sonar.issue.enforce.multicriteria",
				FieldValues: []map[string]string{
					{
						"1.ruleKey":         "squid:S1134",
						"1.resourceKey":     "**/*.java",
						"1.enforceProperty": "severity",
						"1.enforceValue":    "CRITICAL",
					},
				},
			},
			want: v1alpha1.SettingObservation{
				Value:  "",
				Values: nil,
				FieldValues: map[string]string{
					"1.ruleKey":         "squid:S1134",
					"1.resourceKey":     "**/*.java",
					"1.enforceProperty": "severity",
					"1.enforceValue":    "CRITICAL",
				},
			},
		},
		"ObservationWithMultipleFieldValues": {
			observed: &sonar.SettingValue{
				Key: "sonar.multi.field",
				FieldValues: []map[string]string{
					{
						"1.key1": "value1",
						"1.key2": "value2",
					},
					{
						"2.key1": "value3",
						"2.key2": "value4",
					},
				},
			},
			want: v1alpha1.SettingObservation{
				Value:  "",
				Values: nil,
				FieldValues: map[string]string{
					"1.key1": "value1",
					"1.key2": "value2",
					"2.key1": "value3",
					"2.key2": "value4",
				},
			},
		},
		"ObservationWithEmptyFieldValues": {
			observed: &sonar.SettingValue{
				Key:         "sonar.empty.field",
				FieldValues: []map[string]string{},
			},
			want: v1alpha1.SettingObservation{
				Value:       "",
				Values:      nil,
				FieldValues: map[string]string{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingObservation(tc.observed)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSettingObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateSettingsObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		observed *sonar.SettingsValues
		want     v1alpha1.SettingsObservation
	}{
		"ObservationWithSingleSetting": {
			observed: &sonar.SettingsValues{
				Settings: []sonar.SettingValue{
					{
						Key:   "sonar.core.serverBaseURL",
						Value: "https://sonarqube.example.com",
					},
				},
			},
			want: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value:       "https://sonarqube.example.com",
						Values:      nil,
						FieldValues: map[string]string{},
					},
				},
			},
		},
		"ObservationWithMultipleSettings": {
			observed: &sonar.SettingsValues{
				Settings: []sonar.SettingValue{
					{
						Key:   "sonar.core.serverBaseURL",
						Value: "https://sonarqube.example.com",
					},
					{
						Key:    "sonar.exclusions",
						Values: []string{"**/*.test.js", "**/*.spec.js"},
					},
				},
			},
			want: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value:       "https://sonarqube.example.com",
						Values:      nil,
						FieldValues: map[string]string{},
					},
					"sonar.exclusions": {
						Value:       "",
						Values:      []string{"**/*.test.js", "**/*.spec.js"},
						FieldValues: map[string]string{},
					},
				},
			},
		},
		"ObservationWithEmptySettings": {
			observed: &sonar.SettingsValues{
				Settings: []sonar.SettingValue{},
			},
			want: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateSettingsObservation(tc.observed)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateSettingsObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsSettingUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params      v1alpha1.SettingParameters
		observation v1alpha1.SettingObservation
		want        bool
	}{
		"MatchingValue": {
			params: v1alpha1.SettingParameters{
				Key:   "sonar.core.serverBaseURL",
				Value: ptr.To("https://sonarqube.example.com"),
			},
			observation: v1alpha1.SettingObservation{
				Value: "https://sonarqube.example.com",
			},
			want: true,
		},
		"DifferentValue": {
			params: v1alpha1.SettingParameters{
				Key:   "sonar.core.serverBaseURL",
				Value: ptr.To("https://sonarqube.example.com"),
			},
			observation: v1alpha1.SettingObservation{
				Value: "https://different-url.com",
			},
			want: false,
		},
		"NilValueMatchesAnything": {
			params: v1alpha1.SettingParameters{
				Key:   "sonar.core.serverBaseURL",
				Value: nil,
			},
			observation: v1alpha1.SettingObservation{
				Value: "https://sonarqube.example.com",
			},
			want: true,
		},
		"MatchingValues": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.exclusions",
				Values: ptr.To([]string{"**/*.test.js", "**/*.spec.js"}),
			},
			observation: v1alpha1.SettingObservation{
				Values: []string{"**/*.test.js", "**/*.spec.js"},
			},
			want: true,
		},
		"DifferentValues": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.exclusions",
				Values: ptr.To([]string{"**/*.test.js", "**/*.spec.js"}),
			},
			observation: v1alpha1.SettingObservation{
				Values: []string{"**/*.test.js"},
			},
			want: false,
		},
		"NilValuesMatchesAnything": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.exclusions",
				Values: nil,
			},
			observation: v1alpha1.SettingObservation{
				Values: []string{"**/*.test.js", "**/*.spec.js"},
			},
			want: true,
		},
		"MatchingFieldValues": {
			params: v1alpha1.SettingParameters{
				Key: "sonar.issue.enforce.multicriteria",
				FieldValues: ptr.To(map[string]string{
					"1.ruleKey":     "squid:S1134",
					"1.resourceKey": "**/*.java",
				}),
			},
			observation: v1alpha1.SettingObservation{
				FieldValues: map[string]string{
					"1.ruleKey":     "squid:S1134",
					"1.resourceKey": "**/*.java",
				},
			},
			want: true,
		},
		"DifferentFieldValues": {
			params: v1alpha1.SettingParameters{
				Key: "sonar.issue.enforce.multicriteria",
				FieldValues: ptr.To(map[string]string{
					"1.ruleKey":     "squid:S1134",
					"1.resourceKey": "**/*.java",
				}),
			},
			observation: v1alpha1.SettingObservation{
				FieldValues: map[string]string{
					"1.ruleKey": "squid:S1134",
				},
			},
			want: false,
		},
		"NilFieldValuesMatchesAnything": {
			params: v1alpha1.SettingParameters{
				Key:         "sonar.issue.enforce.multicriteria",
				FieldValues: nil,
			},
			observation: v1alpha1.SettingObservation{
				FieldValues: map[string]string{
					"1.ruleKey":     "squid:S1134",
					"1.resourceKey": "**/*.java",
				},
			},
			want: true,
		},
		"EmptyObservedValuesMatchesNilParams": {
			params: v1alpha1.SettingParameters{
				Key:    "sonar.exclusions",
				Values: nil,
			},
			observation: v1alpha1.SettingObservation{
				Values: []string{},
			},
			want: true,
		},
		"EmptyObservedFieldValuesMatchesNilParams": {
			params: v1alpha1.SettingParameters{
				Key:         "sonar.field.setting",
				FieldValues: nil,
			},
			observation: v1alpha1.SettingObservation{
				FieldValues: map[string]string{},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsSettingUpToDate(tc.params, tc.observation)
			if got != tc.want {
				t.Errorf("IsSettingUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAreSettingsUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params      v1alpha1.SettingsParameters
		observation v1alpha1.SettingsObservation
		want        bool
	}{
		"AllSettingsUpToDate": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
					"sonar.exclusions": {
						Key:    "sonar.exclusions",
						Values: ptr.To([]string{"**/*.test.js"}),
					},
				},
			},
			observation: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value: "https://sonarqube.example.com",
					},
					"sonar.exclusions": {
						Values: []string{"**/*.test.js"},
					},
				},
			},
			want: true,
		},
		"OneSettingNotUpToDate": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
					"sonar.exclusions": {
						Key:    "sonar.exclusions",
						Values: ptr.To([]string{"**/*.test.js"}),
					},
				},
			},
			observation: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value: "https://different-url.com",
					},
					"sonar.exclusions": {
						Values: []string{"**/*.test.js"},
					},
				},
			},
			want: false,
		},
		"SettingMissingFromObservation": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
					"sonar.exclusions": {
						Key:    "sonar.exclusions",
						Values: ptr.To([]string{"**/*.test.js"}),
					},
				},
			},
			observation: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value: "https://sonarqube.example.com",
					},
				},
			},
			want: false,
		},
		"EmptySettingsAreUpToDate": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{},
			},
			observation: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{},
			},
			want: true,
		},
		"ExtraObservedSettingsDoNotAffectUpToDate": {
			params: v1alpha1.SettingsParameters{
				Settings: map[string]v1alpha1.SettingParameters{
					"sonar.core.serverBaseURL": {
						Key:   "sonar.core.serverBaseURL",
						Value: ptr.To("https://sonarqube.example.com"),
					},
				},
			},
			observation: v1alpha1.SettingsObservation{
				Settings: map[string]v1alpha1.SettingObservation{
					"sonar.core.serverBaseURL": {
						Value: "https://sonarqube.example.com",
					},
					"sonar.extra.setting": {
						Value: "extra-value",
					},
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreSettingsUpToDate(tc.params, tc.observation)
			if got != tc.want {
				t.Errorf("AreSettingsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
