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

//go:build e2e

package e2e_test

import (
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestFrameworkExample is the living documentation for package e2e: it
// drives a Group resource through the create → ready → verify-in-SonarQube
// → cleanup flow using only the framework's public surface.
func TestFrameworkExample(t *testing.T) {
	f := e2e.New(t)

	const groupName = "e2e-framework-example"

	group := &iamv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name:      groupName,
			Namespace: "default",
		},
		Spec: iamv1alpha1.GroupSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
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
	e2e.AssertSynced(t, group)

	// The provider stores the SonarQube-assigned ID in the external-name
	// annotation; use it to verify the group really exists in SonarQube.
	id := group.GetAnnotations()["crossplane.io/external-name"]
	if id == "" {
		t.Fatalf("expected external-name annotation to be populated after Ready")
	}
	got, err := f.FetchGroup(id)
	if err != nil {
		t.Fatalf("fetching group from SonarQube: %v", err)
	}
	if got == nil {
		t.Fatalf("group %s (id=%s) not found in SonarQube", groupName, id)
	}
	if got.Name != groupName {
		t.Errorf("SonarQube group name = %q, want %q", got.Name, groupName)
	}
}
