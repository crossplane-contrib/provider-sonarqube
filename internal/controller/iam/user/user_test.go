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
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
)

const (
	// testUserID is the test user identifier.
	testUserID = "user-1"
	// testUserLogin is the test user login name.
	testUserLogin = "alice"
)

// notUser is a type for testing non-User resources.
type notUser struct {
	resource.Managed
}

// mockGate is a mock implementation of a gate for testing.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register registers a callback for a GroupVersionKind.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set sets whether a GroupVersionKind is enabled.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// newUserWithSpec creates a new User with the given parameters.
func newUserWithSpec(spec v1alpha1.UserParameters) *v1alpha1.User {
	return &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: v1alpha1.UserSpec{ForProvider: spec},
	}
}

// TestCreate tests the Create method for users.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("invalid managed resource", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Create(context.Background(), &notUser{})
		if err == nil || !strings.Contains(err.Error(), errNotUser) {
			t.Fatalf("Create() error = %v, want %q", err, errNotUser)
		}
	})

	t.Run("local user without secret returns error", func(t *testing.T) {
		t.Parallel()

		user := newUserWithSpec(v1alpha1.UserParameters{Login: "alice", Name: "Alice", Local: ptr.To(true)})

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Create(context.Background(), user)
		if err == nil || !strings.Contains(err.Error(), "passwordSecretRef must be set") {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("creates user", func(t *testing.T) {
		t.Parallel()

		desiredGroups := []v1alpha1.UserGroupsParameters{{GroupId: ptr.To("devs")}, {GroupId: ptr.To("ops")}, {GroupId: nil}}
		user := newUserWithSpec(v1alpha1.UserParameters{
			Login:       testUserLogin,
			Name:        "Alice",
			Local:       ptr.To(false),
			Groups:      &desiredGroups,
			ScmAccounts: &[]string{"github:alice"},
		})

		createdMemberships := map[string]string{}

		created, err := (&external{
			usersClient: &sonarfake.MockUsersClient{CreateFn: func(opt *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error) {
				if opt.Login != testUserLogin || opt.Name != "Alice" {
					t.Fatalf("Create() got %+v", opt)
				}

				return &sonar.UserV2{Id: testUserID, Login: testUserLogin, Name: "Alice", Active: true}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			}},
			groupsClient: &sonarfake.MockGroupsClient{CreateGroupMembershipFn: func(opt *sonar.AuthorizationsCreateGroupMembershipOptions) (*sonar.GroupMembership, *http.Response, error) {
				createdMemberships[opt.GroupId] = "m-" + opt.GroupId

				return &sonar.GroupMembership{Id: "m-" + opt.GroupId, GroupId: opt.GroupId, UserId: opt.UserId}, &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))}, nil
			}},
		}).Create(context.Background(), user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if created.ConnectionDetails != nil {
			t.Fatalf("Create() connection details = %v, want empty", created.ConnectionDetails)
		}

		if meta.GetExternalName(user) != testUserID {
			t.Fatalf("Create() external name = %q, want %s", meta.GetExternalName(user), testUserID)
		}

		if user.Status.AtProvider.Groups != nil {
			t.Fatalf("Create() should not mutate status groups, got %v", user.Status.AtProvider.Groups)
		}

		if len(createdMemberships) != 0 {
			t.Fatalf("Create() memberships created = %d, want 0", len(createdMemberships))
		}
	})

	t.Run("local user writes connection details", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()

		err := corev1.AddToScheme(scheme)
		if err != nil {
			t.Fatalf("AddToScheme(corev1) = %v", err)
		}

		kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pwd", Namespace: "default"}, Data: map[string][]byte{"password": []byte("super-secret")}}).Build()
		user := newUserWithSpec(v1alpha1.UserParameters{
			Login: testUserLogin,
			Name:  "Alice",
			Local: ptr.To(true),
			PasswordSecretRef: &xpv1.SecretKeySelector{Key: "password", SecretReference: xpv1.SecretReference{
				Name:      "pwd",
				Namespace: "default",
			}},
		})

		created, err := (&external{
			kube: kube,
			usersClient: &sonarfake.MockUsersClient{CreateFn: func(_ *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error) {
				return &sonar.UserV2{Id: testUserID, Login: testUserLogin, Name: "Alice", Active: true}, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
			}},
			groupsClient: &sonarfake.MockGroupsClient{},
		}).Create(context.Background(), user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if string(created.ConnectionDetails[passwordKey]) != "super-secret" {
			t.Fatalf("Create() password connection detail missing: %v", created.ConnectionDetails)
		}
	})
}

// TestDelete tests the Delete method for users.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("invalid managed resource", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Delete(context.Background(), &notUser{})
		if err == nil || !strings.Contains(err.Error(), errNotUser) {
			t.Fatalf("Delete() error = %v, want %q", err, errNotUser)
		}
	})

	t.Run("missing external name is no-op", func(t *testing.T) {
		t.Parallel()

		_, err := (&external{usersClient: &sonarfake.MockUsersClient{}}).Delete(context.Background(), &v1alpha1.User{})
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("deactivates with anonymize flag", func(t *testing.T) {
		t.Parallel()

		called := false
		usersClient := &sonarfake.MockUsersClient{
			DeactivateFn: func(opt *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
				called = true

				if opt.Id != testUserID || !opt.Anonymize {
					t.Fatalf("Deactivate() got %+v", opt)
				}

				return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
			},
		}
		user := newUserWithSpec(v1alpha1.UserParameters{Anonymize: ptr.To(true)})
		meta.SetExternalName(user, testUserID)

		_, err := (&external{usersClient: usersClient}).Delete(context.Background(), user)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if !called {
			t.Fatal("Delete() did not call Deactivate")
		}
	})
}

// TestDeleteHandlesNotFoundAndErrors tests delete error handling.
func TestDeleteHandlesNotFoundAndErrors(t *testing.T) {
	t.Parallel()

	user := newUserWithSpec(v1alpha1.UserParameters{})
	meta.SetExternalName(user, testUserID)

	_, err := (&external{usersClient: &sonarfake.MockUsersClient{DeactivateFn: func(_ *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
		//nolint:nilnil // Intentional: simulating partial HTTP failure.
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, errors.New("not found")
	}}}).Delete(context.Background(), user)
	if err != nil {
		t.Fatalf("Delete() not-found should be ignored, got: %v", err)
	}

	_, err = (&external{usersClient: &sonarfake.MockUsersClient{DeactivateFn: func(_ *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
		//nolint:nilnil // Intentional: simulating partial HTTP failure.
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}, errors.New("boom")
	}}}).Delete(context.Background(), user)
	if err == nil || !strings.Contains(err.Error(), "cannot delete User") {
		t.Fatalf("Delete() error = %v", err)
	}
}

// TestSetupGatedCallbackPanicsWithoutManager tests callback behavior.
func TestSetupGatedCallbackPanicsWithoutManager(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}
	opts := controller.DefaultOptions()
	opts.Gate = gate

	err := SetupGated(nil, opts)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if gate.callback == nil {
		t.Fatal("SetupGated() expected callback")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("SetupGated callback expected to panic with nil manager")
		}
	}()

	gate.callback()
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	err := (&external{}).Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// TestSetupGatedRegistersUserGVK tests SetupGated registers the user GVK.
func TestSetupGatedRegistersUserGVK(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}
	opts := controller.DefaultOptions()
	opts.Gate = gate

	err := SetupGated(nil, opts)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if !gate.registered {
		t.Fatal("SetupGated() expected gate registration")
	}

	if gate.callback == nil {
		t.Fatal("SetupGated() expected callback")
	}

	if diff := cmp.Diff(v1alpha1.UserGroupVersionKind, gate.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() gvk mismatch (-want +got):\n%s", diff)
	}
}

// TestConnectTypeAssertion tests Connect type assertion error handling.
func TestConnectTypeAssertion(t *testing.T) {
	t.Parallel()

	_, err := (&connector{}).Connect(context.Background(), &notUser{})
	if err == nil || !strings.Contains(err.Error(), errNotUser) {
		t.Fatalf("Connect() error = %v, want %q", err, errNotUser)
	}
}

// TestConnectGetConfigError tests Connect config retrieval error handling.
func TestConnectGetConfigError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(provider) = %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(user) = %v", err)
	}

	kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).Build()

	user := &v1alpha1.User{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.UserGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.UserKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
			UID:       types.UID("user-uid"),
		},
		Spec: v1alpha1.UserSpec{ManagedResourceSpec: xpv2.ManagedResourceSpec{}, ForProvider: v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice"}},
	}
	user.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	_, err = (&connector{kube: kubeClient, usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{})}).Connect(context.Background(), user)
	if err == nil || !strings.Contains(err.Error(), errGetPC) {
		t.Fatalf("Connect() error = %v, want substring %q", err, errGetPC)
	}
}

// TestConnectSuccess tests successful Connect with valid configuration.
func TestConnectSuccess(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(provider) = %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(user) = %v", err)
	}

	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) = %v", err)
	}

	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000",
			Token: &apisv1alpha1.ProviderCredentials{Source: xpv1.CredentialsSourceSecret, CommonCredentialSelectors: xpv1.CommonCredentialSelectors{SecretRef: &xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "sonar-secret", Namespace: "default"},
				Key:             "token",
			}}},
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "sonar-secret", Namespace: "default"}, Data: map[string][]byte{"token": []byte("my-token")}}
	kubeClient := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig, secret).Build()

	user := &v1alpha1.User{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.UserGroupVersionKind.GroupVersion().String(), Kind: v1alpha1.UserKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
			UID:       types.UID("user-uid"),
		},
		Spec: v1alpha1.UserSpec{ManagedResourceSpec: xpv2.ManagedResourceSpec{}, ForProvider: v1alpha1.UserParameters{Login: testUserLogin, Name: "Alice"}},
	}
	user.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	got, err := (&connector{kube: kubeClient, usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{})}).Connect(context.Background(), user)
	if err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("Connect() expected external client")
	}
}
