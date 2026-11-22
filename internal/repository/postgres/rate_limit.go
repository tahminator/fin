package postgres

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type RateLimitRepository struct {
	pool *pgxpool.Pool
}

func NewRateLimitRepository(pool *pgxpool.Pool) *RateLimitRepository {
	return &RateLimitRepository{pool: pool}
}

const hitRateLimit = `
INSERT INTO rate_limits AS r (
    scope,
    key,
    window_started_at,
    expires_at,
    hits
)
VALUES ($1, $2, now(), now() + $3::float8 * interval '1 second', 1)
ON CONFLICT (scope, key) DO UPDATE
SET window_started_at = CASE WHEN r.expires_at <= now() THEN now() ELSE r.window_started_at END,
    expires_at        = CASE WHEN r.expires_at <= now() THEN now() + $3::float8 * interval '1 second' ELSE r.expires_at END,
    hits              = CASE WHEN r.expires_at <= now() THEN 1 ELSE r.hits + 1 END
RETURNING
    r.hits                                           AS hits,
    extract(epoch FROM r.expires_at - now())::float8 AS retry_after_seconds;
`

func (r *RateLimitRepository) Hit(ctx context.Context, scope domain.RateLimitScope, key string, window time.Duration) (domain.RateLimitHit, error) {
	var (
		hits       int
		retryAfter float64
	)

	err := r.pool.QueryRow(ctx, hitRateLimit, string(scope), key, window.Seconds()).Scan(&hits, &retryAfter)
	if err != nil {
		return domain.RateLimitHit{}, fmt.Errorf("hit rate limit %s: %w", scope, err)
	}

	return domain.RateLimitHit{
		Hits:       hits,
		RetryAfter: time.Duration(math.Ceil(max(retryAfter, 0))) * time.Second,
	}, nil
}

const releaseRateLimit = `
DELETE FROM rate_limits
WHERE scope = $1
  AND key = $2;
`

func (r *RateLimitRepository) Release(ctx context.Context, scope domain.RateLimitScope, key string) error {
	_, err := r.pool.Exec(ctx, releaseRateLimit, string(scope), key)
	if err != nil {
		return fmt.Errorf("release rate limit %s: %w", scope, err)
	}

	return nil
}

const deleteExpiredRateLimits = `
DELETE FROM rate_limits
WHERE expires_at <= now();
`

func (r *RateLimitRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, deleteExpiredRateLimits)
	if err != nil {
		return fmt.Errorf("delete expired rate limits: %w", err)
	}

	return nil
}
