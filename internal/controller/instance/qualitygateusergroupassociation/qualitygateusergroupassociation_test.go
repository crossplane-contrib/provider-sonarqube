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

package qualitygateusergroupassociation

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
	// testGateName is the quality gate name used across tests.
	testGateName = "Sonar way"
	// testGroupName is the group name used across tests.
	testGroupName = "sonar-users"
	// testLogin is the user login used across tests.
	testLogin = "alice"
)

// notQualityGateUsergroupAssociation is a type for testing non-association
// resources.
type notQualityGateUsergroupAssociation struct {
	resource.Managed
}

// fakeQualityGatesClient is an in-test fake of
// qualityGatesAssociationClient.
type fakeQualityGatesClient struct {
	addGroupFn     func(opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error)
	addUserFn      func(opt *sonar.QualitygatesAddUserOptions) (*http.Response, error)
	removeGroupFn  func(opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error)
	removeUserFn   func(opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error)
	searchGroupsFn func(opt *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error)
	searchUsersFn  func(opt *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error)
}

// AddGroup implements qualityGatesAssociationClient.AddGroup.
func (f *fakeQualityGatesClient) AddGroup(_ context.Context, opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error) {
	if f.addGroupFn != nil {
		return f.addGroupFn(opt)
	}

	return mockHTTPResponse(), nil
}

// AddUser implements qualityGatesAssociationClient.AddUser.
func (f *fakeQualityGatesClient) AddUser(_ context.Context, opt *sonar.QualitygatesAddUserOptions) (*http.Response, error) {
	if f.addUserFn != nil {
		return f.addUserFn(opt)
	}

	return mockHTTPResponse(), nil
}

// RemoveGroup implements qualityGatesAssociationClient.RemoveGroup.
func (f *fakeQualityGatesClient) RemoveGroup(_ context.Context, opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error) {
	if f.removeGroupFn != nil {
		return f.removeGroupFn(opt)
	}

	return mockHTTPResponse(), nil
}

// RemoveUser implements qualityGatesAssociationClient.RemoveUser.
func (f *fakeQualityGatesClient) RemoveUser(_ context.Context, opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error) {
	if f.removeUserFn != nil {
		return f.removeUserFn(opt)
	}

	return mockHTTPResponse(), nil
}

// SearchGroups implements qualityGatesAssociationClient.SearchGroups.
func (f *fakeQualityGatesClient) SearchGroups(_ context.Context, opt *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
	if f.searchGroupsFn != nil {
		return f.searchGroupsFn(opt)
	}

	return &sonar.QualitygatesSearchGroups{Paging: sonar.Paging{Total: 0, PageIndex: 1, PageSize: 100}}, mockHTTPResponse(), nil
}

// SearchUsers implements qualityGatesAssociationClient.SearchUsers.
func (f *fakeQualityGatesClient) SearchUsers(_ context.Context, opt *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error) {
	if f.searchUsersFn != nil {
		return f.searchUsersFn(opt)
	}

	return &sonar.QualitygatesSearchUsers{Paging: sonar.Paging{Total: 0, PageIndex: 1, PageSize: 100}}, mockHTTPResponse(), nil
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
func newTestGroupAssociation(externalName, gateName, groupName string) *v1alpha1.QualityGateUsergroupAssociation {
	cr := &v1alpha1.QualityGateUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-association",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.QualityGateUsergroupAssociationSpec{
			ForProvider: v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName:  gateName,
				GroupName: new(groupName),
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}

	return cr
}

// newTestUserAssociation creates a test association for a user.
func newTestUserAssociation(externalName, gateName, login string) *v1alpha1.QualityGateUsergroupAssociation {
	cr := &v1alpha1.QualityGateUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-association",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.QualityGateUsergroupAssociationSpec{
			ForProvider: v1alpha1.QualityGateUsergroupAssociationParameters{
				GateName: gateName,
				Login:    new(login),
			},
		},
	}

	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}

	return cr
}

// TestObserve tests observing a QualityGateUsergroupAssociation.
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

	groupExternalName := instance.BuildQualityGateUsergroupAssociationExternalName(&v1alpha1.QualityGateUsergroupAssociationParameters{
		GateName:  testGateName,
		GroupName: new(testGroupName),
	})
	userExternalName := instance.BuildQualityGateUsergroupAssociationExternalName(&v1alpha1.QualityGateUsergroupAssociationParameters{
		GateName: testGateName,
		Login:    new(testLogin),
	})

	cases := map[string]struct {
		client *fakeQualityGatesClient
		args   args
		want   want
	}{
		"NotAssociationType": {
			client: &fakeQualityGatesClient{},
			args:   args{ctx: context.Background(), mg: &notQualityGateUsergroupAssociation{}},
			want:   want{errSubstr: errNotQualityGateUsergroupAssociation},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fakeQualityGatesClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation("", testGateName, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"UnparseableExternalNameReturnsNotExists": {
			client: &fakeQualityGatesClient{},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation("test-association", testGateName, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"GroupSelectedExistsUpToDate": {
			client: &fakeQualityGatesClient{
				searchGroupsFn: func(_ *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
					return &sonar.QualitygatesSearchGroups{
						Groups: []sonar.QualityGateGroup{{Name: testGroupName, Selected: true}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testGateName, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}},
		},
		"GroupDeselectedReturnsNotExists": {
			client: &fakeQualityGatesClient{
				searchGroupsFn: func(_ *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
					return &sonar.QualitygatesSearchGroups{
						Groups: []sonar.QualityGateGroup{{Name: testGroupName, Selected: false}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testGateName, testGroupName),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"UserSelectedExistsUpToDate": {
			client: &fakeQualityGatesClient{
				searchUsersFn: func(_ *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error) {
					return &sonar.QualitygatesSearchUsers{
						Users:  []sonar.QualityGateUser{{Login: testLogin, Selected: true}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserAssociation(userExternalName, testGateName, testLogin),
			},
			want: want{observation: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}},
		},
		"UserDeselectedReturnsNotExists": {
			client: &fakeQualityGatesClient{
				searchUsersFn: func(_ *sonar.QualitygatesSearchUsersOptions) (*sonar.QualitygatesSearchUsers, *http.Response, error) {
					return &sonar.QualitygatesSearchUsers{
						Users:  []sonar.QualityGateUser{{Login: testLogin, Selected: false}},
						Paging: sonar.Paging{Total: 1, PageIndex: 1, PageSize: 100},
					}, mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestUserAssociation(userExternalName, testGateName, testLogin),
			},
			want: want{observation: managed.ExternalObservation{ResourceExists: false}},
		},
		"SearchErrorWrapped": {
			client: &fakeQualityGatesClient{
				searchGroupsFn: func(_ *sonar.QualitygatesSearchGroupsOptions) (*sonar.QualitygatesSearchGroups, *http.Response, error) {
					return nil, mockHTTPResponse(), errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  newTestGroupAssociation(groupExternalName, testGateName, testGroupName),
			},
			want: want{errSubstr: errObserveQualityGateUsergroupAssociation},
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

// TestCreate tests creating a QualityGateUsergroupAssociation.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("NotAssociationType", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityGatesClient{}}
		_, err := e.Create(context.Background(), &notQualityGateUsergroupAssociation{})
		checkError(t, "Create", errNotQualityGateUsergroupAssociation, err)
	})

	t.Run("GroupCallsAddGroupAndSetsExternalName", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualitygatesAddGroupOptions
		cr := newTestGroupAssociation("", testGateName, testGroupName)
		e := &external{client: &fakeQualityGatesClient{
			addGroupFn: func(opt *sonar.QualitygatesAddGroupOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityGateAddGroupOptions(testGateName, testGroupName)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Create() AddGroup options mismatch (-want +got):\n%s", diff)
		}

		wantName := "group:" + testGroupName + ":" + testGateName
		if gotName := meta.GetExternalName(cr); gotName != wantName {
			t.Fatalf("Create() external name = %q, want %q", gotName, wantName)
		}
	})

	t.Run("UserCallsAddUserAndSetsExternalName", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualitygatesAddUserOptions
		cr := newTestUserAssociation("", testGateName, testLogin)
		e := &external{client: &fakeQualityGatesClient{
			addUserFn: func(opt *sonar.QualitygatesAddUserOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityGateAddUserOptions(testGateName, testLogin)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Create() AddUser options mismatch (-want +got):\n%s", diff)
		}

		wantName := "user:" + testLogin + ":" + testGateName
		if gotName := meta.GetExternalName(cr); gotName != wantName {
			t.Fatalf("Create() external name = %q, want %q", gotName, wantName)
		}
	})
}

// TestDelete tests deleting a QualityGateUsergroupAssociation.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("NotAssociationType", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityGatesClient{}}
		_, err := e.Delete(context.Background(), &notQualityGateUsergroupAssociation{})
		checkError(t, "Delete", errNotQualityGateUsergroupAssociation, err)
	})

	t.Run("EmptyExternalNameSuccess", func(t *testing.T) {
		t.Parallel()

		e := &external{client: &fakeQualityGatesClient{}}
		_, err := e.Delete(context.Background(), newTestGroupAssociation("", testGateName, testGroupName))
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}
	})

	t.Run("GroupCallsRemoveGroup", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualitygatesRemoveGroupOptions
		cr := newTestGroupAssociation("group:"+testGroupName+":"+testGateName, testGateName, testGroupName)
		e := &external{client: &fakeQualityGatesClient{
			removeGroupFn: func(opt *sonar.QualitygatesRemoveGroupOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityGateRemoveGroupOptions(testGateName, testGroupName)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Delete() RemoveGroup options mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UserCallsRemoveUser", func(t *testing.T) {
		t.Parallel()

		var gotOpts *sonar.QualitygatesRemoveUserOptions
		cr := newTestUserAssociation("user:"+testLogin+":"+testGateName, testGateName, testLogin)
		e := &external{client: &fakeQualityGatesClient{
			removeUserFn: func(opt *sonar.QualitygatesRemoveUserOptions) (*http.Response, error) {
				gotOpts = opt

				return mockHTTPResponse(), nil
			},
		}}

		_, err := e.Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		wantOpts := instance.GenerateQualityGateRemoveUserOptions(testGateName, testLogin)
		if diff := cmp.Diff(wantOpts, gotOpts); diff != "" {
			t.Errorf("Delete() RemoveUser options mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestUpdate tests that Update is a no-op.
func TestUpdate(t *testing.T) {
	t.Parallel()

	e := &external{client: &fakeQualityGatesClient{}}
	got, err := e.Update(context.Background(), newTestGroupAssociation("group:"+testGroupName+":"+testGateName, testGateName, testGroupName))
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if diff := cmp.Diff(managed.ExternalUpdate{}, got); diff != "" {
		t.Errorf("Update() mismatch (-want +got):\n%s", diff)
	}
}
