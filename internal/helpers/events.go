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
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewEventRecorder returns a Crossplane event recorder for the named
// controller. crossplane-runtime's APIRecorder still requires the legacy
// record.EventRecorder, which controller-runtime only exposes through the
// deprecated GetEventRecorderFor. This is the single place where that
// deprecated call is made.
func NewEventRecorder(mgr ctrl.Manager, name string) event.Recorder {
	return event.NewAPIRecorder(mgr.GetEventRecorderFor(name)) //nolint:staticcheck // no non-deprecated way to obtain a legacy record.EventRecorder for crossplane-runtime.
}
