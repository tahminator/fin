package service

import (
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/provider"
)

func toDomainAccounts(accounts []provider.Account) []domain.Account {
	mapped := make([]domain.Account, 0, len(accounts))

	for _, account := range accounts {
		mapped = append(mapped, domain.Account{
			ExternalID:       account.ExternalID,
			Name:             account.Name,
			OfficialName:     account.OfficialName,
			Mask:             account.Mask,
			Type:             account.Type,
			Subtype:          account.Subtype,
			CurrentBalance:   account.CurrentBalance,
			AvailableBalance: account.AvailableBalance,
			Currency:         account.Currency,
		})
	}

	return mapped
}

func toDomainTransactions(transactions []provider.Transaction, accountIDs map[string]uuid.UUID) []domain.Transaction {
	mapped := make([]domain.Transaction, 0, len(transactions))

	for _, transaction := range transactions {
		accountID, ok := accountIDs[transaction.AccountExternalID]
		if !ok {
			continue
		}

		mapped = append(mapped, domain.Transaction{
			AccountID:    accountID,
			ExternalID:   transaction.ExternalID,
			Amount:       transaction.Amount,
			Currency:     transaction.Currency,
			PostedOn:     transaction.PostedOn,
			AuthorizedOn: transaction.AuthorizedOn,
			Name:         transaction.Name,
			MerchantName: transaction.MerchantName,
			Category:     transaction.Category,
			Pending:      transaction.Pending,
		})
	}

	return mapped
}
