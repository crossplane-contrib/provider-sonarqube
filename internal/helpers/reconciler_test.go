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
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	resourcefake "github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	iamv1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
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

	const commonOptions = 4

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

			opts := Options{Options: controller.Options{
				Logger:           logging.NewNopLogger(),
				Features:         &feature.Flags{},
				MetricOptions:    test.metrics,
				ChangeLogOptions: &controller.ChangeLogOptions{},
			}}
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
			opts := Options{Options: controller.Options{Logger: logging.NewNopLogger(), MetricOptions: test.metrics}}

			err := AddMRStateRecorder(mgr, opts, &iamv1alpha1.GroupList{})
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

// newTestReconciler returns a reconciler for a QualityGate that is up to date
// with SonarQube, using the options of ReconcilerOptions.
func newTestReconciler(t *testing.T, opts *Options, gate *v1alpha1.QualityGate) *managed.Reconciler {
	t.Helper()

	scheme := runtime.NewScheme()

	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	kube := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(gate).WithStatusSubresource(gate).Build()
	mgr := recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: scheme}}

	connector := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(context.Context, resource.Managed) (managed.ExternalObservation, error) {
				return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	return managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.QualityGateGroupVersionKind),
		ReconcilerOptions(mgr, "test", *opts, managed.WithExternalConnector(connector))...)
}

// TestReconcilerOptionsPollInterval checks the requeue delay of a real
// reconcile for each value of the crossplane.io/poll-interval annotation.
func TestReconcilerOptionsPollInterval(t *testing.T) {
	t.Parallel()

	const (
		pollInterval    = time.Minute
		minPollInterval = 30 * time.Second
	)

	tests := map[string]struct {
		annotation string
		want       time.Duration
	}{
		"NoAnnotationUsesThePollInterval": {
			want: pollInterval,
		},
		"AnnotationAboveTheMinimumIsHonoured": {
			annotation: "2h",
			want:       2 * time.Hour,
		},
		"AnnotationBelowThePollIntervalIsHonoured": {
			annotation: "45s",
			want:       45 * time.Second,
		},
		"AnnotationBelowTheMinimumIsRaised": {
			annotation: "1ms",
			want:       minPollInterval,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gate := &v1alpha1.QualityGate{ObjectMeta: metav1.ObjectMeta{Name: "gate", Namespace: "team"}}
			if test.annotation != "" {
				meta.AddAnnotations(gate, map[string]string{meta.AnnotationKeyPollInterval: test.annotation})
			}

			opts := &Options{
				Options: controller.Options{
					Logger:       logging.NewNopLogger(),
					Features:     &feature.Flags{},
					PollInterval: pollInterval,
				},
				MinPollInterval: minPollInterval,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "gate", Namespace: "team"}}

			got, err := newTestReconciler(t, opts, gate).Reconcile(t.Context(), request)
			if err != nil {
				t.Fatalf("Reconcile() unexpected error: %v", err)
			}

			if got.RequeueAfter != test.want {
				t.Errorf("Reconcile() RequeueAfter = %v, want %v", got.RequeueAfter, test.want)
			}
		})
	}
}
