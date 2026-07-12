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
	"context"
	"net/http"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// UserTokensClient handles SonarQube UserToken API operations.
type UserTokensClient interface {
	// Generate creates a new user access token.
	Generate(ctx context.Context, opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error)
	// Revoke deletes a user access token by name.
	Revoke(ctx context.Context, opt *sonar.UserTokensRevokeOptions) (*http.Response, error)
	// Search lists user access tokens.
	Search(ctx context.Context, opt *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error)
}

// NewUserTokensClient creates a UserTokensClient from the given config.
func NewUserTokensClient(clientConfig common.Config) UserTokensClient {
	newClient := common.NewClient(clientConfig)

	return newClient.UserTokens
}

// GenerateUserTokenCreateOptions converts a UserTokenParameters to
// sonar.UserTokensGenerateOptions for the Create operation.
func GenerateUserTokenCreateOptions(spec *v1alpha1.UserTokenParameters) *sonar.UserTokensGenerateOptions {
	if spec == nil {
		return &sonar.UserTokensGenerateOptions{}
	}

	expirationDate := ComputeExpirationDate(spec.RenewalPeriodDays)
	if expirationDate == "" && spec.ExpirationDate != nil {
		expirationDate = *spec.ExpirationDate
	}

	projectKey := ""
	if spec.ProjectKey != nil {
		projectKey = *spec.ProjectKey
	}

	return &sonar.UserTokensGenerateOptions{
		Name:           spec.Name,
		Login:          ptr.Deref(spec.Login, ""),
		Type:           spec.Type,
		ProjectKey:     projectKey,
		ExpirationDate: expirationDate,
	}
}

// GenerateUserTokenRevokeOptions converts a UserTokenParameters to
// sonar.UserTokensRevokeOptions for the Delete operation.
func GenerateUserTokenRevokeOptions(externalName string, spec *v1alpha1.UserTokenParameters) *sonar.UserTokensRevokeOptions {
	if spec == nil {
		return &sonar.UserTokensRevokeOptions{
			Name: externalName,
		}
	}

	return &sonar.UserTokensRevokeOptions{
		Name:  externalName,
		Login: ptr.Deref(spec.Login, ""),
	}
}

// GenerateUserTokenSearchOptions converts a UserTokenParameters to
// sonar.UserTokensSearchOptions for the Observe operation.
func GenerateUserTokenSearchOptions(spec *v1alpha1.UserTokenParameters) *sonar.UserTokensSearchOptions {
	if spec == nil {
		return &sonar.UserTokensSearchOptions{}
	}

	return &sonar.UserTokensSearchOptions{
		Login: ptr.Deref(spec.Login, ""),
	}
}

// GenerateUserTokenObservation converts a sonar.UserToken API response
// to a v1alpha1.UserTokenObservation.
func GenerateUserTokenObservation(token *sonar.UserToken) v1alpha1.UserTokenObservation {
	if token == nil {
		return v1alpha1.UserTokenObservation{}
	}

	return v1alpha1.UserTokenObservation{
		CreatedAt:      helpers.StringToMetaTime(&token.CreatedAt),
		ExpirationDate: helpers.StringToMetaTime(&token.ExpirationDate),
		IsExpired:      token.IsExpired,
		Name:           token.Name,
		Type:           token.Type,
	}
}

// FindUserToken searches for a token by name in the search result.
// Returns nil if the search result is nil or the token is not found.
func FindUserToken(search *sonar.UserTokensSearch, name string) *sonar.UserToken {
	if search == nil {
		return nil
	}

	for i := range search.UserTokens {
		if search.UserTokens[i].Name == name {
			return &search.UserTokens[i]
		}
	}

	return nil
}

// IsUserTokenUpToDate returns true when no update should be performed.
// Tokens with RenewalPeriodDays trigger renewal when expired or when
// proactive rotation is due (RenewAt has passed).
// All other tokens are immutable and always considered up to date.
func IsUserTokenUpToDate(spec *v1alpha1.UserTokenParameters, obs *v1alpha1.UserTokenObservation) bool {
	if spec == nil {
		return true
	}

	if obs == nil {
		return false
	}

	if spec.RenewalPeriodDays == nil {
		return true
	}

	if obs.IsExpired {
		return false
	}

	if obs.RenewAt != nil && time.Now().After(obs.RenewAt.Time) {
		return false
	}

	return true
}

// ComputeRenewAt computes when to proactively renew the token.
// Returns nil when renewBeforeDays or expirationDate is nil.
func ComputeRenewAt(expirationDate *metav1.Time, renewBeforeDays *int64) *metav1.Time {
	if expirationDate == nil || renewBeforeDays == nil {
		return nil
	}

	renewAt := metav1.NewTime(expirationDate.AddDate(0, 0, -int(*renewBeforeDays)))

	return &renewAt
}

// ComputeExpirationDate returns a date N days from now in YYYY-MM-DD
// format. Returns "" when renewalPeriodDays is nil.
func ComputeExpirationDate(renewalPeriodDays *int64) string {
	if renewalPeriodDays == nil {
		return ""
	}

	return time.Now().AddDate(0, 0, int(*renewalPeriodDays)).Format(time.DateOnly)
}

// LateInitializeUserToken fills empty spec fields from the observation.
// UserTokens have no late-initializable fields, so this is a no-op.
func LateInitializeUserToken(spec *v1alpha1.UserTokenParameters, observation *v1alpha1.UserTokenObservation) {
	if spec == nil || observation == nil {
		return
	}
}

// IsUserTokenLateInitialized returns true when the spec changed
// (i.e., late initialization occurred).
func IsUserTokenLateInitialized(former, current *v1alpha1.UserTokenParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !cmp.Equal(former, current)
}
