package service

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

func historyRow(at time.Time, accountID *uuid.UUID, currency, value string) domain.InvestmentHistoryRow {
	return domain.InvestmentHistoryRow{At: at, AccountID: accountID, Currency: currency, Value: value}
}

func TestSummarizeSeries(t *testing.T) {
	at := fixedNow

	tests := []struct {
		name        string
		values      []string
		startedLate bool
		wantChange  string
		wantPercent *string
		wantLimited bool
	}{
		{name: "gain", values: []string{"1000.0000", "1100.5000"}, wantChange: "100.5000", wantPercent: new("10.05")},
		{name: "loss", values: []string{"200", "150", "50"}, wantChange: "-150.0000", wantPercent: new("-75.00")},
		{name: "first value zero has no percent", values: []string{"0", "25"}, wantChange: "25.0000"},
		{name: "single point is limited", values: []string{"42.1234"}, wantChange: "0.0000", wantPercent: new("0.00"), wantLimited: true},
		{name: "collection started inside the range", values: []string{"10", "11"}, startedLate: true, wantChange: "1.0000", wantPercent: new("10.00"), wantLimited: true},
		{name: "negative first value keeps direction", values: []string{"-100", "-50"}, wantChange: "50.0000", wantPercent: new("50.00")},
		{name: "repeating percent rounds to two decimals", values: []string{"3", "4"}, wantChange: "1.0000", wantPercent: new("33.33")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			points := make([]domain.InvestmentPoint, len(test.values))

			for i, value := range test.values {
				points[i] = domain.InvestmentPoint{At: at.Add(time.Duration(i) * time.Hour), Value: value}
			}

			series, err := summarizeSeries("USD", points, test.startedLate)
			if err != nil {
				t.Fatalf("summarizeSeries: %v", err)
			}

			if series.Change != test.wantChange || !equalStringPtr(series.ChangePercent, test.wantPercent) || series.Limited != test.wantLimited {
				t.Errorf("summary = change %s percent %v limited %v, want %s %v %v", series.Change, deref(series.ChangePercent), series.Limited, test.wantChange, deref(test.wantPercent), test.wantLimited)
			}
		})
	}
}

func TestSummarizeSeriesRejectsNonDecimal(t *testing.T) {
	if _, err := summarizeSeries("USD", []domain.InvestmentPoint{{Value: "1e3x"}}, false); err == nil {
		t.Error("summarizeSeries accepted a non-decimal value")
	}
}

func TestBuildInvestmentSeriesKeepsCurrenciesAndAccountsApart(t *testing.T) {
	brokerage := uuid.MustParse("00000000-0000-7000-8000-000000000001")
	isa := uuid.MustParse("00000000-0000-7000-8000-000000000002")
	early, late := fixedNow, fixedNow.Add(time.Hour)

	rows := []domain.InvestmentHistoryRow{
		historyRow(early, nil, "GBP", "500"),
		historyRow(early, nil, "USD", "1000"),
		historyRow(early, &isa, "GBP", "500"),
		historyRow(early, &brokerage, "USD", "1000"),
		historyRow(late, nil, "GBP", "550"),
		historyRow(late, nil, "USD", "900"),
		historyRow(late, &isa, "GBP", "550"),
		historyRow(late, &brokerage, "USD", "900"),
	}

	total, accounts, err := buildInvestmentSeries(rows, false)
	if err != nil {
		t.Fatalf("buildInvestmentSeries: %v", err)
	}

	if len(total) != 2 || total[0].Currency != "GBP" || total[1].Currency != "USD" {
		t.Fatalf("total series = %+v, want one per currency", total)
	}

	if total[1].Current != "900.0000" || total[1].Change != "-100.0000" || len(total[1].Points) != 2 || !total[1].Points[0].At.Equal(early) {
		t.Errorf("USD total = %+v", total[1])
	}

	if len(accounts) != 2 || accounts[0].AccountID != brokerage || accounts[1].AccountID != isa {
		t.Fatalf("account series = %+v, want brokerage then ISA", accounts)
	}

	if len(accounts[1].Series) != 1 || accounts[1].Series[0].Currency != "GBP" || accounts[1].Series[0].Change != "50.0000" {
		t.Errorf("ISA series = %+v", accounts[1].Series)
	}
}

func TestHistoryWithoutSnapshotsReturnsEmptySeries(t *testing.T) {
	investments := &fakeInvestments{
		historyBounds: func(context.Context, uuid.UUID, repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			return domain.InvestmentHistoryBounds{}, nil
		},
		history: func(context.Context, uuid.UUID, repository.InvestmentHistoryQuery) ([]domain.InvestmentHistoryRow, error) {
			t.Error("history queried without any snapshots")

			return nil, nil
		},
	}

	service := NewInvestmentHistoryService(investments)
	service.now = fixedClock

	history, err := service.History(t.Context(), uuid.New(), InvestmentHistoryRequest{Range: domain.InvestmentRangeWeek})
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if history.Total == nil || history.Accounts == nil || len(history.Total) != 0 {
		t.Errorf("history = %+v, want empty non-nil series", history)
	}

	if history.From == nil || !history.From.Equal(fixedNow.Add(-7*24*time.Hour)) {
		t.Errorf("from = %v, want one week before now", history.From)
	}
}

func TestHistoryPassesRangeFiltersAndMarksLateCollection(t *testing.T) {
	userID, accountID, securityID := uuid.New(), uuid.New(), uuid.New()
	earliest := fixedNow.Add(-2 * 24 * time.Hour)
	filter := repository.InvestmentFilter{ExcludeAccountID: []uuid.UUID{accountID}}

	var got repository.InvestmentHistoryQuery

	investments := &fakeInvestments{
		historyBounds: func(_ context.Context, gotUser uuid.UUID, gotFilter repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			if gotUser != userID || len(gotFilter.ExcludeAccountID) != 1 {
				t.Errorf("bounds for (%v, %+v)", gotUser, gotFilter)
			}

			return domain.InvestmentHistoryBounds{Earliest: &earliest, Latest: &fixedNow}, nil
		},
		history: func(_ context.Context, _ uuid.UUID, query repository.InvestmentHistoryQuery) ([]domain.InvestmentHistoryRow, error) {
			got = query

			return []domain.InvestmentHistoryRow{historyRow(earliest, nil, "USD", "10"), historyRow(fixedNow, nil, "USD", "12")}, nil
		},
	}

	service := NewInvestmentHistoryService(investments)
	service.now = fixedClock

	history, err := service.History(t.Context(), userID, InvestmentHistoryRequest{
		Range:       domain.InvestmentRangeMonth,
		Filter:      filter,
		SecurityIDs: []uuid.UUID{securityID},
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if got.Bucket != 12*time.Hour || got.From == nil || !got.From.Equal(fixedNow.AddDate(0, -1, 0)) || len(got.SecurityIDs) != 1 || got.SecurityIDs[0] != securityID {
		t.Errorf("query = %+v", got)
	}

	if len(history.Total) != 1 || !history.Total[0].Limited || *history.Total[0].ChangePercent != "20.00" {
		t.Errorf("total = %+v, want a limited series because collection began inside the month", history.Total)
	}
}

func TestHistoryRepositoryFailurePropagates(t *testing.T) {
	failure := errors.New("database down")
	investments := &fakeInvestments{
		historyBounds: func(context.Context, uuid.UUID, repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			return domain.InvestmentHistoryBounds{}, failure
		},
	}

	if _, err := NewInvestmentHistoryService(investments).History(t.Context(), uuid.New(), InvestmentHistoryRequest{Range: domain.InvestmentRangeAll}); !errors.Is(err, failure) {
		t.Errorf("err = %v, want %v", err, failure)
	}
}

func TestAllRangeBucketStaysBounded(t *testing.T) {
	tests := []struct {
		name     string
		earliest time.Time
		want     time.Duration
	}{
		{name: "a week of history uses daily points", earliest: fixedNow.Add(-7 * 24 * time.Hour), want: 24 * time.Hour},
		{name: "two years of history uses whole-day buckets", earliest: fixedNow.AddDate(-2, 0, 0), want: 5 * 24 * time.Hour},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bucket := domain.InvestmentRangeAll.Bucket(test.earliest, fixedNow)

			if bucket != test.want {
				t.Errorf("bucket = %v, want %v", bucket, test.want)
			}

			if fixedNow.Sub(test.earliest)/bucket > 180 {
				t.Errorf("bucket %v yields more than 180 points", bucket)
			}
		})
	}
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func deref(value *string) string {
	if value == nil {
		return "<nil>"
	}

	return *value
}
