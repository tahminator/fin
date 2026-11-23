package mcpserver

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type ListInvestmentHoldingsInput struct {
	AccountID         string   `json:"account_id,omitempty" jsonschema:"Only holdings in this investment account. Use an ID from list_accounts."`
	ExcludeAccountIDs []string `json:"exclude_account_ids,omitempty" jsonschema:"Leave these investment accounts out."`
}

type InvestmentHoldingOutput struct {
	ID                   string  `json:"id"`
	AccountID            string  `json:"account_id"`
	AccountName          string  `json:"account_name"`
	SecurityID           string  `json:"security_id"`
	SecurityName         *string `json:"security_name"`
	Ticker               *string `json:"ticker"`
	SecurityType         *string `json:"security_type"`
	InstitutionPrice     string  `json:"institution_price" jsonschema:"Brokerage-reported price per unit as a decimal string."`
	InstitutionPriceAsOf *string `json:"institution_price_as_of"`
	InstitutionValue     string  `json:"institution_value" jsonschema:"Brokerage-reported holding value as a decimal string."`
	CostBasis            *string `json:"cost_basis" jsonschema:"Brokerage-reported total cost basis as a decimal string."`
	Quantity             string  `json:"quantity" jsonschema:"Units held as a decimal string."`
	Currency             *string `json:"currency"`
	VestedQuantity       *string `json:"vested_quantity"`
	VestedValue          *string `json:"vested_value"`
}

type ListInvestmentHoldingsOutput struct {
	Holdings  []InvestmentHoldingOutput `json:"holdings"`
	Truncated bool                      `json:"truncated" jsonschema:"True when more than 1000 holdings matched."`
}

func (t *toolset) listInvestmentHoldings(ctx context.Context, input ListInvestmentHoldingsInput) (ListInvestmentHoldingsOutput, error) {
	filter, err := newInvestmentFilter(input.AccountID, "", input.ExcludeAccountIDs)
	if err != nil {
		return ListInvestmentHoldingsOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	holdings, err := t.deps.Investments.ListHoldingsByUser(ctx, t.userID(), filter)
	if err != nil {
		return ListInvestmentHoldingsOutput{}, err
	}

	truncated := len(holdings) > maxListedRecords
	if truncated {
		holdings = holdings[:maxListedRecords]
	}

	output := ListInvestmentHoldingsOutput{
		Holdings:  make([]InvestmentHoldingOutput, 0, len(holdings)),
		Truncated: truncated,
	}

	for _, holding := range holdings {
		output.Holdings = append(output.Holdings, newInvestmentHoldingOutput(holding))
	}

	return output, nil
}

func newInvestmentHoldingOutput(holding domain.InvestmentHolding) InvestmentHoldingOutput {
	output := InvestmentHoldingOutput{
		ID:               holding.ID.String(),
		AccountID:        holding.AccountID.String(),
		AccountName:      holding.AccountName,
		SecurityID:       holding.SecurityID.String(),
		SecurityName:     holding.SecurityName,
		Ticker:           holding.Ticker,
		SecurityType:     holding.SecurityType,
		InstitutionPrice: holding.InstitutionPrice,
		InstitutionValue: holding.InstitutionValue,
		CostBasis:        holding.CostBasis,
		Quantity:         holding.Quantity,
		Currency:         holding.Currency,
		VestedQuantity:   holding.VestedQuantity,
		VestedValue:      holding.VestedValue,
	}

	if holding.InstitutionPriceAsOf != nil {
		value := formatDate(*holding.InstitutionPriceAsOf)
		output.InstitutionPriceAsOf = &value
	}

	return output
}

type SearchInvestmentTransactionsInput struct {
	AccountID         string   `json:"account_id,omitempty" jsonschema:"Only activity in this investment account. Use an ID from list_accounts."`
	SecurityID        string   `json:"security_id,omitempty" jsonschema:"Only activity for this security. Use an ID from list_investment_holdings or get_investment_performance."`
	ExcludeAccountIDs []string `json:"exclude_account_ids,omitempty" jsonschema:"Leave these investment accounts out."`
	Page              int      `json:"page,omitempty" jsonschema:"1-based page number. Default 1."`
	PageSize          int      `json:"page_size,omitempty" jsonschema:"Results per page, up to 500. Default 100."`
}

type InvestmentTransactionOutput struct {
	ID                  string  `json:"id"`
	AccountID           string  `json:"account_id"`
	AccountName         string  `json:"account_name"`
	SecurityID          *string `json:"security_id"`
	SecurityName        *string `json:"security_name"`
	Ticker              *string `json:"ticker"`
	SecurityType        *string `json:"security_type"`
	PostedOn            string  `json:"posted_on"`
	TransactionDatetime *string `json:"transaction_datetime"`
	Name                string  `json:"name"`
	Quantity            *string `json:"quantity"`
	Amount              string  `json:"amount" jsonschema:"Decimal string; negative is cash leaving the account."`
	Price               *string `json:"price"`
	Fees                *string `json:"fees"`
	Type                string  `json:"type"`
	Subtype             string  `json:"subtype"`
	Currency            *string `json:"currency"`
}

type SearchInvestmentTransactionsOutput struct {
	Transactions []InvestmentTransactionOutput `json:"transactions"`
	Total        int                           `json:"total"`
	Page         int                           `json:"page"`
	PageSize     int                           `json:"page_size"`
	HasNext      bool                          `json:"has_next"`
}

func (t *toolset) searchInvestmentTransactions(ctx context.Context, input SearchInvestmentTransactionsInput) (SearchInvestmentTransactionsOutput, error) {
	filter, err := newInvestmentFilter(input.AccountID, input.SecurityID, input.ExcludeAccountIDs)
	if err != nil {
		return SearchInvestmentTransactionsOutput{}, err
	}

	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}

	if pageSize < 1 || pageSize > maxPageSize {
		return SearchInvestmentTransactionsOutput{}, invalidParams("page_size must be between 1 and %d", maxPageSize)
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	transactions, total, err := t.deps.Investments.ListTransactionsByUser(ctx, t.userID(), filter, pageSize, (page-1)*pageSize)
	if err != nil {
		return SearchInvestmentTransactionsOutput{}, err
	}

	output := SearchInvestmentTransactionsOutput{
		Transactions: make([]InvestmentTransactionOutput, 0, len(transactions)),
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		HasNext:      page*pageSize < total,
	}

	for _, transaction := range transactions {
		output.Transactions = append(output.Transactions, newInvestmentTransactionOutput(transaction))
	}

	return output, nil
}

func newInvestmentTransactionOutput(transaction domain.InvestmentTransaction) InvestmentTransactionOutput {
	output := InvestmentTransactionOutput{
		ID:           transaction.ID.String(),
		AccountID:    transaction.AccountID.String(),
		AccountName:  transaction.AccountName,
		SecurityName: transaction.SecurityName,
		Ticker:       transaction.Ticker,
		SecurityType: transaction.SecurityType,
		PostedOn:     formatDate(transaction.PostedOn),
		Name:         transaction.Name,
		Quantity:     transaction.Quantity,
		Amount:       transaction.Amount,
		Price:        transaction.Price,
		Fees:         transaction.Fees,
		Type:         transaction.Type,
		Subtype:      transaction.Subtype,
		Currency:     transaction.Currency,
	}

	if transaction.SecurityID != nil {
		value := transaction.SecurityID.String()
		output.SecurityID = &value
	}

	if transaction.TransactionDatetime != nil {
		value := transaction.TransactionDatetime.UTC().Format(time.RFC3339)
		output.TransactionDatetime = &value
	}

	return output
}

type GetInvestmentPerformanceInput struct {
	Range      string `json:"range,omitempty" jsonschema:"1h, 1d (default), 3d, 7d, 1mo, 1y or all."`
	AccountID  string `json:"account_id,omitempty" jsonschema:"Only this investment account. Use an ID from list_accounts."`
	SecurityID string `json:"security_id,omitempty" jsonschema:"Only this security. Use an ID from list_investment_holdings."`
}

type PortfolioPointOutput struct {
	Time  string `json:"time"`
	Value string `json:"value"`
}

type PositionPerformanceOutput struct {
	SecurityID         string  `json:"security_id"`
	Name               *string `json:"name"`
	Ticker             *string `json:"ticker"`
	SecurityType       *string `json:"security_type"`
	Quantity           string  `json:"quantity"`
	Price              string  `json:"price"`
	Value              string  `json:"value"`
	StartValue         string  `json:"start_value"`
	Change             string  `json:"change"`
	ChangePercent      *string `json:"change_percent"`
	CostBasis          *string `json:"cost_basis"`
	AverageCost        *string `json:"average_cost"`
	TotalReturn        *string `json:"total_return"`
	TotalReturnPercent *string `json:"total_return_percent"`
	MarketPriced       bool    `json:"market_priced" jsonschema:"Whether values use market prices rather than the brokerage's latest price."`
}

type AccountPerformanceOutput struct {
	AccountID     string  `json:"account_id"`
	Name          string  `json:"name"`
	Value         string  `json:"value"`
	StartValue    string  `json:"start_value"`
	Change        string  `json:"change"`
	ChangePercent *string `json:"change_percent"`
}

type GetInvestmentPerformanceOutput struct {
	Range         string                      `json:"range"`
	Currency      string                      `json:"currency"`
	Start         string                      `json:"start"`
	Points        []PortfolioPointOutput      `json:"points"`
	StartValue    string                      `json:"start_value"`
	EndValue      string                      `json:"end_value"`
	Change        string                      `json:"change"`
	ChangePercent *string                     `json:"change_percent"`
	Positions     []PositionPerformanceOutput `json:"positions"`
	Accounts      []AccountPerformanceOutput  `json:"accounts"`
	MarketData    string                      `json:"market_data" jsonschema:"live, not_configured or unavailable. Non-live values use the brokerage's latest price."`
}

func (t *toolset) getInvestmentPerformance(ctx context.Context, input GetInvestmentPerformanceInput) (GetInvestmentPerformanceOutput, error) {
	portfolioRange := domain.PortfolioRange(input.Range)
	if portfolioRange == "" {
		portfolioRange = domain.PortfolioRangeDay
	}

	if !portfolioRange.Valid() {
		return GetInvestmentPerformanceOutput{}, invalidParams("range must be 1h, 1d, 3d, 7d, 1mo, 1y or all")
	}

	filter, err := newInvestmentFilter(input.AccountID, input.SecurityID, nil)
	if err != nil {
		return GetInvestmentPerformanceOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	performance, err := t.deps.Portfolio.Performance(ctx, t.userID(), portfolioRange, domain.PortfolioFilter{
		AccountID:  filter.AccountID,
		SecurityID: filter.SecurityID,
	})
	if err != nil {
		return GetInvestmentPerformanceOutput{}, err
	}

	return newInvestmentPerformanceOutput(performance), nil
}

func newInvestmentPerformanceOutput(performance domain.PortfolioPerformance) GetInvestmentPerformanceOutput {
	output := GetInvestmentPerformanceOutput{
		Range:         string(performance.Range),
		Currency:      performance.Currency,
		Start:         performance.Start.UTC().Format(time.RFC3339),
		Points:        make([]PortfolioPointOutput, 0, len(performance.Points)),
		StartValue:    performance.StartValue,
		EndValue:      performance.EndValue,
		Change:        performance.Change,
		ChangePercent: performance.ChangePercent,
		Positions:     make([]PositionPerformanceOutput, 0, len(performance.Positions)),
		Accounts:      make([]AccountPerformanceOutput, 0, len(performance.Accounts)),
		MarketData:    string(performance.MarketData),
	}

	for _, point := range performance.Points {
		output.Points = append(output.Points, PortfolioPointOutput{
			Time:  point.Time.UTC().Format(time.RFC3339),
			Value: point.Value,
		})
	}

	for _, position := range performance.Positions {
		output.Positions = append(output.Positions, PositionPerformanceOutput{
			SecurityID:         position.SecurityID.String(),
			Name:               position.Name,
			Ticker:             position.Ticker,
			SecurityType:       position.SecurityType,
			Quantity:           position.Quantity,
			Price:              position.Price,
			Value:              position.Value,
			StartValue:         position.StartValue,
			Change:             position.Change,
			ChangePercent:      position.ChangePercent,
			CostBasis:          position.CostBasis,
			AverageCost:        position.AverageCost,
			TotalReturn:        position.TotalReturn,
			TotalReturnPercent: position.TotalReturnPercent,
			MarketPriced:       position.MarketPriced,
		})
	}

	for _, account := range performance.Accounts {
		output.Accounts = append(output.Accounts, AccountPerformanceOutput{
			AccountID:     account.AccountID.String(),
			Name:          account.Name,
			Value:         account.Value,
			StartValue:    account.StartValue,
			Change:        account.Change,
			ChangePercent: account.ChangePercent,
		})
	}

	return output
}

func newInvestmentFilter(accountID, securityID string, excludeAccountIDs []string) (repository.InvestmentFilter, error) {
	account, err := parseOptionalID("account_id", accountID)
	if err != nil {
		return repository.InvestmentFilter{}, err
	}

	security, err := parseOptionalID("security_id", securityID)
	if err != nil {
		return repository.InvestmentFilter{}, err
	}

	excluded, err := parseIDs("exclude_account_ids", excludeAccountIDs)
	if err != nil {
		return repository.InvestmentFilter{}, err
	}

	return repository.InvestmentFilter{
		AccountID:         account,
		SecurityID:        security,
		ExcludeAccountIDs: excluded,
	}, nil
}

func parseOptionalID(field, raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, invalidParams("%s must be an ID returned by fin", field)
	}

	return &id, nil
}
