package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type TransactionFilter struct {
	AccountID        *uuid.UUID
	ExcludeAccountID []uuid.UUID
}

type TransactionRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID, filter TransactionFilter, limit, offset int) ([]domain.Transaction, int, error)
	Upsert(ctx context.Context, userID uuid.UUID, transactions []domain.Transaction) error
	DeleteByExternalIDs(ctx context.Context, userID uuid.UUID, externalIDs []string) error
}
