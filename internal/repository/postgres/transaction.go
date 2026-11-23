package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

const getTransactionByUser = `
SELECT
    t.id,
    t.account_id,
    a.name             AS account_name,
    t.external_id,
    t.amount::text     AS amount,
    t.currency,
    t.posted_on,
    t.authorized_on,
    t.name,
    t.merchant_name,
    t.category,
    t.pending
FROM transactions AS t
JOIN accounts AS a ON a.id = t.account_id
WHERE t.user_id = $1
  AND t.id = $2;
`

func (r *TransactionRepository) Get(ctx context.Context, userID, id uuid.UUID) (domain.Transaction, error) {
	rows, err := r.pool.Query(ctx, getTransactionByUser, userID, id)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("query transaction: %w", err)
	}

	transaction, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[domain.Transaction],
	)
	if err != nil {
		return domain.Transaction{}, notFound(err)
	}

	return transaction, nil
}

const listTransactionsByUser = `
SELECT
    t.id,
    t.account_id,
    a.name             AS account_name,
    t.external_id,
    t.amount::text     AS amount,
    t.currency,
    t.posted_on,
    t.authorized_on,
    t.name,
    t.merchant_name,
    t.category,
    t.pending
FROM transactions AS t
JOIN accounts AS a ON a.id = t.account_id
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
  AND ($5::uuid[] IS NULL OR t.account_id <> ALL($5::uuid[]))
  AND a.type <> ALL($6::text[])
ORDER BY t.posted_on DESC, t.id DESC
LIMIT $3
OFFSET $4;
`

const countTransactionsByUser = `
SELECT COUNT(*) AS count
FROM transactions AS t
JOIN accounts AS a ON a.id = t.account_id
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
  AND ($3::uuid[] IS NULL OR t.account_id <> ALL($3::uuid[]))
  AND a.type <> ALL($4::text[]);
`

func (r *TransactionRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.TransactionFilter,
	limit, offset int,
) ([]domain.Transaction, int, error) {
	rows, err := r.pool.Query(ctx, listTransactionsByUser, userID, filter.AccountID, limit, offset, filter.ExcludeAccountID, accountTypeStrings(filter.ExcludeAccountTypes))
	if err != nil {
		return nil, 0, fmt.Errorf("query transactions: %w", err)
	}

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Transaction])
	if err != nil {
		return nil, 0, fmt.Errorf("collect transactions: %w", err)
	}

	total, err := count(ctx, r.pool, countTransactionsByUser, userID, filter.AccountID, filter.ExcludeAccountID, accountTypeStrings(filter.ExcludeAccountTypes))
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

const upsertTransactions = `
INSERT INTO transactions (
    user_id,
    account_id,
    external_id,
    amount,
    currency,
    posted_on,
    authorized_on,
    name,
    merchant_name,
    category,
    pending
)
SELECT
    a.user_id,
    a.id,
    input.external_id,
    input.amount::numeric,
    input.currency,
    input.posted_on,
    input.authorized_on,
    input.name,
    input.merchant_name,
    input.category,
    input.pending
FROM unnest(
    $2::uuid[],
    $3::text[],
    $4::text[],
    $5::text[],
    $6::date[],
    $7::date[],
    $8::text[],
    $9::text[],
    $10::text[],
    $11::boolean[]
) AS input (
    account_id,
    external_id,
    amount,
    currency,
    posted_on,
    authorized_on,
    name,
    merchant_name,
    category,
    pending
)
JOIN accounts AS a ON a.id = input.account_id
WHERE a.user_id = $1
ON CONFLICT (account_id, external_id) DO UPDATE
SET amount        = EXCLUDED.amount,
    currency      = EXCLUDED.currency,
    posted_on     = EXCLUDED.posted_on,
    authorized_on = EXCLUDED.authorized_on,
    name          = EXCLUDED.name,
    merchant_name = EXCLUDED.merchant_name,
    category      = EXCLUDED.category,
    pending       = EXCLUDED.pending,
    updated_at    = now();
`

func (r *TransactionRepository) Upsert(ctx context.Context, userID uuid.UUID, transactions []domain.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := newTransactionColumns(transactions)

	_, err := r.pool.Exec(
		ctx,
		upsertTransactions,
		userID,
		columns.accountIDs,
		columns.externalIDs,
		columns.amounts,
		columns.currencies,
		columns.postedOn,
		columns.authorizedOn,
		columns.names,
		columns.merchantNames,
		columns.categories,
		columns.pending,
	)
	if err != nil {
		return fmt.Errorf("upsert transactions: %w", err)
	}

	return nil
}

const deleteTransactionsByExternalIDs = `
DELETE FROM transactions
WHERE user_id = $1
  AND external_id = ANY($2::text[]);
`

func (r *TransactionRepository) DeleteByExternalIDs(ctx context.Context, userID uuid.UUID, externalIDs []string) error {
	if len(externalIDs) == 0 {
		return nil
	}

	_, err := r.pool.Exec(ctx, deleteTransactionsByExternalIDs, userID, externalIDs)
	if err != nil {
		return fmt.Errorf("delete transactions: %w", err)
	}

	return nil
}
