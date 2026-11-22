package repository

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type Direction string

const (
	DirectionAny     Direction = ""
	DirectionInflow  Direction = "inflow"
	DirectionOutflow Direction = "outflow"
)

type TransactionQuery struct {
	From              time.Time
	To                time.Time
	AccountIDs        []uuid.UUID
	ExcludeAccountIDs []uuid.UUID
	ConnectionIDs     []uuid.UUID
	AccountTypes      []domain.AccountType
	Categories        []string
	ExcludeCategories []string
	ExcludeTransfers  bool
	Merchant          string
	Direction         Direction
	MinAmount         *string
	MaxAmount         *string
	Currency          string
	Pending           *bool
}

type TransactionSort string

const (
	SortDateDesc   TransactionSort = "date_desc"
	SortDateAsc    TransactionSort = "date_asc"
	SortAmountAsc  TransactionSort = "amount_asc"
	SortAmountDesc TransactionSort = "amount_desc"
)

type Dimension string

const (
	DimensionDay         Dimension = "day"
	DimensionWeek        Dimension = "week"
	DimensionMonth       Dimension = "month"
	DimensionQuarter     Dimension = "quarter"
	DimensionYear        Dimension = "year"
	DimensionWeekday     Dimension = "weekday"
	DimensionCategory    Dimension = "category"
	DimensionMerchant    Dimension = "merchant"
	DimensionAccount     Dimension = "account"
	DimensionConnection  Dimension = "connection"
	DimensionAccountType Dimension = "account_type"
	DimensionDirection   Dimension = "direction"
	DimensionPending     Dimension = "pending"
)

type Metric string

const (
	MetricSum     Metric = "sum"
	MetricInflow  Metric = "inflow"
	MetricOutflow Metric = "outflow"
	MetricCount   Metric = "count"
	MetricAverage Metric = "avg"
	MetricMin     Metric = "min"
	MetricMax     Metric = "max"
	MetricMedian  Metric = "median"
)

type AggregateSpec struct {
	Query      TransactionQuery
	GroupBy    []Dimension
	Metrics    []Metric
	OrderBy    string
	Descending bool
	Limit      int
}

type AggregateRow struct {
	Dimensions map[Dimension]string
	Currency   string
	Metrics    map[Metric]string
}

type Facet string

const (
	FacetCategory Facet = "category"
	FacetMerchant Facet = "merchant"
	FacetAccount  Facet = "account"
	FacetCurrency Facet = "currency"
)

type FacetRow struct {
	Value    string `db:"value"`
	Currency string `db:"currency"`
	Count    int    `db:"count"`
	Sum      string `db:"sum"`
}

type TransactionDetail struct {
	ID              uuid.UUID  `db:"id"`
	AccountID       uuid.UUID  `db:"account_id"`
	AccountName     string     `db:"account_name"`
	AccountType     string     `db:"account_type"`
	InstitutionName *string    `db:"institution_name"`
	Amount          string     `db:"amount"`
	Currency        *string    `db:"currency"`
	PostedOn        time.Time  `db:"posted_on"`
	AuthorizedOn    *time.Time `db:"authorized_on"`
	Name            string     `db:"name"`
	MerchantName    *string    `db:"merchant_name"`
	Category        *string    `db:"category"`
	Pending         bool       `db:"pending"`
}

type RecurringCandidate struct {
	Merchant string    `db:"merchant"`
	Amount   string    `db:"amount"`
	Currency string    `db:"currency"`
	PostedOn time.Time `db:"posted_on"`
}

type TransactionQueryRepository interface {
	Search(ctx context.Context, userID uuid.UUID, query TransactionQuery, sort TransactionSort, limit, offset int) ([]TransactionDetail, int, error)
	Get(ctx context.Context, userID, id uuid.UUID) (TransactionDetail, error)
	Aggregate(ctx context.Context, userID uuid.UUID, spec AggregateSpec) ([]AggregateRow, bool, error)
	Facets(ctx context.Context, userID uuid.UUID, query TransactionQuery, facet Facet, limit int) ([]FacetRow, error)
	RecurringCandidates(ctx context.Context, userID uuid.UUID, from, to time.Time, limit int) ([]RecurringCandidate, error)
}
