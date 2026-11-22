package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/marketdata"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

type fakeInvestments struct {
	repository.InvestmentRepository
	holdings []domain.InvestmentHolding
}

func (f *fakeInvestments) ListHoldingsByUser(context.Context, uuid.UUID, repository.InvestmentFilter) ([]domain.InvestmentHolding, error) {
	return f.holdings, nil
}

func (f *fakeInvestments) ListTransactionsSince(context.Context, uuid.UUID, time.Time) ([]domain.InvestmentTransaction, error) {
	return nil, nil
}

type unusedMarket struct {
	marketdata.Provider
}

func newInvestmentHandler(investments *fakeInvestments) *handler.InvestmentHandler {
	credentials := &fakeCredentials{
		find: func(context.Context, uuid.UUID, domain.Provider) (domain.Credential, error) {
			return domain.Credential{}, domain.ErrNotFound
		},
	}

	return handler.NewInvestmentHandler(investments, service.NewPortfolioService(investments, credentials, unusedMarket{}))
}

func TestInvestmentPerformanceWithoutMarketData(t *testing.T) {
	ticker, securityType, currency := "AAPL", "equity", "USD"
	investments := &fakeInvestments{holdings: []domain.InvestmentHolding{{
		AccountID: uuid.New(), AccountName: "Brokerage", SecurityID: uuid.New(), Ticker: &ticker, SecurityType: &securityType,
		InstitutionPrice: "100", Quantity: "3", Currency: &currency,
	}}}
	api := newAPI(t, newUser(), newInvestmentHandler(investments))

	rec := api.Get("/api/investments/performance?range=7d")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	performance := decode[handler.PortfolioPerformance](t, rec)

	if performance.Range != domain.PortfolioRangeWeek || performance.MarketData != domain.MarketDataNotConfigured || performance.EndValue != "300.00" {
		t.Errorf("performance = %+v", performance)
	}

	if len(performance.Positions) != 1 || len(performance.Accounts) != 1 || len(performance.Points) < 2 {
		t.Errorf("positions %d, accounts %d, points %d", len(performance.Positions), len(performance.Accounts), len(performance.Points))
	}
}

func TestInvestmentPerformanceRejectsBadInput(t *testing.T) {
	api := newAPI(t, newUser(), newInvestmentHandler(&fakeInvestments{}))

	for _, path := range []string{"/api/investments/performance?range=2w", "/api/investments/performance?securityId=nope"} {
		if rec := api.Get(path); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s status = %d, want %d", path, rec.Code, http.StatusUnprocessableEntity)
		}
	}
}
