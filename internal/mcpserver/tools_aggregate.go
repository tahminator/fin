package mcpserver

import (
	"context"
	"math/big"
	"slices"

	"github.com/tahminator/fin/internal/repository"
)

const (
	defaultAggregateLimit = 200
	maxAggregateLimit     = 1000
	maxDimensions         = 3
)

var dimensionNames = []string{
	string(repository.DimensionDay),
	string(repository.DimensionWeek),
	string(repository.DimensionMonth),
	string(repository.DimensionQuarter),
	string(repository.DimensionYear),
	string(repository.DimensionWeekday),
	string(repository.DimensionCategory),
	string(repository.DimensionMerchant),
	string(repository.DimensionAccount),
	string(repository.DimensionConnection),
	string(repository.DimensionAccountType),
	string(repository.DimensionDirection),
	string(repository.DimensionPending),
}

var metricNames = []string{
	string(repository.MetricSum),
	string(repository.MetricInflow),
	string(repository.MetricOutflow),
	string(repository.MetricCount),
	string(repository.MetricAverage),
	string(repository.MetricMin),
	string(repository.MetricMax),
	string(repository.MetricMedian),
}

type AggregateTransactionsInput struct {
	TransactionFilter
	GroupBy []string `json:"group_by,omitempty" jsonschema:"Up to 3 dimensions to group by. Rows are always also split by currency."`
	Metrics []string `json:"metrics" jsonschema:"What to compute per group. sum is net (in minus out); inflow and outflow are both positive; count is the number of transactions."`
	OrderBy string   `json:"order_by,omitempty" jsonschema:"A requested dimension or metric to sort by. Defaults to the first dimension."`
	Order   string   `json:"order,omitempty" jsonschema:"asc or desc. Default asc."`
	Limit   int      `json:"limit,omitempty" jsonschema:"Maximum rows, up to 1000. Default 200."`
}

type AggregateRowOutput struct {
	Group    map[string]string `json:"group" jsonschema:"Dimension values for this row"`
	Currency string            `json:"currency"`
	Values   map[string]string `json:"values" jsonschema:"Metric values as decimal strings"`
}

type AggregateTransactionsOutput struct {
	Rows      []AggregateRowOutput `json:"rows"`
	Truncated bool                 `json:"truncated" jsonschema:"True when more rows matched than limit"`
}

func (t *toolset) aggregateTransactions(ctx context.Context, input AggregateTransactionsInput) (AggregateTransactionsOutput, error) {
	spec, err := aggregateSpec(input, t)
	if err != nil {
		return AggregateTransactionsOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, truncated, err := t.deps.Transactions.Aggregate(ctx, t.userID(), spec)
	if err != nil {
		return AggregateTransactionsOutput{}, err
	}

	output := AggregateTransactionsOutput{Rows: make([]AggregateRowOutput, 0, len(rows)), Truncated: truncated}

	for _, row := range rows {
		group := make(map[string]string, len(row.Dimensions))

		for dimension, value := range row.Dimensions {
			group[string(dimension)] = value
		}

		values := make(map[string]string, len(row.Metrics))

		for metric, value := range row.Metrics {
			values[string(metric)] = value
		}

		output.Rows = append(output.Rows, AggregateRowOutput{Group: group, Currency: row.Currency, Values: values})
	}

	return output, nil
}

func aggregateSpec(input AggregateTransactionsInput, t *toolset) (repository.AggregateSpec, error) {
	query, err := input.query(t.deps.Now(), false)
	if err != nil {
		return repository.AggregateSpec{}, err
	}

	if len(input.GroupBy) > maxDimensions {
		return repository.AggregateSpec{}, invalidParams("group_by may hold at most %d dimensions", maxDimensions)
	}

	if len(input.Metrics) == 0 {
		return repository.AggregateSpec{}, invalidParams("metrics needs at least one of %v", metricNames)
	}

	spec := repository.AggregateSpec{Query: query, OrderBy: input.OrderBy, Descending: input.Order == "desc"}

	for _, name := range input.GroupBy {
		if !slices.Contains(dimensionNames, name) || slices.Contains(spec.GroupBy, repository.Dimension(name)) {
			return repository.AggregateSpec{}, invalidParams("group_by values must be distinct values from %v", dimensionNames)
		}

		spec.GroupBy = append(spec.GroupBy, repository.Dimension(name))
	}

	for _, name := range input.Metrics {
		if !slices.Contains(metricNames, name) || slices.Contains(spec.Metrics, repository.Metric(name)) {
			return repository.AggregateSpec{}, invalidParams("metrics must be distinct values from %v", metricNames)
		}

		spec.Metrics = append(spec.Metrics, repository.Metric(name))
	}

	if input.OrderBy != "" && !slices.Contains(input.GroupBy, input.OrderBy) && !slices.Contains(input.Metrics, input.OrderBy) {
		return repository.AggregateSpec{}, invalidParams("order_by must be one of the requested group_by or metrics values")
	}

	if input.Order != "" && input.Order != "asc" && input.Order != "desc" {
		return repository.AggregateSpec{}, invalidParams("order must be asc or desc")
	}

	spec.Limit = input.Limit
	if spec.Limit == 0 {
		spec.Limit = defaultAggregateLimit
	}

	if spec.Limit < 1 || spec.Limit > maxAggregateLimit {
		return repository.AggregateSpec{}, invalidParams("limit must be between 1 and %d", maxAggregateLimit)
	}

	return spec, nil
}

type CashFlowInput struct {
	TransactionFilter
	Interval string `json:"interval,omitempty" jsonschema:"day, week, month (default), quarter or year. Transfers between accounts are excluded unless exclude_transfers is false."`
}

type CashFlowPeriod struct {
	Period      string  `json:"period" jsonschema:"First day of the period"`
	Currency    string  `json:"currency"`
	Inflow      string  `json:"inflow"`
	Outflow     string  `json:"outflow" jsonschema:"Positive amount of money out"`
	Net         string  `json:"net"`
	SavingsRate *string `json:"savings_rate" jsonschema:"net / inflow, as a decimal fraction; null when there was no inflow"`
}

type CashFlowOutput struct {
	Interval string           `json:"interval"`
	Periods  []CashFlowPeriod `json:"periods"`
}

var intervals = map[string]repository.Dimension{
	"":        repository.DimensionMonth,
	"day":     repository.DimensionDay,
	"week":    repository.DimensionWeek,
	"month":   repository.DimensionMonth,
	"quarter": repository.DimensionQuarter,
	"year":    repository.DimensionYear,
}

func (t *toolset) cashFlow(ctx context.Context, input CashFlowInput) (CashFlowOutput, error) {
	query, err := input.query(t.deps.Now(), true)
	if err != nil {
		return CashFlowOutput{}, err
	}

	dimension, ok := intervals[input.Interval]
	if !ok {
		return CashFlowOutput{}, invalidParams("interval must be day, week, month, quarter or year")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, _, err := t.deps.Transactions.Aggregate(ctx, t.userID(), repository.AggregateSpec{
		Query:   query,
		GroupBy: []repository.Dimension{dimension},
		Metrics: []repository.Metric{repository.MetricInflow, repository.MetricOutflow, repository.MetricSum},
		Limit:   maxAggregateLimit,
	})
	if err != nil {
		return CashFlowOutput{}, err
	}

	output := CashFlowOutput{Interval: string(dimension), Periods: make([]CashFlowPeriod, 0, len(rows))}

	for _, row := range rows {
		inflow := row.Metrics[repository.MetricInflow]
		net := row.Metrics[repository.MetricSum]

		output.Periods = append(output.Periods, CashFlowPeriod{
			Period:      row.Dimensions[dimension],
			Currency:    row.Currency,
			Inflow:      inflow,
			Outflow:     row.Metrics[repository.MetricOutflow],
			Net:         net,
			SavingsRate: ratio(net, inflow),
		})
	}

	return output, nil
}

func ratio(numerator, denominator string) *string {
	top, okTop := new(big.Rat).SetString(numerator)
	bottom, okBottom := new(big.Rat).SetString(denominator)

	if !okTop || !okBottom || bottom.Sign() == 0 {
		return nil
	}

	formatted := new(big.Rat).Quo(top, bottom).FloatString(4)

	return &formatted
}
