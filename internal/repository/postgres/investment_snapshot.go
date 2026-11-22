package postgres

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

const (
	snapshotMinSpacing     = 15 * time.Minute
	snapshotFullResolution = 90 * 24 * time.Hour
)

const deleteRecentInvestmentSnapshots = `
DELETE FROM investment_account_snapshots
WHERE user_id = $1
  AND account_id = ANY($2::uuid[])
  AND captured_at > $3::timestamptz - make_interval(secs => $4)
  AND captured_at < $3::timestamptz;
`

const upsertInvestmentAccountSnapshots = `
INSERT INTO investment_account_snapshots (
    user_id,
    account_id,
    captured_at,
    value,
    currency
)
SELECT
    a.user_id,
    a.id,
    $3,
    coalesce(a.current_balance, held.value),
    a.currency
FROM accounts AS a
LEFT JOIN LATERAL (
    SELECT sum(h.institution_value) AS value
    FROM investment_holdings AS h
    WHERE h.user_id = a.user_id
      AND h.account_id = a.id
) AS held ON true
WHERE a.user_id = $1
  AND a.id = ANY($2::uuid[])
  AND coalesce(a.current_balance, held.value) IS NOT NULL
ON CONFLICT (account_id, captured_at) DO UPDATE
SET value    = EXCLUDED.value,
    currency = EXCLUDED.currency;
`

const deleteInvestmentHoldingSnapshotsAt = `
DELETE FROM investment_holding_snapshots
WHERE user_id = $1
  AND account_id = ANY($2::uuid[])
  AND captured_at = $3;
`

const insertInvestmentHoldingSnapshots = `
INSERT INTO investment_holding_snapshots (
    snapshot_id,
    user_id,
    account_id,
    security_id,
    captured_at,
    quantity,
    institution_price,
    institution_value,
    currency
)
SELECT
    s.id,
    h.user_id,
    h.account_id,
    h.security_id,
    s.captured_at,
    h.quantity,
    h.institution_price,
    h.institution_value,
    h.currency
FROM investment_account_snapshots AS s
JOIN investment_holdings AS h ON h.user_id = s.user_id AND h.account_id = s.account_id
WHERE s.user_id = $1
  AND s.account_id = ANY($2::uuid[])
  AND s.captured_at = $3;
`

const thinInvestmentSnapshots = `
DELETE FROM investment_account_snapshots AS s
WHERE s.user_id = $1
  AND s.account_id = ANY($2::uuid[])
  AND s.captured_at < $3::timestamptz - make_interval(secs => $4)
  AND EXISTS (
      SELECT 1
      FROM investment_account_snapshots AS later
      WHERE later.account_id = s.account_id
        AND later.captured_at > s.captured_at
        AND date_trunc('day', later.captured_at, 'UTC') = date_trunc('day', s.captured_at, 'UTC')
  );
`

func (r *InvestmentRepository) CaptureSnapshot(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID, capturedAt time.Time) error {
	if len(accountIDs) == 0 {
		return nil
	}

	steps := []struct {
		name  string
		query string
		args  []any
	}{
		{"delete superseded snapshots", deleteRecentInvestmentSnapshots, []any{userID, accountIDs, capturedAt, snapshotMinSpacing.Seconds()}},
		{"save account snapshots", upsertInvestmentAccountSnapshots, []any{userID, accountIDs, capturedAt}},
		{"clear holding snapshots", deleteInvestmentHoldingSnapshotsAt, []any{userID, accountIDs, capturedAt}},
		{"save holding snapshots", insertInvestmentHoldingSnapshots, []any{userID, accountIDs, capturedAt}},
		{"thin old snapshots", thinInvestmentSnapshots, []any{userID, accountIDs, capturedAt, snapshotFullResolution.Seconds()}},
	}

	for _, step := range steps {
		if _, err := r.pool.Exec(ctx, step.query, step.args...); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}
