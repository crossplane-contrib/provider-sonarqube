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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

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

type notQualityProfile struct {
	resource.Managed
}

func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Error() == b.Error()
}

func TestObserve(t *testing.T) {
	type args struct {
		ctx context.Context
		mg  resource.Managed
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
		"NotQualityProfileError": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			rulesClient:           &fake.MockRulesClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notQualityProfile{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotQualityProfile),
			},
		},
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
		"ShowFailsReturnsNotExists": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonargo.QualityprofilesShowOption) (*sonargo.QualityprofilesShowObject, *http.Response, error) {
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
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"SuccessfulObserveResourceExists": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				ShowFn: func(opt *sonargo.QualityprofilesShowOption) (*sonargo.QualityprofilesShowObject, *http.Response, error) {
					return &sonargo.QualityprofilesShowObject{
						Profile: sonargo.QualityprofilesShowObject_sub1{
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
				SearchFn: func(opt *sonargo.RulesSearchOption) (*sonargo.RulesSearchObject, *http.Response, error) {
					return &sonargo.RulesSearchObject{
						Rules: []sonargo.RulesSearchObject_sub11{},
						Paging: sonargo.RulesSearchObject_sub6{
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
								Default:  ptr.To(false),
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
				ShowFn: func(opt *sonargo.QualityprofilesShowOption) (*sonargo.QualityprofilesShowObject, *http.Response, error) {
					return &sonargo.QualityprofilesShowObject{
						Profile: sonargo.QualityprofilesShowObject_sub1{
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
				SearchFn: func(opt *sonargo.RulesSearchOption) (*sonargo.RulesSearchObject, *http.Response, error) {
					return &sonargo.RulesSearchObject{
						Rules: []sonargo.RulesSearchObject_sub11{},
						Paging: sonargo.RulesSearchObject_sub6{
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
								Default:  ptr.To(false),
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
				ShowFn: func(opt *sonargo.QualityprofilesShowOption) (*sonargo.QualityprofilesShowObject, *http.Response, error) {
					return &sonargo.QualityprofilesShowObject{
						Profile: sonargo.QualityprofilesShowObject_sub1{
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
				SearchFn: func(opt *sonargo.RulesSearchOption) (*sonargo.RulesSearchObject, *http.Response, error) {
					return &sonargo.RulesSearchObject{
						Rules: []sonargo.RulesSearchObject_sub11{},
						Paging: sonargo.RulesSearchObject_sub6{
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

func TestCreate(t *testing.T) {
	type args struct {
		ctx context.Context
		mg  resource.Managed
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
		"NotQualityProfileError": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notQualityProfile{},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotQualityProfile),
			},
		},
		"SuccessfulCreate": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				CreateFn: func(opt *sonargo.QualityprofilesCreateOption) (*sonargo.QualityprofilesCreateObject, *http.Response, error) {
					return &sonargo.QualityprofilesCreateObject{
						Profile: sonargo.QualityprofilesCreateObject_sub1{
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
				CreateFn: func(opt *sonargo.QualityprofilesCreateOption) (*sonargo.QualityprofilesCreateObject, *http.Response, error) {
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
				CreateFn: func(opt *sonargo.QualityprofilesCreateOption) (*sonargo.QualityprofilesCreateObject, *http.Response, error) {
					return &sonargo.QualityprofilesCreateObject{
						Profile: sonargo.QualityprofilesCreateObject_sub1{
							Key:      "AU-TpxcA-iU5OvuD2FLz",
							Name:     opt.Name,
							Language: opt.Language,
						},
					}, nil, nil
				},
				SetDefaultFn: func(opt *sonargo.QualityprofilesSetDefaultOption) (*http.Response, error) {
					return nil, nil
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
							Default:  ptr.To(true),
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

func TestDelete(t *testing.T) {
	type args struct {
		ctx context.Context
		mg  resource.Managed
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
		"NotQualityProfileError": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notQualityProfile{},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.New(errNotQualityProfile),
			},
		},
		"SuccessfulDelete": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeleteFn: func(opt *sonargo.QualityprofilesDeleteOption) (*http.Response, error) {
					return nil, nil
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
				DeleteFn: func(opt *sonargo.QualityprofilesDeleteOption) (*http.Response, error) {
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

func TestSyncQualityProfileRules(t *testing.T) {
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
				ActivateRuleFn: func(opt *sonargo.QualityprofilesActivateRuleOption) (*http.Response, error) {
					return nil, nil
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
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144"},
						Observation: nil,
						UpToDate:    false,
					},
				},
			},
			want: want{err: nil},
		},
		"DeactivateUnwantedRules": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeactivateRuleFn: func(opt *sonargo.QualityprofilesDeactivateRuleOption) (*http.Response, error) {
					return nil, nil
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
				ActivateRuleFn: func(opt *sonargo.QualityprofilesActivateRuleOption) (*http.Response, error) {
					return nil, nil
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
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1144", Severity: ptr.To("CRITICAL")},
						Observation: &v1alpha1.QualityProfileRuleObservation{Key: "java:S1144", Severity: "MAJOR"},
						UpToDate:    false,
					},
				},
			},
			want: want{err: nil},
		},
		"ErrorAggregation": {
			qualityProfilesClient: &fake.MockQualityProfilesClient{
				DeactivateRuleFn: func(opt *sonargo.QualityprofilesDeactivateRuleOption) (*http.Response, error) {
					return nil, errors.New("deactivate error")
				},
				ActivateRuleFn: func(opt *sonargo.QualityprofilesActivateRuleOption) (*http.Response, error) {
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
						Spec:        &v1alpha1.QualityProfileRuleParameters{Rule: "java:S1145"},
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
			e := &external{qualityProfilesClient: tc.qualityProfilesClient}
			err := e.syncQualityProfileRules(tc.args.cr, tc.args.associations)

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
