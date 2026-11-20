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
