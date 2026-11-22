package mcpserver

import (
	"context"
	"slices"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type ListConnectionsInput struct{}

type ConnectionOutput struct {
	ID             string  `json:"id"`
	Institution    string  `json:"institution"`
	Status         string  `json:"status" jsonschema:"active, or needs_reauth when the person must reconnect the bank in the fin app"`
	LastSyncedAt   *string `json:"last_synced_at" jsonschema:"When fin last pulled data from this bank (RFC 3339), or null if never"`
	HasError       bool    `json:"has_error" jsonschema:"True when the last sync failed"`
	AccountCount   int     `json:"account_count"`
	ConnectedSince string  `json:"connected_since"`
}

type ListConnectionsOutput struct {
	Connections []ConnectionOutput `json:"connections"`
}

func (t *toolset) listConnections(ctx context.Context, _ ListConnectionsInput) (ListConnectionsOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	connections, _, err := t.deps.Connections.ListByUser(ctx, t.userID(), maxListedRecords, 0)
	if err != nil {
		return ListConnectionsOutput{}, err
	}

	accounts, _, err := t.deps.Accounts.ListByUser(ctx, t.userID(), repository.AccountFilter{}, maxListedRecords, 0)
	if err != nil {
		return ListConnectionsOutput{}, err
	}

	counts := map[uuid.UUID]int{}

	for _, account := range accounts {
		counts[account.ConnectionID]++
	}

	output := ListConnectionsOutput{Connections: make([]ConnectionOutput, 0, len(connections))}

	for _, connection := range connections {
		output.Connections = append(output.Connections, ConnectionOutput{
			ID:             connection.ID.String(),
			Institution:    institutionName(connection.InstitutionName),
			Status:         string(connection.Status),
			LastSyncedAt:   formatOptionalTime(connection.LastSyncedAt),
			HasError:       connection.LastError != nil,
			AccountCount:   counts[connection.ID],
			ConnectedSince: formatDate(connection.CreatedAt),
		})
	}

	return output, nil
}

type ListAccountsInput struct {
	ConnectionIDs []string `json:"connection_ids,omitempty" jsonschema:"Only accounts at these banks."`
	AccountTypes  []string `json:"account_types,omitempty" jsonschema:"Only these account types."`
}

type AccountOutput struct {
	ID               string  `json:"id"`
	ConnectionID     string  `json:"connection_id"`
	Institution      string  `json:"institution"`
	Name             string  `json:"name"`
	OfficialName     *string `json:"official_name"`
	Mask             *string `json:"mask" jsonschema:"Last digits of the account number"`
	Type             string  `json:"type"`
	Subtype          *string `json:"subtype"`
	CurrentBalance   *string `json:"current_balance" jsonschema:"Decimal string. For credit and loan accounts this is the amount owed."`
	AvailableBalance *string `json:"available_balance"`
	Currency         *string `json:"currency"`
}

type ListAccountsOutput struct {
	Accounts []AccountOutput `json:"accounts"`
}

func (t *toolset) listAccounts(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
	connectionIDs, err := parseIDs("connection_ids", input.ConnectionIDs)
	if err != nil {
		return ListAccountsOutput{}, err
	}

	types, err := parseAccountTypes(input.AccountTypes)
	if err != nil {
		return ListAccountsOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	accounts, _, err := t.deps.Accounts.ListByUser(ctx, t.userID(), repository.AccountFilter{}, maxListedRecords, 0)
	if err != nil {
		return ListAccountsOutput{}, err
	}

	output := ListAccountsOutput{Accounts: []AccountOutput{}}

	for _, account := range accounts {
		if len(connectionIDs) > 0 && !slices.Contains(connectionIDs, account.ConnectionID) {
			continue
		}

		if len(types) > 0 && !slices.Contains(types, account.Type) {
			continue
		}

		output.Accounts = append(output.Accounts, newAccountOutput(account))
	}

	return output, nil
}

func newAccountOutput(account domain.Account) AccountOutput {
	return AccountOutput{
		ID:               account.ID.String(),
		ConnectionID:     account.ConnectionID.String(),
		Institution:      institutionName(account.InstitutionName),
		Name:             account.Name,
		OfficialName:     account.OfficialName,
		Mask:             account.Mask,
		Type:             string(account.Type),
		Subtype:          account.Subtype,
		CurrentBalance:   account.CurrentBalance,
		AvailableBalance: account.AvailableBalance,
		Currency:         account.Currency,
	}
}

type GetBalancesInput struct {
	ExcludeAccountIDs []string `json:"exclude_account_ids,omitempty" jsonschema:"Leave these accounts out of the totals."`
}

type BalanceOutput struct {
	Currency    string `json:"currency"`
	Assets      string `json:"assets"`
	Liabilities string `json:"liabilities"`
	NetWorth    string `json:"net_worth"`
}

type GetBalancesOutput struct {
	Balances []BalanceOutput `json:"balances" jsonschema:"One entry per currency; never add them together."`
}

func (t *toolset) getBalances(ctx context.Context, input GetBalancesInput) (GetBalancesOutput, error) {
	exclude, err := parseIDs("exclude_account_ids", input.ExcludeAccountIDs)
	if err != nil {
		return GetBalancesOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	balances, err := t.deps.Balances.Balances(ctx, t.userID(), exclude)
	if err != nil {
		return GetBalancesOutput{}, err
	}

	output := GetBalancesOutput{Balances: make([]BalanceOutput, 0, len(balances))}

	for _, balance := range balances {
		output.Balances = append(output.Balances, BalanceOutput{
			Currency:    balance.Currency,
			Assets:      balance.Assets,
			Liabilities: balance.Liabilities,
			NetWorth:    balance.Net,
		})
	}

	return output, nil
}

func institutionName(name *string) string {
	if name == nil || *name == "" {
		return "Unknown institution"
	}

	return *name
}
