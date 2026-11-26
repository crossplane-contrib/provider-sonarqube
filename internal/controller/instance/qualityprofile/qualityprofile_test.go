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

package qualityprofile

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// errComparer compares errors by their message.
func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Error() == b.Error()
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) { //nolint:maintidx // table-driven test covering many cases
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  *v1alpha1.QualityProfile
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		qualityProfilesClient *fake.MockQualityProfilesClient
		rulesClient           *fake.MockRulesClient
		args                  args
		want                  want
	}{
		"EmptyExternalNameReturnsNotExists": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			rulesClient:           &fake.MockRulesClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.QualityProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-profile",
						Annotations: map[string]string{},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ShowFailsReturnsError": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
					return nil, nil, errors.New("api error")
				},
			},
			rulesClient: &fake.MockRulesClient{},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-profile",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errors.New("api error"), errShowQualityProfile),
			},
		},
		"ShowNotFoundReturnsNotExists": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
					return nil, &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, errors.New("not found")
				},
			},
			rulesClient: &fake.MockRulesClient{},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-profile",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"SuccessfulObserveResourceExists": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
					return &sonar.QualityprofilesShow{
						Profile: sonar.QualityprofilesShownProfile{
							Key:             "AU-TpxcA-iU5OvuD2FLz",
							Name:            "test-profile",
							Language:        "java",
							LanguageName:    "Java",
							IsBuiltIn:       false,
							IsDefault:       false,
							IsInherited:     false,
							ActiveRuleCount: 0,
						},
					}, nil, nil
				},
			},
			rulesClient: &fake.MockRulesClient{
				SearchFn: func(opt *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return &sonar.RulesSearch{
						Rules: []sonar.RulesDetails{},
						Paging: sonar.Paging{
							Total: 0,
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-profile",
							Annotations: map[string]string{},
						},
						Spec: v1alpha1.QualityProfileSpec{
							ForProvider: v1alpha1.QualityProfileParameters{
								Name:     "test-profile",
								Language: "java",
								Default:  new(false),
							},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: false,
				},
				err: nil,
			},
		},
		"ResourceNotUpToDateWhenNamesDiffer": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
					return &sonar.QualityprofilesShow{
						Profile: sonar.QualityprofilesShownProfile{
							Key:          "AU-TpxcA-iU5OvuD2FLz",
							Name:         "different-name",
							Language:     "java",
							LanguageName: "Java",
							IsBuiltIn:    false,
							IsDefault:    false,
						},
					}, nil, nil
				},
			},
			rulesClient: &fake.MockRulesClient{
				SearchFn: func(opt *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return &sonar.RulesSearch{
						Rules: []sonar.RulesDetails{},
						Paging: sonar.Paging{
							Total: 0,
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-profile",
							Annotations: map[string]string{},
						},
						Spec: v1alpha1.QualityProfileSpec{
							ForProvider: v1alpha1.QualityProfileParameters{
								Name:     "test-profile",
								Language: "java",
								Default:  new(false),
							},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        false,
					ResourceLateInitialized: false,
				},
				err: nil,
			},
		},
		"LateInitializeDefault": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
					return &sonar.QualityprofilesShow{
						Profile: sonar.QualityprofilesShownProfile{
							Key:          "AU-TpxcA-iU5OvuD2FLz",
							Name:         "test-profile",
							Language:     "java",
							LanguageName: "Java",
							IsBuiltIn:    false,
							IsDefault:    true,
						},
					}, nil, nil
				},
			},
			rulesClient: &fake.MockRulesClient{
				SearchFn: func(opt *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
					return &sonar.RulesSearch{
						Rules: []sonar.RulesDetails{},
						Paging: sonar.Paging{
							Total: 0,
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-profile",
							Annotations: map[string]string{},
						},
						Spec: v1alpha1.QualityProfileSpec{
							ForProvider: v1alpha1.QualityProfileParameters{
								Name:     "test-profile",
								Language: "java",
								Default:  nil,
							},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: true,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{
				qualityProfilesClient: tc.qualityProfilesClient,
				rulesClient:           tc.rulesClient,
			}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Observe() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  *v1alpha1.QualityProfile
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		qualityProfilesClient *fake.MockQualityProfilesClient
		args                  args
		want                  want
	}{
		"SuccessfulCreate": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				CreateFn: func(opt *sonar.QualityprofilesCreateOptions) (*sonar.QualityprofilesCreate, *http.Response, error) {
					return &sonar.QualityprofilesCreate{
						Profile: sonar.QualityprofilesCreatedProfile{
							Key:          "AU-TpxcA-iU5OvuD2FLz",
							Name:         opt.Name,
							Language:     opt.Language,
							LanguageName: "Java",
							IsInherited:  false,
							IsDefault:    false,
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.QualityProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-profile",
					},
					Spec: v1alpha1.QualityProfileSpec{
						ForProvider: v1alpha1.QualityProfileParameters{
							Name:     "test-profile",
							Language: "java",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
		"CreateFails": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				CreateFn: func(opt *sonar.QualityprofilesCreateOptions) (*sonar.QualityprofilesCreate, *http.Response, error) {
					return nil, nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.QualityProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-profile",
					},
					Spec: v1alpha1.QualityProfileSpec{
						ForProvider: v1alpha1.QualityProfileParameters{
							Name:     "test-profile",
							Language: "java",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("api error"), errCreateQualityProfile),
			},
		},
		"CreateWithDefault": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				CreateFn: func(opt *sonar.QualityprofilesCreateOptions) (*sonar.QualityprofilesCreate, *http.Response, error) {
					return &sonar.QualityprofilesCreate{
						Profile: sonar.QualityprofilesCreatedProfile{
							Key:      "AU-TpxcA-iU5OvuD2FLz",
							Name:     opt.Name,
							Language: opt.Language,
						},
					}, mockHTTPResponse(), nil
				},
				SetDefaultFn: func(opt *sonar.QualityprofilesSetDefaultOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.QualityProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-profile",
					},
					Spec: v1alpha1.QualityProfileSpec{
						ForProvider: v1alpha1.QualityProfileParameters{
							Name:     "test-profile",
							Language: "java",
							Default:  new(true),
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{qualityProfilesClient: tc.qualityProfilesClient}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Create() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  *v1alpha1.QualityProfile
	}

	type want struct {
		o   managed.ExternalDelete
		err error
	}

	cases := map[string]struct {
		qualityProfilesClient *fake.MockQualityProfilesClient
		args                  args
		want                  want
	}{
		"SuccessfulDelete": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeleteFn: func(opt *sonar.QualityprofilesDeleteOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-profile",
						},
						Spec: v1alpha1.QualityProfileSpec{
							ForProvider: v1alpha1.QualityProfileParameters{
								Name:     "test-profile",
								Language: "java",
							},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteFails": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeleteFn: func(opt *sonar.QualityprofilesDeleteOptions) (*http.Response, error) {
					return nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-profile",
						},
						Spec: v1alpha1.QualityProfileSpec{
							ForProvider: v1alpha1.QualityProfileParameters{
								Name:     "test-profile",
								Language: "java",
							},
						},
					}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("api error"), errDeleteQualityProfile),
			},
		},
		"EmptyExternalNameReturnsNil": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.QualityProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-profile",
						Annotations: map[string]string{},
					},
					Spec: v1alpha1.QualityProfileSpec{
						ForProvider: v1alpha1.QualityProfileParameters{
							Name:     "test-profile",
							Language: "java",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{qualityProfilesClient: tc.qualityProfilesClient}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Delete() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestSyncQualityProfileRules tests syncQualityProfileRules.
func TestSyncQualityProfileRules(t *testing.T) {
	t.Parallel()

	type args struct {
		cr           *v1alpha1.QualityProfile
		associations map[string]instance.QualityProfileRuleAssociation
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		qualityProfilesClient *fake.MockQualityProfilesClient
		args                  args
		want                  want
	}{
		"EmptyAssociations": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			args: args{
				cr: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
				associations: map[string]instance.QualityProfileRuleAssociation{},
			},
			want: want{err: nil},
		},
		"ActivateNewRules": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ActivateRuleFn: func(opt *sonar.QualityprofilesActivateRuleOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			args: args{
				cr: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
				associations: map[string]instance.QualityProfileRuleAssociation{
					"java:S1144": {
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144")},
						Observation: nil,
						UpToDate:    false,
					},
				},
			},
			want: want{err: nil},
		},
		"DeactivateUnwantedRules": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeactivateRuleFn: func(opt *sonar.QualityprofilesDeactivateRuleOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			args: args{
				cr: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
				associations: map[string]instance.QualityProfileRuleAssociation{
					"java:S1144": {
						Spec:        nil,
						Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
						UpToDate:    false,
					},
				},
			},
			want: want{err: nil},
		},
		"UpdateOutdatedRules": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ActivateRuleFn: func(opt *sonar.QualityprofilesActivateRuleOptions) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			args: args{
				cr: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
				associations: map[string]instance.QualityProfileRuleAssociation{
					"java:S1144": {
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1144"), Severity: new("CRITICAL")},
						Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144", Severity: "MAJOR"},
						UpToDate:    false,
					},
				},
			},
			want: want{err: nil},
		},
		"ErrorAggregation": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeactivateRuleFn: func(opt *sonar.QualityprofilesDeactivateRuleOptions) (*http.Response, error) {
					return nil, errors.New("deactivate error")
				},
				ActivateRuleFn: func(opt *sonar.QualityprofilesActivateRuleOptions) (*http.Response, error) {
					return nil, errors.New("activate error")
				},
			},
			args: args{
				cr: func() *v1alpha1.QualityProfile {
					qp := &v1alpha1.QualityProfile{}
					meta.SetExternalName(qp, "AU-TpxcA-iU5OvuD2FLz")

					return qp
				}(),
				associations: map[string]instance.QualityProfileRuleAssociation{
					"java:S1144": {
						Spec:        nil,
						Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144"},
						UpToDate:    false,
					},
					"java:S1145": {
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: new("java:S1145")},
						Observation: nil,
						UpToDate:    false,
					},
				},
			},
			want: want{
				// Error should be returned but should contain aggregated errors
				err: nil, // We just check it's not nil below
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{qualityProfilesClient: tc.qualityProfilesClient}
			err := e.syncQualityProfileRules(context.Background(), tc.args.cr, tc.args.associations)

			// Special case for error aggregation test
			if name == "ErrorAggregation" {
				if err == nil {
					t.Error("syncQualityProfileRules() expected error for ErrorAggregation case, got nil")
				}

				return
			}

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("syncQualityProfileRules() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObserveParent tests that Observe reports the parent of a Quality
// Profile and detects its drift.
func TestObserveParent(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		reason         string
		isInherited    bool
		inheritanceErr error
		parent         *string
		wantParent     string
		wantUpToDate   bool
		wantErr        error
	}{
		"UnmanagedParentIsUpToDate": {
			reason:       "An unset parent is not managed",
			isInherited:  true,
			wantParent:   "Sonar way",
			wantUpToDate: true,
		},
		"MatchingParentIsUpToDate": {
			reason:       "The desired parent is observed",
			isInherited:  true,
			parent:       new("Sonar way"),
			wantParent:   "Sonar way",
			wantUpToDate: true,
		},
		"MissingParentIsOutOfDate": {
			reason: "A desired parent is not set yet",
			parent: new("Sonar way"),
		},
		"UnwantedParentIsOutOfDate": {
			reason:      "An empty parent requires the inheritance to be removed",
			isInherited: true,
			parent:      new(""),
			wantParent:  "Sonar way",
		},
		"NoParentIsUpToDate": {
			reason:       "An empty parent matches a profile without parent",
			parent:       new(""),
			wantUpToDate: true,
		},
		"InheritanceError": {
			reason:         "A failure to read the inheritance is returned",
			isInherited:    true,
			inheritanceErr: errors.New("boom"),
			wantErr:        errors.Wrap(errors.New("boom"), errInheritanceQualityProfile),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := external{
				qualityProfilesClient: &fake.MockQualityProfilesClient{
					ShowFn: func(_ *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
						return &sonar.QualityprofilesShow{Profile: sonar.QualityprofilesShownProfile{
							Key: "key", Name: "child", Language: "go", IsInherited: tc.isInherited,
						}}, mockHTTPResponse(), nil
					},
					// Left unset for profiles without parent: calling it fails.
					InheritanceFn: func(opt *sonar.QualityprofilesInheritanceOptions) (*sonar.QualityprofilesInheritance, *http.Response, error) {
						if !tc.isInherited || opt.QualityProfile != "child" || opt.Language != "go" {
							return nil, nil, errors.Errorf("unexpected inheritance call: %+v", opt)
						}

						if tc.inheritanceErr != nil {
							return nil, nil, tc.inheritanceErr
						}

						return &sonar.QualityprofilesInheritance{
							Ancestors: []sonar.QualityprofilesInheritanceProfile{{Key: "parent", Name: "Sonar way"}},
						}, mockHTTPResponse(), nil
					},
				},
				rulesClient: &fake.MockRulesClient{
					SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
						return &sonar.RulesSearch{
							Rules:   []sonar.RulesDetails{{Key: "go:S100"}},
							Actives: map[string][]sonar.RulesActivation{"go:S100": {{QProfile: "key", Inherit: "INHERITED"}}},
							Paging:  sonar.Paging{Total: 1},
						}, mockHTTPResponse(), nil
					},
				},
			}

			profile := &v1alpha1.QualityProfile{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{meta.AnnotationKeyExternalName: "key"}},
				Spec: v1alpha1.QualityProfileSpec{ForProvider: v1alpha1.QualityProfileParameters{
					Name: "child", Language: "go", Parent: tc.parent,
				}},
			}

			got, err := e.Observe(context.Background(), profile)
			if diff := cmp.Diff(tc.wantErr, err, cmp.Comparer(errComparer)); diff != "" {
				t.Fatalf("\n%s\ne.Observe(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if got.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("\n%s\ne.Observe(...): ResourceUpToDate = %t, want %t", tc.reason, got.ResourceUpToDate, tc.wantUpToDate)
			}

			if profile.Status.AtProvider.Parent != tc.wantParent {
				t.Errorf("\n%s\ne.Observe(...): parent = %q, want %q", tc.reason, profile.Status.AtProvider.Parent, tc.wantParent)
			}

			if diff := cmp.Diff(tc.parent, profile.Spec.ForProvider.Parent); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): parent must not be late initialized: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestUpdateParent tests that Update changes the parent before synchronizing
// the rules against the resulting profile.
func TestUpdateParent(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	cases := map[string]struct {
		reason    string
		parent    *string
		changeErr error
		searchErr error
		wantCalls []string
		wantErr   error
	}{
		"UnmanagedParent": {
			reason:    "The parent is left alone, and stale rules are synchronized",
			wantCalls: []string{"reset go:S2", "activate go:S3"},
		},
		"UpToDateParent": {
			reason:    "The parent is not changed when it already matches",
			parent:    new("old"),
			wantCalls: []string{"reset go:S2", "activate go:S3"},
		},
		"SetParent": {
			reason:    "Rules inherited from the new parent are neither activated nor deactivated",
			parent:    new("new"),
			wantCalls: []string{"parent new"},
		},
		"RemoveParent": {
			reason:    "An empty parent removes the inheritance",
			parent:    new(""),
			wantCalls: []string{"parent "},
		},
		"ChangeParentError": {
			reason:    "A failure to change the parent stops the update",
			parent:    new("new"),
			changeErr: errBoom,
			wantCalls: []string{"parent new"},
			wantErr:   errors.Wrap(errBoom, errParentQualityProfile),
		},
		"RefreshRulesError": {
			reason:    "A failure to read the rules after the change is returned",
			parent:    new("new"),
			searchErr: errBoom,
			wantCalls: []string{"parent new"},
			wantErr:   errors.Wrap(errBoom, errShowQualityProfile),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls []string

			e := external{
				qualityProfilesClient: &fake.MockQualityProfilesClient{
					ChangeParentFn: func(opt *sonar.QualityprofilesChangeParentOptions) (*http.Response, error) {
						calls = append(calls, "parent "+opt.ParentQualityProfile)

						if tc.changeErr != nil {
							return nil, tc.changeErr
						}

						return mockHTTPResponse(), nil
					},
					ActivateRuleFn: func(opt *sonar.QualityprofilesActivateRuleOptions) (*http.Response, error) {
						verb := "activate "
						if opt.Reset {
							verb = "reset "
						}

						calls = append(calls, verb+opt.Rule)

						return mockHTTPResponse(), nil
					},
				},
				rulesClient: &fake.MockRulesClient{
					SearchFn: func(_ *sonar.RulesSearchOptions) (*sonar.RulesSearch, *http.Response, error) {
						return &sonar.RulesSearch{
							Rules:   []sonar.RulesDetails{{Key: "go:S3"}},
							Actives: map[string][]sonar.RulesActivation{"go:S3": {{QProfile: "key", Inherit: "INHERITED"}}},
							Paging:  sonar.Paging{Total: 1},
						}, mockHTTPResponse(), tc.searchErr
					},
				},
			}

			profile := &v1alpha1.QualityProfile{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{meta.AnnotationKeyExternalName: "key"}},
				Spec: v1alpha1.QualityProfileSpec{ForProvider: v1alpha1.QualityProfileParameters{
					Name: "child", Language: "go", Parent: tc.parent,
					Rules: []v1alpha1.QualityProfileRuleParameters{{Rule: new("go:S3")}},
				}},
				Status: v1alpha1.QualityProfileStatus{AtProvider: v1alpha1.QualityProfileObservation{
					Name: "child", Parent: "old",
					Rules: []v1alpha1.QualityProfileRuleObservation{
						{Key: "go:S1", Inheritance: "INHERITED"},
						{Key: "go:S2", Inheritance: "OVERRIDES"},
					},
				}},
			}

			_, err := e.Update(context.Background(), profile)
			if diff := cmp.Diff(tc.wantErr, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.wantCalls, calls); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want calls, +got calls:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestResetUnlistedQualityProfileRuleOverridesError tests that a failed reset
// is reported.
func TestResetUnlistedQualityProfileRuleOverridesError(t *testing.T) {
	t.Parallel()

	e := external{qualityProfilesClient: &fake.MockQualityProfilesClient{}}
	associations := instance.GenerateQualityProfileRulesAssociation(nil, []v1alpha1.QualityProfileRuleObservation{{Key: "go:S1", Inheritance: "OVERRIDES"}})

	errs := e.resetUnlistedQualityProfileRuleOverrides(context.Background(), "key", associations)
	if len(errs) != 1 {
		t.Errorf("resetUnlistedQualityProfileRuleOverrides(...): got %d errors, want 1", len(errs))
	}
}
