package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type AccountRepository interface {
	Upsert(ctx context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error)
	DeleteExcept(ctx context.Context, userID, connectionID uuid.UUID, keepIDs []uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Account, int, error)
}
