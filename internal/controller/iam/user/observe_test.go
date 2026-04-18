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
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
)

func TestObserve(t *testing.T) {
	t.Parallel()

	t.Run("invalid managed resource", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Observe(context.Background(), &notUser{})
		if err == nil || !strings.Contains(err.Error(), errNotUser) {
			t.Fatalf("Observe() error = %v, want %q", err, errNotUser)
		}
	})

	t.Run("missing external name", func(t *testing.T) {
		t.Parallel()

		obs, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Observe(context.Background(), &v1alpha1.User{})
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}

		if obs.ResourceExists {
			t.Fatalf("Observe() ResourceExists = %v, want false", obs.ResourceExists)
		}
	})

	t.Run("fetch not found returns non-existing", func(t *testing.T) {
		t.Parallel()

		user := newUserWithSpec(v1alpha1.UserParameters{Login: "alice", Name: "Alice"})
		meta.SetExternalName(user, testUserID)

		obs, err := (&external{usersClient: &sonarfake.MockUsersClient{FetchFn: func(_ string) (*sonar.UserV2, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, errors.New("not found")
		}}}).Observe(context.Background(), user)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}

		if obs.ResourceExists {
			t.Fatalf("Observe() ResourceExists = %v, want false", obs.ResourceExists)
		}
	})

	t.Run("existing user observes memberships", func(t *testing.T) {
		t.Parallel()

		login := testUserLogin
		email := "alice@example.com"
		local := true
		user := newUserWithSpec(v1alpha1.UserParameters{Login: login, Name: "Alice", Email: &email, Local: &local})
		meta.SetExternalName(user, testUserID)

		usersClient := &sonarfake.MockUsersClient{
			FetchFn: func(userID string) (*sonar.UserV2, *http.Response, error) {
				if userID != testUserID {
					t.Fatalf("Fetch() userID = %q, want %s", userID, testUserID)
				}

				return &sonar.UserV2{Id: testUserID, Login: login, Name: "Alice", Email: email, Local: true, Active: true}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			},
		}
		groupsClient := &sonarfake.MockGroupsClient{
			SearchGroupMembershipsFn: func(opt *sonar.AuthorizationsSearchGroupMembershipsOptions) (*sonar.AuthorizationsGroupMembershipsSearch, *http.Response, error) {
				if opt.UserId != testUserID {
					t.Fatalf("SearchGroupMemberships() UserId = %q, want %s", opt.UserId, testUserID)
				}

				return &sonar.AuthorizationsGroupMembershipsSearch{GroupMemberships: []sonar.GroupMembership{{GroupId: "devs", Id: "m-devs", UserId: testUserID}, {GroupId: "ops", Id: "m-ops", UserId: testUserID}}}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			},
		}

		obs, err := (&external{usersClient: usersClient, groupsClient: groupsClient}).Observe(context.Background(), user)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}

		if !obs.ResourceExists || !obs.ResourceUpToDate {
			t.Fatalf("Observe() = %+v, want exists and up-to-date", obs)
		}

		if diff := cmp.Diff(map[string]string{"devs": "m-devs", "ops": "m-ops"}, user.Status.AtProvider.Groups); diff != "" {
			t.Fatalf("Observe() groups mismatch (-want +got):\n%s", diff)
		}

		if user.Spec.ForProvider.Email == nil || *user.Spec.ForProvider.Email != email {
			t.Fatalf("Observe() late init email = %v", user.Spec.ForProvider.Email)
		}
	})
}

func TestGetSavedPasswordAndMembershipObservationErrors(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) = %v", err)
	}

	kube := fakekube.NewClientBuilder().WithScheme(scheme).Build()
	user := newUserWithSpec(v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice"})
	user.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{Name: "missing"}

	savedPassword, err := (&external{kube: kube}).getSavedPassword(context.Background(), user)
	if err != nil {
		t.Fatalf("getSavedPassword() unexpected error: %v", err)
	}

	if savedPassword != "" {
		t.Fatalf("getSavedPassword() = %q, want empty", savedPassword)
	}

	_, err = (&external{groupsClient: &sonarfake.MockGroupsClient{SearchGroupMembershipsFn: func(_ *sonar.AuthorizationsSearchGroupMembershipsOptions) (*sonar.AuthorizationsGroupMembershipsSearch, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}, errors.New("boom")
	}}}).getUserGroupsObservation(testUserID)
	if err == nil || !strings.Contains(err.Error(), "cannot fetch user groups") {
		t.Fatalf("getUserGroupsObservation() error = %v", err)
	}
}
