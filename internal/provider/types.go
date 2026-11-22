package provider

import (
	"time"

	"github.com/tahminator/fin/internal/domain"
)

type LinkedItem struct {
	ExternalID      string
	AccessToken     string
	InstitutionID   *string
	InstitutionName *string
}

type InstitutionInfo struct {
	URL *string
}

type Account struct {
	ExternalID       string
	Name             string
	OfficialName     *string
	Mask             *string
	Type             domain.AccountType
	Subtype          *string
	CurrentBalance   *string
	AvailableBalance *string
	Currency         *string
}

type Transaction struct {
	ExternalID        string
	AccountExternalID string
	Amount            string
	Currency          *string
	PostedOn          time.Time
	AuthorizedOn      *time.Time
	Name              string
	MerchantName      *string
	Category          *string
	Pending           bool
}

type SyncResult struct {
	Added      []Transaction
	Modified   []Transaction
	Removed    []string
	NextCursor string
}

type InvestmentSecurity struct {
	ExternalID string
	Name       *string
	Ticker     *string
	Type       *string
	Subtype    *string
	Currency   *string
}

type InvestmentHolding struct {
	AccountExternalID    string
	SecurityExternalID   string
	InstitutionPrice     string
	InstitutionPriceAsOf *time.Time
	InstitutionValue     string
	CostBasis            *string
	Quantity             string
	Currency             *string
	VestedQuantity       *string
	VestedValue          *string
}

type InvestmentHoldingsResult struct {
	Securities []InvestmentSecurity
	Holdings   []InvestmentHolding
}

type InvestmentTransaction struct {
	ExternalID          string
	AccountExternalID   string
	SecurityExternalID  *string
	PostedOn            time.Time
	TransactionDatetime *time.Time
	Name                string
	Quantity            string
	Amount              string
	Price               string
	Fees                *string
	Type                string
	Subtype             string
	Currency            *string
}

type InvestmentTransactionsResult struct {
	Securities   []InvestmentSecurity
	Transactions []InvestmentTransaction
}
