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

package iam

import (
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

const (
	// testEmail is a test constant for email addresses.
	testEmail = "alice@example.com"
	// testSecret is a test constant for secret values.
	testSecret = "secret"
	// testUserName is a test constant for user names.
	testUserName = "Alice"
	// testUserLogin is a test constant for user logins.
	testUserLogin = "alice"
)

// TestNewUsersClient tests creating a new users client.
func TestNewUsersClient(t *testing.T) {
	t.Parallel()

	client := NewUsersClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})

	if client == nil {
		t.Fatal("NewUsersClient() expected non-nil client")
	}
}

// TestLateInitializeUser tests late initialization of user parameters.
func TestLateInitializeUser(t *testing.T) {
	t.Parallel()

	t.Run("nil inputs are ignored", func(t *testing.T) {
		t.Parallel()

		LateInitializeUser(nil, nil)
		LateInitializeUser(&v1alpha1.UserParameters{}, nil)
		LateInitializeUser(nil, &v1alpha1.UserObservation{})
	})

	t.Run("fills missing optional fields", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.UserParameters{Login: "alice", Name: "Alice"}
		obs := &v1alpha1.UserObservation{
			Email:       testEmail,
			Local:       true,
			ScmAccounts: []string{"github:alice"},
		}

		LateInitializeUser(spec, obs)

		if spec.Email == nil || *spec.Email != testEmail {
			t.Fatalf("LateInitializeUser() email = %v, want %s", spec.Email, testEmail)
		}

		if spec.Local == nil || !*spec.Local {
			t.Fatalf("LateInitializeUser() local = %v, want true", spec.Local)
		}

		if spec.ScmAccounts == nil || cmp.Diff([]string{"github:alice"}, *spec.ScmAccounts) != "" {
			t.Fatalf("LateInitializeUser() scm accounts = %v", spec.ScmAccounts)
		}
	})

	t.Run("does not overwrite existing fields", func(t *testing.T) {
		t.Parallel()

		email := "custom@example.com"
		local := false
		accounts := []string{"gitlab:alice"}
		spec := &v1alpha1.UserParameters{
			Login:       "alice",
			Name:        "Alice",
			Email:       &email,
			Local:       &local,
			ScmAccounts: &accounts,
		}
		obs := &v1alpha1.UserObservation{Email: testEmail, Local: true, ScmAccounts: []string{"github:alice"}}

		LateInitializeUser(spec, obs)

		if spec.Email == nil || *spec.Email != email {
			t.Fatalf("LateInitializeUser() overwrote email = %v", spec.Email)
		}

		if spec.Local == nil || *spec.Local != local {
			t.Fatalf("LateInitializeUser() overwrote local = %v", spec.Local)
		}

		if spec.ScmAccounts == nil || cmp.Diff(accounts, *spec.ScmAccounts) != "" {
			t.Fatalf("LateInitializeUser() overwrote scm accounts = %v", spec.ScmAccounts)
		}
	})
}

// TestIsUserLateInitialized tests checking user late initialization.
func TestIsUserLateInitialized(t *testing.T) {
	t.Parallel()

	email := testEmail
	local := true
	accounts := []string{"github:alice", "gitlab:alice"}

	cases := map[string]struct {
		former  *v1alpha1.UserParameters
		current *v1alpha1.UserParameters
		want    bool
	}{
		"nil former": {
			current: &v1alpha1.UserParameters{Login: "alice", Name: "Alice"},
			want:    false,
		},
		"same values": {
			former:  &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, Local: &local, ScmAccounts: &accounts},
			current: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: new(email), Local: new(local), ScmAccounts: &[]string{"gitlab:alice", "github:alice"}},
			want:    false,
		},
		"different external fields are ignored": {
			former:  &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, Local: &local, ScmAccounts: &accounts, ExternalId: new("former-id"), ExternalLogin: new("former-login"), ExternalProvider: new("github")},
			current: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: new(email), Local: new(local), ScmAccounts: &[]string{"github:alice", "gitlab:alice"}, ExternalId: new("current-id"), ExternalLogin: new("current-login"), ExternalProvider: new("gitlab")},
			want:    false,
		},
		"different login": {
			former:  &v1alpha1.UserParameters{Login: "alice", Name: "Alice"},
			current: &v1alpha1.UserParameters{Login: "bob", Name: "Alice"},
			want:    true,
		},
		"different scm accounts": {
			former:  &v1alpha1.UserParameters{Login: "alice", Name: "Alice", ScmAccounts: &[]string{"github:alice"}},
			current: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", ScmAccounts: &[]string{"gitlab:alice"}},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsUserLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsUserLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsUserUpToDate tests checking if user is up to date.
func TestIsUserUpToDate(t *testing.T) {
	t.Parallel()

	email := testEmail
	local := true
	specGroups := []v1alpha1.UserGroupsParameters{{GroupId: new("devs")}, {GroupId: new("ops")}}

	cases := map[string]struct {
		spec *v1alpha1.UserParameters
		obs  *v1alpha1.UserObservation
		want bool
	}{
		"nil spec": {
			obs:  &v1alpha1.UserObservation{Login: "alice", Name: "Alice"},
			want: true,
		},
		"nil observation": {
			spec: &v1alpha1.UserParameters{Login: "alice", Name: "Alice"},
			want: false,
		},
		"matching values": {
			spec: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, Local: &local, ScmAccounts: &[]string{"gitlab:alice", "github:alice"}, Groups: &specGroups},
			obs:  &v1alpha1.UserObservation{Login: "alice", Name: "Alice", Email: testEmail, Local: true, ScmAccounts: []string{"github:alice", "gitlab:alice"}, Groups: map[string]string{"ops": "membership-2", "devs": "membership-1"}},
			want: true,
		},
		"different external fields are ignored": {
			spec: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, Local: &local, ExternalId: new("wanted-id"), ExternalLogin: new("wanted-login"), ExternalProvider: new("wanted-provider"), ScmAccounts: &[]string{"gitlab:alice", "github:alice"}, Groups: &specGroups},
			obs:  &v1alpha1.UserObservation{Login: "alice", Name: "Alice", Email: testEmail, Local: true, ExternalId: "actual-id", ExternalLogin: "actual-login", ExternalProvider: "actual-provider", ScmAccounts: []string{"github:alice", "gitlab:alice"}, Groups: map[string]string{"ops": "membership-2", "devs": "membership-1"}},
			want: true,
		},
		"group mismatch": {
			spec: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Groups: &[]v1alpha1.UserGroupsParameters{{GroupId: new("devs")}}},
			obs:  &v1alpha1.UserObservation{Login: "alice", Name: "Alice", Groups: map[string]string{"ops": "membership-2"}},
			want: false,
		},
		"nil group id entries are ignored": {
			spec: &v1alpha1.UserParameters{Login: "alice", Name: "Alice", Groups: &[]v1alpha1.UserGroupsParameters{{GroupId: nil}, {GroupId: new("")}}},
			obs:  &v1alpha1.UserObservation{Login: "alice", Name: "Alice", Groups: map[string]string{}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsUserUpToDate(tc.spec, tc.obs); got != tc.want {
				t.Fatalf("IsUserUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAreUserScmAccountsUpToDate tests checking if user SCM
// accounts are up to date.
func TestAreUserScmAccountsUpToDate(t *testing.T) {
	t.Parallel()

	if !AreUserScmAccountsUpToDate(nil, nil) {
		t.Fatal("AreUserScmAccountsUpToDate(nil,nil) = false, want true")
	}

	if AreUserScmAccountsUpToDate(&[]string{"a"}, nil) {
		t.Fatal("AreUserScmAccountsUpToDate(spec,nil) = true, want false")
	}

	if !AreUserScmAccountsUpToDate(&[]string{"a", "a", "b"}, &[]string{"b", "a"}) {
		t.Fatal("AreUserScmAccountsUpToDate() deduped compare failed")
	}
}

// TestAreUserGroupsUpToDate tests checking if user groups are up to date.
func TestAreUserGroupsUpToDate(t *testing.T) {
	t.Parallel()

	if !AreUserGroupsUpToDate(nil, nil) {
		t.Fatal("AreUserGroupsUpToDate(nil,nil) = false, want true")
	}

	if AreUserGroupsUpToDate(&[]v1alpha1.UserGroupsParameters{{GroupId: new("g1")}}, nil) {
		t.Fatal("AreUserGroupsUpToDate(spec,nil) = true, want false")
	}

	groups := []v1alpha1.UserGroupsParameters{{GroupId: new("g1")}, {GroupId: nil}, {GroupId: new("")}, {GroupId: new("g2")}}
	obs := map[string]string{"g2": "m2", "g1": "m1"}

	if !AreUserGroupsUpToDate(&groups, &obs) {
		t.Fatal("AreUserGroupsUpToDate() = false, want true")
	}
}

// TestGenerateCreateUserOptions tests creating user creation options.
func TestGenerateCreateUserOptions(t *testing.T) {
	t.Parallel()

	if GenerateCreateUserOptions(nil, nil) != nil {
		t.Fatal("GenerateCreateUserOptions(nil,nil) must return nil")
	}

	email := testEmail
	local := true
	accounts := []string{"github:alice", "gitlab:alice"}
	password := testSecret

	options := GenerateCreateUserOptions(&v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, Local: &local, ScmAccounts: &accounts}, &password)

	if options == nil {
		t.Fatal("GenerateCreateUserOptions() returned nil")
	}

	if options.Login != testUserLogin || options.Name != testUserName || options.Password != testSecret {
		t.Fatalf("GenerateCreateUserOptions() got %+v", options)
	}

	if options.Email != email {
		t.Fatalf("GenerateCreateUserOptions() email = %q, want %q", options.Email, email)
	}

	if options.Local == nil || !*options.Local {
		t.Fatalf("GenerateCreateUserOptions() local = %v, want true", options.Local)
	}

	if cmp.Diff(accounts, options.ScmAccounts) != "" {
		t.Fatalf("GenerateCreateUserOptions() scm accounts = %v", options.ScmAccounts)
	}
}

// TestGenerateUpdateUserOptions tests creating user update options.
func TestGenerateUpdateUserOptions(t *testing.T) {
	t.Parallel()

	if GenerateUpdateUserOptions(nil) != nil {
		t.Fatal("GenerateUpdateUserOptions(nil) must return nil")
	}

	email := testEmail
	accounts := []string{"github:alice", "gitlab:alice"}
	externalID := "ext-id"
	externalLogin := "ext-login"
	externalProvider := "github"

	options := GenerateUpdateUserOptions(&v1alpha1.UserParameters{Login: "alice", Name: "Alice", Email: &email, ScmAccounts: &accounts, ExternalId: &externalID, ExternalLogin: &externalLogin, ExternalProvider: &externalProvider})

	if options == nil {
		t.Fatal("GenerateUpdateUserOptions() returned nil")
	}

	if options.Login != testUserLogin || options.Name != "Alice" {
		t.Fatalf("GenerateUpdateUserOptions() got %+v", options)
	}

	if options.Email != email {
		t.Fatalf("GenerateUpdateUserOptions() email = %q, want %q", options.Email, email)
	}

	if options.ExternalId != externalID || options.ExternalLogin != externalLogin || options.ExternalProvider != externalProvider {
		t.Fatalf("GenerateUpdateUserOptions() external fields mismatch: %+v", options)
	}

	if options.ScmAccounts == nil || !options.ScmAccounts.Defined || cmp.Diff(accounts, options.ScmAccounts.Value) != "" {
		t.Fatalf("GenerateUpdateUserOptions() scm accounts = %+v", options.ScmAccounts)
	}
}

// TestGenerateUserObservation tests generating user observations.
func TestGenerateUserObservation(t *testing.T) {
	t.Parallel()

	if diff := cmp.Diff(v1alpha1.UserObservation{}, GenerateUserObservation(nil)); diff != "" {
		t.Fatalf("GenerateUserObservation(nil) mismatch (-want +got):\n%s", diff)
	}

	user := &sonar.UserV2{
		Id:                          "user-1",
		Login:                       "alice",
		Name:                        "Alice",
		Email:                       testEmail,
		ExternalId:                  "ext-1",
		ExternalLogin:               "alice-ext",
		ExternalProvider:            "github",
		Local:                       true,
		Managed:                     false,
		Active:                      true,
		Avatar:                      "https://example.org/avatar.png",
		ScmAccounts:                 []string{"github:alice"},
		SonarLintLastConnectionDate: "2024-01-01T00:00:00+0000",
		SonarQubeLastConnectionDate: "2024-01-02T00:00:00+0000",
	}

	got := GenerateUserObservation(user)

	if got.Id != user.Id || got.Login != user.Login || got.Name != user.Name || got.Email != user.Email {
		t.Fatalf("GenerateUserObservation() basic fields mismatch: %+v", got)
	}

	if got.ExternalId != user.ExternalId || got.ExternalLogin != user.ExternalLogin || got.ExternalProvider != user.ExternalProvider {
		t.Fatalf("GenerateUserObservation() external fields mismatch: %+v", got)
	}

	if got.Active != user.Active || got.Local != user.Local || got.Managed != user.Managed {
		t.Fatalf("GenerateUserObservation() boolean fields mismatch: %+v", got)
	}

	if cmp.Diff(user.ScmAccounts, got.ScmAccounts) != "" {
		t.Fatalf("GenerateUserObservation() scm accounts = %v", got.ScmAccounts)
	}

	if got.Avatar != user.Avatar {
		t.Fatalf("GenerateUserObservation() avatar = %q, want %q", got.Avatar, user.Avatar)
	}
}
