package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/marketdata"
	"github.com/tahminator/fin/internal/repository"
)

const (
	sipDelay         = 16 * time.Minute
	maxHistory       = 2
	intradayLookup   = 4 * 24 * time.Hour
	dailyLookup      = 10 * 24 * time.Hour
	cashSecurityType = "cash"
)

type PortfolioService struct {
	investments repository.InvestmentRepository
	credentials repository.CredentialRepository
	market      marketdata.Provider
	now         func() time.Time
}

func NewPortfolioService(investments repository.InvestmentRepository, credentials repository.CredentialRepository, market marketdata.Provider) *PortfolioService {
	return &PortfolioService{investments: investments, credentials: credentials, market: market, now: time.Now}
}

type rangeWindow struct {
	start     time.Time
	timeframe marketdata.Timeframe
	lookback  time.Duration
	session   bool
}

type marketPrices struct {
	status domain.MarketDataStatus
	bars   map[string][]marketdata.Bar
	latest map[string]marketdata.Trade
}

func (s *PortfolioService) Performance(ctx context.Context, userID uuid.UUID, portfolioRange domain.PortfolioRange, filter domain.PortfolioFilter) (domain.PortfolioPerformance, error) {
	now := s.now().UTC()

	holdings, err := s.investments.ListHoldingsByUser(ctx, userID, repository.InvestmentFilter{AccountID: filter.AccountID})
	if err != nil {
		return domain.PortfolioPerformance{}, fmt.Errorf("list holdings: %w", err)
	}

	transactions, err := s.investments.ListTransactionsSince(ctx, userID, now.AddDate(-maxHistory, 0, 0))
	if err != nil {
		return domain.PortfolioPerformance{}, fmt.Errorf("list investment transactions: %w", err)
	}

	window := windowFor(portfolioRange, now, transactions)
	positions := buildPositions(holdings, transactions, filter)

	prices := s.prices(ctx, userID, positions, window, now)
	start := sessionStart(window, now, prices.bars)
	end := endOf(now, prices)
	times := seriesTimes(start, end.time, prices.bars)

	for _, p := range positions {
		p.values = valuesOf(p, times, prices, end)
	}

	positions = slices.DeleteFunc(positions, func(p *position) bool {
		return !p.heldIn(dayOf(start))
	})

	return summarize(portfolioRange, start, times, positions, prices, end, filter.SecurityID != nil), nil
}

func windowFor(portfolioRange domain.PortfolioRange, now time.Time, transactions []domain.InvestmentTransaction) rangeWindow {
	switch portfolioRange {
	case domain.PortfolioRangeHour:
		return rangeWindow{start: now.Add(-time.Hour), timeframe: marketdata.TimeframeMinute, lookback: intradayLookup, session: true}
	case domain.PortfolioRangeDay:
		return rangeWindow{start: now.Add(-24 * time.Hour), timeframe: marketdata.TimeframeFiveMinutes, lookback: intradayLookup, session: true}
	case domain.PortfolioRangeThreeDays:
		return rangeWindow{start: now.Add(-72 * time.Hour), timeframe: marketdata.TimeframeFifteenMinute, lookback: intradayLookup}
	case domain.PortfolioRangeWeek:
		return rangeWindow{start: now.AddDate(0, 0, -7), timeframe: marketdata.TimeframeHour, lookback: intradayLookup}
	case domain.PortfolioRangeMonth:
		return rangeWindow{start: now.AddDate(0, -1, 0), timeframe: marketdata.TimeframeHour, lookback: intradayLookup}
	case domain.PortfolioRangeYear:
		return rangeWindow{start: now.AddDate(-1, 0, 0), timeframe: marketdata.TimeframeDay, lookback: dailyLookup}
	case domain.PortfolioRangeAll:
		start := now.AddDate(-1, 0, 0)
		if len(transactions) > 0 {
			start = transactions[0].PostedOn
		}

		return rangeWindow{start: start, timeframe: marketdata.TimeframeDay, lookback: dailyLookup}
	}

	return rangeWindow{start: now.Add(-24 * time.Hour), timeframe: marketdata.TimeframeFiveMinutes, lookback: intradayLookup, session: true}
}

func sessionStart(window rangeWindow, now time.Time, bars map[string][]marketdata.Bar) time.Time {
	if !window.session {
		return window.start
	}

	lastBar := lastBarTime(bars)

	if lastBar.IsZero() || !lastBar.Before(window.start) {
		return window.start
	}

	return lastBar.Add(-now.Sub(window.start))
}

func (s *PortfolioService) prices(ctx context.Context, userID uuid.UUID, positions []*position, window rangeWindow, now time.Time) marketPrices {
	credential, err := s.credentials.Find(ctx, userID, domain.ProviderAlpaca)
	if errors.Is(err, domain.ErrNotFound) {
		return marketPrices{status: domain.MarketDataNotConfigured}
	}

	if err != nil {
		slog.WarnContext(ctx, "find market data credential", "err", err)

		return marketPrices{status: domain.MarketDataUnavailable}
	}

	var symbols []string

	for _, p := range positions {
		if symbol := p.symbol(); symbol != "" && !slices.Contains(symbols, symbol) {
			symbols = append(symbols, symbol)
		}
	}

	slices.Sort(symbols)

	bars, err := s.market.Bars(ctx, credential, symbols, window.timeframe, window.start.Add(-window.lookback), now.Add(-sipDelay))
	if err != nil {
		slog.WarnContext(ctx, "fetch market bars", "err", err)

		return marketPrices{status: domain.MarketDataUnavailable}
	}

	latest, err := s.market.LatestTrades(ctx, credential, symbols)
	if err != nil {
		slog.WarnContext(ctx, "fetch latest trades", "err", err)

		latest = nil
	}

	return marketPrices{status: domain.MarketDataLive, bars: bars, latest: latest}
}

type seriesEnd struct {
	time time.Time
}

func endOf(now time.Time, prices marketPrices) seriesEnd {
	if prices.status != domain.MarketDataLive {
		return seriesEnd{time: now}
	}

	lastBar := lastBarTime(prices.bars)

	var lastTrade time.Time

	for _, trade := range prices.latest {
		if trade.Time.After(lastTrade) && !trade.Time.After(now) {
			lastTrade = trade.Time
		}
	}

	switch {
	case lastTrade.After(lastBar):
		return seriesEnd{time: lastTrade}
	case !lastBar.IsZero():
		return seriesEnd{time: lastBar}
	}

	return seriesEnd{time: now}
}

func valuesOf(p *position, times []time.Time, prices marketPrices, end seriesEnd) []*big.Rat {
	fallback := p.price
	if bars := prices.bars[p.symbol()]; len(bars) > 0 {
		fallback = parseDecimal(bars[0].Close)
	}

	values := make([]*big.Rat, len(times))

	for i, t := range times {
		price, ok := p.marketPriceAt(prices, t, end)
		if !ok {
			price = fallback
		}

		values[i] = new(big.Rat).Mul(p.quantityAt(t), price)
	}

	return values
}
