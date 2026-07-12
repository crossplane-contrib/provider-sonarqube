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

package usertoken

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// testTokenName is the token name used across controller tests.
const testTokenName = "my-ci-token"

// notUserToken is a sentinel for testing type assertion failures.
type notUserToken struct {
	resource.Managed
}

// mockHTTPOK returns a 200 OK HTTP response for testing.
func mockHTTPOK() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}
}

// mockHTTPNoContent returns a 204 No Content HTTP response for testing.
func mockHTTPNoContent() *http.Response {
	return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content"}
}

// errBoom is a generic test error.
var errBoom = errors.New("boom")

// errComparer compares errors by message string.
var errComparer = cmp.Comparer(func(x, y error) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}

	return x.Error() == y.Error()
})

// newToken builds a UserToken CR with the given external name.
func newToken(name, externalName string) *v1alpha1.UserToken {
	t := &v1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.UserTokenSpec{
			ForProvider: v1alpha1.UserTokenParameters{
				Name: name,
				Type: sonar.TokenTypeUserToken,
			},
		},
	}
	if externalName != "" {
		meta.SetExternalName(t, externalName)
	}

	return t
}

// withRenewalDays returns t with RenewalPeriodDays set.
func withRenewalDays(t *v1alpha1.UserToken, days int64) *v1alpha1.UserToken {
	t.Spec.ForProvider.RenewalPeriodDays = &days

	return t
}

// TestObserve covers early returns and the main observation flow.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		client *fake.MockUserTokensClient
		args   args
		want   want
	}{
		"NotUserTokenError": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: &notUserToken{}},
			want:   want{o: managed.ExternalObservation{}, err: errors.New(errNotUserToken)},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: newToken(testTokenName, "")},
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"SearchAPIError": {
			client: &fake.MockUserTokensClient{
				SearchFn: func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errObserveToken),
			},
		},
		"TokenNotFoundReturnsNotExists": {
			client: &fake.MockUserTokensClient{
				SearchFn: func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
					return &sonar.UserTokensSearch{
						UserTokens: []sonar.UserToken{{Name: "other-token"}},
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"TokenFoundUpToDate": {
			client: &fake.MockUserTokensClient{
				SearchFn: func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
					return &sonar.UserTokensSearch{
						Login: "admin",
						UserTokens: []sonar.UserToken{
							{Name: testTokenName, Type: sonar.TokenTypeUserToken},
						},
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Observe() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() -want +got:\n%s", diff)
			}
		})
	}
}

// TestObserveRenewalTriggered covers the renewal path in Observe.
func TestObserveRenewalTriggered(t *testing.T) {
	t.Parallel()

	expiredSearch := func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
		return &sonar.UserTokensSearch{
			UserTokens: []sonar.UserToken{
				{Name: testTokenName, Type: sonar.TokenTypeUserToken, IsExpired: true},
			},
		}, mockHTTPOK(), nil
	}

	cases := map[string]struct {
		mg   *v1alpha1.UserToken
		want bool
	}{
		"ExpiredWithRenewalPeriodIsNotUpToDate": {
			mg:   withRenewalDays(newToken(testTokenName, testTokenName), 30),
			want: false,
		},
		"ExpiredWithoutRenewalPeriodIsUpToDate": {
			mg:   newToken(testTokenName, testTokenName),
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: &fake.MockUserTokensClient{SearchFn: expiredSearch}}

			obs, err := e.Observe(context.Background(), tc.mg)
			if err != nil {
				t.Fatalf("Observe() unexpected error: %v", err)
			}

			if obs.ResourceUpToDate != tc.want {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tc.want)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o            managed.ExternalCreation
		err          error
		externalName string
	}

	cases := map[string]struct {
		client *fake.MockUserTokensClient
		args   args
		want   want
	}{
		"NotUserTokenError": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: &notUserToken{}},
			want:   want{o: managed.ExternalCreation{}, err: errors.New(errNotUserToken)},
		},
		"GenerateAPIError": {
			client: &fake.MockUserTokensClient{
				GenerateFn: func(_ *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, "")},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errCreateToken),
			},
		},
		"SuccessSetsExternalNameAndTokenSecret": {
			client: &fake.MockUserTokensClient{
				GenerateFn: func(_ *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
					return &sonar.UserTokensGenerate{
						Name:  testTokenName,
						Token: "secret-token-value",
					}, mockHTTPOK(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, "")},
			want: want{
				o: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						connectionSecretTokenKey: []byte("secret-token-value"),
					},
				},
				externalName: testTokenName,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Create() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Create() -want +got:\n%s", diff)
			}

			if tc.want.externalName != "" {
				if tok, ok := tc.args.mg.(*v1alpha1.UserToken); ok {
					if got := meta.GetExternalName(tok); got != tc.want.externalName {
						t.Errorf("Create() external name = %q, want %q", got, tc.want.externalName)
					}
				}
			}
		})
	}
}

// TestUpdate tests the Update (renewal) method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalUpdate
		err error
	}

	cases := map[string]struct {
		client *fake.MockUserTokensClient
		args   args
		want   want
	}{
		"NotUserTokenError": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: &notUserToken{}},
			want:   want{o: managed.ExternalUpdate{}, err: errors.New(errNotUserToken)},
		},
		"EmptyExternalNameReturnsError": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: newToken(testTokenName, "")},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New("cannot renew UserToken without external name"),
			},
		},
		"RevokeAPIError": {
			client: &fake.MockUserTokensClient{
				RevokeFn: func(_ *sonar.UserTokensRevokeOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errRenewToken),
			},
		},
		"GenerateAPIErrorAfterRevoke": {
			client: &fake.MockUserTokensClient{
				RevokeFn: func(_ *sonar.UserTokensRevokeOptions) (*http.Response, error) {
					return mockHTTPNoContent(), nil
				},
				GenerateFn: func(_ *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
					return nil, nil, errBoom
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  withRenewalDays(newToken(testTokenName, testTokenName), 30),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errRegenerateToken),
			},
		},
		"SuccessReturnsNewTokenInConnectionDetails": {
			client: &fake.MockUserTokensClient{
				RevokeFn: func(_ *sonar.UserTokensRevokeOptions) (*http.Response, error) {
					return mockHTTPNoContent(), nil
				},
				GenerateFn: func(_ *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
					return &sonar.UserTokensGenerate{
						Name:  testTokenName,
						Token: "new-secret-token-value",
					}, mockHTTPOK(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  withRenewalDays(newToken(testTokenName, testTokenName), 30),
			},
			want: want{
				o: managed.ExternalUpdate{
					ConnectionDetails: managed.ConnectionDetails{
						connectionSecretTokenKey: []byte("new-secret-token-value"),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Update() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Update() -want +got:\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalDelete
		err error
	}

	cases := map[string]struct {
		client *fake.MockUserTokensClient
		args   args
		want   want
	}{
		"NotUserTokenError": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: &notUserToken{}},
			want:   want{o: managed.ExternalDelete{}, err: errors.New(errNotUserToken)},
		},
		"EmptyExternalNameIsNoOp": {
			client: &fake.MockUserTokensClient{},
			args:   args{ctx: context.Background(), mg: newToken(testTokenName, "")},
			want:   want{o: managed.ExternalDelete{}, err: nil},
		},
		"RevokeAPIError": {
			client: &fake.MockUserTokensClient{
				RevokeFn: func(_ *sonar.UserTokensRevokeOptions) (*http.Response, error) {
					return nil, errBoom
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errBoom, errDeleteToken),
			},
		},
		"Success": {
			client: &fake.MockUserTokensClient{
				RevokeFn: func(_ *sonar.UserTokensRevokeOptions) (*http.Response, error) {
					return mockHTTPNoContent(), nil
				},
			},
			args: args{ctx: context.Background(), mg: newToken(testTokenName, testTokenName)},
			want: want{o: managed.ExternalDelete{}, err: nil},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, errComparer); diff != "" {
				t.Errorf("Delete() error -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Delete() -want +got:\n%s", diff)
			}
		})
	}
}

// TestCreatePassesCorrectOptions verifies Generate receives correct options.
func TestCreatePassesCorrectOptions(t *testing.T) {
	t.Parallel()

	var captured *sonar.UserTokensGenerateOptions

	tok := newToken(testTokenName, "")
	tok.Spec.ForProvider.Type = sonar.TokenTypeGlobalAnalysisToken

	e := &external{client: &fake.MockUserTokensClient{
		GenerateFn: func(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
			captured = opt

			return &sonar.UserTokensGenerate{Name: testTokenName, Token: "t"}, mockHTTPOK(), nil
		},
	}}

	_, err := e.Create(context.Background(), tok)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if captured == nil {
		t.Fatal("Create() Generate was not called")
	}

	if captured.Name != testTokenName {
		t.Errorf("Generate() Name = %q, want %q", captured.Name, testTokenName)
	}

	if captured.Type != sonar.TokenTypeGlobalAnalysisToken {
		t.Errorf("Generate() Type = %q, want %q", captured.Type, sonar.TokenTypeGlobalAnalysisToken)
	}
}

// TestDeletePassesExternalNameAsTokenName verifies Revoke is called
// with the external name as the token name.
func TestDeletePassesExternalNameAsTokenName(t *testing.T) {
	t.Parallel()

	var capturedName string

	e := &external{client: &fake.MockUserTokensClient{
		RevokeFn: func(opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
			capturedName = opt.Name

			return mockHTTPNoContent(), nil
		},
	}}

	_, err := e.Delete(context.Background(), newToken(testTokenName, testTokenName))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if capturedName != testTokenName {
		t.Errorf("Revoke() called with name %q, want %q", capturedName, testTokenName)
	}
}

// TestObserveProactiveRenewal verifies that Observe triggers renewal
// when RenewAt is in the past (proactive rotation before expiry).
func TestObserveProactiveRenewal(t *testing.T) {
	t.Parallel()

	// Token expires in 5 days; RenewBeforeDays = 7 → RenewAt = today - 2d.
	expirationDate := time.Now().AddDate(0, 0, 5).Format(time.DateOnly)

	var renewBeforeDays int64 = 7

	proactiveSearch := func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
		return &sonar.UserTokensSearch{
			UserTokens: []sonar.UserToken{
				{
					Name:           testTokenName,
					Type:           sonar.TokenTypeUserToken,
					ExpirationDate: expirationDate,
					IsExpired:      false,
				},
			},
		}, mockHTTPOK(), nil
	}

	tok := withRenewalDays(newToken(testTokenName, testTokenName), 30)
	tok.Spec.ForProvider.RenewBeforeDays = &renewBeforeDays

	e := &external{client: &fake.MockUserTokensClient{SearchFn: proactiveSearch}}

	obs, err := e.Observe(context.Background(), tok)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if obs.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = true, want false (proactive renewal due)")
	}
}

// TestObserveComputesRenewAt verifies that Observe populates RenewAt
// in AtProvider when RenewBeforeDays is set.
func TestObserveComputesRenewAt(t *testing.T) {
	t.Parallel()

	expirationDate := time.Now().AddDate(0, 0, 30).Format(time.DateOnly)

	var renewBeforeDays int64 = 7

	tok := withRenewalDays(newToken(testTokenName, testTokenName), 30)
	tok.Spec.ForProvider.RenewBeforeDays = &renewBeforeDays

	e := &external{client: &fake.MockUserTokensClient{
		SearchFn: func(_ *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
			return &sonar.UserTokensSearch{
				UserTokens: []sonar.UserToken{
					{
						Name:           testTokenName,
						Type:           sonar.TokenTypeUserToken,
						ExpirationDate: expirationDate,
					},
				},
			}, mockHTTPOK(), nil
		},
	}}

	_, err := e.Observe(context.Background(), tok)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if tok.Status.AtProvider.RenewAt == nil {
		t.Fatal("Observe() RenewAt = nil, want non-nil")
	}

	expected := time.Now().AddDate(0, 0, 30-int(renewBeforeDays))

	diff := tok.Status.AtProvider.RenewAt.Sub(expected)
	if diff < -24*time.Hour || diff > 24*time.Hour {
		t.Errorf("Observe() RenewAt = %v, want ~%v", tok.Status.AtProvider.RenewAt, expected)
	}
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockUserTokensClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v, want nil", err)
	}
}

const (
	// testProjectKey is a project key used in tests.
	testProjectKey = "test-project"
	// testLoginUser is a test user login name.
	testLoginUser = "testuser"
)

// TestCreateWithRenewalPeriod verifies Create computes ExpirationDate.
func TestCreateWithRenewalPeriod(t *testing.T) {
	t.Parallel()

	renewalDays := int64(30)

	e := &external{client: &fake.MockUserTokensClient{
		GenerateFn: func(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
			if opt.ExpirationDate == "" {
				t.Error("GenerateFn called with empty ExpirationDate")
			}

			return &sonar.UserTokensGenerate{
				Name:  testTokenName,
				Token: "secret",
			}, mockHTTPOK(), nil
		},
	}}

	token := &v1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: testTokenName},
		Spec: v1alpha1.UserTokenSpec{
			ForProvider: v1alpha1.UserTokenParameters{
				Name:              testTokenName,
				Type:              sonar.TokenTypeUserToken,
				RenewalPeriodDays: &renewalDays,
			},
		},
	}

	_, err := e.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

// TestCreateWithProjectAnalysisToken tests PROJECT_ANALYSIS_TOKEN.
func TestCreateWithProjectAnalysisToken(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockUserTokensClient{
		GenerateFn: func(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
			if opt.Type != sonar.TokenTypeProjectAnalysisToken {
				t.Errorf("GenerateFn Type = %q, want %q", opt.Type, sonar.TokenTypeProjectAnalysisToken)
			}

			if opt.ProjectKey != testProjectKey {
				t.Errorf("GenerateFn ProjectKey = %q, want %q", opt.ProjectKey, testProjectKey)
			}

			return &sonar.UserTokensGenerate{
				Name:  testTokenName,
				Token: "secret",
			}, mockHTTPOK(), nil
		},
	}}

	token := &v1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: testTokenName},
		Spec: v1alpha1.UserTokenSpec{
			ForProvider: v1alpha1.UserTokenParameters{
				Name:       testTokenName,
				Type:       sonar.TokenTypeProjectAnalysisToken,
				ProjectKey: new(string),
			},
		},
	}
	*token.Spec.ForProvider.ProjectKey = testProjectKey

	_, err := e.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

// TestUpdateWithLogin verifies Update passes login to Revoke and then
// regenerates the token.
func TestUpdateWithLogin(t *testing.T) {
	t.Parallel()

	var capturedRevokeOpt *sonar.UserTokensRevokeOptions

	e := &external{client: &fake.MockUserTokensClient{
		RevokeFn: func(opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
			capturedRevokeOpt = opt

			return mockHTTPNoContent(), nil
		},
		GenerateFn: func(_ *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
			return &sonar.UserTokensGenerate{Name: testTokenName, Token: "renewed-token"}, mockHTTPOK(), nil
		},
	}}

	token := &v1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: testTokenName},
		Spec: v1alpha1.UserTokenSpec{
			ForProvider: v1alpha1.UserTokenParameters{
				Name:  testTokenName,
				Type:  sonar.TokenTypeUserToken,
				Login: new(string),
			},
		},
	}
	*token.Spec.ForProvider.Login = testLoginUser
	meta.SetExternalName(token, testTokenName)

	_, err := e.Update(context.Background(), token)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if capturedRevokeOpt == nil {
		t.Fatal("Update() did not call Revoke")
	}

	if capturedRevokeOpt.Login != testLoginUser {
		t.Errorf("Update() Revoke Login = %q, want %q", capturedRevokeOpt.Login, testLoginUser)
	}
}

// TestDeleteWithLogin verifies Delete passes login to Revoke.
func TestDeleteWithLogin(t *testing.T) {
	t.Parallel()

	var capturedRevokeOpt *sonar.UserTokensRevokeOptions

	e := &external{client: &fake.MockUserTokensClient{
		RevokeFn: func(opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
			capturedRevokeOpt = opt

			return mockHTTPNoContent(), nil
		},
	}}

	token := &v1alpha1.UserToken{
		ObjectMeta: metav1.ObjectMeta{Name: testTokenName},
		Spec: v1alpha1.UserTokenSpec{
			ForProvider: v1alpha1.UserTokenParameters{
				Name:  testTokenName,
				Type:  sonar.TokenTypeUserToken,
				Login: new(string),
			},
		},
	}
	*token.Spec.ForProvider.Login = testLoginUser
	meta.SetExternalName(token, testTokenName)

	_, err := e.Delete(context.Background(), token)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if capturedRevokeOpt == nil {
		t.Fatal("Delete() did not call Revoke")
	}

	if capturedRevokeOpt.Login != testLoginUser {
		t.Errorf("Delete() Revoke Login = %q, want %q", capturedRevokeOpt.Login, testLoginUser)
	}
}

// TestConnectNotUserToken verifies Connect rejects non-UserToken resources.
func TestConnectNotUserToken(t *testing.T) {
	t.Parallel()

	c := &connector{
		kube:         nil,
		usage:        nil,
		newServiceFn: func(config common.Config) iam.UserTokensClient { return nil },
	}

	_, err := c.Connect(context.Background(), &notUserToken{})
	if err == nil {
		t.Fatal("Connect() error = nil, want error")
	}

	if err.Error() != errNotUserToken {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotUserToken)
	}
}
