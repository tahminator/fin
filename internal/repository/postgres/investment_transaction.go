package postgres

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const upsertInvestmentTransactions = `
INSERT INTO investment_transactions (
    user_id,
    account_id,
    security_id,
    external_id,
    posted_on,
    transaction_datetime,
    name,
    quantity,
    amount,
    price,
    fees,
    type,
    subtype,
    currency
)
SELECT
    a.user_id,
    a.id,
    s.id,
    input.external_id,
    input.posted_on,
    input.transaction_datetime,
    input.name,
    input.quantity::numeric,
    input.amount::numeric,
    input.price::numeric,
    input.fees::numeric,
    input.type,
    input.subtype,
    input.currency
FROM unnest(
    $2::uuid[],
    $3::text[],
    $4::text[],
    $5::date[],
    $6::timestamptz[],
    $7::text[],
    $8::text[],
    $9::text[],
    $10::text[],
    $11::text[],
    $12::text[],
    $13::text[],
    $14::text[]
) AS input (
    account_id,
    security_external_id,
    external_id,
    posted_on,
    transaction_datetime,
    name,
    quantity,
    amount,
    price,
    fees,
    type,
    subtype,
    currency
)
JOIN accounts AS a ON a.id = input.account_id AND a.user_id = $1
LEFT JOIN investment_securities AS s ON s.user_id = $1 AND s.external_id = input.security_external_id
ON CONFLICT (account_id, external_id) DO UPDATE
SET security_id          = EXCLUDED.security_id,
    posted_on            = EXCLUDED.posted_on,
    transaction_datetime = EXCLUDED.transaction_datetime,
    name                 = EXCLUDED.name,
    quantity             = EXCLUDED.quantity,
    amount               = EXCLUDED.amount,
    price                = EXCLUDED.price,
    fees                 = EXCLUDED.fees,
    type                 = EXCLUDED.type,
    subtype              = EXCLUDED.subtype,
    currency             = EXCLUDED.currency,
    updated_at           = now();
`

const deleteStaleInvestmentTransactions = `
DELETE FROM investment_transactions AS t
WHERE t.user_id = $1
  AND t.account_id = ANY($2::uuid[])
  AND (
      t.posted_on < $3::date
      OR NOT EXISTS (
          SELECT 1
          FROM unnest($4::uuid[], $5::text[]) AS kept (account_id, external_id)
          WHERE kept.account_id = t.account_id
            AND kept.external_id = t.external_id
      )
  );
`

func (r *InvestmentRepository) ReplaceTransactions(
	ctx context.Context,
	userID uuid.UUID,
	accountIDs []uuid.UUID,
	since time.Time,
	transactions []domain.InvestmentTransaction,
) error {
	columns := newInvestmentTransactionColumns(transactions)

	if len(transactions) > 0 {
		_, err := r.pool.Exec(
			ctx,
			upsertInvestmentTransactions,
			userID,
			columns.accountIDs,
			columns.securityExternalIDs,
			columns.externalIDs,
			columns.postedOn,
			columns.transactionDatetimes,
			columns.names,
			columns.quantities,
			columns.amounts,
			columns.prices,
			columns.fees,
			columns.types,
			columns.subtypes,
			columns.currencies,
		)
		if err != nil {
			return fmt.Errorf("upsert investment transactions: %w", err)
		}
	}

	_, err := r.pool.Exec(ctx, deleteStaleInvestmentTransactions, userID, accountIDs, since, columns.accountIDs, columns.externalIDs)
	if err != nil {
		return fmt.Errorf("delete stale investment transactions: %w", err)
	}

	return nil
}

type investmentTransactionColumns struct {
	accountIDs           []uuid.UUID
	securityExternalIDs  []*string
	externalIDs          []string
	postedOn             []time.Time
	transactionDatetimes []*time.Time
	names                []string
	quantities           []string
	amounts              []string
	prices               []string
	fees                 []*string
	types                []string
	subtypes             []string
	currencies           []*string
}

func newInvestmentTransactionColumns(transactions []domain.InvestmentTransaction) investmentTransactionColumns {
	count := len(transactions)
	columns := investmentTransactionColumns{
		accountIDs:           make([]uuid.UUID, count),
		securityExternalIDs:  make([]*string, count),
		externalIDs:          make([]string, count),
		postedOn:             make([]time.Time, count),
		transactionDatetimes: make([]*time.Time, count),
		names:                make([]string, count),
		quantities:           make([]string, count),
		amounts:              make([]string, count),
		prices:               make([]string, count),
		fees:                 make([]*string, count),
		types:                make([]string, count),
		subtypes:             make([]string, count),
		currencies:           make([]*string, count),
	}

	for i, transaction := range transactions {
		columns.accountIDs[i] = transaction.AccountID
		columns.securityExternalIDs[i] = transaction.SecurityExternalID
		columns.externalIDs[i] = transaction.ExternalID
		columns.postedOn[i] = transaction.PostedOn
		columns.transactionDatetimes[i] = transaction.TransactionDatetime
		columns.names[i] = transaction.Name
		columns.quantities[i] = transaction.Quantity
		columns.amounts[i] = transaction.Amount
		columns.prices[i] = transaction.Price
		columns.fees[i] = transaction.Fees
		columns.types[i] = transaction.Type
		columns.subtypes[i] = transaction.Subtype
		columns.currencies[i] = transaction.Currency
	}

	return columns
}

const listInvestmentTransactions = `
SELECT
    t.id,
    t.account_id,
    a.name                 AS account_name,
    t.security_id,
    s.external_id          AS security_external_id,
    s.name                 AS security_name,
    s.ticker_symbol,
    t.external_id,
    t.posted_on,
    t.transaction_datetime,
    t.name,
    t.quantity::text       AS quantity,
    t.amount::text         AS amount,
    t.price::text          AS price,
    t.fees::text           AS fees,
    t.type,
    t.subtype,
    t.currency
FROM investment_transactions AS t
JOIN accounts AS a ON a.id = t.account_id
LEFT JOIN investment_securities AS s ON s.id = t.security_id
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
  AND ($3::uuid[] IS NULL OR t.account_id <> ALL($3::uuid[]))
ORDER BY t.posted_on DESC, t.id DESC
LIMIT $4
OFFSET $5;
`

const countInvestmentTransactions = `
SELECT COUNT(*) AS count
FROM investment_transactions AS t
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
  AND ($3::uuid[] IS NULL OR t.account_id <> ALL($3::uuid[]));
`

func (r *InvestmentRepository) ListTransactions(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
	limit, offset int,
) ([]domain.InvestmentTransaction, int, error) {
	rows, err := r.pool.Query(ctx, listInvestmentTransactions, userID, filter.AccountID, filter.ExcludeAccountID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query investment transactions: %w", err)
	}

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentTransaction])
	if err != nil {
		return nil, 0, fmt.Errorf("collect investment transactions: %w", err)
	}

	total, err := count(ctx, r.pool, countInvestmentTransactions, userID, filter.AccountID, filter.ExcludeAccountID)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
