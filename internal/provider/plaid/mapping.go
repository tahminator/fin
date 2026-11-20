package plaid

import (
	"errors"
	"fmt"
	"strconv"

	plaidsdk "github.com/plaid/plaid-go/v50"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/provider"
)

const itemLoginRequired = "ITEM_LOGIN_REQUIRED"

type plaidError interface {
	GetBody() *plaidsdk.PlaidError
}

type Error struct {
	Code    string
	Message string
	cause   error
}

func (e *Error) Error() string {
	return fmt.Sprintf("plaid %s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.cause
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	var plaidErr plaidError

	if !errors.As(err, &plaidErr) || plaidErr.GetBody() == nil {
		return fmt.Errorf("plaid: %w", err)
	}

	body := plaidErr.GetBody()

	if body.ErrorCode == itemLoginRequired {
		return fmt.Errorf("%w: %s", domain.ErrReauthRequired, body.ErrorMessage)
	}

	return &Error{Code: body.ErrorCode, Message: body.ErrorMessage, cause: err}
}

func mapAccount(account *plaidsdk.AccountBase) provider.Account {
	mapped := provider.Account{
		ExternalID:   account.AccountID,
		Name:         account.Name,
		OfficialName: account.OfficialName,
		Mask:         account.Mask,
		Type:         mapAccountType(account.Type),
	}

	if account.Subtype != nil {
		subtype := string(*account.Subtype)
		mapped.Subtype = &subtype
	}

	if balances := account.Balances; balances != nil {
		mapped.CurrentBalance = formatBalance(balances.Current)
		mapped.AvailableBalance = formatBalance(balances.Available)
		mapped.Currency = currencyOf(balances.IsoCurrencyCode, balances.UnofficialCurrencyCode)
	}

	return mapped
}

func mapAccountType(accountType plaidsdk.AccountType) domain.AccountType {
	switch accountType {
	case plaidsdk.AccountTypeDepository:
		return domain.AccountTypeDepository
	case plaidsdk.AccountTypeCredit:
		return domain.AccountTypeCredit
	case plaidsdk.AccountTypeLoan:
		return domain.AccountTypeLoan
	case plaidsdk.AccountTypeInvestment, plaidsdk.AccountTypeBrokerage:
		return domain.AccountTypeInvestment
	case plaidsdk.AccountTypeOther:
		return domain.AccountTypeOther
	}

	return domain.AccountTypeOther
}

func mapTransactions(transactions []*plaidsdk.Transaction) []provider.Transaction {
	mapped := make([]provider.Transaction, 0, len(transactions))

	for _, transaction := range transactions {
		mapped = append(mapped, mapTransaction(transaction))
	}

	return mapped
}

func mapTransaction(transaction *plaidsdk.Transaction) provider.Transaction {
	mapped := provider.Transaction{
		ExternalID:        transaction.TransactionID,
		AccountExternalID: transaction.AccountID,
		Amount:            formatAmount(transaction.Amount),
		Currency:          currencyOf(transaction.IsoCurrencyCode, transaction.UnofficialCurrencyCode),
		PostedOn:          transaction.Date,
		AuthorizedOn:      transaction.AuthorizedDate,
		Name:              transaction.Name,
		MerchantName:      transaction.MerchantName,
		Pending:           transaction.Pending,
	}

	if category := transaction.PersonalFinanceCategory; category != nil && category.Primary != "" {
		mapped.Category = &category.Primary
	}

	return mapped
}

func formatAmount(plaidAmount float64) string {
	if plaidAmount == 0 {
		return "0"
	}

	return strconv.FormatFloat(-plaidAmount, 'f', -1, 64)
}

func formatBalance(balance *float64) *string {
	if balance == nil {
		return nil
	}

	formatted := strconv.FormatFloat(*balance, 'f', -1, 64)

	return &formatted
}

func currencyOf(iso, unofficial *string) *string {
	if iso != nil {
		return iso
	}

	return unofficial
}
