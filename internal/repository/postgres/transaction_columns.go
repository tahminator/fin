package postgres

import (
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type transactionColumns struct {
	accountIDs    []uuid.UUID
	externalIDs   []string
	amounts       []string
	currencies    []*string
	postedOn      []time.Time
	authorizedOn  []*time.Time
	names         []string
	merchantNames []*string
	categories    []*string
	pending       []bool
}

func newTransactionColumns(transactions []domain.Transaction) transactionColumns {
	var columns transactionColumns

	for _, transaction := range transactions {
		columns.accountIDs = append(columns.accountIDs, transaction.AccountID)
		columns.externalIDs = append(columns.externalIDs, transaction.ExternalID)
		columns.amounts = append(columns.amounts, transaction.Amount)
		columns.currencies = append(columns.currencies, transaction.Currency)
		columns.postedOn = append(columns.postedOn, transaction.PostedOn)
		columns.authorizedOn = append(columns.authorizedOn, transaction.AuthorizedOn)
		columns.names = append(columns.names, transaction.Name)
		columns.merchantNames = append(columns.merchantNames, transaction.MerchantName)
		columns.categories = append(columns.categories, transaction.Category)
		columns.pending = append(columns.pending, transaction.Pending)
	}

	return columns
}
