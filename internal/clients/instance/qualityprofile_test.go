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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

func TestGenerateCreateQualityProfileOption(t *testing.T) {
	tests := map[string]struct {
		params v1alpha1.QualityProfileParameters
		want   *sonargo.QualityprofilesCreateOption
	}{
		"BasicProfile": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			want: &sonargo.QualityprofilesCreateOption{
				Name:     "my-profile",
				Language: "java",
			},
		},
		"ProfileWithDefault": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "default-profile",
				Language: "go",
				Default:  ptr.To(true),
			},
			want: &sonargo.QualityprofilesCreateOption{
				Name:     "default-profile",
				Language: "go",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateCreateQualityProfileOption(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateCreateQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateDeleteQualityProfileOption(t *testing.T) {
	tests := map[string]struct {
		params v1alpha1.QualityProfileParameters
		want   *sonargo.QualityprofilesDeleteOption
	}{
		"BasicDelete": {
			params: v1alpha1.QualityProfileParameters{
				Name:     "my-profile",
				Language: "java",
			},
			want: &sonargo.QualityprofilesDeleteOption{
				QualityProfile: "my-profile",
				Language:       "java",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateDeleteQualityProfileOption(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateDeleteQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateRenameQualityProfileOption(t *testing.T) {
	tests := map[string]struct {
		key    string
		params v1alpha1.QualityProfileParameters
		want   *sonargo.QualityprofilesRenameOption
	}{
		"BasicRename": {
			key: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileParameters{
				Name:     "new-name",
				Language: "java",
			},
			want: &sonargo.QualityprofilesRenameOption{
				Key:  "AU-TpxcA-iU5OvuD2FLz",
				Name: "new-name",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateRenameQualityProfileOption(tc.key, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateRenameQualityProfileOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsQualityProfileUpToDate(t *testing.T) {
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
				Default:  ptr.To(true),
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
				Default:  ptr.To(true),
			},
			observation: &v1alpha1.QualityProfileObservation{
				Name:      "my-profile",
				Language:  "java",
				IsDefault: true,
			},
			associations: map[string]QualityProfileRuleAssociation{
				"java:S1144": {
					Spec: &v1alpha1.QualityProfileRuleParameters{
						Rule: "java:S1144",
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
						Rule:     "java:S1144",
						Severity: ptr.To("CRITICAL"),
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
			got := IsQualityProfileUpToDate(tc.spec, tc.observation, tc.associations)
			if got != tc.want {
				t.Errorf("IsQualityProfileUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateQualityProfileRulesAssociation(t *testing.T) {
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
				{Rule: "java:S1144"},
				{Rule: "java:S1145"},
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
				{Rule: "java:S1144"},
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
				{Rule: "java:S1144"},
				{Rule: "java:S1146"}, // New rule to be activated
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

func TestFindNonExistingQualityProfileRules(t *testing.T) {
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
					Observation: nil, // No observation means needs activation
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1145"},
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
			got := FindNonExistingQualityProfileRules(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNonExistingQualityProfileRules() returned %d rules, want %d", len(got), tc.wantCount)
			}

			for _, wantRule := range tc.wantRules {
				found := false
				for _, rule := range got {
					if rule.Rule == wantRule {
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

func TestFindMissingQualityProfileRules(t *testing.T) {
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1145"},
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

func TestFindNotUpToDateQualityProfileRules(t *testing.T) {
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144", Severity: ptr.To("CRITICAL")},
					Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144", Severity: "MAJOR"},
					UpToDate:    false,
				},
				"java:S1145": {
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1145"},
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
					Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1145"},
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
			got := FindNotUpToDateQualityProfileRules(tc.associations)
			if len(got) != tc.wantCount {
				t.Errorf("FindNotUpToDateQualityProfileRules() returned %d rules, want %d", len(got), tc.wantCount)
			}

			for _, wantRule := range tc.wantRules {
				found := false
				for _, assoc := range got {
					if assoc.Spec != nil && assoc.Spec.Rule == wantRule {
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

func TestAreQualityProfileRulesUpToDate(t *testing.T) {
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
			got := AreQualityProfileRulesUpToDate(tc.associations)
			if got != tc.want {
				t.Errorf("AreQualityProfileRulesUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWereQualityProfileRulesLateInitialized(t *testing.T) {
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
				{Rule: "java:S1144"},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144"},
				{Rule: "java:S1145"},
			},
			want: true,
		},
		"RuleKeyMissing": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144"},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1145"},
			},
			want: true,
		},
		"SeverityLateInitialized": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Severity: nil},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Severity: ptr.To("MAJOR")},
			},
			want: true,
		},
		"PrioritizedLateInitialized": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Prioritized: nil},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Prioritized: ptr.To(true)},
			},
			want: true,
		},
		"NoChange": {
			original: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Severity: ptr.To("MAJOR"), Prioritized: ptr.To(false)},
			},
			updated: []v1alpha1.QualityProfileRuleParameters{
				{Rule: "java:S1144", Severity: ptr.To("MAJOR"), Prioritized: ptr.To(false)},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := WereQualityProfileRulesLateInitialized(tc.original, tc.updated)
			if got != tc.want {
				t.Errorf("WereQualityProfileRulesLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateQualityProfileActivateRuleOption(t *testing.T) {
	tests := map[string]struct {
		profileKey string
		params     v1alpha1.QualityProfileRuleParameters
		want       *sonargo.QualityprofilesActivateRuleOption
	}{
		"BasicRule": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule: "java:S1144",
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				PrioritizedRule: "false",
			},
		},
		"RuleWithSeverity": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     "java:S1144",
				Severity: ptr.To("CRITICAL"),
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Severity:        "CRITICAL",
				PrioritizedRule: "false",
			},
		},
		"RuleWithPrioritized": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:        "java:S1144",
				Prioritized: ptr.To(true),
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				PrioritizedRule: "true",
			},
		},
		"RuleWithImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:    "java:S1144",
				Impacts: &map[string]string{"MAINTAINABILITY": "HIGH"},
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Impacts:         "MAINTAINABILITY=HIGH",
				PrioritizedRule: "false",
			},
		},
		"RuleWithParams": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:       "java:S1144",
				Parameters: &map[string]string{"max": "10"},
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Params:          "max=10",
				PrioritizedRule: "false",
			},
		},
		"RuleWithBothImpactsAndSeverityPrioritizesImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     "java:S1144",
				Severity: ptr.To("CRITICAL"),
				Impacts:  &map[string]string{"MAINTAINABILITY": "HIGH"},
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Impacts:         "MAINTAINABILITY=HIGH",
				PrioritizedRule: "false",
				// Note: Severity should NOT be set when Impacts is present
			},
		},
		"RuleWithOnlySeverityNoImpacts": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			params: v1alpha1.QualityProfileRuleParameters{
				Rule:     "java:S1144",
				Severity: ptr.To("BLOCKER"),
			},
			want: &sonargo.QualityprofilesActivateRuleOption{
				Key:             "AU-TpxcA-iU5OvuD2FLz",
				Rule:            "java:S1144",
				Severity:        "BLOCKER",
				PrioritizedRule: "false",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityProfileActivateRuleOption(tc.profileKey, tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileActivateRuleOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateQualityProfileDeactivateRuleOption(t *testing.T) {
	tests := map[string]struct {
		profileKey string
		ruleKey    string
		want       *sonargo.QualityprofilesDeactivateRuleOption
	}{
		"BasicDeactivate": {
			profileKey: "AU-TpxcA-iU5OvuD2FLz",
			ruleKey:    "java:S1144",
			want: &sonargo.QualityprofilesDeactivateRuleOption{
				Key:  "AU-TpxcA-iU5OvuD2FLz",
				Rule: "java:S1144",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateQualityProfileDeactivateRuleOption(tc.profileKey, tc.ruleKey)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileDeactivateRuleOption() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLateInitializeQualityProfile(t *testing.T) {
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
			wantDefault: ptr.To(true),
		},
		"DefaultAlreadySet": {
			spec:        &v1alpha1.QualityProfileParameters{Name: "test", Language: "java", Default: ptr.To(false)},
			observation: &v1alpha1.QualityProfileObservation{IsDefault: true},
			wantDefault: ptr.To(false),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			LateInitializeQualityProfile(tc.spec, tc.observation)
			if tc.spec != nil {
				if diff := cmp.Diff(tc.wantDefault, tc.spec.Default); diff != "" {
					t.Errorf("LateInitializeQualityProfile() Default mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
