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

// newE2EQualityGate builds a QualityGate CR with a single condition.
func newE2EQualityGate(f *e2e.Framework, crName, gateName string) *instancev1alpha1.QualityGate {
	return &instancev1alpha1.QualityGate{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.QualityGateSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityGateParameters{
				Name: gateName,
				Conditions: []instancev1alpha1.QualityGateConditionParameters{
					{Metric: "blocker_violations", Op: ptr.To("GT"), Error: "0"},
				},
			},
		},
	}
}

// newE2EQualityGateUsergroupAssociation builds a
// QualityGateUsergroupAssociation CR with the given parameters.
func newE2EQualityGateUsergroupAssociation(f *e2e.Framework, crName string, params iamv1alpha1.QualityGateUsergroupAssociationParameters) *iamv1alpha1.QualityGateUsergroupAssociation {
	return &iamv1alpha1.QualityGateUsergroupAssociation{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: iamv1alpha1.QualityGateUsergroupAssociationSpec{
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

// waitForSelected polls check until it reports want or the timeout
// elapses.
func waitForSelected(t *testing.T, want bool, desc string, check func(context.Context) (bool, error)) {
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

// TestQualityGateUsergroupAssociationGroupByReference creates a QualityGate
// and a Group, associates them through references, and verifies SonarQube
// grants the group edit rights on the gate. It then deletes the association
// and verifies the edit rights are revoked.
func TestQualityGateUsergroupAssociationGroupByReference(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		gateCRName  = "e2e-qgassoc-group-gate"
		gateName    = "e2e-qgassoc-group-gate"
		groupCRName = "e2e-qgassoc-group"
		groupName   = "e2e-qgassoc-group"
		assocCRName = "e2e-qgassoc-group-assoc"
	)

	gate := newE2EQualityGate(f, gateCRName, gateName)
	f.CreateAndWaitForReady(t, gate, 2*time.Minute)
	e2e.AssertReady(t, gate)

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

	assoc := newE2EQualityGateUsergroupAssociation(f, assocCRName, iamv1alpha1.QualityGateUsergroupAssociationParameters{
		GateNameRef:  &xpv1.NamespacedReference{Name: gateCRName},
		GroupNameRef: &xpv1.NamespacedReference{Name: groupCRName},
	})
	f.CreateAndWaitForReady(t, assoc, 2*time.Minute)
	e2e.AssertReady(t, assoc)
	e2e.AssertSynced(t, assoc)
	e2e.AssertExternalName(t, assoc, "group:"+groupName+":"+gateName)

	if got := assoc.Spec.ForProvider.GateName; got != gateName {
		t.Errorf("resolved gateName = %q, want %q", got, gateName)
	}
	if got := ptr.Deref(assoc.Spec.ForProvider.GroupName, ""); got != groupName {
		t.Errorf("resolved groupName = %q, want %q", got, groupName)
	}

	wantObs := iamv1alpha1.QualityGateUsergroupAssociationObservation{GateName: gateName, GroupName: groupName}
	if assoc.Status.AtProvider != wantObs {
		t.Errorf("status.atProvider = %+v, want %+v", assoc.Status.AtProvider, wantObs)
	}

	selected, err := f.QualityGateGroupSelected(context.Background(), gateName, groupName)
	if err != nil {
		t.Fatalf("searching quality gate groups: %v", err)
	}
	if !selected {
		t.Fatalf("group %q is not allowed to edit quality gate %q in SonarQube", groupName, gateName)
	}

	// Delete the association and verify SonarQube revokes the edit rights.
	f.Delete(t, assoc)
	if err := f.WaitForDeletion(context.Background(), assoc, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for association deletion: %v", err)
	}
	waitForSelected(t, false, "group association after delete", func(ctx context.Context) (bool, error) {
		return f.QualityGateGroupSelected(ctx, gateName, groupName)
	})
}

// TestQualityGateUsergroupAssociationUserDrift creates a QualityGate and a
// User, associates them by name, and verifies SonarQube grants the user
// edit rights on the gate. It then revokes the rights directly in
// SonarQube and verifies the controller restores them.
func TestQualityGateUsergroupAssociationUserDrift(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		gateCRName  = "e2e-qgassoc-user-gate"
		gateName    = "e2e-qgassoc-user-gate"
		userCRName  = "e2e-qgassoc-user"
		userLogin   = "e2e-qgassoc-user"
		userName    = "E2E QG Association User"
		secretName  = "e2e-qgassoc-user-pwd"
		secretKey   = "password"
		assocCRName = "e2e-qgassoc-user-assoc"
	)

	gate := newE2EQualityGate(f, gateCRName, gateName)
	f.CreateAndWaitForReady(t, gate, 2*time.Minute)
	e2e.AssertReady(t, gate)

	pwSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: "e2e-qgassoc-pw-123!"},
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

	assoc := newE2EQualityGateUsergroupAssociation(f, assocCRName, iamv1alpha1.QualityGateUsergroupAssociationParameters{
		GateName: gateName,
		Login:    ptr.To(userLogin),
	})
	f.CreateAndWaitForReady(t, assoc, 2*time.Minute)
	e2e.AssertReady(t, assoc)
	e2e.AssertSynced(t, assoc)
	e2e.AssertExternalName(t, assoc, "user:"+userLogin+":"+gateName)

	wantObs := iamv1alpha1.QualityGateUsergroupAssociationObservation{GateName: gateName, Login: userLogin}
	if assoc.Status.AtProvider != wantObs {
		t.Errorf("status.atProvider = %+v, want %+v", assoc.Status.AtProvider, wantObs)
	}

	selected, err := f.QualityGateUserSelected(context.Background(), gateName, userLogin)
	if err != nil {
		t.Fatalf("searching quality gate users: %v", err)
	}
	if !selected {
		t.Fatalf("user %q is not allowed to edit quality gate %q in SonarQube", userLogin, gateName)
	}

	// Revoke the edit rights out-of-band to simulate drift.
	resp, err := f.Sonar.Qualitygates.RemoveUser(context.Background(), &sonar.QualitygatesRemoveUserOptions{
		GateName: gateName,
		Login:    userLogin,
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("removing user from quality gate in SonarQube: %v", err)
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

	waitForSelected(t, true, "user association after drift", func(ctx context.Context) (bool, error) {
		return f.QualityGateUserSelected(ctx, gateName, userLogin)
	})
}

// TestQualityGateUsergroupAssociationRejectsBothPrincipals verifies the CRD
// validation rejects an association that sets both a group and a user.
func TestQualityGateUsergroupAssociationRejectsBothPrincipals(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	assoc := newE2EQualityGateUsergroupAssociation(f, "e2e-qgassoc-invalid", iamv1alpha1.QualityGateUsergroupAssociationParameters{
		GateName:  "e2e-qgassoc-invalid-gate",
		GroupName: ptr.To("e2e-qgassoc-invalid-group"),
		LoginRef:  &xpv1.NamespacedReference{Name: "e2e-qgassoc-invalid-user"},
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
