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
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ReconcilerOptions returns the managed reconciler options shared by every
// controller, followed by the options specific to the calling controller.
func ReconcilerOptions(mgr ctrl.Manager, name string, opts controller.Options, specific ...managed.ReconcilerOption) []managed.ReconcilerOption {
	options := []managed.ReconcilerOption{
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(NewEventRecorder(mgr, name, opts.EventFilterFunctions...)),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		options = append(options, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		options = append(options, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		options = append(options, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	return append(options, specific...)
}

// AddMRStateRecorder registers a recorder of the state metrics of the managed
// resources in list with mgr. It does nothing when state metrics are disabled.
func AddMRStateRecorder(mgr ctrl.Manager, opts controller.Options, list resource.ManagedList) error {
	if opts.MetricOptions == nil || opts.MetricOptions.MRStateMetrics == nil {
		return nil
	}

	recorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, list, opts.MetricOptions.PollStateMetricInterval,
	)

	err := mgr.Add(recorder)
	if err != nil {
		return errors.Wrapf(err, "cannot register MR state metrics recorder for kind %T", list)
	}

	return nil
}
