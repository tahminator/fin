package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type SummaryRepository interface {
	Balances(ctx context.Context, userID uuid.UUID, excludeAccountIDs []uuid.UUID, excludeAccountTypes []domain.AccountType) ([]domain.CurrencyBalance, error)
	Counts(ctx context.Context, userID uuid.UUID) (domain.SummaryCounts, error)
	CashFlow(ctx context.Context, userID uuid.UUID, days int, excludeAccountIDs []uuid.UUID, excludeAccountTypes []domain.AccountType) ([]domain.DailyCashFlow, error)
}
