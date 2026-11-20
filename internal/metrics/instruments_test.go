package metrics

import (
	"os"
	"slices"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type seriesSnapshot struct {
	count   int
	nonZero int
}

var initialSeries map[string]seriesSnapshot

func TestMain(m *testing.M) {
	initialSeries = snapshotSeries()

	os.Exit(m.Run())
}

func snapshotSeries() map[string]seriesSnapshot {
	r := prometheus.NewRegistry()
	r.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		loginsTotal,
		providerRequestsTotal,
		providerRequestDuration,
		syncsTotal,
		syncDuration,
		scheduledRunsTotal,
		scheduledRunDuration,
		scheduledRunFailedConnections,
		scheduledRunLastTimestamp,
		scheduledRunLastSuccessTimestamp,
	)

	families, err := r.Gather()
	if err != nil {
		panic(err)
	}

	snapshot := make(map[string]seriesSnapshot, len(families))

	for _, family := range families {
		series := seriesSnapshot{count: len(family.GetMetric())}

		for _, metric := range family.GetMetric() {
			if metric.GetCounter().GetValue() != 0 || metric.GetGauge().GetValue() != 0 || metric.GetHistogram().GetSampleCount() != 0 {
				series.nonZero++
			}
		}

		snapshot[family.GetName()] = series
	}

	return snapshot
}

var expectedSeries = []struct {
	name      string
	collector prometheus.Collector
	count     int
}{
	{name: "fin_http_requests_total", collector: httpRequestsTotal, count: 120},
	{name: "fin_http_request_duration_seconds", collector: httpRequestDuration, count: 24},
	{name: "fin_logins_total", collector: loginsTotal, count: 4},
	{name: "fin_provider_requests_total", collector: providerRequestsTotal, count: 12},
	{name: "fin_provider_request_duration_seconds", collector: providerRequestDuration, count: 12},
	{name: "fin_syncs_total", collector: syncsTotal, count: 12},
	{name: "fin_sync_duration_seconds", collector: syncDuration, count: 4},
	{name: "fin_scheduled_sync_runs_total", collector: scheduledRunsTotal, count: 3},
	{name: "fin_scheduled_sync_duration_seconds", collector: scheduledRunDuration, count: 1},
	{name: "fin_scheduled_sync_last_run_failed_connections", collector: scheduledRunFailedConnections, count: 1},
	{name: "fin_scheduled_sync_last_run_timestamp_seconds", collector: scheduledRunLastTimestamp, count: 1},
	{name: "fin_scheduled_sync_last_success_timestamp_seconds", collector: scheduledRunLastSuccessTimestamp, count: 1},
}

func TestEveryLabelCombinationIsPreInitializedToZero(t *testing.T) {
	if len(initialSeries) != len(expectedSeries) {
		t.Fatalf("got %d metric families at startup, want %d", len(initialSeries), len(expectedSeries))
	}

	for _, tt := range expectedSeries {
		t.Run(tt.name, func(t *testing.T) {
			initial, ok := initialSeries[tt.name]
			if !ok {
				t.Fatal("metric family missing at startup")
			}

			if initial.count != tt.count {
				t.Errorf("got %d series at startup, want %d", initial.count, tt.count)
			}

			if initial.nonZero != 0 {
				t.Errorf("got %d non-zero series at startup, want 0", initial.nonZero)
			}
		})
	}
}

func TestSeriesCountsMatchLabelCombinations(t *testing.T) {
	for _, tt := range expectedSeries {
		t.Run(tt.name, func(t *testing.T) {
			if got := testutil.CollectAndCount(tt.collector, tt.name); got != tt.count {
				t.Fatalf("got %d series, want %d", got, tt.count)
			}
		})
	}
}

func TestCombinationsIsCartesianProduct(t *testing.T) {
	got := combinations([]string{"a", "b"}, []string{"x", "y", "z"})
	want := [][]string{{"a", "x"}, {"a", "y"}, {"a", "z"}, {"b", "x"}, {"b", "y"}, {"b", "z"}}

	if !slices.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
