package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

const upsertAccounts = `
INSERT INTO accounts AS a (
    user_id,
    connection_id,
    external_id,
    name,
    official_name,
    mask,
    type,
    subtype,
    current_balance,
    available_balance,
    currency
)
SELECT
    c.user_id,
    c.id,
    input.external_id,
    input.name,
    input.official_name,
    input.mask,
    input.type,
    input.subtype,
    input.current_balance::numeric,
    input.available_balance::numeric,
    input.currency
FROM connections AS c
CROSS JOIN unnest(
    $3::text[],
    $4::text[],
    $5::text[],
    $6::text[],
    $7::text[],
    $8::text[],
    $9::text[],
    $10::text[],
    $11::text[]
) AS input (
    external_id,
    name,
    official_name,
    mask,
    type,
    subtype,
    current_balance,
    available_balance,
    currency
)
WHERE c.user_id = $1
  AND c.id = $2
ON CONFLICT (connection_id, external_id) DO UPDATE
SET name              = EXCLUDED.name,
    official_name     = EXCLUDED.official_name,
    mask              = EXCLUDED.mask,
    type              = EXCLUDED.type,
    subtype           = EXCLUDED.subtype,
    current_balance   = EXCLUDED.current_balance,
    available_balance = EXCLUDED.available_balance,
    currency          = EXCLUDED.currency,
    updated_at        = now()
RETURNING
    a.external_id,
    a.id;
`

type upsertedAccount struct {
	ExternalID string    `db:"external_id"`
	ID         uuid.UUID `db:"id"`
}

func (r *AccountRepository) Upsert(ctx context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error) {
	columns := newAccountColumns(accounts)

	rows, err := r.pool.Query(
		ctx,
		upsertAccounts,
		userID,
		connectionID,
		columns.externalIDs,
		columns.names,
		columns.officialNames,
		columns.masks,
		columns.types,
		columns.subtypes,
		columns.currentBalances,
		columns.availableBalances,
		columns.currencies,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert accounts: %w", err)
	}

	upserted, err := pgx.CollectRows(rows, pgx.RowToStructByName[upsertedAccount])
	if err != nil {
		return nil, fmt.Errorf("upsert accounts: %w", err)
	}

	ids := make(map[string]uuid.UUID, len(upserted))

	for _, account := range upserted {
		ids[account.ExternalID] = account.ID
	}

	return ids, nil
}

const deleteAccountsExcept = `
DELETE FROM accounts
WHERE user_id = $1
  AND connection_id = $2
  AND id <> ALL(COALESCE($3::uuid[], '{}'));
`

func (r *AccountRepository) DeleteExcept(ctx context.Context, userID, connectionID uuid.UUID, keepIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx, deleteAccountsExcept, userID, connectionID, keepIDs)
	if err != nil {
		return fmt.Errorf("delete accounts: %w", err)
	}

	return nil
}

const listAccountsByUser = `
SELECT
    a.id,
    a.user_id,
    a.connection_id,
    c.institution_name,
    c.institution_url,
    a.external_id,
    a.name,
    a.official_name,
    a.mask,
    a.type,
    a.subtype,
    a.current_balance::text   AS current_balance,
    a.available_balance::text AS available_balance,
    a.currency
FROM accounts AS a
JOIN connections AS c ON c.id = a.connection_id
WHERE a.user_id = $1
ORDER BY c.institution_name, a.name, a.id
LIMIT $2
OFFSET $3;
`

const countAccountsByUser = `
SELECT COUNT(*) AS count
FROM accounts AS a
WHERE a.user_id = $1;
`

func (r *AccountRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Account, int, error) {
	rows, err := r.pool.Query(ctx, listAccountsByUser, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query accounts: %w", err)
	}

	accounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Account])
	if err != nil {
		return nil, 0, fmt.Errorf("collect accounts: %w", err)
	}

	total, err := count(ctx, r.pool, countAccountsByUser, userID)
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}
