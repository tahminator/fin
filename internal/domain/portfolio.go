package domain

import (
	"time"
	"uuid"
)

type PortfolioRange string

const (
	PortfolioRangeHour      PortfolioRange = "1h"
	PortfolioRangeDay       PortfolioRange = "1d"
	PortfolioRangeThreeDays PortfolioRange = "3d"
	PortfolioRangeWeek      PortfolioRange = "7d"
	PortfolioRangeMonth     PortfolioRange = "1mo"
	PortfolioRangeYear      PortfolioRange = "1y"
	PortfolioRangeAll       PortfolioRange = "all"
)

const PortfolioCurrency = "USD"

type PortfolioFilter struct {
	AccountID  *uuid.UUID
	SecurityID *uuid.UUID
}

func (r PortfolioRange) Valid() bool {
	switch r {
	case PortfolioRangeHour, PortfolioRangeDay, PortfolioRangeThreeDays, PortfolioRangeWeek, PortfolioRangeMonth, PortfolioRangeYear, PortfolioRangeAll:
		return true
	}

	return false
}

type MarketDataStatus string

const (
	MarketDataLive          MarketDataStatus = "live"
	MarketDataNotConfigured MarketDataStatus = "not_configured"
	MarketDataUnavailable   MarketDataStatus = "unavailable"
)

type PortfolioPoint struct {
	Time  time.Time
	Value string
}

type PositionPerformance struct {
	SecurityID         uuid.UUID
	Name               *string
	Ticker             *string
	SecurityType       *string
	Quantity           string
	Price              string
	Value              string
	StartValue         string
	Change             string
	ChangePercent      *string
	CostBasis          *string
	AverageCost        *string
	TotalReturn        *string
	TotalReturnPercent *string
	MarketPriced       bool
}

type AccountPerformance struct {
	AccountID     uuid.UUID
	Name          string
	Value         string
	StartValue    string
	Change        string
	ChangePercent *string
}

type PortfolioPerformance struct {
	Range         PortfolioRange
	Currency      string
	Start         time.Time
	Points        []PortfolioPoint
	StartValue    string
	EndValue      string
	Change        string
	ChangePercent *string
	Positions     []PositionPerformance
	Accounts      []AccountPerformance
	MarketData    MarketDataStatus
}
