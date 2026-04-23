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

package instance_test

import (
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestProjectCRUD creates a Project, waits for Ready, verifies it is
// queryable via SonarQube's Projects.Search with the matching key,
// visibility and display name.
func TestProjectCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName    = "e2e-project-crud"
		projKey   = "e2e-project-crud"
		projName  = "E2E Project CRUD"
		visPublic = "public"
	)

	project := &instancev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: instancev1alpha1.ProjectSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ProjectParameters{
				Key:           projKey,
				Name:          projName,
				Visibility:    stringPtr(visPublic),
				DefaultBranch: stringPtr("main"),
				Tags:          &[]string{"crossplane", "e2e"},
				// Pin to the built-in "Sonar way" gate. Leaving this unset
				// causes Crossplane's reference resolver to issue a server
				// side apply with qualityGateName="" which fails the field's
				// MinLength=1 CRD validation.
				QualityGateName: stringPtr("Sonar way"),
			},
		},
	}

	f.CreateAndWaitForReady(t, project, 2*time.Minute)
	e2e.AssertReady(t, project)
	e2e.AssertSynced(t, project)
	e2e.AssertExternalName(t, project, projKey)

	got, err := f.FindProjectByKey(projKey)
	if err != nil {
		t.Fatalf("searching projects: %v", err)
	}
	if got == nil {
		t.Fatalf("project %q not found in SonarQube", projKey)
	}
	if got.Name != projName {
		t.Errorf("project name = %q, want %q", got.Name, projName)
	}
	if got.Visibility != visPublic {
		t.Errorf("project visibility = %q, want %q", got.Visibility, visPublic)
	}
}
