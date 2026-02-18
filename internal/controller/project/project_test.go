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

package project

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

type notProject struct {
	resource.Managed
}

func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}
}

// checkError is a helper to compare errors.
// If wantSubstr is not empty, it checks that the actual error message contains wantSubstr.
func checkError(t *testing.T, method string, wantErr error, gotErr error) {
	t.Helper()

	if wantErr == nil && gotErr == nil {
		return
	}

	if wantErr == nil && gotErr != nil {
		t.Errorf("%s() unexpected error: %v", method, gotErr)

		return
	}

	if wantErr != nil && gotErr == nil {
		t.Errorf("%s() expected error containing %q, got nil", method, wantErr.Error())

		return
	}

	if !strings.Contains(gotErr.Error(), wantErr.Error()) {
		t.Errorf("%s() error mismatch: want error containing %q, got %q", method, wantErr.Error(), gotErr.Error())
	}
}

func newTestProject(externalName string, spec v1alpha1.ProjectParameters) *v1alpha1.Project {
	proj := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-project",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.ProjectSpec{
			ForProvider: spec,
		},
	}
	if externalName != "" {
		meta.SetExternalName(proj, externalName)
	}

	return proj
}

func newTestExternalClient(
	projectsClient *fake.MockProjectsClient,
	projectLinksClient *fake.MockProjectLinksClient,
	projectBranchesClient *fake.MockProjectBranchesClient,
	newCodePeriodsClient *fake.MockNewCodePeriodsClient,
	qualityGatesClient *fake.MockQualityGatesClient,
	qualityProfilesClient *fake.MockQualityProfilesClient,
	projectTagsClient *fake.MockProjectTagsClient,
) *external {
	return &external{
		projectsClient:              projectsClient,
		projectLinksClient:          projectLinksClient,
		projectBranchesClient:       projectBranchesClient,
		projectNewCodePeriodsClient: newCodePeriodsClient,
		qualityGatesClient:          qualityGatesClient,
		qualityProfilesClient:       qualityProfilesClient,
		projectTagsClient:           projectTagsClient,
	}
}

func defaultMockClients() (*fake.MockProjectsClient, *fake.MockProjectLinksClient, *fake.MockProjectBranchesClient, *fake.MockNewCodePeriodsClient, *fake.MockQualityGatesClient, *fake.MockQualityProfilesClient, *fake.MockProjectTagsClient) {
	return &fake.MockProjectsClient{},
		&fake.MockProjectLinksClient{},
		&fake.MockProjectBranchesClient{},
		&fake.MockNewCodePeriodsClient{},
		&fake.MockQualityGatesClient{},
		&fake.MockQualityProfilesClient{},
		&fake.MockProjectTagsClient{}
}

func successfulObserveMocks() (*fake.MockProjectsClient, *fake.MockProjectLinksClient, *fake.MockProjectBranchesClient, *fake.MockNewCodePeriodsClient, *fake.MockQualityGatesClient, *fake.MockQualityProfilesClient, *fake.MockProjectTagsClient) {
	projectsClient := &fake.MockProjectsClient{
		SearchFn: func(opt *sonar.ProjectsSearchOption) (*sonar.ProjectsSearch, *http.Response, error) {
			return &sonar.ProjectsSearch{
				Components: []sonar.ProjectSearchComponent{
					{Key: "test-key", Name: "test-project", Visibility: "public", Qualifier: "TRK"},
				},
			}, mockHTTPResponse(), nil
		},
	}
	branchesClient := &fake.MockProjectBranchesClient{
		ListFn: func(opt *sonar.ProjectBranchesListOption) (*sonar.ProjectBranchesList, *http.Response, error) {
			return &sonar.ProjectBranchesList{
				Branches: []sonar.Branch{
					{Name: "main", IsMain: true, Type: "LONG"},
				},
			}, mockHTTPResponse(), nil
		},
	}
	linksClient := &fake.MockProjectLinksClient{
		SearchFn: func(opt *sonar.ProjectLinksSearchOption) (*sonar.ProjectLinksSearch, *http.Response, error) {
			return &sonar.ProjectLinksSearch{Links: []sonar.ProjectLink{}}, mockHTTPResponse(), nil
		},
	}
	ncpClient := &fake.MockNewCodePeriodsClient{
		ShowFn: func(opt *sonar.NewCodePeriodsShowOption) (*sonar.NewCodePeriodsShow, *http.Response, error) {
			return &sonar.NewCodePeriodsShow{Type: "PREVIOUS_VERSION", Inherited: true}, mockHTTPResponse(), nil
		},
		ListFn: func(opt *sonar.NewCodePeriodsListOption) (*sonar.NewCodePeriodsList, *http.Response, error) {
			return &sonar.NewCodePeriodsList{NewCodePeriods: []sonar.NewCodePeriod{}}, mockHTTPResponse(), nil
		},
	}
	qgClient := &fake.MockQualityGatesClient{
		GetByProjectFn: func(opt *sonar.QualitygatesGetByProjectOption) (*sonar.QualitygatesGetByProject, *http.Response, error) {
			return &sonar.QualitygatesGetByProject{
				QualityGate: sonar.ProjectQualityGate{Name: "Sonar way"},
			}, mockHTTPResponse(), nil
		},
	}
	qpClient := &fake.MockQualityProfilesClient{
		SearchFn: func(opt *sonar.QualityprofilesSearchOption) (*sonar.QualityprofilesSearch, *http.Response, error) {
			return &sonar.QualityprofilesSearch{
				Profiles: []sonar.QualityProfile{
					{Key: "java-profile", Name: "Java Default", Language: "java"},
				},
			}, mockHTTPResponse(), nil
		},
	}
	tagsClient := &fake.MockProjectTagsClient{}

	return projectsClient, linksClient, branchesClient, ncpClient, qgClient, qpClient, tagsClient
}

func TestObserve(t *testing.T) { //nolint:maintidx // table-driven test with many cases
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
		"NotProjectError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg:  &notProject{},
			},
			want: want{
				observation: managed.ExternalObservation{},
				errSubstr:   errNotProject,
			},
		},
		"EmptyExternalNameReturnsNotExists": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("", v1alpha1.ProjectParameters{
					Name: "test",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"SearchFailsReturnsError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.SearchFn = func(opt *sonar.ProjectsSearchOption) (*sonar.ProjectsSearch, *http.Response, error) {
					return nil, nil, errors.New("api error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{},
				errSubstr:   "cannot observe Project existence",
			},
		},
		"SearchReturnsEmptyComponentsNoError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.SearchFn = func(opt *sonar.ProjectsSearchOption) (*sonar.ProjectsSearch, *http.Response, error) {
					return &sonar.ProjectsSearch{Components: []sonar.ProjectSearchComponent{}}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"SearchReturnsWrongKeyReturnsNotExists": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.SearchFn = func(opt *sonar.ProjectsSearchOption) (*sonar.ProjectsSearch, *http.Response, error) {
					return &sonar.ProjectsSearch{
						Components: []sonar.ProjectSearchComponent{
							{Key: "other-key", Name: "other"},
						},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"SuccessfulObserve": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: ptr.To("public"),
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists: true,
				},
			},
		},
		"ObserveWithBranchListError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				b.ListFn = func(opt *sonar.ProjectBranchesListOption) (*sonar.ProjectBranchesList, *http.Response, error) {
					return nil, nil, errors.New("branch error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot observe Project branches",
			},
		},
		"ObserveWithLinksSearchError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOption) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return nil, nil, errors.New("links error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot observe Project links",
			},
		},
		"ObserveWithNewCodePeriodShowError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				n.ShowFn = func(opt *sonar.NewCodePeriodsShowOption) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return nil, nil, errors.New("ncp error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot show Project new code period",
			},
		},
		"ObserveWithQualityGateError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				q.GetByProjectFn = func(opt *sonar.QualitygatesGetByProjectOption) (*sonar.QualitygatesGetByProject, *http.Response, error) {
					return nil, nil, errors.New("qg error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot observe Project quality gate",
			},
		},
		"ObserveWithQualityProfileError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				qp.SearchFn = func(opt *sonar.QualityprofilesSearchOption) (*sonar.QualityprofilesSearch, *http.Response, error) {
					return nil, nil, errors.New("qp error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot observe Project quality profile",
			},
		},
		"ObserveWithNewCodePeriodsListError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				n.ListFn = func(opt *sonar.NewCodePeriodsListOption) (*sonar.NewCodePeriodsList, *http.Response, error) {
					return nil, nil, errors.New("ncp list error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				observation: managed.ExternalObservation{ResourceExists: true},
				errSubstr:   "cannot list Project new code periods",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Observe(tc.args.ctx, tc.args.mg)

			if tc.want.errSubstr == "" {
				if err != nil {
					t.Errorf("Observe() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Observe() expected error containing %q, got nil", tc.want.errSubstr)
				} else if !strings.Contains(err.Error(), tc.want.errSubstr) {
					t.Errorf("Observe() error: want substring %q, got %q", tc.want.errSubstr, err.Error())
				}
			}

			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		creation managed.ExternalCreation
		err      error
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotProjectError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg:  &notProject{},
			},
			want: want{
				creation: managed.ExternalCreation{},
				err:      errors.New(errNotProject),
			},
		},
		"CreateFailsReturnsError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.CreateFn = func(opt *sonar.ProjectsCreateOption) (*sonar.ProjectsCreate, *http.Response, error) {
					return nil, nil, errors.New("create error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				creation: managed.ExternalCreation{},
				err:      errors.Wrap(errors.New("create error"), "cannot create Project"),
			},
		},
		"SuccessfulCreate": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.CreateFn = func(opt *sonar.ProjectsCreateOption) (*sonar.ProjectsCreate, *http.Response, error) {
					return &sonar.ProjectsCreate{
						Project: sonar.Project{Key: "test-key"},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				creation: managed.ExternalCreation{},
				err:      nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Create(tc.args.ctx, tc.args.mg)

			checkError(t, "Create", tc.want.err, err)

			if diff := cmp.Diff(tc.want.creation, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdate(t *testing.T) { //nolint:maintidx // table-driven test with many cases
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
		"NotProjectError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg:  &notProject{},
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: errNotProject,
			},
		},
		"EmptyExternalNameReturnsError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "external name is not set",
			},
		},
		"SuccessfulUpdateNoChanges": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name: "test-project",
						Key:  "test-key",
					})
					proj.Status.AtProvider = v1alpha1.ProjectObservation{}

					return proj
				}(),
			},
			want: want{
				update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}},
			},
		},
		"UpdateVisibilityFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.UpdateVisibilityFn = func(opt *sonar.ProjectsUpdateVisibilityOption) (*http.Response, error) {
					return nil, errors.New("visibility error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name:       "test-project",
						Key:        "test-key",
						Visibility: ptr.To("private"),
					})
					proj.Status.AtProvider.Visibility = "public"

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot update Project visibility",
			},
		},
		"UpdateTagsFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				tg.SetFn = func(opt *sonar.ProjectTagsSetOption) (*http.Response, error) {
					return nil, errors.New("tags error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name: "test-project",
						Key:  "test-key",
						Tags: &[]string{"tag1"},
					})

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot update Project tags",
			},
		},
		"UpdateSuccessWithVisibilityChange": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.UpdateVisibilityFn = func(opt *sonar.ProjectsUpdateVisibilityOption) (*http.Response, error) {
					return mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name:       "test-project",
						Key:        "test-key",
						Visibility: ptr.To("private"),
					})
					proj.Status.AtProvider.Visibility = "public"

					return proj
				}(),
			},
			want: want{
				update: managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}},
			},
		},
		"UpdateDefaultBranchFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				b.SetMainFn = func(opt *sonar.ProjectBranchesSetMainOption) (*http.Response, error) {
					return nil, errors.New("branch error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name:          "test-project",
						Key:           "test-key",
						DefaultBranch: ptr.To("develop"),
					})
					proj.Status.AtProvider.DefaultBranch = "main"

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot update Project default branch",
			},
		},
		"UpdateQualityGateFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				q.SelectFn = func(opt *sonar.QualitygatesSelectOption) (*http.Response, error) {
					return nil, errors.New("qg error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name:            "test-project",
						Key:             "test-key",
						QualityGateName: ptr.To("new-gate"),
					})
					proj.Status.AtProvider.QualityGateName = "old-gate"

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot update Project quality gate",
			},
		},
		"UpdateNewCodePeriodFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				n.SetFn = func(opt *sonar.NewCodePeriodsSetOption) (*http.Response, error) {
					return nil, errors.New("ncp error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name: "test-project",
						Key:  "test-key",
						NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
							Type:  "NUMBER_OF_DAYS",
							Value: ptr.To("30"),
						},
					})
					proj.Status.AtProvider.NewCodePeriod = v1alpha1.ProjectNewCodePeriodObservation{
						Type:  "PREVIOUS_VERSION",
						Value: "",
					}

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot update Project new code period",
			},
		},
		"UpdateProjectLinksFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				l.CreateFn = func(opt *sonar.ProjectLinksCreateOption) (*sonar.ProjectLinksCreate, *http.Response, error) {
					return nil, nil, errors.New("link create error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name: "test-project",
						Key:  "test-key",
						Links: []v1alpha1.ProjectLinkParameters{
							{Name: "homepage", URL: "https://example.com"},
						},
					})

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot create Project link",
			},
		},
		"UpdateQualityProfileFails": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				qp.ShowFn = func(opt *sonar.QualityprofilesShowOption) (*sonar.QualityprofilesShow, *http.Response, error) {
					return nil, nil, errors.New("qp show error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.Project {
					proj := newTestProject("test-key", v1alpha1.ProjectParameters{
						Name: "test-project",
						Key:  "test-key",
						QualityProfiles: map[string]v1alpha1.ProjectQualityProfileReference{
							"java": {Id: ptr.To("new-profile-id")},
						},
					})
					proj.Status.AtProvider.QualityProfiles = map[string]v1alpha1.ProjectQualityProfileObservation{
						"java": {Id: "old-profile-id"},
					}

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "cannot retrieve quality profile",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Update(tc.args.ctx, tc.args.mg)

			if tc.want.errSubstr == "" {
				if err != nil {
					t.Errorf("Update() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Update() expected error containing %q, got nil", tc.want.errSubstr)
				} else if !strings.Contains(err.Error(), tc.want.errSubstr) {
					t.Errorf("Update() error: want substring %q, got %q", tc.want.errSubstr, err.Error())
				}
			}

			if diff := cmp.Diff(tc.want.update, got); diff != "" {
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
		deletion managed.ExternalDelete
		err      error
	}

	cases := map[string]struct {
		ext  *external
		args args
		want want
	}{
		"NotProjectError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg:  &notProject{},
			},
			want: want{
				deletion: managed.ExternalDelete{},
				err:      errors.New(errNotProject),
			},
		},
		"EmptyExternalNameReturnsSuccess": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				deletion: managed.ExternalDelete{},
				err:      nil,
			},
		},
		"DeleteFailsReturnsError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.DeleteFn = func(opt *sonar.ProjectsDeleteOption) (*http.Response, error) {
					return nil, errors.New("delete error")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				deletion: managed.ExternalDelete{},
				err:      errors.Wrap(errors.New("delete error"), "cannot delete Project"),
			},
		},
		"SuccessfulDelete": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := defaultMockClients()
				p.DeleteFn = func(opt *sonar.ProjectsDeleteOption) (*http.Response, error) {
					return mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
				}),
			},
			want: want{
				deletion: managed.ExternalDelete{},
				err:      nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.ext.Delete(tc.args.ctx, tc.args.mg)

			checkError(t, "Delete", tc.want.err, err)

			if diff := cmp.Diff(tc.want.deletion, got); diff != "" {
				t.Errorf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	p, l, b, n, q, qp, tg := defaultMockClients()
	ext := newTestExternalClient(p, l, b, n, q, qp, tg)

	err := ext.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}
