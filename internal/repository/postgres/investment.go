package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type InvestmentRepository struct {
	pool *pgxpool.Pool
}

func NewInvestmentRepository(pool *pgxpool.Pool) *InvestmentRepository {
	return &InvestmentRepository{pool: pool}
}

const upsertInvestmentSecurities = `
INSERT INTO investment_securities (
    user_id,
    external_id,
    name,
    ticker_symbol,
    type,
    subtype,
    currency
)
SELECT DISTINCT ON (input.external_id)
    $1,
    input.external_id,
    input.name,
    input.ticker_symbol,
    input.type,
    input.subtype,
    input.currency
FROM unnest(
    $2::text[],
    $3::text[],
    $4::text[],
    $5::text[],
    $6::text[],
    $7::text[]
) AS input (
    external_id,
    name,
    ticker_symbol,
    type,
    subtype,
    currency
)
ORDER BY input.external_id
ON CONFLICT (user_id, external_id) DO UPDATE
SET name          = COALESCE(EXCLUDED.name, investment_securities.name),
    ticker_symbol = COALESCE(EXCLUDED.ticker_symbol, investment_securities.ticker_symbol),
    type          = COALESCE(EXCLUDED.type, investment_securities.type),
    subtype       = COALESCE(EXCLUDED.subtype, investment_securities.subtype),
    currency      = COALESCE(EXCLUDED.currency, investment_securities.currency),
    updated_at    = now();
`

func (r *InvestmentRepository) UpsertSecurities(ctx context.Context, userID uuid.UUID, securities []domain.InvestmentSecurity) error {
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

	_, err := r.pool.Exec(ctx, upsertInvestmentSecurities, userID, externalIDs, names, tickers, types, subtypes, currencies)
	if err != nil {
		return fmt.Errorf("upsert investment securities: %w", err)
	}

	return nil
}
