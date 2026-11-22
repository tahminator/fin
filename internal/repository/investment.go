package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type InvestmentFilter struct {
	AccountID         *uuid.UUID
	ExcludeAccountIDs []uuid.UUID
}

type InvestmentRepository interface {
	ReplaceHoldings(ctx context.Context, userID uuid.UUID, accountIDs map[string]uuid.UUID, securities []domain.InvestmentSecurity, holdings []domain.InvestmentHolding) error
	UpsertTransactions(ctx context.Context, userID uuid.UUID, accountIDs map[string]uuid.UUID, securities []domain.InvestmentSecurity, transactions []domain.InvestmentTransaction) error
	ListHoldingsByUser(ctx context.Context, userID uuid.UUID, filter InvestmentFilter) ([]domain.InvestmentHolding, error)
	ListTransactionsByUser(ctx context.Context, userID uuid.UUID, filter InvestmentFilter, limit, offset int) ([]domain.InvestmentTransaction, int, error)
}
