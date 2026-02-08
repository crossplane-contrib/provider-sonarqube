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
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"
)

func TestGenerateQualityProfileRulesSearchOption(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key      string
		page     int
		wantPage int64
	}{
		"FirstPage": {
			key:      "test-profile-key",
			page:     1,
			wantPage: 1,
		},
		"SecondPage": {
			key:      "another-key",
			page:     2,
			wantPage: 2,
		},
		"LargePage": {
			key:      "profile",
			page:     100,
			wantPage: 100,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRulesSearchOption(tc.key, tc.page)

			if got.Qprofile != tc.key {
				t.Errorf("GenerateQualityProfileRulesSearchOption().Qprofile = %v, want %v", got.Qprofile, tc.key)
			}

			if !got.Activation {
				t.Errorf("GenerateQualityProfileRulesSearchOption().Activation = false, want true")
			}

			if got.Page != tc.wantPage {
				t.Errorf("GenerateQualityProfileRulesSearchOption().Page = %v, want %v", got.Page, tc.wantPage)
			}

			if got.PageSize != maxRulesPerPage {
				t.Errorf("GenerateQualityProfileRulesSearchOption().PageSize = %v, want %v", got.PageSize, maxRulesPerPage)
			}

			if len(got.Fields) == 0 {
				t.Errorf("GenerateQualityProfileRulesSearchOption().Fields is empty, want non-empty")
			}
		})
	}
}

func TestGenerateQualityProfileRulesObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		qualityProfile string
		rules          *sonar.RulesSearch
		wantCount      int
	}{
		"NilRules": {
			qualityProfile: "test-profile",
			rules:          nil,
			wantCount:      0,
		},
		"NilRulesList": {
			qualityProfile: "test-profile",
			rules: &sonar.RulesSearch{
				Rules: nil,
			},
			wantCount: 0,
		},
		"EmptyRules": {
			qualityProfile: "test-profile",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{},
			},
			wantCount: 0,
		},
		"SingleRule": {
			qualityProfile: "test-profile",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{
					{
						Key:  "java:S1144",
						Name: "Remove unused code",
					},
				},
				Actives: map[string][]sonar.RuleActivation{
					"test-profile": {
						{
							QProfile: "test-profile",
							Severity: "MAJOR",
						},
					},
				},
			},
			wantCount: 1,
		},
		"MultipleRules": {
			qualityProfile: "test-profile",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{
					{Key: "java:S1144"},
					{Key: "java:S1145"},
					{Key: "java:S1146"},
				},
			},
			wantCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRulesObservation(tc.qualityProfile, tc.rules)

			if len(got) != tc.wantCount {
				t.Errorf("GenerateQualityProfileRulesObservation() returned %d rules, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFindQualityProfileActiveRuleSettings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		qualityProfile string
		activeRules    *[]sonar.RuleActivation
		wantNil        bool
		wantSeverity   *string
		wantParamCount int
	}{
		"NilActiveRules": {
			qualityProfile: "test-profile",
			activeRules:    nil,
			wantNil:        true,
		},
		"EmptyActiveRules": {
			qualityProfile: "test-profile",
			activeRules:    &[]sonar.RuleActivation{},
			wantNil:        true,
		},
		"NoMatchingProfile": {
			qualityProfile: "test-profile",
			activeRules: &[]sonar.RuleActivation{
				{
					QProfile: "different-profile",
					Severity: "MAJOR",
				},
			},
			wantNil: true,
		},
		"MatchingProfile": {
			qualityProfile: "test-profile",
			activeRules: &[]sonar.RuleActivation{
				{
					QProfile: "test-profile",
					Severity: "MAJOR",
					Params:   []sonar.ParamKV{},
				},
			},
			wantNil:        false,
			wantSeverity:   ptr.To("MAJOR"),
			wantParamCount: 0,
		},
		"MatchingProfileWithParams": {
			qualityProfile: "test-profile",
			activeRules: &[]sonar.RuleActivation{
				{
					QProfile: "test-profile",
					Severity: "CRITICAL",
					Params: []sonar.ParamKV{
						{Key: "max", Value: "10"},
						{Key: "min", Value: "5"},
					},
				},
			},
			wantNil:        false,
			wantSeverity:   ptr.To("CRITICAL"),
			wantParamCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := findQualityProfileActiveRuleSettings(tc.qualityProfile, tc.activeRules)

			if tc.wantNil {
				if got != nil {
					t.Errorf("findQualityProfileActiveRuleSettings() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("findQualityProfileActiveRuleSettings() = nil, want non-nil")
			}

			if got.Severity == nil || *got.Severity != *tc.wantSeverity {
				t.Errorf("findQualityProfileActiveRuleSettings().Severity = %v, want %v", got.Severity, tc.wantSeverity)
			}

			if got.Params != nil && len(*got.Params) != tc.wantParamCount {
				t.Errorf("findQualityProfileActiveRuleSettings().Params has %d items, want %d", len(*got.Params), tc.wantParamCount)
			}
		})
	}
}

func TestGenerateQualityProfileRuleObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rule              sonar.RuleDetails
		activatedSettings *ruleActiveSettings
		wantKey           string
		wantSeverity      string
		wantParamCount    int
	}{
		"BasicRule": {
			rule: sonar.RuleDetails{
				Key:      "java:S1144",
				Name:     "Remove unused code",
				Severity: "INFO",
			},
			activatedSettings: nil,
			wantKey:           "java:S1144",
			wantSeverity:      "INFO",
			wantParamCount:    0,
		},
		"RuleWithActivatedSettings": {
			rule: sonar.RuleDetails{
				Key:      "java:S1144",
				Name:     "Remove unused code",
				Severity: "INFO",
			},
			activatedSettings: &ruleActiveSettings{
				Severity: ptr.To("MAJOR"),
				Params:   &map[string]string{"max": "10"},
			},
			wantKey:        "java:S1144",
			wantSeverity:   "MAJOR",
			wantParamCount: 1,
		},
		"RuleWithOnlySeverityOverride": {
			rule: sonar.RuleDetails{
				Key:      "java:S1145",
				Severity: "MINOR",
			},
			activatedSettings: &ruleActiveSettings{
				Severity: ptr.To("BLOCKER"),
			},
			wantKey:        "java:S1145",
			wantSeverity:   "BLOCKER",
			wantParamCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRuleObservation(tc.rule, tc.activatedSettings)

			if got.Key != tc.wantKey {
				t.Errorf("GenerateQualityProfileRuleObservation().Key = %v, want %v", got.Key, tc.wantKey)
			}

			if got.Severity != tc.wantSeverity {
				t.Errorf("GenerateQualityProfileRuleObservation().Severity = %v, want %v", got.Severity, tc.wantSeverity)
			}

			if len(got.Parameters) != tc.wantParamCount {
				t.Errorf("GenerateQualityProfileRuleObservation().Parameters has %d items, want %d", len(got.Parameters), tc.wantParamCount)
			}
		})
	}
}
func TestGenerateQualityProfileImpactsObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		impacts   *[]sonar.RuleImpact
		wantCount int
	}{
		"NilImpacts": {
			impacts:   nil,
			wantCount: 0,
		},
		"EmptyImpacts": {
			impacts:   &[]sonar.RuleImpact{},
			wantCount: 0,
		},
		"SingleImpact": {
			impacts: &[]sonar.RuleImpact{
				{
					SoftwareQuality: "MAINTAINABILITY",
					Severity:        "HIGH",
				},
			},
			wantCount: 1,
		},
		"MultipleImpacts": {
			impacts: &[]sonar.RuleImpact{
				{SoftwareQuality: "MAINTAINABILITY", Severity: "HIGH"},
				{SoftwareQuality: "SECURITY", Severity: "MEDIUM"},
				{SoftwareQuality: "RELIABILITY", Severity: "LOW"},
			},
			wantCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileImpactsObservation(tc.impacts)

			if len(got) != tc.wantCount {
				t.Errorf("GenerateQualityProfileImpactsObservation() returned %d impacts, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestGenerateQualityProfileRuleImpactObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		impact sonar.RuleImpact
		want   v1alpha1.QualityProfileRuleImpact
	}{
		"HighMaintainability": {
			impact: sonar.RuleImpact{
				SoftwareQuality: "MAINTAINABILITY",
				Severity:        "HIGH",
			},
			want: v1alpha1.QualityProfileRuleImpact{
				SoftwareQuality: "MAINTAINABILITY",
				Severity:        "HIGH",
			},
		},
		"MediumSecurity": {
			impact: sonar.RuleImpact{
				SoftwareQuality: "SECURITY",
				Severity:        "MEDIUM",
			},
			want: v1alpha1.QualityProfileRuleImpact{
				SoftwareQuality: "SECURITY",
				Severity:        "MEDIUM",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRuleImpactObservation(tc.impact)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileRuleImpactObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsQualityProfileRuleUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.QualityProfileRuleParameters
		observation *v1alpha1.QualityProfileRuleObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
			observation: nil,
			want:        false,
		},
		"DifferentRuleKeyReturnsFalse": {
			spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
			observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1145"},
			want:        false,
		},
		"MatchingRuleNoSeverityNoParams": {
			spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
			observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
			want:        true,
		},
		"MatchingRuleWithMatchingSeverity": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:     "java:S1144",
				Severity: ptr.To("MAJOR"),
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:      "java:S1144",
				Severity: "MAJOR",
			},
			want: true,
		},
		"DifferentSeverityReturnsFalse": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:     "java:S1144",
				Severity: ptr.To("MAJOR"),
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:      "java:S1144",
				Severity: "MINOR",
			},
			want: false,
		},
		"MatchingParameters": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:       "java:S1144",
				Parameters: &map[string]string{"max": "10"},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1144",
				Parameters: map[string]string{"max": "10"},
			},
			want: true,
		},
		"DifferentParametersReturnsFalse": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:       "java:S1144",
				Parameters: &map[string]string{"max": "10"},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1144",
				Parameters: map[string]string{"max": "20"},
			},
			want: false,
		},
		"NilSpecParametersDoesNotCheck": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule: "java:S1144",
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1144",
				Parameters: map[string]string{"max": "20"},
			},
			want: true,
		},
		"EmptySpecParametersMatchesEmptyObservation": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:       "java:S1144",
				Parameters: &map[string]string{},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1144",
				Parameters: map[string]string{},
			},
			want: true,
		},
		"AllFieldsMatching": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:       "java:S1144",
				Severity:   ptr.To("CRITICAL"),
				Parameters: &map[string]string{"max": "15", "min": "5"},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1144",
				Severity:   "CRITICAL",
				Parameters: map[string]string{"max": "15", "min": "5"},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsQualityProfileRuleUpToDate(tc.spec, tc.observation)

			if got != tc.want {
				t.Errorf("IsQualityProfileRuleUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
