package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

type InvestmentHandler struct {
	investments repository.InvestmentRepository
	history     *service.InvestmentHistoryService
}

func NewInvestmentHandler(investments repository.InvestmentRepository, history *service.InvestmentHistoryService) *InvestmentHandler {
	return &InvestmentHandler{investments: investments, history: history}
}

func (h *InvestmentHandler) Register(api huma.API) {
	tags := []string{"investments"}

	huma.Register(api, huma.Operation{
		OperationID: "list-investment-holdings",
		Method:      http.MethodGet,
		Path:        "/investments/holdings",
		Summary:     "List current investment holdings",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.listHoldings)

	huma.Register(api, huma.Operation{
		OperationID: "list-investment-transactions",
		Method:      http.MethodGet,
		Path:        "/investments/transactions",
		Summary:     "List investment transactions, newest first",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.listTransactions)

	huma.Register(api, huma.Operation{
		OperationID: "list-investment-securities",
		Method:      http.MethodGet,
		Path:        "/investments/securities",
		Summary:     "List securities that have value history",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.listSecurities)

	huma.Register(api, huma.Operation{
		OperationID: "get-investment-history",
		Method:      http.MethodGet,
		Path:        "/investments/history",
		Summary:     "Portfolio value history from synced snapshots, per currency",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.getHistory)
}

func (h *InvestmentHandler) listHoldings(ctx context.Context, input *ListInvestmentHoldingsInput) (*InvestmentHoldingsOutput, error) {
	filter, err := investmentFilter(input.AccountFilterParams)
	if err != nil {
		return nil, err
	}

	holdings, err := h.investments.ListHoldings(ctx, UserFrom(ctx).ID, filter)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &InvestmentHoldingsOutput{Body: mapItems(holdings, newInvestmentHolding)}, nil
}

func (h *InvestmentHandler) listTransactions(ctx context.Context, input *ListInvestmentTransactionsInput) (*PageOutput[InvestmentTransaction], error) {
	filter, err := investmentFilter(input.AccountFilterParams)
	if err != nil {
		return nil, err
	}

	transactions, total, err := h.investments.ListTransactions(ctx, UserFrom(ctx).ID, filter, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[InvestmentTransaction]{Body: NewPage(mapItems(transactions, newInvestmentTransaction), input.PageParams, total)}, nil
}

func (h *InvestmentHandler) listSecurities(ctx context.Context, input *ListInvestmentSecuritiesInput) (*InvestmentSecuritiesOutput, error) {
	filter, err := investmentFilter(input.AccountFilterParams)
	if err != nil {
		return nil, err
	}

	securities, err := h.investments.ListSecurities(ctx, UserFrom(ctx).ID, filter)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &InvestmentSecuritiesOutput{Body: mapItems(securities, newInvestmentSecurity)}, nil
}

func (h *InvestmentHandler) getHistory(ctx context.Context, input *InvestmentHistoryInput) (*InvestmentHistoryOutput, error) {
	filter, err := investmentFilter(input.AccountFilterParams)
	if err != nil {
		return nil, err
	}

	securityIDs, err := parseUUIDs(input.SecurityID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("securityId must be a list of UUIDs")
	}

	history, err := h.history.History(ctx, UserFrom(ctx).ID, service.InvestmentHistoryRequest{
		Range:       input.Range,
		Filter:      filter,
		SecurityIDs: securityIDs,
	})
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &InvestmentHistoryOutput{Body: newInvestmentHistory(history)}, nil
}

func investmentFilter(params AccountFilterParams) (repository.InvestmentFilter, error) {
	parsed, err := params.parse()
	if err != nil {
		return repository.InvestmentFilter{}, err
	}

	return repository.InvestmentFilter{AccountID: parsed.accountID, ExcludeAccountID: parsed.excludeAccountID}, nil
}
