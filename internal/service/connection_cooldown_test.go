package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

const testCooldown = 10 * time.Minute

type fakeSyncLimiter struct {
	refuse     bool
	retryAfter time.Duration
	err        error
	calls      []string
	limits     []domain.RateLimit
}

func (l *fakeSyncLimiter) Allow(_ context.Context, limit domain.RateLimit, key string) (domain.RateLimitDecision, error) {
	l.calls = append(l.calls, "allow:"+key)
	l.limits = append(l.limits, limit)

	if l.err != nil {
		return domain.RateLimitDecision{}, l.err
	}

	return domain.RateLimitDecision{Allowed: !l.refuse, RetryAfter: l.retryAfter}, nil
}

func (l *fakeSyncLimiter) Stamp(_ context.Context, limit domain.RateLimit, key string) error {
	l.calls = append(l.calls, "stamp:"+key)
	l.limits = append(l.limits, limit)

	return l.err
}

func (l *fakeSyncLimiter) Release(_ context.Context, limit domain.RateLimit, key string) error {
	l.calls = append(l.calls, "release:"+key)
	l.limits = append(l.limits, limit)

	return nil
}

func withCooldown(f *connectionFixture) *fakeSyncLimiter {
	limiter := &fakeSyncLimiter{}
	f.service.cooldown = SyncCooldown{Limiter: limiter, Window: testCooldown}

	return limiter
}

func (f *connectionFixture) calledProvider() bool {
	return slices.ContainsFunc(f.calls, func(call string) bool {
		return strings.HasPrefix(call, "provider.")
	})
}

func TestManualSyncInsideCooldownIsThrottledWithoutProviderCall(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := withCooldown(f)
	limiter.refuse = true
	limiter.retryAfter = 4 * time.Minute

	_, err := f.service.SyncOne(t.Context(), f.userID, f.connection.ID)

	var throttled *domain.ThrottledError
	if !errors.As(err, &throttled) || !errors.Is(err, domain.ErrThrottled) {
		t.Fatalf("SyncOne error = %v, want ThrottledError", err)
	}

	if throttled.RetryAfter != 4*time.Minute {
		t.Errorf("RetryAfter = %s, want 4m", throttled.RetryAfter)
	}

	if f.calledProvider() {
		t.Errorf("provider called while throttled: %v", f.calls)
	}

	if !slices.Equal(limiter.calls, []string{"allow:" + f.connection.ID.String()}) {
		t.Errorf("limiter calls = %v", limiter.calls)
	}

	want := domain.RateLimit{Scope: domain.RateLimitScopeSync, Max: 1, Window: testCooldown}
	if limiter.limits[0] != want {
		t.Errorf("limit = %+v, want %+v", limiter.limits[0], want)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeThrottled})
}

func TestManualSyncOutsideCooldownRuns(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := withCooldown(f)

	if _, err := f.service.SyncOne(t.Context(), f.userID, f.connection.ID); err != nil {
		t.Fatalf("SyncOne: %v", err)
	}

	if !f.called("connections.updateSyncState") {
		t.Errorf("sync did not run: %v", f.calls)
	}

	if !slices.Equal(limiter.calls, []string{"allow:" + f.connection.ID.String()}) {
		t.Errorf("limiter calls = %v, want a single allow", limiter.calls)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeSuccess})
}

func TestNonManualSyncsStampCooldownAndAreNeverRefused(t *testing.T) {
	for _, trigger := range []metrics.SyncTrigger{
		metrics.SyncTriggerScheduled,
		metrics.SyncTriggerLink,
		metrics.SyncTriggerReauth,
		metrics.SyncTriggerAccountSelection,
	} {
		t.Run(trigger.String(), func(t *testing.T) {
			f := newConnectionFixture(t)
			limiter := withCooldown(f)
			limiter.refuse = true

			if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, trigger); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			if !slices.Equal(limiter.calls, []string{"stamp:" + f.connection.ID.String()}) {
				t.Errorf("limiter calls = %v, want a single stamp", limiter.calls)
			}

			f.assertSynced(t, syncedCall{trigger: trigger, outcome: metrics.SyncOutcomeSuccess})
		})
	}
}

func TestFailedSyncReleasesCooldownButReauthDoesNot(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantRelease bool
	}{
		{name: "provider failure", err: errProviderDown, wantRelease: true},
		{name: "reauth required", err: fmt.Errorf("plaid ITEM_LOGIN_REQUIRED: %w", domain.ErrReauthRequired), wantRelease: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)
			limiter := withCooldown(f)

			f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
				return nil, tt.err
			}

			_ = f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual)

			released := slices.Contains(limiter.calls, "release:"+f.connection.ID.String())
			if released != tt.wantRelease {
				t.Errorf("released = %v, want %v (calls %v)", released, tt.wantRelease, limiter.calls)
			}
		})
	}
}

func TestCooldownLimiterErrorFailsSyncWithoutProviderCall(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := withCooldown(f)
	limiter.err = errors.New("database down")

	err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual)
	if !errors.Is(err, limiter.err) {
		t.Fatalf("Sync error = %v, want %v", err, limiter.err)
	}

	if f.calledProvider() {
		t.Errorf("provider called after limiter error: %v", f.calls)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeError})
}

func TestZeroCooldownNeverCallsLimiter(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := &fakeSyncLimiter{refuse: true}
	f.service.cooldown = SyncCooldown{Limiter: limiter}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(limiter.calls) != 0 {
		t.Errorf("limiter calls = %v, want none", limiter.calls)
	}
}

func TestCooldownIsNotCheckedForAnotherUsersConnection(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := withCooldown(f)

	err := f.service.Sync(t.Context(), f.userID, f.accountID, metrics.SyncTriggerManual)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Sync error = %v, want %v", err, domain.ErrNotFound)
	}

	if len(limiter.calls) != 0 {
		t.Errorf("limiter calls = %v, want none for an unknown connection", limiter.calls)
	}
}

func TestSyncAllCountsThrottledConnections(t *testing.T) {
	f := newConnectionFixture(t)
	limiter := withCooldown(f)
	limiter.refuse = true

	f.connections.listByUserAndStatus = func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
		return []domain.Connection{f.connection}, 1, nil
	}

	summary, err := f.service.SyncAll(t.Context(), f.userID, metrics.SyncTriggerManual)
	if err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	if summary != (SyncSummary{Throttled: 1}) {
		t.Errorf("summary = %+v, want one throttled", summary)
	}
}
