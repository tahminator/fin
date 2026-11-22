package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/marketdata"
	"github.com/tahminator/fin/internal/repository"
)

var portfolioNow = time.Date(2026, time.November, 20, 20, 0, 0, 0, time.UTC)

type portfolioFixture struct {
	userID       uuid.UUID
	accountID    uuid.UUID
	appleID      uuid.UUID
	cashID       uuid.UUID
	holdings     []domain.InvestmentHolding
	transactions []domain.InvestmentTransaction
	credErr      error
	barsErr      error
	bars         map[string][]marketdata.Bar
	trades       map[string]marketdata.Trade
	gotSymbols   []string
	gotStart     time.Time
	gotEnd       time.Time
	barsCalled   bool
	service      *PortfolioService
}

func ptr[T any](value T) *T {
	return &value
}

func newPortfolioFixture() *portfolioFixture {
	f := &portfolioFixture{userID: uuid.New(), accountID: uuid.New(), appleID: uuid.New(), cashID: uuid.New()}

	f.holdings = []domain.InvestmentHolding{
		{AccountID: f.accountID, AccountName: "Brokerage", SecurityID: f.appleID, SecurityName: ptr("Apple"), Ticker: ptr("AAPL"), SecurityType: ptr("equity"), InstitutionPrice: "115", Quantity: "10", CostBasis: ptr("1000"), Currency: ptr("USD")},
		{AccountID: f.accountID, AccountName: "Brokerage", SecurityID: f.cashID, SecurityName: ptr("Cash"), Ticker: ptr("CUR:USD"), SecurityType: ptr("cash"), InstitutionPrice: "1", Quantity: "50", Currency: ptr("USD")},
	}

	f.transactions = []domain.InvestmentTransaction{
		{AccountID: f.accountID, AccountName: "Brokerage", SecurityID: &f.appleID, Ticker: ptr("AAPL"), SecurityType: ptr("equity"), PostedOn: time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC), Quantity: ptr("4"), Amount: "-440", Type: "buy", Subtype: "buy"},
	}

	f.bars = map[string][]marketdata.Bar{
		"AAPL": {
			{Time: time.Date(2026, time.November, 19, 19, 55, 0, 0, time.UTC), Close: "100"},
			{Time: time.Date(2026, time.November, 20, 15, 0, 0, 0, time.UTC), Close: "110"},
		},
	}

	f.trades = map[string]marketdata.Trade{"AAPL": {Time: portfolioNow, Price: "120"}}

	investments := &fakeInvestments{
		listHoldingsByUser: func(_ context.Context, userID uuid.UUID, filter repository.InvestmentFilter) ([]domain.InvestmentHolding, error) {
			if userID != f.userID {
				return nil, errors.New("wrong user")
			}

			var holdings []domain.InvestmentHolding

			for _, holding := range f.holdings {
				if filter.AccountID == nil || holding.AccountID == *filter.AccountID {
					holdings = append(holdings, holding)
				}
			}

			return holdings, nil
		},
		listTransactionsSince: func(context.Context, uuid.UUID, time.Time) ([]domain.InvestmentTransaction, error) {
			return f.transactions, nil
		},
	}

	credentials := &fakeCredentials{
		find: func(_ context.Context, _ uuid.UUID, provider domain.Provider) (domain.Credential, error) {
			if provider != domain.ProviderAlpaca {
				return domain.Credential{}, errors.New("wrong provider")
			}

			if f.credErr != nil {
				return domain.Credential{}, f.credErr
			}

			return domain.Credential{Provider: domain.ProviderAlpaca, ClientID: "key", Secret: "secret"}, nil
		},
	}

	market := &fakeMarket{
		bars: func(_ context.Context, _ domain.Credential, symbols []string, _ marketdata.Timeframe, start, end time.Time) (map[string][]marketdata.Bar, error) {
			f.barsCalled = true
			f.gotSymbols, f.gotStart, f.gotEnd = symbols, start, end

			return f.bars, f.barsErr
		},
		latestTrades: func(context.Context, domain.Credential, []string) (map[string]marketdata.Trade, error) {
			return f.trades, nil
		},
	}

	f.service = NewPortfolioService(investments, credentials, market)
	f.service.now = func() time.Time { return portfolioNow }

	return f
}

func pointValues(points []domain.PortfolioPoint) []string {
	values := make([]string, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	return values
}

func TestPerformanceValuesRebuiltQuantitiesAtMarketPrices(t *testing.T) {
	f := newPortfolioFixture()

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if want := []string{"1090.00", "1150.00", "1250.00"}; !slices.Equal(pointValues(performance.Points), want) {
		t.Errorf("points = %v, want cash rebuilt before the buy: %v", pointValues(performance.Points), want)
	}

	if performance.Change != "160.00" || performance.ChangePercent == nil || *performance.ChangePercent != "15.38" {
		t.Errorf("change = %s (%v), want market gain 160.00 (15.38)", performance.Change, performance.ChangePercent)
	}

	if performance.MarketData != domain.MarketDataLive {
		t.Errorf("market data = %s, want live", performance.MarketData)
	}

	if !slices.Equal(f.gotSymbols, []string{"AAPL"}) {
		t.Errorf("symbols = %v, want only AAPL", f.gotSymbols)
	}

	if !f.gotEnd.Equal(portfolioNow.Add(-sipDelay)) || !f.gotStart.Equal(portfolioNow.Add(-24*time.Hour-intradayLookup)) {
		t.Errorf("bars window = %v..%v", f.gotStart, f.gotEnd)
	}

	if len(performance.Positions) != 2 {
		t.Fatalf("positions = %+v, want AAPL and cash", performance.Positions)
	}

	apple := performance.Positions[0]
	if apple.SecurityID != f.appleID || apple.Value != "1200.00" || apple.StartValue != "600.00" || apple.Change != "160.00" || apple.Quantity != "10" || apple.Price != "120.00" || !apple.MarketPriced {
		t.Errorf("AAPL = %+v", apple)
	}

	if apple.AverageCost == nil || *apple.AverageCost != "100.00" || apple.TotalReturn == nil || *apple.TotalReturn != "200.00" || apple.TotalReturnPercent == nil || *apple.TotalReturnPercent != "20.00" {
		t.Errorf("AAPL returns = %v %v %v, want 100.00, 200.00, 20.00", apple.AverageCost, apple.TotalReturn, apple.TotalReturnPercent)
	}

	if cash := performance.Positions[1]; cash.Value != "50.00" || cash.Change != "0.00" || cash.MarketPriced {
		t.Errorf("cash = %+v", cash)
	}

	if len(performance.Accounts) != 1 || performance.Accounts[0].Value != "1250.00" || performance.Accounts[0].StartValue != "1090.00" || performance.Accounts[0].Change != "160.00" {
		t.Errorf("accounts = %+v", performance.Accounts)
	}
}

func TestPerformanceWithoutAlpacaKeyUsesPlaidPrices(t *testing.T) {
	f := newPortfolioFixture()
	f.credErr = domain.ErrNotFound

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if f.barsCalled {
		t.Error("market data requested without a key")
	}

	if performance.MarketData != domain.MarketDataNotConfigured {
		t.Errorf("market data = %s, want not_configured", performance.MarketData)
	}

	if want := []string{"1180.00", "1200.00"}; !slices.Equal(pointValues(performance.Points), want) {
		t.Errorf("points = %v, want %v", pointValues(performance.Points), want)
	}
}

func TestPerformanceMarketFailureFallsBackToPlaidPrices(t *testing.T) {
	f := newPortfolioFixture()
	f.barsErr = domain.ErrUnavailable

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if performance.MarketData != domain.MarketDataUnavailable || performance.EndValue != "1200.00" {
		t.Errorf("performance = %s %s, want unavailable at Plaid prices", performance.MarketData, performance.EndValue)
	}
}

func TestPerformanceIncludesPositionsSoldDuringTheRange(t *testing.T) {
	f := newPortfolioFixture()
	soldID := uuid.New()
	f.transactions = append(f.transactions, domain.InvestmentTransaction{
		AccountID: f.accountID, AccountName: "Brokerage", SecurityID: &soldID, Ticker: ptr("MSFT"), SecurityType: ptr("equity"),
		PostedOn: time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC), Quantity: ptr("-2"), Price: ptr("400"), Amount: "800", Type: "sell", Subtype: "sell",
	})
	f.bars["MSFT"] = []marketdata.Bar{{Time: time.Date(2026, time.November, 19, 19, 55, 0, 0, time.UTC), Close: "410"}}

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{SecurityID: &soldID})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if !slices.Equal(f.gotSymbols, []string{"MSFT"}) {
		t.Errorf("symbols = %v, want MSFT", f.gotSymbols)
	}

	if len(performance.Positions) != 1 || performance.Positions[0].StartValue != "820.00" || performance.Positions[0].Value != "0.00" || performance.Positions[0].Quantity != "0" {
		t.Errorf("positions = %+v, want MSFT sold from 820.00 to 0", performance.Positions)
	}
}

func TestPerformanceAllStartsAtFirstTrade(t *testing.T) {
	f := newPortfolioFixture()
	first := time.Date(2025, time.June, 2, 0, 0, 0, 0, time.UTC)
	f.transactions = append([]domain.InvestmentTransaction{{AccountID: f.accountID, SecurityID: &f.appleID, PostedOn: first, Quantity: ptr("6"), Type: "buy", Subtype: "buy"}}, f.transactions...)

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeAll, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if !performance.Start.Equal(first) || !f.gotEnd.Equal(portfolioNow.Add(-sipDelay)) {
		t.Errorf("start = %v, want %v", performance.Start, first)
	}
}

func TestPerformanceSkipsOtherCurrencies(t *testing.T) {
	f := newPortfolioFixture()
	f.holdings = append(f.holdings, domain.InvestmentHolding{AccountID: f.accountID, SecurityID: uuid.New(), InstitutionPrice: "10", Quantity: "10", Currency: ptr("CAD")})

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if len(performance.Positions) != 2 {
		t.Errorf("positions = %+v, want CAD holding left out", performance.Positions)
	}
}

func TestPerformanceDayRangeShowsLastSessionWhenMarketIsClosed(t *testing.T) {
	f := newPortfolioFixture()
	sunday := time.Date(2026, time.November, 22, 20, 0, 0, 0, time.UTC)
	friday := time.Date(2026, time.November, 20, 20, 55, 0, 0, time.UTC)
	f.service.now = func() time.Time { return sunday }
	f.transactions = nil
	f.bars = map[string][]marketdata.Bar{"AAPL": {
		{Time: friday.Add(-24 * time.Hour), Close: "100"},
		{Time: friday.Add(-time.Hour), Close: "105"},
		{Time: friday, Close: "110"},
	}}
	f.trades = map[string]marketdata.Trade{"AAPL": {Time: friday, Price: "110"}}

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if !performance.Start.Equal(friday.Add(-24 * time.Hour)) {
		t.Errorf("start = %v, want 24h before Friday's last bar", performance.Start)
	}

	if want := []string{"1050.00", "1100.00", "1150.00", "1150.00"}; !slices.Equal(pointValues(performance.Points), want) {
		t.Errorf("points = %v, want Friday's session %v", pointValues(performance.Points), want)
	}

	if performance.Change != "100.00" {
		t.Errorf("change = %s, want 100.00", performance.Change)
	}
}

func TestPerformanceBuyingIsNotAGain(t *testing.T) {
	f := newPortfolioFixture()
	f.bars = map[string][]marketdata.Bar{"AAPL": {{Time: time.Date(2026, time.November, 19, 19, 55, 0, 0, time.UTC), Close: "110"}}}
	f.trades = map[string]marketdata.Trade{"AAPL": {Time: portfolioNow, Price: "110"}}
	f.holdings = f.holdings[:1]

	performance, err := f.service.Performance(t.Context(), f.userID, domain.PortfolioRangeDay, PortfolioFilter{})
	if err != nil {
		t.Fatalf("Performance: %v", err)
	}

	if performance.Change != "0.00" {
		t.Errorf("change = %s, want 0.00 when only shares were bought at an unchanged price", performance.Change)
	}
}
