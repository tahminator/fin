package postgres

import "github.com/tahminator/fin/internal/domain"

func accountTypeStrings(types []domain.AccountType) []string {
	values := make([]string, len(types))
	for i, accountType := range types {
		values[i] = string(accountType)
	}

	return values
}
