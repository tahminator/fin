package plaid

import (
	"context"
	"errors"
	"strconv"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"
	"github.com/plaid/plaid-go/v50/core"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

const investmentTransactionsPageSize = 500

var investmentsUnavailableCodes = map[string]domain.InvestmentsStatus{
	"ADDITIONAL_CONSENT_REQUIRED": domain.InvestmentsStatusConsentRequired,
	"PRODUCT_NOT_READY":           domain.InvestmentsStatusPending,
	"INVALID_PRODUCT":             domain.InvestmentsStatusNotEnabled,
	"PRODUCTS_NOT_SUPPORTED":      domain.InvestmentsStatusUnsupported,
	"NO_INVESTMENT_ACCOUNTS":      domain.InvestmentsStatusUnsupported,
	"NO_INVESTMENT_AUTH_ACCOUNTS": domain.InvestmentsStatusUnsupported,
}

func (p *Provider) InvestmentHoldings(ctx context.Context, creds domain.Credential, accessToken string) (provider.InvestmentHoldingsResult, error) {
	started := time.Now()

	result, err := p.investmentHoldings(ctx, creds, accessToken)

	p.observe(metrics.OperationInvestmentHoldings, started, err)

	return result, err
}

func (p *Provider) InvestmentTransactions(
	ctx context.Context,
	creds domain.Credential,
	accessToken string,
	start, end time.Time,
) (provider.InvestmentTransactionsResult, error) {
	started := time.Now()

	result, err := p.investmentTransactions(ctx, creds, accessToken, start, end)

	p.observe(metrics.OperationInvestmentTransactions, started, err)

	return result, err
}

func (p *Provider) investmentHoldings(ctx context.Context, creds domain.Credential, accessToken string) (provider.InvestmentHoldingsResult, error) {
	c, err := p.client(creds)
	if err != nil {
		return provider.InvestmentHoldingsResult{}, err
	}

	response, err := c.Investments.GetHoldings(ctx, &plaidsdk.InvestmentsHoldingsGetRequest{AccessToken: accessToken})
	if err != nil {
		return provider.InvestmentHoldingsResult{}, mapInvestmentsError(err)
	}

	holdings := make([]provider.InvestmentHolding, 0, len(response.Holdings))

	for _, holding := range response.Holdings {
		if holding != nil {
			holdings = append(holdings, mapInvestmentHolding(holding))
		}
	}

	return provider.InvestmentHoldingsResult{Securities: mapSecurities(response.Securities), Holdings: holdings}, nil
}

func (p *Provider) investmentTransactions(
	ctx context.Context,
	creds domain.Credential,
	accessToken string,
	start, end time.Time,
) (provider.InvestmentTransactionsResult, error) {
	c, err := p.client(creds)
	if err != nil {
		return provider.InvestmentTransactionsResult{}, err
	}

	page, err := c.Investments.GetTransactions(ctx, &plaidsdk.InvestmentsTransactionsGetRequest{
		AccessToken: accessToken,
		StartDate:   start,
		EndDate:     end,
		Options:     &plaidsdk.InvestmentsTransactionsGetRequestOptions{Count: new(investmentTransactionsPageSize)},
	})

	result := provider.InvestmentTransactionsResult{
		Securities:   []provider.InvestmentSecurity{},
		Transactions: []provider.InvestmentTransaction{},
	}

	for err == nil {
		if page.Response != nil {
			result.Securities = append(result.Securities, mapSecurities(page.Response.Securities)...)
		}

		for _, transaction := range page.Results {
			if transaction != nil {
				result.Transactions = append(result.Transactions, mapInvestmentTransaction(transaction))
			}
		}

		page, err = page.GetNextPage(ctx)
	}

	if !errors.Is(err, core.ErrNoPages) {
		return provider.InvestmentTransactionsResult{}, mapInvestmentsError(err)
	}

	return result, nil
}

func mapInvestmentsError(err error) error {
	var plaidErr plaidError

	if errors.As(err, &plaidErr) && plaidErr.GetBody() != nil {
		body := plaidErr.GetBody()

		if status, ok := investmentsUnavailableCodes[body.ErrorCode]; ok {
			return &domain.InvestmentsUnavailableError{Status: status, Reason: body.ErrorCode + ": " + body.ErrorMessage}
		}
	}

	return mapError(err)
}

func mapSecurities(securities []*plaidsdk.Security) []provider.InvestmentSecurity {
	mapped := make([]provider.InvestmentSecurity, 0, len(securities))

	for _, security := range securities {
		if security != nil {
			mapped = append(mapped, mapInvestmentSecurity(security))
		}
	}

	return mapped
}

func mapInvestmentSecurity(security *plaidsdk.Security) provider.InvestmentSecurity {
	return provider.InvestmentSecurity{
		ExternalID: security.SecurityID,
		Name:       security.Name,
		Ticker:     security.TickerSymbol,
		Type:       security.Type,
		Subtype:    security.Subtype,
		Currency:   currencyOf(security.IsoCurrencyCode, security.UnofficialCurrencyCode),
	}
}

func mapInvestmentHolding(holding *plaidsdk.Holding) provider.InvestmentHolding {
	return provider.InvestmentHolding{
		AccountExternalID:    holding.AccountID,
		SecurityExternalID:   holding.SecurityID,
		InstitutionPrice:     formatDecimal(holding.InstitutionPrice),
		InstitutionPriceAsOf: holding.InstitutionPriceAsOf,
		InstitutionValue:     formatDecimal(holding.InstitutionValue),
		CostBasis:            formatBalance(holding.CostBasis),
		Quantity:             formatDecimal(holding.Quantity),
		Currency:             currencyOf(holding.IsoCurrencyCode, holding.UnofficialCurrencyCode),
		VestedQuantity:       formatBalance(holding.VestedQuantity),
		VestedValue:          formatBalance(holding.VestedValue),
	}
}

func mapInvestmentTransaction(transaction *plaidsdk.InvestmentTransaction) provider.InvestmentTransaction {
	return provider.InvestmentTransaction{
		ExternalID:          transaction.InvestmentTransactionID,
		AccountExternalID:   transaction.AccountID,
		SecurityExternalID:  transaction.SecurityID,
		PostedOn:            transaction.Date,
		TransactionDatetime: transaction.TransactionDatetime,
		Name:                transaction.Name,
		Quantity:            formatDecimal(transaction.Quantity),
		Amount:              formatAmount(transaction.Amount),
		Price:               formatDecimal(transaction.Price),
		Fees:                formatBalance(transaction.Fees),
		Type:                string(transaction.Type),
		Subtype:             string(transaction.Subtype),
		Currency:            currencyOf(transaction.IsoCurrencyCode, transaction.UnofficialCurrencyCode),
	}
}

func formatDecimal(value float64) string {
	if value == 0 {
		return "0"
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}
