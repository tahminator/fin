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
      AND a.id <> ALL($2::uuid[])
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

func (r *SummaryRepository) Balances(ctx context.Context, userID uuid.UUID, excludeAccountIDs []uuid.UUID) ([]domain.CurrencyBalance, error) {
	rows, err := r.pool.Query(ctx, summaryBalances, userID, excludeAccountIDs)
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

const summaryCashFlow = `
WITH flows AS (
    SELECT
        COALESCE(t.currency, 'XXX')                       AS currency,
        t.posted_on                                       AS day,
        COALESCE(SUM(t.amount) FILTER (WHERE t.amount > 0), 0)  AS money_in,
        COALESCE(-SUM(t.amount) FILTER (WHERE t.amount < 0), 0) AS money_out
    FROM transactions AS t
    WHERE t.user_id = $1
      AND t.posted_on > current_date - $2::int
      AND t.posted_on <= current_date
      AND t.account_id <> ALL($3::uuid[])
      AND (t.category IS NULL OR t.category NOT IN ('TRANSFER_IN', 'TRANSFER_OUT', 'LOAN_PAYMENTS'))
    GROUP BY COALESCE(t.currency, 'XXX'), t.posted_on
),
days AS (
    SELECT generate_series(current_date - ($2::int - 1), current_date, interval '1 day')::date AS day
)
SELECT
    c.currency,
    d.day,
    COALESCE(f.money_in, 0)::text  AS money_in,
    COALESCE(f.money_out, 0)::text AS money_out
FROM (SELECT DISTINCT currency FROM flows) AS c
CROSS JOIN days AS d
LEFT JOIN flows AS f ON f.currency = c.currency AND f.day = d.day
ORDER BY c.currency, d.day;
`

func (r *SummaryRepository) CashFlow(ctx context.Context, userID uuid.UUID, days int, excludeAccountIDs []uuid.UUID) ([]domain.DailyCashFlow, error) {
	rows, err := r.pool.Query(ctx, summaryCashFlow, userID, days, excludeAccountIDs)
	if err != nil {
		return nil, fmt.Errorf("query cash flow: %w", err)
	}

	flows, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.DailyCashFlow])
	if err != nil {
		return nil, fmt.Errorf("collect cash flow: %w", err)
	}

	return flows, nil
}
