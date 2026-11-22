package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

type investmentSync struct {
	*connectionFixture
	investmentAccountID uuid.UUID
	statuses            []*domain.InvestmentsStatus
	securities          [][]domain.InvestmentSecurity
	holdingAccounts     []uuid.UUID
	holdings            []domain.InvestmentHolding
	snapshotAccounts    []uuid.UUID
	snapshotAt          time.Time
	transactionAccounts []uuid.UUID
	since               time.Time
	savedTransactions   []domain.InvestmentTransaction
	requestedStart      time.Time
	requestedEnd        time.Time
	holdingsErr         error
	transactionsErr     error
}

func newInvestmentSync(t *testing.T) *investmentSync {
	t.Helper()

	f := &investmentSync{connectionFixture: newConnectionFixture(t), investmentAccountID: uuid.New()}

	f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
		f.record("provider.accounts")

		return []provider.Account{
			{ExternalID: "acct-1", Name: "Checking", Type: domain.AccountTypeDepository},
			{ExternalID: "acct-inv", Name: "Brokerage", Type: domain.AccountTypeInvestment},
		}, nil
	}

	f.accounts.upsert = func(context.Context, uuid.UUID, uuid.UUID, []domain.Account) (map[string]uuid.UUID, error) {
		f.record("accounts.upsert")

		return map[string]uuid.UUID{"acct-1": f.accountID, "acct-inv": f.investmentAccountID}, nil
	}

	f.connections.updateInvestments = func(_ context.Context, userID, id uuid.UUID, status *domain.InvestmentsStatus) error {
		f.record("connections.updateInvestmentsStatus")

		if !f.owns(userID, id) {
			return domain.ErrNotFound
		}

		f.statuses = append(f.statuses, status)

		return nil
	}

	f.provider.investmentHoldings = func(context.Context, domain.Credential, string) (provider.InvestmentHoldingsResult, error) {
		f.record("provider.investmentHoldings")

		return provider.InvestmentHoldingsResult{
			Securities: []provider.InvestmentSecurity{{ExternalID: "sec-vti", Ticker: new("VTI")}},
			Holdings: []provider.InvestmentHolding{
				{AccountExternalID: "acct-inv", SecurityExternalID: "sec-vti", InstitutionValue: "1002", Quantity: "4"},
				{AccountExternalID: "acct-unknown", SecurityExternalID: "sec-vti", InstitutionValue: "1", Quantity: "1"},
			},
		}, f.holdingsErr
	}

	f.provider.investmentTransactions = func(_ context.Context, _ domain.Credential, _ string, start, end time.Time) (provider.InvestmentTransactionsResult, error) {
		f.record("provider.investmentTransactions")
		f.requestedStart, f.requestedEnd = start, end

		return provider.InvestmentTransactionsResult{
			Securities: []provider.InvestmentSecurity{
				{ExternalID: "sec-sold", Name: new("Sold Fund"), Ticker: new("SOLD")},
				{ExternalID: "sec-sold"},
			},
			Transactions: []provider.InvestmentTransaction{
				{ExternalID: "tx-sell", AccountExternalID: "acct-inv", SecurityExternalID: new("sec-sold"), Amount: "510.25"},
				{ExternalID: "tx-elsewhere", AccountExternalID: "acct-1", Amount: "-5"},
			},
		}, f.transactionsErr
	}

	investments := &fakeInvestments{
		upsertSecurities: func(_ context.Context, _ uuid.UUID, securities []domain.InvestmentSecurity) error {
			f.record("investments.upsertSecurities")
			f.securities = append(f.securities, securities)

			return nil
		},
		replaceHoldings: func(_ context.Context, _ uuid.UUID, accountIDs []uuid.UUID, holdings []domain.InvestmentHolding) error {
			f.record("investments.replaceHoldings")
			f.holdingAccounts, f.holdings = accountIDs, holdings

			return nil
		},
		captureSnapshot: func(_ context.Context, _ uuid.UUID, accountIDs []uuid.UUID, capturedAt time.Time) error {
			f.record("investments.captureSnapshot")
			f.snapshotAccounts, f.snapshotAt = accountIDs, capturedAt

			return nil
		},
		replaceTransactions: func(_ context.Context, _ uuid.UUID, accountIDs []uuid.UUID, since time.Time, transactions []domain.InvestmentTransaction) error {
			f.record("investments.replaceTransactions")
			f.transactionAccounts, f.since, f.savedTransactions = accountIDs, since, transactions

			return nil
		},
	}

	f.service = NewConnectionService(ConnectionRepositories{
		Credentials:  f.credentials,
		Connections:  f.connections,
		Accounts:     f.accounts,
		Transactions: f.transactions,
		Investments:  investments,
	}, f.provider, f, f.cipher, f.recorder, SyncCooldown{}, testRedirectURI)
	f.service.now = func() time.Time { return fixedNow.Add(450 * time.Millisecond) }

	return f
}

func (f *investmentSync) sync(t *testing.T) error {
	t.Helper()

	return f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerScheduled)
}

func (f *investmentSync) assertStatuses(t *testing.T, want ...*domain.InvestmentsStatus) {
	t.Helper()

	if !slices.EqualFunc(f.statuses, want, sameInvestmentsStatus) {
		t.Errorf("investments statuses = %v, want %v", show(f.statuses), show(want))
	}
}

func show(statuses []*domain.InvestmentsStatus) []string {
	shown := make([]string, 0, len(statuses))

	for _, status := range statuses {
		if status == nil {
			shown = append(shown, "<nil>")
		} else {
			shown = append(shown, string(*status))
		}
	}

	return shown
}

func TestSyncInvestmentsSavesHoldingsSnapshotAndTransactionsBeforeCursor(t *testing.T) {
	f := newInvestmentSync(t)

	if err := f.sync(t); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	investmentCalls := slices.DeleteFunc(slices.Clone(f.calls), func(call string) bool {
		return !slices.Contains([]string{
			"provider.investmentHoldings",
			"investments.upsertSecurities",
			"investments.replaceHoldings",
			"investments.captureSnapshot",
			"provider.investmentTransactions",
			"investments.replaceTransactions",
			"connections.updateInvestmentsStatus",
			"connections.updateSyncState",
		}, call)
	})

	wantCalls := []string{
		"provider.investmentHoldings",
		"investments.upsertSecurities",
		"investments.replaceHoldings",
		"investments.captureSnapshot",
		"provider.investmentTransactions",
		"investments.upsertSecurities",
		"investments.replaceTransactions",
		"connections.updateInvestmentsStatus",
		"connections.updateSyncState",
	}

	if !slices.Equal(investmentCalls, wantCalls) {
		t.Errorf("investment call order = %v, want %v", investmentCalls, wantCalls)
	}

	onlyInvestment := []uuid.UUID{f.investmentAccountID}

	for name, got := range map[string][]uuid.UUID{"holdings": f.holdingAccounts, "snapshot": f.snapshotAccounts, "transactions": f.transactionAccounts} {
		if !slices.Equal(got, onlyInvestment) {
			t.Errorf("%s accounts = %v, want only the investment account %v", name, got, onlyInvestment)
		}
	}

	if len(f.holdings) != 1 || f.holdings[0].AccountID != f.investmentAccountID || f.holdings[0].SecurityExternalID != "sec-vti" {
		t.Errorf("holdings = %+v, want the one holding of a known investment account", f.holdings)
	}

	if !f.snapshotAt.Equal(fixedNow) {
		t.Errorf("snapshot captured at %v, want sync time truncated to the second %v", f.snapshotAt, fixedNow)
	}

	wantStart := fixedNow.AddDate(-2, 0, 0)

	if !f.requestedStart.Equal(wantStart) || !f.requestedEnd.Equal(fixedNow) || !f.since.Equal(wantStart) {
		t.Errorf("transactions window = %v..%v retained since %v, want %v..%v", f.requestedStart, f.requestedEnd, f.since, wantStart, fixedNow)
	}

	if len(f.savedTransactions) != 1 || f.savedTransactions[0].ExternalID != "tx-sell" || f.savedTransactions[0].AccountID != f.investmentAccountID {
		t.Errorf("transactions = %+v, want only the investment account's transaction", f.savedTransactions)
	}

	sold := f.securities[1]

	if len(sold) != 1 || sold[0].Name == nil || *sold[0].Name != "Sold Fund" || *sold[0].Ticker != "SOLD" {
		t.Errorf("transaction securities = %+v, want sold security once with its name and ticker", sold)
	}

	f.assertStatuses(t, new(domain.InvestmentsStatusReady))
	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerScheduled, outcome: metrics.SyncOutcomeSuccess})
}

func TestSyncInvestmentsUnavailableKeepsRegularSyncSuccessful(t *testing.T) {
	tests := []struct {
		name           string
		holdingsErr    error
		transactionErr error
		status         domain.InvestmentsStatus
		wantSnapshot   bool
	}{
		{
			name:        "consent missing",
			holdingsErr: &domain.InvestmentsUnavailableError{Status: domain.InvestmentsStatusConsentRequired, Reason: "ADDITIONAL_CONSENT_REQUIRED"},
			status:      domain.InvestmentsStatusConsentRequired,
		},
		{
			name:        "product not enabled for the client",
			holdingsErr: &domain.InvestmentsUnavailableError{Status: domain.InvestmentsStatusNotEnabled, Reason: "INVALID_PRODUCT"},
			status:      domain.InvestmentsStatusNotEnabled,
		},
		{
			name:        "institution unsupported",
			holdingsErr: &domain.InvestmentsUnavailableError{Status: domain.InvestmentsStatusUnsupported, Reason: "PRODUCTS_NOT_SUPPORTED"},
			status:      domain.InvestmentsStatusUnsupported,
		},
		{
			name:           "transactions history not ready",
			transactionErr: &domain.InvestmentsUnavailableError{Status: domain.InvestmentsStatusPending, Reason: "PRODUCT_NOT_READY"},
			status:         domain.InvestmentsStatusPending,
			wantSnapshot:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newInvestmentSync(t)
			f.holdingsErr = test.holdingsErr
			f.transactionsErr = test.transactionErr

			if err := f.sync(t); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			if f.savedCursor != "cursor-2" || len(f.lastErrors) != 0 {
				t.Errorf("cursor %q last errors %v, want the regular sync saved without an error", f.savedCursor, f.lastErrors)
			}

			if f.called("investments.captureSnapshot") != test.wantSnapshot {
				t.Errorf("snapshot captured = %v, want %v", f.called("investments.captureSnapshot"), test.wantSnapshot)
			}

			if f.called("investments.replaceTransactions") {
				t.Error("stored transactions are reconciled even though the fetch did not complete")
			}

			f.assertStatuses(t, &test.status)
			f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerScheduled, outcome: metrics.SyncOutcomeSuccess})
		})
	}
}

func TestSyncInvestmentsUnexpectedFailureIsVisibleAndRetryable(t *testing.T) {
	f := newInvestmentSync(t)
	f.transactionsErr = errors.New("plaid INTERNAL_SERVER_ERROR: boom")

	err := f.sync(t)
	if err == nil {
		t.Fatal("Sync succeeded, want the investment failure")
	}

	if f.savedCursor != "" {
		t.Errorf("cursor saved as %q, want it kept so the next sync retries", f.savedCursor)
	}

	if len(f.lastErrors) != 1 {
		t.Errorf("last errors = %v, want the failure recorded", f.lastErrors)
	}

	if f.called("investments.replaceTransactions") {
		t.Error("stored transactions reconciled after a failed fetch")
	}

	f.assertStatuses(t)
	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerScheduled, outcome: metrics.SyncOutcomeError})
}

func TestSyncInvestmentsStatusIsWrittenOnlyWhenItChanges(t *testing.T) {
	t.Run("unchanged ready status", func(t *testing.T) {
		f := newInvestmentSync(t)
		f.connection.InvestmentsStatus = new(domain.InvestmentsStatusReady)

		if err := f.sync(t); err != nil {
			t.Fatalf("Sync: %v", err)
		}

		f.assertStatuses(t)
	})

	t.Run("investment accounts removed", func(t *testing.T) {
		f := newInvestmentSync(t)
		f.connection.InvestmentsStatus = new(domain.InvestmentsStatusReady)
		f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
			return []provider.Account{{ExternalID: "acct-1", Type: domain.AccountTypeDepository}}, nil
		}

		if err := f.sync(t); err != nil {
			t.Fatalf("Sync: %v", err)
		}

		if f.called("provider.investmentHoldings") {
			t.Error("investments fetched for a connection without investment accounts")
		}

		f.assertStatuses(t, nil)
	})
}

func TestCreateInvestmentsConsentLinkTokenRequiresActiveConnection(t *testing.T) {
	f := newInvestmentSync(t)

	var gotToken string

	f.provider.createInvestmentsLinkToken = func(_ context.Context, _ domain.Credential, userID, accessToken, redirectURI string) (string, error) {
		gotToken = accessToken

		if userID != f.userID.String() || redirectURI != testRedirectURI {
			t.Errorf("link token for (%q, %q), want (%q, %q)", userID, redirectURI, f.userID, testRedirectURI)
		}

		return "investments-link-token", nil
	}

	token, err := f.service.CreateInvestmentsConsentLinkToken(t.Context(), f.userID, f.connection.ID)
	if err != nil || token != "investments-link-token" || gotToken != testProviderToken {
		t.Fatalf("CreateInvestmentsConsentLinkToken = (%q, %v) with access token %q", token, err, gotToken)
	}

	f.connection.Status = domain.ConnectionStatusNeedsReauth

	if _, err := f.service.CreateInvestmentsConsentLinkToken(t.Context(), f.userID, f.connection.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("needs_reauth connection err = %v, want ErrConflict", err)
	}
}

func TestCompleteInvestmentsConsentSyncsWithItsTrigger(t *testing.T) {
	f := newInvestmentSync(t)

	connection, err := f.service.CompleteInvestmentsConsent(t.Context(), f.userID, f.connection.ID)
	if err != nil || connection.ID != f.connection.ID {
		t.Fatalf("CompleteInvestmentsConsent = (%+v, %v)", connection, err)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerInvestmentsConsent, outcome: metrics.SyncOutcomeSuccess})
}
