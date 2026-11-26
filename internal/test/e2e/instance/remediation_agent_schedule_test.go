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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestRemediationAgentScheduleGlobal adopts the instance-wide schedule and
// verifies SonarQube applies it. Requires SONARQUBE_LICENSE_KEY.
func TestRemediationAgentScheduleGlobal(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	schedule := newRemediationAgentSchedule(f, "e2e-remediation-schedule-global", instancev1alpha1.RemediationAgentScheduleParameters{
		Enabled:              new(true),
		Frequency:            "WEEKLY",
		DayOfWeek:            new("SUNDAY"),
		Hour:                 new(int32(3)),
		Timezone:             new("Europe/Paris"),
		MaxOpenPRs:           new(int32(-1)),
		ProjectSelectionMode: new("ALL"),
	})

	f.CreateAndWaitForReady(t, schedule, 2*time.Minute)
	e2e.AssertSynced(t, schedule)
	e2e.AssertExternalName(t, schedule, "*")

	got, err := f.FindRemediationAgentSchedule(context.Background(), "*")
	if err != nil {
		t.Fatalf("fetching the instance-wide schedule: %v", err)
	}

	if got == nil {
		t.Fatal("instance-wide schedule not found in SonarQube")
	}

	if !got.Enabled || got.Frequency != "WEEKLY" || got.DayOfWeek != "SUNDAY" || got.Hour != 3 || got.Timezone != "Europe/Paris" || got.MaxOpenPRs != -1 {
		t.Errorf("unexpected instance-wide schedule: %+v", got)
	}
}

// TestRemediationAgentScheduleProject creates the schedule of a referenced
// Project. Requires SONARQUBE_LICENSE_KEY.
func TestRemediationAgentScheduleProject(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	requireLicense(t, f)

	const (
		crName     = "e2e-remediation-schedule-project"
		projectKey = "e2e-remediation-schedule-project"
	)

	project := &instancev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.ProjectSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{Kind: "ClusterProviderConfig", Name: f.ProviderConfigName},
			},
			ForProvider: instancev1alpha1.ProjectParameters{Key: projectKey, Name: projectKey},
		},
	}
	f.CreateAndWaitForReady(t, project, 2*time.Minute)

	schedule := newRemediationAgentSchedule(f, crName, instancev1alpha1.RemediationAgentScheduleParameters{
		ProjectKeyRef: &xpv1.Reference{Name: project.Name},
		Enabled:       new(true),
		Frequency:     "DAILY",
		Hour:          new(int32(4)),
	})

	f.CreateAndWaitForReady(t, schedule, 2*time.Minute)
	e2e.AssertSynced(t, schedule)
	e2e.AssertExternalName(t, schedule, projectKey)

	got, err := f.FindRemediationAgentSchedule(context.Background(), projectKey)
	if err != nil {
		t.Fatalf("fetching the project schedule: %v", err)
	}

	if got == nil {
		t.Fatalf("schedule of project %q not found in SonarQube", projectKey)
	}

	if !got.Enabled || got.Frequency != "DAILY" || got.Hour != 4 {
		t.Errorf("unexpected project schedule: %+v", got)
	}
}

// newRemediationAgentSchedule builds a RemediationAgentSchedule managed
// resource using the e2e ClusterProviderConfig.
func newRemediationAgentSchedule(f *e2e.Framework, name string, params instancev1alpha1.RemediationAgentScheduleParameters) *instancev1alpha1.RemediationAgentSchedule {
	return &instancev1alpha1.RemediationAgentSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace},
		Spec: instancev1alpha1.RemediationAgentScheduleSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{Kind: "ClusterProviderConfig", Name: f.ProviderConfigName},
			},
			ForProvider: params,
		},
	}
}
