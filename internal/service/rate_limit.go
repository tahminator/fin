package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const rateLimitSweepInterval = 10 * time.Minute

type RateLimiter struct {
	limits    repository.RateLimitRepository
	now       func() time.Time
	mu        sync.Mutex
	lastSweep time.Time
}

func NewRateLimiter(limits repository.RateLimitRepository) *RateLimiter {
	return &RateLimiter{limits: limits, now: time.Now}
}

func (l *RateLimiter) Allow(ctx context.Context, limit domain.RateLimit, key string) (domain.RateLimitDecision, error) {
	if !limit.Enabled() {
		return domain.RateLimitDecision{Allowed: true}, nil
	}

	hit, err := l.limits.Hit(ctx, limit.Scope, key, limit.Window)
	if err != nil {
		return domain.RateLimitDecision{}, err
	}

	l.sweep(ctx)

	if hit.Hits <= limit.Max {
		return domain.RateLimitDecision{Allowed: true}, nil
	}

	return domain.RateLimitDecision{Allowed: false, RetryAfter: hit.RetryAfter}, nil
}

func (l *RateLimiter) Stamp(ctx context.Context, limit domain.RateLimit, key string) error {
	if !limit.Enabled() {
		return nil
	}

	_, err := l.limits.Hit(ctx, limit.Scope, key, limit.Window)

	return err
}

func (l *RateLimiter) Release(ctx context.Context, limit domain.RateLimit, key string) error {
	if !limit.Enabled() {
		return nil
	}

	return l.limits.Release(ctx, limit.Scope, key)
}

func (l *RateLimiter) sweep(ctx context.Context) {
	l.mu.Lock()

	now := l.now()
	due := now.Sub(l.lastSweep) >= rateLimitSweepInterval

	if due {
		l.lastSweep = now
	}

	l.mu.Unlock()

	if !due {
		return
	}

	if err := l.limits.DeleteExpired(ctx); err != nil {
		slog.WarnContext(ctx, "delete expired rate limits", "err", err)
	}
}
