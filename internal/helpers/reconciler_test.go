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
	"math"
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

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
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

	const commonOptions = 5

	tests := map[string]struct {
		features []feature.Flag
		metrics  *controller.MetricOptions
		specific int
		jitter   int
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
		"PollJitter": {
			jitter: MaxPollJitterPercentage,
			want:   commonOptions + 1,
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

			opts := Options{
				Options: controller.Options{
					Logger:           logging.NewNopLogger(),
					Features:         &feature.Flags{},
					MetricOptions:    test.metrics,
					ChangeLogOptions: &controller.ChangeLogOptions{},
				},
				PollJitterPercentage: test.jitter,
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
			opts := Options{Options: controller.Options{Logger: logging.NewNopLogger(), MetricOptions: test.metrics}}

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

// newTestReconciler returns a reconciler for a Group that is up to date with
// SonarQube, using the options of ReconcilerOptions.
func newTestReconciler(t *testing.T, opts Options, group *v1alpha1.Group) *managed.Reconciler {
	t.Helper()

	reconciler, _ := newObservingTestReconciler(t, opts, group, managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true})

	return reconciler
}

// newObservingTestReconciler is newTestReconciler with an external client
// whose Observe reports observation, and that also returns the fake client
// holding group.
func newObservingTestReconciler(t *testing.T, opts Options, group *v1alpha1.Group, observation managed.ExternalObservation) (*managed.Reconciler, client.Client) {
	t.Helper()

	scheme := runtime.NewScheme()

	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	kube := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(group).WithStatusSubresource(group).Build()
	mgr := recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: scheme}}

	connector := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(context.Context, resource.Managed) (managed.ExternalObservation, error) {
				return observation, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	return managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.GroupGroupVersionKind),
		ReconcilerOptions(mgr, "test", opts, managed.WithExternalConnector(connector))...), kube
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
		noMinimum  bool
		want       time.Duration
	}{
		"NoAnnotationUsesThePollInterval": {
			want: pollInterval,
		},
		"InvalidAnnotationUsesThePollInterval": {
			annotation: "-1s",
			want:       pollInterval,
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
		"AnnotationAtTheMinimumIsHonoured": {
			annotation: "30s",
			want:       minPollInterval,
		},
		"ZeroMinimumRemovesTheLimit": {
			annotation: "1ms",
			noMinimum:  true,
			want:       time.Millisecond,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			group := &v1alpha1.Group{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: "team"}}
			if test.annotation != "" {
				meta.AddAnnotations(group, map[string]string{meta.AnnotationKeyPollInterval: test.annotation})
			}

			opts := Options{
				Options: controller.Options{
					Logger:       logging.NewNopLogger(),
					Features:     &feature.Flags{},
					PollInterval: pollInterval,
				},
				MinPollInterval: minPollInterval,
			}
			if test.noMinimum {
				opts.MinPollInterval = 0
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "group", Namespace: "team"}}

			got, err := newTestReconciler(t, opts, group).Reconcile(t.Context(), request)
			if err != nil {
				t.Fatalf("Reconcile() unexpected error: %v", err)
			}

			if got.RequeueAfter != test.want {
				t.Errorf("Reconcile() RequeueAfter = %v, want %v", got.RequeueAfter, test.want)
			}
		})
	}
}

// TestValidateMinPollInterval checks that only a negative minimum is rejected.
func TestValidateMinPollInterval(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		minimum time.Duration
		wantErr bool
	}{
		"Positive": {minimum: time.Second},
		"Zero":     {},
		"Negative": {minimum: -time.Second, wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMinPollInterval(test.minimum)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateMinPollInterval() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateCreationGracePeriod checks that only a negative period is
// rejected.
func TestValidateCreationGracePeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		period  time.Duration
		wantErr bool
	}{
		"Default":  {period: DefaultCreationGracePeriod},
		"Zero":     {},
		"Negative": {period: -time.Second, wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCreationGracePeriod(test.period)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateCreationGracePeriod() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestReconcilerOptionsCreationGracePeriod deletes a resource right after its
// create succeeded, while SonarQube already reports it gone: the finalizer is
// only released once the creation grace period is over.
func TestReconcilerOptionsCreationGracePeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		period        time.Duration
		wantRequeue   bool
		wantFinalizer bool
	}{
		"WithinThePeriodTheFinalizerIsKept": {
			period:        DefaultCreationGracePeriod,
			wantRequeue:   true,
			wantFinalizer: true,
		},
		"ZeroReleasesTheFinalizer": {},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deletedAt := metav1.Now()
			group := &v1alpha1.Group{ObjectMeta: metav1.ObjectMeta{
				Name:              "group",
				Namespace:         "team",
				DeletionTimestamp: &deletedAt,
				Finalizers:        []string{managed.FinalizerName},
			}}
			meta.SetExternalCreateSucceeded(group, time.Now())

			opts := Options{
				Options: controller.Options{
					Logger:       logging.NewNopLogger(),
					Features:     &feature.Flags{},
					PollInterval: time.Minute,
				},
				CreationGracePeriod: test.period,
			}

			reconciler, kube := newObservingTestReconciler(t, opts, group, managed.ExternalObservation{ResourceExists: false})
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "group", Namespace: "team"}}

			got, err := reconciler.Reconcile(t.Context(), request)
			if err != nil {
				t.Fatalf("Reconcile() unexpected error: %v", err)
			}

			if got.Requeue != test.wantRequeue { //nolint:staticcheck // crossplane-runtime still requeues through Result.Requeue.
				t.Errorf("Reconcile() Requeue = %v, want %v", got.Requeue, test.wantRequeue) //nolint:staticcheck // see above.
			}

			current := &v1alpha1.Group{}

			err = kube.Get(t.Context(), request.NamespacedName, current)
			if client.IgnoreNotFound(err) != nil {
				t.Fatalf("Get() unexpected error: %v", err)
			}

			if gotFinalizer := err == nil && slices.Contains(current.GetFinalizers(), managed.FinalizerName); gotFinalizer != test.wantFinalizer {
				t.Errorf("finalizer kept = %v, want %v", gotFinalizer, test.wantFinalizer)
			}
		})
	}
}

// TestValidatePollJitterPercentage checks the accepted range of the percentage.
func TestValidatePollJitterPercentage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		percentage int
		wantErr    bool
	}{
		"Zero":         {},
		"Default":      {percentage: 10},
		"Maximum":      {percentage: MaxPollJitterPercentage},
		"Negative":     {percentage: -1, wantErr: true},
		"AboveMaximum": {percentage: MaxPollJitterPercentage + 1, wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePollJitterPercentage(test.percentage)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidatePollJitterPercentage() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestPollJitterHook checks the bounds of the jittered interval, and that
// samples do not pile up on a single value, the minimum in particular.
func TestPollJitterHook(t *testing.T) {
	t.Parallel()

	const (
		samples     = 2000
		minDistinct = samples * 9 / 10
	)

	tests := map[string]struct {
		percentage      int
		minPollInterval time.Duration
		pollInterval    time.Duration
		wantMin         time.Duration
		wantMax         time.Duration
	}{
		"SpreadsInBothDirections": {
			percentage:   10,
			pollInterval: time.Minute,
			wantMin:      54 * time.Second,
			wantMax:      66 * time.Second,
		},
		"ScalesWithTheInterval": {
			percentage:   10,
			pollInterval: time.Hour,
			wantMin:      54 * time.Minute,
			wantMax:      66 * time.Minute,
		},
		"NeverGoesBelowTheMinimum": {
			percentage:      10,
			minPollInterval: 30 * time.Second,
			pollInterval:    30 * time.Second,
			wantMin:         30 * time.Second,
			wantMax:         33 * time.Second,
		},
		"KeepsAnIntervalBelowTheMinimum": {
			percentage:      10,
			minPollInterval: 30 * time.Second,
			pollInterval:    10 * time.Second,
			wantMin:         10 * time.Second,
			wantMax:         11 * time.Second,
		},
		"MaximumPercentageStaysPositive": {
			percentage:   MaxPollJitterPercentage,
			pollInterval: time.Minute,
			wantMin:      30 * time.Second,
			wantMax:      90 * time.Second,
		},
		"SaturatesInsteadOfOverflowing": {
			percentage:      10,
			minPollInterval: 30 * time.Second,
			pollInterval:    math.MaxInt64 - time.Hour,
			wantMin:         math.MaxInt64 / 2,
			wantMax:         math.MaxInt64,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			hook := pollJitterHook(test.percentage, test.minPollInterval)
			seen := make(map[time.Duration]struct{}, samples)

			for range samples {
				got := hook(nil, test.pollInterval)
				if got < test.wantMin || got > test.wantMax {
					t.Fatalf("hook() = %v, want between %v and %v", got, test.wantMin, test.wantMax)
				}

				seen[got] = struct{}{}
			}

			if len(seen) < minDistinct {
				t.Errorf("hook() returned %d distinct intervals out of %d samples, want at least %d", len(seen), samples, minDistinct)
			}
		})
	}
}

// TestPollJitterHookKeepsUnspreadableIntervals checks that intervals with
// nothing to spread are returned as they are.
func TestPollJitterHookKeepsUnspreadableIntervals(t *testing.T) {
	t.Parallel()

	hook := pollJitterHook(10, 30*time.Second)

	for _, pollInterval := range []time.Duration{-time.Minute, 0, time.Nanosecond} {
		if got := hook(nil, pollInterval); got != pollInterval {
			t.Errorf("hook(%v) = %v, want it unchanged", pollInterval, got)
		}
	}
}

// TestReconcilerOptionsApplyPollJitter checks the requeue delay of a real
// reconcile when jitter is enabled: an exact interval means the hook is not
// wired, and a minimum that is not passed on lets jitter go below it.
func TestReconcilerOptionsApplyPollJitter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		annotation string
		exact      time.Duration
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		"SpreadsThePollInterval": {
			exact:   time.Minute,
			wantMin: 54 * time.Second,
			wantMax: 66 * time.Second,
		},
		"SpreadsTheAnnotation": {
			annotation: "1h",
			exact:      time.Hour,
			wantMin:    54 * time.Minute,
			wantMax:    66 * time.Minute,
		},
		"StaysAboveTheMinimum": {
			annotation: "1s",
			exact:      30 * time.Second,
			wantMin:    30 * time.Second,
			wantMax:    33 * time.Second,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			group := &v1alpha1.Group{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: "team"}}
			if test.annotation != "" {
				meta.AddAnnotations(group, map[string]string{meta.AnnotationKeyPollInterval: test.annotation})
			}

			opts := Options{
				Options: controller.Options{
					Logger:       logging.NewNopLogger(),
					Features:     &feature.Flags{},
					PollInterval: time.Minute,
				},
				MinPollInterval:      30 * time.Second,
				PollJitterPercentage: 10,
			}
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "group", Namespace: "team"}}

			got, err := newTestReconciler(t, opts, group).Reconcile(t.Context(), request)
			if err != nil {
				t.Fatalf("Reconcile() unexpected error: %v", err)
			}

			if got.RequeueAfter < test.wantMin || got.RequeueAfter > test.wantMax {
				t.Errorf("Reconcile() RequeueAfter = %v, want between %v and %v", got.RequeueAfter, test.wantMin, test.wantMax)
			}

			if got.RequeueAfter == test.exact {
				t.Errorf("Reconcile() RequeueAfter = %v, want it spread", got.RequeueAfter)
			}
		})
	}
}

// gateName returns the SonarQube identifier of a QualityGate.
func gateName(gate *instancev1alpha1.QualityGate) string {
	return gate.Spec.ForProvider.Name
}

// newGateClient returns a Kubernetes client holding the supplied QualityGate.
func newGateClient(t *testing.T, gate *instancev1alpha1.QualityGate) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()

	err := instancev1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() unexpected error: %v", err)
	}

	return clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(gate).WithStatusSubresource(gate).Build()
}

// TestSpecExternalNameInitializer checks the external name left on a
// managed resource by the initializer.
func TestSpecExternalNameInitializer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		externalName string
		want         string
	}{
		"UsesTheSpecRatherThanTheResourceName": {
			want: "sonar-way",
		},
		"KeepsAnExistingExternalName": {
			externalName: "imported",
			want:         "imported",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gate := &instancev1alpha1.QualityGate{ObjectMeta: metav1.ObjectMeta{Name: "gate", Namespace: "team"}}
			gate.Spec.ForProvider.Name = "sonar-way"
			meta.SetExternalName(gate, test.externalName)

			kube := newGateClient(t, gate)

			err := newSpecExternalNameInitializer(kube, gateName).Initialize(t.Context(), gate)
			if err != nil {
				t.Fatalf("Initialize() unexpected error: %v", err)
			}

			persisted := &instancev1alpha1.QualityGate{}

			err = kube.Get(t.Context(), client.ObjectKeyFromObject(gate), persisted)
			if err != nil {
				t.Fatalf("Get() unexpected error: %v", err)
			}

			if got := meta.GetExternalName(persisted); got != test.want {
				t.Errorf("external name = %q, want %q", got, test.want)
			}
		})
	}
}

// TestSpecExternalNameInitializerRejectsOtherKinds checks that a resource of
// another kind is reported rather than given a wrong external name.
func TestSpecExternalNameInitializerRejectsOtherKinds(t *testing.T) {
	t.Parallel()

	initializer := newSpecExternalNameInitializer(nil, gateName)

	err := initializer.Initialize(t.Context(), &instancev1alpha1.Project{})
	if err == nil {
		t.Fatal("Initialize() expected an error for another kind")
	}
}

// TestDeterministicExternalNameRecoversInterruptedCreate reconciles a resource
// whose Create was interrupted before its result was recorded, as after a
// provider crash. It must find the resource created by that attempt instead of
// staying blocked or creating it again.
func TestDeterministicExternalNameRecoversInterruptedCreate(t *testing.T) {
	t.Parallel()

	gate := &instancev1alpha1.QualityGate{ObjectMeta: metav1.ObjectMeta{Name: "gate", Namespace: "team"}}
	gate.Spec.ForProvider.Name = "sonar-way"
	meta.SetExternalCreatePending(gate, time.Now())

	kube := newGateClient(t, gate)

	var observed, created string

	connector := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(_ context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
				observed = meta.GetExternalName(mg)

				return managed.ExternalObservation{ResourceExists: observed == "sonar-way", ResourceUpToDate: true}, nil
			},
			CreateFn: func(_ context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
				created = meta.GetExternalName(mg)

				return managed.ExternalCreation{}, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	mgr := recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: kube.Scheme()}}
	opts := Options{Options: controller.Options{Logger: logging.NewNopLogger(), Features: &feature.Flags{}, PollInterval: time.Minute}}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.QualityGateGroupVersionKind),
		ReconcilerOptions(mgr, "test", opts,
			managed.WithExternalConnector(connector),
			WithSpecExternalName(kube, gateName),
		)...)

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "gate", Namespace: "team"}}

	_, err := reconciler.Reconcile(t.Context(), request)
	if err != nil {
		t.Fatalf("Reconcile() unexpected error: %v", err)
	}

	if observed != "sonar-way" {
		t.Errorf("Observe() saw external name %q, want %q", observed, "sonar-way")
	}

	if created != "" {
		t.Errorf("Create() was called again with external name %q", created)
	}
}

// TestExternalNameSetByObservePersistedBeforeCreate pins the reconciler
// behaviour that the kinds with a name built from resolved references rely on:
// an external name set by Observe reaches the API server with the pending
// annotation, before Create runs, so it survives an interrupted Create.
func TestExternalNameSetByObservePersistedBeforeCreate(t *testing.T) {
	t.Parallel()

	gate := &instancev1alpha1.QualityGate{ObjectMeta: metav1.ObjectMeta{Name: "gate", Namespace: "team"}}

	kube := newGateClient(t, gate)

	var persisted string

	connector := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(_ context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
				meta.SetExternalName(mg, "group:devs")

				return managed.ExternalObservation{}, nil
			},
			CreateFn: func(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
				stored := &instancev1alpha1.QualityGate{}

				err := kube.Get(ctx, client.ObjectKeyFromObject(mg), stored)
				if err != nil {
					return managed.ExternalCreation{}, err
				}

				persisted = meta.GetExternalName(stored)

				return managed.ExternalCreation{}, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	mgr := recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: kube.Scheme()}}
	opts := Options{Options: controller.Options{Logger: logging.NewNopLogger(), Features: &feature.Flags{}, PollInterval: time.Minute}}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.QualityGateGroupVersionKind),
		ReconcilerOptions(mgr, "test", opts, managed.WithExternalConnector(connector))...)

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "gate", Namespace: "team"}}

	_, err := reconciler.Reconcile(t.Context(), request)
	if err != nil {
		t.Fatalf("Reconcile() unexpected error: %v", err)
	}

	if persisted != "group:devs" {
		t.Errorf("external name persisted before Create = %q, want %q", persisted, "group:devs")
	}
}
