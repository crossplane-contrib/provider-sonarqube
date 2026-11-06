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

package helpers

import (
	"errors"
	"slices"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
)

// recorderManager is a manager that only provides event recorders.
type recorderManager struct {
	ctrl.Manager
}

// GetEventRecorderFor returns a fake event recorder.
func (recorderManager) GetEventRecorderFor(string) record.EventRecorder {
	return record.NewFakeRecorder(1)
}

// TestReconcilerOptions checks which options are enabled by each setting, and
// that the options of the calling controller are kept, in order, after the
// shared ones so that they can override them.
func TestReconcilerOptions(t *testing.T) {
	t.Parallel()

	const commonOptions = 3

	tests := map[string]struct {
		features []feature.Flag
		metrics  *controller.MetricOptions
		specific int
		want     int
	}{
		"NoFeatures": {
			want: commonOptions,
		},
		"ManagementPolicies": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies},
			want:     commonOptions + 1,
		},
		"ChangeLogs": {
			features: []feature.Flag{feature.EnableAlphaChangeLogs},
			want:     commonOptions + 1,
		},
		"Metrics": {
			metrics: &controller.MetricOptions{},
			want:    commonOptions + 1,
		},
		"SpecificOptions": {
			specific: 2,
			want:     commonOptions + 2,
		},
		"AllOptions": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies, feature.EnableAlphaChangeLogs},
			metrics:  &controller.MetricOptions{},
			specific: 2,
			want:     commonOptions + 5,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := controller.Options{
				Logger:           logging.NewNopLogger(),
				Features:         &feature.Flags{},
				MetricOptions:    test.metrics,
				ChangeLogOptions: &controller.ChangeLogOptions{},
			}
			for _, flag := range test.features {
				opts.Features.Enable(flag)
			}

			var applied []int

			wantApplied := make([]int, test.specific)
			specific := make([]managed.ReconcilerOption, test.specific)

			for i := range specific {
				wantApplied[i] = i
				specific[i] = func(*managed.Reconciler) { applied = append(applied, i) }
			}

			got := ReconcilerOptions(recorderManager{}, "test", opts, specific...)
			if len(got) != test.want {
				t.Fatalf("ReconcilerOptions() returned %d options, want %d", len(got), test.want)
			}

			for _, option := range got[len(got)-test.specific:] {
				option(&managed.Reconciler{})
			}

			if !slices.Equal(applied, wantApplied) {
				t.Errorf("last options applied specific options %v, want %v", applied, wantApplied)
			}
		})
	}
}

// runnableManager is a manager that records the runnables added to it.
type runnableManager struct {
	ctrl.Manager

	added []manager.Runnable
	err   error
}

// GetClient returns a nil client, as the recorder is never started.
func (*runnableManager) GetClient() client.Client { return nil }

// Add records the runnable and returns the configured error.
func (m *runnableManager) Add(runnable manager.Runnable) error {
	m.added = append(m.added, runnable)

	return m.err
}

// TestAddMRStateRecorder checks that a recorder is only registered when state
// metrics are enabled, and that a registration failure is reported.
func TestAddMRStateRecorder(t *testing.T) {
	t.Parallel()

	errAdd := errors.New("add failed")

	tests := map[string]struct {
		metrics   *controller.MetricOptions
		addErr    error
		wantAdded int
		wantErr   error
	}{
		"NoMetricOptions": {},
		"NoStateMetrics": {
			metrics: &controller.MetricOptions{},
		},
		"StateMetrics": {
			metrics:   &controller.MetricOptions{MRStateMetrics: &statemetrics.MRStateMetrics{}},
			wantAdded: 1,
		},
		"AddFails": {
			metrics:   &controller.MetricOptions{MRStateMetrics: &statemetrics.MRStateMetrics{}},
			addErr:    errAdd,
			wantAdded: 1,
			wantErr:   errAdd,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mgr := &runnableManager{err: test.addErr}
			opts := controller.Options{Logger: logging.NewNopLogger(), MetricOptions: test.metrics}

			err := AddMRStateRecorder(mgr, opts, &v1alpha1.GroupList{})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("AddMRStateRecorder() error = %v, want %v", err, test.wantErr)
			}

			if len(mgr.added) != test.wantAdded {
				t.Fatalf("AddMRStateRecorder() added %d runnables, want %d", len(mgr.added), test.wantAdded)
			}

			for _, runnable := range mgr.added {
				if _, isRecorder := runnable.(*statemetrics.MRStateRecorder); !isRecorder {
					t.Errorf("AddMRStateRecorder() added a %T, want a *statemetrics.MRStateRecorder", runnable)
				}
			}
		})
	}
}
