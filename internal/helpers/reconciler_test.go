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
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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
// that the options of the calling controller are kept.
func TestReconcilerOptions(t *testing.T) {
	t.Parallel()

	const commonOptions = 3

	specific := managed.WithPollInterval(time.Hour)

	tests := map[string]struct {
		features []feature.Flag
		metrics  *controller.MetricOptions
		specific []managed.ReconcilerOption
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
			specific: []managed.ReconcilerOption{specific, specific},
			want:     commonOptions + 2,
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

			got := ReconcilerOptions(recorderManager{}, "test", opts, test.specific...)
			if len(got) != test.want {
				t.Errorf("ReconcilerOptions() returned %d options, want %d", len(got), test.want)
			}
		})
	}
}
