package handler

import (
	"context"
	"net/http"
	"uuid"

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

type SummaryInput struct {
	ExcludeAccountID   []string             `query:"excludeAccountId,explode" format:"uuid" doc:"Account IDs to exclude from balances"`
	ExcludeAccountType []domain.AccountType `query:"excludeAccountType,explode" enum:"depository,credit,loan,investment,other" doc:"Account types to exclude from balances"`
}

type SummaryOutput struct {
	Body Summary
}

type CashFlowDay struct {
	Date string `json:"date" format:"date"`
	In   string `json:"in" doc:"Decimal string; money received"`
	Out  string `json:"out" doc:"Decimal string; money spent, as a positive amount"`
}

type CurrencyCashFlow struct {
	Currency string        `json:"currency"`
	Days     []CashFlowDay `json:"days" nullable:"false"`
}

type CashFlow struct {
	Currencies []CurrencyCashFlow `json:"currencies" nullable:"false"`
}

func newCashFlow(flows []domain.DailyCashFlow) CashFlow {
	currencies := []CurrencyCashFlow{}

	for _, flow := range flows {
		if len(currencies) == 0 || currencies[len(currencies)-1].Currency != flow.Currency {
			currencies = append(currencies, CurrencyCashFlow{Currency: flow.Currency, Days: []CashFlowDay{}})
		}

		current := &currencies[len(currencies)-1]
		current.Days = append(current.Days, CashFlowDay{Date: flow.Day.Format(dateLayout), In: flow.In, Out: flow.Out})
	}

	return CashFlow{Currencies: currencies}
}

type CashFlowInput struct {
	Days               int                  `query:"days" minimum:"1" maximum:"365" default:"30" doc:"Number of days, ending today"`
	ExcludeAccountID   []string             `query:"excludeAccountId,explode" format:"uuid" doc:"Account IDs to exclude from cash flow"`
	ExcludeAccountType []domain.AccountType `query:"excludeAccountType,explode" enum:"depository,credit,loan,investment,other" doc:"Account types to exclude from cash flow"`
}

func parseExcludeAccountIDs(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(raw))

	for _, value := range raw {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("excludeAccountId must be a UUID")
		}

		ids = append(ids, id)
	}

	return ids, nil
}

type CashFlowOutput struct {
	Body CashFlow
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

	huma.Register(api, huma.Operation{
		OperationID: "get-cash-flow",
		Method:      http.MethodGet,
		Path:        "/summary/cash-flow",
		Summary:     "Per-currency daily money in and out, excluding transfers and loan payments",
		Tags:        []string{"summary"},
		Security:    cookieAuth,
	}, h.cashFlow)
}

func (h *SummaryHandler) get(ctx context.Context, input *SummaryInput) (*SummaryOutput, error) {
	userID := UserFrom(ctx).ID

	excludeAccountIDs, err := parseExcludeAccountIDs(input.ExcludeAccountID)
	if err != nil {
		return nil, err
	}

	balances, err := h.summary.Balances(ctx, userID, excludeAccountIDs, input.ExcludeAccountType)
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

func (h *SummaryHandler) cashFlow(ctx context.Context, input *CashFlowInput) (*CashFlowOutput, error) {
	excludeAccountIDs, err := parseExcludeAccountIDs(input.ExcludeAccountID)
	if err != nil {
		return nil, err
	}

	flows, err := h.summary.CashFlow(ctx, UserFrom(ctx).ID, input.Days, excludeAccountIDs, input.ExcludeAccountType)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &CashFlowOutput{Body: newCashFlow(flows)}, nil
}
