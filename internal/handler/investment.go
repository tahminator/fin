package handler

import (
	"context"
	"net/http"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

type InvestmentHolding struct {
	ID                   uuid.UUID `json:"id"`
	AccountID            uuid.UUID `json:"accountId"`
	AccountName          string    `json:"accountName"`
	SecurityName         *string   `json:"securityName"`
	Ticker               *string   `json:"ticker"`
	InstitutionPrice     string    `json:"institutionPrice"`
	InstitutionPriceAsOf *string   `json:"institutionPriceAsOf" format:"date"`
	InstitutionValue     string    `json:"institutionValue"`
	CostBasis            *string   `json:"costBasis"`
	Quantity             string    `json:"quantity"`
	Currency             *string   `json:"currency"`
	VestedQuantity       *string   `json:"vestedQuantity"`
	VestedValue          *string   `json:"vestedValue"`
}

type InvestmentTransaction struct {
	ID                  uuid.UUID `json:"id"`
	AccountID           uuid.UUID `json:"accountId"`
	AccountName         string    `json:"accountName"`
	SecurityName        *string   `json:"securityName"`
	Ticker              *string   `json:"ticker"`
	PostedOn            string    `json:"postedOn" format:"date"`
	TransactionDatetime *string   `json:"transactionDatetime" format:"date-time"`
	Name                string    `json:"name"`
	Quantity            *string   `json:"quantity"`
	Amount              string    `json:"amount" doc:"Decimal string; negative is cash leaving the account"`
	Price               *string   `json:"price"`
	Fees                *string   `json:"fees"`
	Type                string    `json:"type"`
	Subtype             string    `json:"subtype"`
	Currency            *string   `json:"currency"`
}

type ListInvestmentHoldingsInput struct {
	AccountID        string   `query:"accountId" format:"uuid" doc:"Only holdings of this account"`
	ExcludeAccountID []string `query:"excludeAccountId,explode" format:"uuid" doc:"Exclude holdings of these accounts"`
}

type ListInvestmentTransactionsInput struct {
	PageParams
	AccountID        string   `query:"accountId" format:"uuid" doc:"Only transactions of this account"`
	SecurityID       string   `query:"securityId" format:"uuid" doc:"Only transactions of this security"`
	ExcludeAccountID []string `query:"excludeAccountId,explode" format:"uuid" doc:"Exclude transactions of these accounts"`
}

type InvestmentHandler struct {
	investments repository.InvestmentRepository
	portfolio   *service.PortfolioService
}

type InvestmentHoldingsOutput struct {
	Body []InvestmentHolding
}

func NewInvestmentHandler(investments repository.InvestmentRepository, portfolio *service.PortfolioService) *InvestmentHandler {
	return &InvestmentHandler{investments: investments, portfolio: portfolio}
}

func (h *InvestmentHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-investment-holdings",
		Method:      http.MethodGet,
		Path:        "/investments/holdings",
		Summary:     "List investment holdings",
		Tags:        []string{"investments"},
		Security:    cookieAuth,
	}, h.listHoldings)

	huma.Register(api, huma.Operation{
		OperationID: "list-investment-transactions",
		Method:      http.MethodGet,
		Path:        "/investments/transactions",
		Summary:     "List investment transactions",
		Tags:        []string{"investments"},
		Security:    cookieAuth,
	}, h.listTransactions)

	huma.Register(api, huma.Operation{
		OperationID: "get-investment-performance",
		Method:      http.MethodGet,
		Path:        "/investments/performance",
		Summary:     "Portfolio value over a time range, by security and by account",
		Tags:        []string{"investments"},
		Security:    cookieAuth,
	}, h.performance)
}

func (h *InvestmentHandler) listHoldings(ctx context.Context, input *ListInvestmentHoldingsInput) (*InvestmentHoldingsOutput, error) {
	filter, err := investmentFilter(input.AccountID, input.ExcludeAccountID)
	if err != nil {
		return nil, err
	}

	holdings, err := h.investments.ListHoldingsByUser(ctx, UserFrom(ctx).ID, filter)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &InvestmentHoldingsOutput{Body: mapInvestmentHoldings(holdings)}, nil
}

func (h *InvestmentHandler) listTransactions(ctx context.Context, input *ListInvestmentTransactionsInput) (*PageOutput[InvestmentTransaction], error) {
	filter, err := investmentFilter(input.AccountID, input.ExcludeAccountID)
	if err != nil {
		return nil, err
	}

	filter.SecurityID, err = optionalUUID("securityId", input.SecurityID)
	if err != nil {
		return nil, err
	}

	transactions, total, err := h.investments.ListTransactionsByUser(ctx, UserFrom(ctx).ID, filter, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[InvestmentTransaction]{Body: NewPage(mapInvestmentTransactions(transactions), input.PageParams, total)}, nil
}

func investmentFilter(accountID string, excludeAccountIDs []string) (repository.InvestmentFilter, error) {
	var filter repository.InvestmentFilter

	if accountID != "" {
		id, err := uuid.Parse(accountID)
		if err != nil {
			return repository.InvestmentFilter{}, huma.Error422UnprocessableEntity("accountId must be a UUID")
		}

		filter.AccountID = &id
	}

	filter.ExcludeAccountIDs = make([]uuid.UUID, len(excludeAccountIDs))
	for i, raw := range excludeAccountIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return repository.InvestmentFilter{}, huma.Error422UnprocessableEntity("excludeAccountId must be a list of UUIDs")
		}

		filter.ExcludeAccountIDs[i] = id
	}

	return filter, nil
}

func mapInvestmentHoldings(holdings []domain.InvestmentHolding) []InvestmentHolding {
	mapped := make([]InvestmentHolding, 0, len(holdings))
	for _, holding := range holdings {
		item := InvestmentHolding{
			ID:               holding.ID,
			AccountID:        holding.AccountID,
			AccountName:      holding.AccountName,
			SecurityName:     holding.SecurityName,
			Ticker:           holding.Ticker,
			InstitutionPrice: holding.InstitutionPrice,
			InstitutionValue: holding.InstitutionValue,
			CostBasis:        holding.CostBasis,
			Quantity:         holding.Quantity,
			Currency:         holding.Currency,
			VestedQuantity:   holding.VestedQuantity,
			VestedValue:      holding.VestedValue,
		}
		if holding.InstitutionPriceAsOf != nil {
			value := holding.InstitutionPriceAsOf.Format(time.DateOnly)
			item.InstitutionPriceAsOf = &value
		}
		mapped = append(mapped, item)
	}
	return mapped
}

func mapInvestmentTransactions(transactions []domain.InvestmentTransaction) []InvestmentTransaction {
	mapped := make([]InvestmentTransaction, 0, len(transactions))
	for _, transaction := range transactions {
		item := InvestmentTransaction{
			ID:           transaction.ID,
			AccountID:    transaction.AccountID,
			AccountName:  transaction.AccountName,
			SecurityName: transaction.SecurityName,
			Ticker:       transaction.Ticker,
			PostedOn:     transaction.PostedOn.Format(time.DateOnly),
			Name:         transaction.Name,
			Quantity:     transaction.Quantity,
			Amount:       transaction.Amount,
			Price:        transaction.Price,
			Fees:         transaction.Fees,
			Type:         transaction.Type,
			Subtype:      transaction.Subtype,
			Currency:     transaction.Currency,
		}
		if transaction.TransactionDatetime != nil {
			value := transaction.TransactionDatetime.Format(time.RFC3339)
			item.TransactionDatetime = &value
		}
		mapped = append(mapped, item)
	}
	return mapped
}
