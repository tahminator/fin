package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type CurrencyBalance struct {
	Currency    string `json:"currency"`
	Assets      string `json:"assets" doc:"Decimal string"`
	Liabilities string `json:"liabilities" doc:"Decimal string"`
	Net         string `json:"net" doc:"Decimal string"`
}

func newCurrencyBalance(balance domain.CurrencyBalance) CurrencyBalance {
	return CurrencyBalance{
		Currency:    balance.Currency,
		Assets:      balance.Assets,
		Liabilities: balance.Liabilities,
		Net:         balance.Net,
	}
}

type Summary struct {
	Balances        []CurrencyBalance `json:"balances" nullable:"false"`
	ConnectionCount int               `json:"connectionCount"`
	AccountCount    int               `json:"accountCount"`
}

type SummaryOutput struct {
	Body Summary
}

type SummaryHandler struct {
	summary repository.SummaryRepository
}

func NewSummaryHandler(summary repository.SummaryRepository) *SummaryHandler {
	return &SummaryHandler{summary: summary}
}

func (h *SummaryHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-summary",
		Method:      http.MethodGet,
		Path:        "/summary",
		Summary:     "Per-currency balances and counts",
		Tags:        []string{"summary"},
		Security:    cookieAuth,
	}, h.get)
}

func (h *SummaryHandler) get(ctx context.Context, _ *struct{}) (*SummaryOutput, error) {
	userID := UserFrom(ctx).ID

	balances, err := h.summary.Balances(ctx, userID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	counts, err := h.summary.Counts(ctx, userID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &SummaryOutput{Body: Summary{
		Balances:        mapItems(balances, newCurrencyBalance),
		ConnectionCount: counts.Connections,
		AccountCount:    counts.Accounts,
	}}, nil
}
