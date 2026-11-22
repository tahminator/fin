package mcpserver

import (
	"regexp"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const (
	dateLayout        = "2006-01-02"
	defaultLookback   = 90 * 24 * time.Hour
	maxQuerySpan      = 5 * 366 * 24 * time.Hour
	maxFilterListSize = 100
)

var decimalPattern = regexp.MustCompile(`^\d{1,15}(\.\d{1,4})?$`)

type TransactionFilter struct {
	From              string   `json:"from,omitempty" jsonschema:"Inclusive start date (YYYY-MM-DD). Defaults to 90 days before to."`
	To                string   `json:"to,omitempty" jsonschema:"Inclusive end date (YYYY-MM-DD). Defaults to today."`
	AccountIDs        []string `json:"account_ids,omitempty" jsonschema:"Only these accounts (IDs from list_accounts)."`
	ExcludeAccountIDs []string `json:"exclude_account_ids,omitempty" jsonschema:"Leave out these accounts."`
	ConnectionIDs     []string `json:"connection_ids,omitempty" jsonschema:"Only accounts at these banks (IDs from list_connections)."`
	AccountTypes      []string `json:"account_types,omitempty" jsonschema:"Only these account types."`
	Categories        []string `json:"categories,omitempty" jsonschema:"Only these categories (values from list_facets; UNCATEGORIZED matches missing categories)."`
	ExcludeCategories []string `json:"exclude_categories,omitempty" jsonschema:"Leave out these categories."`
	ExcludeTransfers  *bool    `json:"exclude_transfers,omitempty" jsonschema:"Leave out transfers between accounts and loan payments (TRANSFER_IN, TRANSFER_OUT, LOAN_PAYMENTS)."`
	Merchant          string   `json:"merchant,omitempty" jsonschema:"Case-insensitive text matched against the merchant name and the transaction name."`
	Direction         string   `json:"direction,omitempty" jsonschema:"inflow (money in), outflow (money out) or any."`
	MinAmount         string   `json:"min_amount,omitempty" jsonschema:"Minimum absolute amount, as a decimal string."`
	MaxAmount         string   `json:"max_amount,omitempty" jsonschema:"Maximum absolute amount, as a decimal string."`
	Currency          string   `json:"currency,omitempty" jsonschema:"ISO currency code, such as USD."`
	Pending           *bool    `json:"pending,omitempty" jsonschema:"true for pending only, false for posted only; omit for both."`
}

var filterEnums = map[string][]any{
	"account_types": strings2any(
		string(domain.AccountTypeDepository),
		string(domain.AccountTypeCredit),
		string(domain.AccountTypeLoan),
		string(domain.AccountTypeInvestment),
		string(domain.AccountTypeOther),
	),
	"direction": strings2any("inflow", "outflow", "any"),
}

func withFilterEnums(extra map[string][]any) map[string][]any {
	merged := make(map[string][]any, len(filterEnums)+len(extra))

	for key, values := range filterEnums {
		merged[key] = values
	}

	for key, values := range extra {
		merged[key] = values
	}

	return merged
}

func (f TransactionFilter) query(now time.Time, excludeTransfersByDefault bool) (repository.TransactionQuery, error) {
	from, to, err := dateRange(f.From, f.To, now, defaultLookback)
	if err != nil {
		return repository.TransactionQuery{}, err
	}

	query := repository.TransactionQuery{
		From:              from,
		To:                to,
		Categories:        f.Categories,
		ExcludeCategories: f.ExcludeCategories,
		ExcludeTransfers:  excludeTransfersByDefault,
		Merchant:          f.Merchant,
		Currency:          f.Currency,
		Pending:           f.Pending,
	}

	if f.ExcludeTransfers != nil {
		query.ExcludeTransfers = *f.ExcludeTransfers
	}

	if len(f.Categories) > maxFilterListSize || len(f.ExcludeCategories) > maxFilterListSize || len(f.Merchant) > 200 {
		return repository.TransactionQuery{}, invalidParams("filters are too long")
	}

	if query.AccountIDs, err = parseIDs("account_ids", f.AccountIDs); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.ExcludeAccountIDs, err = parseIDs("exclude_account_ids", f.ExcludeAccountIDs); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.ConnectionIDs, err = parseIDs("connection_ids", f.ConnectionIDs); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.AccountTypes, err = parseAccountTypes(f.AccountTypes); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.Direction, err = parseDirection(f.Direction); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.MinAmount, err = parseAmount("min_amount", f.MinAmount); err != nil {
		return repository.TransactionQuery{}, err
	}

	if query.MaxAmount, err = parseAmount("max_amount", f.MaxAmount); err != nil {
		return repository.TransactionQuery{}, err
	}

	return query, nil
}

func dateRange(rawFrom, rawTo string, now time.Time, lookback time.Duration) (time.Time, time.Time, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	to := today

	if rawTo != "" {
		parsed, err := time.Parse(dateLayout, rawTo)
		if err != nil {
			return time.Time{}, time.Time{}, invalidParams("to must be a date like 2026-01-31")
		}

		to = parsed
	}

	from := to.Add(-lookback)

	if rawFrom != "" {
		parsed, err := time.Parse(dateLayout, rawFrom)
		if err != nil {
			return time.Time{}, time.Time{}, invalidParams("from must be a date like 2026-01-01")
		}

		from = parsed
	}

	if from.After(to) {
		return time.Time{}, time.Time{}, invalidParams("from must not be after to")
	}

	if to.Sub(from) > maxQuerySpan {
		return time.Time{}, time.Time{}, invalidParams("the date range may span at most 5 years")
	}

	return from, to, nil
}

func parseIDs(field string, raw []string) ([]uuid.UUID, error) {
	if len(raw) > maxFilterListSize {
		return nil, invalidParams("%s may hold at most %d IDs", field, maxFilterListSize)
	}

	ids := make([]uuid.UUID, 0, len(raw))

	for _, value := range raw {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, invalidParams("%s must contain IDs returned by fin, got %q", field, value)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func parseAccountTypes(raw []string) ([]domain.AccountType, error) {
	types := make([]domain.AccountType, 0, len(raw))

	for _, value := range raw {
		accountType := domain.AccountType(value)

		if !accountType.Valid() {
			return nil, invalidParams("unknown account type %q", value)
		}

		types = append(types, accountType)
	}

	return types, nil
}

func parseDirection(raw string) (repository.Direction, error) {
	switch raw {
	case "", "any":
		return repository.DirectionAny, nil
	case "inflow":
		return repository.DirectionInflow, nil
	case "outflow":
		return repository.DirectionOutflow, nil
	}

	return "", invalidParams("direction must be inflow, outflow or any")
}

func parseAmount(field, raw string) (*string, error) {
	if raw == "" {
		return nil, nil
	}

	if !decimalPattern.MatchString(raw) {
		return nil, invalidParams("%s must be a positive decimal such as 25 or 19.99", field)
	}

	return &raw, nil
}

func formatDate(t time.Time) string {
	return t.Format(dateLayout)
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}

	formatted := t.UTC().Format(time.RFC3339)

	return &formatted
}
