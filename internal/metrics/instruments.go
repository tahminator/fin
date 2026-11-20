package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var registry = newRegistry()

var (
	httpRequestsTotal = newCounterVec(
		"fin_http_requests_total",
		"HTTP requests served, by route group, method and status class.",
		[]string{"group", "method", "status_class"},
		labelValues(allRouteGroups()),
		labelValues(allMethods()),
		labelValues(allStatusClasses()),
	)

	httpRequestDuration = newHistogramVec(
		"fin_http_request_duration_seconds",
		"HTTP request latency, by route group and method.",
		[]string{"group", "method"},
		labelValues(allRouteGroups()),
		labelValues(allMethods()),
	)

	loginsTotal = newCounterVec(
		"fin_logins_total",
		"Google login attempts, by outcome.",
		[]string{"outcome"},
		labelValues(allLoginOutcomes()),
	)

	providerRequestsTotal = newCounterVec(
		"fin_provider_requests_total",
		"Calls to the account aggregation provider, by provider, operation and outcome.",
		[]string{"provider", "operation", "outcome"},
		labelValues(allProviders()),
		labelValues(allProviderOperations()),
		labelValues(allCallOutcomes()),
	)

	providerRequestDuration = newHistogramVec(
		"fin_provider_request_duration_seconds",
		"Latency of calls to the account aggregation provider.",
		[]string{"provider", "operation", "outcome"},
		labelValues(allProviders()),
		labelValues(allProviderOperations()),
		labelValues(allCallOutcomes()),
	)

	syncsTotal = newCounterVec(
		"fin_syncs_total",
		"Connection syncs, by trigger and outcome.",
		[]string{"trigger", "outcome"},
		labelValues(allSyncTriggers()),
		labelValues(allSyncOutcomes()),
	)

	syncDuration = newHistogramVec(
		"fin_sync_duration_seconds",
		"Connection sync latency, by trigger.",
		[]string{"trigger"},
		labelValues(allSyncTriggers()),
	)

	scheduledRunsTotal = newCounterVec(
		"fin_scheduled_sync_runs_total",
		"Scheduled sync runs, by outcome.",
		[]string{"outcome"},
		labelValues(allScheduledRunOutcomes()),
	)

	scheduledRunDuration = newHistogram(
		"fin_scheduled_sync_duration_seconds",
		"Duration of scheduled sync runs.",
	)

	scheduledRunFailedConnections = newGauge(
		"fin_scheduled_sync_last_run_failed_connections",
		"Connections that failed during the last scheduled sync run.",
	)

	scheduledRunLastTimestamp = newGauge(
		"fin_scheduled_sync_last_run_timestamp_seconds",
		"Unix time the last scheduled sync run finished.",
	)

	scheduledRunLastSuccessTimestamp = newGauge(
		"fin_scheduled_sync_last_success_timestamp_seconds",
		"Unix time the last fully successful scheduled sync run finished.",
	)
)

func newRegistry() *prometheus.Registry {
	r := prometheus.NewRegistry()

	r.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return r
}

func newCounterVec(name, help string, labels []string, values ...[]string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, labels)

	for _, combination := range combinations(values...) {
		vec.WithLabelValues(combination...)
	}

	registry.MustRegister(vec)

	return vec
}

func newHistogramVec(name, help string, labels []string, values ...[]string) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Help: help}, labels)

	for _, combination := range combinations(values...) {
		vec.WithLabelValues(combination...)
	}

	registry.MustRegister(vec)

	return vec
}

func newHistogram(name, help string) prometheus.Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.ExponentialBuckets(1, 2, 12),
	})

	registry.MustRegister(histogram)

	return histogram
}

func newGauge(name, help string) prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: name, Help: help})

	registry.MustRegister(gauge)

	return gauge
}

func labelValues[T fmt.Stringer](values []T) []string {
	labels := make([]string, len(values))

	for i, value := range values {
		labels[i] = value.String()
	}

	return labels
}

func combinations(sets ...[]string) [][]string {
	result := [][]string{{}}

	for _, set := range sets {
		var next [][]string

		for _, prefix := range result {
			for _, value := range set {
				next = append(next, append(append([]string{}, prefix...), value))
			}
		}

		result = next
	}

	return result
}
