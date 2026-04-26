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

package rule

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// notRule is a test type that does not implement Rule.
type notRule struct {
	resource.Managed
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}
}

// checkError verifies that an error matches expected behavior.
func checkError(t *testing.T, method, wantErr string, gotErr error) {
	t.Helper()

	if wantErr == "" && gotErr == nil {
		return
	}

	if wantErr == "" && gotErr != nil {
		t.Errorf("%s() unexpected error: %v", method, gotErr)

		return
	}

	if wantErr != "" && gotErr == nil {
		t.Errorf("%s() expected error containing %q, got nil", method, wantErr)

		return
	}

	if !strings.Contains(gotErr.Error(), wantErr) {
		t.Errorf("%s() error mismatch: want error containing %q, got %q", method, wantErr, gotErr.Error())
	}
}

// newTestRule creates a new Rule for testing with the given parameters.
func newTestRule(externalName string, spec v1alpha1.RuleParameters) *v1alpha1.Rule {
	rule := &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-rule",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.RuleSpec{
			ForProvider: spec,
		},
	}

	if externalName != "" {
		meta.SetExternalName(rule, externalName)
	}

	return rule
}

// newTestExternalClient creates a test external client.
func newTestExternalClient(rulesClient *fake.MockRulesClient) *external {
	return &external{
		rulesClient: rulesClient,
	}
}

// minimalRuleSpec returns a minimal rule specification for testing.
func minimalRuleSpec() v1alpha1.RuleParameters {
	return v1alpha1.RuleParameters{
		Key:                 "custom:rule",
		Name:                "Custom Rule",
		TemplateKey:         "java:TemplateRule",
		MarkdownDescription: "A custom rule",
	}
}

// TestObserve tests the Observe method for rules.
func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		errSubstr   string
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotRuleError": {
			ext:  newTestExternalClient(&fake.MockRulesClient{}),
			args: args{ctx: context.Background(), mg: &notRule{}},
			want: want{
				observation: managed.ExternalObservation{},
				errSubstr:   errNotRule,
			},
		},
		"EmptyExternalNameReturnsNotExists": {
			ext: newTestExternalClient(&fake.MockRulesClient{}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("", minimalRuleSpec()),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"ShowFailsReturnsError": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				ShowFn: func(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
					return nil, nil, errRulesNotImplemented
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{
				observation: managed.ExternalObservation{},
				errSubstr:   "failed to get SonarQube Rule",
			},
		},
		"SuccessfulObserveResourceUpToDate": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				ShowFn: func(opt *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
					return &sonar.RulesShow{
						Rule: sonar.RuleDetails{
							Key:         "custom:rule",
							Name:        "Custom Rule",
							TemplateKey: "java:TemplateRule",
							Status:      "READY",
							Severity:    "MAJOR",
						},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg: newTestRule("custom:rule", v1alpha1.RuleParameters{
					Key:                 "custom:rule",
					Name:                "Custom Rule",
					TemplateKey:         "java:TemplateRule",
					MarkdownDescription: "A rule",
					Severity:            ptr.To("MAJOR"),
					Status:              ptr.To("READY"),
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: false,
					ConnectionDetails:       managed.ConnectionDetails{},
				},
			},
		},
		"SuccessfulObserveResourceNotUpToDate": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				ShowFn: func(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
					return &sonar.RulesShow{
						Rule: sonar.RuleDetails{
							Key:         "custom:rule",
							Name:        "Different Name",
							TemplateKey: "java:TemplateRule",
						},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg: newTestRule("custom:rule", v1alpha1.RuleParameters{
					Key:                 "custom:rule",
					Name:                "Custom Rule",
					TemplateKey:         "java:TemplateRule",
					MarkdownDescription: "desc",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        false,
					ResourceLateInitialized: false,
					ConnectionDetails:       managed.ConnectionDetails{},
				},
			},
		},
		"RemovedRuleTreatedAsNotExists": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				ShowFn: func(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
					return &sonar.RulesShow{
						Rule: sonar.RuleDetails{
							Key:    "custom:rule",
							Status: "REMOVED",
						},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"NoLateInitializationForSeverity": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				ShowFn: func(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
					return &sonar.RulesShow{
						Rule: sonar.RuleDetails{
							Key:         "custom:rule",
							Name:        "Custom Rule",
							TemplateKey: "java:TemplateRule",
							Severity:    "MAJOR",
						},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg: newTestRule("custom:rule", v1alpha1.RuleParameters{
					Key:                 "custom:rule",
					Name:                "Custom Rule",
					TemplateKey:         "java:TemplateRule",
					MarkdownDescription: "desc",
					// Severity is nil and is intentionally not late-initialized.
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: false,
					ConnectionDetails:       managed.ConnectionDetails{},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Observe(tc.args.ctx, tc.args.mg)

			checkError(t, "Observe", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// errRulesNotImplemented is returned when a mock function is not implemented.
var errRulesNotImplemented = errors.New("mock function not implemented")

// TestCreate tests the Create method for rules.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation  managed.ExternalCreation
		errSubstr string
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotRuleError": {
			ext:  newTestExternalClient(&fake.MockRulesClient{}),
			args: args{ctx: context.Background(), mg: &notRule{}},
			want: want{
				creation:  managed.ExternalCreation{},
				errSubstr: errNotRule,
			},
		},
		"CreateFails": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				CreateFn: func(_ *sonar.RulesCreateOptions) (*sonar.RulesCreate, *http.Response, error) {
					return nil, nil, errRulesNotImplemented
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("", minimalRuleSpec()),
			},
			want: want{
				creation:  managed.ExternalCreation{},
				errSubstr: "failed to create SonarQube Rule",
			},
		},
		"SuccessfulCreate": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				CreateFn: func(_ *sonar.RulesCreateOptions) (*sonar.RulesCreate, *http.Response, error) {
					return &sonar.RulesCreate{
						Rule: sonar.Rule{Key: "custom:rule"},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("", minimalRuleSpec()),
			},
			want: want{
				creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
			},
		},
		"ExternalNameSetAfterCreate": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				CreateFn: func(_ *sonar.RulesCreateOptions) (*sonar.RulesCreate, *http.Response, error) {
					return &sonar.RulesCreate{
						Rule: sonar.Rule{Key: "custom:generated-key"},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("", minimalRuleSpec()),
			},
			want: want{
				creation: managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Create(tc.args.ctx, tc.args.mg)

			checkError(t, "Create", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestUpdate tests the Update method for rules.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		update    managed.ExternalUpdate
		errSubstr string
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotRuleError": {
			ext:  newTestExternalClient(&fake.MockRulesClient{}),
			args: args{ctx: context.Background(), mg: &notRule{}},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: errNotRule,
			},
		},
		"UpdateFails": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				UpdateFn: func(_ *sonar.RulesUpdateOptions) (*sonar.RulesUpdate, *http.Response, error) {
					return nil, nil, errRulesNotImplemented
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "failed to update SonarQube Rule",
			},
		},
		"SuccessfulUpdate": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				UpdateFn: func(_ *sonar.RulesUpdateOptions) (*sonar.RulesUpdate, *http.Response, error) {
					return &sonar.RulesUpdate{
						Rule: sonar.Rule{Key: "custom:rule"},
					}, mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{
				update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Update(tc.args.ctx, tc.args.mg)

			checkError(t, "Update", tc.want.errSubstr, err)

			if tc.want.errSubstr != "" {
				return
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method for rules.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		errSubstr string
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotRuleError": {
			ext:  newTestExternalClient(&fake.MockRulesClient{}),
			args: args{ctx: context.Background(), mg: &notRule{}},
			want: want{errSubstr: errNotRule},
		},
		"EmptyExternalNameSucceeds": {
			ext:  newTestExternalClient(&fake.MockRulesClient{}),
			args: args{ctx: context.Background(), mg: newTestRule("", minimalRuleSpec())},
			want: want{},
		},
		"DeleteFails": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				DeleteFn: func(_ *sonar.RulesDeleteOptions) (*http.Response, error) {
					return nil, errRulesNotImplemented
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{errSubstr: "failed to delete SonarQube Rule"},
		},
		"SuccessfulDelete": {
			ext: newTestExternalClient(&fake.MockRulesClient{
				DeleteFn: func(_ *sonar.RulesDeleteOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			}),
			args: args{
				ctx: context.Background(),
				mg:  newTestRule("custom:rule", minimalRuleSpec()),
			},
			want: want{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.ext.Delete(tc.args.ctx, tc.args.mg)

			checkError(t, "Delete", tc.want.errSubstr, err)
		})
	}
}

// TestDisconnect tests the Disconnect method for rules.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := newTestExternalClient(&fake.MockRulesClient{})

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestObserveSetsReadyCondition tests Observe sets ready condition.
func TestObserveSetsReadyCondition(t *testing.T) {
	t.Parallel()

	r := newTestRule("custom:rule", v1alpha1.RuleParameters{
		Key:         "custom:rule",
		Name:        "Custom Rule",
		TemplateKey: "java:TemplateRule",
	})

	e := newTestExternalClient(&fake.MockRulesClient{
		ShowFn: func(_ *sonar.RulesShowOptions) (*sonar.RulesShow, *http.Response, error) {
			return &sonar.RulesShow{Rule: sonar.RuleDetails{
				Key:         "custom:rule",
				Name:        "Custom Rule",
				TemplateKey: "java:TemplateRule",
			}}, mockHTTPResponse(), nil
		},
	})

	_, err := e.Observe(context.Background(), r)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	ready := r.Status.GetCondition(xpv1.TypeReady)
	if ready.Status != corev1.ConditionTrue {
		t.Fatalf("Observe() ready status = %s, want %s", ready.Status, corev1.ConditionTrue)
	}

	if ready.Reason != xpv1.ReasonAvailable {
		t.Fatalf("Observe() ready reason = %s, want %s", ready.Reason, xpv1.ReasonAvailable)
	}
}

// TestDeleteClearsExternalNameWhenDeleted tests Delete clears external name.
func TestDeleteClearsExternalNameWhenDeleted(t *testing.T) {
	t.Parallel()

	r := newTestRule("custom:rule", minimalRuleSpec())
	e := newTestExternalClient(&fake.MockRulesClient{
		DeleteFn: func(_ *sonar.RulesDeleteOptions) (*http.Response, error) {
			return mockHTTPResponse(), nil
		},
	})

	_, err := e.Delete(context.Background(), r)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if got := meta.GetExternalName(r); got != "" {
		t.Fatalf("Delete() external name = %q, want empty", got)
	}
}

// TestDeleteClearsExternalNameOnNotFound tests Delete clears name on not found.
func TestDeleteClearsExternalNameOnNotFound(t *testing.T) {
	t.Parallel()

	r := newTestRule("custom:rule", minimalRuleSpec())
	e := newTestExternalClient(&fake.MockRulesClient{
		DeleteFn: func(_ *sonar.RulesDeleteOptions) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"}, errors.New("not found")
		},
	})

	_, err := e.Delete(context.Background(), r)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if got := meta.GetExternalName(r); got != "" {
		t.Fatalf("Delete() external name = %q, want empty", got)
	}
}

// TestConnectTypeAssertion verifies that Connect returns errNotRule when the
// managed resource is not a *v1alpha1.Rule.
// TestConnectTypeAssertion tests Connect type assertion error handling.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &notRule{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Rule type, got nil")
	}

	if !strings.Contains(err.Error(), errNotRule) {
		t.Errorf("Connect() error: want %q, got %q", errNotRule, err.Error())
	}
}
