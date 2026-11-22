package mcpserver

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/repository"
)

const (
	defaultPageSize  = 100
	maxPageSize      = 500
	defaultFacetSize = 50
	maxFacetSize     = 500
)

type SearchTransactionsInput struct {
	TransactionFilter
	Sort     string `json:"sort,omitempty" jsonschema:"date_desc (default), date_asc, amount_asc (largest outflows first) or amount_desc (largest inflows first)."`
	Page     int    `json:"page,omitempty" jsonschema:"1-based page number. Default 1."`
	PageSize int    `json:"page_size,omitempty" jsonschema:"Results per page, up to 500. Default 100."`
}

type TransactionOutput struct {
	ID           string  `json:"id"`
	PostedOn     string  `json:"posted_on"`
	AuthorizedOn *string `json:"authorized_on,omitempty"`
	Amount       string  `json:"amount" jsonschema:"Decimal string; negative is money out"`
	Currency     *string `json:"currency"`
	Name         string  `json:"name"`
	Merchant     *string `json:"merchant"`
	Category     *string `json:"category"`
	Pending      bool    `json:"pending"`
	AccountID    string  `json:"account_id"`
	AccountName  string  `json:"account_name"`
	AccountType  string  `json:"account_type"`
	Institution  string  `json:"institution"`
}

type SearchTransactionsOutput struct {
	Transactions []TransactionOutput `json:"transactions"`
	Total        int                 `json:"total"`
	Page         int                 `json:"page"`
	PageSize     int                 `json:"page_size"`
	HasNext      bool                `json:"has_next"`
}

var sorts = map[string]repository.TransactionSort{
	"":            repository.SortDateDesc,
	"date_desc":   repository.SortDateDesc,
	"date_asc":    repository.SortDateAsc,
	"amount_asc":  repository.SortAmountAsc,
	"amount_desc": repository.SortAmountDesc,
}

func (t *toolset) searchTransactions(ctx context.Context, input SearchTransactionsInput) (SearchTransactionsOutput, error) {
	query, err := input.query(t.deps.Now(), false)
	if err != nil {
		return SearchTransactionsOutput{}, err
	}

	sort, ok := sorts[input.Sort]
	if !ok {
		return SearchTransactionsOutput{}, invalidParams("sort must be date_desc, date_asc, amount_asc or amount_desc")
	}

	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}

	if pageSize < 1 || pageSize > maxPageSize {
		return SearchTransactionsOutput{}, invalidParams("page_size must be between 1 and %d", maxPageSize)
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	transactions, total, err := t.deps.Transactions.Search(ctx, t.userID(), query, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		return SearchTransactionsOutput{}, err
	}

	output := SearchTransactionsOutput{
		Transactions: make([]TransactionOutput, 0, len(transactions)),
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		HasNext:      page*pageSize < total,
	}

	for _, transaction := range transactions {
		output.Transactions = append(output.Transactions, newTransactionOutput(transaction))
	}

	return output, nil
}

type GetTransactionInput struct {
	TransactionID string `json:"transaction_id" jsonschema:"ID returned by search_transactions"`
}

func (t *toolset) getTransaction(ctx context.Context, input GetTransactionInput) (TransactionOutput, error) {
	id, err := uuid.Parse(input.TransactionID)
	if err != nil {
		return TransactionOutput{}, invalidParams("transaction_id must be an ID returned by search_transactions")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	transaction, err := t.deps.Transactions.Get(ctx, t.userID(), id)
	if err != nil {
		return TransactionOutput{}, err
	}

	return newTransactionOutput(transaction), nil
}

func newTransactionOutput(transaction repository.TransactionDetail) TransactionOutput {
	output := TransactionOutput{
		ID:          transaction.ID.String(),
		PostedOn:    formatDate(transaction.PostedOn),
		Amount:      transaction.Amount,
		Currency:    transaction.Currency,
		Name:        transaction.Name,
		Merchant:    transaction.MerchantName,
		Category:    transaction.Category,
		Pending:     transaction.Pending,
		AccountID:   transaction.AccountID.String(),
		AccountName: transaction.AccountName,
		AccountType: transaction.AccountType,
		Institution: institutionName(transaction.InstitutionName),
	}

	if transaction.AuthorizedOn != nil {
		authorized := formatDate(*transaction.AuthorizedOn)
		output.AuthorizedOn = &authorized
	}

	return output
}

type ListFacetsInput struct {
	TransactionFilter
	Facet string `json:"facet" jsonschema:"Which values to list: category, merchant, account or currency."`
	Limit int    `json:"limit,omitempty" jsonschema:"Most common values to return, up to 500. Default 50."`
}

type FacetOutput struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
	Count    int    `json:"count"`
	Sum      string `json:"sum" jsonschema:"Net amount of the matching transactions in this currency"`
}

type ListFacetsOutput struct {
	Facet  string        `json:"facet"`
	Values []FacetOutput `json:"values"`
}

var facets = map[string]repository.Facet{
	"category": repository.FacetCategory,
	"merchant": repository.FacetMerchant,
	"account":  repository.FacetAccount,
	"currency": repository.FacetCurrency,
}

func (t *toolset) listFacets(ctx context.Context, input ListFacetsInput) (ListFacetsOutput, error) {
	query, err := input.query(t.deps.Now(), false)
	if err != nil {
		return ListFacetsOutput{}, err
	}

	facet, ok := facets[input.Facet]
	if !ok {
		return ListFacetsOutput{}, invalidParams("facet must be category, merchant, account or currency")
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultFacetSize
	}

	if limit < 1 || limit > maxFacetSize {
		return ListFacetsOutput{}, invalidParams("limit must be between 1 and %d", maxFacetSize)
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := t.deps.Transactions.Facets(ctx, t.userID(), query, facet, limit)
	if err != nil {
		return ListFacetsOutput{}, err
	}

	output := ListFacetsOutput{Facet: input.Facet, Values: make([]FacetOutput, 0, len(rows))}

	for _, row := range rows {
		output.Values = append(output.Values, FacetOutput(row))
	}

	return output, nil
}
