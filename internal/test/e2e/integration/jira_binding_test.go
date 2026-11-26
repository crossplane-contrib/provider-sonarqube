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

package integration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	integrationv1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// waitForSyncError polls obj until its Synced condition is False with a
// message containing want, failing the test after two minutes.
func waitForSyncError(t *testing.T, f *e2e.Framework, obj e2e.Conditioned, want string) {
	t.Helper()

	var last xpv1.Condition
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := f.Kube.Get(ctx, kubeKey(obj), obj); err != nil {
			return false, err
		}

		last = obj.GetCondition(xpv1.TypeSynced)

		return strings.Contains(last.Message, want), nil
	}); err != nil {
		t.Fatalf("waiting for %s to report %q: %v; last condition: %+v", obj.GetName(), want, err, last)
	}
}

// TestJiraBindingsWithoutJiraConnection covers what can run without a Jira
// Cloud instance, as connecting one requires an Atlassian OAuth consent in a
// browser: on an unconnected SonarQube, the organization binding reports that
// it cannot be created, and both resources can be deleted.
func TestJiraBindingsWithoutJiraConnection(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		organizationName = "e2e-jira-organization-binding"
		projectName      = "e2e-jira-project"
		bindingName      = "e2e-jira-project-binding"
	)

	providerConfig := &xpv1.ProviderConfigReference{Kind: "ClusterProviderConfig", Name: f.ProviderConfigName}

	project := &instancev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: projectName, Namespace: f.Namespace},
		Spec: instancev1alpha1.ProjectSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{ProviderConfigReference: providerConfig},
			ForProvider:         instancev1alpha1.ProjectParameters{Key: projectName, Name: projectName},
		},
	}
	f.CreateAndWaitForReady(t, project, 2*time.Minute)

	organization := &integrationv1alpha1.JiraOrganizationBinding{
		ObjectMeta: metav1.ObjectMeta{Name: organizationName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.JiraOrganizationBindingSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{ProviderConfigReference: providerConfig},
		},
	}
	f.Apply(t, organization)
	t.Cleanup(func() { f.Delete(t, organization) })

	// This also checks that SonarQube answers "not found" for a missing
	// binding, which the controllers rely on.
	waitForSyncError(t, f, organization, "cannot be created through the API")

	jiraOrganizationBindingID := "e2e-unknown-binding"
	binding := &integrationv1alpha1.JiraProjectBinding{
		ObjectMeta: metav1.ObjectMeta{Name: bindingName, Namespace: f.Namespace},
		Spec: integrationv1alpha1.JiraProjectBindingSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{ProviderConfigReference: providerConfig},
			ForProvider: integrationv1alpha1.JiraProjectBindingParameters{
				ProjectKeyRef:             &xpv1.Reference{Name: projectName},
				JiraOrganizationBindingID: &jiraOrganizationBindingID,
				JiraProjectKey:            "E2E",
			},
		},
	}
	f.Apply(t, binding)
	t.Cleanup(func() { f.Delete(t, binding) })

	// SonarQube refuses to bind a project while no Jira instance is connected.
	waitForSyncError(t, f, binding, "cannot create Jira project binding")

	for _, obj := range []e2e.Conditioned{binding, organization} {
		f.Delete(t, obj)
		if err := f.WaitForDeletion(context.Background(), obj, e2e.DefaultDeleteTimeout); err != nil {
			t.Fatalf("waiting for %s deletion: %v", obj.GetName(), err)
		}
	}
}
