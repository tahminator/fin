package postgres

import (
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/repository"
)

func TestBuildAggregateQueryAlwaysScopesToTheUserFirst(t *testing.T) {
	userID := uuid.New()

	statement, args, err := buildAggregateQuery(userID, repository.AggregateSpec{
		Query:   repository.TransactionQuery{Merchant: "50%_off\\", ExcludeTransfers: true, From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		GroupBy: []repository.Dimension{repository.DimensionMonth, repository.DimensionCategory},
		Metrics: []repository.Metric{repository.MetricOutflow, repository.MetricCount},
		OrderBy: "outflow",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("buildAggregateQuery: %v", err)
	}

	if args[0] != userID || !strings.Contains(statement, "WHERE t.user_id = $1") {
		t.Fatalf("user id is not $1:\n%s", statement)
	}

	for _, fragment := range []string{
		"date_trunc('month', t.posted_on)::date::text AS d0",
		"coalesce(t.category, 'UNCATEGORIZED') AS d1",
		"coalesce(t.currency, '') AS currency",
		"GROUP BY 1, 2, 3",
		"coalesce(-sum(t.amount) FILTER (WHERE t.amount < 0), 0)::text AS m0",
		"count(*)::text AS m1",
		"ESCAPE",
		"<> ALL(",
	} {
		if !strings.Contains(statement, fragment) {
			t.Errorf("statement is missing %q:\n%s", fragment, statement)
		}
	}

	if !slices.Contains(args, any(`%50\%\_off\\%`)) {
		t.Errorf("merchant pattern not escaped: %v", args)
	}

	if args[len(args)-1] != 11 {
		t.Errorf("limit arg = %v, want limit+1 to detect truncation", args[len(args)-1])
	}

	if strings.Contains(statement, "50%") {
		t.Error("a filter value was interpolated into the SQL")
	}
}

func TestBuildAggregateQueryRejectsUnknownIdentifiers(t *testing.T) {
	tests := map[string]repository.AggregateSpec{
		"dimension": {GroupBy: []repository.Dimension{"t.user_id; DROP TABLE users"}, Metrics: []repository.Metric{repository.MetricSum}},
		"metric":    {Metrics: []repository.Metric{"sum(1)"}},
		"order by":  {Metrics: []repository.Metric{repository.MetricSum}, OrderBy: "count"},
	}

	for name, spec := range tests {
		if _, _, err := buildAggregateQuery(uuid.New(), spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestEveryDimensionAndMetricHasAnExpression(t *testing.T) {
	dimensions := []repository.Dimension{
		repository.DimensionDay, repository.DimensionWeek, repository.DimensionMonth, repository.DimensionQuarter,
		repository.DimensionYear, repository.DimensionWeekday, repository.DimensionCategory, repository.DimensionMerchant,
		repository.DimensionAccount, repository.DimensionConnection, repository.DimensionAccountType,
		repository.DimensionDirection, repository.DimensionPending,
	}

	for _, dimension := range dimensions {
		if _, ok := dimensionExpressions[dimension]; !ok {
			t.Errorf("dimension %s has no expression", dimension)
		}
	}

	metrics := []repository.Metric{
		repository.MetricSum, repository.MetricInflow, repository.MetricOutflow, repository.MetricCount,
		repository.MetricAverage, repository.MetricMin, repository.MetricMax, repository.MetricMedian,
	}

	for _, metric := range metrics {
		if _, ok := metricExpressions[metric]; !ok {
			t.Errorf("metric %s has no expression", metric)
		}
	}
}

func TestTransactionWhereBindsEveryFilter(t *testing.T) {
	pending := true
	minimum := "10"
	args := newQueryArgs(uuid.New())

	where := buildTransactionWhere(repository.TransactionQuery{
		AccountIDs:    []uuid.UUID{uuid.New()},
		ConnectionIDs: []uuid.UUID{uuid.New()},
		Categories:    []string{"FOOD_AND_DRINK"},
		Direction:     repository.DirectionOutflow,
		MinAmount:     &minimum,
		Currency:      "USD",
		Pending:       &pending,
	}, args)

	if strings.Count(where, "$") != len(args.values) {
		t.Errorf("placeholders %d != args %d:\n%s", strings.Count(where, "$"), len(args.values), where)
	}

	if !strings.Contains(where, "t.amount < 0") || !strings.Contains(where, "abs(t.amount) >=") {
		t.Errorf("where = %s", where)
	}
}
