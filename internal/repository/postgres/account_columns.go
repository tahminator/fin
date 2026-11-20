package postgres

import "github.com/tahminator/fin/internal/domain"

type accountColumns struct {
	externalIDs       []string
	names             []string
	officialNames     []*string
	masks             []*string
	types             []string
	subtypes          []*string
	currentBalances   []*string
	availableBalances []*string
	currencies        []*string
}

func newAccountColumns(accounts []domain.Account) accountColumns {
	var columns accountColumns

	for _, account := range accounts {
		columns.externalIDs = append(columns.externalIDs, account.ExternalID)
		columns.names = append(columns.names, account.Name)
		columns.officialNames = append(columns.officialNames, account.OfficialName)
		columns.masks = append(columns.masks, account.Mask)
		columns.types = append(columns.types, string(account.Type))
		columns.subtypes = append(columns.subtypes, account.Subtype)
		columns.currentBalances = append(columns.currentBalances, account.CurrentBalance)
		columns.availableBalances = append(columns.availableBalances, account.AvailableBalance)
		columns.currencies = append(columns.currencies, account.Currency)
	}

	return columns
}
