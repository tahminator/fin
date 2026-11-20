package metrics

import "time"

type ScheduledRun struct {
	Outcome           ScheduledRunOutcome
	FailedConnections int
	Elapsed           time.Duration
	FinishedAt        time.Time
}

type Recorder interface {
	HTTPRequestServed(group RouteGroup, method Method, status int, elapsed time.Duration)
	LoginAttempted(outcome LoginOutcome)
	ProviderCalled(provider Provider, operation ProviderOperation, outcome CallOutcome, elapsed time.Duration)
	ConnectionSynced(trigger SyncTrigger, outcome SyncOutcome, elapsed time.Duration)
	ScheduledRunFinished(run ScheduledRun)
}

type Prometheus struct{}

func (Prometheus) HTTPRequestServed(group RouteGroup, method Method, status int, elapsed time.Duration) {
	httpRequestsTotal.WithLabelValues(group.String(), method.String(), statusClassOf(status).String()).Inc()
	httpRequestDuration.WithLabelValues(group.String(), method.String()).Observe(elapsed.Seconds())
}

func (Prometheus) LoginAttempted(outcome LoginOutcome) {
	loginsTotal.WithLabelValues(outcome.String()).Inc()
}

func (Prometheus) ProviderCalled(provider Provider, operation ProviderOperation, outcome CallOutcome, elapsed time.Duration) {
	providerRequestsTotal.WithLabelValues(provider.String(), operation.String(), outcome.String()).Inc()
	providerRequestDuration.WithLabelValues(provider.String(), operation.String(), outcome.String()).Observe(elapsed.Seconds())
}

func (Prometheus) ConnectionSynced(trigger SyncTrigger, outcome SyncOutcome, elapsed time.Duration) {
	syncsTotal.WithLabelValues(trigger.String(), outcome.String()).Inc()
	syncDuration.WithLabelValues(trigger.String()).Observe(elapsed.Seconds())
}

func (Prometheus) ScheduledRunFinished(run ScheduledRun) {
	scheduledRunsTotal.WithLabelValues(run.Outcome.String()).Inc()
	scheduledRunDuration.Observe(run.Elapsed.Seconds())
	scheduledRunFailedConnections.Set(float64(run.FailedConnections))
	scheduledRunLastTimestamp.Set(unixSeconds(run.FinishedAt))

	if run.Outcome == ScheduledRunSuccess {
		scheduledRunLastSuccessTimestamp.Set(unixSeconds(run.FinishedAt))
	}
}

type Noop struct{}

func (Noop) HTTPRequestServed(RouteGroup, Method, int, time.Duration) {}

func (Noop) LoginAttempted(LoginOutcome) {}

func (Noop) ProviderCalled(Provider, ProviderOperation, CallOutcome, time.Duration) {}

func (Noop) ConnectionSynced(SyncTrigger, SyncOutcome, time.Duration) {}

func (Noop) ScheduledRunFinished(ScheduledRun) {}

func unixSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(time.Second)
}
