package handler

import (
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type InvestmentHolding struct {
	ID                   uuid.UUID `json:"id"`
	AccountID            uuid.UUID `json:"accountId"`
	AccountName          string    `json:"accountName"`
	SecurityID           uuid.UUID `json:"securityId"`
	SecurityName         *string   `json:"securityName"`
	Ticker               *string   `json:"ticker"`
	SecurityType         *string   `json:"securityType" doc:"Plaid security type, such as equity, etf, mutual fund or cash"`
	InstitutionPrice     string    `json:"institutionPrice" doc:"Decimal string"`
	InstitutionPriceAsOf *string   `json:"institutionPriceAsOf" format:"date"`
	InstitutionValue     string    `json:"institutionValue" doc:"Decimal string"`
	CostBasis            *string   `json:"costBasis" doc:"Decimal string"`
	Quantity             string    `json:"quantity" doc:"Decimal string"`
	Currency             *string   `json:"currency"`
	VestedQuantity       *string   `json:"vestedQuantity" doc:"Decimal string"`
	VestedValue          *string   `json:"vestedValue" doc:"Decimal string"`
}

func newInvestmentHolding(holding domain.InvestmentHolding) InvestmentHolding {
	return InvestmentHolding{
		ID:                   holding.ID,
		AccountID:            holding.AccountID,
		AccountName:          holding.AccountName,
		SecurityID:           holding.SecurityID,
		SecurityName:         holding.SecurityName,
		Ticker:               holding.Ticker,
		SecurityType:         holding.SecurityType,
		InstitutionPrice:     holding.InstitutionPrice,
		InstitutionPriceAsOf: formatOptionalDate(holding.InstitutionPriceAsOf),
		InstitutionValue:     holding.InstitutionValue,
		CostBasis:            holding.CostBasis,
		Quantity:             holding.Quantity,
		Currency:             holding.Currency,
		VestedQuantity:       holding.VestedQuantity,
		VestedValue:          holding.VestedValue,
	}
}

type InvestmentTransaction struct {
	ID                  uuid.UUID  `json:"id"`
	AccountID           uuid.UUID  `json:"accountId"`
	AccountName         string     `json:"accountName"`
	SecurityID          *uuid.UUID `json:"securityId"`
	SecurityName        *string    `json:"securityName"`
	Ticker              *string    `json:"ticker"`
	PostedOn            string     `json:"postedOn" format:"date"`
	TransactionDatetime *time.Time `json:"transactionDatetime"`
	Name                string     `json:"name"`
	Quantity            string     `json:"quantity" doc:"Decimal string; positive for buys, negative for sells"`
	Amount              string     `json:"amount" doc:"Decimal string; negative is cash leaving the account, positive is cash coming in"`
	Price               string     `json:"price" doc:"Decimal string"`
	Fees                *string    `json:"fees" doc:"Decimal string; total fees as reported, already included in amount"`
	Type                string     `json:"type" doc:"Plaid investment transaction type, such as buy, sell, cash, fee or transfer"`
	Subtype             string     `json:"subtype"`
	Currency            *string    `json:"currency"`
}

func newInvestmentTransaction(transaction domain.InvestmentTransaction) InvestmentTransaction {
	return InvestmentTransaction{
		ID:                  transaction.ID,
		AccountID:           transaction.AccountID,
		AccountName:         transaction.AccountName,
		SecurityID:          transaction.SecurityID,
		SecurityName:        transaction.SecurityName,
		Ticker:              transaction.Ticker,
		PostedOn:            transaction.PostedOn.Format(dateLayout),
		TransactionDatetime: transaction.TransactionDatetime,
		Name:                transaction.Name,
		Quantity:            transaction.Quantity,
		Amount:              transaction.Amount,
		Price:               transaction.Price,
		Fees:                transaction.Fees,
		Type:                transaction.Type,
		Subtype:             transaction.Subtype,
		Currency:            transaction.Currency,
	}
}

type InvestmentSecurity struct {
	ID       uuid.UUID `json:"id"`
	Name     *string   `json:"name"`
	Ticker   *string   `json:"ticker"`
	Type     *string   `json:"type" doc:"Plaid security type, such as equity, etf, mutual fund, fixed income or cash"`
	Subtype  *string   `json:"subtype"`
	Currency *string   `json:"currency"`
	Held     bool      `json:"held" doc:"Whether any visible account currently holds this security"`
}

func newInvestmentSecurity(security domain.InvestmentSecuritySummary) InvestmentSecurity {
	return InvestmentSecurity{
		ID:       security.ID,
		Name:     security.Name,
		Ticker:   security.Ticker,
		Type:     security.Type,
		Subtype:  security.Subtype,
		Currency: security.Currency,
		Held:     security.Held,
	}
}

type InvestmentPoint struct {
	At    time.Time `json:"at" doc:"When the underlying snapshot was captured"`
	Value string    `json:"value" doc:"Decimal string"`
}

type InvestmentSeries struct {
	Currency      string            `json:"currency" doc:"ISO currency code; empty when the institution reported none"`
	Points        []InvestmentPoint `json:"points" nullable:"false"`
	First         string            `json:"first" doc:"Decimal string; first value in the range"`
	Current       string            `json:"current" doc:"Decimal string; latest value in the range"`
	Change        string            `json:"change" doc:"Decimal string; current minus first"`
	ChangePercent *string           `json:"changePercent" doc:"Decimal string; null when the first value is zero"`
	Limited       bool              `json:"limited" doc:"True when collection started inside the range or fewer than two points exist"`
}

func newInvestmentSeries(series domain.InvestmentSeries) InvestmentSeries {
	return InvestmentSeries{
		Currency:      series.Currency,
		Points:        mapItems(series.Points, func(point domain.InvestmentPoint) InvestmentPoint { return InvestmentPoint(point) }),
		First:         series.First,
		Current:       series.Current,
		Change:        series.Change,
		ChangePercent: series.ChangePercent,
		Limited:       series.Limited,
	}
}

type AccountInvestmentSeries struct {
	AccountID uuid.UUID          `json:"accountId"`
	Series    []InvestmentSeries `json:"series" nullable:"false"`
}

type InvestmentHistory struct {
	Range              domain.InvestmentRange    `json:"range" enum:"1d,3d,1w,1m,3m,1y,all"`
	From               *time.Time                `json:"from" doc:"Start of the range; null for all"`
	EarliestCapturedAt *time.Time                `json:"earliestCapturedAt" doc:"First snapshot available for the selected accounts"`
	LatestCapturedAt   *time.Time                `json:"latestCapturedAt" doc:"Most recent snapshot available for the selected accounts"`
	Total              []InvestmentSeries        `json:"total" nullable:"false" doc:"One series per currency across the selected accounts"`
	Accounts           []AccountInvestmentSeries `json:"accounts" nullable:"false"`
}

func newInvestmentHistory(history domain.InvestmentHistory) InvestmentHistory {
	return InvestmentHistory{
		Range:              history.Range,
		From:               history.From,
		EarliestCapturedAt: history.Bounds.Earliest,
		LatestCapturedAt:   history.Bounds.Latest,
		Total:              mapItems(history.Total, newInvestmentSeries),
		Accounts: mapItems(history.Accounts, func(account domain.AccountInvestmentSeries) AccountInvestmentSeries {
			return AccountInvestmentSeries{AccountID: account.AccountID, Series: mapItems(account.Series, newInvestmentSeries)}
		}),
	}
}

func formatOptionalDate(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format(dateLayout)

	return &formatted
}

type ListInvestmentHoldingsInput struct {
	AccountFilterParams
}

type ListInvestmentTransactionsInput struct {
	PageParams
	AccountFilterParams
}

type ListInvestmentSecuritiesInput struct {
	AccountFilterParams
}

type InvestmentHistoryInput struct {
	AccountFilterParams
	Range      domain.InvestmentRange `query:"range" required:"true" enum:"1d,3d,1w,1m,3m,1y,all"`
	SecurityID []string               `query:"securityId,explode" format:"uuid" doc:"Value only these securities; omit for whole account values including cash"`
}

type InvestmentHoldingsOutput struct {
	Body []InvestmentHolding `nullable:"false"`
}

type InvestmentSecuritiesOutput struct {
	Body []InvestmentSecurity `nullable:"false"`
}

type InvestmentHistoryOutput struct {
	Body InvestmentHistory
}
