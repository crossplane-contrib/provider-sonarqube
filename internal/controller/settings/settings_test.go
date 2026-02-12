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

package settings

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type notSettings struct {
	resource.Managed
}

// mockHTTPResponse creates a minimal mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
}

// errComparer compares error messages for testing.
func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Error() == b.Error()
}

//nolint:maintidx // Test function complexity is acceptable for comprehensive table-driven tests
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
		client *fake.MockSettingsClient
		args   args
		want   want
	}{
		"NotSettingsError": {
			client: &fake.MockSettingsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notSettings{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotSettings),
			},
		},
		"DeletingReturnsNotExists": {
			client: &fake.MockSettingsClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-settings",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ValuesCallFails": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					return nil, nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errors.New("api error"), "failed to get settings values"),
			},
		},
		"SuccessfulObserveResourceExists": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					return &sonar.SettingsValues{
						Settings: []sonar.SettingValue{
							{
								Key:   "sonar.core.serverBaseURL",
								Value: "https://sonarqube.example.com",
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"ResourceNotUpToDateWhenValuesDiffer": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					return &sonar.SettingsValues{
						Settings: []sonar.SettingValue{
							{
								Value: "https://different-url.com",
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				err: nil,
			},
		},
		"ResourceNotUpToDateWhenSettingMissing": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					return &sonar.SettingsValues{
						Settings: []sonar.SettingValue{},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				err: nil,
			},
		},
		"MultipleSettingsAllUpToDate": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					return &sonar.SettingsValues{
						Settings: []sonar.SettingValue{
							{
								Key:   "sonar.core.serverBaseURL",
								Value: "https://sonarqube.example.com",
							},
							{
								Key:    "sonar.exclusions",
								Values: []string{"**/*.test.js", "**/*.spec.js"},
							},
							{
								Key: "sonar.issue.enforce.multicriteria",
								FieldValues: []map[string]string{
									{
										"1.ruleKey":     "squid:S1134",
										"1.resourceKey": "**/*.java",
									},
								},
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.test.js", "**/*.spec.js"}),
								},
								"sonar.issue.enforce.multicriteria": {
									FieldValues: ptr.To(map[string]string{
										"1.ruleKey":     "squid:S1134",
										"1.resourceKey": "**/*.java",
									}),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"ObserveWithComponent": {
			client: &fake.MockSettingsClient{
				ValuesFn: func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
					// Verify component is passed correctly
					if opt.Component != "my-project-key" {
						return nil, nil, errors.New("expected component to be 'my-project-key'")
					}

					return &sonar.SettingsValues{
						Settings: []sonar.SettingValue{
							{
								Key:   "sonar.coverage.jacoco.xmlReportPaths",
								Value: "target/site/jacoco/jacoco.xml",
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-settings",
					},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Component: ptr.To("my-project-key"),
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.coverage.jacoco.xmlReportPaths": {
									Value: ptr.To("target/site/jacoco/jacoco.xml"),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{settingsClient: tc.client}
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

//nolint:maintidx // Test function complexity is acceptable for comprehensive table-driven tests
func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		client *fake.MockSettingsClient
		args   args
		want   want
	}{
		"NotSettingsError": {
			client: &fake.MockSettingsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notSettings{},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotSettings),
			},
		},
		"SetFailsForSingleSetting": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					return nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New("failed to set setting sonar.core.serverBaseURL: api error"),
			},
		},
		"SuccessfulCreateSingleSetting": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Key != "sonar.core.serverBaseURL" {
						return nil, errors.New("unexpected key: " + opt.Key)
					}

					if opt.Value != "https://sonarqube.example.com" {
						return nil, errors.New("unexpected value")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
		"SuccessfulCreateMultipleSettings": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					// Accept any valid setting key
					validKeys := map[string]bool{
						"sonar.core.serverBaseURL": true,
						"sonar.exclusions":         true,
					}
					if !validKeys[opt.Key] {
						return nil, errors.New("unexpected key: " + opt.Key)
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.test.js"}),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
		"CreateWithComponent": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Component != "my-project-key" {
						return nil, errors.New("expected component to be 'my-project-key'")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Component: ptr.To("my-project-key"),
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.coverage.jacoco.xmlReportPaths": {
									Value: ptr.To("target/site/jacoco/jacoco.xml"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
		"PartialFailureReturnsAllErrors": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					// Fail on one specific key
					if opt.Key == "sonar.exclusions" {
						return nil, errors.New("api error for exclusions")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.test.js"}),
								},
							},
						},
					},
				},
			},
			want: want{
				o: managed.ExternalCreation{},
				// Error should contain information about the failed setting
				err: errors.New("failed to set setting sonar.exclusions: api error for exclusions"),
			},
		},
		"CreateWithFieldValues": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Key != "sonar.issue.enforce.multicriteria" {
						return nil, errors.New("unexpected key: " + opt.Key)
					}

					if len(opt.FieldValues) == 0 {
						return nil, errors.New("expected field values to be set")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.issue.enforce.multicriteria": {
									FieldValues: ptr.To(map[string]string{
										"1.ruleKey":     "squid:S1134",
										"1.resourceKey": "**/*.java",
									}),
								},
							},
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

			e := &external{settingsClient: tc.client}
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

//nolint:maintidx // Test function complexity is acceptable for comprehensive table-driven tests
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
		client *fake.MockSettingsClient
		args   args
		want   want
	}{
		"NotSettingsError": {
			client: &fake.MockSettingsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notSettings{},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errNotSettings),
			},
		},
		"UpdateOutOfDateSetting": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Key != "sonar.core.serverBaseURL" {
						return nil, errors.New("unexpected key: " + opt.Key)
					}

					if opt.Value != "https://new-url.com" {
						return nil, errors.New("unexpected value")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://new-url.com"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://old-url.com",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
		"NoUpdateWhenAllSettingsUpToDate": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					return nil, errors.New("should not be called when settings are up to date")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://sonarqube.example.com",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
		"ResetSettingsNotInDesiredState": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					// Verify that the obsolete setting is being reset
					found := false

					for _, key := range opt.Keys {
						if key == "sonar.obsolete.setting" {
							found = true
						}
					}

					if !found {
						return nil, errors.New("expected sonar.obsolete.setting to be reset")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://sonarqube.example.com",
								},
								"sonar.obsolete.setting": {
									Value: "some-value",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
		"UpdateMultipleOutOfDateSettings": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					validKeys := map[string]bool{
						"sonar.core.serverBaseURL": true,
						"sonar.exclusions":         true,
					}
					if !validKeys[opt.Key] {
						return nil, errors.New("unexpected key: " + opt.Key)
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://new-url.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.new.js"}),
								},
								"sonar.uptodate.setting": {
									Value: ptr.To("same-value"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://old-url.com",
								},
								"sonar.exclusions": {
									Values: []string{"**/*.old.js"},
								},
								"sonar.uptodate.setting": {
									Value: "same-value",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
		"UpdateFailsForOneSetting": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Key == "sonar.exclusions" {
						return nil, errors.New("api error for exclusions")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://new-url.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.new.js"}),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://old-url.com",
								},
								"sonar.exclusions": {
									Values: []string{"**/*.old.js"},
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New("failed to update setting sonar.exclusions: api error for exclusions"),
			},
		},
		"ResetFailsForObsoleteSettings": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					return nil, errors.New("api error during reset")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.core.serverBaseURL": {
									Value: "https://sonarqube.example.com",
								},
								"sonar.obsolete.setting": {
									Value: "some-value",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New("failed to reset settings that are not in the desired state: api error during reset"),
			},
		},
		"UpdateWithComponent": {
			client: &fake.MockSettingsClient{
				SetFn: func(opt *sonar.SettingsSetOption) (*http.Response, error) {
					if opt.Component != "my-project-key" {
						return nil, errors.New("expected component to be 'my-project-key'")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Component: ptr.To("my-project-key"),
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.coverage.jacoco.xmlReportPaths": {
									Value: ptr.To("target/new/jacoco.xml"),
								},
							},
						},
					},
					Status: v1alpha1.SettingsStatus{
						AtProvider: v1alpha1.SettingsObservation{
							Settings: map[string]v1alpha1.SettingObservation{
								"sonar.coverage.jacoco.xmlReportPaths": {
									Value: "target/old/jacoco.xml",
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{settingsClient: tc.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Update() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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
		client *fake.MockSettingsClient
		args   args
		want   want
	}{
		"NotSettingsError": {
			client: &fake.MockSettingsClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notSettings{},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.New(errNotSettings),
			},
		},
		"SuccessfulDelete": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					// Verify all settings are being reset
					expectedKeys := map[string]bool{
						"sonar.core.serverBaseURL": true,
						"sonar.exclusions":         true,
					}
					if len(opt.Keys) != len(expectedKeys) {
						return nil, errors.New("expected all settings to be reset")
					}

					for _, key := range opt.Keys {
						if !expectedKeys[key] {
							return nil, errors.New("unexpected key: " + key)
						}
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
								"sonar.exclusions": {
									Values: ptr.To([]string{"**/*.test.js"}),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteWithComponent": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					if opt.Component != "my-project-key" {
						return nil, errors.New("expected component to be 'my-project-key'")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Component: ptr.To("my-project-key"),
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.coverage.jacoco.xmlReportPaths": {
									Value: ptr.To("target/site/jacoco/jacoco.xml"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteResetFails": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					return nil, errors.New("api error during reset")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{
								"sonar.core.serverBaseURL": {
									Value: ptr.To("https://sonarqube.example.com"),
								},
							},
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("api error during reset"), "failed to reset settings during deletion"),
			},
		},
		"DeleteEmptySettings": {
			client: &fake.MockSettingsClient{
				ResetFn: func(opt *sonar.SettingsResetOption) (*http.Response, error) {
					if len(opt.Keys) != 0 {
						return nil, errors.New("expected no keys to reset")
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.Settings{
					ObjectMeta: metav1.ObjectMeta{Name: "test-settings"},
					Spec: v1alpha1.SettingsSpec{
						ForProvider: v1alpha1.SettingsParameters{
							Settings: map[string]v1alpha1.SettingParameters{},
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

			e := &external{settingsClient: tc.client}
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

func TestDisconnect(t *testing.T) {
	t.Parallel()

	client := &fake.MockSettingsClient{}
	e := &external{settingsClient: client}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}
