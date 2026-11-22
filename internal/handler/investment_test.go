package handler_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/repository"
)

func TestInvestmentListsApplyAccountFilters(t *testing.T) {
	accountID, hiddenA, hiddenB := uuid.New(), uuid.New(), uuid.New()
	query := "?accountId=" + accountID.String() + "&excludeAccountId=" + hiddenA.String() + "&excludeAccountId=" + hiddenB.String()

	var filters []repository.InvestmentFilter

	user := newUser()
	record := func(userID uuid.UUID, filter repository.InvestmentFilter) {
		if userID != user.ID {
			t.Errorf("repository called for user %v, want %v", userID, user.ID)
		}

		filters = append(filters, filter)
	}

	api := newInvestmentAPI(t, user, &fakeInvestments{
		listHoldings: func(_ context.Context, userID uuid.UUID, filter repository.InvestmentFilter) ([]domain.InvestmentHolding, error) {
			record(userID, filter)

			return nil, nil
		},
		listTransactions: func(_ context.Context, userID uuid.UUID, filter repository.InvestmentFilter, _, _ int) ([]domain.InvestmentTransaction, int, error) {
			record(userID, filter)

			return nil, 0, nil
		},
		listSecurities: func(_ context.Context, userID uuid.UUID, filter repository.InvestmentFilter) ([]domain.InvestmentSecuritySummary, error) {
			record(userID, filter)

			return nil, nil
		},
		historyBounds: func(_ context.Context, userID uuid.UUID, filter repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			record(userID, filter)

			return domain.InvestmentHistoryBounds{}, nil
		},
	})

	for _, path := range []string{"/api/investments/holdings", "/api/investments/transactions", "/api/investments/securities", "/api/investments/history"} {
		suffix := query

		if path == "/api/investments/history" {
			suffix += "&range=1w"
		}

		rec := api.Get(path + suffix)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d; body %s", path, rec.Code, rec.Body.String())
		}

		if body := rec.Body.String(); body == "null\n" || body == "null" {
			t.Errorf("%s returned null, want an empty array or object", path)
		}
	}

	if len(filters) != 4 {
		t.Fatalf("repository saw %d filters, want 4", len(filters))
	}

	for _, filter := range filters {
		if filter.AccountID == nil || *filter.AccountID != accountID || !slices.Equal(filter.ExcludeAccountID, []uuid.UUID{hiddenA, hiddenB}) {
			t.Errorf("filter = %+v, want account %v excluding %v and %v", filter, accountID, hiddenA, hiddenB)
		}
	}
}

func TestInvestmentTransactionsPaginate(t *testing.T) {
	var gotLimit, gotOffset int

	transaction := domain.InvestmentTransaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		AccountName: "Brokerage",
		Ticker:      new("SOLD"),
		PostedOn:    time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC),
		Name:        "SELL Sold Fund",
		Amount:      "510.2500",
		Fees:        new("1.2500"),
		Type:        "sell",
	}

	api := newInvestmentAPI(t, newUser(), &fakeInvestments{
		listTransactions: func(_ context.Context, _ uuid.UUID, _ repository.InvestmentFilter, limit, offset int) ([]domain.InvestmentTransaction, int, error) {
			gotLimit, gotOffset = limit, offset

			return []domain.InvestmentTransaction{transaction}, 21, nil
		},
	})

	rec := api.Get("/api/investments/transactions?page=2&pageSize=10")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", rec.Code, rec.Body.String())
	}

	page := decode[handler.Page[handler.InvestmentTransaction]](t, rec)

	if gotLimit != 10 || gotOffset != 10 || !page.HasNext || page.Total != 21 {
		t.Errorf("page = %+v with limit %d offset %d, want page 2 of 21 with more", page, gotLimit, gotOffset)
	}

	if len(page.Items) != 1 || page.Items[0].PostedOn != "2026-03-02" || page.Items[0].Amount != "510.2500" || *page.Items[0].Ticker != "SOLD" {
		t.Errorf("items = %+v", page.Items)
	}
}

func TestInvestmentEndpointsRejectInvalidInput(t *testing.T) {
	api := newInvestmentAPI(t, newUser(), &fakeInvestments{})

	for _, path := range []string{
		"/api/investments/holdings?accountId=nope",
		"/api/investments/transactions?excludeAccountId=nope",
		"/api/investments/history",
		"/api/investments/history?range=2y",
		"/api/investments/history?range=1d&securityId=nope",
	} {
		if rec := api.Get(path); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s status = %d, want 422; body %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestInvestmentFailuresAreServerErrors(t *testing.T) {
	failure := errors.New("database down")

	api := newInvestmentAPI(t, newUser(), &fakeInvestments{
		listHoldings: func(context.Context, uuid.UUID, repository.InvestmentFilter) ([]domain.InvestmentHolding, error) {
			return nil, failure
		},
		historyBounds: func(context.Context, uuid.UUID, repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			return domain.InvestmentHistoryBounds{}, failure
		},
	})

	for _, path := range []string{"/api/investments/holdings", "/api/investments/history?range=all"} {
		if rec := api.Get(path); rec.Code != http.StatusInternalServerError {
			t.Errorf("%s status = %d, want 500", path, rec.Code)
		}
	}
}

func TestInvestmentHistoryPassesSelectedSecurities(t *testing.T) {
	securityA, securityB := uuid.New(), uuid.New()
	earliest := time.Now().Add(-time.Hour)

	var got repository.InvestmentHistoryQuery

	api := newInvestmentAPI(t, newUser(), &fakeInvestments{
		historyBounds: func(context.Context, uuid.UUID, repository.InvestmentFilter) (domain.InvestmentHistoryBounds, error) {
			return domain.InvestmentHistoryBounds{Earliest: &earliest, Latest: &earliest}, nil
		},
		history: func(_ context.Context, _ uuid.UUID, query repository.InvestmentHistoryQuery) ([]domain.InvestmentHistoryRow, error) {
			got = query

			return []domain.InvestmentHistoryRow{{At: earliest, Currency: "USD", Value: "10.0000"}}, nil
		},
	})

	rec := api.Get("/api/investments/history?range=1d&securityId=" + securityA.String() + "&securityId=" + securityB.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", rec.Code, rec.Body.String())
	}

	if !slices.Equal(got.SecurityIDs, []uuid.UUID{securityA, securityB}) || got.Bucket != 15*time.Minute {
		t.Errorf("query = %+v", got)
	}

	history := decode[handler.InvestmentHistory](t, rec)

	if history.Range != domain.InvestmentRangeDay || len(history.Total) != 1 || !history.Total[0].Limited || history.Accounts == nil {
		t.Errorf("history = %+v", history)
	}
}
