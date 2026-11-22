package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const investmentHistoryBounds = `
SELECT
    min(s.captured_at) AS earliest,
    max(s.captured_at) AS latest
FROM investment_account_snapshots AS s
WHERE s.user_id = $1
  AND ($2::uuid IS NULL OR s.account_id = $2)
  AND ($3::uuid[] IS NULL OR s.account_id <> ALL($3::uuid[]));
`

func (r *InvestmentRepository) HistoryBounds(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
) (domain.InvestmentHistoryBounds, error) {
	rows, err := r.pool.Query(ctx, investmentHistoryBounds, userID, filter.AccountID, filter.ExcludeAccountID)
	if err != nil {
		return domain.InvestmentHistoryBounds{}, fmt.Errorf("query investment history bounds: %w", err)
	}

	bounds, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.InvestmentHistoryBounds])
	if err != nil {
		return domain.InvestmentHistoryBounds{}, fmt.Errorf("collect investment history bounds: %w", err)
	}

	return bounds, nil
}

const investmentHistory = `
WITH scoped AS (
    SELECT
        s.id,
        s.account_id,
        s.captured_at,
        s.value,
        s.currency
    FROM investment_account_snapshots AS s
    WHERE s.user_id = $1
      AND ($2::uuid IS NULL OR s.account_id = $2)
      AND ($3::uuid[] IS NULL OR s.account_id <> ALL($3::uuid[]))
),
points AS (
    SELECT max(scoped.captured_at) AS at
    FROM scoped
    WHERE $4::timestamptz IS NULL
       OR scoped.captured_at >= $4::timestamptz
    GROUP BY date_bin(make_interval(secs => $5), scoped.captured_at, TIMESTAMPTZ '2000-01-01 00:00:00+00')
),
latest AS (
    SELECT
        points.at,
        snapshot.id,
        snapshot.account_id,
        snapshot.value,
        snapshot.currency
    FROM points
    CROSS JOIN LATERAL (
        SELECT DISTINCT ON (scoped.account_id)
            scoped.id,
            scoped.account_id,
            scoped.value,
            scoped.currency
        FROM scoped
        WHERE scoped.captured_at <= points.at
        ORDER BY scoped.account_id, scoped.captured_at DESC
    ) AS snapshot
),
valued AS (
    SELECT
        latest.at,
        latest.account_id,
        coalesce(latest.currency, '') AS currency,
        latest.value
    FROM latest
    WHERE $6::uuid[] IS NULL
    UNION ALL
    SELECT
        latest.at,
        latest.account_id,
        coalesce(h.currency, latest.currency, '') AS currency,
        h.institution_value AS value
    FROM latest
    JOIN investment_holding_snapshots AS h ON h.snapshot_id = latest.id
    WHERE $6::uuid[] IS NOT NULL
      AND h.user_id = $1
      AND h.security_id = ANY($6::uuid[])
)
SELECT
    valued.at,
    valued.account_id,
    valued.currency,
    sum(valued.value)::text AS value
FROM valued
GROUP BY GROUPING SETS ((valued.at, valued.currency, valued.account_id), (valued.at, valued.currency))
ORDER BY valued.at, valued.currency, valued.account_id NULLS FIRST;
`

func (r *InvestmentRepository) History(
	ctx context.Context,
	userID uuid.UUID,
	query repository.InvestmentHistoryQuery,
) ([]domain.InvestmentHistoryRow, error) {
	rows, err := r.pool.Query(
		ctx,
		investmentHistory,
		userID,
		query.Filter.AccountID,
		query.Filter.ExcludeAccountID,
		query.From,
		query.Bucket.Seconds(),
		query.SecurityIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query investment history: %w", err)
	}

	history, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentHistoryRow])
	if err != nil {
		return nil, fmt.Errorf("collect investment history: %w", err)
	}

	return history, nil
}

const listInvestmentSecurities = `
SELECT
    s.id,
    s.name,
    s.ticker_symbol,
    s.type,
    s.subtype,
    s.currency,
    EXISTS (
        SELECT 1
        FROM investment_holdings AS h
        WHERE h.user_id = $1
          AND h.security_id = s.id
          AND ($2::uuid IS NULL OR h.account_id = $2)
          AND ($3::uuid[] IS NULL OR h.account_id <> ALL($3::uuid[]))
    ) AS held
FROM investment_securities AS s
WHERE s.user_id = $1
  AND EXISTS (
      SELECT 1
      FROM investment_holding_snapshots AS hs
      WHERE hs.user_id = $1
        AND hs.security_id = s.id
        AND ($2::uuid IS NULL OR hs.account_id = $2)
        AND ($3::uuid[] IS NULL OR hs.account_id <> ALL($3::uuid[]))
  )
ORDER BY held DESC, coalesce(s.ticker_symbol, s.name, s.external_id), s.id;
`

func (r *InvestmentRepository) ListSecurities(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
) ([]domain.InvestmentSecuritySummary, error) {
	rows, err := r.pool.Query(ctx, listInvestmentSecurities, userID, filter.AccountID, filter.ExcludeAccountID)
	if err != nil {
		return nil, fmt.Errorf("query investment securities: %w", err)
	}

	securities, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentSecuritySummary])
	if err != nil {
		return nil, fmt.Errorf("collect investment securities: %w", err)
	}

	return securities, nil
}
