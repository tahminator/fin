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

func toDomainSecurities(securities []provider.InvestmentSecurity) []domain.InvestmentSecurity {
	byExternalID := make(map[string]domain.InvestmentSecurity, len(securities))
	order := make([]string, 0, len(securities))

	for _, security := range securities {
		existing, seen := byExternalID[security.ExternalID]
		if !seen {
			order = append(order, security.ExternalID)
		}

		byExternalID[security.ExternalID] = domain.InvestmentSecurity{
			ExternalID: security.ExternalID,
			Name:       firstPresent(security.Name, existing.Name),
			Ticker:     firstPresent(security.Ticker, existing.Ticker),
			Type:       firstPresent(security.Type, existing.Type),
			Subtype:    firstPresent(security.Subtype, existing.Subtype),
			Currency:   firstPresent(security.Currency, existing.Currency),
		}
	}

	mapped := make([]domain.InvestmentSecurity, 0, len(order))

	for _, externalID := range order {
		mapped = append(mapped, byExternalID[externalID])
	}

	return mapped
}

func firstPresent(values ...*string) *string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}

	return nil
}

func toDomainHoldings(holdings []provider.InvestmentHolding, accountIDs map[string]uuid.UUID) []domain.InvestmentHolding {
	mapped := make([]domain.InvestmentHolding, 0, len(holdings))

	for _, holding := range holdings {
		accountID, ok := accountIDs[holding.AccountExternalID]
		if !ok {
			continue
		}

		mapped = append(mapped, domain.InvestmentHolding{
			AccountID:            accountID,
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

	return mapped
}

func toDomainInvestmentTransactions(transactions []provider.InvestmentTransaction, accountIDs map[string]uuid.UUID) []domain.InvestmentTransaction {
	mapped := make([]domain.InvestmentTransaction, 0, len(transactions))

	for _, transaction := range transactions {
		accountID, ok := accountIDs[transaction.AccountExternalID]
		if !ok {
			continue
		}

		mapped = append(mapped, domain.InvestmentTransaction{
			AccountID:           accountID,
			SecurityExternalID:  transaction.SecurityExternalID,
			ExternalID:          transaction.ExternalID,
			PostedOn:            transaction.PostedOn,
			TransactionDatetime: transaction.TransactionDatetime,
			Name:                transaction.Name,
			Quantity:            transaction.Quantity,
			Amount:              transaction.Amount,
			Price:               transaction.Price,
			Fees:                transaction.Fees,
			Type:                transaction.Type,
			Subtype:             transaction.Subtype,
			Currency:            transaction.Currency,
		})
	}

	return mapped
}

func investmentAccountIDs(accounts []provider.Account, accountIDs map[string]uuid.UUID) map[string]uuid.UUID {
	investmentIDs := make(map[string]uuid.UUID)

	for _, account := range accounts {
		accountID, ok := accountIDs[account.ExternalID]
		if ok && account.Type == domain.AccountTypeInvestment {
			investmentIDs[account.ExternalID] = accountID
		}
	}

	return investmentIDs
}
