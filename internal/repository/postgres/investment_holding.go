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

const deleteInvestmentHoldings = `
DELETE FROM investment_holdings
WHERE user_id = $1
  AND account_id = ANY($2::uuid[]);
`

const insertInvestmentHoldings = `
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
    a.user_id,
    a.id,
    s.id,
    max(input.institution_price::numeric),
    max(input.institution_price_as_of),
    sum(input.institution_value::numeric),
    sum(input.cost_basis::numeric),
    sum(input.quantity::numeric),
    max(input.currency),
    sum(input.vested_quantity::numeric),
    sum(input.vested_value::numeric)
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
JOIN accounts AS a ON a.id = input.account_id AND a.user_id = $1
JOIN investment_securities AS s ON s.user_id = $1 AND s.external_id = input.security_external_id
GROUP BY a.user_id, a.id, s.id;
`

func (r *InvestmentRepository) ReplaceHoldings(
	ctx context.Context,
	userID uuid.UUID,
	accountIDs []uuid.UUID,
	holdings []domain.InvestmentHolding,
) error {
	if _, err := r.pool.Exec(ctx, deleteInvestmentHoldings, userID, accountIDs); err != nil {
		return fmt.Errorf("delete investment holdings: %w", err)
	}

	if len(holdings) == 0 {
		return nil
	}

	holdingAccountIDs := make([]uuid.UUID, len(holdings))
	securityExternalIDs := make([]string, len(holdings))
	institutionPrices := make([]string, len(holdings))
	institutionPriceAsOf := make([]*time.Time, len(holdings))
	institutionValues := make([]string, len(holdings))
	costBases := make([]*string, len(holdings))
	quantities := make([]string, len(holdings))
	currencies := make([]*string, len(holdings))
	vestedQuantities := make([]*string, len(holdings))
	vestedValues := make([]*string, len(holdings))

	for i, holding := range holdings {
		holdingAccountIDs[i] = holding.AccountID
		securityExternalIDs[i] = holding.SecurityExternalID
		institutionPrices[i] = holding.InstitutionPrice
		institutionPriceAsOf[i] = holding.InstitutionPriceAsOf
		institutionValues[i] = holding.InstitutionValue
		costBases[i] = holding.CostBasis
		quantities[i] = holding.Quantity
		currencies[i] = holding.Currency
		vestedQuantities[i] = holding.VestedQuantity
		vestedValues[i] = holding.VestedValue
	}

	_, err := r.pool.Exec(
		ctx,
		insertInvestmentHoldings,
		userID,
		holdingAccountIDs,
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
		return fmt.Errorf("insert investment holdings: %w", err)
	}

	return nil
}

const listInvestmentHoldings = `
SELECT
    h.id,
    h.account_id,
    a.name                         AS account_name,
    h.security_id,
    s.external_id                  AS security_external_id,
    s.name                         AS security_name,
    s.ticker_symbol,
    s.type                         AS security_type,
    h.institution_price::text      AS institution_price,
    h.institution_price_as_of,
    h.institution_value::text      AS institution_value,
    h.cost_basis::text             AS cost_basis,
    h.quantity::text               AS quantity,
    h.currency,
    h.vested_quantity::text        AS vested_quantity,
    h.vested_value::text           AS vested_value
FROM investment_holdings AS h
JOIN accounts AS a ON a.id = h.account_id
JOIN investment_securities AS s ON s.id = h.security_id
WHERE h.user_id = $1
  AND ($2::uuid IS NULL OR h.account_id = $2)
  AND ($3::uuid[] IS NULL OR h.account_id <> ALL($3::uuid[]))
ORDER BY h.institution_value DESC, s.name, h.id;
`

func (r *InvestmentRepository) ListHoldings(
	ctx context.Context,
	userID uuid.UUID,
	filter repository.InvestmentFilter,
) ([]domain.InvestmentHolding, error) {
	rows, err := r.pool.Query(ctx, listInvestmentHoldings, userID, filter.AccountID, filter.ExcludeAccountID)
	if err != nil {
		return nil, fmt.Errorf("query investment holdings: %w", err)
	}

	holdings, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.InvestmentHolding])
	if err != nil {
		return nil, fmt.Errorf("collect investment holdings: %w", err)
	}

	return holdings, nil
}
