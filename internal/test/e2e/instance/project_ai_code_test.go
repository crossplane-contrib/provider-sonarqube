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
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// waitForProjectContainsAiCode polls SonarQube until the AI code flag of the
// project equals want.
func waitForProjectContainsAiCode(t *testing.T, f *e2e.Framework, key string, want bool) {
	t.Helper()

	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, err := f.FetchProjectContainsAiCode(ctx, key)

		return got == want, err
	})
	if err != nil {
		t.Fatalf("project %q containsAiCode did not become %t: %v", key, want, err)
	}
}

// TestProjectContainsAiCode creates a Project flagged as containing AI code,
// verifies SonarQube reports the flag, changes only that field, then changes
// the flag directly in SonarQube and verifies the controller restores it. AI
// Code Assurance is a commercial edition feature, so this requires
// SONARQUBE_LICENSE_KEY; it skips otherwise.
func TestProjectContainsAiCode(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const key = "e2e-project-ai-code"

	project := newE2EApplicationProject(f, key)
	project.Spec.ForProvider.ContainsAiCode = new(true)

	f.CreateAndWaitForReady(t, project, 2*time.Minute)
	waitForProjectContainsAiCode(t, f, key, true)

	// Only the flag changes.
	err := f.Kube.Get(context.Background(), kubeKey(project), project)
	if err != nil {
		t.Fatalf("re-fetching project CR: %v", err)
	}

	project.Spec.ForProvider.ContainsAiCode = new(false)
	f.Update(t, project)
	waitForProjectContainsAiCode(t, f, key, false)

	// Drift: flip the flag in SonarQube, then force a reconcile.
	resp, err := f.Sonar.Projects.SetContainsAiCode(context.Background(), &sonar.ProjectsSetContainsAiCodeOptions{Project: key, ContainsAiCode: true})
	helpers.CloseBody(resp)

	if err != nil {
		t.Fatalf("setting AI code flag in SonarQube: %v", err)
	}

	err = f.Kube.Get(context.Background(), kubeKey(project), project)
	if err != nil {
		t.Fatalf("re-fetching project CR: %v", err)
	}

	annotations := project.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	project.SetAnnotations(annotations)
	f.Update(t, project)
	waitForProjectContainsAiCode(t, f, key, false)
}
