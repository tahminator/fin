package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type fakeRateLimits struct {
	repository.RateLimitRepository
	hit      domain.RateLimitHit
	err      error
	hits     []string
	released []string
	sweeps   int
}

func (f *fakeRateLimits) Hit(_ context.Context, scope domain.RateLimitScope, key string, _ time.Duration) (domain.RateLimitHit, error) {
	f.hits = append(f.hits, string(scope)+":"+key)

	return f.hit, f.err
}

func (f *fakeRateLimits) Release(_ context.Context, scope domain.RateLimitScope, key string) error {
	f.released = append(f.released, string(scope)+":"+key)

	return nil
}

func (f *fakeRateLimits) DeleteExpired(context.Context) error {
	f.sweeps++

	return nil
}

var testLimit = domain.RateLimit{Scope: domain.RateLimitScopeSync, Max: 2, Window: time.Minute}

func TestRateLimiterAllowsUpToMax(t *testing.T) {
	tests := []struct {
		hits        int
		wantAllowed bool
	}{
		{hits: 1, wantAllowed: true},
		{hits: 2, wantAllowed: true},
		{hits: 3, wantAllowed: false},
	}

	for _, tt := range tests {
		limits := &fakeRateLimits{hit: domain.RateLimitHit{Hits: tt.hits, RetryAfter: 30 * time.Second}}
		limiter := NewRateLimiter(limits)

		decision, err := limiter.Allow(t.Context(), testLimit, "k")
		if err != nil {
			t.Fatalf("Allow: %v", err)
		}

		if decision.Allowed != tt.wantAllowed {
			t.Errorf("hits %d: allowed = %v, want %v", tt.hits, decision.Allowed, tt.wantAllowed)
		}

		if !tt.wantAllowed && decision.RetryAfter != 30*time.Second {
			t.Errorf("hits %d: RetryAfter = %s, want 30s", tt.hits, decision.RetryAfter)
		}
	}
}

func TestRateLimiterDisabledLimitSkipsRepository(t *testing.T) {
	limits := &fakeRateLimits{hit: domain.RateLimitHit{Hits: 99}}
	limiter := NewRateLimiter(limits)
	disabled := domain.RateLimit{Scope: domain.RateLimitScopeSync, Max: 1}

	decision, err := limiter.Allow(t.Context(), disabled, "k")
	if err != nil || !decision.Allowed {
		t.Fatalf("Allow = %+v, %v; want allowed", decision, err)
	}

	if err := limiter.Stamp(t.Context(), disabled, "k"); err != nil {
		t.Fatalf("Stamp: %v", err)
	}

	if err := limiter.Release(t.Context(), disabled, "k"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if len(limits.hits) != 0 || len(limits.released) != 0 {
		t.Errorf("repository used for a disabled limit: hits %v released %v", limits.hits, limits.released)
	}
}

func TestRateLimiterRepositoryErrorIsNeverAllowed(t *testing.T) {
	limits := &fakeRateLimits{err: errors.New("database down")}
	limiter := NewRateLimiter(limits)

	decision, err := limiter.Allow(t.Context(), testLimit, "k")
	if !errors.Is(err, limits.err) {
		t.Fatalf("Allow error = %v, want %v", err, limits.err)
	}

	if decision.Allowed {
		t.Error("decision allowed after a repository error")
	}
}

func TestRateLimiterStampNeverRefusesAndReleaseDeletes(t *testing.T) {
	limits := &fakeRateLimits{hit: domain.RateLimitHit{Hits: 50}}
	limiter := NewRateLimiter(limits)

	if err := limiter.Stamp(t.Context(), testLimit, "k"); err != nil {
		t.Fatalf("Stamp: %v", err)
	}

	if err := limiter.Release(t.Context(), testLimit, "k"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if len(limits.hits) != 1 || len(limits.released) != 1 || limits.released[0] != "sync:k" {
		t.Errorf("hits %v released %v", limits.hits, limits.released)
	}
}

func TestRateLimiterSweepsAtMostEveryInterval(t *testing.T) {
	limits := &fakeRateLimits{hit: domain.RateLimitHit{Hits: 1}}
	limiter := NewRateLimiter(limits)
	now := fixedNow
	limiter.now = func() time.Time { return now }

	for range 3 {
		if _, err := limiter.Allow(t.Context(), testLimit, "k"); err != nil {
			t.Fatalf("Allow: %v", err)
		}
	}

	if limits.sweeps != 1 {
		t.Fatalf("sweeps = %d after three calls at once, want 1", limits.sweeps)
	}

	now = now.Add(rateLimitSweepInterval)

	if _, err := limiter.Allow(t.Context(), testLimit, "k"); err != nil {
		t.Fatalf("Allow: %v", err)
	}

	if limits.sweeps != 2 {
		t.Errorf("sweeps = %d after the interval passed, want 2", limits.sweeps)
	}
}
