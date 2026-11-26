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

//nolint:bodyclose // mocks hand out empty bodies
package remediationagentschedule

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// errBoom is the error returned by failing mocks.
var errBoom = errors.New("boom")

// mockGate records the callback registered on the feature gate.
type mockGate struct{ callback func() }

// Register records the callback.
func (m *mockGate) Register(callback func(), _ ...schema.GroupVersionKind) { m.callback = callback }

// Set does nothing.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool { return false }

// response returns an HTTP response with the given status.
func response(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: http.NoBody}
}

// newSchedule returns a daily schedule for the given scope.
func newSchedule(projectKey *string) *v1alpha1.RemediationAgentSchedule {
	return &v1alpha1.RemediationAgentSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "schedule", Namespace: "default"},
		Spec: v1alpha1.RemediationAgentScheduleSpec{ForProvider: v1alpha1.RemediationAgentScheduleParameters{
			ProjectKey: projectKey, Enabled: new(true), Frequency: "DAILY", Hour: new(int32(2)),
		}},
	}
}

// unresolvedSchedule returns a schedule whose project is not resolved yet.
func unresolvedSchedule() *v1alpha1.RemediationAgentSchedule {
	schedule := newSchedule(nil)
	schedule.Spec.ForProvider.ProjectKeyRef = &xpv1.Reference{Name: "project"}

	return schedule
}

// deleted marks the schedule as being deleted.
func deleted(schedule *v1alpha1.RemediationAgentSchedule) *v1alpha1.RemediationAgentSchedule {
	schedule.DeletionTimestamp = new(metav1.Now())

	return schedule
}

// getter returns a successful get mock.
func getter(enabled bool) func(string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
	return func(id string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
		return &sonar.RemediationAgentScheduledAgentConfigResource{
			Id: id, Enabled: enabled, Frequency: "DAILY", Hour: 2, Timezone: "UTC", MaxIssuesPerRun: 5, MaxOpenPRs: 3, ProjectSelectionMode: "ALL",
		}, response(http.StatusOK), nil
	}
}

// failingGetter returns a get mock failing with the given status.
func failingGetter(status int) func(string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
	return func(string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
		return nil, response(status), errBoom
	}
}

// TestObserve verifies the observation of each scope.
func TestObserve(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		schedule         *v1alpha1.RemediationAgentSchedule
		client           *fake.MockRemediationAgentSchedulesClient
		wantExists       bool
		wantUpToDate     bool
		wantExternalName string
		wantErr          bool
	}{
		"Unresolved":             {schedule: unresolvedSchedule(), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: failingGetter(http.StatusInternalServerError)}},
		"NotFound":               {schedule: newSchedule(new("p")), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: failingGetter(http.StatusNotFound)}},
		"GetError":               {schedule: newSchedule(new("p")), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: failingGetter(http.StatusForbidden)}, wantErr: true},
		"AdoptGlobal":            {schedule: newSchedule(nil), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(true)}, wantExists: true, wantUpToDate: true, wantExternalName: "*"},
		"Outdated":               {schedule: newSchedule(new("p")), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(false)}, wantExists: true, wantExternalName: "p"},
		"DeletedGlobalDisabled":  {schedule: deleted(newSchedule(nil)), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(false)}},
		"DeletedGlobalEnabled":   {schedule: deleted(newSchedule(nil)), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(true)}, wantExists: true, wantUpToDate: true, wantExternalName: "*"},
		"DeletedProjectDisabled": {schedule: deleted(newSchedule(new("p"))), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(false)}, wantExists: true, wantExternalName: "p"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			got, err := e.Observe(context.Background(), tc.schedule)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got.ResourceExists != tc.wantExists || got.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("Observe() = %+v, want exists %v, up to date %v", got, tc.wantExists, tc.wantUpToDate)
			}

			if name := meta.GetExternalName(tc.schedule); name != tc.wantExternalName {
				t.Errorf("external name = %q, want %q", name, tc.wantExternalName)
			}

			if got.ResourceExists && !got.ResourceLateInitialized {
				t.Error("Observe() must report late initialization")
			}
		})
	}
}

// TestObserveStable verifies a second observation changes nothing.
func TestObserveStable(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: getter(true)}}
	schedule := newSchedule(nil)

	_, err := e.Observe(context.Background(), schedule)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}

	got, err := e.Observe(context.Background(), schedule)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}

	if got.ResourceLateInitialized || !got.ResourceUpToDate {
		t.Errorf("second Observe() = %+v, want up to date without late initialization", got)
	}
}

// TestCreate verifies creation.
func TestCreate(t *testing.T) {
	t.Parallel()

	failingCreate := func(*sonar.RemediationAgentCreateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
		return nil, response(http.StatusConflict), errBoom
	}

	cases := map[string]struct {
		schedule         *v1alpha1.RemediationAgentSchedule
		client           *fake.MockRemediationAgentSchedulesClient
		wantExternalName string
		wantErr          bool
	}{
		"Project":    {schedule: newSchedule(new("p")), client: &fake.MockRemediationAgentSchedulesClient{}, wantExternalName: "p"},
		"Global":     {schedule: newSchedule(nil), client: &fake.MockRemediationAgentSchedulesClient{}, wantExternalName: "*"},
		"Unresolved": {schedule: unresolvedSchedule(), client: &fake.MockRemediationAgentSchedulesClient{}, wantErr: true},
		"Error":      {schedule: newSchedule(new("p")), client: &fake.MockRemediationAgentSchedulesClient{CreateScheduledAgentConfigFn: failingCreate}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			_, err := e.Create(context.Background(), tc.schedule)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got := meta.GetExternalName(tc.schedule); got != tc.wantExternalName {
				t.Errorf("external name = %q, want %q", got, tc.wantExternalName)
			}
		})
	}
}

// TestUpdate verifies updates.
func TestUpdate(t *testing.T) {
	t.Parallel()

	failingUpdate := func(string, *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
		return nil, response(http.StatusBadRequest), errBoom
	}

	cases := map[string]struct {
		schedule *v1alpha1.RemediationAgentSchedule
		client   *fake.MockRemediationAgentSchedulesClient
		wantErr  bool
	}{
		"Success":     {schedule: newSchedule(nil), client: &fake.MockRemediationAgentSchedulesClient{}},
		"Unresolved":  {schedule: unresolvedSchedule(), client: &fake.MockRemediationAgentSchedulesClient{}, wantErr: true},
		"GetError":    {schedule: newSchedule(nil), client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: failingGetter(http.StatusForbidden)}, wantErr: true},
		"UpdateError": {schedule: newSchedule(nil), client: &fake.MockRemediationAgentSchedulesClient{UpdateScheduledAgentConfigFn: failingUpdate}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			_, err := e.Update(context.Background(), tc.schedule)
			if (err != nil) != tc.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestDeleteProject verifies the deletion of a project schedule.
func TestDeleteProject(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status  int
		err     error
		wantErr bool
	}{
		"Success":     {status: http.StatusNoContent},
		"AlreadyGone": {status: http.StatusNotFound, err: errBoom},
		"Error":       {status: http.StatusForbidden, err: errBoom, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: &fake.MockRemediationAgentSchedulesClient{
				DeleteScheduledAgentConfigFn: func(string) (*http.Response, error) { return response(tc.status), tc.err }, //nolint:nilnil // errors come with their response
			}}

			_, err := e.Delete(context.Background(), newSchedule(new("p")))
			if (err != nil) != tc.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestDeleteGlobal verifies the instance-wide schedule is disabled.
func TestDeleteGlobal(t *testing.T) {
	t.Parallel()

	var got *sonar.RemediationAgentUpdateScheduledAgentConfigOptions

	e := &external{client: &fake.MockRemediationAgentSchedulesClient{
		GetScheduledAgentConfigFn: getter(true),
		UpdateScheduledAgentConfigFn: func(_ string, opt *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
			got = opt

			return nil, response(http.StatusOK), nil
		},
		DeleteScheduledAgentConfigFn: func(string) (*http.Response, error) {
			t.Error("the instance-wide schedule must not be deleted")

			return response(http.StatusBadRequest), errBoom //nolint:nilnil // errors come with their response
		},
	}}

	// Pending spec changes must not be applied while deleting.
	schedule := newSchedule(nil)
	schedule.Spec.ForProvider.Hour = new(int32(23))

	_, err := e.Delete(context.Background(), schedule)
	if err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	want := &sonar.RemediationAgentUpdateScheduledAgentConfigOptions{
		Enabled: new(false), Frequency: "DAILY", Hour: new(int32(2)), Timezone: "UTC",
		MaxIssuesPerRun: new(int32(5)), MaxOpenPRs: new(int32(3)), ProjectSelectionMode: "ALL",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("update options: -want, +got:\n%s", diff)
	}
}

// TestDeleteGlobalEmptySelection verifies a pruned selection does not block the
// deletion.
func TestDeleteGlobalEmptySelection(t *testing.T) {
	t.Parallel()

	var got string

	e := &external{client: &fake.MockRemediationAgentSchedulesClient{
		GetScheduledAgentConfigFn: func(id string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
			return &sonar.RemediationAgentScheduledAgentConfigResource{Id: id, Enabled: true, ProjectSelectionMode: "SELECTED"}, response(http.StatusOK), nil
		},
		UpdateScheduledAgentConfigFn: func(_ string, opt *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
			got = opt.ProjectSelectionMode

			return nil, response(http.StatusOK), nil
		},
	}}

	_, err := e.Delete(context.Background(), newSchedule(nil))
	if err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	if got != "ALL" {
		t.Errorf("projectSelectionMode = %q, want ALL", got)
	}
}

// TestDeleteNoop verifies the remaining deletion paths.
func TestDeleteNoop(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockRemediationAgentSchedulesClient{}}

	_, err := e.Delete(context.Background(), unresolvedSchedule())
	if err != nil {
		t.Errorf("Delete(): %v", err)
	}

	err = e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect(): %v", err)
	}

	failing := &external{client: &fake.MockRemediationAgentSchedulesClient{GetScheduledAgentConfigFn: failingGetter(http.StatusForbidden)}}

	_, err = failing.Delete(context.Background(), newSchedule(nil))
	if err == nil {
		t.Error("Delete() must fail when the schedule cannot be read")
	}
}

// TestConnect verifies ProviderConfig usage tracking errors.
func TestConnect(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(): %v", err)
	}

	kube := fakekube.NewClientBuilder().WithScheme(scheme).Build()
	c := &connector{
		kube:  kube,
		usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
		newServiceFn: func(common.Config) instance.RemediationAgentSchedulesClient {
			return &fake.MockRemediationAgentSchedulesClient{}
		},
	}

	// Without a ProviderConfig reference usage tracking fails.
	_, err = c.Connect(context.Background(), newSchedule(nil))
	if err == nil {
		t.Error("Connect() must fail without a ProviderConfig reference")
	}
}

// TestSetupGated verifies the controller registers behind the gate.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}

	err := SetupGated(nil, helpers.Options{Options: controller.Options{Gate: gate}})
	if err != nil {
		t.Fatalf("SetupGated(): %v", err)
	}

	if gate.callback == nil {
		t.Error("SetupGated() must register a callback")
	}
}
