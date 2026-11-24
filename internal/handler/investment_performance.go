package handler

import (
	"context"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
)

type PortfolioPoint struct {
	Time  string `json:"time" format:"date-time"`
	Value string `json:"value" doc:"Decimal string"`
}

type PositionPerformance struct {
	SecurityID               uuid.UUID        `json:"securityId"`
	Name                     *string          `json:"name"`
	Ticker                   *string          `json:"ticker"`
	SecurityType             *string          `json:"securityType"`
	Quantity                 string           `json:"quantity" doc:"Decimal string; shares held now"`
	Price                    string           `json:"price" doc:"Decimal string; latest price per share"`
	Value                    string           `json:"value" doc:"Decimal string; value now"`
	StartValue               string           `json:"startValue" doc:"Decimal string; value at the start of the range"`
	Change                   string           `json:"change" doc:"Decimal string; value now minus value at the start of the range"`
	ChangePercent            *string          `json:"changePercent" doc:"Decimal string percentage; null when the start value is zero"`
	MarketPriceChange        *string          `json:"marketPriceChange" doc:"Decimal string change per share over the range, when market prices are available"`
	MarketPriceChangePercent *string          `json:"marketPriceChangePercent" doc:"Decimal string percentage change in the security price over the range"`
	MarketPricePoints        []PortfolioPoint `json:"marketPricePoints" nullable:"false" doc:"Per-share market prices over the selected range when a security filter is set"`
	CostBasis                *string          `json:"costBasis" doc:"Decimal string; total cost basis when the brokerage reports it"`
	AverageCost              *string          `json:"averageCost" doc:"Decimal string; cost basis per share"`
	TotalReturn              *string          `json:"totalReturn" doc:"Decimal string; value now minus cost basis"`
	TotalReturnPercent       *string          `json:"totalReturnPercent" doc:"Decimal string percentage of cost basis"`
	MarketPriced             bool             `json:"marketPriced" doc:"Whether the value uses market prices rather than the brokerage's price"`
}

type AccountPerformance struct {
	AccountID     uuid.UUID `json:"accountId"`
	Name          string    `json:"name"`
	Value         string    `json:"value" doc:"Decimal string"`
	StartValue    string    `json:"startValue" doc:"Decimal string"`
	Change        string    `json:"change" doc:"Decimal string"`
	ChangePercent *string   `json:"changePercent" doc:"Decimal string percentage; null when the start value is zero"`
}

type PortfolioPerformance struct {
	Range         domain.PortfolioRange   `json:"range" enum:"1h,1d,3d,7d,1mo,1y,all"`
	Currency      string                  `json:"currency"`
	Start         string                  `json:"start" format:"date-time"`
	Points        []PortfolioPoint        `json:"points" nullable:"false"`
	StartValue    string                  `json:"startValue" doc:"Decimal string"`
	EndValue      string                  `json:"endValue" doc:"Decimal string"`
	Change        string                  `json:"change" doc:"Decimal string"`
	ChangePercent *string                 `json:"changePercent" doc:"Decimal string percentage; null when the start value is zero"`
	Positions     []PositionPerformance   `json:"positions" nullable:"false"`
	Accounts      []AccountPerformance    `json:"accounts" nullable:"false"`
	MarketData    domain.MarketDataStatus `json:"marketData" enum:"live,not_configured,unavailable" doc:"live uses Alpaca prices; otherwise values use the brokerage's latest price"`
}

type PortfolioPerformanceInput struct {
	Range      domain.PortfolioRange `query:"range" enum:"1h,1d,3d,7d,1mo,1y,all" default:"1d" doc:"Time range ending now"`
	AccountID  string                `query:"accountId" format:"uuid" doc:"Only this investment account"`
	SecurityID string                `query:"securityId" format:"uuid" doc:"Only this security"`
}

type PortfolioPerformanceOutput struct {
	Body PortfolioPerformance
}

func optionalUUID(name, raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(name + " must be a UUID")
	}

	return &id, nil
}

func (h *InvestmentHandler) performance(ctx context.Context, input *PortfolioPerformanceInput) (*PortfolioPerformanceOutput, error) {
	accountID, err := optionalUUID("accountId", input.AccountID)
	if err != nil {
		return nil, err
	}

	securityID, err := optionalUUID("securityId", input.SecurityID)
	if err != nil {
		return nil, err
	}

	performance, err := h.portfolio.Performance(ctx, UserFrom(ctx).ID, input.Range, domain.PortfolioFilter{AccountID: accountID, SecurityID: securityID})
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PortfolioPerformanceOutput{Body: newPortfolioPerformance(performance)}, nil
}

func newPortfolioPerformance(performance domain.PortfolioPerformance) PortfolioPerformance {
	return PortfolioPerformance{
		Range:         performance.Range,
		Currency:      performance.Currency,
		Start:         performance.Start.UTC().Format(time.RFC3339),
		Points:        mapItems(performance.Points, newPortfolioPoint),
		StartValue:    performance.StartValue,
		EndValue:      performance.EndValue,
		Change:        performance.Change,
		ChangePercent: performance.ChangePercent,
		Positions:     mapItems(performance.Positions, newPositionPerformance),
		Accounts:      mapItems(performance.Accounts, newAccountPerformance),
		MarketData:    performance.MarketData,
	}
}

func newPortfolioPoint(point domain.PortfolioPoint) PortfolioPoint {
	return PortfolioPoint{Time: point.Time.UTC().Format(time.RFC3339), Value: point.Value}
}

func newPositionPerformance(position domain.PositionPerformance) PositionPerformance {
	return PositionPerformance{
		SecurityID:               position.SecurityID,
		Name:                     position.Name,
		Ticker:                   position.Ticker,
		SecurityType:             position.SecurityType,
		Quantity:                 position.Quantity,
		Price:                    position.Price,
		Value:                    position.Value,
		StartValue:               position.StartValue,
		Change:                   position.Change,
		ChangePercent:            position.ChangePercent,
		MarketPriceChange:        position.MarketPriceChange,
		MarketPriceChangePercent: position.MarketPriceChangePercent,
		MarketPricePoints:        mapItems(position.MarketPricePoints, newPortfolioPoint),
		CostBasis:                position.CostBasis,
		AverageCost:              position.AverageCost,
		TotalReturn:              position.TotalReturn,
		TotalReturnPercent:       position.TotalReturnPercent,
		MarketPriced:             position.MarketPriced,
	}
}

func newAccountPerformance(account domain.AccountPerformance) AccountPerformance {
	return AccountPerformance{
		AccountID:     account.AccountID,
		Name:          account.Name,
		Value:         account.Value,
		StartValue:    account.StartValue,
		Change:        account.Change,
		ChangePercent: account.ChangePercent,
	}
}
