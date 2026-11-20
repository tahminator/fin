package repository

import (
	"context"

	"github.com/tahminator/fin/internal/domain"
)

type MetricsRepository interface {
	Snapshot(ctx context.Context) (domain.MetricsSnapshot, error)
}
