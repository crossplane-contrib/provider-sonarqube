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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
)

//nolint:gocognit // Extensive scenario coverage for core update logic.
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

	t.Run("updates fields, password and group memberships", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()

		err := corev1.AddToScheme(scheme)
		if err != nil {
			t.Fatalf("AddToScheme(corev1) = %v", err)
		}

		kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pwd", Namespace: "default"}, Data: map[string][]byte{"password": []byte("new-password")}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "conn", Namespace: "default"}, Data: map[string][]byte{passwordKey: []byte("old-password")}},
		).Build()

		desiredGroups := []v1alpha1.UserGroupsParameters{{GroupId: ptr.To("ops")}}
		user := newUserWithSpec(v1alpha1.UserParameters{
			Login:             testUserLogin,
			Name:              "Alice Updated",
			Groups:            &desiredGroups,
			Local:             ptr.To(true),
			PasswordManaged:   ptr.To(true),
			PasswordSecretRef: &xpv1.SecretKeySelector{Key: "password", SecretReference: xpv1.SecretReference{Name: "pwd", Namespace: "default"}},
		})
		user.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{Name: "conn"}
		user.Status.AtProvider.Groups = map[string]string{"devs": "membership-devs"}
		meta.SetExternalName(user, testUserID)

		createdMemberships := []string{}
		deletedMemberships := []string{}
		passwordChangeCalls := 0

		updated, err := (&external{
			kube: kube,
			usersClient: &sonarfake.MockUsersClient{UpdateFn: func(userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
				if userID != testUserID || opt.Name != "Alice Updated" {
					t.Fatalf("Update() got id=%q opt=%+v", userID, opt)
				}

				return &sonar.UserV2{Id: testUserID, Login: testUserLogin, Name: "Alice Updated", Active: true}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			}},
			usersv1Client: &mockPasswordClient{changePasswordFn: func(opt *sonar.UsersChangePasswordOptions) (*http.Response, error) {
				passwordChangeCalls++

				if opt.Login != testUserLogin {
					t.Fatalf("ChangePassword() login = %q, want %s", opt.Login, testUserLogin)
				}

				if opt.PreviousPassword != "old-password" || opt.Password != "new-password" {
					t.Fatalf("ChangePassword() got %+v", opt)
				}

				return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
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

		if passwordChangeCalls != 1 {
			t.Fatalf("Update() password change calls = %d, want 1", passwordChangeCalls)
		}

		if diff := cmp.Diff([]string{"ops"}, createdMemberships); diff != "" {
			t.Fatalf("Update() created memberships mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]string{"membership-devs"}, deletedMemberships); diff != "" {
			t.Fatalf("Update() deleted memberships mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(map[string]string{"ops": "membership-ops"}, user.Status.AtProvider.Groups); diff != "" {
			t.Fatalf("Update() status groups mismatch (-want +got):\n%s", diff)
		}

		if string(updated.ConnectionDetails[passwordKey]) != "new-password" {
			t.Fatalf("Update() password connection detail missing: %v", updated.ConnectionDetails)
		}
	})
}

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

//nolint:wsl_v5 // Focused branch coverage for update/password edge-cases.
func TestUpdatePasswordAndFieldFailures(t *testing.T) {
	t.Parallel()

	user := newUserWithSpec(v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice", Local: ptr.To(true), PasswordManaged: ptr.To(true)})
	meta.SetExternalName(user, testUserID)

	_, err := (&external{usersClient: &sonarfake.MockUsersClient{UpdateFn: func(_ string, _ *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}, errors.New("update failed")
	}}}).Update(context.Background(), user)
	if err == nil || !strings.Contains(err.Error(), "cannot update User") {
		t.Fatalf("Update() field error = %v", err)
	}

	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) = %v", err)
	}
	kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pwd", Namespace: "default"}, Data: map[string][]byte{"password": []byte("same")}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "conn", Namespace: "default"}, Data: map[string][]byte{passwordKey: []byte("same")}},
	).Build()

	user = newUserWithSpec(v1alpha1.UserParameters{
		Login:             testUserLogin,
		Name:              "Alice",
		Local:             ptr.To(true),
		PasswordManaged:   ptr.To(true),
		PasswordSecretRef: &xpv1.SecretKeySelector{Key: "password", SecretReference: xpv1.SecretReference{Name: "pwd", Namespace: "default"}},
	})
	user.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{Name: "conn"}
	meta.SetExternalName(user, testUserID)

	called := false
	updated, err := (&external{
		kube: kube,
		usersClient: &sonarfake.MockUsersClient{UpdateFn: func(_ string, _ *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
			return &sonar.UserV2{Id: testUserID, Login: testUserLogin, Name: "Alice"}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		}},
		usersv1Client: &mockPasswordClient{changePasswordFn: func(_ *sonar.UsersChangePasswordOptions) (*http.Response, error) {
			called = true

			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
		}},
	}).Update(context.Background(), user)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if called {
		t.Fatal("Update() should not call ChangePassword when password is unchanged")
	}
	if len(updated.ConnectionDetails) != 0 {
		t.Fatalf("Update() connection details = %v, want empty", updated.ConnectionDetails)
	}
}
