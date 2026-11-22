package plaid

import (
	"context"
	"errors"
	"fmt"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"
	"github.com/plaid/plaid-go/v50/client"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

const (
	syncPageSize      = 500
	maxSyncRestarts   = 5
	mutationErrorCode = "TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION"
	initialUpdateWait = 20 * time.Second
	initialUpdatePoll = 2 * time.Second
)

func (p *Provider) Accounts(ctx context.Context, creds domain.Credential, accessToken string) ([]provider.Account, error) {
	started := time.Now()

	accounts, err := p.accounts(ctx, creds, accessToken)

	p.observe(metrics.OperationAccounts, started, err)

	return accounts, err
}

func (p *Provider) SyncTransactions(ctx context.Context, creds domain.Credential, accessToken, cursor string) (provider.SyncResult, error) {
	started := time.Now()

	result, err := p.syncTransactions(ctx, creds, accessToken, cursor)

	p.observe(metrics.OperationSyncTransactions, started, err)

	return result, err
}

func (p *Provider) accounts(ctx context.Context, creds domain.Credential, accessToken string) ([]provider.Account, error) {
	c, err := p.client(creds)
	if err != nil {
		return nil, err
	}

	response, err := c.Accounts.Get(ctx, &plaidsdk.AccountsGetRequest{AccessToken: accessToken})
	if err != nil {
		return nil, mapError(err)
	}

	accounts := make([]provider.Account, 0, len(response.Accounts))

	for _, account := range response.Accounts {
		accounts = append(accounts, mapAccount(account))
	}

	return accounts, nil
}

func (p *Provider) syncTransactions(ctx context.Context, creds domain.Credential, accessToken, cursor string) (provider.SyncResult, error) {
	c, err := p.client(creds)
	if err != nil {
		return provider.SyncResult{}, err
	}

	deadline := time.Now().Add(initialUpdateWait)

	for {
		result, err := syncWithRestarts(ctx, c, accessToken, cursor)
		if err != nil || cursor != "" || result.NextCursor != "" || time.Now().After(deadline) {
			return result, err
		}

		if err := pause(ctx, initialUpdatePoll); err != nil {
			return provider.SyncResult{}, err
		}
	}
}

func syncWithRestarts(ctx context.Context, c *client.Client, accessToken, cursor string) (provider.SyncResult, error) {
	for range maxSyncRestarts {
		result, err := syncPages(ctx, c, accessToken, cursor)
		if isMutationDuringPagination(err) {
			continue
		}

		if err != nil {
			return provider.SyncResult{}, mapError(err)
		}

		return result, nil
	}

	return provider.SyncResult{}, fmt.Errorf("transactions sync restarted %d times after mutations during pagination", maxSyncRestarts)
}

func syncPages(ctx context.Context, c *client.Client, accessToken, cursor string) (provider.SyncResult, error) {
	result := provider.SyncResult{NextCursor: cursor}

	for {
		page, err := c.Transactions.Sync(ctx, &plaidsdk.TransactionsSyncRequest{
			AccessToken: accessToken,
			Cursor:      &result.NextCursor,
			Count:       new(syncPageSize),
		})
		if err != nil {
			return provider.SyncResult{}, err
		}

		result.Added = append(result.Added, mapTransactions(page.Added)...)
		result.Modified = append(result.Modified, mapTransactions(page.Modified)...)

		for _, removed := range page.Removed {
			result.Removed = append(result.Removed, removed.TransactionID)
		}

		result.NextCursor = page.NextCursor

		if !page.HasMore {
			return result, nil
		}
	}
}

func pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isMutationDuringPagination(err error) bool {
	var plaidErr plaidError

	return errors.As(err, &plaidErr) && plaidErr.GetBody() != nil && plaidErr.GetBody().ErrorCode == mutationErrorCode
}
