package repository

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type InvestmentFilter struct {
	AccountID        *uuid.UUID
	ExcludeAccountID []uuid.UUID
}

type InvestmentHistoryQuery struct {
	Filter      InvestmentFilter
	SecurityIDs []uuid.UUID
	From        *time.Time
	Bucket      time.Duration
}

type InvestmentRepository interface {
	UpsertSecurities(ctx context.Context, userID uuid.UUID, securities []domain.InvestmentSecurity) error
	ReplaceHoldings(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID, holdings []domain.InvestmentHolding) error
	ReplaceTransactions(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID, since time.Time, transactions []domain.InvestmentTransaction) error
	CaptureSnapshot(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID, capturedAt time.Time) error
	ListHoldings(ctx context.Context, userID uuid.UUID, filter InvestmentFilter) ([]domain.InvestmentHolding, error)
	ListTransactions(ctx context.Context, userID uuid.UUID, filter InvestmentFilter, limit, offset int) ([]domain.InvestmentTransaction, int, error)
	ListSecurities(ctx context.Context, userID uuid.UUID, filter InvestmentFilter) ([]domain.InvestmentSecuritySummary, error)
	HistoryBounds(ctx context.Context, userID uuid.UUID, filter InvestmentFilter) (domain.InvestmentHistoryBounds, error)
	History(ctx context.Context, userID uuid.UUID, query InvestmentHistoryQuery) ([]domain.InvestmentHistoryRow, error)
}
