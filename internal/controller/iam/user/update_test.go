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

package user

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
)

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("invalid managed resource", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Update(context.Background(), &notUser{})
		if err == nil || !strings.Contains(err.Error(), errNotUser) {
			t.Fatalf("Update() error = %v, want %q", err, errNotUser)
		}
	})

	t.Run("missing external name", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Update(context.Background(), newUserWithSpec(v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice"}))
		if err == nil || !strings.Contains(err.Error(), "external name is not set") {
			t.Fatalf("Update() error = %v", err)
		}
	})

	t.Run("updates fields and group memberships", func(t *testing.T) {
		t.Parallel()

		desiredGroups := []v1alpha1.UserGroupsParameters{{GroupId: ptr.To("ops")}}
		user := newUserWithSpec(v1alpha1.UserParameters{
			Login:  testUserLogin,
			Name:   "Alice Updated",
			Groups: &desiredGroups,
		})
		user.Status.AtProvider.Groups = map[string]string{"devs": "membership-devs"}
		meta.SetExternalName(user, testUserID)

		createdMemberships := []string{}
		deletedMemberships := []string{}

		updated, err := (&external{
			usersClient: &sonarfake.MockUsersClient{UpdateFn: func(userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
				if userID != testUserID || opt.Name != "Alice Updated" {
					t.Fatalf("Update() got id=%q opt=%+v", userID, opt)
				}

				return &sonar.UserV2{Id: testUserID, Login: testUserLogin, Name: "Alice Updated", Active: true}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			}},
			groupsClient: &sonarfake.MockGroupsClient{
				CreateGroupMembershipFn: func(opt *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error) {
					createdMemberships = append(createdMemberships, opt.GroupId)

					return &sonar.GroupMembership{Id: "membership-ops", GroupId: "ops", UserId: testUserID}, &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))}, nil
				},

				DeleteGroupMembershipFn: func(membershipID string) (*http.Response, error) {
					deletedMemberships = append(deletedMemberships, membershipID)

					return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
				},
			},
		}).Update(context.Background(), user)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if diff := cmp.Diff([]string{"ops"}, createdMemberships); diff != "" {
			t.Fatalf("Update() created memberships mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]string{"membership-devs"}, deletedMemberships); diff != "" {
			t.Fatalf("Update() deleted memberships mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(map[string]string{"devs": "membership-devs"}, user.Status.AtProvider.Groups); diff != "" {
			t.Fatalf("Update() should not mutate status groups (-want +got):\n%s", diff)
		}

		if len(updated.ConnectionDetails) != 0 {
			t.Fatalf("Update() connection details = %v, want empty", updated.ConnectionDetails)
		}
	})
}

// TestBuildGroupsDiff tests the buildGroupsDiff function.
func TestBuildGroupsDiff(t *testing.T) {
	t.Parallel()

	toAdd, toRemove := buildGroupsDiff([]string{"g1", "g2", "g1"}, map[string]string{"g2": "m2", "g3": "m3"})
	if diff := cmp.Diff([]string{"g1"}, toAdd); diff != "" {
		t.Fatalf("buildGroupsDiff() toAdd mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"m3"}, toRemove); diff != "" {
		t.Fatalf("buildGroupsDiff() toRemove mismatch (-want +got):\n%s", diff)
	}
}

// TestDesiredGroupIDs tests the desiredGroupIDs function.
func TestDesiredGroupIDs(t *testing.T) {
	t.Parallel()

	if desiredGroupIDs(nil) != nil {
		t.Fatal("desiredGroupIDs(nil) must return nil")
	}

	groups := []v1alpha1.UserGroupsParameters{{GroupId: nil}, {GroupId: ptr.To("")}, {GroupId: ptr.To("devs")}}
	if diff := cmp.Diff([]string{"devs"}, desiredGroupIDs(&groups)); diff != "" {
		t.Fatalf("desiredGroupIDs() mismatch (-want +got):\n%s", diff)
	}
}

// TestUpdateAggregatesFieldAndGroupErrors tests Update aggregates errors.
func TestUpdateAggregatesFieldAndGroupErrors(t *testing.T) {
	t.Parallel()

	desiredGroups := []v1alpha1.UserGroupsParameters{{GroupId: ptr.To("ops")}}
	user := newUserWithSpec(v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice", Groups: &desiredGroups})
	meta.SetExternalName(user, testUserID)

	updated, err := (&external{
		usersClient: &sonarfake.MockUsersClient{UpdateFn: func(_ string, _ *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(""))}, errors.New("update failed")
		}},
		groupsClient: &sonarfake.MockGroupsClient{CreateGroupMembershipFn: func(_ *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(""))}, errors.New("group update failed")
		}},
	}).Update(context.Background(), user)
	if err == nil {
		t.Fatal("Update() expected error, got nil")
	}

	for _, expected := range []string{"cannot update User", "cannot add User to Group ops"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("Update() error = %v, missing %q", err, expected)
		}
	}

	if len(updated.ConnectionDetails) != 0 {
		t.Fatalf("Update() connection details should be empty: %v", updated.ConnectionDetails)
	}
}
