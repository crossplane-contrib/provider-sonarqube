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

package qualityprofileusergroupassociation

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

const (
	// testQualityProfile is the quality profile name used across tests.
	testQualityProfile = "Sonar way"
	// testLanguage is the quality profile language used across tests.
	testLanguage = "go"
	// testGroupName is the group name used across tests.
	testGroupName = "sonar-users"
	// testLogin is the user login used across tests.
	testLogin = "alice"
)

// notQualityProfileUsergroupAssociation is a type for testing
// non-association resources.
type notQualityProfileUsergroupAssociation struct {
	resource.Managed
}

// fakeQualityProfilesClient is an in-test fake of
// qualityProfilesAssociationClient.
type fakeQualityProfilesClient struct {
	addGroupFn     func(opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error)
	addUserFn      func(opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error)
	removeGroupFn  func(opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error)
	removeUserFn   func(opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error)
	searchGroupsFn func(opt *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error)
	searchUsersFn  func(opt *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error)
}

// AddGroup implements qualityProfilesAssociationClient.AddGroup.
func (f *fakeQualityProfilesClient) AddGroup(_ context.Context, opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error) {
	if f.addGroupFn != nil {
		return f.addGroupFn(opt)
	}

	return mockHTTPResponse(), nil
}

// AddUser implements qualityProfilesAssociationClient.AddUser.
func (f *fakeQualityProfilesClient) AddUser(_ context.Context, opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error) {
	if f.addUserFn != nil {
		return f.addUserFn(opt)
	}

	return mockHTTPResponse(), nil
}

// RemoveGroup implements qualityProfilesAssociationClient.RemoveGroup.
func (f *fakeQualityProfilesClient) RemoveGroup(_ context.Context, opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error) {
	if f.removeGroupFn != nil {
		return f.removeGroupFn(opt)
	}

	return mockHTTPResponse(), nil
}

// RemoveUser implements qualityProfilesAssociationClient.RemoveUser.
func (f *fakeQualityProfilesClient) RemoveUser(_ context.Context, opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error) {
	if f.removeUserFn != nil {
		return f.removeUserFn(opt)
	}

	return mockHTTPResponse(), nil
}

// SearchGroups implements qualityProfilesAssociationClient.SearchGroups.
func (f *fakeQualityProfilesClient) SearchGroups(_ context.Context, opt *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
	if f.searchGroupsFn != nil {
		return f.searchGroupsFn(opt)
	}

	return &sonar.QualityprofilesSearchGroups{Paging: sonar.Paging{Total: 0, PageIndex: 1, PageSize: 100}}, mockHTTPResponse(), nil
}

// SearchUsers implements qualityProfilesAssociationClient.SearchUsers.
func (f *fakeQualityProfilesClient) SearchUsers(_ context.Context, opt *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error) {
	if f.searchUsersFn != nil {
		return f.searchUsersFn(opt)
	}

	return &sonar.QualityprofilesSearchUsers{Paging: sonar.Paging{Total: 0, PageIndex: 1, PageSize: 100}}, mockHTTPResponse(), nil
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       http.NoBody,
	}
}

// checkError asserts the error matches the expected substring.
func checkError(t *testing.T, method, wantErrSubstr string, gotErr error) {
	t.Helper()

	if wantErrSubstr == "" && gotErr == nil {
		return
	}

	if wantErrSubstr == "" && gotErr != nil {
		t.Errorf("%s() unexpected error: %v", method, gotErr)

		return
	}

	if wantErrSubstr != "" && gotErr == nil {
		t.Errorf("%s() expected error containing %q, got nil", method, wantErrSubstr)

		return
	}

	if !strings.Contains(gotErr.Error(), wantErrSubstr) {
		t.Errorf("%s() error = %q, want containing %q", method, gotErr.Error(), wantErrSubstr)
	}
}

// newTestGroupAssociation creates a test association for a group.
func newTestGroupAssociation(externalName, language, qualityProfile, groupName string) *v1alpha1.QualityProfileUsergroupAssociation {
	cr := &v1alpha1.QualityProfileUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-association",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.QualityProfileUsergroupAssociationSpec{
			ForProvider: v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: qualityProfile,
				Language:       language,
				GroupName:      new(groupName),
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}

	return cr
}

// newTestUserAssociation creates a test association for a user.
func newTestUserAssociation(externalName, language, qualityProfile, login string) *v1alpha1.QualityProfileUsergroupAssociation {
	cr := &v1alpha1.QualityProfileUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-association",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.QualityProfileUsergroupAssociationSpec{
			ForProvider: v1alpha1.QualityProfileUsergroupAssociationParameters{
				QualityProfile: qualityProfile,
				Language:       language,
				Login:          new(login),
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}

	return cr
}

// TestObserve tests observing a QualityProfileUsergroupAssociation.
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

	groupExternalName := instance.BuildQualityProfileUsergroupAssociationExternalName(&v1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: testQualityProfile,
		Language:       testLanguage,
		GroupName:      new(testGroupName),
	})
	userExternalName := instance.BuildQualityProfileUsergroupAssociationExternalName(&v1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: testQualityProfile,
		Language:       testLanguage,
		Login:          new(testLogin),
	})

	cases := map[string]struct {
		client *fakeQualityProfilesClient
		args   args
		want   want
	}{
		"NotAssociationType": {
			client: &fakeQualityProfilesClient{},
			args:   args{ctx: context.Background(), mg: &notQualityProfileUsergroupAssociation{}},
			want:   want{errSubstr: errNotQualityProfileUsergroupAssociation},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fakeQualityProfilesClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation("", testLanguage, testQualityProfile, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"UnparseableExternalNameReturnsNotExists": {
			client: &fakeQualityProfilesClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation("test-association", testLanguage, testQualityProfile, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"GroupSelectedExistsUpToDate": {
			client: &fakeQualityProfilesClient{
				searchGroupsFn: func(_ *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
					return &sonar.QualityprofilesSearchGroups{
						Groups: []sonar.QualityprofilesProfileGroup{{Name: testGroupName, Selected: true}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testLanguage, testQualityProfile, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}},
		},
		"GroupDeselectedReturnsNotExists": {
			client: &fakeQualityProfilesClient{
				searchGroupsFn: func(_ *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
					return &sonar.QualityprofilesSearchGroups{
						Groups: []sonar.QualityprofilesProfileGroup{{Name: testGroupName, Selected: false}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testLanguage, testQualityProfile, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"UserSelectedExistsUpToDate": {
			client: &fakeQualityProfilesClient{
				searchUsersFn: func(_ *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error) {
					return &sonar.QualityprofilesSearchUsers{
						Users:  []sonar.QualityprofilesProfileUser{{Login: testLogin, Selected: true}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserAssociation(userExternalName, testLanguage, testQualityProfile, testLogin),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}},
		},
		"UserDeselectedReturnsNotExists": {
			client: &fakeQualityProfilesClient{
				searchUsersFn: func(_ *sonar.QualityprofilesSearchUsersOptions) (*sonar.QualityprofilesSearchUsers, *http.Response, error) {
					return &sonar.QualityprofilesSearchUsers{
						Users:  []sonar.QualityprofilesProfileUser{{Login: testLogin, Selected: false}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserAssociation(userExternalName, testLanguage, testQualityProfile, testLogin),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"SearchErrorWrapped": {
			client: &fakeQualityProfilesClient{
				searchGroupsFn: func(_ *sonar.QualityprofilesSearchGroupsOptions) (*sonar.QualityprofilesSearchGroups, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testLanguage, testQualityProfile, testGroupName),
			},
			want: want{errSubstr: errObserveQualityProfileUsergroupAssociation},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

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

// TestCreate tests creating a QualityProfileUsergroupAssociation.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("NotAssociationType", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityProfilesClient{}}
		_, err := e.Create(context.Background(), &notQualityProfileUsergroupAssociation{})
		checkError(t, "Create", errNotQualityProfileUsergroupAssociation, err)
	})

	t.Run("GroupCallsAddGroupAndSetsExternalName", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualityprofilesAddGroupOptions
		cr := newTestGroupAssociation("", testLanguage, testQualityProfile, testGroupName)
		e := &external{client: &fakeQualityProfilesClient{
			addGroupFn: func(opt *sonar.QualityprofilesAddGroupOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityProfileAddGroupOptions(testLanguage, testQualityProfile, testGroupName)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Create() AddGroup options mismatch (-want +got):\n%s", diff)
		}

		wantName := "group:" + testGroupName + ":" + testLanguage + ":" + testQualityProfile
		if gotName := meta.GetExternalName(cr); gotName != wantName {
			t.Fatalf("Create() external name = %q, want %q", gotName, wantName)
		}
	})

	t.Run("UserCallsAddUserAndSetsExternalName", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualityprofilesAddUserOptions
		cr := newTestUserAssociation("", testLanguage, testQualityProfile, testLogin)
		e := &external{client: &fakeQualityProfilesClient{
			addUserFn: func(opt *sonar.QualityprofilesAddUserOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityProfileAddUserOptions(testLanguage, testQualityProfile, testLogin)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Create() AddUser options mismatch (-want +got):\n%s", diff)
		}

		wantName := "user:" + testLogin + ":" + testLanguage + ":" + testQualityProfile
		if gotName := meta.GetExternalName(cr); gotName != wantName {
			t.Fatalf("Create() external name = %q, want %q", gotName, wantName)
		}
	})
}

// TestDelete tests deleting a QualityProfileUsergroupAssociation.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("NotAssociationType", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityProfilesClient{}}
		_, err := e.Delete(context.Background(), &notQualityProfileUsergroupAssociation{})
		checkError(t, "Delete", errNotQualityProfileUsergroupAssociation, err)
	})

	t.Run("EmptyExternalNameSuccess", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityProfilesClient{}}
		_, err := e.Delete(context.Background(), newTestGroupAssociation("", testLanguage, testQualityProfile, testGroupName))
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}
	})

	t.Run("GroupCallsRemoveGroup", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualityprofilesRemoveGroupOptions
		cr := newTestGroupAssociation("group:"+testGroupName+":"+testLanguage+":"+testQualityProfile, testLanguage, testQualityProfile, testGroupName)
		e := &external{client: &fakeQualityProfilesClient{
			removeGroupFn: func(opt *sonar.QualityprofilesRemoveGroupOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityProfileRemoveGroupOptions(testLanguage, testQualityProfile, testGroupName)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Delete() RemoveGroup options mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UserCallsRemoveUser", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualityprofilesRemoveUserOptions
		cr := newTestUserAssociation("user:"+testLogin+":"+testLanguage+":"+testQualityProfile, testLanguage, testQualityProfile, testLogin)
		e := &external{client: &fakeQualityProfilesClient{
			removeUserFn: func(opt *sonar.QualityprofilesRemoveUserOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityProfileRemoveUserOptions(testLanguage, testQualityProfile, testLogin)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Delete() RemoveUser options mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestUpdate tests that Update is a no-op.
func TestUpdate(t *testing.T) {
	t.Parallel()

	e := &external{client: &fakeQualityProfilesClient{}}
	got, err := e.Update(context.Background(), newTestGroupAssociation("group:"+testGroupName+":"+testLanguage+":"+testQualityProfile, testLanguage, testQualityProfile, testGroupName))
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if diff := cmp.Diff(managed.ExternalUpdate{}, got); diff != "" {
		t.Errorf("Update() mismatch (-want +got):\n%s", diff)
	}
}
