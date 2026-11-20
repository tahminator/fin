package domain

import "uuid"

type AccountType string

const (
	AccountTypeDepository AccountType = "depository"
	AccountTypeCredit     AccountType = "credit"
	AccountTypeLoan       AccountType = "loan"
	AccountTypeInvestment AccountType = "investment"
	AccountTypeOther      AccountType = "other"
)

func (t AccountType) Valid() bool {
	switch t {
	case AccountTypeDepository, AccountTypeCredit, AccountTypeLoan, AccountTypeInvestment, AccountTypeOther:
		return true
	}

	return false
}

type Account struct {
	ID               uuid.UUID   `db:"id"`
	UserID           uuid.UUID   `db:"user_id"`
	ConnectionID     uuid.UUID   `db:"connection_id"`
	InstitutionName  *string     `db:"institution_name"`
	InstitutionURL   *string     `db:"institution_url"`
	ExternalID       string      `db:"external_id"`
	Name             string      `db:"name"`
	OfficialName     *string     `db:"official_name"`
	Mask             *string     `db:"mask"`
	Type             AccountType `db:"type"`
	Subtype          *string     `db:"subtype"`
	CurrentBalance   *string     `db:"current_balance"`
	AvailableBalance *string     `db:"available_balance"`
	Currency         *string     `db:"currency"`
}
