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

func toDomainInvestmentHoldings(result provider.InvestmentHoldingsResult) ([]domain.InvestmentSecurity, []domain.InvestmentHolding) {
	holdings := make([]domain.InvestmentHolding, 0, len(result.Holdings))
	for _, holding := range result.Holdings {
		holdings = append(holdings, domain.InvestmentHolding{
			AccountExternalID:    holding.AccountExternalID,
			SecurityExternalID:   holding.SecurityExternalID,
			InstitutionPrice:     holding.InstitutionPrice,
			InstitutionPriceAsOf: holding.InstitutionPriceAsOf,
			InstitutionValue:     holding.InstitutionValue,
			CostBasis:            holding.CostBasis,
			Quantity:             holding.Quantity,
			Currency:             holding.Currency,
			VestedQuantity:       holding.VestedQuantity,
			VestedValue:          holding.VestedValue,
		})
	}

	return toDomainInvestmentSecurities(result.Securities), holdings
}

func toDomainInvestmentTransactions(result provider.InvestmentTransactionsResult) ([]domain.InvestmentSecurity, []domain.InvestmentTransaction) {
	securities := result.Securities
	known := make(map[string]bool, len(securities))
	for _, security := range securities {
		known[security.ExternalID] = true
	}

	mapped := make([]domain.InvestmentTransaction, 0, len(result.Transactions))
	for _, transaction := range result.Transactions {
		if id := transaction.SecurityExternalID; id != nil && !known[*id] {
			known[*id] = true
			securities = append(securities, provider.InvestmentSecurity{ExternalID: *id})
		}

		mapped = append(mapped, domain.InvestmentTransaction{
			AccountExternalID:   transaction.AccountExternalID,
			SecurityExternalID:  transaction.SecurityExternalID,
			ExternalID:          transaction.ExternalID,
			CancelExternalID:    transaction.CancelExternalID,
			PostedOn:            transaction.PostedOn,
			TransactionDatetime: transaction.TransactionDateTime,
			Name:                transaction.Name,
			Quantity:            &transaction.Quantity,
			Amount:              transaction.Amount,
			Price:               &transaction.Price,
			Fees:                transaction.Fees,
			Type:                transaction.Type,
			Subtype:             transaction.Subtype,
			Currency:            transaction.Currency,
		})
	}

	return toDomainInvestmentSecurities(securities), mapped
}

func toDomainInvestmentSecurities(securities []provider.InvestmentSecurity) []domain.InvestmentSecurity {
	seen := make(map[string]bool, len(securities))
	mapped := make([]domain.InvestmentSecurity, 0, len(securities))
	for _, security := range securities {
		if seen[security.ExternalID] {
			continue
		}

		seen[security.ExternalID] = true
		mapped = append(mapped, domain.InvestmentSecurity{
			ExternalID: security.ExternalID,
			Name:       security.Name,
			Ticker:     security.Ticker,
			Type:       security.Type,
			Subtype:    security.Subtype,
			Currency:   security.Currency,
		})
	}

	return mapped
}
