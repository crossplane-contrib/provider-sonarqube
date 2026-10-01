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

package cache

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// metricsNamespace prefixes every metric of this package.
	metricsNamespace = "provider_sonarqube"
	// metricsSubsystem groups the metrics of this package.
	metricsSubsystem = "observe_cache"

	// labelNamespace is the metric label holding the Key.Namespace.
	labelNamespace = "namespace"
	// labelResult is the metric label holding the outcome of a request.
	labelResult = "result"

	// resultHit means the value was served from the cache.
	resultHit = "hit"
	// resultMiss means the value was fetched from SonarQube.
	resultMiss = "miss"
	// resultCoalesced means the value was fetched from SonarQube by a
	// concurrent caller and shared.
	resultCoalesced = "coalesced"
)

var (
	// requestsTotal counts cached reads by namespace and result.
	requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "requests_total",
		Help:      "Total number of observe cache reads, by dataset namespace and result (hit, miss or coalesced).",
	}, []string{labelNamespace, labelResult})

	// invalidationsTotal counts invalidations by namespace.
	invalidationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "invalidations_total",
		Help:      "Total number of observe cache invalidations, by dataset namespace.",
	}, []string{labelNamespace})

	// entriesGauge reports the number of entries held by the default Store,
	// computed at scrape time.
	entriesGauge = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "entries",
		Help:      "Number of entries currently held by the observe cache.",
	}, func() float64 { return float64(Default().Len()) })

	// registerMetricsOnce ensures the metrics are registered at most once.
	registerMetricsOnce sync.Once
)

// registerMetrics registers the metrics of this package with the
// controller-runtime metrics registry. It is only called once the cache is
// enabled, so a disabled cache exposes no metric.
func registerMetrics() {
	registerMetricsOnce.Do(func() {
		metrics.Registry.MustRegister(requestsTotal, invalidationsTotal, entriesGauge)
	})
}
