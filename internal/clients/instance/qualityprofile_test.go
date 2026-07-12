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

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestGenerateCreateQualityProfileOption tests creation options.
func TestGenerateCreateQualityProfileOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params v1alpha1.QualityProfileParameters
		want   *sonar.QualityprofilesCreateOptions
	}{
		"BasicProfile": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			want: &sonar.QualityprofilesCreateOptions{
				Name:     "my-profile",
				Language: "java",
			},
		},
		"ProfileWithDefault": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "default-profile",
				Language: "go",
				Default:  new(true),
			},
			want: &sonar.QualityprofilesCreateOptions{
				Name:     "default-profile",
				Language: "go",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateCreateQualityProfileOption(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateCreateQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateDeleteQualityProfileOption tests deletion options.
func TestGenerateDeleteQualityProfileOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params v1alpha1.QualityProfileParameters
		want   *sonar.QualityprofilesDeleteOptions
	}{
		"BasicDelete": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			want: &sonar.QualityprofilesDeleteOptions{
				QualityProfile: "my-profile",
				Language:       "java",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateDeleteQualityProfileOption(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateDeleteQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateRenameQualityProfileOption tests rename options.
func TestGenerateRenameQualityProfileOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key    string
		params v1alpha1.QualityProfileParameters
		want   *sonar.QualityprofilesRenameOptions
	}{
		"BasicRename": {
			key: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileParameters{
				Name:     "new-name",
				Language: "java",
			},
			want: &sonar.QualityprofilesRenameOptions{
				Key:  "AU-TpxcA-iU5OvuD2FLz",
				Name: "new-name",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRenameQualityProfileOption(tc.key, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateRenameQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsQualityProfileUpToDate tests quality profile up-to-date status.
func TestIsQualityProfileUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec         *v1alpha1.QualityProfileParameters
		observation  *v1alpha1.QualityProfileObservation
		associations map[string]QualityProfileRuleAssociation
		want         bool
	}{
		"NilSpec": {
			spec:         nil,
			observation:  &v1alpha1.QualityProfileObservation{},
			associations: map[string]QualityProfileRuleAssociation{},
			want:         true,
		},
		"NilObservation": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			observation:  nil,
			associations: map[string]QualityProfileRuleAssociation{},
			want:         false,
		},
		"NameMismatch": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "new-name",
				Language: "java",
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:     "old-name",
				Language: "java",
			},
			associations: map[string]QualityProfileRuleAssociation{},
			want:         false,
		},
		"LanguageMismatch": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "go",
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:     "my-profile",
				Language: "java",
			},
			associations: map[string]QualityProfileRuleAssociation{},
			want:         false,
		},
		"DefaultMismatch": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
				Default:  new(true),
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:      "my-profile",
				Language:  "java",
				IsDefault: false,
			},
			associations: map[string]QualityProfileRuleAssociation{},
			want:         false,
		},
		"AllUpToDate": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
				Default:  new(true),
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:      "my-profile",
				Language:  "java",
				IsDefault: true,
			},
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule: new("java:S1144"),
					},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key: "java:S1144",
					},
					UpToDate: true,
				},
			},
			want: true,
		},
		"RulesNotUpToDate": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:     "my-profile",
				Language: "java",
			},
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule:     new("java:S1144"),
						Severity: new("CRITICAL"),
					},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key:      "java:S1144",
						Severity: "MAJOR",
					},
					UpToDate: false,
				},
			},
			want: false,
		},
		"EmptyAssociations": {
			spec: &v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:     "my-profile",
				Language: "java",
			},
			associations: map[string]QualityProfileRuleAssociation{},
			want:         true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsQualityProfileUpToDate(tc.spec, tc.observation, tc.associations)
			if got != tc.want {
				t.Errorf("IsQualityProfileUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateQualityProfileRulesAssociation tests rule associations.
func TestGenerateQualityProfileRulesAssociation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specs        []v1alpha1.QualityProfileRuleParameters
		observations []v1alpha1.QualityProfileRuleObservation
		wantKeys     []string
		wantUpToDate map[string]bool
	}{
		"EmptySpecsAndObservations": {
			specs:        []v1alpha1.QualityProfileRuleParameters{},
			observations: []v1alpha1.QualityProfileRuleObservation{},
			wantKeys:     []string{},
			wantUpToDate: map[string]bool{},
		},
		"OnlySpecs": {
			specs: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
				{Rule: new("java:S1145")},
			},
			observations: []v1alpha1.QualityProfileRuleObservation{},
			wantKeys:     []string{"java:S1144", "java:S1145"},
			wantUpToDate: map[string]bool{
				"java:S1144": false,
				"java:S1145": false,
			},
		},
		"OnlyObservations": {
			specs: []v1alpha1.QualityProfileRuleParameters{},
			observations: []v1alpha1.QualityProfileRuleObservation{
				{Key: "java:S1144"},
				{Key: "java:S1145"},
			},
			wantKeys: []string{"java:S1144", "java:S1145"},
			wantUpToDate: map[string]bool{
				"java:S1144": false,
				"java:S1145": false,
			},
		},
		"MatchingSpecsAndObservations": {
			specs: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
			},
			observations: []v1alpha1.QualityProfileRuleObservation{
				{Key: "java:S1144"},
			},
			wantKeys: []string{"java:S1144"},
			wantUpToDate: map[string]bool{
				"java:S1144": true,
			},
		},
		"MixedSpecsAndObservations": {
			specs: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
				{Rule: new("java:S1146")}, // New rule to be activated
			},
			observations: []v1alpha1.QualityProfileRuleObservation{
				{Key: "java:S1144"},
				{Key: "java:S1145"}, // Rule to be deactivated
			},
			wantKeys: []string{"java:S1144", "java:S1145", "java:S1146"},
			wantUpToDate: map[string]bool{
				"java:S1144": true,  // Matching
				"java:S1145": false, // Only in observation (to deactivate)
				"java:S1146": false, // Only in spec (to activate)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRulesAssociation(tc.specs, tc.observations)

			// Check keys
			if len(got) != len(tc.wantKeys) {
				t.Errorf("GenerateQualityProfileRulesAssociation() returned %d keys, want %d", len(got), len(tc.wantKeys))
			}

			for _, key := range tc.wantKeys {
				if _, exists := got[key]; !exists {
					t.Errorf("GenerateQualityProfileRulesAssociation() missing key %s", key)
				}
			}

			// Check UpToDate values
			for key, wantUpToDate := range tc.wantUpToDate {
				if assoc, exists := got[key]; exists {
					if assoc.UpToDate != wantUpToDate {
						t.Errorf("GenerateQualityProfileRulesAssociation()[%s].UpToDate = %v, want %v", key, assoc.UpToDate, wantUpToDate)
					}
				}
			}
		})
	}
}

// TestFindNonExistingQualityProfileRules tests finding non-existing rules.
func TestFindNonExistingQualityProfileRules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		associations map[string]QualityProfileRuleAssociation
		wantCount    int
		wantRules    []string
	}{
		"Empty": {
			associations: map[string]QualityProfileRuleAssociation{},
			wantCount:    0,
			wantRules:    []string{},
		},
		"NoNonExisting": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
					UpToDate:    true,
				},
			},
			wantCount: 0,
			wantRules: []string{},
		},
		"SomeNonExisting": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144")},
					Observation: nil, // No observation means needs activation
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1145")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1145"},
					UpToDate:    true,
				},
			},
			wantCount: 1,
			wantRules: []string{"java:S1144"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindNonExistingQualityProfileRules(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNonExistingQualityProfileRules() returned %d rules, want %d", len(got), tc.wantCount)
			}

			for _, wantRule := range tc.wantRules {
				found := false

				for _, rule := range got {
					if ptr.Deref(rule.Rule, "") == wantRule {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("FindNonExistingQualityProfileRules() missing rule %s", wantRule)
				}
			}
		})
	}
}

// TestFindMissingQualityProfileRules tests finding missing rules.
func TestFindMissingQualityProfileRules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		associations map[string]QualityProfileRuleAssociation
		wantCount    int
		wantRules    []string
	}{
		"Empty": {
			associations: map[string]QualityProfileRuleAssociation{},
			wantCount:    0,
			wantRules:    []string{},
		},
		"NoMissing": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
					UpToDate:    true,
				},
			},
			wantCount: 0,
			wantRules: []string{},
		},
		"SomeMissing": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        nil, // No spec means needs deactivation
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1145")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1145"},
					UpToDate:    true,
				},
			},
			wantCount: 1,
			wantRules: []string{"java:S1144"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindMissingQualityProfileRules(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindMissingQualityProfileRules() returned %d rules, want %d", len(got), tc.wantCount)
			}

			for _, wantRule := range tc.wantRules {
				found := false

				for _, rule := range got {
					if rule.Key == wantRule {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("FindMissingQualityProfileRules() missing rule %s", wantRule)
				}
			}
		})
	}
}

// TestFindNotUpToDateQualityProfileRules tests finding out-of-date rules.
func TestFindNotUpToDateQualityProfileRules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		associations map[string]QualityProfileRuleAssociation
		wantCount    int
		wantRules    []string
	}{
		"Empty": {
			associations: map[string]QualityProfileRuleAssociation{},
			wantCount:    0,
			wantRules:    []string{},
		},
		"AllUpToDate": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
					UpToDate:    true,
				},
			},
			wantCount: 0,
			wantRules: []string{},
		},
		"SomeNotUpToDate": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144"), Severity: new("CRITICAL")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144", Severity: "MAJOR"},
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1145")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1145"},
					UpToDate:    true,
				},
			},
			wantCount: 1,
			wantRules: []string{"java:S1144"},
		},
		"IgnoreNilSpecOrObservation": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec:        nil, // Should be ignored
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1145")},
					Observation: nil, // Should be ignored
					UpToDate:    false,
				},
			},
			wantCount: 0,
			wantRules: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindNotUpToDateQualityProfileRules(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNotUpToDateQualityProfileRules() returned %d rules, want %d", len(got), tc.wantCount)
			}

			for _, wantRule := range tc.wantRules {
				found := false

				for _, assoc := range got {
					if assoc.Spec != nil && ptr.Deref(assoc.Spec.Rule, "") == wantRule {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("FindNotUpToDateQualityProfileRules() missing rule %s", wantRule)
				}
			}
		})
	}
}

// TestAreQualityProfileRulesUpToDate tests checking all rules are up-to-date.
func TestAreQualityProfileRulesUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		associations map[string]QualityProfileRuleAssociation
		want         bool
	}{
		"Empty": {
			associations: map[string]QualityProfileRuleAssociation{},
			want:         true,
		},
		"AllUpToDate": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {UpToDate: true},
				"java:S1145": {UpToDate: true},
			},
			want: true,
		},
		"SomeNotUpToDate": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {UpToDate: true},
				"java:S1145": {UpToDate: false},
			},
			want: false,
		},
		"AllNotUpToDate": {
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {UpToDate: false},
				"java:S1145": {UpToDate: false},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreQualityProfileRulesUpToDate(tc.associations)
			if got != tc.want {
				t.Errorf("AreQualityProfileRulesUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestWereQualityProfileRulesLateInitialized tests rule late init detection.
func TestWereQualityProfileRulesLateInitialized(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		original []v1alpha1.QualityProfileRuleParameters
		updated  []v1alpha1.QualityProfileRuleParameters
		want     bool
	}{
		"Empty": {
			original: []v1alpha1.QualityProfileRuleParameters{},
			updated:  []v1alpha1.QualityProfileRuleParameters{},
			want:     false,
		},
		"LengthMismatch": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
				{Rule: new("java:S1145")},
			},
			want: true,
		},
		"RuleKeyMissing": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144")},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1145")},
			},
			want: true,
		},
		"SeverityLateInitialized": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Severity: nil},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Severity: new("MAJOR")},
			},
			want: true,
		},
		"PrioritizedLateInitialized": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Prioritized: nil},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Prioritized: new(true)},
			},
			want: true,
		},
		"NoChange": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Severity: new("MAJOR"), Prioritized: new(false)},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: new("java:S1144"), Severity: new("MAJOR"), Prioritized: new(false)},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := WereQualityProfileRulesLateInitialized(tc.original, tc.updated)
			if got != tc.want {
				t.Errorf("WereQualityProfileRulesLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateQualityProfileActivateRuleOption tests rule activation options.
func TestGenerateQualityProfileActivateRuleOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		profileKey string
		params     v1alpha1.QualityProfileRuleParameters
		want       *sonar.QualityprofilesActivateRuleOptions
	}{
		"BasicRule": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule: new("java:S1144"),
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				PrioritizedRule: false,
			},
		},
		"RuleWithSeverity": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     new("java:S1144"),
				Severity: new("CRITICAL"),
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Severity:        "CRITICAL",
				PrioritizedRule: false,
			},
		},
		"RuleWithPrioritized": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:        new("java:S1144"),
				Prioritized: new(true),
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				PrioritizedRule: true,
			},
		},
		"RuleWithImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:    new("java:S1144"),
				Impacts: &map[string]string{"MAINTAINABILITY": "HIGH"},
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Impacts:         map[string]string{"MAINTAINABILITY": "HIGH"},
				PrioritizedRule: false,
			},
		},
		"RuleWithParams": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:       new("java:S1144"),
				Parameters: &map[string]string{"max": "10"},
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Params:          map[string]string{"max": "10"},
				PrioritizedRule: false,
			},
		},
		"RuleWithBothImpactsAndSeverityPrioritizesImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     new("java:S1144"),
				Severity: new("CRITICAL"),
				Impacts:  &map[string]string{"MAINTAINABILITY": "HIGH"},
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Impacts:         map[string]string{"MAINTAINABILITY": "HIGH"},
				PrioritizedRule: false,
				// Note: Severity should NOT be set when Impacts is present
			},
		},
		"RuleWithOnlySeverityNoImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     new("java:S1144"),
				Severity: new("BLOCKER"),
			},
			want: &sonar.QualityprofilesActivateRuleOptions{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Severity:        "BLOCKER",
				PrioritizedRule: false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileActivateRuleOption(tc.profileKey, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileActivateRuleOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateQualityProfileDeactivateRuleOption tests deactivation options.
func TestGenerateQualityProfileDeactivateRuleOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		profileKey string
		ruleKey    string
		want       *sonar.QualityprofilesDeactivateRuleOptions
	}{
		"BasicDeactivate": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			ruleKey:    "java:S1144",
			want: &sonar.QualityprofilesDeactivateRuleOptions{
				Key:  "AU-TpxcA-iU5OvuD2FLz",
				Rule: "java:S1144",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileDeactivateRuleOption(tc.profileKey, tc.ruleKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileDeactivateRuleOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLateInitializeQualityProfile tests quality profile late initialization.
func TestLateInitializeQualityProfile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.QualityProfileParameters
		observation *v1alpha1.QualityProfileObservation
		wantDefault *bool
	}{
		"NilSpec": {
			spec:        nil,
			observation: &v1alpha1.QualityProfileObservation{IsDefault: true},
			wantDefault: nil,
		},
		"NilObservation": {
			spec:        &v1alpha1.QualityProfileParameters{Name: "test", Language: "java"},
			observation: nil,
			wantDefault: nil,
		},
		"DefaultNotSet": {
			spec:        &v1alpha1.QualityProfileParameters{Name: "test", Language: "java"},
			observation: &v1alpha1.QualityProfileObservation{IsDefault: true},
			wantDefault: new(true),
		},
		"DefaultAlreadySet": {
			spec:        &v1alpha1.QualityProfileParameters{Name: "test", Language: "java", Default: new(false)},
			observation: &v1alpha1.QualityProfileObservation{IsDefault: true},
			wantDefault: new(false),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeQualityProfile(tc.spec, tc.observation, nil)

			if tc.spec != nil {
				if diff := cmp.Diff(tc.wantDefault, tc.spec.Default); diff != "" {
					t.Errorf("LateInitializeQualityProfile() Default mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestGenerateQualityProfilesSearchProjectObservation verifies that the
// observation is keyed by language (not by profile UUID) so that
// AreProjectQualityProfilesUpToDate can look up profiles via the
// language key used in the spec.
// TestGenerateQualityProfilesSearchProjectObservation tests
// project observation.
func TestGenerateQualityProfilesSearchProjectObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input *sonar.QualityprofilesSearch
		want  map[string]v1alpha1.ProjectQualityProfileObservation
	}{
		"EmptyProfiles": {
			input: &sonar.QualityprofilesSearch{},
			want:  map[string]v1alpha1.ProjectQualityProfileObservation{},
		},
		"SingleProfile_KeyedByLanguage": {
			input: &sonar.QualityprofilesSearch{
				Profiles: []sonar.QualityProfile{
					{Key: "uuid-java-sonarway", Name: "Sonar way", Language: "java"},
				},
			},
			want: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java-sonarway", Name: "Sonar way"},
			},
		},
		"MultipleProfiles_EachKeyedByItsLanguage": {
			input: &sonar.QualityprofilesSearch{
				Profiles: []sonar.QualityProfile{
					{Key: "uuid-java", Name: "Sonar way", Language: "java"},
					{Key: "uuid-go", Name: "Sonar way Go", Language: "go"},
					{Key: "uuid-ts", Name: "Sonar way TS", Language: "ts"},
				},
			},
			want: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
				"go":   {Id: "uuid-go", Name: "Sonar way Go"},
				"ts":   {Id: "uuid-ts", Name: "Sonar way TS"},
			},
		},
		// Regression test: previously keyed by UUID, causing infinite reconcile loops
		// when the spec referenced profiles by language but the observation was keyed by UUID.
		"ProfileUUIDIsStoredInId_NotAsMapKey": {
			input: &sonar.QualityprofilesSearch{
				Profiles: []sonar.QualityProfile{
					{Key: "0dd2332b-b4b4-4ac1-a2e4-07adec0ae6e5", Name: "Sonar way", Language: "java"},
				},
			},
			want: map[string]v1alpha1.ProjectQualityProfileObservation{
				// Key must be "java" (language), not the UUID
				"java": {Id: "0dd2332b-b4b4-4ac1-a2e4-07adec0ae6e5", Name: "Sonar way"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfilesSearchProjectObservation(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfilesSearchProjectObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAreProjectQualityProfilesUpToDate verifies the up-to-date check,
// with special attention to the regression cases that caused
// infinite reconcile loops:
//   - empty spec + non-empty observation must be considered up to date
//   - observation may have more entries than spec (default built-in profiles)
//
// TestAreProjectQualityProfilesUpToDate tests project profiles up-to-date.
func TestAreProjectQualityProfilesUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        map[string]v1alpha1.ProjectQualityProfileReference
		observation map[string]v1alpha1.ProjectQualityProfileObservation
		want        bool
	}{
		// Regression: spec has no qualityProfiles but SonarQube returns 27 default ones.
		// Previously returned false (len 0 != len 27) → infinite update loop.
		"NilSpec_NonEmptyObservation_ShouldBeUpToDate": {
			spec: nil,
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
				"go":   {Id: "uuid-go", Name: "Sonar way"},
			},
			want: true,
		},
		"EmptySpec_NonEmptyObservation_ShouldBeUpToDate": {
			spec: map[string]v1alpha1.ProjectQualityProfileReference{},
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
			},
			want: true,
		},
		"BothEmpty_ShouldBeUpToDate": {
			spec:        nil,
			observation: nil,
			want:        true,
		},
		// Spec declares a profile for java and the observation has the same UUID → up to date.
		"SpecMatchesObservation_UpToDate": {
			spec: map[string]v1alpha1.ProjectQualityProfileReference{
				"java": {Id: new("uuid-java")},
			},
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
			},
			want: true,
		},
		// Spec declares java but the UUID in observations differs → not up to date.
		"SpecIDMismatch_NotUpToDate": {
			spec: map[string]v1alpha1.ProjectQualityProfileReference{
				"java": {Id: new("uuid-new-java")},
			},
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-old-java", Name: "Sonar way"},
			},
			want: false,
		},
		// Spec declares a language that is not in the observation → not up to date.
		"SpecLanguageMissingFromObservation_NotUpToDate": {
			spec: map[string]v1alpha1.ProjectQualityProfileReference{
				"go": {Id: new("uuid-go")},
			},
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
			},
			want: false,
		},
		// Observation has extra languages not in spec → still up to date (unmanaged profiles).
		"ObservationHasExtraLanguages_UpToDate": {
			spec: map[string]v1alpha1.ProjectQualityProfileReference{
				"java": {Id: new("uuid-java")},
			},
			observation: map[string]v1alpha1.ProjectQualityProfileObservation{
				"java": {Id: "uuid-java", Name: "Sonar way"},
				"go":   {Id: "uuid-go", Name: "Sonar way"},
				"ts":   {Id: "uuid-ts", Name: "Sonar way"},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreProjectQualityProfilesUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("AreProjectQualityProfilesUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateQualityProfileObservation tests profile observations.
func TestGenerateQualityProfileObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		observation *sonar.QualityprofilesShow
		rules       *sonar.RulesSearch
		want        v1alpha1.QualityProfileObservation
	}{
		"BasicObservation": {
			observation: &sonar.QualityprofilesShow{
				Profile: sonar.QualityprofilesShownProfile{
					ActiveDeprecatedRuleCount: 1,
					ActiveRuleCount:           10,
					IsBuiltIn:                 false,
					IsDefault:                 true,
					IsInherited:               false,
					Key:                       "AXqPwMhVHYprcJvnedvY",
					Language:                  "java",
					LanguageName:              "Java",
					Name:                      "my-profile",
					ProjectCount:              5,
				},
			},
			rules: nil,
			want: v1alpha1.QualityProfileObservation{
				ActiveDeprecatedRuleCount: 1,
				ActiveRuleCount:           10,
				IsBuiltIn:                 false,
				IsDefault:                 true,
				IsInherited:               false,
				Key:                       "AXqPwMhVHYprcJvnedvY",
				Language:                  "java",
				LanguageName:              "Java",
				Name:                      "my-profile",
				ProjectCount:              5,
				Rules:                     []v1alpha1.QualityProfileRuleObservation{},
			},
		},
		"ObservationWithRules": {
			observation: &sonar.QualityprofilesShow{
				Profile: sonar.QualityprofilesShownProfile{
					Key:      "profile-key",
					Language: "go",
					Name:     "Go Profile",
				},
			},
			rules: &sonar.RulesSearch{
				Rules: []sonar.RulesDetails{
					{Key: "go:S1000", Name: "Rule 1"},
				},
			},
			want: v1alpha1.QualityProfileObservation{
				Key:      "profile-key",
				Language: "go",
				Name:     "Go Profile",
				Rules: []v1alpha1.QualityProfileRuleObservation{
					{Key: "go:S1000", Name: "Rule 1", Impacts: []v1alpha1.QualityProfileRuleImpact{}},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileObservation(tc.observation, tc.rules)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateQualityprofilesSetDefaultOption tests set default options.
func TestGenerateQualityprofilesSetDefaultOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params v1alpha1.QualityProfileParameters
		want   *sonar.QualityprofilesSetDefaultOptions
	}{
		"BasicSetDefault": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			want: &sonar.QualityprofilesSetDefaultOptions{
				QualityProfile: "my-profile",
				Language:       "java",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityprofilesSetDefaultOption(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityprofilesSetDefaultOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateQualityProfilesSearchProjectOptions tests search options.
func TestGenerateQualityProfilesSearchProjectOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey string
		want       *sonar.QualityprofilesSearchOptions
	}{
		"BasicSearch": {
			projectKey: "my-project",
			want: &sonar.QualityprofilesSearchOptions{
				Project: "my-project",
			},
		},
		"EmptyProjectKey": {
			projectKey: "",
			want: &sonar.QualityprofilesSearchOptions{
				Project: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfilesSearchProjectOptions(tc.projectKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfilesSearchProjectOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateQualityProfileAddProjectOptions tests generating
// add project options.
func TestGenerateQualityProfileAddProjectOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		projectKey         string
		qualityProfileName string
		language           string
		want               *sonar.QualityprofilesAddProjectOptions
	}{
		"BasicAddProject": {
			projectKey:         "my-project",
			qualityProfileName: "my-profile",
			language:           "java",
			want: &sonar.QualityprofilesAddProjectOptions{
				Project:        "my-project",
				QualityProfile: "my-profile",
				Language:       "java",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileAddProjectOptions(tc.projectKey, tc.qualityProfileName, tc.language)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileAddProjectOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateQualityProfileShowOptions tests generating show profile options.
func TestGenerateQualityProfileShowOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		qualityProfileKey string
		want              *sonar.QualityprofilesShowOptions
	}{
		"BasicShowOption": {
			qualityProfileKey: "AXqPwMhVHYprcJvnedvY",
			want: &sonar.QualityprofilesShowOptions{
				Key: "AXqPwMhVHYprcJvnedvY",
			},
		},
		"EmptyKey": {
			qualityProfileKey: "",
			want: &sonar.QualityprofilesShowOptions{
				Key: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileShowOptions(tc.qualityProfileKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileShowOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLateInitializeQualityProfileRules tests late initializing
// quality profile rules.
func TestLateInitializeQualityProfileRules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		associations map[string]QualityProfileRuleAssociation
		wantUpdated  map[string]QualityProfileRuleAssociation
	}{
		"EmptyAssociations": {
			associations: map[string]QualityProfileRuleAssociation{},
			wantUpdated:  map[string]QualityProfileRuleAssociation{},
		},
		"NilAssociations": {
			associations: nil,
			wantUpdated:  nil,
		},
		"SpecWithoutObservationSkipped": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: nil,
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: nil,
				},
			},
		},
		"ObservationWithoutSpecSkipped": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        nil,
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Severity: "MAJOR"},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        nil,
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Severity: "MAJOR"},
				},
			},
		},
		"SeverityLateInitialized": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Severity: "MAJOR"},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1"), Severity: new("MAJOR"), Prioritized: new(false)},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Severity: "MAJOR"},
				},
			},
		},
		"PrioritizedLateInitialized": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Prioritized: true},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1"), Severity: new(""), Prioritized: new(true)},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Prioritized: true},
				},
			},
		},
		"ImpactsLateInitialized": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec: &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key: "rule1",
						Impacts: []v1alpha1.QualityProfileRuleImpact{
							{SoftwareQuality: "SECURITY", Severity: "HIGH"},
						},
					},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule:        new("rule1"),
						Severity:    new(""),
						Prioritized: new(false),
						Impacts:     &map[string]string{"SECURITY": "HIGH"},
					},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key: "rule1",
						Impacts: []v1alpha1.QualityProfileRuleImpact{
							{SoftwareQuality: "SECURITY", Severity: "HIGH"},
						},
					},
				},
			},
		},
		"ImpactsNotOverwritten": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule:    new("rule1"),
						Impacts: &map[string]string{"MAINTAINABILITY": "LOW"},
					},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key: "rule1",
						Impacts: []v1alpha1.QualityProfileRuleImpact{
							{SoftwareQuality: "SECURITY", Severity: "HIGH"},
						},
					},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule:        new("rule1"),
						Severity:    new(""),
						Prioritized: new(false),
						Impacts:     &map[string]string{"MAINTAINABILITY": "LOW"},
					},
					Observation: &v1alpha1.QualityProfileRuleObservation{
						Key: "rule1",
						Impacts: []v1alpha1.QualityProfileRuleImpact{
							{SoftwareQuality: "SECURITY", Severity: "HIGH"},
						},
					},
				},
			},
		},
		"EmptyObservationImpactsNotAssigned": {
			associations: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Impacts: []v1alpha1.QualityProfileRuleImpact{}},
				},
			},
			wantUpdated: map[string]QualityProfileRuleAssociation{
				"rule1": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("rule1"), Severity: new(""), Prioritized: new(false)},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "rule1", Impacts: []v1alpha1.QualityProfileRuleImpact{}},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeQualityProfileRules(tc.associations)

			if diff := cmp.Diff(tc.wantUpdated, tc.associations); diff != "" {
				t.Errorf("LateInitializeQualityProfileRules() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
