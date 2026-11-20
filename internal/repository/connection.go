package repository

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type ConnectionRepository interface {
	Create(ctx context.Context, connection domain.Connection) (domain.Connection, error)
	Get(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Connection, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Connection, int, error)
	ListByStatus(ctx context.Context, status domain.ConnectionStatus) ([]domain.Connection, error)
	ListByUserAndStatus(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus, limit, offset int) ([]domain.Connection, int, error)
	UpdateSyncState(ctx context.Context, userID, id uuid.UUID, cursor string, syncedAt time.Time) error
	UpdateLastError(ctx context.Context, userID, id uuid.UUID, lastError string) error
	UpdateStatus(ctx context.Context, userID, id uuid.UUID, status domain.ConnectionStatus, lastError *string) error
	UpdateInstitutionURL(ctx context.Context, userID, id uuid.UUID, url string) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
