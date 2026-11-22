package repository

import (
	"context"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

type RateLimitRepository interface {
	Hit(ctx context.Context, scope domain.RateLimitScope, key string, window time.Duration) (domain.RateLimitHit, error)
	Release(ctx context.Context, scope domain.RateLimitScope, key string) error
	DeleteExpired(ctx context.Context) error
}
