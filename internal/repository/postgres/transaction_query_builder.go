package postgres

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/tahminator/fin/internal/repository"
)

const transactionQueryFrom = `
FROM transactions AS t
JOIN accounts AS a ON a.id = t.account_id
JOIN connections AS c ON c.id = a.connection_id`

var transferCategories = []string{"TRANSFER_IN", "TRANSFER_OUT", "LOAN_PAYMENTS"}

var dimensionExpressions = map[repository.Dimension]string{
	repository.DimensionDay:         "t.posted_on::text",
	repository.DimensionWeek:        "date_trunc('week', t.posted_on)::date::text",
	repository.DimensionMonth:       "date_trunc('month', t.posted_on)::date::text",
	repository.DimensionQuarter:     "date_trunc('quarter', t.posted_on)::date::text",
	repository.DimensionYear:        "extract(year FROM t.posted_on)::int::text",
	repository.DimensionWeekday:     "to_char(t.posted_on, 'FMDay')",
	repository.DimensionCategory:    "coalesce(t.category, 'UNCATEGORIZED')",
	repository.DimensionMerchant:    "coalesce(t.merchant_name, t.name)",
	repository.DimensionAccount:     "a.name || coalesce(' ••' || a.mask, '')",
	repository.DimensionConnection:  "coalesce(c.institution_name, 'Unknown institution')",
	repository.DimensionAccountType: "a.type",
	repository.DimensionDirection:   "CASE WHEN t.amount < 0 THEN 'outflow' ELSE 'inflow' END",
	repository.DimensionPending:     "CASE WHEN t.pending THEN 'pending' ELSE 'posted' END",
}

var metricExpressions = map[repository.Metric]string{
	repository.MetricSum:     "sum(t.amount)::text",
	repository.MetricInflow:  "coalesce(sum(t.amount) FILTER (WHERE t.amount > 0), 0)::text",
	repository.MetricOutflow: "coalesce(-sum(t.amount) FILTER (WHERE t.amount < 0), 0)::text",
	repository.MetricCount:   "count(*)::text",
	repository.MetricAverage: "round(avg(t.amount), 4)::text",
	repository.MetricMin:     "min(t.amount)::text",
	repository.MetricMax:     "max(t.amount)::text",
	repository.MetricMedian:  "round(percentile_cont(0.5) WITHIN GROUP (ORDER BY t.amount)::numeric, 4)::text",
}

var facetExpressions = map[repository.Facet]string{
	repository.FacetCategory: dimensionExpressions[repository.DimensionCategory],
	repository.FacetMerchant: dimensionExpressions[repository.DimensionMerchant],
	repository.FacetAccount:  dimensionExpressions[repository.DimensionAccount],
	repository.FacetCurrency: "coalesce(t.currency, '')",
}

var sortExpressions = map[repository.TransactionSort]string{
	repository.SortDateDesc:   "t.posted_on DESC, t.id DESC",
	repository.SortDateAsc:    "t.posted_on ASC, t.id ASC",
	repository.SortAmountAsc:  "t.amount ASC, t.id ASC",
	repository.SortAmountDesc: "t.amount DESC, t.id DESC",
}

type queryArgs struct {
	values []any
}

func newQueryArgs(userID uuid.UUID) *queryArgs {
	return &queryArgs{values: []any{userID}}
}

func (a *queryArgs) add(value any) string {
	a.values = append(a.values, value)

	return "$" + strconv.Itoa(len(a.values))
}

func buildTransactionWhere(query repository.TransactionQuery, args *queryArgs) string {
	conditions := []string{"t.user_id = $1"}

	if !query.From.IsZero() {
		conditions = append(conditions, "t.posted_on >= "+args.add(query.From)+"::date")
	}

	if !query.To.IsZero() {
		conditions = append(conditions, "t.posted_on <= "+args.add(query.To)+"::date")
	}

	if len(query.AccountIDs) > 0 {
		conditions = append(conditions, "t.account_id = ANY("+args.add(query.AccountIDs)+"::uuid[])")
	}

	if len(query.ExcludeAccountIDs) > 0 {
		conditions = append(conditions, "t.account_id <> ALL("+args.add(query.ExcludeAccountIDs)+"::uuid[])")
	}

	if len(query.ConnectionIDs) > 0 {
		conditions = append(conditions, "a.connection_id = ANY("+args.add(query.ConnectionIDs)+"::uuid[])")
	}

	if len(query.AccountTypes) > 0 {
		types := make([]string, 0, len(query.AccountTypes))

		for _, accountType := range query.AccountTypes {
			types = append(types, string(accountType))
		}

		conditions = append(conditions, "a.type = ANY("+args.add(types)+"::text[])")
	}

	if len(query.Categories) > 0 {
		conditions = append(conditions, "coalesce(t.category, 'UNCATEGORIZED') = ANY("+args.add(query.Categories)+"::text[])")
	}

	excluded := slices.Clone(query.ExcludeCategories)
	if query.ExcludeTransfers {
		excluded = append(excluded, transferCategories...)
	}

	if len(excluded) > 0 {
		conditions = append(conditions, "coalesce(t.category, 'UNCATEGORIZED') <> ALL("+args.add(excluded)+"::text[])")
	}

	if query.Merchant != "" {
		pattern := args.add("%" + escapeLike(query.Merchant) + "%")
		conditions = append(conditions, fmt.Sprintf("(t.merchant_name ILIKE %[1]s ESCAPE '\\' OR t.name ILIKE %[1]s ESCAPE '\\')", pattern))
	}

	switch query.Direction {
	case repository.DirectionInflow:
		conditions = append(conditions, "t.amount > 0")
	case repository.DirectionOutflow:
		conditions = append(conditions, "t.amount < 0")
	case repository.DirectionAny:
	}

	if query.MinAmount != nil {
		conditions = append(conditions, "abs(t.amount) >= "+args.add(*query.MinAmount)+"::numeric")
	}

	if query.MaxAmount != nil {
		conditions = append(conditions, "abs(t.amount) <= "+args.add(*query.MaxAmount)+"::numeric")
	}

	if query.Currency != "" {
		conditions = append(conditions, "t.currency = "+args.add(query.Currency))
	}

	if query.Pending != nil {
		conditions = append(conditions, "t.pending = "+args.add(*query.Pending))
	}

	return "WHERE " + strings.Join(conditions, "\n  AND ")
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func buildAggregateQuery(userID uuid.UUID, spec repository.AggregateSpec) (string, []any, error) {
	args := newQueryArgs(userID)
	where := buildTransactionWhere(spec.Query, args)

	selects := make([]string, 0, len(spec.GroupBy)+len(spec.Metrics)+1)
	groups := make([]string, 0, len(spec.GroupBy)+1)
	aliases := map[string]string{}

	for i, dimension := range spec.GroupBy {
		expression, ok := dimensionExpressions[dimension]
		if !ok {
			return "", nil, fmt.Errorf("unknown dimension %q", dimension)
		}

		alias := "d" + strconv.Itoa(i)
		selects = append(selects, expression+" AS "+alias)
		groups = append(groups, strconv.Itoa(i+1))
		aliases[string(dimension)] = alias
	}

	selects = append(selects, "coalesce(t.currency, '') AS currency")
	groups = append(groups, strconv.Itoa(len(spec.GroupBy)+1))

	for i, metric := range spec.Metrics {
		expression, ok := metricExpressions[metric]
		if !ok {
			return "", nil, fmt.Errorf("unknown metric %q", metric)
		}

		alias := "m" + strconv.Itoa(i)
		selects = append(selects, expression+" AS "+alias)
		aliases[string(metric)] = alias
	}

	order, err := aggregateOrder(spec, aliases)
	if err != nil {
		return "", nil, err
	}

	query := "SELECT\n    " + strings.Join(selects, ",\n    ") +
		transactionQueryFrom + "\n" + where +
		"\nGROUP BY " + strings.Join(groups, ", ") +
		"\nORDER BY " + order +
		"\nLIMIT " + args.add(spec.Limit+1) + ";"

	return query, args.values, nil
}

func aggregateOrder(spec repository.AggregateSpec, aliases map[string]string) (string, error) {
	direction := " ASC NULLS LAST"
	if spec.Descending {
		direction = " DESC NULLS LAST"
	}

	if spec.OrderBy != "" {
		alias, ok := aliases[spec.OrderBy]
		if !ok {
			return "", fmt.Errorf("order_by %q must be one of the requested dimensions or metrics", spec.OrderBy)
		}

		if metric := repository.Metric(spec.OrderBy); slices.Contains(spec.Metrics, metric) {
			return "(" + metricExpressions[metric] + ")::numeric" + direction + ", currency", nil
		}

		return alias + direction + ", currency", nil
	}

	if len(spec.GroupBy) > 0 {
		return aliases[string(spec.GroupBy[0])] + " ASC, currency", nil
	}

	return "currency", nil
}
