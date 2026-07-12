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
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestPermissionsTemplateCRUD creates a PermissionsTemplate with a
// project-key pattern and verifies SonarQube reports it via SearchTemplates.
func TestPermissionsTemplateCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName       = "e2e-permstemplate-crud"
		templateName = "e2e-permstemplate-crud"
		keyPattern   = "^e2e-permstemplate-.*"
	)

	pt := &iamv1alpha1.PermissionsTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: iamv1alpha1.PermissionsTemplateSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PermissionsTemplateParameters{
				Name:              templateName,
				Description:       ptr.To("E2E-managed permissions template"),
				ProjectKeyPattern: ptr.To(keyPattern),
			},
		},
	}

	f.CreateAndWaitForReady(t, pt, 2*time.Minute)
	e2e.AssertReady(t, pt)
	e2e.AssertSynced(t, pt)

	got, err := f.FindPermissionsTemplate(templateName)
	if err != nil {
		t.Fatalf("searching permissions templates: %v", err)
	}
	if got == nil {
		t.Fatalf("permissions template %q not found in SonarQube", templateName)
	}
	if got.ProjectKeyPattern != keyPattern {
		t.Errorf("template projectKeyPattern = %q, want %q", got.ProjectKeyPattern, keyPattern)
	}
}
