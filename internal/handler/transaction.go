package handler

import (
	"context"
	"net/http"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const dateLayout = time.DateOnly

type Transaction struct {
	ID           uuid.UUID `json:"id"`
	AccountID    uuid.UUID `json:"accountId"`
	AccountName  string    `json:"accountName"`
	Amount       string    `json:"amount" doc:"Decimal string; negative is an outflow"`
	Currency     *string   `json:"currency"`
	PostedOn     string    `json:"postedOn" format:"date"`
	AuthorizedOn *string   `json:"authorizedOn" format:"date"`
	Name         string    `json:"name"`
	MerchantName *string   `json:"merchantName"`
	Category     *string   `json:"category"`
	Pending      bool      `json:"pending"`
}

func newTransaction(transaction domain.Transaction) Transaction {
	mapped := Transaction{
		ID:           transaction.ID,
		AccountID:    transaction.AccountID,
		AccountName:  transaction.AccountName,
		Amount:       transaction.Amount,
		Currency:     transaction.Currency,
		PostedOn:     transaction.PostedOn.Format(dateLayout),
		Name:         transaction.Name,
		MerchantName: transaction.MerchantName,
		Category:     transaction.Category,
		Pending:      transaction.Pending,
	}

	if transaction.AuthorizedOn != nil {
		authorizedOn := transaction.AuthorizedOn.Format(dateLayout)
		mapped.AuthorizedOn = &authorizedOn
	}

	return mapped
}

type ListTransactionsInput struct {
	PageParams
	AccountID        string   `query:"accountId" format:"uuid" doc:"Only transactions of this account"`
	ExcludeAccountID []string `query:"excludeAccountId,explode" format:"uuid" doc:"Exclude transactions of these accounts"`
}

type TransactionHandler struct {
	transactions repository.TransactionRepository
}

func NewTransactionHandler(transactions repository.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

func (h *TransactionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-transactions",
		Method:      http.MethodGet,
		Path:        "/transactions",
		Summary:     "List transactions, newest first",
		Tags:        []string{"transactions"},
		Security:    cookieAuth,
	}, h.list)
}

func (h *TransactionHandler) list(ctx context.Context, input *ListTransactionsInput) (*PageOutput[Transaction], error) {
	var filter repository.TransactionFilter

	if input.AccountID != "" {
		accountID, err := uuid.Parse(input.AccountID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("accountId must be a UUID")
		}

		filter.AccountID = &accountID
	}

	if len(input.ExcludeAccountID) > 0 {
		excludeAccountIDs := make([]uuid.UUID, len(input.ExcludeAccountID))

		for i, raw := range input.ExcludeAccountID {
			excludeAccountID, err := uuid.Parse(raw)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("excludeAccountId must be a list of UUIDs")
			}

			excludeAccountIDs[i] = excludeAccountID
		}

		filter.ExcludeAccountID = excludeAccountIDs
	}

	transactions, total, err := h.transactions.ListByUser(ctx, UserFrom(ctx).ID, filter, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[Transaction]{Body: NewPage(mapItems(transactions, newTransaction), input.PageParams, total)}, nil
}
