package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type AccountFilter struct {
	ConnectionID *uuid.UUID
}

type AccountRepository interface {
	Upsert(ctx context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error)
	DeleteExcept(ctx context.Context, userID, connectionID uuid.UUID, keepIDs []uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, filter AccountFilter, limit, offset int) ([]domain.Account, int, error)
	MatchCardArt(ctx context.Context, userID, id uuid.UUID, art domain.CardArt) error
	SelectCardArt(ctx context.Context, userID, id uuid.UUID, art domain.CardArt) (domain.Account, error)
}
