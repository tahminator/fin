package service

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const (
	moneyDecimals   = 4
	percentDecimals = 2
)

type InvestmentHistoryRequest struct {
	Range       domain.InvestmentRange
	Filter      repository.InvestmentFilter
	SecurityIDs []uuid.UUID
}

type InvestmentHistoryService struct {
	investments repository.InvestmentRepository
	now         func() time.Time
}

func NewInvestmentHistoryService(investments repository.InvestmentRepository) *InvestmentHistoryService {
	return &InvestmentHistoryService{investments: investments, now: time.Now}
}

func (s *InvestmentHistoryService) History(ctx context.Context, userID uuid.UUID, request InvestmentHistoryRequest) (domain.InvestmentHistory, error) {
	now := s.now()
	from := request.Range.Start(now)

	history := domain.InvestmentHistory{
		Range:    request.Range,
		From:     from,
		Total:    []domain.InvestmentSeries{},
		Accounts: []domain.AccountInvestmentSeries{},
	}

	bounds, err := s.investments.HistoryBounds(ctx, userID, request.Filter)
	if err != nil {
		return domain.InvestmentHistory{}, err
	}

	history.Bounds = bounds

	if bounds.Earliest == nil {
		return history, nil
	}

	rows, err := s.investments.History(ctx, userID, repository.InvestmentHistoryQuery{
		Filter:      request.Filter,
		SecurityIDs: request.SecurityIDs,
		From:        from,
		Bucket:      request.Range.Bucket(*bounds.Earliest, now),
	})
	if err != nil {
		return domain.InvestmentHistory{}, err
	}

	collectionStartedInRange := from != nil && bounds.Earliest.After(*from)

	total, accounts, err := buildInvestmentSeries(rows, collectionStartedInRange)
	if err != nil {
		return domain.InvestmentHistory{}, err
	}

	history.Total = total
	history.Accounts = accounts

	return history, nil
}

type seriesKey struct {
	accountID uuid.UUID
	total     bool
	currency  string
}

func buildInvestmentSeries(rows []domain.InvestmentHistoryRow, collectionStartedInRange bool) ([]domain.InvestmentSeries, []domain.AccountInvestmentSeries, error) {
	points := make(map[seriesKey][]domain.InvestmentPoint)
	keys := make([]seriesKey, 0)

	for _, row := range rows {
		key := seriesKey{total: row.AccountID == nil, currency: row.Currency}

		if row.AccountID != nil {
			key.accountID = *row.AccountID
		}

		if _, seen := points[key]; !seen {
			keys = append(keys, key)
		}

		points[key] = append(points[key], domain.InvestmentPoint{At: row.At, Value: row.Value})
	}

	slices.SortStableFunc(keys, compareSeriesKeys)

	total := []domain.InvestmentSeries{}
	accounts := []domain.AccountInvestmentSeries{}

	for _, key := range keys {
		series, err := summarizeSeries(key.currency, points[key], collectionStartedInRange)
		if err != nil {
			return nil, nil, err
		}

		if key.total {
			total = append(total, series)

			continue
		}

		if last := len(accounts) - 1; last >= 0 && accounts[last].AccountID == key.accountID {
			accounts[last].Series = append(accounts[last].Series, series)

			continue
		}

		accounts = append(accounts, domain.AccountInvestmentSeries{AccountID: key.accountID, Series: []domain.InvestmentSeries{series}})
	}

	return total, accounts, nil
}

func compareSeriesKeys(a, b seriesKey) int {
	if a.total != b.total {
		if a.total {
			return -1
		}

		return 1
	}

	if order := a.accountID.Compare(b.accountID); order != 0 {
		return order
	}

	return cmp.Compare(a.currency, b.currency)
}

func summarizeSeries(currency string, points []domain.InvestmentPoint, collectionStartedInRange bool) (domain.InvestmentSeries, error) {
	series := domain.InvestmentSeries{
		Currency: currency,
		Points:   points,
		Limited:  collectionStartedInRange || len(points) < 2,
	}

	if len(points) == 0 {
		return series, nil
	}

	first, ok := new(big.Rat).SetString(points[0].Value)
	if !ok {
		return domain.InvestmentSeries{}, fmt.Errorf("invalid decimal %q", points[0].Value)
	}

	current, ok := new(big.Rat).SetString(points[len(points)-1].Value)
	if !ok {
		return domain.InvestmentSeries{}, fmt.Errorf("invalid decimal %q", points[len(points)-1].Value)
	}

	change := new(big.Rat).Sub(current, first)

	series.First = first.FloatString(moneyDecimals)
	series.Current = current.FloatString(moneyDecimals)
	series.Change = change.FloatString(moneyDecimals)

	if first.Sign() != 0 {
		percent := new(big.Rat).Quo(change, new(big.Rat).Abs(first))
		percent.Mul(percent, big.NewRat(100, 1))
		formatted := percent.FloatString(percentDecimals)
		series.ChangePercent = &formatted
	}

	return series, nil
}
