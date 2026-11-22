package provider

import (
	"context"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

type Provider interface {
	CreateLinkToken(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error)
	CreateUpdateLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	CreateAccountSelectionLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	CreateInvestmentsConsentLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	Exchange(ctx context.Context, creds domain.Credential, publicToken string) (LinkedItem, error)
	Accounts(ctx context.Context, creds domain.Credential, accessToken string) ([]Account, error)
	SyncTransactions(ctx context.Context, creds domain.Credential, accessToken, cursor string) (SyncResult, error)
	InvestmentHoldings(ctx context.Context, creds domain.Credential, accessToken string) (InvestmentHoldingsResult, error)
	InvestmentTransactions(ctx context.Context, creds domain.Credential, accessToken string, start, end time.Time) (InvestmentTransactionsResult, error)
	Remove(ctx context.Context, creds domain.Credential, accessToken string) error
	Institution(ctx context.Context, creds domain.Credential, institutionID string) (InstitutionInfo, error)
}
