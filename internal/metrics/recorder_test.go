package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type counterDelta struct {
	name    string
	counter prometheus.Counter
	before  float64
	want    float64
}

func watch(name string, counter prometheus.Counter, want float64) *counterDelta {
	return &counterDelta{name: name, counter: counter, before: testutil.ToFloat64(counter), want: want}
}

func assertDeltas(t *testing.T, deltas ...*counterDelta) {
	t.Helper()

	for _, d := range deltas {
		if got := testutil.ToFloat64(d.counter) - d.before; got != d.want {
			t.Errorf("%s: got delta %v, want %v", d.name, got, d.want)
		}
	}
}

func TestConnectionSyncedIncrementsOnlyItsSeries(t *testing.T) {
	target := watch(`syncs{trigger="scheduled",outcome="error"}`, syncsTotal.WithLabelValues("scheduled", "error"), 1)
	sameTrigger := watch(`syncs{trigger="scheduled",outcome="success"}`, syncsTotal.WithLabelValues("scheduled", "success"), 0)
	sameOutcome := watch(`syncs{trigger="manual",outcome="error"}`, syncsTotal.WithLabelValues("manual", "error"), 0)

	Prometheus{}.ConnectionSynced(SyncTriggerScheduled, SyncOutcomeError, 2*time.Second)

	assertDeltas(t, target, sameTrigger, sameOutcome)

	if got := testutil.CollectAndCount(syncsTotal); got != 24 {
		t.Fatalf("got %d sync series, want 24", got)
	}
}

func TestHTTPRequestServedIncrementsGroupMethodAndStatusClass(t *testing.T) {
	target := watch(`http{group="api",method="post",status_class="4xx"}`, httpRequestsTotal.WithLabelValues("api", "post", "4xx"), 1)
	otherClass := watch(`http{group="api",method="post",status_class="2xx"}`, httpRequestsTotal.WithLabelValues("api", "post", "2xx"), 0)
	otherMethod := watch(`http{group="api",method="get",status_class="4xx"}`, httpRequestsTotal.WithLabelValues("api", "get", "4xx"), 0)

	Prometheus{}.HTTPRequestServed(RouteGroupAPI, MethodPost, 404, time.Millisecond)

	assertDeltas(t, target, otherClass, otherMethod)
}

func TestLoginAttemptedIncrementsOutcome(t *testing.T) {
	target := watch(`logins{outcome="not_allowed"}`, loginsTotal.WithLabelValues("not_allowed"), 1)
	other := watch(`logins{outcome="success"}`, loginsTotal.WithLabelValues("success"), 0)

	Prometheus{}.LoginAttempted(LoginNotAllowed)

	assertDeltas(t, target, other)
}

func TestProviderCalledIncrementsOperationAndOutcome(t *testing.T) {
	target := watch(`provider{operation="exchange",outcome="error"}`, providerRequestsTotal.WithLabelValues("plaid", "exchange", "error"), 1)
	other := watch(`provider{operation="exchange",outcome="success"}`, providerRequestsTotal.WithLabelValues("plaid", "exchange", "success"), 0)

	Prometheus{}.ProviderCalled(ProviderPlaid, OperationExchange, CallError, time.Millisecond)

	assertDeltas(t, target, other)
}

func TestScheduledRunFinished(t *testing.T) {
	successAt := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)
	partialAt := successAt.Add(time.Hour)
	failureAt := partialAt.Add(time.Hour)

	steps := []struct {
		name        string
		run         ScheduledRun
		wantSuccess time.Time
	}{
		{
			name:        "success sets every gauge",
			run:         ScheduledRun{Outcome: ScheduledRunSuccess, FailedConnections: 0, Elapsed: time.Second, FinishedAt: successAt},
			wantSuccess: successAt,
		},
		{
			name:        "partial failure leaves success timestamp unchanged",
			run:         ScheduledRun{Outcome: ScheduledRunPartialFailure, FailedConnections: 2, Elapsed: time.Second, FinishedAt: partialAt},
			wantSuccess: successAt,
		},
		{
			name:        "failure leaves success timestamp unchanged",
			run:         ScheduledRun{Outcome: ScheduledRunFailure, FailedConnections: 5, Elapsed: time.Second, FinishedAt: failureAt},
			wantSuccess: successAt,
		},
	}

	for _, step := range steps {
		outcome := watch(step.name+": scheduled_runs{outcome="+step.run.Outcome.String()+"}", scheduledRunsTotal.WithLabelValues(step.run.Outcome.String()), 1)

		Prometheus{}.ScheduledRunFinished(step.run)

		assertDeltas(t, outcome)

		if got, want := testutil.ToFloat64(scheduledRunLastTimestamp), float64(step.run.FinishedAt.Unix()); got != want {
			t.Errorf("%s: last run timestamp: got %v, want %v", step.name, got, want)
		}

		if got, want := testutil.ToFloat64(scheduledRunFailedConnections), float64(step.run.FailedConnections); got != want {
			t.Errorf("%s: failed connections: got %v, want %v", step.name, got, want)
		}

		if got, want := testutil.ToFloat64(scheduledRunLastSuccessTimestamp), float64(step.wantSuccess.Unix()); got != want {
			t.Errorf("%s: last success timestamp: got %v, want %v", step.name, got, want)
		}
	}
}
