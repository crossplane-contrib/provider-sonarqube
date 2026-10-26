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
	"slices"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// newE2EApplicationProject returns a Project managed resource used as an
// application member.
func newE2EApplicationProject(f *e2e.Framework, key string) *instancev1alpha1.Project {
	return &instancev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: key, Namespace: f.Namespace},
		Spec: instancev1alpha1.ProjectSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ProjectParameters{
				Key:             key,
				Name:            key,
				Visibility:      new("public"),
				QualityGateName: new("Sonar way"),
			},
		},
	}
}

// waitForApplicationProjects polls SonarQube until the application members
// match want, ignoring order.
func waitForApplicationProjects(t *testing.T, f *e2e.Framework, key string, want []string) {
	t.Helper()

	var got []string
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		app, err := f.FindApplicationByKey(ctx, key)
		if err != nil || app == nil {
			return false, err
		}
		got = got[:0]
		for _, project := range app.Projects {
			got = append(got, project.Key)
		}
		slices.Sort(got)
		return slices.Equal(got, want), nil
	}); err != nil {
		t.Fatalf("application %q members = %v, want %v: %v", key, got, want, err)
	}
}

// TestApplicationCRUD creates two Projects and an Application referencing
// them, verifies the application and its membership in SonarQube, then
// drops a member from the spec and verifies it is removed. Applications
// are an Enterprise Edition feature that only function on a licensed
// instance, so this requires SONARQUBE_LICENSE_KEY; it skips otherwise.
//
// Application branches are not exercised: they require analyzed project
// branches, which the e2e environment does not produce.
func TestApplicationCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName   = "e2e-application-crud"
		appKey   = "e2e-application-crud"
		appName  = "E2E Application CRUD"
		projectA = "e2e-application-crud-a"
		projectB = "e2e-application-crud-b"
	)

	f.CreateAndWaitForReady(t, newE2EApplicationProject(f, projectA), 2*time.Minute)
	f.CreateAndWaitForReady(t, newE2EApplicationProject(f, projectB), 2*time.Minute)

	application := &instancev1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.ApplicationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ApplicationParameters{
				Key:         appKey,
				Name:        appName,
				Description: new("created by the enterprise e2e suite"),
				Visibility:  "public",
				ProjectRefs: []xpv1.Reference{{Name: projectA}, {Name: projectB}},
			},
		},
	}

	f.CreateAndWaitForReady(t, application, 2*time.Minute)
	e2e.AssertReady(t, application)
	e2e.AssertSynced(t, application)
	e2e.AssertExternalName(t, application, appKey)

	got, err := f.FindApplicationByKey(context.Background(), appKey)
	if err != nil {
		t.Fatalf("fetching application: %v", err)
	}
	if got == nil {
		t.Fatalf("application %q not found in SonarQube", appKey)
	}
	if got.Name != appName {
		t.Errorf("application name = %q, want %q", got.Name, appName)
	}
	if got.Visibility != "public" {
		t.Errorf("application visibility = %q, want %q", got.Visibility, "public")
	}

	waitForApplicationProjects(t, f, appKey, []string{projectA, projectB})

	// Drop projectB from the membership.
	if err := f.Kube.Get(context.Background(), kubeKey(application), application); err != nil {
		t.Fatalf("re-fetching application CR: %v", err)
	}
	application.Spec.ForProvider.Projects = nil
	application.Spec.ForProvider.ProjectRefs = []xpv1.Reference{{Name: projectA}}
	f.Update(t, application)

	waitForApplicationProjects(t, f, appKey, []string{projectA})
}
