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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestGroupCRUD creates a Group with two permissions, waits for Ready,
// verifies the SonarQube group exists with the expected name and that the
// global permissions API reports the requested set.
func TestGroupCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName    = "e2e-group-crud"
		groupName = "e2e-group-crud"
	)
	wantPerms := []string{"scan", "provisioning"}

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.GroupParameters{
				Name:        groupName,
				Description: ptr.To("E2E-managed group"),
				Permissions: &wantPerms,
			},
		},
	}

	f.CreateAndWaitForReady(t, group, 2*time.Minute)
	e2e.AssertReady(t, group)
	e2e.AssertSynced(t, group)

	// External-name annotation holds the SonarQube group ID.
	id := group.GetAnnotations()["crossplane.io/external-name"]
	if id == "" {
		t.Fatalf("expected external-name to be populated after Ready")
	}

	got, err := f.FetchGroup(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching group: %v", err)
	}
	if got == nil {
		t.Fatalf("group %q (id=%s) not found in SonarQube", groupName, id)
	}
	if got.Name != groupName {
		t.Errorf("group name = %q, want %q", got.Name, groupName)
	}

	gotPerms, err := f.GroupPermissions(context.Background(), groupName)
	if err != nil {
		t.Fatalf("fetching group permissions: %v", err)
	}
	if !equalStringSets(gotPerms, wantPerms) {
		t.Errorf("group permissions = %v, want %v (order ignored)", gotPerms, wantPerms)
	}
}
