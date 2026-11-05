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
	"math"
	"math/rand/v2"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MaxPollJitterPercentage is the highest accepted Options.PollJitterPercentage.
// It keeps the poll interval well above zero, as a zero requeue delay would
// stop polling the resource altogether.
const MaxPollJitterPercentage = 50

// percent is the divisor of a percentage.
const percent = 100.0

// Options are the crossplane-runtime controller options, extended with the
// settings of this provider.
type Options struct {
	controller.Options

	// MinPollInterval is the shortest poll interval a resource may request
	// through the crossplane.io/poll-interval annotation. Shorter requests are
	// raised to it, which keeps the annotation from letting one resource use
	// the reconcile budget shared by all tenants.
	MinPollInterval time.Duration

	// PollJitterPercentage spreads each poll by up to this percentage of its
	// interval, in both directions, so that resources created or restarted
	// together do not keep polling SonarQube together. An interval close to
	// MinPollInterval is only spread as far down as that minimum. Zero disables
	// it.
	PollJitterPercentage int
}

// ValidateMinPollInterval returns an error if minimum is negative. A negative
// minimum would silently remove the limit, which only 0 is meant to do.
func ValidateMinPollInterval(minimum time.Duration) error {
	if minimum < 0 {
		return errors.Errorf("minimum poll interval must not be negative, got %s", minimum)
	}

	return nil
}

// ValidatePollJitterPercentage returns an error if percentage is negative or
// above MaxPollJitterPercentage.
func ValidatePollJitterPercentage(percentage int) error {
	if percentage < 0 || percentage > MaxPollJitterPercentage {
		return errors.Errorf("poll jitter percentage must be between 0 and %d, got %d", MaxPollJitterPercentage, percentage)
	}

	return nil
}

// ReconcilerOptions returns the managed reconciler options shared by every
// controller, followed by the options specific to the calling controller.
func ReconcilerOptions(mgr ctrl.Manager, name string, opts Options, specific ...managed.ReconcilerOption) []managed.ReconcilerOption {
	options := []managed.ReconcilerOption{
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithMinPollInterval(opts.MinPollInterval),
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

	if opts.PollJitterPercentage > 0 {
		options = append(options, managed.WithPollIntervalHook(pollJitterHook(opts.PollJitterPercentage, opts.MinPollInterval)))
	}

	return append(options, specific...)
}

// AddMRStateRecorder registers a recorder of the state metrics of the managed
// resources in list with mgr. It does nothing when state metrics are disabled.
func AddMRStateRecorder(mgr ctrl.Manager, opts Options, list resource.ManagedList) error {
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

// pollJitterHook returns a hook spreading the poll interval by a percentage of
// its value, which unlike the fixed jitter of managed.WithPollJitterHook scales
// with the per-resource intervals set through crossplane.io/poll-interval.
// Jitter never takes an interval that respects minPollInterval below it: the
// range is cut at the minimum rather than the result, so that intervals close
// to it do not all land on the same value. The upper bound saturates, because
// the annotation accepts intervals close to the largest duration.
func pollJitterHook(percentage int, minPollInterval time.Duration) managed.PollIntervalHook {
	fraction := float64(percentage) / percent

	return func(_ resource.Managed, pollInterval time.Duration) time.Duration {
		spread := time.Duration(float64(pollInterval) * fraction)
		if spread <= 0 {
			return pollInterval
		}

		lowest := max(pollInterval-spread, min(pollInterval, minPollInterval))
		highest := pollInterval + min(spread, math.MaxInt64-pollInterval)

		return lowest + rand.N(highest-lowest+1) //nolint:gosec // No need for secure randomness.
	}
}

// NewSpecExternalNameInitializer returns an initializer that sets the external
// name of a managed resource from its spec, instead of the default
// metadata.name. It is meant for the kinds whose SonarQube identifier is chosen
// by the spec: Observe then looks for the right resource before Create runs,
// which is what makes managed.WithDeterministicExternalName safe. Without it, a
// Create interrupted before its result was recorded would be sent again for a
// resource Observe cannot find.
//
// Initializers run before references are resolved, so externalName must only
// read fields that are set directly in the spec.
func NewSpecExternalNameInitializer[T resource.Managed](kube client.Client, externalName func(T) string) managed.Initializer {
	return managed.InitializerFn(func(ctx context.Context, managedResource resource.Managed) error {
		if meta.GetExternalName(managedResource) != "" {
			return nil
		}

		typed, ok := managedResource.(T)
		if !ok {
			return errors.Errorf("cannot initialize the external name of unexpected type %T", managedResource)
		}

		meta.SetExternalName(managedResource, externalName(typed))

		return errors.Wrap(kube.Update(ctx, managedResource), "cannot update the managed resource external name")
	})
}
