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
	"errors"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// mockRulesClient is a test-local mock implementation of the RulesClient interface.
type mockRulesClient struct {
	SearchFn func(opt *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error)
}

var errMockNotImplemented = errors.New("not implemented")

func (m *mockRulesClient) App() (*sonar.RulesApp, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Create(_ *sonar.RulesCreateOptions) (*sonar.RulesCreate, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Delete(_ *sonar.RulesDeleteOptions) (*http.Response, error) {
	return nil, errMockNotImplemented
}

func (m *mockRulesClient) List(_ *sonar.RulesListOptions) (*string, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Repositories(_ *sonar.RulesRepositoriesOptions) (*sonar.RulesRepositories, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Search(opt *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Show(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Tags(_ *sonar.RulesTagsOptions) (*sonar.RulesTags, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

func (m *mockRulesClient) Update(opt *sonar.RulesUpdateOptions) (*sonar.RulesUpdate, *http.Response, error) {
	return nil, nil, errMockNotImplemented
}

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

func TestIsQualityProfileRuleUpToDatePrioritized(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.QualityProfileRuleParameters
		observation *v1alpha1.QualityProfileRuleObservation
		want        bool
	}{
		"PrioritizedMatchReturnsTrue": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:        "java:S1000",
				Prioritized: ptr.To(true),
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:         "java:S1000",
				Prioritized: true,
			},
			want: true,
		},
		"PrioritizedMismatchReturnsFalse": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:        "java:S1000",
				Prioritized: ptr.To(true),
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:         "java:S1000",
				Prioritized: false,
			},
			want: false,
		},
		"ImpactsMatchReturnsTrue": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:    "java:S1000",
				Impacts: &map[string]string{"SECURITY": "HIGH"},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:     "java:S1000",
				Impacts: []v1alpha1.QualityProfileRuleImpact{{SoftwareQuality: "SECURITY", Severity: "HIGH"}},
			},
			want: true,
		},
		"ImpactsMismatchReturnsFalse": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule:    "java:S1000",
				Impacts: &map[string]string{"SECURITY": "HIGH"},
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:     "java:S1000",
				Impacts: []v1alpha1.QualityProfileRuleImpact{{SoftwareQuality: "SECURITY", Severity: "LOW"}},
			},
			want: false,
		},
		"SpecImpactsNilObservationHasImpactsReturnsTrue": {
			spec: &v1alpha1.QualityProfileRuleParameters{
				Rule: "java:S1000",
			},
			observation: &v1alpha1.QualityProfileRuleObservation{
				Key:     "java:S1000",
				Impacts: []v1alpha1.QualityProfileRuleImpact{{SoftwareQuality: "SECURITY", Severity: "HIGH"}},
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

func TestAreQualityProfileRuleImpactsUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *map[string]string
		observation []v1alpha1.QualityProfileRuleImpact
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: []v1alpha1.QualityProfileRuleImpact{{SoftwareQuality: "SECURITY", Severity: "HIGH"}},
			want:        true,
		},
		"EmptySpecEmptyObservationReturnsTrue": {
			spec:        &map[string]string{},
			observation: nil,
			want:        true,
		},
		"MatchingImpactsReturnsTrue": {
			spec: &map[string]string{"SECURITY": "HIGH", "MAINTAINABILITY": "LOW"},
			observation: []v1alpha1.QualityProfileRuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				{SoftwareQuality: "MAINTAINABILITY", Severity: "LOW"},
			},
			want: true,
		},
		"DifferentSeverityReturnsFalse": {
			spec: &map[string]string{"SECURITY": "HIGH"},
			observation: []v1alpha1.QualityProfileRuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "LOW"},
			},
			want: false,
		},
		"ExtraSpecImpactReturnsFalse": {
			spec: &map[string]string{"SECURITY": "HIGH", "MAINTAINABILITY": "LOW"},
			observation: []v1alpha1.QualityProfileRuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "HIGH"},
			},
			want: false,
		},
		"ExtraObservationImpactReturnsFalse": {
			spec: &map[string]string{"SECURITY": "HIGH"},
			observation: []v1alpha1.QualityProfileRuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				{SoftwareQuality: "MAINTAINABILITY", Severity: "LOW"},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := areQualityProfileRuleImpactsUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("areQualityProfileRuleImpactsUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFetchAllQualityProfileRules(t *testing.T) {
	t.Parallel()

	errSearch := errors.New("search error")

	tests := map[string]struct {
		rulesClient RulesClient
		profileKey  string
		wantErr     bool
		wantRules   int
	}{
		"SinglePageFetchesAllRules": {
			rulesClient: &mockRulesClient{
				SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return &sonar.RulesSearch{
						Rules: []sonar.RuleDetails{
							{Key: "java:S1000", Name: "Rule 1"},
							{Key: "java:S1001", Name: "Rule 2"},
						},
						Paging: sonar.Paging{
							Total: 2,
						},
						Actives: map[string][]sonar.RuleActivation{
							"java:S1000": {{QProfile: "profile-key", Severity: "MAJOR"}},
						},
					}, nil, nil
				},
			},
			profileKey: "profile-key",
			wantErr:    false,
			wantRules:  2,
		},
		"MultiPageFetchesAllRules": {
			rulesClient: func() RulesClient {
				callCount := 0

				return &mockRulesClient{
					SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
						callCount++
						if callCount == 1 {
							return &sonar.RulesSearch{
								Rules: []sonar.RuleDetails{
									{Key: "java:S1000", Name: "Rule 1"},
									{Key: "java:S1001", Name: "Rule 2"},
								},
								Paging: sonar.Paging{
									Total: 3,
								},
							}, nil, nil
						}

						return &sonar.RulesSearch{
							Rules: []sonar.RuleDetails{
								{Key: "java:S1002", Name: "Rule 3"},
							},
							Paging: sonar.Paging{
								Total: 3,
							},
						}, nil, nil
					},
				}
			}(),
			profileKey: "profile-key",
			wantErr:    false,
			wantRules:  3,
		},
		"SearchErrorReturnsError": {
			rulesClient: &mockRulesClient{
				SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return nil, nil, errSearch
				},
			},
			profileKey: "profile-key",
			wantErr:    true,
			wantRules:  0,
		},
		"EmptyRulesReturnsEmpty": {
			rulesClient: &mockRulesClient{
				SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return &sonar.RulesSearch{
						Rules:  nil,
						Paging: sonar.Paging{Total: 0},
					}, nil, nil
				},
			},
			profileKey: "profile-key",
			wantErr:    false,
			wantRules:  0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := FetchAllQualityProfileRules(tc.rulesClient, tc.profileKey)
			if (err != nil) != tc.wantErr {
				t.Errorf("FetchAllQualityProfileRules() error = %v, wantErr %v", err, tc.wantErr)

				return
			}

			if tc.wantErr {
				return
			}

			if got == nil {
				t.Fatal("FetchAllQualityProfileRules() returned nil, want non-nil")
			}

			if len(got.Rules) != tc.wantRules {
				t.Errorf("FetchAllQualityProfileRules() returned %d rules, want %d", len(got.Rules), tc.wantRules)
			}
		})
	}
}

func TestGenerateQualityProfileRuleObservationWithActivatedSettings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rule              sonar.RuleDetails
		activatedSettings *ruleActiveSettings
		want              v1alpha1.QualityProfileRuleObservation
	}{
		"NilActivatedSettings": {
			rule: sonar.RuleDetails{
				Key:  "java:S1000",
				Name: "Rule 1",
			},
			activatedSettings: nil,
			want: v1alpha1.QualityProfileRuleObservation{
				Key:     "java:S1000",
				Name:    "Rule 1",
				Impacts: []v1alpha1.QualityProfileRuleImpact{},
			},
		},
		"ActivatedSettingsWithSeverity": {
			rule: sonar.RuleDetails{
				Key:      "java:S1000",
				Name:     "Rule 1",
				Severity: "MINOR",
			},
			activatedSettings: &ruleActiveSettings{
				Severity: ptr.To("MAJOR"),
			},
			want: v1alpha1.QualityProfileRuleObservation{
				Key:      "java:S1000",
				Name:     "Rule 1",
				Severity: "MAJOR",
				Impacts:  []v1alpha1.QualityProfileRuleImpact{},
			},
		},
		"ActivatedSettingsWithParams": {
			rule: sonar.RuleDetails{
				Key:  "java:S1000",
				Name: "Rule 1",
			},
			activatedSettings: &ruleActiveSettings{
				Params: &map[string]string{"max": "10"},
			},
			want: v1alpha1.QualityProfileRuleObservation{
				Key:        "java:S1000",
				Name:       "Rule 1",
				Parameters: map[string]string{"max": "10"},
				Impacts:    []v1alpha1.QualityProfileRuleImpact{},
			},
		},
		"ActivatedSettingsWithImpacts": {
			rule: sonar.RuleDetails{
				Key:  "java:S1000",
				Name: "Rule 1",
			},
			activatedSettings: &ruleActiveSettings{
				Impacts: []v1alpha1.QualityProfileRuleImpact{
					{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				},
			},
			want: v1alpha1.QualityProfileRuleObservation{
				Key:  "java:S1000",
				Name: "Rule 1",
				Impacts: []v1alpha1.QualityProfileRuleImpact{
					{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				},
			},
		},
		"ActivatedSettingsWithPrioritized": {
			rule: sonar.RuleDetails{
				Key:  "java:S1000",
				Name: "Rule 1",
			},
			activatedSettings: &ruleActiveSettings{
				Prioritized: ptr.To(true),
			},
			want: v1alpha1.QualityProfileRuleObservation{
				Key:         "java:S1000",
				Name:        "Rule 1",
				Prioritized: true,
				Impacts:     []v1alpha1.QualityProfileRuleImpact{},
			},
		},
		"ActivatedSettingsWithAllFields": {
			rule: sonar.RuleDetails{
				Key:      "java:S1000",
				Name:     "Rule 1",
				Severity: "MINOR",
			},
			activatedSettings: &ruleActiveSettings{
				Severity:    ptr.To("CRITICAL"),
				Params:      &map[string]string{"max": "10"},
				Prioritized: ptr.To(true),
				Impacts: []v1alpha1.QualityProfileRuleImpact{
					{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				},
			},
			want: v1alpha1.QualityProfileRuleObservation{
				Key:         "java:S1000",
				Name:        "Rule 1",
				Severity:    "CRITICAL",
				Prioritized: true,
				Parameters:  map[string]string{"max": "10"},
				Impacts: []v1alpha1.QualityProfileRuleImpact{
					{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRuleObservation(tc.rule, tc.activatedSettings)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileRuleObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateRuleCreateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		parameters *v1alpha1.RuleParameters
		wantKey    string
		wantName   string
		wantTpl    string
	}{
		"NilParametersReturnsEmpty": {
			parameters: nil,
			wantKey:    "",
			wantName:   "",
			wantTpl:    "",
		},
		"FullParameters": {
			parameters: &v1alpha1.RuleParameters{
				Key:                 "my:rule",
				Name:                "My Rule",
				TemplateKey:         "java:template",
				MarkdownDescription: "some description",
				CleanCodeAttribute:  ptr.To("CLEAR"),
				Severity:            ptr.To("MAJOR"),
				Status:              ptr.To("READY"),
				Type:                ptr.To("BUG"),
				Impacts:             &map[string]string{"SECURITY": "HIGH"},
				Parameters: &map[string]v1alpha1.RuleParameterConfiguration{
					"param1": {DefaultValue: "val1"},
				},
			},
			wantKey:  "my:rule",
			wantName: "My Rule",
			wantTpl:  "java:template",
		},
		"NilOptionalFields": {
			parameters: &v1alpha1.RuleParameters{
				Key:                 "my:rule2",
				Name:                "Name2",
				TemplateKey:         "tpl2",
				MarkdownDescription: "desc",
			},
			wantKey:  "my:rule2",
			wantName: "Name2",
			wantTpl:  "tpl2",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleCreateOptions(tc.parameters)

			if got.CustomKey != tc.wantKey {
				t.Errorf("GenerateRuleCreateOptions().CustomKey = %v, want %v", got.CustomKey, tc.wantKey)
			}

			if got.Name != tc.wantName {
				t.Errorf("GenerateRuleCreateOptions().Name = %v, want %v", got.Name, tc.wantName)
			}

			if got.TemplateKey != tc.wantTpl {
				t.Errorf("GenerateRuleCreateOptions().TemplateKey = %v, want %v", got.TemplateKey, tc.wantTpl)
			}
		})
	}
}

func TestGenerateRuleUpdateOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key        string
		parameters *v1alpha1.RuleParameters
		wantKey    string
		wantName   string
	}{
		"NilParametersReturnsEmpty": {
			key:        "",
			parameters: nil,
			wantKey:    "",
			wantName:   "",
		},
		"FullParameters": {
			key: "my:rule",
			parameters: &v1alpha1.RuleParameters{
				Key:                        "my:rule",
				Name:                       "My Rule",
				TemplateKey:                "tpl",
				MarkdownDescription:        "desc",
				MarkdownNote:               ptr.To("note"),
				Severity:                   ptr.To("BLOCKER"),
				Status:                     ptr.To("DEPRECATED"),
				RemediationFnBaseEffort:    ptr.To("1h"),
				RemediationFnType:          ptr.To("LINEAR"),
				RemediationFyGapMultiplier: ptr.To("5min"),
				Impacts:                    &map[string]string{"MAINTAINABILITY": "LOW"},
				Tags:                       &[]string{"tag1", "tag2"},
				Parameters: &map[string]v1alpha1.RuleParameterConfiguration{
					"p": {DefaultValue: "v"},
				},
			},
			wantKey:  "my:rule",
			wantName: "My Rule",
		},
		"AllNilOptional": {
			key: "k",
			parameters: &v1alpha1.RuleParameters{
				Key:                 "k",
				Name:                "n",
				TemplateKey:         "t",
				MarkdownDescription: "d",
			},
			wantKey:  "k",
			wantName: "n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleUpdateOptions(tc.key, tc.parameters)

			if got.Key != tc.wantKey {
				t.Errorf("GenerateRuleUpdateOptions().Key = %v, want %v", got.Key, tc.wantKey)
			}

			if got.Name != tc.wantName {
				t.Errorf("GenerateRuleUpdateOptions().Name = %v, want %v", got.Name, tc.wantName)
			}
		})
	}
}

func TestGenerateRuleDeleteOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key     string
		wantKey string
	}{
		"ReturnsKey": {
			key:     "my:rule",
			wantKey: "my:rule",
		},
		"EmptyKey": {
			key:     "",
			wantKey: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleDeleteOptions(tc.key)

			if got.Key != tc.wantKey {
				t.Errorf("GenerateRuleDeleteOptions().Key = %v, want %v", got.Key, tc.wantKey)
			}
		})
	}
}

func TestGenerateRuleGetOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key     string
		wantKey string
	}{
		"ReturnsKey": {
			key:     "my:rule",
			wantKey: "my:rule",
		},
		"EmptyKey": {
			key:     "",
			wantKey: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleGetOptions(tc.key)

			if got.Key != tc.wantKey {
				t.Errorf("GenerateRuleGetOptions().Key = %v, want %v", got.Key, tc.wantKey)
			}
		})
	}
}

func TestGenerateRuleObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rule          *sonar.RulesShow
		wantKey       string
		wantName      string
		wantRemFnType string
	}{
		"NilRuleReturnsEmpty": {
			rule:          nil,
			wantKey:       "",
			wantName:      "",
			wantRemFnType: "",
		},
		"FullRule": {
			rule: &sonar.RulesShow{
				Rule: sonar.RuleDetails{
					Key:                        "java:S001",
					Name:                       "Test Rule",
					Severity:                   "MAJOR",
					Status:                     "READY",
					Type:                       "BUG",
					CleanCodeAttribute:         "CLEAR",
					CleanCodeAttributeCategory: "INTENTIONAL",
					RemFnBaseEffort:            "1h",
					RemFnGapMultiplier:         "5min",
					RemFnType:                  "LINEAR",
					RemFnOverloaded:            true,
					DefaultRemFnBaseEffort:     "2h",
					DefaultRemFnType:           "CONSTANT",
					DefaultRemFnGapMultiplier:  "0",
					InternalKey:                "S001",
					Lang:                       "java",
					LangName:                   "Java",
					GapDescription:             "gap desc",
					Repo:                       "java",
					Scope:                      "MAIN",
					TemplateKey:                "java:template",
					HTMLDesc:                   "<p>Test</p>",
					HTMLNote:                   "<p>Note</p>",
					MdNote:                     "Note",
					NoteLogin:                  "user",
					Tags:                       []any{"tag1", "tag2"},
					SysTags:                    []string{"systag"},
					IsTemplate:                 false,
					IsExternal:                 false,
					Template:                   false,
					Impacts: []sonar.RuleImpact{
						{SoftwareQuality: "SECURITY", Severity: "HIGH"},
					},
					Params: []sonar.RuleParam{
						{Key: "p1", DefaultValue: "dv", HTMLDesc: "html desc", Type: "STRING"},
					},
					DescriptionSections: []sonar.RuleDescriptionSection{
						{Content: "content", Key: "ROOT_CAUSE"},
					},
				},
			},
			wantKey:       "java:S001",
			wantName:      "Test Rule",
			wantRemFnType: "LINEAR",
		},
		"RuleWithNonStringTags": {
			rule: &sonar.RulesShow{
				Rule: sonar.RuleDetails{
					Key:  "java:S002",
					Name: "Rule 2",
					Tags: []any{"str-tag", 42, nil, "str2"},
				},
			},
			wantKey:       "java:S002",
			wantName:      "Rule 2",
			wantRemFnType: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleObservation(tc.rule)

			if got.Key != tc.wantKey {
				t.Errorf("GenerateRuleObservation().Key = %v, want %v", got.Key, tc.wantKey)
			}

			if got.Name != tc.wantName {
				t.Errorf("GenerateRuleObservation().Name = %v, want %v", got.Name, tc.wantName)
			}

			if got.RemFnType != tc.wantRemFnType {
				t.Errorf("GenerateRuleObservation().RemFnType = %v, want %v", got.RemFnType, tc.wantRemFnType)
			}
		})
	}
}

func TestGenerateRuleObservationTags(t *testing.T) {
	t.Parallel()

	rule := &sonar.RulesShow{
		Rule: sonar.RuleDetails{
			Key:  "java:S001",
			Tags: []any{"alpha", "beta"},
		},
	}

	got := GenerateRuleObservation(rule)

	if len(got.Tags) != 2 {
		t.Errorf("GenerateRuleObservation().Tags length = %d, want 2", len(got.Tags))
	}

	if got.Tags[0] != "alpha" || got.Tags[1] != "beta" {
		t.Errorf("GenerateRuleObservation().Tags = %v, want [alpha beta]", got.Tags)
	}
}

func TestGenerateRuleImpactsObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		impacts   *[]sonar.RuleImpact
		wantCount int
		wantMap   map[string]string
	}{
		"NilImpactsReturnsEmpty": {
			impacts:   nil,
			wantCount: 0,
			wantMap:   map[string]string{},
		},
		"EmptyImpacts": {
			impacts:   &[]sonar.RuleImpact{},
			wantCount: 0,
			wantMap:   map[string]string{},
		},
		"SingleImpact": {
			impacts: &[]sonar.RuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "HIGH"},
			},
			wantCount: 1,
			wantMap:   map[string]string{"SECURITY": "HIGH"},
		},
		"MultipleImpacts": {
			impacts: &[]sonar.RuleImpact{
				{SoftwareQuality: "SECURITY", Severity: "HIGH"},
				{SoftwareQuality: "MAINTAINABILITY", Severity: "LOW"},
				{SoftwareQuality: "RELIABILITY", Severity: "MEDIUM"},
			},
			wantCount: 3,
			wantMap:   map[string]string{"SECURITY": "HIGH", "MAINTAINABILITY": "LOW", "RELIABILITY": "MEDIUM"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleImpactsObservation(tc.impacts)

			if len(got) != tc.wantCount {
				t.Errorf("GenerateRuleImpactsObservation() returned %d impacts, want %d", len(got), tc.wantCount)
			}

			if diff := cmp.Diff(tc.wantMap, got); diff != "" {
				t.Errorf("GenerateRuleImpactsObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateRuleParametersObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rule      *[]sonar.RuleParam
		wantCount int
	}{
		"NilRuleReturnsEmpty": {
			rule:      nil,
			wantCount: 0,
		},
		"EmptyParams": {
			rule:      &[]sonar.RuleParam{},
			wantCount: 0,
		},
		"SingleParam": {
			rule: &[]sonar.RuleParam{
				{Key: "p1", DefaultValue: "dv", HTMLDesc: "html", Desc: "desc", Type: "STRING"},
			},
			wantCount: 1,
		},
		"MultipleParams": {
			rule: &[]sonar.RuleParam{
				{Key: "p1", DefaultValue: "1", Type: "INTEGER"},
				{Key: "p2", DefaultValue: "text", Type: "STRING"},
				{Key: "p3", DefaultValue: "true", Type: "BOOLEAN"},
			},
			wantCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleParametersObservation(tc.rule)

			if len(got) != tc.wantCount {
				t.Errorf("GenerateRuleParametersObservation() returned %d params, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestGenerateRuleParametersObservationFields(t *testing.T) {
	t.Parallel()

	rule := &[]sonar.RuleParam{
		{Key: "myParam", DefaultValue: "myDefault", HTMLDesc: "myHTML", Desc: "myDesc", Type: "TEXT"},
	}

	got := GenerateRuleParametersObservation(rule)

	if len(got) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(got))
	}

	want := v1alpha1.RuleParameterObservation{
		Key:          "myParam",
		DefaultValue: "myDefault",
		HTMLDesc:     "myHTML",
		Desc:         "myDesc",
		Type:         "TEXT",
	}

	if diff := cmp.Diff(want, got[0]); diff != "" {
		t.Errorf("GenerateRuleParametersObservation()[0] mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateRuleDescriptionSectionsObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sections  *[]sonar.RuleDescriptionSection
		wantCount int
	}{
		"NilSectionsReturnsEmpty": {
			sections:  nil,
			wantCount: 0,
		},
		"EmptySections": {
			sections:  &[]sonar.RuleDescriptionSection{},
			wantCount: 0,
		},
		"SingleSection": {
			sections: &[]sonar.RuleDescriptionSection{
				{Content: "content", Key: "ROOT_CAUSE"},
			},
			wantCount: 1,
		},
		"MultipleSections": {
			sections: &[]sonar.RuleDescriptionSection{
				{Content: "c1", Key: "ROOT_CAUSE"},
				{Content: "c2", Key: "HOW_TO_FIX"},
				{Content: "c3", Key: "ASSESS_THE_PROBLEM"},
			},
			wantCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleDescriptionSectionsObservation(tc.sections)

			if len(got) != tc.wantCount {
				t.Errorf("GenerateRuleDescriptionSectionsObservation() returned %d sections, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestGenerateRuleDescriptionSectionsObservationWithContext(t *testing.T) {
	t.Parallel()

	sections := &[]sonar.RuleDescriptionSection{
		{
			Content: "some content",
			Key:     "ROOT_CAUSE",
			Context: sonar.RuleDescriptionSectionContext{
				DisplayName: "Spring",
				Key:         "spring",
			},
		},
	}

	got := GenerateRuleDescriptionSectionsObservation(sections)

	if len(got) != 1 {
		t.Fatalf("Expected 1 section, got %d", len(got))
	}

	if got[0].Context.DisplayName != "Spring" {
		t.Errorf("Expected context DisplayName 'Spring', got %q", got[0].Context.DisplayName)
	}

	if got[0].Context.Key != "spring" {
		t.Errorf("Expected context Key 'spring', got %q", got[0].Context.Key)
	}
}

func TestGenerateRuleDescriptionContextObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		context     *sonar.RuleDescriptionSectionContext
		wantDisplay string
		wantKey     string
	}{
		"NilContextReturnsEmpty": {
			context:     nil,
			wantDisplay: "",
			wantKey:     "",
		},
		"ValidContext": {
			context: &sonar.RuleDescriptionSectionContext{
				DisplayName: "Spring Boot",
				Key:         "spring-boot",
			},
			wantDisplay: "Spring Boot",
			wantKey:     "spring-boot",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateRuleDescriptionContextObservation(tc.context)

			if got.DisplayName != tc.wantDisplay {
				t.Errorf("GenerateRuleDescriptionContextObservation().DisplayName = %v, want %v", got.DisplayName, tc.wantDisplay)
			}

			if got.Key != tc.wantKey {
				t.Errorf("GenerateRuleDescriptionContextObservation().Key = %v, want %v", got.Key, tc.wantKey)
			}
		})
	}
}

func TestIsRuleUpToDate(t *testing.T) { //nolint:maintidx // comprehensive table-driven test covering many field combinations
	t.Parallel()

	type fields struct {
		spec        *v1alpha1.RuleParameters
		observation *v1alpha1.RuleObservation
	}

	tests := map[string]struct {
		fields fields
		want   bool
	}{
		"NilSpecReturnsTrue": {
			fields: fields{
				spec:        nil,
				observation: &v1alpha1.RuleObservation{Key: "k"},
			},
			want: true,
		},
		"NilObservationReturnsFalse": {
			fields: fields{
				spec:        &v1alpha1.RuleParameters{Key: "k", Name: "n", TemplateKey: "t"},
				observation: nil,
			},
			want: false,
		},
		"AllMatchingFieldsReturnsTrue": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "java:S001",
					Name:        "Test Rule",
					TemplateKey: "java:template",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "java:S001",
					Name:        "Test Rule",
					TemplateKey: "java:template",
				},
			},
			want: true,
		},
		"DifferentKeyReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "java:custom-rule-1",
					Name:        "Test",
					TemplateKey: "t",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "java:custom-rule-2",
					Name:        "Test",
					TemplateKey: "t",
				},
			},
			want: false,
		},
		"UnprefixedSpecKeyMatchesPrefixedObservationKey": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "custom-rule-1",
					Name:        "Test",
					TemplateKey: "t",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "java:custom-rule-1",
					Name:        "Test",
					TemplateKey: "t",
				},
			},
			want: true,
		},
		"DifferentNameReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "Name1",
					TemplateKey: "t",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "Name2",
					TemplateKey: "t",
				},
			},
			want: false,
		},
		"DifferentTemplateKeyReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "tpl1",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "tpl2",
				},
			},
			want: false,
		},
		"MatchingCleanCodeAttribute": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					CleanCodeAttribute: ptr.To("CLEAR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					CleanCodeAttribute: "CLEAR",
				},
			},
			want: true,
		},
		"DifferentCleanCodeAttributeReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					CleanCodeAttribute: ptr.To("CLEAR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					CleanCodeAttribute: "COMPLETE",
				},
			},
			want: false,
		},
		"MatchingImpacts": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Impacts:     &map[string]string{"SECURITY": "HIGH"},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Impacts:     map[string]string{"SECURITY": "HIGH"},
				},
			},
			want: true,
		},
		"DifferentImpactsReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Impacts:     &map[string]string{"SECURITY": "HIGH"},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Impacts:     map[string]string{"SECURITY": "LOW"},
				},
			},
			want: false,
		},
		"MatchingSeverity": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Severity:    ptr.To("MAJOR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Severity:    "MAJOR",
				},
			},
			want: true,
		},
		"DifferentSeverityReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Severity:    ptr.To("MAJOR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Severity:    "MINOR",
				},
			},
			want: false,
		},
		"MatchingStatus": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Status:      ptr.To("READY"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Status:      "READY",
				},
			},
			want: true,
		},
		"DifferentStatusReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Status:      ptr.To("READY"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Status:      "DEPRECATED",
				},
			},
			want: false,
		},
		"MatchingRemFnBaseEffort": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                     "k",
					Name:                    "n",
					TemplateKey:             "t",
					RemediationFnBaseEffort: ptr.To("1h"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:             "k",
					Name:            "n",
					TemplateKey:     "t",
					RemFnBaseEffort: "1h",
				},
			},
			want: true,
		},
		"DifferentRemFnBaseEffortReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                     "k",
					Name:                    "n",
					TemplateKey:             "t",
					RemediationFnBaseEffort: ptr.To("1h"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:             "k",
					Name:            "n",
					TemplateKey:     "t",
					RemFnBaseEffort: "2h",
				},
			},
			want: false,
		},
		"MatchingRemFnType": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:               "k",
					Name:              "n",
					TemplateKey:       "t",
					RemediationFnType: ptr.To("LINEAR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					RemFnType:   "LINEAR",
				},
			},
			want: true,
		},
		"DifferentRemFnTypeReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:               "k",
					Name:              "n",
					TemplateKey:       "t",
					RemediationFnType: ptr.To("LINEAR"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					RemFnType:   "CONSTANT",
				},
			},
			want: false,
		},
		"MatchingRemFnGapMultiplier": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                        "k",
					Name:                       "n",
					TemplateKey:                "t",
					RemediationFyGapMultiplier: ptr.To("5min"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					RemFnGapMultiplier: "5min",
				},
			},
			want: true,
		},
		"DifferentRemFnGapMultiplierReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:                        "k",
					Name:                       "n",
					TemplateKey:                "t",
					RemediationFyGapMultiplier: ptr.To("5min"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:                "k",
					Name:               "n",
					TemplateKey:        "t",
					RemFnGapMultiplier: "10min",
				},
			},
			want: false,
		},
		"MatchingType": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Type:        ptr.To("BUG"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Type:        "BUG",
				},
			},
			want: true,
		},
		"DifferentTypeReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Type:        ptr.To("BUG"),
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Type:        "CODE_SMELL",
				},
			},
			want: false,
		},
		"MatchingTags": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        &[]string{"tag1", "tag2"},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        []string{"tag1", "tag2"},
				},
			},
			want: true,
		},
		"MatchingTagsDifferentOrder": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        &[]string{"crossplane", "example", "integration"},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        []string{"crossplane", "integration", "example"},
				},
			},
			want: true,
		},
		"DifferentTagsReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        &[]string{"tag1"},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        []string{"tag2"},
				},
			},
			want: false,
		},
		"NilSpecTagsIgnored": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Tags:        []string{"tag1", "tag2"},
				},
			},
			want: true,
		},
		"MatchingParameters": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Parameters: &map[string]v1alpha1.RuleParameterConfiguration{
						"p1": {DefaultValue: "val1", Type: "STRING"},
					},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Params: []v1alpha1.RuleParameterObservation{
						{Key: "p1", DefaultValue: "val1", Type: "STRING"},
					},
				},
			},
			want: true,
		},
		"DifferentParametersReturnsFalse": {
			fields: fields{
				spec: &v1alpha1.RuleParameters{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Parameters: &map[string]v1alpha1.RuleParameterConfiguration{
						"p1": {DefaultValue: "val1", Type: "STRING"},
					},
				},
				observation: &v1alpha1.RuleObservation{
					Key:         "k",
					Name:        "n",
					TemplateKey: "t",
					Params: []v1alpha1.RuleParameterObservation{
						{Key: "p1", DefaultValue: "val2", Type: "STRING"},
					},
				},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsRuleUpToDate(tc.fields.spec, tc.fields.observation)

			if got != tc.want {
				t.Errorf("IsRuleUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAreRuleParametersUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *map[string]v1alpha1.RuleParameterConfiguration
		observation *[]v1alpha1.RuleParameterObservation
		want        bool
	}{
		"NilSpecReturnsTrue": {
			spec:        nil,
			observation: &[]v1alpha1.RuleParameterObservation{{Key: "p1"}},
			want:        true,
		},
		"NilObservationReturnsFalse": {
			spec:        &map[string]v1alpha1.RuleParameterConfiguration{"p1": {DefaultValue: "v"}},
			observation: nil,
			want:        false,
		},
		"DifferentLengthReturnsFalse": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"p1": {DefaultValue: "v1"},
				"p2": {DefaultValue: "v2"},
			},
			observation: &[]v1alpha1.RuleParameterObservation{
				{Key: "p1", DefaultValue: "v1"},
			},
			want: false,
		},
		"MatchingParametersReturnsTrue": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"p1": {DefaultValue: "v1", Type: "STRING"},
			},
			observation: &[]v1alpha1.RuleParameterObservation{
				{Key: "p1", DefaultValue: "v1", Type: "STRING"},
			},
			want: true,
		},
		"DifferentDefaultValueReturnsFalse": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"p1": {DefaultValue: "v1"},
			},
			observation: &[]v1alpha1.RuleParameterObservation{
				{Key: "p1", DefaultValue: "v2"},
			},
			want: false,
		},
		"DifferentTypeReturnsFalse": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"p1": {DefaultValue: "v1", Type: "STRING"},
			},
			observation: &[]v1alpha1.RuleParameterObservation{
				{Key: "p1", DefaultValue: "v1", Type: "INTEGER"},
			},
			want: false,
		},
		"EmptySpecEmptyObservationReturnsTrue": {
			spec:        &map[string]v1alpha1.RuleParameterConfiguration{},
			observation: &[]v1alpha1.RuleParameterObservation{},
			want:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := areRuleParametersUpToDate(tc.spec, tc.observation)

			if got != tc.want {
				t.Errorf("areRuleParametersUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLateInitializeRule(t *testing.T) { //nolint:gocognit,maintidx // complexity from comprehensive parallel sub-tests
	t.Parallel()

	tests := map[string]struct {
		spec        *v1alpha1.RuleParameters
		observation *v1alpha1.RuleObservation
		checkFn     func(t *testing.T, spec *v1alpha1.RuleParameters)
	}{
		"NilSpecNoOp": {
			spec:        nil,
			observation: &v1alpha1.RuleObservation{Severity: "MAJOR"},
			checkFn:     func(t *testing.T, _ *v1alpha1.RuleParameters) { t.Helper() },
		},
		"NilObservationNoOp": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: nil,
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Severity != nil {
					t.Errorf("Expected Severity to remain nil, got %v", spec.Severity)
				}
			},
		},
		"InitializesSeverity": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{Severity: "MAJOR"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Severity == nil || *spec.Severity != "MAJOR" {
					t.Errorf("Expected Severity to be initialized to 'MAJOR', got %v", spec.Severity)
				}
			},
		},
		"DoesNotOverrideSeverity": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
				Severity:    ptr.To("MINOR"),
			},
			observation: &v1alpha1.RuleObservation{Severity: "MAJOR"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Severity == nil || *spec.Severity != "MINOR" {
					t.Errorf("Expected Severity to remain 'MINOR', got %v", spec.Severity)
				}
			},
		},
		"InitializesStatus": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{Status: "READY"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Status == nil || *spec.Status != "READY" {
					t.Errorf("Expected Status to be initialized to 'READY', got %v", spec.Status)
				}
			},
		},
		"InitializesCleanCodeAttribute": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{CleanCodeAttribute: "CLEAR"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.CleanCodeAttribute == nil || *spec.CleanCodeAttribute != "CLEAR" {
					t.Errorf("Expected CleanCodeAttribute to be initialized to 'CLEAR', got %v", spec.CleanCodeAttribute)
				}
			},
		},
		"InitializesMarkdownNote": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{MdNote: "mynote"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.MarkdownNote == nil || *spec.MarkdownNote != "mynote" {
					t.Errorf("Expected MarkdownNote to be initialized to 'mynote', got %v", spec.MarkdownNote)
				}
			},
		},
		"InitializesRemediationFnBaseEffort": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{RemFnBaseEffort: "1h"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.RemediationFnBaseEffort == nil || *spec.RemediationFnBaseEffort != "1h" {
					t.Errorf("Expected RemediationFnBaseEffort to be '1h', got %v", spec.RemediationFnBaseEffort)
				}
			},
		},
		"InitializesRemediationFnType": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{RemFnType: "LINEAR"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.RemediationFnType == nil || *spec.RemediationFnType != "LINEAR" {
					t.Errorf("Expected RemediationFnType to be 'LINEAR', got %v", spec.RemediationFnType)
				}
			},
		},
		"InitializesRemediationFyGapMultiplier": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{RemFnGapMultiplier: "5min"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.RemediationFyGapMultiplier == nil || *spec.RemediationFyGapMultiplier != "5min" {
					t.Errorf("Expected RemediationFyGapMultiplier to be '5min', got %v", spec.RemediationFyGapMultiplier)
				}
			},
		},
		"InitializesType": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{Type: "BUG"},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Type == nil || *spec.Type != "BUG" {
					t.Errorf("Expected Type to be initialized to 'BUG', got %v", spec.Type)
				}
			},
		},
		"InitializesImpacts": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{
				Impacts: map[string]string{"SECURITY": "HIGH"},
			},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Impacts == nil {
					t.Error("Expected Impacts to be initialized, got nil")

					return
				}

				imp := *spec.Impacts
				if imp["SECURITY"] != "HIGH" {
					t.Errorf("Expected impact SECURITY=HIGH, got %v", imp)
				}
			},
		},
		"DoesNotOverrideImpacts": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
				Impacts:     &map[string]string{"MAINTAINABILITY": "LOW"},
			},
			observation: &v1alpha1.RuleObservation{
				Impacts: map[string]string{"SECURITY": "HIGH"},
			},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				imp := *spec.Impacts
				if _, ok := imp["MAINTAINABILITY"]; !ok {
					t.Error("Expected Impacts to remain MAINTAINABILITY, got overridden")
				}
			},
		},
		"InitializesTags": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{
				Tags: []string{"tag1", "tag2"},
			},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Tags == nil {
					t.Error("Expected Tags to be initialized, got nil")

					return
				}

				if len(*spec.Tags) != 2 {
					t.Errorf("Expected 2 tags, got %d", len(*spec.Tags))
				}
			},
		},
		"DoesNotOverrideTags": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
				Tags:        &[]string{"existing"},
			},
			observation: &v1alpha1.RuleObservation{
				Tags: []string{"tag1", "tag2"},
			},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if len(*spec.Tags) != 1 || (*spec.Tags)[0] != "existing" {
					t.Errorf("Expected Tags to remain ['existing'], got %v", *spec.Tags)
				}
			},
		},
		"EmptyObservationTagsDoesNotInitialize": {
			spec: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			observation: &v1alpha1.RuleObservation{Tags: []string{}},
			checkFn: func(t *testing.T, spec *v1alpha1.RuleParameters) {
				t.Helper()

				if spec.Tags != nil {
					t.Errorf("Expected Tags to remain nil, got %v", spec.Tags)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeRule(tc.spec, tc.observation)
			tc.checkFn(t, tc.spec)
		})
	}
}

func TestIsRuleLateInitialized(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		before *v1alpha1.RuleParameters
		after  *v1alpha1.RuleParameters
		want   bool
	}{
		"NilBeforeNilAfterReturnsFalse": {
			before: &v1alpha1.RuleParameters{Key: "k", Name: "n", TemplateKey: "t"},
			after:  &v1alpha1.RuleParameters{Key: "k", Name: "n", TemplateKey: "t"},
			want:   false,
		},
		"SeverityAddedReturnsTrue": {
			before: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
			},
			after: &v1alpha1.RuleParameters{
				Key:         "k",
				Name:        "n",
				TemplateKey: "t",
				Severity:    ptr.To("MAJOR"),
			},
			want: true,
		},
		"NothingChangedReturnsFalse": {
			before: &v1alpha1.RuleParameters{
				Key:      "k",
				Name:     "n",
				Severity: ptr.To("MAJOR"),
			},
			after: &v1alpha1.RuleParameters{
				Key:      "k",
				Name:     "n",
				Severity: ptr.To("MAJOR"),
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsRuleLateInitialized(tc.before, tc.after)

			if got != tc.want {
				t.Errorf("IsRuleLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateRuleParametersOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec      *map[string]v1alpha1.RuleParameterConfiguration
		wantKeys  []string
		wantEmpty bool
	}{
		"NilSpecReturnsEmpty": {
			spec:      nil,
			wantEmpty: true,
		},
		"EmptySpecReturnsEmpty": {
			spec:      &map[string]v1alpha1.RuleParameterConfiguration{},
			wantEmpty: true,
		},
		"SingleParameter": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"maxLine": {DefaultValue: "100"},
			},
			wantKeys:  []string{"maxLine"},
			wantEmpty: false,
		},
		"MultipleParameters": {
			spec: &map[string]v1alpha1.RuleParameterConfiguration{
				"maxLine": {DefaultValue: "100"},
				"minLine": {DefaultValue: "1"},
			},
			wantKeys:  []string{"maxLine", "minLine"},
			wantEmpty: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := generateRuleParametersOptions(tc.spec)

			if tc.wantEmpty && len(got) != 0 {
				t.Errorf("generateRuleParametersOptions() want empty map, got %v", got)
			}

			for _, key := range tc.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("generateRuleParametersOptions() missing key %q", key)
				}
			}
		})
	}
}

func TestAnySliceToStringSlice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input []any
		want  []string
	}{
		"NilInputReturnsNil": {
			input: nil,
			want:  nil,
		},
		"EmptyInput": {
			input: []any{},
			want:  []string{},
		},
		"AllStrings": {
			input: []any{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		"NonStringElementsSkipped": {
			input: []any{"a", 42, nil, "b", true},
			want:  []string{"a", "b"},
		},
		"AllNonStrings": {
			input: []any{1, 2, 3},
			want:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := anySliceToStringSlice(tc.input)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("anySliceToStringSlice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateQualityProfileRulesObservationWithActives(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		qualityProfileId string
		rules            *sonar.RulesSearch
		want             []v1alpha1.QualityProfileRuleObservation
	}{
		"RulesWithActivesForProfile": {
			qualityProfileId: "profile-1",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{
					{Key: "java:S1000", Name: "Rule 1"},
				},
				Actives: map[string][]sonar.RuleActivation{
					"java:S1000": {
						{
							QProfile:        "profile-1",
							Severity:        "MAJOR",
							PrioritizedRule: true,
							Params: []sonar.ParamKV{
								{Key: "max", Value: "10"},
							},
							Impacts: []sonar.RuleImpact{
								{SoftwareQuality: "SECURITY", Severity: "HIGH"},
							},
						},
					},
				},
			},
			want: []v1alpha1.QualityProfileRuleObservation{
				{
					Key:         "java:S1000",
					Name:        "Rule 1",
					Severity:    "MAJOR",
					Prioritized: true,
					Parameters:  map[string]string{"max": "10"},
					Impacts:     []v1alpha1.QualityProfileRuleImpact{{SoftwareQuality: "SECURITY", Severity: "HIGH"}},
				},
			},
		},
		"RulesWithActivesForDifferentProfile": {
			qualityProfileId: "profile-1",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{
					{Key: "java:S1000", Name: "Rule 1"},
				},
				Actives: map[string][]sonar.RuleActivation{
					"java:S1000": {
						{
							QProfile: "profile-2",
							Severity: "MAJOR",
						},
					},
				},
			},
			want: []v1alpha1.QualityProfileRuleObservation{
				{
					Key:     "java:S1000",
					Name:    "Rule 1",
					Impacts: []v1alpha1.QualityProfileRuleImpact{},
				},
			},
		},
		"RulesWithNoActives": {
			qualityProfileId: "profile-1",
			rules: &sonar.RulesSearch{
				Rules: []sonar.RuleDetails{
					{Key: "java:S1000", Name: "Rule 1"},
				},
				Actives: map[string][]sonar.RuleActivation{},
			},
			want: []v1alpha1.QualityProfileRuleObservation{
				{
					Key:     "java:S1000",
					Name:    "Rule 1",
					Impacts: []v1alpha1.QualityProfileRuleImpact{},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileRulesObservation(tc.qualityProfileId, tc.rules)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileRulesObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
