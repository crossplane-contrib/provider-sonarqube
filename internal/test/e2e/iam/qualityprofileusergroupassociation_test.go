//go:build e2e

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

package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// qpAssocLanguage is the language of the Quality Profiles created by the
// QualityProfileUsergroupAssociation e2e tests.
const qpAssocLanguage = "go"

// newE2EQualityProfile builds a non-default QualityProfile CR.
func newE2EQualityProfile(f *e2e.Framework, crName, profileName string) *instancev1alpha1.QualityProfile {
	return &instancev1alpha1.QualityProfile{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.QualityProfileSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityProfileParameters{
				Name:     profileName,
				Language: qpAssocLanguage,
				Default:  ptr.To(false),
			},
		},
	}
}

// newE2EQualityProfileUsergroupAssociation builds a
// QualityProfileUsergroupAssociation CR with the given parameters.
func newE2EQualityProfileUsergroupAssociation(f *e2e.Framework, crName string, params iamv1alpha1.QualityProfileUsergroupAssociationParameters) *iamv1alpha1.QualityProfileUsergroupAssociation {
	return &iamv1alpha1.QualityProfileUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: iamv1alpha1.QualityProfileUsergroupAssociationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: params,
		},
	}
}

// waitForQualityProfileSelected polls check until it reports want or the
// timeout elapses.
func waitForQualityProfileSelected(t *testing.T, want bool, desc string, check func(context.Context) (bool, error)) {
	t.Helper()

	var got bool
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var checkErr error
		got, checkErr = check(ctx)
		if checkErr != nil {
			return false, checkErr
		}
		return got == want, nil
	}); err != nil {
		t.Fatalf("%s: selected = %v, want %v: %v", desc, got, want, err)
	}
}

// TestQualityProfileUsergroupAssociationGroupByReference creates a
// QualityProfile and a Group, associates them through references, and
// verifies SonarQube grants the group edit rights on the profile. It then
// deletes the association and verifies the edit rights are revoked.
func TestQualityProfileUsergroupAssociationGroupByReference(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		profileCRName = "e2e-qpassoc-group-profile"
		profileName   = "e2e-qpassoc-group-profile"
		groupCRName   = "e2e-qpassoc-group"
		groupName     = "e2e-qpassoc-group"
		assocCRName   = "e2e-qpassoc-group-assoc"
	)

	profile := newE2EQualityProfile(f, profileCRName, profileName)
	f.CreateAndWaitForReady(t, profile, 2*time.Minute)
	e2e.AssertReady(t, profile)

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: groupCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.GroupParameters{
				Name: groupName,
			},
		},
	}
	f.CreateAndWaitForReady(t, group, 2*time.Minute)
	e2e.AssertReady(t, group)

	assoc := newE2EQualityProfileUsergroupAssociation(f, assocCRName, iamv1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfileRef: &xpv1.NamespacedReference{Name: profileCRName},
		Language:          qpAssocLanguage,
		GroupNameRef:      &xpv1.NamespacedReference{Name: groupCRName},
	})
	f.CreateAndWaitForReady(t, assoc, 2*time.Minute)
	e2e.AssertReady(t, assoc)
	e2e.AssertSynced(t, assoc)
	e2e.AssertExternalName(t, assoc, "group:"+groupName+":"+qpAssocLanguage+":"+profileName)

	// The QualityProfile reference must resolve to the display name, not
	// to the profile key stored as the QualityProfile external name.
	if got := assoc.Spec.ForProvider.QualityProfile; got != profileName {
		t.Errorf("resolved qualityProfile = %q, want %q", got, profileName)
	}
	if got := ptr.Deref(assoc.Spec.ForProvider.GroupName, ""); got != groupName {
		t.Errorf("resolved groupName = %q, want %q", got, groupName)
	}

	wantObs := iamv1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: profileName,
		Language:       qpAssocLanguage,
		GroupName:      groupName,
	}
	if assoc.Status.AtProvider != wantObs {
		t.Errorf("status.atProvider = %+v, want %+v", assoc.Status.AtProvider, wantObs)
	}

	selected, err := f.QualityProfileGroupSelected(context.Background(), qpAssocLanguage, profileName, groupName)
	if err != nil {
		t.Fatalf("searching quality profile groups: %v", err)
	}
	if !selected {
		t.Fatalf("group %q is not allowed to edit quality profile %q (%s) in SonarQube", groupName, profileName, qpAssocLanguage)
	}

	// Delete the association and verify SonarQube revokes the edit rights.
	f.Delete(t, assoc)
	if err := f.WaitForDeletion(context.Background(), assoc, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for association deletion: %v", err)
	}
	waitForQualityProfileSelected(t, false, "group association after delete", func(ctx context.Context) (bool, error) {
		return f.QualityProfileGroupSelected(ctx, qpAssocLanguage, profileName, groupName)
	})
}

// TestQualityProfileUsergroupAssociationUserDrift creates a QualityProfile
// and a User, associates them by name, and verifies SonarQube grants the
// user edit rights on the profile. It then revokes the rights directly in
// SonarQube and verifies the controller restores them.
func TestQualityProfileUsergroupAssociationUserDrift(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		profileCRName = "e2e-qpassoc-user-profile"
		profileName   = "e2e-qpassoc-user-profile"
		userCRName    = "e2e-qpassoc-user"
		userLogin     = "e2e-qpassoc-user"
		userName      = "E2E QP Association User"
		secretName    = "e2e-qpassoc-user-pwd"
		secretKey     = "password"
		assocCRName   = "e2e-qpassoc-user-assoc"
	)

	profile := newE2EQualityProfile(f, profileCRName, profileName)
	f.CreateAndWaitForReady(t, profile, 2*time.Minute)
	e2e.AssertReady(t, profile)

	pwSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-qpassoc-pw-123!"},
	}
	if err := f.Kube.Create(context.Background(), pwSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating password secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), pwSecret)
	})

	user := &iamv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: userCRName, Namespace: f.Namespace},
		Spec: iamv1alpha1.UserSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.UserParameters{
				Login:     userLogin,
				Name:      userName,
				Local:     ptr.To(true),
				Anonymize: ptr.To(true),
				PasswordSecretRef: &xpv1.SecretKeySelector{
					Key: secretKey,
					SecretReference: xpv1.SecretReference{
						Name:      secretName,
						Namespace: f.Namespace,
					},
				},
			},
		},
	}
	f.CreateAndWaitForReady(t, user, 2*time.Minute)
	e2e.AssertReady(t, user)

	assoc := newE2EQualityProfileUsergroupAssociation(f, assocCRName, iamv1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: profileName,
		Language:       qpAssocLanguage,
		Login:          ptr.To(userLogin),
	})
	f.CreateAndWaitForReady(t, assoc, 2*time.Minute)
	e2e.AssertReady(t, assoc)
	e2e.AssertSynced(t, assoc)
	e2e.AssertExternalName(t, assoc, "user:"+userLogin+":"+qpAssocLanguage+":"+profileName)

	wantObs := iamv1alpha1.QualityProfileUsergroupAssociationObservation{
		QualityProfile: profileName,
		Language:       qpAssocLanguage,
		Login:          userLogin,
	}
	if assoc.Status.AtProvider != wantObs {
		t.Errorf("status.atProvider = %+v, want %+v", assoc.Status.AtProvider, wantObs)
	}

	selected, err := f.QualityProfileUserSelected(context.Background(), qpAssocLanguage, profileName, userLogin)
	if err != nil {
		t.Fatalf("searching quality profile users: %v", err)
	}
	if !selected {
		t.Fatalf("user %q is not allowed to edit quality profile %q (%s) in SonarQube", userLogin, profileName, qpAssocLanguage)
	}

	// Revoke the edit rights out-of-band to simulate drift.
	resp, err := f.Sonar.Qualityprofiles.RemoveUser(context.Background(), &sonar.QualityprofilesRemoveUserOptions{
		Language:       qpAssocLanguage,
		Login:          userLogin,
		QualityProfile: profileName,
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("removing user from quality profile in SonarQube: %v", err)
	}

	// Touch an annotation to trigger a reconcile rather than waiting for
	// the provider poll interval.
	if err := f.Kube.Get(context.Background(), kubeKey(assoc), assoc); err != nil {
		t.Fatalf("re-fetching association CR: %v", err)
	}
	annotations := assoc.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	assoc.SetAnnotations(annotations)
	f.Update(t, assoc)

	waitForQualityProfileSelected(t, true, "user association after drift", func(ctx context.Context) (bool, error) {
		return f.QualityProfileUserSelected(ctx, qpAssocLanguage, profileName, userLogin)
	})
}

// TestQualityProfileUsergroupAssociationRejectsBothPrincipals verifies the
// CRD validation rejects an association that sets both a group and a user.
func TestQualityProfileUsergroupAssociationRejectsBothPrincipals(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	assoc := newE2EQualityProfileUsergroupAssociation(f, "e2e-qpassoc-invalid", iamv1alpha1.QualityProfileUsergroupAssociationParameters{
		QualityProfile: "e2e-qpassoc-invalid-profile",
		Language:       qpAssocLanguage,
		GroupName:      ptr.To("e2e-qpassoc-invalid-group"),
		LoginRef:       &xpv1.NamespacedReference{Name: "e2e-qpassoc-invalid-user"},
	})

	err := f.Kube.Create(context.Background(), assoc)
	if err == nil {
		_ = f.Kube.Delete(context.Background(), assoc)
		t.Fatal("creating an association with both groupName and loginRef succeeded, want validation error")
	}
	if !apierrors.IsInvalid(err) {
		t.Fatalf("creating an association with both groupName and loginRef: got %v, want an Invalid error", err)
	}
}
