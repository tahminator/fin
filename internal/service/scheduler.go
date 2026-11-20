package service

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/robfig/cron/v3"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
)

type connectionSyncer interface {
	Sync(ctx context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) error
}

type SyncScheduler struct {
	schedule    cron.Schedule
	connections repository.ConnectionRepository
	syncer      connectionSyncer
	recorder    metrics.Recorder
	logger      *slog.Logger
	now         func() time.Time
}

func NewSyncScheduler(
	schedule cron.Schedule,
	connections repository.ConnectionRepository,
	syncer connectionSyncer,
	recorder metrics.Recorder,
	logger *slog.Logger,
) *SyncScheduler {
	return &SyncScheduler{
		schedule:    schedule,
		connections: connections,
		syncer:      syncer,
		recorder:    recorder,
		logger:      logger,
		now:         time.Now,
	}
}

func (s *SyncScheduler) Run(ctx context.Context) {
	if s.schedule == nil {
		return
	}

	for {
		wait := s.schedule.Next(s.now()).Sub(s.now())
		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()

			return
		case <-timer.C:
			s.RunOnce(ctx)
		}
	}
}

func (s *SyncScheduler) RunOnce(ctx context.Context) {
	started := s.now()

	connections, err := s.connections.ListByStatus(ctx, domain.ConnectionStatusActive)
	if err != nil {
		s.logger.ErrorContext(ctx, "scheduled sync: list connections", "err", err)
		s.finish(metrics.ScheduledRunFailure, 0, started)

		return
	}

	failed := 0

	for _, connection := range connections {
		err := s.syncer.Sync(ctx, connection.UserID, connection.ID, metrics.SyncTriggerScheduled)
		if err != nil {
			failed++
			s.logger.WarnContext(ctx, "scheduled sync: connection failed", "connection_id", connection.ID, "err", err)
		}
	}

	outcome := metrics.ScheduledRunSuccess

	if failed > 0 {
		outcome = metrics.ScheduledRunPartialFailure
	}

	s.finish(outcome, failed, started)
}

func (s *SyncScheduler) finish(outcome metrics.ScheduledRunOutcome, failed int, started time.Time) {
	finished := s.now()

	s.recorder.ScheduledRunFinished(metrics.ScheduledRun{
		Outcome:           outcome,
		FailedConnections: failed,
		Elapsed:           finished.Sub(started),
		FinishedAt:        finished,
	})
}
