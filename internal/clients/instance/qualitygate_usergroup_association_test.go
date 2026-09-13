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

// TestNewQualityGateUsergroupAssociationClient tests creating an
// association client from a config.
func TestNewQualityGateUsergroupAssociationClient(t *testing.T) {
	t.Parallel()

	client := NewQualityGateUsergroupAssociationClient(newTestConfig())
	if client == nil {
		t.Error("NewQualityGateUsergroupAssociationClient() returned nil")
	}
}

// TestBuildAndParseQualityGateUsergroupAssociationExternalName tests
// building and parsing association external names.
func TestBuildAndParseQualityGateUsergroupAssociationExternalName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params       *v1alpha1.QualityGateUsergroupAssociationParameters
		wantBuilt    string
		wantType     string
		wantSubject  string
		wantGateName string
		skipParse    bool
	}{
		"GroupRoundTrip": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName:  "Sonar way",
				GroupName: new("sonar-users"),
			},
			wantBuilt:    "group:sonar-users:Sonar way",
			wantType:     SubjectTypeGroup,
			wantSubject:  "sonar-users",
			wantGateName: "Sonar way",
		},
		"UserRoundTrip": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName: "MyGate",
				Login:    new("alice"),
			},
			wantBuilt:    "user:alice:MyGate",
			wantType:     SubjectTypeUser,
			wantSubject:  "alice",
			wantGateName: "MyGate",
		},
		"GateNameContainingColon": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName:  "org:team:gate",
				GroupName: new("devs"),
			},
			wantBuilt:    "group:devs:org:team:gate",
			wantType:     SubjectTypeGroup,
			wantSubject:  "devs",
			wantGateName: "org:team:gate",
		},
		"UserGateNameContainingColon": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName: "a:b",
				Login:    new("bob"),
			},
			wantBuilt:    "user:bob:a:b",
			wantType:     SubjectTypeUser,
			wantSubject:  "bob",
			wantGateName: "a:b",
		},
		"NoPrincipalReturnsEmpty": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName: "MyGate",
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

			gotBuilt := BuildQualityGateUsergroupAssociationExternalName(tc.params)
			if diff := cmp.Diff(tc.wantBuilt, gotBuilt); diff != "" {
				t.Errorf("BuildQualityGateUsergroupAssociationExternalName() mismatch (-want +got):\n%s", diff)
			}

			if tc.skipParse {
				return
			}

			gotType, gotSubject, gotGate, err := ParseQualityGateUsergroupAssociationExternalName(gotBuilt)
			if err != nil {
				t.Fatalf("ParseQualityGateUsergroupAssociationExternalName() unexpected error: %v", err)
			}

			if gotType != tc.wantType || gotSubject != tc.wantSubject || gotGate != tc.wantGateName {
				t.Errorf("ParseQualityGateUsergroupAssociationExternalName() = (%q, %q, %q), want (%q, %q, %q)",
					gotType, gotSubject, gotGate, tc.wantType, tc.wantSubject, tc.wantGateName)
			}
		})
	}
}

// TestParseQualityGateUsergroupAssociationExternalNameFailures tests parse
// failures for empty names, wrong part counts, and unknown types.
func TestParseQualityGateUsergroupAssociationExternalNameFailures(t *testing.T) {
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
		"UnknownType": {
			externalName: "robot:r2d2:MyGate",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotType, gotSubject, gotGate, err := ParseQualityGateUsergroupAssociationExternalName(tc.externalName)
			if err == nil {
				t.Fatalf("ParseQualityGateUsergroupAssociationExternalName(%q) expected error, got (%q, %q, %q)",
					tc.externalName, gotType, gotSubject, gotGate)
			}

			if gotType != "" || gotSubject != "" || gotGate != "" {
				t.Errorf("ParseQualityGateUsergroupAssociationExternalName() on error = (%q, %q, %q), want empty",
					gotType, gotSubject, gotGate)
			}
		})
	}
}

// TestGenerateQualityGateUsergroupAssociationObservation tests observation
// generation from spec parameters.
func TestGenerateQualityGateUsergroupAssociationObservation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params *v1alpha1.QualityGateUsergroupAssociationParameters
		want   v1alpha1.QualityGateUsergroupAssociationObservation
	}{
		"NilParams": {
			params: nil,
			want:   v1alpha1.QualityGateUsergroupAssociationObservation{},
		},
		"GroupPrincipal": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName:  "Sonar way",
				GroupName: new("sonar-users"),
			},
			want: v1alpha1.QualityGateUsergroupAssociationObservation{
				GateName:  "Sonar way",
				GroupName: "sonar-users",
			},
		},
		"UserPrincipal": {
			params: &v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName: "MyGate",
				Login:    new("alice"),
			},
			want: v1alpha1.QualityGateUsergroupAssociationObservation{
				GateName: "MyGate",
				Login:    "alice",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateQualityGateUsergroupAssociationObservation(tc.params)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateQualityGateUsergroupAssociationObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsQualityGateUsergroupAssociationUpToDate tests association up-to-date
// comparison between spec and observation.
func TestIsQualityGateUsergroupAssociationUpToDate(t *testing.T) {
	t.Parallel()

	groupSpec := &v1alpha1.QualityGateUsergroupAssociationParameters{
		GateName:  "Sonar way",
		GroupName: new("sonar-users"),
	}
	groupObs := &v1alpha1.QualityGateUsergroupAssociationObservation{
		GateName:  "Sonar way",
		GroupName: "sonar-users",
	}
	userSpec := &v1alpha1.QualityGateUsergroupAssociationParameters{
		GateName: "Sonar way",
		Login:    new("alice"),
	}
	userObs := &v1alpha1.QualityGateUsergroupAssociationObservation{
		GateName: "Sonar way",
		Login:    "alice",
	}

	tests := map[string]struct {
		spec        *v1alpha1.QualityGateUsergroupAssociationParameters
		observation *v1alpha1.QualityGateUsergroupAssociationObservation
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
		"GateNameMismatch": {
			spec: groupSpec,
			observation: &v1alpha1.QualityGateUsergroupAssociationObservation{
				GateName:  "other-gate",
				GroupName: "sonar-users",
			},
			want: false,
		},
		"GroupValueMismatch": {
			spec: groupSpec,
			observation: &v1alpha1.QualityGateUsergroupAssociationObservation{
				GateName:  "Sonar way",
				GroupName: "admins",
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

			got := IsQualityGateUsergroupAssociationUpToDate(tc.spec, tc.observation)
			if got != tc.want {
				t.Errorf("IsQualityGateUsergroupAssociationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerateQualityGateAssociationOptions tests generating add, remove,
// and search options for groups and users.
func TestGenerateQualityGateAssociationOptions(t *testing.T) {
	t.Parallel()

	t.Run("AddGroup", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateAddGroupOptions("MyGate", "devs")
		want := &sonar.QualitygatesAddGroupOptions{GateName: "MyGate", GroupName: "devs"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateAddGroupOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("AddUser", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateAddUserOptions("MyGate", "alice")
		want := &sonar.QualitygatesAddUserOptions{GateName: "MyGate", Login: "alice"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateAddUserOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RemoveGroup", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateRemoveGroupOptions("MyGate", "devs")
		want := &sonar.QualitygatesRemoveGroupOptions{GateName: "MyGate", GroupName: "devs"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateRemoveGroupOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RemoveUser", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateRemoveUserOptions("MyGate", "alice")
		want := &sonar.QualitygatesRemoveUserOptions{GateName: "MyGate", Login: "alice"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateRemoveUserOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SearchGroupsNilPagination", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateSearchGroupsOptions("MyGate", "devs", nil)
		want := &sonar.QualitygatesSearchGroupsOptions{
			GateName: "MyGate",
			Query:    "devs",
			Selected: "all",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateSearchGroupsOptions() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SearchUsersWithPagination", func(t *testing.T) {
		t.Parallel()

		got := GenerateQualityGateSearchUsersOptions("MyGate", "alice", &sonar.PaginationArgs{Page: 2, PageSize: 100})
		want := &sonar.QualitygatesSearchUsersOptions{
			PaginationArgs: sonar.PaginationArgs{Page: 2, PageSize: 100},
			GateName:       "MyGate",
			Query:          "alice",
			Selected:       "all",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GenerateQualityGateSearchUsersOptions() mismatch (-want +got):\n%s", diff)
		}
	})
}
