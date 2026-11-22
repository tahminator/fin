package postgres

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/repository"
)

type TransactionQueryRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionQueryRepository(pool *pgxpool.Pool) *TransactionQueryRepository {
	return &TransactionQueryRepository{pool: pool}
}

const transactionDetailColumns = `
SELECT
    t.id,
    t.account_id,
    a.name             AS account_name,
    a.type             AS account_type,
    c.institution_name,
    t.amount::text     AS amount,
    t.currency,
    t.posted_on,
    t.authorized_on,
    t.name,
    t.merchant_name,
    t.category,
    t.pending`

func (r *TransactionQueryRepository) Search(
	ctx context.Context,
	userID uuid.UUID,
	query repository.TransactionQuery,
	sort repository.TransactionSort,
	limit, offset int,
) ([]repository.TransactionDetail, int, error) {
	order, ok := sortExpressions[sort]
	if !ok {
		return nil, 0, fmt.Errorf("unknown sort %q", sort)
	}

	args := newQueryArgs(userID)
	where := buildTransactionWhere(query, args)
	countArgs := append([]any(nil), args.values...)

	statement := transactionDetailColumns + transactionQueryFrom + "\n" + where +
		"\nORDER BY " + order +
		"\nLIMIT " + args.add(limit) +
		"\nOFFSET " + args.add(offset) + ";"

	rows, err := r.pool.Query(ctx, statement, args.values...)
	if err != nil {
		return nil, 0, fmt.Errorf("search transactions: %w", err)
	}

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[repository.TransactionDetail])
	if err != nil {
		return nil, 0, fmt.Errorf("scan transactions: %w", err)
	}

	total, err := count(ctx, r.pool, "SELECT COUNT(*)"+transactionQueryFrom+"\n"+where+";", countArgs...)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (r *TransactionQueryRepository) Get(ctx context.Context, userID, id uuid.UUID) (repository.TransactionDetail, error) {
	statement := transactionDetailColumns + transactionQueryFrom + "\nWHERE t.user_id = $1\n  AND t.id = $2;"

	rows, err := r.pool.Query(ctx, statement, userID, id)
	if err != nil {
		return repository.TransactionDetail{}, fmt.Errorf("get transaction: %w", err)
	}

	transaction, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[repository.TransactionDetail])
	if err != nil {
		return repository.TransactionDetail{}, notFound(err)
	}

	return transaction, nil
}

func (r *TransactionQueryRepository) Aggregate(ctx context.Context, userID uuid.UUID, spec repository.AggregateSpec) ([]repository.AggregateRow, bool, error) {
	statement, args, err := buildAggregateQuery(userID, spec)
	if err != nil {
		return nil, false, err
	}

	rows, err := r.pool.Query(ctx, statement, args...)
	if err != nil {
		return nil, false, fmt.Errorf("aggregate transactions: %w", err)
	}

	defer rows.Close()

	var result []repository.AggregateRow

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, false, fmt.Errorf("scan aggregate: %w", err)
		}

		result = append(result, aggregateRowFrom(spec, values))
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("aggregate transactions: %w", err)
	}

	truncated := len(result) > spec.Limit
	if truncated {
		result = result[:spec.Limit]
	}

	return result, truncated, nil
}

func aggregateRowFrom(spec repository.AggregateSpec, values []any) repository.AggregateRow {
	row := repository.AggregateRow{
		Dimensions: make(map[repository.Dimension]string, len(spec.GroupBy)),
		Metrics:    make(map[repository.Metric]string, len(spec.Metrics)),
	}

	for i, dimension := range spec.GroupBy {
		row.Dimensions[dimension] = textValue(values[i])
	}

	row.Currency = textValue(values[len(spec.GroupBy)])

	for i, metric := range spec.Metrics {
		row.Metrics[metric] = textValue(values[len(spec.GroupBy)+1+i])
	}

	return row
}

func textValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func (r *TransactionQueryRepository) Facets(
	ctx context.Context,
	userID uuid.UUID,
	query repository.TransactionQuery,
	facet repository.Facet,
	limit int,
) ([]repository.FacetRow, error) {
	expression, ok := facetExpressions[facet]
	if !ok {
		return nil, fmt.Errorf("unknown facet %q", facet)
	}

	args := newQueryArgs(userID)
	where := buildTransactionWhere(query, args)

	statement := "SELECT\n    " + expression + " AS value,\n    coalesce(t.currency, '') AS currency,\n    count(*)::int AS count,\n    sum(t.amount)::text AS sum" +
		transactionQueryFrom + "\n" + where +
		"\nGROUP BY 1, 2\nORDER BY 3 DESC, 1 ASC\nLIMIT " + args.add(limit) + ";"

	rows, err := r.pool.Query(ctx, statement, args.values...)
	if err != nil {
		return nil, fmt.Errorf("transaction facets: %w", err)
	}

	facets, err := pgx.CollectRows(rows, pgx.RowToStructByName[repository.FacetRow])
	if err != nil {
		return nil, fmt.Errorf("scan facets: %w", err)
	}

	return facets, nil
}

const listRecurringCandidates = `
SELECT
    coalesce(t.merchant_name, t.name) AS merchant,
    t.amount::text                    AS amount,
    coalesce(t.currency, '')          AS currency,
    t.posted_on
FROM transactions AS t
WHERE t.user_id = $1
  AND t.posted_on >= $2::date
  AND t.posted_on <= $3::date
  AND NOT t.pending
  AND (t.category IS NULL OR t.category NOT IN ('TRANSFER_IN', 'TRANSFER_OUT'))
ORDER BY merchant, t.posted_on
LIMIT $4;
`

func (r *TransactionQueryRepository) RecurringCandidates(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int) ([]repository.RecurringCandidate, error) {
	rows, err := r.pool.Query(ctx, listRecurringCandidates, userID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("recurring candidates: %w", err)
	}

	candidates, err := pgx.CollectRows(rows, pgx.RowToStructByName[repository.RecurringCandidate])
	if err != nil {
		return nil, fmt.Errorf("scan recurring candidates: %w", err)
	}

	return candidates, nil
}
