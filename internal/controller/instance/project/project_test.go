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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

const (
	// projectVisibilityPublic is a test project visibility.
	projectVisibilityPublic = "public"
)

// notProject is a type for testing non-Project resources.
type notProject struct {
	resource.Managed
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}
}

// checkError is a helper to compare errors.
// If wantSubstr is not empty, it checks that the actual error message contains
// wantSubstr.
func checkError(t *testing.T, method string, wantErr, gotErr error) {
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

// newTestProject creates a test Project resource.
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

// newTestExternalClient creates a test external client with mock clients.
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

// defaultMockClients returns default mock clients for testing.
func defaultMockClients() (*fake.MockProjectsClient, *fake.MockProjectLinksClient, *fake.MockProjectBranchesClient, *fake.MockNewCodePeriodsClient, *fake.MockQualityGatesClient, *fake.MockQualityProfilesClient, *fake.MockProjectTagsClient) {
	return &fake.MockProjectsClient{},
		&fake.MockProjectLinksClient{},
		&fake.MockProjectBranchesClient{},
		&fake.MockNewCodePeriodsClient{},
		&fake.MockQualityGatesClient{},
		&fake.MockQualityProfilesClient{},
		&fake.MockProjectTagsClient{}
}

// successfulObserveMocks returns mock clients for successful observe tests.
func successfulObserveMocks() (*fake.MockProjectsClient, *fake.MockProjectLinksClient, *fake.MockProjectBranchesClient, *fake.MockNewCodePeriodsClient, *fake.MockQualityGatesClient, *fake.MockQualityProfilesClient, *fake.MockProjectTagsClient) {
	projectsClient := &fake.MockProjectsClient{
		SearchFn: func(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
			return &sonar.ProjectsSearch{
				Components: []sonar.ProjectSearchComponent{
					{Key: "test-key", Name: "test-project", Visibility: projectVisibilityPublic, Qualifier: "TRK"},
				},
			}, mockHTTPResponse(), nil
		},
	}
	branchesClient := &fake.MockProjectBranchesClient{
		ListFn: func(opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error) {
			return &sonar.ProjectBranchesList{
				Branches: []sonar.Branch{
					{Name: "main", IsMain: true, Type: "LONG"},
				},
			}, mockHTTPResponse(), nil
		},
	}
	linksClient := &fake.MockProjectLinksClient{
		SearchFn: func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
			return &sonar.ProjectLinksSearch{Links: []sonar.ProjectLink{}}, mockHTTPResponse(), nil
		},
	}
	ncpClient := &fake.MockNewCodePeriodsClient{
		ShowFn: func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
			return &sonar.NewCodePeriodsShow{Type: "PREVIOUS_VERSION", Inherited: true}, mockHTTPResponse(), nil
		},
		ListFn: func(opt *sonar.NewCodePeriodsListOptions) (*sonar.NewCodePeriodsList, *http.Response, error) {
			return &sonar.NewCodePeriodsList{NewCodePeriods: []sonar.NewCodePeriod{}}, mockHTTPResponse(), nil
		},
	}
	qgClient := &fake.MockQualityGatesClient{
		GetByProjectFn: func(opt *sonar.QualitygatesGetByProjectOptions) (*sonar.QualitygatesGetByProject, *http.Response, error) {
			return &sonar.QualitygatesGetByProject{
				QualityGate: sonar.ProjectQualityGate{Name: "Sonar way"},
			}, mockHTTPResponse(), nil
		},
	}
	qpClient := &fake.MockQualityProfilesClient{
		SearchFn: func(opt *sonar.QualityprofilesSearchOptions) (*sonar.QualityprofilesSearch, *http.Response, error) {
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

// TestObserve tests the Observe method.
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
				p.SearchFn = func(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
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
				p.SearchFn = func(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
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
				p.SearchFn = func(opt *sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
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
					Visibility: new("public"),
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		"ObserveWithBranchListError": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				b.ListFn = func(opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error) {
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
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
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
				n.ShowFn = func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
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
				q.GetByProjectFn = func(opt *sonar.QualitygatesGetByProjectOptions) (*sonar.QualitygatesGetByProject, *http.Response, error) {
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
				qp.SearchFn = func(opt *sonar.QualityprofilesSearchOptions) (*sonar.QualityprofilesSearch, *http.Response, error) {
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
				n.ListFn = func(opt *sonar.NewCodePeriodsListOptions) (*sonar.NewCodePeriodsList, *http.Response, error) {
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
		// Regression: when spec has no qualityProfiles but SonarQube returns default
		// built-in profiles for all languages, the reconciler must not set
		// ResourceUpToDate=false (which was causing an infinite update loop).
		"SpecWithNoQualityProfiles_SonarReturnsDefaults_IsUpToDate": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				// Override QP search to return many default profiles (like the real cluster)
				qp.SearchFn = func(opt *sonar.QualityprofilesSearchOptions) (*sonar.QualityprofilesSearch, *http.Response, error) {
					return &sonar.QualityprofilesSearch{
						Profiles: []sonar.QualityProfile{
							{Key: "uuid-1", Name: "Sonar way", Language: "java"},
							{Key: "uuid-2", Name: "Sonar way", Language: "go"},
							{Key: "uuid-3", Name: "Sonar way", Language: "ts"},
							{Key: "uuid-4", Name: "Sonar way", Language: "py"},
						},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: new("public"),
					// No qualityProfiles field - user did not specify any
				}),
			},
			want: want{
				// Must be up-to-date; the controller must not trigger an update.
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		// Regression: links were stored without the Name field in the observation,
		// so IsProjectLinkUpToDate always returned false (infinite update loop).
		"SpecWithLinks_ObservationHasMatchingLinks_IsUpToDate": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return &sonar.ProjectLinksSearch{
						Links: []sonar.ProjectLink{
							{ID: "link-uuid-1", Name: "GitHub Repository", Type: "", URL: "https://github.com/crossplane-contrib/provider-sonarqube"},
						},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: new("public"),
					Links: []v1alpha1.ProjectLinkParameters{
						{ID: new("link-uuid-1"), Name: "GitHub Repository", URL: "https://github.com/crossplane-contrib/provider-sonarqube"},
					},
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		// Verifies the opposite: a link URL change is detected and triggers an update.
		"SpecWithLinks_URLChanged_IsNotUpToDate": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return &sonar.ProjectLinksSearch{
						Links: []sonar.ProjectLink{
							{ID: "link-uuid-1", Name: "GitHub Repository", URL: "https://github.com/old-url"},
						},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: new("public"),
					Links: []v1alpha1.ProjectLinkParameters{
						{ID: new("link-uuid-1"), Name: "GitHub Repository", URL: "https://github.com/new-url"},
					},
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
			},
		},
		// Late-initialization: spec.Visibility is nil but SonarQube reports "public".
		// LateInitializeProject should fill in the visibility, so ResourceLateInitialized
		// must be true on the first reconcile pass.
		"VisibilityNilInSpec_LateInitializesFromObservation": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name: "test-project",
					Key:  "test-key",
					// Visibility intentionally omitted: should be late-initialized to "public"
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: true,
				},
			},
		},
		// Late-initialization: a link in the spec has no ID, but the observed state
		// has the matching link with an ID.  LateInitializeProjectLinks should
		// back-fill the ID, so ResourceLateInitialized must be true.
		"LinkWithNoID_LateInitializesIDFromObservation": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return &sonar.ProjectLinksSearch{
						Links: []sonar.ProjectLink{
							{ID: "link-uuid-1", Name: "GitHub Repository", Type: "", URL: "https://github.com/crossplane-contrib/provider-sonarqube"},
						},
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: new("public"),
					Links: []v1alpha1.ProjectLinkParameters{
						// ID is nil: should be back-filled by LateInitializeProjectLinks
						{Name: "GitHub Repository", URL: "https://github.com/crossplane-contrib/provider-sonarqube"},
					},
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: true,
				},
			},
		},
		// Regression: when spec.NewCodePeriod.Value is nil but SonarQube returns a value,
		// LateInitializeProjectNewCodePeriod must back-fill it so ResourceLateInitialized is
		// true on the first reconcile.  Previously, missing nil-guard + MinLength=1 caused
		// either a panic or a CRD validation error that triggered an infinite error loop.
		"SpecWithNewCodePeriod_ValueNil_LateInitializesFromObservation": {
			ext: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				// Override NCP Show to return a non-empty Value so that back-fill is exercised.
				n.ShowFn = func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
					return &sonar.NewCodePeriodsShow{
						Type:  "NUMBER_OF_DAYS",
						Value: "30",
					}, mockHTTPResponse(), nil
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			}(),
			args: args{
				ctx: context.Background(),
				mg: newTestProject("test-key", v1alpha1.ProjectParameters{
					Name:       "test-project",
					Key:        "test-key",
					Visibility: new("public"),
					NewCodePeriod: &v1alpha1.ProjectNewCodePeriodParameters{
						Type: "NUMBER_OF_DAYS",
						// Value intentionally nil: should be back-filled to "30"
					},
				}),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: true,
				},
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

// TestCreate tests the Create method.
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
				p.CreateFn = func(opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error) {
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
				p.CreateFn = func(opt *sonar.ProjectsCreateOptions) (*sonar.ProjectsCreate, *http.Response, error) {
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

// TestUpdate tests the Update method.
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
				p.UpdateVisibilityFn = func(opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error) {
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
						Visibility: new("private"),
					})
					proj.Status.AtProvider.Visibility = projectVisibilityPublic

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
				tg.SetFn = func(opt *sonar.ProjectTagsSetOptions) (*http.Response, error) {
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
				p.UpdateVisibilityFn = func(opt *sonar.ProjectsUpdateVisibilityOptions) (*http.Response, error) {
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
						Visibility: new("private"),
					})
					proj.Status.AtProvider.Visibility = projectVisibilityPublic

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
				b.SetMainFn = func(opt *sonar.ProjectBranchesSetMainOptions) (*http.Response, error) {
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
						DefaultBranch: new("develop"),
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
				q.SelectFn = func(opt *sonar.QualitygatesSelectOptions) (*http.Response, error) {
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
						QualityGateName: new("new-gate"),
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
				n.SetFn = func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
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
							Value: new("30"),
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
				l.CreateFn = func(opt *sonar.ProjectLinksCreateOptions) (*sonar.ProjectLinksCreate, *http.Response, error) {
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
				qp.ShowFn = func(opt *sonar.QualityprofilesShowOptions) (*sonar.QualityprofilesShow, *http.Response, error) {
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
							"java": {Id: new("new-profile-id")},
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
		// Regression: previously panicked with nil pointer dereference when
		// qualityProfile.Id was nil (reference not yet resolved) and the
		// language was missing from the observation. Must return a descriptive
		// error instead of crashing.
		"UpdateQualityProfile_NilId_ReturnsErrorNotPanic": {
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
						QualityProfiles: map[string]v1alpha1.ProjectQualityProfileReference{
							// Id is nil: simulates an unresolved IdRef.
							"go": {Id: nil},
						},
					})
					// No observation for "go" → AreProjectQualityProfilesUpToDate
					// returns false, triggering the update path.
					proj.Status.AtProvider.QualityProfiles = map[string]v1alpha1.ProjectQualityProfileObservation{}

					return proj
				}(),
			},
			want: want{
				update:    managed.ExternalUpdate{},
				errSubstr: "not yet resolved",
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

// TestDelete tests the Delete method.
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
				p.DeleteFn = func(opt *sonar.ProjectsDeleteOptions) (*http.Response, error) {
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
				p.DeleteFn = func(opt *sonar.ProjectsDeleteOptions) (*http.Response, error) {
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

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	p, l, b, n, q, qp, tg := defaultMockClients()
	ext := newTestExternalClient(p, l, b, n, q, qp, tg)

	err := ext.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnectTypeAssertion verifies that Connect returns errNotProject when
// the managed resource is not a *v1alpha1.Project (regression test for the
// copy-paste bug where the connector asserted *v1alpha1.QualityGate).
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	c := &connector{}

	// Passing a non-Project type must return errNotProject immediately.
	_, err := c.Connect(context.Background(), &notProject{})
	if err == nil {
		t.Fatal("Connect() expected error for non-Project type, got nil")
	}

	if !strings.Contains(err.Error(), errNotProject) {
		t.Errorf("Connect() error: want %q, got %q", errNotProject, err.Error())
	}
}

// TestApplyObserveResultNilMapInit tests that applyObserveResult always
// initialises Branches and Links to at least an empty map, preventing the
// "Required value" Kubernetes validation errors that occur when nil maps are
// serialised as JSON null.
func TestApplyObserveResultNilMapInit(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		startObs  v1alpha1.ProjectObservation
		result    observeResult
		wantLinks map[string]v1alpha1.ProjectLinkObservation
		wantBrs   map[string]v1alpha1.ProjectBranchObservation
	}{
		"BothNilResultInitialisesToEmptyMaps": {
			startObs:  v1alpha1.ProjectObservation{},
			result:    observeResult{},
			wantLinks: map[string]v1alpha1.ProjectLinkObservation{},
			wantBrs:   map[string]v1alpha1.ProjectBranchObservation{},
		},
		"ExistingMapsArePreservedWhenResultIsNil": {
			startObs: v1alpha1.ProjectObservation{
				Branches: map[string]v1alpha1.ProjectBranchObservation{"main": {Name: "main"}},
				Links:    map[string]v1alpha1.ProjectLinkObservation{"home": {Name: "home"}},
			},
			result:    observeResult{},
			wantLinks: map[string]v1alpha1.ProjectLinkObservation{"home": {Name: "home"}},
			wantBrs:   map[string]v1alpha1.ProjectBranchObservation{"main": {Name: "main"}},
		},
		"ResultBranchesOverwriteObservation": {
			startObs: v1alpha1.ProjectObservation{},
			result: observeResult{
				branches: map[string]v1alpha1.ProjectBranchObservation{
					"main": {Name: "main", IsMain: true, Type: "LONG"},
				},
				mainBranch: "main",
			},
			wantLinks: map[string]v1alpha1.ProjectLinkObservation{},
			wantBrs:   map[string]v1alpha1.ProjectBranchObservation{"main": {Name: "main", IsMain: true, Type: "LONG"}},
		},
		"ResultLinksOverwriteObservation": {
			startObs: v1alpha1.ProjectObservation{},
			result: observeResult{
				links: map[string]v1alpha1.ProjectLinkObservation{
					"homepage": {ID: "1", Name: "homepage", URL: "https://example.com"},
				},
			},
			wantLinks: map[string]v1alpha1.ProjectLinkObservation{"homepage": {ID: "1", Name: "homepage", URL: "https://example.com"}},
			wantBrs:   map[string]v1alpha1.ProjectBranchObservation{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, l, b, n, q, qp, tg := defaultMockClients()
			ext := newTestExternalClient(p, l, b, n, q, qp, tg)

			obs := tc.startObs
			// branchNewCodePeriods must always be initialised (mimics observeProjectDetails).
			tc.result.branchNewCodePeriods = make(map[string]v1alpha1.ProjectNewCodePeriodObservation)

			ext.applyObserveResult(&obs, &tc.result)

			if obs.Branches == nil {
				t.Error("applyObserveResult() left Branches as nil; expected non-nil map")
			}

			if obs.Links == nil {
				t.Error("applyObserveResult() left Links as nil; expected non-nil map")
			}

			if diff := cmp.Diff(tc.wantBrs, obs.Branches); diff != "" {
				t.Errorf("Branches mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantLinks, obs.Links); diff != "" {
				t.Errorf("Links mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObserveNilMapsNeverOccur verifies that after Observe completes (even with
// partial API failures), the managed resource's AtProvider.Branches and
// AtProvider.Links are never nil, preventing the CRD required-field validation
// failure.
func TestObserveNilMapsNeverOccur(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		buildExt func() *external
		name     string
	}{
		"BranchAPIFailure": {
			name: "BranchAPIFailure",
			buildExt: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				b.ListFn = func(opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error) {
					return nil, nil, errors.New("branch api down")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			},
		},
		"LinksAPIFailure": {
			name: "LinksAPIFailure",
			buildExt: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return nil, nil, errors.New("links api down")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			},
		},
		"BothBranchesAndLinksAPIFailure": {
			name: "BothBranchesAndLinksAPIFailure",
			buildExt: func() *external {
				p, l, b, n, q, qp, tg := successfulObserveMocks()
				b.ListFn = func(opt *sonar.ProjectBranchesListOptions) (*sonar.ProjectBranchesList, *http.Response, error) {
					return nil, nil, errors.New("branch api down")
				}
				l.SearchFn = func(opt *sonar.ProjectLinksSearchOptions) (*sonar.ProjectLinksSearch, *http.Response, error) {
					return nil, nil, errors.New("links api down")
				}

				return newTestExternalClient(p, l, b, n, q, qp, tg)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			proj := newTestProject("test-key", v1alpha1.ProjectParameters{
				Name: "test-project",
				Key:  "test-key",
			})

			_, _ = tc.buildExt().Observe(context.Background(), proj)

			if proj.Status.AtProvider.Branches == nil {
				t.Errorf("[%s] AtProvider.Branches is nil after Observe; CRD validation would fail", name)
			}

			if proj.Status.AtProvider.Links == nil {
				t.Errorf("[%s] AtProvider.Links is nil after Observe; CRD validation would fail", name)
			}
		})
	}
}
