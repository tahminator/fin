package domain

import (
	"time"
	"uuid"
)

type Transaction struct {
	ID           uuid.UUID  `db:"id"`
	AccountID    uuid.UUID  `db:"account_id"`
	AccountName  string     `db:"account_name"`
	ExternalID   string     `db:"external_id"`
	Amount       string     `db:"amount"`
	Currency     *string    `db:"currency"`
	PostedOn     time.Time  `db:"posted_on"`
	AuthorizedOn *time.Time `db:"authorized_on"`
	Name         string     `db:"name"`
	MerchantName *string    `db:"merchant_name"`
	Category     *string    `db:"category"`
	Pending      bool       `db:"pending"`
}
