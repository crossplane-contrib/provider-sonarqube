//go:build e2e && enterprise

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

package instance_test

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestLicenseCRUD verifies the shared License managed resource (applied by
// requireLicense) reports Ready with an observation that matches what
// SonarQube itself reports for the currently applied license. Requires
// SONARQUBE_LICENSE_KEY; skips otherwise since a valid Enterprise Edition
// license is a paid artifact not every environment has.
func TestLicenseCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	lic := &instancev1alpha1.License{ObjectMeta: metav1.ObjectMeta{Name: licenseCRName, Namespace: f.Namespace}}
	if err := f.Kube.Get(context.Background(), kubeKey(lic), lic); err != nil {
		t.Fatalf("get License %s: %v", licenseCRName, err)
	}
	e2e.AssertReady(t, lic)
	e2e.AssertSynced(t, lic)

	if !lic.Status.AtProvider.IsValidEdition {
		t.Errorf("License AtProvider.IsValidEdition = false, want true")
	}
	if lic.Status.AtProvider.ProductEdition == "" {
		t.Errorf("License AtProvider.ProductEdition is empty, want a non-empty edition name")
	}

	got, err := f.FetchLicense()
	if err != nil {
		t.Fatalf("fetching license via SonarQube API: %v", err)
	}
	if !got.IsValidEdition {
		t.Errorf("SonarQube license IsValidEdition = false, want true")
	}
	if got.ProductEdition != lic.Status.AtProvider.ProductEdition {
		t.Errorf("SonarQube license productEdition = %q, want %q (License AtProvider)", got.ProductEdition, lic.Status.AtProvider.ProductEdition)
	}
}
