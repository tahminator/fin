package plaid

import (
	"errors"
	"testing"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"

	"github.com/tahminator/fin/internal/domain"
)

func TestMapInvestmentTransactionNegatesCashOutflows(t *testing.T) {
	tests := []struct {
		name       string
		amount     float64
		fees       *float64
		wantAmount string
		wantFees   *string
	}{
		{name: "stock purchase debits cash", amount: 400.5, wantAmount: "-400.5"},
		{name: "sale credits cash", amount: -510.25, wantAmount: "510.25"},
		{name: "zero amount", amount: 0, wantAmount: "0"},
		{name: "fee debit is an outflow and keeps its fee total", amount: 4.95, fees: new(4.95), wantAmount: "-4.95", wantFees: new("4.95")},
		{name: "sale fees are reported as a positive total", amount: -99, fees: new(1.0), wantAmount: "99", wantFees: new("1")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapInvestmentTransaction(&plaidsdk.InvestmentTransaction{
				InvestmentTransactionID: "tx-1",
				AccountID:               "acct-1",
				SecurityID:              new("sec-1"),
				Date:                    time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC),
				Name:                    "Trade",
				Quantity:                -2,
				Amount:                  test.amount,
				Price:                   200.25,
				Fees:                    test.fees,
				Type:                    plaidsdk.InvestmentTransactionTypeBuy,
				Subtype:                 plaidsdk.InvestmentTransactionSubtypeBuy,
				IsoCurrencyCode:         new("USD"),
			})

			if mapped.Amount != test.wantAmount || !equalPtr(mapped.Fees, test.wantFees) {
				t.Errorf("amount %s fees %s, want %s %s", mapped.Amount, show(mapped.Fees), test.wantAmount, show(test.wantFees))
			}

			if mapped.Quantity != "-2" || mapped.Price != "200.25" || *mapped.SecurityExternalID != "sec-1" || *mapped.Currency != "USD" || mapped.Type != "buy" {
				t.Errorf("mapped = %+v", mapped)
			}
		})
	}
}

func TestMapInvestmentHoldingKeepsOptionalDecimalsNil(t *testing.T) {
	mapped := mapInvestmentHolding(&plaidsdk.Holding{
		AccountID:              "acct-1",
		SecurityID:             "sec-1",
		InstitutionPrice:       250.5,
		InstitutionValue:       1002,
		Quantity:               4,
		CostBasis:              new(900.1),
		UnofficialCurrencyCode: new("BTC"),
	})

	if mapped.InstitutionValue != "1002" || mapped.Quantity != "4" || *mapped.CostBasis != "900.1" || mapped.VestedValue != nil || *mapped.Currency != "BTC" {
		t.Errorf("mapped = %+v", mapped)
	}
}

func TestMapInvestmentsErrorClassifiesUnavailableProduct(t *testing.T) {
	tests := []struct {
		code string
		want domain.InvestmentsStatus
	}{
		{code: "ADDITIONAL_CONSENT_REQUIRED", want: domain.InvestmentsStatusConsentRequired},
		{code: "PRODUCT_NOT_READY", want: domain.InvestmentsStatusPending},
		{code: "INVALID_PRODUCT", want: domain.InvestmentsStatusNotEnabled},
		{code: "PRODUCTS_NOT_SUPPORTED", want: domain.InvestmentsStatusUnsupported},
		{code: "NO_INVESTMENT_ACCOUNTS", want: domain.InvestmentsStatusUnsupported},
		{code: "NO_INVESTMENT_AUTH_ACCOUNTS", want: domain.InvestmentsStatusUnsupported},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			var unavailable *domain.InvestmentsUnavailableError

			err := mapInvestmentsError(badRequest(test.code, "message"))
			if !errors.As(err, &unavailable) || unavailable.Status != test.want || !errors.Is(err, domain.ErrInvestmentsUnavailable) {
				t.Errorf("mapInvestmentsError(%s) = %v, want status %s", test.code, err, test.want)
			}
		})
	}
}

func TestMapInvestmentsErrorKeepsOtherFailures(t *testing.T) {
	if err := mapInvestmentsError(badRequest(itemLoginRequired, "login")); !errors.Is(err, domain.ErrReauthRequired) {
		t.Errorf("login required = %v, want ErrReauthRequired", err)
	}

	var plaidErr *Error

	if err := mapInvestmentsError(badRequest("INTERNAL_SERVER_ERROR", "boom")); errors.Is(err, domain.ErrInvestmentsUnavailable) || !errors.As(err, &plaidErr) {
		t.Errorf("unexpected failure = %v, want a plain Plaid error", err)
	}
}
