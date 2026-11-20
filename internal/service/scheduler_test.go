package service

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
)

type syncRequest struct {
	userID       uuid.UUID
	connectionID uuid.UUID
	trigger      metrics.SyncTrigger
}

type fakeSyncer struct {
	requests []syncRequest
	failures map[uuid.UUID]error
}

func (f *fakeSyncer) Sync(_ context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) error {
	f.requests = append(f.requests, syncRequest{userID: userID, connectionID: connectionID, trigger: trigger})

	return f.failures[connectionID]
}

func steppingClock() func() time.Time {
	current := fixedNow

	return func() time.Time {
		now := current
		current = current.Add(time.Second)

		return now
	}
}

func newTestScheduler(connections repository.ConnectionRepository, syncer *fakeSyncer, recorder *scheduledRunRecorder) *SyncScheduler {
	scheduler := NewSyncScheduler(nil, connections, syncer, recorder, slog.New(slog.DiscardHandler))
	scheduler.now = steppingClock()

	return scheduler
}

func activeConnections(count int) []domain.Connection {
	connections := make([]domain.Connection, 0, count)

	for range count {
		connections = append(connections, domain.Connection{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Status: domain.ConnectionStatusActive,
		})
	}

	return connections
}

func TestRunOnceSyncsEveryActiveConnection(t *testing.T) {
	errSync := errors.New("sync failed")

	tests := []struct {
		name        string
		connections int
		failing     []int
		wantOutcome metrics.ScheduledRunOutcome
	}{
		{name: "no connections", connections: 0, wantOutcome: metrics.ScheduledRunSuccess},
		{name: "all succeed", connections: 3, wantOutcome: metrics.ScheduledRunSuccess},
		{name: "first fails and the rest still sync", connections: 3, failing: []int{0}, wantOutcome: metrics.ScheduledRunPartialFailure},
		{name: "middle fails", connections: 4, failing: []int{1}, wantOutcome: metrics.ScheduledRunPartialFailure},
		{name: "several fail", connections: 4, failing: []int{0, 2, 3}, wantOutcome: metrics.ScheduledRunPartialFailure},
		{name: "all fail", connections: 2, failing: []int{0, 1}, wantOutcome: metrics.ScheduledRunPartialFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listed := activeConnections(tt.connections)
			syncer := &fakeSyncer{failures: make(map[uuid.UUID]error)}
			recorder := &scheduledRunRecorder{}

			for _, index := range tt.failing {
				syncer.failures[listed[index].ID] = errSync
			}

			var statuses []domain.ConnectionStatus

			connections := &fakeConnections{
				listByStatus: func(_ context.Context, status domain.ConnectionStatus) ([]domain.Connection, error) {
					statuses = append(statuses, status)

					return listed, nil
				},
			}

			newTestScheduler(connections, syncer, recorder).RunOnce(t.Context())

			if !slices.Equal(statuses, []domain.ConnectionStatus{domain.ConnectionStatusActive}) {
				t.Errorf("ListByStatus statuses = %v, want [active]", statuses)
			}

			want := make([]syncRequest, 0, len(listed))

			for _, connection := range listed {
				want = append(want, syncRequest{userID: connection.UserID, connectionID: connection.ID, trigger: metrics.SyncTriggerScheduled})
			}

			if !slices.Equal(syncer.requests, want) {
				t.Errorf("sync requests = %+v, want %+v", syncer.requests, want)
			}

			wantRun := metrics.ScheduledRun{
				Outcome:           tt.wantOutcome,
				FailedConnections: len(tt.failing),
				Elapsed:           time.Second,
				FinishedAt:        fixedNow.Add(time.Second),
			}

			if !slices.Equal(recorder.runs, []metrics.ScheduledRun{wantRun}) {
				t.Errorf("scheduled runs = %+v, want [%+v]", recorder.runs, wantRun)
			}
		})
	}
}

func TestRunOnceListFailureRecordsFailure(t *testing.T) {
	syncer := &fakeSyncer{}
	recorder := &scheduledRunRecorder{}

	connections := &fakeConnections{
		listByStatus: func(context.Context, domain.ConnectionStatus) ([]domain.Connection, error) {
			return nil, errors.New("database unavailable")
		},
	}

	newTestScheduler(connections, syncer, recorder).RunOnce(t.Context())

	if len(syncer.requests) != 0 {
		t.Errorf("sync requests = %+v, want none", syncer.requests)
	}

	wantRun := metrics.ScheduledRun{
		Outcome:           metrics.ScheduledRunFailure,
		FailedConnections: 0,
		Elapsed:           time.Second,
		FinishedAt:        fixedNow.Add(time.Second),
	}

	if !slices.Equal(recorder.runs, []metrics.ScheduledRun{wantRun}) {
		t.Errorf("scheduled runs = %+v, want [%+v]", recorder.runs, wantRun)
	}
}

func TestRunWithNilScheduleReturnsImmediately(t *testing.T) {
	syncer := &fakeSyncer{}
	recorder := &scheduledRunRecorder{}

	newTestScheduler(&fakeConnections{}, syncer, recorder).Run(context.Background())

	if len(syncer.requests) != 0 || len(recorder.runs) != 0 {
		t.Errorf("nil schedule ran work: requests %+v, runs %+v", syncer.requests, recorder.runs)
	}
}
