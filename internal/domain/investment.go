package domain

import (
	"errors"
	"time"
	"uuid"
)

type InvestmentsStatus string

const (
	InvestmentsStatusReady           InvestmentsStatus = "ready"
	InvestmentsStatusPending         InvestmentsStatus = "pending"
	InvestmentsStatusConsentRequired InvestmentsStatus = "consent_required"
	InvestmentsStatusNotEnabled      InvestmentsStatus = "not_enabled"
	InvestmentsStatusUnsupported     InvestmentsStatus = "unsupported"
)

func (s InvestmentsStatus) Valid() bool {
	switch s {
	case InvestmentsStatusReady,
		InvestmentsStatusPending,
		InvestmentsStatusConsentRequired,
		InvestmentsStatusNotEnabled,
		InvestmentsStatusUnsupported:
		return true
	}

	return false
}

var ErrInvestmentsUnavailable = errors.New("investments unavailable")

type InvestmentsUnavailableError struct {
	Status InvestmentsStatus
	Reason string
}

func (e *InvestmentsUnavailableError) Error() string {
	return "investments " + string(e.Status) + ": " + e.Reason
}

func (e *InvestmentsUnavailableError) Is(target error) bool {
	return target == ErrInvestmentsUnavailable
}

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
	ID                   uuid.UUID  `db:"id"`
	AccountID            uuid.UUID  `db:"account_id"`
	AccountName          string     `db:"account_name"`
	SecurityID           uuid.UUID  `db:"security_id"`
	SecurityExternalID   string     `db:"security_external_id"`
	SecurityName         *string    `db:"security_name"`
	Ticker               *string    `db:"ticker_symbol"`
	SecurityType         *string    `db:"security_type"`
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
	ID                  uuid.UUID  `db:"id"`
	AccountID           uuid.UUID  `db:"account_id"`
	AccountName         string     `db:"account_name"`
	SecurityID          *uuid.UUID `db:"security_id"`
	SecurityExternalID  *string    `db:"security_external_id"`
	SecurityName        *string    `db:"security_name"`
	Ticker              *string    `db:"ticker_symbol"`
	ExternalID          string     `db:"external_id"`
	PostedOn            time.Time  `db:"posted_on"`
	TransactionDatetime *time.Time `db:"transaction_datetime"`
	Name                string     `db:"name"`
	Quantity            string     `db:"quantity"`
	Amount              string     `db:"amount"`
	Price               string     `db:"price"`
	Fees                *string    `db:"fees"`
	Type                string     `db:"type"`
	Subtype             string     `db:"subtype"`
	Currency            *string    `db:"currency"`
}

type InvestmentSecuritySummary struct {
	ID       uuid.UUID `db:"id"`
	Name     *string   `db:"name"`
	Ticker   *string   `db:"ticker_symbol"`
	Type     *string   `db:"type"`
	Subtype  *string   `db:"subtype"`
	Currency *string   `db:"currency"`
	Held     bool      `db:"held"`
}
