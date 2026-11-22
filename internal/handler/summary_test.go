package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
)

func day(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}

	return parsed
}

func TestCashFlowGroupsDaysByCurrency(t *testing.T) {
	user := newUser()

	summary := &fakeSummary{
		cashFlow: func(_ context.Context, userID uuid.UUID, _ int, _ []uuid.UUID, _ []domain.AccountType) ([]domain.DailyCashFlow, error) {
			if userID != user.ID {
				t.Errorf("userID = %s, want %s", userID, user.ID)
			}

			return []domain.DailyCashFlow{
				{Currency: "CAD", Day: day("2026-11-19"), In: "0", Out: "4.5000"},
				{Currency: "CAD", Day: day("2026-11-20"), In: "10.0000", Out: "0"},
				{Currency: "USD", Day: day("2026-11-19"), In: "2500.0000", Out: "61.2300"},
				{Currency: "USD", Day: day("2026-11-20"), In: "0", Out: "0"},
			}, nil
		},
	}
	api := newAPI(t, user, handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary/cash-flow")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	got := decode[handler.CashFlow](t, rec)
	want := handler.CashFlow{Currencies: []handler.CurrencyCashFlow{
		{Currency: "CAD", Days: []handler.CashFlowDay{
			{Date: "2026-11-19", In: "0", Out: "4.5000"},
			{Date: "2026-11-20", In: "10.0000", Out: "0"},
		}},
		{Currency: "USD", Days: []handler.CashFlowDay{
			{Date: "2026-11-19", In: "2500.0000", Out: "61.2300"},
			{Date: "2026-11-20", In: "0", Out: "0"},
		}},
	}}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("cash flow = %+v, want %+v", got, want)
	}
}

func TestCashFlowWithoutTransactionsIsEmptyList(t *testing.T) {
	summary := &fakeSummary{
		cashFlow: func(context.Context, uuid.UUID, int, []uuid.UUID, []domain.AccountType) ([]domain.DailyCashFlow, error) {
			return nil, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary/cash-flow")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	got := decode[struct {
		Currencies json.RawMessage `json:"currencies"`
	}](t, rec)

	if string(got.Currencies) != "[]" {
		t.Errorf("currencies = %s, want []", got.Currencies)
	}
}

func TestCashFlowDays(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		status int
		days   int
	}{
		{name: "defaults to 30", query: "", status: http.StatusOK, days: 30},
		{name: "accepts one day", query: "?days=1", status: http.StatusOK, days: 1},
		{name: "accepts a year", query: "?days=365", status: http.StatusOK, days: 365},
		{name: "rejects zero", query: "?days=0", status: http.StatusUnprocessableEntity},
		{name: "rejects more than a year", query: "?days=366", status: http.StatusUnprocessableEntity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false

			summary := &fakeSummary{
				cashFlow: func(_ context.Context, _ uuid.UUID, days int, _ []uuid.UUID, _ []domain.AccountType) ([]domain.DailyCashFlow, error) {
					called = true

					if days != tc.days {
						t.Errorf("days = %d, want %d", days, tc.days)
					}

					return nil, nil
				},
			}
			api := newAPI(t, newUser(), handler.NewSummaryHandler(summary))

			rec := api.Get("/api/summary/cash-flow" + tc.query)

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body.String())
			}

			if called != (tc.status == http.StatusOK) {
				t.Errorf("repository called = %t for status %d", called, tc.status)
			}
		})
	}
}

func TestSummaryExcludesAccounts(t *testing.T) {
	user := newUser()
	excludeIDs := []uuid.UUID{uuid.New(), uuid.New()}
	var gotExclude []uuid.UUID
	var gotTypes []domain.AccountType

	summary := &fakeSummary{
		balances: func(_ context.Context, userID uuid.UUID, excludeAccountIDs []uuid.UUID, excludeAccountTypes []domain.AccountType) ([]domain.CurrencyBalance, error) {
			if userID != user.ID {
				t.Errorf("userID = %s, want %s", userID, user.ID)
			}

			gotExclude = excludeAccountIDs
			gotTypes = excludeAccountTypes

			return nil, nil
		},
		counts: func(context.Context, uuid.UUID) (domain.SummaryCounts, error) {
			return domain.SummaryCounts{}, nil
		},
	}
	api := newAPI(t, user, handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary?excludeAccountId=" + excludeIDs[0].String() + "&excludeAccountId=" + excludeIDs[1].String() + "&excludeAccountType=investment")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if !slices.Equal(gotExclude, excludeIDs) {
		t.Errorf("excludeAccountIDs = %v, want %v", gotExclude, excludeIDs)
	}

	if !slices.Equal(gotTypes, []domain.AccountType{domain.AccountTypeInvestment}) {
		t.Errorf("excludeAccountTypes = %v, want [investment]", gotTypes)
	}
}

func TestSummaryRejectsUnknownExcludeAccountType(t *testing.T) {
	summary := &fakeSummary{
		balances: func(context.Context, uuid.UUID, []uuid.UUID, []domain.AccountType) ([]domain.CurrencyBalance, error) {
			t.Error("repository called for unknown account type")

			return nil, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary?excludeAccountType=crypto")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestSummaryRejectsInvalidExcludeAccountID(t *testing.T) {
	summary := &fakeSummary{
		balances: func(context.Context, uuid.UUID, []uuid.UUID, []domain.AccountType) ([]domain.CurrencyBalance, error) {
			t.Error("repository called for invalid exclude id")

			return nil, nil
		},
		counts: func(context.Context, uuid.UUID) (domain.SummaryCounts, error) {
			t.Error("repository called for invalid exclude id")

			return domain.SummaryCounts{}, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary?excludeAccountId=not-a-uuid")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCashFlowExcludesAccounts(t *testing.T) {
	user := newUser()
	excludeIDs := []uuid.UUID{uuid.New(), uuid.New()}
	var gotExclude []uuid.UUID
	var gotTypes []domain.AccountType

	summary := &fakeSummary{
		cashFlow: func(_ context.Context, userID uuid.UUID, _ int, excludeAccountIDs []uuid.UUID, excludeAccountTypes []domain.AccountType) ([]domain.DailyCashFlow, error) {
			if userID != user.ID {
				t.Errorf("userID = %s, want %s", userID, user.ID)
			}

			gotExclude = excludeAccountIDs
			gotTypes = excludeAccountTypes

			return nil, nil
		},
	}
	api := newAPI(t, user, handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary/cash-flow?excludeAccountId=" + excludeIDs[0].String() + "&excludeAccountId=" + excludeIDs[1].String() + "&excludeAccountType=investment")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if !slices.Equal(gotExclude, excludeIDs) {
		t.Errorf("excludeAccountIDs = %v, want %v", gotExclude, excludeIDs)
	}

	if !slices.Equal(gotTypes, []domain.AccountType{domain.AccountTypeInvestment}) {
		t.Errorf("excludeAccountTypes = %v, want [investment]", gotTypes)
	}
}

func TestCashFlowRejectsInvalidExcludeAccountID(t *testing.T) {
	summary := &fakeSummary{
		cashFlow: func(context.Context, uuid.UUID, int, []uuid.UUID, []domain.AccountType) ([]domain.DailyCashFlow, error) {
			t.Error("repository called for invalid exclude id")

			return nil, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewSummaryHandler(summary))

	rec := api.Get("/api/summary/cash-flow?excludeAccountId=not-a-uuid")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}
