package domain

import (
	"time"
	"uuid"
)

type InvestmentSecurity struct {
	ID         uuid.UUID `db:"id"`
	ExternalID string    `db:"external_id"`
	Name       *string   `db:"name"`
	Ticker     *string   `db:"ticker_symbol"`
	Type       *string   `db:"type"`
	Subtype    *string   `db:"subtype"`
	Currency   *string   `db:"currency"`
}

type InvestmentHolding struct {
	ID                   uuid.UUID `db:"id"`
	AccountID            uuid.UUID `db:"account_id"`
	AccountExternalID    string
	AccountName          string    `db:"account_name"`
	SecurityID           uuid.UUID `db:"security_id"`
	SecurityExternalID   string
	SecurityName         *string    `db:"security_name"`
	Ticker               *string    `db:"ticker_symbol"`
	InstitutionPrice     string     `db:"institution_price"`
	InstitutionPriceAsOf *time.Time `db:"institution_price_as_of"`
	InstitutionValue     string     `db:"institution_value"`
	CostBasis            *string    `db:"cost_basis"`
	Quantity             string     `db:"quantity"`
	Currency             *string    `db:"currency"`
	VestedQuantity       *string    `db:"vested_quantity"`
	VestedValue          *string    `db:"vested_value"`
}

type InvestmentTransaction struct {
	ID                  uuid.UUID `db:"id"`
	AccountID           uuid.UUID `db:"account_id"`
	AccountExternalID   string
	AccountName         string     `db:"account_name"`
	SecurityID          *uuid.UUID `db:"security_id"`
	SecurityExternalID  *string
	SecurityName        *string    `db:"security_name"`
	Ticker              *string    `db:"ticker_symbol"`
	ExternalID          string     `db:"external_id"`
	CancelExternalID    *string    `db:"cancel_external_id"`
	PostedOn            time.Time  `db:"posted_on"`
	TransactionDatetime *time.Time `db:"transaction_datetime"`
	Name                string     `db:"name"`
	Quantity            *string    `db:"quantity"`
	Amount              string     `db:"amount"`
	Price               *string    `db:"price"`
	Fees                *string    `db:"fees"`
	Type                string     `db:"type"`
	Subtype             string     `db:"subtype"`
	Currency            *string    `db:"currency"`
}
