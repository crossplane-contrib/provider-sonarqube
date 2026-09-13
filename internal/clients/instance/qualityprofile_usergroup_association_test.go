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

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// TestNewQualityProfileUsergroupAssociationClient tests creating an
// association client from a config.
func TestNewQualityProfileUsergroupAssociationClient(t *testing.T) {
	t.Parallel()

	client := NewQualityProfileUsergroupAssociationClient(newTestConfig())
	if client == nil {
		t.Error("NewQualityProfileUsergroupAssociationClient() returned nil")
	}
}

// TestBuildAndParseQualityProfileUsergroupAssociationExternalName tests
// building and parsing association external names.
func TestBuildAndParseQualityProfileUsergroupAssociationExternalName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params             *v1alpha1.QualityProfileUsergroupAssociationParameters
		wantBuilt          string
		wantType           string
		wantSubject        string
		wantLanguage       string
		wantQualityProfile string
		skipParse          bool
	}{
		"GroupRoundTrip": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "Sonar way",
				Language:       "go",
				GroupName:      new("sonar-users"),
			},
			wantBuilt:          "group:sonar-users:go:Sonar way",
			wantType:           SubjectTypeGroup,
			wantSubject:        "sonar-users",
			wantLanguage:       "go",
			wantQualityProfile: "Sonar way",
		},
		"UserRoundTrip": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "MyProfile",
				Language:       "java",
				Login:          new("alice"),
			},
			wantBuilt:          "user:alice:java:MyProfile",
			wantType:           SubjectTypeUser,
			wantSubject:        "alice",
			wantLanguage:       "java",
			wantQualityProfile: "MyProfile",
		},
		"QualityProfileContainingColon": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "org:team:profile",
				Language:       "go",
				GroupName:      new("devs"),
			},
			wantBuilt:          "group:devs:go:org:team:profile",
			wantType:           SubjectTypeGroup,
			wantSubject:        "devs",
			wantLanguage:       "go",
			wantQualityProfile: "org:team:profile",
		},
		"UserQualityProfileContainingColon": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "a:b",
				Language:       "py",
				Login:          new("bob"),
			},
			wantBuilt:          "user:bob:py:a:b",
			wantType:           SubjectTypeUser,
			wantSubject:        "bob",
			wantLanguage:       "py",
			wantQualityProfile: "a:b",
		},
		"NoPrincipalReturnsEmpty": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "MyProfile",
				Language:       "go",
			},
			wantBuilt: "",
			skipParse: true,
		},
		"NilParamsReturnsEmpty": {
			params:    nil,
			wantBuilt: "",
			skipParse: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotBuilt := BuildQualityProfileUsergroupAssociationExternalName(tc.params)
			if diff := cmp.Diff(tc.wantBuilt, gotBuilt); diff != "" {
				t.Errorf("BuildQualityProfileUsergroupAssociationExternalName() mismatch (-want +got):\n%s", diff)
			}

			if tc.skipParse {
				return
			}

			gotType, gotSubject, gotLanguage, gotProfile, err := ParseQualityProfileUsergroupAssociationExternalName(gotBuilt)
			if err != nil {
				t.Fatalf("ParseQualityProfileUsergroupAssociationExternalName() unexpected error: %v", err)
			}

			if gotType != tc.wantType || gotSubject != tc.wantSubject || gotLanguage != tc.wantLanguage || gotProfile != tc.wantQualityProfile {
				t.Errorf("ParseQualityProfileUsergroupAssociationExternalName() = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					gotType, gotSubject, gotLanguage, gotProfile, tc.wantType, tc.wantSubject, tc.wantLanguage, tc.wantQualityProfile)
			}
		})
	}
}

// TestParseQualityProfileUsergroupAssociationExternalNameFailures tests
// parse failures for empty names, wrong part counts, and unknown types.
func TestParseQualityProfileUsergroupAssociationExternalNameFailures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		externalName string
	}{
		"Empty": {
			externalName: "",
		},
		"NoColon": {
			externalName: "just-a-name",
		},
		"TwoParts": {
			externalName: "group:devs",
		},
		"ThreeParts": {
			externalName: "group:devs:go",
		},
		"UnknownType": {
			externalName: "robot:r2d2:go:MyProfile",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotType, gotSubject, gotLanguage, gotProfile, err := ParseQualityProfileUsergroupAssociationExternalName(tc.externalName)
			if err == nil {
				t.Fatalf("ParseQualityProfileUsergroupAssociationExternalName(%q) expected error, got (%q, %q, %q, %q)",
					tc.externalName, gotType, gotSubject, gotLanguage, gotProfile)
			}

			if gotType != "" || gotSubject != "" || gotLanguage != "" || gotProfile != "" {
				t.Errorf("ParseQualityProfileUsergroupAssociationExternalName() on error = (%q, %q, %q, %q), want empty",
					gotType, gotSubject, gotLanguage, gotProfile)
			}
		})
	}
}

// TestGenerateQualityProfileUsergroupAssociationObservation tests
// observation generation from spec parameters.
func TestGenerateQualityProfileUsergroupAssociationObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params *v1alpha1.QualityProfileUsergroupAssociationParameters
		want   v1alpha1.QualityProfileUsergroupAssociationObservation
	}{
		"NilParams": {
			params: nil,
			want:   v1alpha1.QualityProfileUsergroupAssociationObservation{},
		},
		"GroupPrincipal": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "Sonar way",
				Language:       "go",
				GroupName:      new("sonar-users"),
			},
			want: v1alpha1.QualityProfileUsergroupAssociationObservation{
				QualityProfile: "Sonar way",
				Language:       "go",
				GroupName:      "sonar-users",
			},
		},
		"UserPrincipal": {
			params: &v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: "MyProfile",
				Language:       "java",
				Login:          new("alice"),
			},
			want: v1alpha1.QualityProfileUsergroupAssociationObservation{
				QualityProfile: "MyProfile",
				Language:       "java",
				Login:          "alice",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityProfileUsergroupAssociationObservation(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityProfileUsergroupAssociationObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsQualityProfileUsergroupAssociationUpToDate tests association
// up-to-date comparison between spec and observation.
func TestIsQualityProfileUsergroupAssociationUpToDate(t *testing.T) {
	t.Parallel()

	groupSpec := &v1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: "Sonar way",
		Language:       "go",
		GroupName:      new("sonar-users"),
	}
	groupObs := &v1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: "Sonar way",
		Language:       "go",
		GroupName:      "sonar-users",
	}
	userSpec := &v1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: "Sonar way",
		Language:       "go",
		Login:          new("alice"),
	}
	userObs := &v1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: "Sonar way",
		Language:       "go",
		Login:          "alice",
	}

	tests := map[string]struct {
		spec        *v1alpha1.QualityProfileUsergroupAssociationParameters
		observation *v1alpha1.QualityProfileUsergroupAssociationObservation
		want        bool
	}{
		"NilSpec": {
			spec:        nil,
			observation: groupObs,
			want:        true,
		},
		"NilObservation": {
			spec:        groupSpec,
			observation: nil,
			want:        false,
		},
		"GroupMatch": {
			spec:        groupSpec,
			observation: groupObs,
			want:        true,
		},
		"UserMatch": {
			spec:        userSpec,
			observation: userObs,
			want:        true,
		},
		"QualityProfileMismatch": {
			spec: groupSpec,
			observation: &v1alpha1.QualityProfileUsergroupAssociationObservation{
				QualityProfile: "other-profile",
				Language:       "go",
				GroupName:      "sonar-users",
			},
			want: false,
		},
		"LanguageMismatch": {
			spec: groupSpec,
			observation: &v1alpha1.QualityProfileUsergroupAssociationObservation{
				QualityProfile: "Sonar way",
				Language:       "java",
				GroupName:      "sonar-users",
			},
			want: false,
		},
		"GroupValueMismatch": {
			spec: groupSpec,
			observation: &v1alpha1.QualityProfileUsergroupAssociationObservation{
				QualityProfile: "Sonar way",
				Language:       "go",
				GroupName:      "admins",
			},
			want: false,
		},
		"PrincipalKindMismatch": {
			spec:        groupSpec,
			observation: userObs,
			want:        false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsQualityProfileUsergroupAssociationUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsQualityProfileUsergroupAssociationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateQualityProfileAssociationOptions tests generating add,
// remove, and search options for groups and users.
func TestGenerateQualityProfileAssociationOptions(t *testing.T) {
	t.Parallel()

	t.Run("AddGroup", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileAddGroupOptions("go", "MyProfile", "devs")
		want := &sonar.QualityprofilesAddGroupOptions{
			Group:          "devs",
			Language:       "go",
			QualityProfile: "MyProfile",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileAddGroupOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("AddUser", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileAddUserOptions("go", "MyProfile", "alice")
		want := &sonar.QualityprofilesAddUserOptions{
			Language:       "go",
			Login:          "alice",
			QualityProfile: "MyProfile",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileAddUserOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RemoveGroup", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileRemoveGroupOptions("go", "MyProfile", "devs")
		want := &sonar.QualityprofilesRemoveGroupOptions{
			Group:          "devs",
			Language:       "go",
			QualityProfile: "MyProfile",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileRemoveGroupOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RemoveUser", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileRemoveUserOptions("go", "MyProfile", "alice")
		want := &sonar.QualityprofilesRemoveUserOptions{
			Language:       "go",
			Login:          "alice",
			QualityProfile: "MyProfile",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileRemoveUserOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SearchGroupsNilPagination", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileSearchGroupsOptions("go", "MyProfile", "devs", nil)
		want := &sonar.QualityprofilesSearchGroupsOptions{
			Language:       "go",
			QualityProfile: "MyProfile",
			Query:          "devs",
			Selected:       "all",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileSearchGroupsOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SearchGroupsWithPagination", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileSearchGroupsOptions("go", "MyProfile", "devs", &sonar.PaginationArgs{Page: 2, PageSize: 100})
		want := &sonar.QualityprofilesSearchGroupsOptions{
			PaginationArgs: sonar.PaginationArgs{Page: 2, PageSize: 100},
			Language:       "go",
			QualityProfile: "MyProfile",
			Query:          "devs",
			Selected:       "all",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileSearchGroupsOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SearchUsersWithPagination", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityProfileSearchUsersOptions("go", "MyProfile", "alice", &sonar.PaginationArgs{Page: 2, PageSize: 100})
		want := &sonar.QualityprofilesSearchUsersOptions{
			PaginationArgs: sonar.PaginationArgs{Page: 2, PageSize: 100},
			Language:       "go",
			QualityProfile: "MyProfile",
			Query:          "alice",
			Selected:       "all",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityProfileSearchUsersOptions() mismatch (-want +got):\n%s", diff)
		}
	})
}
