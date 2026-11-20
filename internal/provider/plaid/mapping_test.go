package plaid

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"
	"github.com/plaid/plaid-go/v50/core"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/provider"
)

func badRequest(code, message string) error {
	return &plaidsdk.BadRequestError{
		APIError: core.NewAPIError(400, nil, nil),
		Body: &plaidsdk.PlaidError{
			ErrorType:    plaidsdk.PlaidErrorTypeItemError,
			ErrorCode:    code,
			ErrorMessage: message,
		},
	}
}

func equalPtr[T comparable](got, want *T) bool {
	if got == nil || want == nil {
		return got == want
	}

	return *got == *want
}

func show[T any](value *T) string {
	if value == nil {
		return "<nil>"
	}

	return fmt.Sprint(*value)
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{name: "outflow becomes negative", amount: 12.5, want: "-12.5"},
		{name: "inflow becomes positive", amount: -1043.27, want: "1043.27"},
		{name: "zero", amount: 0, want: "0"},
		{name: "negative zero", amount: math.Copysign(0, -1), want: "0"},
		{name: "whole number", amount: 42, want: "-42"},
		{name: "huge value", amount: 1e21, want: "-1000000000000000000000"},
		{name: "huge inflow", amount: -2.5e22, want: "25000000000000000000000"},
		{name: "tiny value", amount: 0.0001, want: "-0.0001"},
		{name: "tinier value", amount: -1e-7, want: "0.0000001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAmount(tt.amount)

			if strings.ContainsAny(got, "eE") {
				t.Fatalf("got %q, contains an exponent", got)
			}

			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatBalance(t *testing.T) {
	tests := []struct {
		name    string
		balance *float64
		want    *string
	}{
		{name: "nil stays nil", balance: nil, want: nil},
		{name: "sign is preserved", balance: new(-12.5), want: new("-12.5")},
		{name: "zero", balance: new(0.0), want: new("0")},
		{name: "huge value has no exponent", balance: new(1e21), want: new("1000000000000000000000")},
		{name: "tiny value has no exponent", balance: new(0.0001), want: new("0.0001")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBalance(tt.balance); !equalPtr(got, tt.want) {
				t.Fatalf("got %s, want %s", show(got), show(tt.want))
			}
		})
	}
}

func TestMapErrorReturnsNilForNil(t *testing.T) {
	if err := mapError(nil); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
}

func TestMapErrorItemLoginRequiredIsReauthRequired(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "bare SDK error", err: badRequest("ITEM_LOGIN_REQUIRED", "the login details of this item have changed")},
		{name: "wrapped SDK error", err: fmt.Errorf("call: %w", badRequest("ITEM_LOGIN_REQUIRED", "the login details of this item have changed"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapError(tt.err)

			if !errors.Is(err, domain.ErrReauthRequired) {
				t.Fatalf("got %v, want ErrReauthRequired", err)
			}

			if !strings.Contains(err.Error(), "the login details of this item have changed") {
				t.Fatalf("message %q does not contain the Plaid message", err.Error())
			}
		})
	}
}

func TestMapErrorOtherPlaidErrorsKeepTheirCode(t *testing.T) {
	serverErr := &plaidsdk.InternalServerError{
		APIError: core.NewAPIError(500, nil, nil),
		Body:     &plaidsdk.PlaidError{ErrorType: plaidsdk.PlaidErrorTypeAPIError, ErrorCode: "INTERNAL_SERVER_ERROR", ErrorMessage: "unexpected error"},
	}

	tests := []struct {
		name        string
		err         error
		wantCode    string
		wantMessage string
	}{
		{name: "invalid access token", err: badRequest("INVALID_ACCESS_TOKEN", "bad token"), wantCode: "INVALID_ACCESS_TOKEN", wantMessage: "bad token"},
		{name: "mutation during pagination", err: badRequest(mutationErrorCode, "data changed"), wantCode: mutationErrorCode, wantMessage: "data changed"},
		{name: "server error", err: serverErr, wantCode: "INTERNAL_SERVER_ERROR", wantMessage: "unexpected error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapError(tt.err)

			if errors.Is(err, domain.ErrReauthRequired) {
				t.Fatalf("got ErrReauthRequired for %s", tt.wantCode)
			}

			var mapped *Error
			if !errors.As(err, &mapped) {
				t.Fatalf("got %T, want *Error", err)
			}

			if mapped.Code != tt.wantCode || mapped.Message != tt.wantMessage {
				t.Fatalf("got code %q message %q, want %q %q", mapped.Code, mapped.Message, tt.wantCode, tt.wantMessage)
			}

			if !strings.Contains(err.Error(), tt.wantCode) {
				t.Fatalf("message %q does not contain code %q", err.Error(), tt.wantCode)
			}

			if !errors.Is(err, tt.err) {
				t.Fatal("mapped error does not wrap the SDK error")
			}
		})
	}
}

func TestMapErrorWrapsNonPlaidErrors(t *testing.T) {
	missingBody := &plaidsdk.BadRequestError{APIError: core.NewAPIError(400, nil, nil)}

	tests := []struct {
		name string
		err  error
	}{
		{name: "plain error", err: errors.New("connection reset")},
		{name: "SDK error without body", err: missingBody},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapError(tt.err)

			if !errors.Is(err, tt.err) {
				t.Fatalf("got %v, does not wrap the original error", err)
			}

			if errors.Is(err, domain.ErrReauthRequired) {
				t.Fatal("got ErrReauthRequired")
			}

			var mapped *Error
			if errors.As(err, &mapped) {
				t.Fatalf("got *Error with code %q, want a plain wrap", mapped.Code)
			}

			if !strings.HasPrefix(err.Error(), "plaid") {
				t.Fatalf("message %q is not prefixed with plaid", err.Error())
			}
		})
	}
}

func TestIsMutationDuringPagination(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "mutation during pagination", err: badRequest(mutationErrorCode, "data changed"), want: true},
		{name: "wrapped mutation during pagination", err: fmt.Errorf("sync: %w", badRequest(mutationErrorCode, "data changed")), want: true},
		{name: "other Plaid code", err: badRequest("ITEM_LOGIN_REQUIRED", "login"), want: false},
		{name: "SDK error without body", err: &plaidsdk.BadRequestError{APIError: core.NewAPIError(400, nil, nil)}, want: false},
		{name: "plain error with matching text", err: errors.New(mutationErrorCode), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMutationDuringPagination(tt.err); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAccountType(t *testing.T) {
	tests := []struct {
		accountType plaidsdk.AccountType
		want        domain.AccountType
	}{
		{accountType: plaidsdk.AccountTypeDepository, want: domain.AccountTypeDepository},
		{accountType: plaidsdk.AccountTypeCredit, want: domain.AccountTypeCredit},
		{accountType: plaidsdk.AccountTypeLoan, want: domain.AccountTypeLoan},
		{accountType: plaidsdk.AccountTypeInvestment, want: domain.AccountTypeInvestment},
		{accountType: plaidsdk.AccountTypeBrokerage, want: domain.AccountTypeInvestment},
		{accountType: plaidsdk.AccountTypeOther, want: domain.AccountTypeOther},
		{accountType: plaidsdk.AccountType("crypto"), want: domain.AccountTypeOther},
		{accountType: plaidsdk.AccountType(""), want: domain.AccountTypeOther},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			if got := mapAccountType(tt.accountType); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapAccount(t *testing.T) {
	subtype := plaidsdk.AccountSubtypeChecking

	tests := []struct {
		name    string
		account *plaidsdk.AccountBase
		want    provider.Account
	}{
		{
			name: "balances formatted and iso currency preferred",
			account: &plaidsdk.AccountBase{
				AccountID:    "acc-1",
				Name:         "Checking",
				OfficialName: new("Premier Checking"),
				Mask:         new("0000"),
				Type:         plaidsdk.AccountTypeDepository,
				Subtype:      &subtype,
				Balances: &plaidsdk.AccountBalance{
					Current:                new(1234.56),
					Available:              new(1e21),
					IsoCurrencyCode:        new("USD"),
					UnofficialCurrencyCode: new("BTC"),
				},
			},
			want: provider.Account{
				ExternalID:       "acc-1",
				Name:             "Checking",
				OfficialName:     new("Premier Checking"),
				Mask:             new("0000"),
				Type:             domain.AccountTypeDepository,
				Subtype:          new("checking"),
				CurrentBalance:   new("1234.56"),
				AvailableBalance: new("1000000000000000000000"),
				Currency:         new("USD"),
			},
		},
		{
			name: "unofficial currency used without iso and nil balance fields stay nil",
			account: &plaidsdk.AccountBase{
				AccountID: "acc-2",
				Name:      "Wallet",
				Type:      plaidsdk.AccountType("crypto"),
				Balances: &plaidsdk.AccountBalance{
					Current:                new(-0.5),
					UnofficialCurrencyCode: new("BTC"),
				},
			},
			want: provider.Account{
				ExternalID:     "acc-2",
				Name:           "Wallet",
				Type:           domain.AccountTypeOther,
				CurrentBalance: new("-0.5"),
				Currency:       new("BTC"),
			},
		},
		{
			name: "nil balances",
			account: &plaidsdk.AccountBase{
				AccountID: "acc-3",
				Name:      "Brokerage",
				Type:      plaidsdk.AccountTypeBrokerage,
			},
			want: provider.Account{
				ExternalID: "acc-3",
				Name:       "Brokerage",
				Type:       domain.AccountTypeInvestment,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAccount(tt.account)

			if got.ExternalID != tt.want.ExternalID || got.Name != tt.want.Name || got.Type != tt.want.Type {
				t.Errorf("got id %q name %q type %q, want %q %q %q", got.ExternalID, got.Name, got.Type, tt.want.ExternalID, tt.want.Name, tt.want.Type)
			}

			pointers := []struct {
				field     string
				got, want *string
			}{
				{field: "OfficialName", got: got.OfficialName, want: tt.want.OfficialName},
				{field: "Mask", got: got.Mask, want: tt.want.Mask},
				{field: "Subtype", got: got.Subtype, want: tt.want.Subtype},
				{field: "CurrentBalance", got: got.CurrentBalance, want: tt.want.CurrentBalance},
				{field: "AvailableBalance", got: got.AvailableBalance, want: tt.want.AvailableBalance},
				{field: "Currency", got: got.Currency, want: tt.want.Currency},
			}

			for _, p := range pointers {
				if !equalPtr(p.got, p.want) {
					t.Errorf("%s: got %s, want %s", p.field, show(p.got), show(p.want))
				}
			}
		})
	}
}

func TestMapTransaction(t *testing.T) {
	posted := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	authorized := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		transaction *plaidsdk.Transaction
		want        provider.Transaction
	}{
		{
			name: "full transaction",
			transaction: &plaidsdk.Transaction{
				TransactionID:           "txn-1",
				AccountID:               "acc-1",
				Amount:                  12.5,
				IsoCurrencyCode:         new("USD"),
				UnofficialCurrencyCode:  new("BTC"),
				Date:                    posted,
				AuthorizedDate:          &authorized,
				Name:                    "Coffee Shop",
				MerchantName:            new("Coffee Co"),
				PersonalFinanceCategory: &plaidsdk.PersonalFinanceCategory{Primary: "FOOD_AND_DRINK", Detailed: "FOOD_AND_DRINK_COFFEE"},
				Category:                []string{"Legacy", "Category"},
				Pending:                 true,
			},
			want: provider.Transaction{
				ExternalID:        "txn-1",
				AccountExternalID: "acc-1",
				Amount:            "-12.5",
				Currency:          new("USD"),
				PostedOn:          posted,
				AuthorizedOn:      &authorized,
				Name:              "Coffee Shop",
				MerchantName:      new("Coffee Co"),
				Category:          new("FOOD_AND_DRINK"),
				Pending:           true,
			},
		},
		{
			name: "unofficial currency without iso and no category",
			transaction: &plaidsdk.Transaction{
				TransactionID:          "txn-2",
				AccountID:              "acc-2",
				Amount:                 -1043.27,
				UnofficialCurrencyCode: new("BTC"),
				Date:                   posted,
				Name:                   "Deposit",
			},
			want: provider.Transaction{
				ExternalID:        "txn-2",
				AccountExternalID: "acc-2",
				Amount:            "1043.27",
				Currency:          new("BTC"),
				PostedOn:          posted,
				Name:              "Deposit",
			},
		},
		{
			name: "empty primary category is nil",
			transaction: &plaidsdk.Transaction{
				TransactionID:           "txn-3",
				AccountID:               "acc-3",
				Amount:                  0,
				Date:                    posted,
				Name:                    "Adjustment",
				PersonalFinanceCategory: &plaidsdk.PersonalFinanceCategory{Detailed: "OTHER"},
			},
			want: provider.Transaction{
				ExternalID:        "txn-3",
				AccountExternalID: "acc-3",
				Amount:            "0",
				PostedOn:          posted,
				Name:              "Adjustment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTransaction(tt.transaction)

			if got.ExternalID != tt.want.ExternalID || got.AccountExternalID != tt.want.AccountExternalID || got.Name != tt.want.Name || got.Pending != tt.want.Pending {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}

			if got.Amount != tt.want.Amount {
				t.Errorf("Amount: got %q, want %q", got.Amount, tt.want.Amount)
			}

			if !got.PostedOn.Equal(tt.want.PostedOn) {
				t.Errorf("PostedOn: got %v, want %v", got.PostedOn, tt.want.PostedOn)
			}

			if !equalPtr(got.AuthorizedOn, tt.want.AuthorizedOn) {
				t.Errorf("AuthorizedOn: got %s, want %s", show(got.AuthorizedOn), show(tt.want.AuthorizedOn))
			}

			pointers := []struct {
				field     string
				got, want *string
			}{
				{field: "Currency", got: got.Currency, want: tt.want.Currency},
				{field: "MerchantName", got: got.MerchantName, want: tt.want.MerchantName},
				{field: "Category", got: got.Category, want: tt.want.Category},
			}

			for _, p := range pointers {
				if !equalPtr(p.got, p.want) {
					t.Errorf("%s: got %s, want %s", p.field, show(p.got), show(p.want))
				}
			}
		})
	}
}

func TestMapTransactionsPreservesOrderAndIsNeverNil(t *testing.T) {
	empty := mapTransactions(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("got %#v, want empty non-nil slice", empty)
	}

	mapped := mapTransactions([]*plaidsdk.Transaction{
		{TransactionID: "txn-1", Amount: 1},
		{TransactionID: "txn-2", Amount: -2},
	})

	if len(mapped) != 2 || mapped[0].ExternalID != "txn-1" || mapped[1].ExternalID != "txn-2" {
		t.Fatalf("got %+v, want txn-1 then txn-2", mapped)
	}

	if mapped[0].Amount != "-1" || mapped[1].Amount != "2" {
		t.Fatalf("got amounts %q %q, want -1 and 2", mapped[0].Amount, mapped[1].Amount)
	}
}
