package postgres

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type InvestmentRepository struct {
	pool *pgxpool.Pool
}

func NewInvestmentRepository(pool *pgxpool.Pool) *InvestmentRepository {
	return &InvestmentRepository{pool: pool}
}

const upsertInvestmentSecurities = `
INSERT INTO investment_securities (
    external_id,
    name,
    ticker_symbol,
    type,
    subtype,
    currency
)
SELECT
    input.external_id,
    input.name,
    input.ticker,
    input.type,
    input.subtype,
    input.currency
FROM unnest(
    $1::text[],
    $2::text[],
    $3::text[],
    $4::text[],
    $5::text[],
    $6::text[]
) AS input (
    external_id,
    name,
    ticker,
    type,
    subtype,
    currency
)
ON CONFLICT (external_id) DO UPDATE
SET name          = COALESCE(EXCLUDED.name, investment_securities.name),
    ticker_symbol = COALESCE(EXCLUDED.ticker_symbol, investment_securities.ticker_symbol),
    type          = COALESCE(EXCLUDED.type, investment_securities.type),
    subtype       = COALESCE(EXCLUDED.subtype, investment_securities.subtype),
    currency      = COALESCE(EXCLUDED.currency, investment_securities.currency),
    updated_at    = now();
`

func upsertSecurities(ctx context.Context, db execer, securities []domain.InvestmentSecurity) error {
	if len(securities) == 0 {
		return nil
	}

	externalIDs := make([]string, len(securities))
	names := make([]*string, len(securities))
	tickers := make([]*string, len(securities))
	types := make([]*string, len(securities))
	subtypes := make([]*string, len(securities))
	currencies := make([]*string, len(securities))

	for i, security := range securities {
		externalIDs[i] = security.ExternalID
		names[i] = security.Name
		tickers[i] = security.Ticker
		types[i] = security.Type
		subtypes[i] = security.Subtype
		currencies[i] = security.Currency
	}

	_, err := db.Exec(ctx, upsertInvestmentSecurities, externalIDs, names, tickers, types, subtypes, currencies)
	if err != nil {
		return fmt.Errorf("upsert investment securities: %w", err)
	}

	return nil
}

const replaceInvestmentHoldings = `
WITH incoming AS (
    SELECT
        $1::uuid AS user_id,
        input.account_id,
        s.id AS security_id,
        input.institution_price::numeric AS institution_price,
        input.institution_price_as_of,
        input.institution_value::numeric AS institution_value,
        input.cost_basis::numeric AS cost_basis,
        input.quantity::numeric AS quantity,
        input.currency,
        input.vested_quantity::numeric AS vested_quantity,
        input.vested_value::numeric AS vested_value
    FROM unnest(
        $2::uuid[],
        $3::text[],
        $4::text[],
        $5::date[],
        $6::text[],
        $7::text[],
        $8::text[],
        $9::text[],
        $10::text[],
        $11::text[]
    ) AS input (
        account_id,
        security_external_id,
        institution_price,
        institution_price_as_of,
        institution_value,
        cost_basis,
        quantity,
        currency,
        vested_quantity,
        vested_value
    )
    JOIN investment_securities AS s ON s.external_id = input.security_external_id
)
INSERT INTO investment_holdings (
    user_id,
    account_id,
    security_id,
    institution_price,
    institution_price_as_of,
    institution_value,
    cost_basis,
    quantity,
    currency,
    vested_quantity,
    vested_value
)
SELECT
    incoming.user_id,
    incoming.account_id,
    incoming.security_id,
    incoming.institution_price,
    incoming.institution_price_as_of,
    incoming.institution_value,
    incoming.cost_basis,
    incoming.quantity,
    incoming.currency,
    incoming.vested_quantity,
    incoming.vested_value
FROM incoming
JOIN accounts AS a ON a.id = incoming.account_id
WHERE a.user_id = $1
ON CONFLICT (account_id, security_id) DO UPDATE
SET institution_price       = EXCLUDED.institution_price,
    institution_price_as_of = EXCLUDED.institution_price_as_of,
    institution_value       = EXCLUDED.institution_value,
    cost_basis              = EXCLUDED.cost_basis,
    quantity                = EXCLUDED.quantity,
    currency                = EXCLUDED.currency,
    vested_quantity         = EXCLUDED.vested_quantity,
    vested_value            = EXCLUDED.vested_value,
    updated_at              = now();
`

const deleteInvestmentHoldings = `
DELETE FROM investment_holdings
WHERE user_id = $1
  AND account_id = ANY($2::uuid[]);
`

func (r *InvestmentRepository) ReplaceHoldings(
	ctx context.Context,
	userID uuid.UUID,
	accountIDs map[string]uuid.UUID,
	securities []domain.InvestmentSecurity,
	holdings []domain.InvestmentHolding,
) error {
	allAccountIDs := slices.Collect(maps.Values(accountIDs))

	resolvedAccountIDs := make([]uuid.UUID, 0, len(holdings))
	securityExternalIDs := make([]string, 0, len(holdings))
	institutionPrices := make([]string, 0, len(holdings))
	institutionPriceAsOf := make([]*time.Time, 0, len(holdings))
	institutionValues := make([]string, 0, len(holdings))
	costBases := make([]*string, 0, len(holdings))
	quantities := make([]string, 0, len(holdings))
	currencies := make([]*string, 0, len(holdings))
	vestedQuantities := make([]*string, 0, len(holdings))
	vestedValues := make([]*string, 0, len(holdings))

	for _, holding := range holdings {
		accountID, ok := accountIDs[holding.AccountExternalID]
		if !ok {
			continue
		}

		resolvedAccountIDs = append(resolvedAccountIDs, accountID)
		securityExternalIDs = append(securityExternalIDs, holding.SecurityExternalID)
		institutionPrices = append(institutionPrices, holding.InstitutionPrice)
		institutionPriceAsOf = append(institutionPriceAsOf, holding.InstitutionPriceAsOf)
		institutionValues = append(institutionValues, holding.InstitutionValue)
		costBases = append(costBases, holding.CostBasis)
		quantities = append(quantities, holding.Quantity)
		currencies = append(currencies, holding.Currency)
		vestedQuantities = append(vestedQuantities, holding.VestedQuantity)
		vestedValues = append(vestedValues, holding.VestedValue)
	}

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := upsertSecurities(ctx, tx, securities); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, deleteInvestmentHoldings, userID, allAccountIDs); err != nil {
			return fmt.Errorf("delete investment holdings: %w", err)
		}

		if len(resolvedAccountIDs) == 0 {
			return nil
		}

		_, err := tx.Exec(
			ctx,
			replaceInvestmentHoldings,
			userID,
			resolvedAccountIDs,
			securityExternalIDs,
			institutionPrices,
			institutionPriceAsOf,
			institutionValues,
			costBases,
			quantities,
			currencies,
			vestedQuantities,
			vestedValues,
		)
		if err != nil {
			return fmt.Errorf("replace investment holdings: %w", err)
		}

		return nil
	})
}

const upsertInvestmentTransactions = `
WITH incoming AS (
    SELECT
        input.account_id,
        s.id AS security_id,
        input.external_id,
        input.cancel_external_id,
        input.posted_on,
        input.transaction_datetime,
        input.name,
        input.quantity::numeric AS quantity,
        input.amount::numeric AS amount,
        input.price::numeric AS price,
        input.fees::numeric AS fees,
        input.type,
        input.subtype,
        input.currency
    FROM unnest(
        $2::uuid[],
        $3::text[],
        $4::text[],
        $5::text[],
        $6::date[],
        $7::timestamptz[],
        $8::text[],
        $9::text[],
        $10::text[],
        $11::text[],
        $12::text[],
        $13::text[],
        $14::text[],
        $15::text[]
    ) AS input (
        account_id,
        security_external_id,
        external_id,
        cancel_external_id,
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
    LEFT JOIN investment_securities AS s ON s.external_id = input.security_external_id
)
INSERT INTO investment_transactions (
    user_id,
    account_id,
    security_id,
    external_id,
    cancel_external_id,
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
    $1,
    incoming.account_id,
    incoming.security_id,
    incoming.external_id,
    incoming.cancel_external_id,
    incoming.posted_on,
    incoming.transaction_datetime,
    incoming.name,
    incoming.quantity,
    incoming.amount,
    incoming.price,
    incoming.fees,
    incoming.type,
    incoming.subtype,
    incoming.currency
FROM incoming
JOIN accounts AS a ON a.id = incoming.account_id
WHERE a.user_id = $1
ON CONFLICT (account_id, external_id) DO UPDATE
SET security_id          = EXCLUDED.security_id,
    cancel_external_id   = EXCLUDED.cancel_external_id,
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

func (r *InvestmentRepository) UpsertTransactions(
	ctx context.Context,
	userID uuid.UUID,
	accountIDs map[string]uuid.UUID,
	securities []domain.InvestmentSecurity,
	transactions []domain.InvestmentTransaction,
) error {
	resolvedAccountIDs := make([]uuid.UUID, 0, len(transactions))
	securityExternalIDs := make([]*string, 0, len(transactions))
	externalIDs := make([]string, 0, len(transactions))
	cancelExternalIDs := make([]*string, 0, len(transactions))
	postedOn := make([]time.Time, 0, len(transactions))
	transactionDateTimes := make([]*time.Time, 0, len(transactions))
	names := make([]string, 0, len(transactions))
	quantities := make([]*string, 0, len(transactions))
	amounts := make([]string, 0, len(transactions))
	prices := make([]*string, 0, len(transactions))
	fees := make([]*string, 0, len(transactions))
	types := make([]string, 0, len(transactions))
	subtypes := make([]string, 0, len(transactions))
	currencies := make([]*string, 0, len(transactions))

	for _, transaction := range transactions {
		accountID, ok := accountIDs[transaction.AccountExternalID]
		if !ok {
			continue
		}

		resolvedAccountIDs = append(resolvedAccountIDs, accountID)
		securityExternalIDs = append(securityExternalIDs, transaction.SecurityExternalID)
		externalIDs = append(externalIDs, transaction.ExternalID)
		cancelExternalIDs = append(cancelExternalIDs, transaction.CancelExternalID)
		postedOn = append(postedOn, transaction.PostedOn)
		transactionDateTimes = append(transactionDateTimes, transaction.TransactionDatetime)
		names = append(names, transaction.Name)
		quantities = append(quantities, transaction.Quantity)
		amounts = append(amounts, transaction.Amount)
		prices = append(prices, transaction.Price)
		fees = append(fees, transaction.Fees)
		types = append(types, transaction.Type)
		subtypes = append(subtypes, transaction.Subtype)
		currencies = append(currencies, transaction.Currency)
	}

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := upsertSecurities(ctx, tx, securities); err != nil {
			return err
		}

		if len(resolvedAccountIDs) == 0 {
			return nil
		}

		_, err := tx.Exec(
			ctx,
			upsertInvestmentTransactions,
			userID,
			resolvedAccountIDs,
			securityExternalIDs,
			externalIDs,
			cancelExternalIDs,
			postedOn,
			transactionDateTimes,
			names,
			quantities,
			amounts,
			prices,
			fees,
			types,
			subtypes,
			currencies,
		)
		if err != nil {
			return fmt.Errorf("upsert investment transactions: %w", err)
		}

		return nil
	})
}

const listInvestmentHoldingsByUser = `
SELECT
    h.id,
    h.account_id,
    a.name AS account_name,
    h.security_id,
    s.name AS security_name,
    s.ticker_symbol,
    h.institution_price::text AS institution_price,
    h.institution_price_as_of,
    h.institution_value::text AS institution_value,
    h.cost_basis::text AS cost_basis,
    h.quantity::text AS quantity,
    h.currency,
    h.vested_quantity::text AS vested_quantity,
    h.vested_value::text AS vested_value
FROM investment_holdings AS h
JOIN accounts AS a ON a.id = h.account_id
JOIN investment_securities AS s ON s.id = h.security_id
WHERE h.user_id = $1
  AND ($2::uuid IS NULL OR h.account_id = $2)
ORDER BY h.institution_value DESC, s.name, h.id;
`

func (r *InvestmentRepository) ListHoldingsByUser(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
) ([]domain.InvestmentHolding, error) {
	rows, err := r.pool.Query(ctx, listInvestmentHoldingsByUser, userID, filter.AccountID)
	if err != nil {
		return nil, fmt.Errorf("query investment holdings: %w", err)
	}

	holdings, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentHolding])
	if err != nil {
		return nil, fmt.Errorf("collect investment holdings: %w", err)
	}

	return holdings, nil
}

const listInvestmentTransactionsByUser = `
SELECT
    t.id,
    t.account_id,
    a.name AS account_name,
    t.security_id,
    s.name AS security_name,
    s.ticker_symbol,
    t.external_id,
    t.cancel_external_id,
    t.posted_on,
    t.transaction_datetime,
    t.name,
    t.quantity::text AS quantity,
    t.amount::text AS amount,
    t.price::text AS price,
    t.fees::text AS fees,
    t.type,
    t.subtype,
    t.currency
FROM investment_transactions AS t
JOIN accounts AS a ON a.id = t.account_id
LEFT JOIN investment_securities AS s ON s.id = t.security_id
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
ORDER BY t.posted_on DESC, t.id DESC
LIMIT $3
OFFSET $4;
`

const countInvestmentTransactionsByUser = `
SELECT COUNT(*) AS count
FROM investment_transactions AS t
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2);
`

func (r *InvestmentRepository) ListTransactionsByUser(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
	limit, offset int,
) ([]domain.InvestmentTransaction, int, error) {
	rows, err := r.pool.Query(ctx, listInvestmentTransactionsByUser, userID, filter.AccountID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query investment transactions: %w", err)
	}

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentTransaction])
	if err != nil {
		return nil, 0, fmt.Errorf("collect investment transactions: %w", err)
	}

	total, err := count(ctx, r.pool, countInvestmentTransactionsByUser, userID, filter.AccountID)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
