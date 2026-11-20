package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type SummaryRepository struct {
	pool *pgxpool.Pool
}

func NewSummaryRepository(pool *pgxpool.Pool) *SummaryRepository {
	return &SummaryRepository{pool: pool}
}

const summaryBalances = `
WITH totals AS (
    SELECT
        COALESCE(a.currency, 'XXX') AS currency,
        COALESCE(SUM(a.current_balance) FILTER (WHERE a.type NOT IN ('credit', 'loan')), 0) AS assets,
        COALESCE(SUM(a.current_balance) FILTER (WHERE a.type IN ('credit', 'loan')), 0)     AS liabilities
    FROM accounts AS a
    WHERE a.user_id = $1
      AND a.current_balance IS NOT NULL
    GROUP BY COALESCE(a.currency, 'XXX')
)
SELECT
    t.currency,
    t.assets::text                   AS assets,
    t.liabilities::text              AS liabilities,
    (t.assets - t.liabilities)::text AS net
FROM totals AS t
ORDER BY t.currency;
`

func (r *SummaryRepository) Balances(ctx context.Context, userID uuid.UUID) ([]domain.CurrencyBalance, error) {
	rows, err := r.pool.Query(ctx, summaryBalances, userID)
	if err != nil {
		return nil, fmt.Errorf("query balances: %w", err)
	}

	balances, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.CurrencyBalance])
	if err != nil {
		return nil, fmt.Errorf("collect balances: %w", err)
	}

	return balances, nil
}

const summaryCounts = `
SELECT
    (SELECT COUNT(*) FROM connections AS c WHERE c.user_id = $1) AS connections,
    (SELECT COUNT(*) FROM accounts AS a WHERE a.user_id = $1)    AS accounts;
`

func (r *SummaryRepository) Counts(ctx context.Context, userID uuid.UUID) (domain.SummaryCounts, error) {
	rows, err := r.pool.Query(ctx, summaryCounts, userID)
	if err != nil {
		return domain.SummaryCounts{}, fmt.Errorf("query counts: %w", err)
	}

	counts, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.SummaryCounts])
	if err != nil {
		return domain.SummaryCounts{}, fmt.Errorf("collect counts: %w", err)
	}

	return counts, nil
}
