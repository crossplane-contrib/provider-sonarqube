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
	"time"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// pastTime returns a metav1.Time set to one hour ago.
func pastTime() *metav1.Time {
	mt := metav1.NewTime(time.Now().Add(-time.Hour))

	return &mt
}

// futureTime returns a metav1.Time set to 24 hours from now.
func futureTime() *metav1.Time {
	mt := metav1.NewTime(time.Now().Add(24 * time.Hour))

	return &mt
}

// TestGenerateUserTokenObservation tests generating an observation.
func TestGenerateUserTokenObservation(t *testing.T) {
	t.Parallel()

	createdAtStr := "2026-01-15T10:30:00+0000"
	expiredAtStr := "2026-04-15"

	parsedCreatedAt := metav1.NewTime(func() time.Time {
		tVal, _ := time.Parse("2006-01-02T15:04:05-0700", createdAtStr)

		return tVal.UTC()
	}())
	parsedExpiredAt := metav1.NewTime(func() time.Time {
		tVal, _ := time.Parse(time.DateOnly, expiredAtStr)

		return tVal.UTC()
	}())

	cases := map[string]struct {
		token *sonar.UserToken
		want  v1alpha1.UserTokenObservation
	}{
		"NilReturnsEmpty": {
			token: nil,
			want:  v1alpha1.UserTokenObservation{},
		},
		"AllFieldsMapped": {
			token: &sonar.UserToken{
				Name:           "my-token",
				Type:           sonar.TokenTypeUserToken,
				CreatedAt:      createdAtStr,
				ExpirationDate: expiredAtStr,
				IsExpired:      true,
			},
			want: v1alpha1.UserTokenObservation{
				Name:           "my-token",
				Type:           sonar.TokenTypeUserToken,
				CreatedAt:      &parsedCreatedAt,
				ExpirationDate: &parsedExpiredAt,
				IsExpired:      true,
			},
		},
		"NoExpirationDateProducesNilTime": {
			token: &sonar.UserToken{
				Name:      "no-expiry",
				Type:      sonar.TokenTypeGlobalAnalysisToken,
				CreatedAt: createdAtStr,
			},
			want: v1alpha1.UserTokenObservation{
				Name:      "no-expiry",
				Type:      sonar.TokenTypeGlobalAnalysisToken,
				CreatedAt: &parsedCreatedAt,
			},
		},
		"EmptyCreatedAtProducesNilTime": {
			token: &sonar.UserToken{
				Name: "empty-dates",
				Type: sonar.TokenTypeUserToken,
			},
			want: v1alpha1.UserTokenObservation{
				Name: "empty-dates",
				Type: sonar.TokenTypeUserToken,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUserTokenObservation(tc.token)
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("GenerateUserTokenObservation() -want +got:\n%s", diff)
			}
		})
	}
}

// TestFindUserToken tests finding a token by name in search results.
func TestFindUserToken(t *testing.T) {
	t.Parallel()

	twoTokens := []sonar.UserToken{
		{Name: "token-a", Type: sonar.TokenTypeUserToken},
		{Name: "token-b", Type: sonar.TokenTypeGlobalAnalysisToken},
	}

	cases := map[string]struct {
		search  *sonar.UserTokensSearch
		name    string
		wantNil bool
		wantKey string
	}{
		"NilSearchReturnsNil": {
			search:  nil,
			name:    "token-a",
			wantNil: true,
		},
		"EmptyTokensReturnsNil": {
			search:  &sonar.UserTokensSearch{},
			name:    "token-a",
			wantNil: true,
		},
		"NameAbsentReturnsNil": {
			search:  &sonar.UserTokensSearch{UserTokens: twoTokens},
			name:    "token-c",
			wantNil: true,
		},
		"FirstTokenFound": {
			search:  &sonar.UserTokensSearch{UserTokens: twoTokens},
			name:    "token-a",
			wantKey: "token-a",
		},
		"SecondTokenFound": {
			search:  &sonar.UserTokensSearch{UserTokens: twoTokens},
			name:    "token-b",
			wantKey: "token-b",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindUserToken(tc.search, tc.name)
			if tc.wantNil {
				if got != nil {
					t.Errorf("FindUserToken() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("FindUserToken() = nil, want non-nil")
			}

			if got.Name != tc.wantKey {
				t.Errorf("FindUserToken() Name = %q, want %q", got.Name, tc.wantKey)
			}
		})
	}
}

// TestIsUserTokenUpToDate tests checking whether the token spec is up to date.
func TestIsUserTokenUpToDate(t *testing.T) {
	t.Parallel()

	days30 := int64(30)

	cases := map[string]struct {
		spec *v1alpha1.UserTokenParameters
		obs  *v1alpha1.UserTokenObservation
		want bool
	}{
		"NilSpecReturnsTrue": {
			spec: nil,
			obs:  &v1alpha1.UserTokenObservation{IsExpired: true},
			want: true,
		},
		"NilObsReturnsFalse": {
			spec: &v1alpha1.UserTokenParameters{Name: "t"},
			obs:  nil,
			want: false,
		},
		"NoRenewalPeriodExpiredIsUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t"},
			obs:  &v1alpha1.UserTokenObservation{IsExpired: true},
			want: true,
		},
		"RenewalPeriodNotExpiredIsUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t", RenewalPeriodDays: &days30},
			obs:  &v1alpha1.UserTokenObservation{IsExpired: false},
			want: true,
		},
		"RenewalPeriodExpiredIsNotUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t", RenewalPeriodDays: &days30},
			obs:  &v1alpha1.UserTokenObservation{IsExpired: true},
			want: false,
		},
		"RenewAtInPastIsNotUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t", RenewalPeriodDays: &days30},
			obs: &v1alpha1.UserTokenObservation{
				IsExpired: false,
				RenewAt:   pastTime(),
			},
			want: false,
		},
		"RenewAtInFutureIsUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t", RenewalPeriodDays: &days30},
			obs: &v1alpha1.UserTokenObservation{
				IsExpired: false,
				RenewAt:   futureTime(),
			},
			want: true,
		},
		"NilRenewAtWithRenewalPeriodIsUpToDate": {
			spec: &v1alpha1.UserTokenParameters{Name: "t", RenewalPeriodDays: &days30},
			obs:  &v1alpha1.UserTokenObservation{IsExpired: false},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsUserTokenUpToDate(tc.spec, tc.obs)
			if got != tc.want {
				t.Errorf("IsUserTokenUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestComputeExpirationDate tests computing expiration dates.
func TestComputeExpirationDate(t *testing.T) {
	t.Parallel()

	days30 := int64(30)

	cases := map[string]struct {
		days    *int64
		wantLen int
	}{
		"NilReturnsEmpty": {
			days:    nil,
			wantLen: 0,
		},
		"PositiveDaysReturnsDateString": {
			days:    &days30,
			wantLen: 10, // "YYYY-MM-DD"
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ComputeExpirationDate(tc.days)
			if len(got) != tc.wantLen {
				t.Errorf("ComputeExpirationDate() len = %d, want %d (got %q)", len(got), tc.wantLen, got)
			}

			if tc.days != nil {
				// Verify it's a valid YYYY-MM-DD date
				parsed, err := time.Parse(time.DateOnly, got)
				if err != nil {
					t.Errorf("ComputeExpirationDate() produced invalid date %q: %v", got, err)
				}

				// Verify it's approximately now + N days (within 1 day for test timing)
				expected := time.Now().AddDate(0, 0, int(*tc.days))

				diff := expected.Sub(parsed)
				if diff < -24*time.Hour || diff > 24*time.Hour {
					t.Errorf("ComputeExpirationDate() = %q, want ~%s", got, expected.Format(time.DateOnly))
				}
			}
		})
	}
}

// TestComputeRenewAt tests proactive renewal time computation.
func TestComputeRenewAt(t *testing.T) {
	t.Parallel()

	days7 := int64(7)
	expiration := metav1.NewTime(time.Now().AddDate(0, 0, 30))

	cases := map[string]struct {
		expirationDate  *metav1.Time
		renewBeforeDays *int64
		wantNil         bool
	}{
		"NilExpirationDateReturnsNil": {
			expirationDate:  nil,
			renewBeforeDays: &days7,
			wantNil:         true,
		},
		"NilRenewBeforeDaysReturnsNil": {
			expirationDate:  &expiration,
			renewBeforeDays: nil,
			wantNil:         true,
		},
		"ComputesRenewAtBeforeExpiration": {
			expirationDate:  &expiration,
			renewBeforeDays: &days7,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ComputeRenewAt(tc.expirationDate, tc.renewBeforeDays)
			if tc.wantNil {
				if got != nil {
					t.Errorf("ComputeRenewAt() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("ComputeRenewAt() = nil, want non-nil")
			}

			expected := tc.expirationDate.AddDate(0, 0, -int(*tc.renewBeforeDays))
			if !got.Time.Equal(expected) {
				t.Errorf("ComputeRenewAt() = %v, want %v", got.Time, expected)
			}
		})
	}
}

// TestLateInitializeUserToken tests the no-op late initialization.
func TestLateInitializeUserToken(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.UserTokenParameters
		observation *v1alpha1.UserTokenObservation
		wantSpec    *v1alpha1.UserTokenParameters
	}{
		"NilSpecIsNoOp": {
			spec:        nil,
			observation: &v1alpha1.UserTokenObservation{Type: sonar.TokenTypeUserToken},
			wantSpec:    nil,
		},
		"NilObservationIsNoOp": {
			spec:        &v1alpha1.UserTokenParameters{Name: "t"},
			observation: nil,
			wantSpec:    &v1alpha1.UserTokenParameters{Name: "t"},
		},
		"NoFieldsModified": {
			spec:        &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeUserToken},
			observation: &v1alpha1.UserTokenObservation{Type: sonar.TokenTypeUserToken},
			wantSpec:    &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeUserToken},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			LateInitializeUserToken(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantSpec, tc.spec); diff != "" {
				t.Errorf("LateInitializeUserToken() spec -want +got:\n%s", diff)
			}
		})
	}
}

// TestIsUserTokenLateInitialized tests checking for late-initialized fields.
func TestIsUserTokenLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.UserTokenParameters
		current *v1alpha1.UserTokenParameters
		want    bool
	}{
		"BothNilReturnsFalse": {
			former:  nil,
			current: nil,
			want:    false,
		},
		"FormerNilReturnsFalse": {
			former:  nil,
			current: &v1alpha1.UserTokenParameters{Name: "t"},
			want:    false,
		},
		"CurrentNilReturnsFalse": {
			former:  &v1alpha1.UserTokenParameters{Name: "t"},
			current: nil,
			want:    false,
		},
		"EqualSpecsReturnFalse": {
			former:  &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeUserToken},
			current: &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeUserToken},
			want:    false,
		},
		"DifferentSpecsReturnTrue": {
			former:  &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeUserToken},
			current: &v1alpha1.UserTokenParameters{Name: "t", Type: sonar.TokenTypeGlobalAnalysisToken},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsUserTokenLateInitialized(tc.former, tc.current)
			if got != tc.want {
				t.Errorf("IsUserTokenLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNewUserTokensClient tests creating a new UserTokens client.
func TestNewUserTokensClient(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsValidClient", func(t *testing.T) {
		t.Parallel()

		config := common.Config{
			AuthType: common.PersonalAccessToken,
			BaseURL:  "http://localhost:9000",
			Token:    "test-token",
		}

		client := NewUserTokensClient(config)
		if client == nil {
			t.Fatal("NewUserTokensClient() returned nil")
		}
	})
}

// TestGenerateUserTokenCreateOptions tests building create options from spec.
func TestGenerateUserTokenCreateOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.UserTokenParameters
		want *sonar.UserTokensGenerateOptions
	}{
		"NilSpec": {
			spec: nil,
			want: &sonar.UserTokensGenerateOptions{},
		},
		"MinimalSpec": {
			spec: &v1alpha1.UserTokenParameters{
				Name: "test-token",
				Type: sonar.TokenTypeUserToken,
			},
			want: &sonar.UserTokensGenerateOptions{
				Name: "test-token",
				Type: sonar.TokenTypeUserToken,
			},
		},
		"SpecWithLogin": {
			spec: &v1alpha1.UserTokenParameters{
				Name:  "test-token",
				Type:  sonar.TokenTypeUserToken,
				Login: new("user1"),
			},
			want: &sonar.UserTokensGenerateOptions{
				Name:  "test-token",
				Type:  sonar.TokenTypeUserToken,
				Login: "user1",
			},
		},
		"SpecWithExpirationDate": {
			spec: &v1alpha1.UserTokenParameters{
				Name:           "test-token",
				Type:           sonar.TokenTypeUserToken,
				ExpirationDate: new("2026-12-31"),
			},
			want: &sonar.UserTokensGenerateOptions{
				Name:           "test-token",
				Type:           sonar.TokenTypeUserToken,
				ExpirationDate: "2026-12-31",
			},
		},
		"SpecWithProjectKey": {
			spec: &v1alpha1.UserTokenParameters{
				Name:       "test-token",
				Type:       sonar.TokenTypeProjectAnalysisToken,
				ProjectKey: new("my-project"),
			},
			want: &sonar.UserTokensGenerateOptions{
				Name:       "test-token",
				Type:       sonar.TokenTypeProjectAnalysisToken,
				ProjectKey: "my-project",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUserTokenCreateOptions(tc.spec)

			if tc.want.Name != got.Name {
				t.Errorf("GenerateUserTokenCreateOptions() Name = %q, want %q", got.Name, tc.want.Name)
			}

			if tc.want.Type != got.Type {
				t.Errorf("GenerateUserTokenCreateOptions() Type = %q, want %q", got.Type, tc.want.Type)
			}

			if tc.want.Login != got.Login {
				t.Errorf("GenerateUserTokenCreateOptions() Login = %q, want %q", got.Login, tc.want.Login)
			}

			if tc.want.ProjectKey != got.ProjectKey {
				t.Errorf("GenerateUserTokenCreateOptions() ProjectKey = %q, want %q", got.ProjectKey, tc.want.ProjectKey)
			}

			if tc.spec != nil && tc.spec.ExpirationDate != nil {
				if *tc.spec.ExpirationDate != got.ExpirationDate {
					t.Errorf("GenerateUserTokenCreateOptions() ExpirationDate = %q, want %q", got.ExpirationDate, *tc.spec.ExpirationDate)
				}
			}
		})
	}
}

// TestGenerateUserTokenRevokeOptions tests building revoke options.
func TestGenerateUserTokenRevokeOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		externalName string
		spec         *v1alpha1.UserTokenParameters
		want         *sonar.UserTokensRevokeOptions
	}{
		"NilSpec": {
			externalName: "token-name",
			spec:         nil,
			want: &sonar.UserTokensRevokeOptions{
				Name: "token-name",
			},
		},
		"SpecWithLogin": {
			externalName: "token-name",
			spec: &v1alpha1.UserTokenParameters{
				Login: new("user1"),
			},
			want: &sonar.UserTokensRevokeOptions{
				Name:  "token-name",
				Login: "user1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUserTokenRevokeOptions(tc.externalName, tc.spec)

			if tc.want.Name != got.Name {
				t.Errorf("GenerateUserTokenRevokeOptions() Name = %q, want %q", got.Name, tc.want.Name)
			}

			if tc.want.Login != got.Login {
				t.Errorf("GenerateUserTokenRevokeOptions() Login = %q, want %q", got.Login, tc.want.Login)
			}
		})
	}
}

// TestGenerateUserTokenSearchOptions tests building search options.
func TestGenerateUserTokenSearchOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec *v1alpha1.UserTokenParameters
		want *sonar.UserTokensSearchOptions
	}{
		"NilSpec": {
			spec: nil,
			want: &sonar.UserTokensSearchOptions{},
		},
		"SpecWithLogin": {
			spec: &v1alpha1.UserTokenParameters{
				Login: new("user1"),
			},
			want: &sonar.UserTokensSearchOptions{
				Login: "user1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUserTokenSearchOptions(tc.spec)

			if tc.want.Login != got.Login {
				t.Errorf("GenerateUserTokenSearchOptions() Login = %q, want %q", got.Login, tc.want.Login)
			}
		})
	}
}
