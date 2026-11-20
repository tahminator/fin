package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/encryption"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

const (
	testRedirectURI   = "https://fin.example.test/connections"
	testProviderToken = "access-sandbox-0001"
)

var errProviderDown = errors.New("provider unavailable")

type statusUpdate struct {
	status    domain.ConnectionStatus
	lastError *string
}

type connectionFixture struct {
	userID      uuid.UUID
	credential  domain.Credential
	cipher      *encryption.Cipher
	connection  domain.Connection
	accountID   uuid.UUID
	syncResult  provider.SyncResult
	calls       []string
	tokens      []string
	statuses    []statusUpdate
	lastErrors  []string
	savedCursor string
	savedAt     time.Time
	upsertedTx  []domain.Transaction
	deletedTx   []string
	keptAccount []uuid.UUID

	credentials  *fakeCredentials
	connections  *fakeConnections
	accounts     *fakeAccounts
	transactions *fakeTransactions
	provider     *fakeProvider
	recorder     *syncRecorder
	service      *ConnectionService
}

func newTestCipher(t *testing.T) *encryption.Cipher {
	t.Helper()

	cipher, err := encryption.New([]byte("0123456789abcdefghijklmnopqrstuv"))
	if err != nil {
		t.Fatalf("encryption.New: %v", err)
	}

	return cipher
}

func newConnectionFixture(t *testing.T) *connectionFixture {
	t.Helper()

	f := &connectionFixture{
		userID:    uuid.New(),
		cipher:    newTestCipher(t),
		accountID: uuid.New(),
		recorder:  &syncRecorder{},
	}

	f.credential = domain.Credential{
		UserID:      f.userID,
		Provider:    domain.ProviderPlaid,
		Environment: domain.ProviderEnvironmentSandbox,
		ClientID:    "client-id",
	}

	f.connection = domain.Connection{
		ID:                    uuid.New(),
		UserID:                f.userID,
		Provider:              domain.ProviderPlaid,
		ExternalID:            "item-1",
		AccessTokenCiphertext: f.cipher.Encrypt(testProviderToken),
		Status:                domain.ConnectionStatusActive,
		SyncCursor:            "cursor-1",
	}

	f.syncResult = provider.SyncResult{
		Added:      []provider.Transaction{{ExternalID: "tx-added", AccountExternalID: "acct-1", Amount: "12.50"}},
		Modified:   []provider.Transaction{{ExternalID: "tx-modified", AccountExternalID: "acct-1", Amount: "-3.00"}},
		Removed:    []string{"tx-removed"},
		NextCursor: "cursor-2",
	}

	f.credentials = &fakeCredentials{
		find: func(_ context.Context, userID uuid.UUID, providerName domain.Provider) (domain.Credential, error) {
			f.record("credentials.find")

			if userID != f.userID || providerName != domain.ProviderPlaid {
				return domain.Credential{}, domain.ErrNotFound
			}

			return f.credential, nil
		},
	}

	f.connections = &fakeConnections{
		get: func(_ context.Context, userID, id uuid.UUID) (domain.Connection, error) {
			f.record("connections.get")

			if !f.owns(userID, id) {
				return domain.Connection{}, domain.ErrNotFound
			}

			return f.connection, nil
		},
		updateSyncState: func(_ context.Context, userID, id uuid.UUID, cursor string, syncedAt time.Time) error {
			f.record("connections.updateSyncState")

			if !f.owns(userID, id) {
				return domain.ErrNotFound
			}

			f.savedCursor = cursor
			f.savedAt = syncedAt

			return nil
		},
		updateLastError: func(_ context.Context, userID, id uuid.UUID, lastError string) error {
			f.record("connections.updateLastError")

			if !f.owns(userID, id) {
				return domain.ErrNotFound
			}

			f.lastErrors = append(f.lastErrors, lastError)

			return nil
		},
		updateStatus: func(_ context.Context, userID, id uuid.UUID, status domain.ConnectionStatus, lastError *string) error {
			f.record("connections.updateStatus")

			if !f.owns(userID, id) {
				return domain.ErrNotFound
			}

			f.statuses = append(f.statuses, statusUpdate{status: status, lastError: lastError})
			f.connection.Status = status

			return nil
		},
		delete: func(_ context.Context, userID, id uuid.UUID) error {
			f.record("connections.delete")

			if !f.owns(userID, id) {
				return domain.ErrNotFound
			}

			return nil
		},
	}

	f.accounts = &fakeAccounts{
		upsert: func(_ context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error) {
			f.record("accounts.upsert")

			if userID != f.userID || connectionID != f.connection.ID {
				t.Errorf("accounts.Upsert got (%v, %v), want (%v, %v)", userID, connectionID, f.userID, f.connection.ID)
			}

			ids := make(map[string]uuid.UUID, len(accounts))

			for _, account := range accounts {
				ids[account.ExternalID] = f.accountID
			}

			return ids, nil
		},
		deleteExcept: func(_ context.Context, userID, connectionID uuid.UUID, keepIDs []uuid.UUID) error {
			f.record("accounts.deleteExcept")

			if userID != f.userID || connectionID != f.connection.ID {
				t.Errorf("accounts.DeleteExcept got (%v, %v), want (%v, %v)", userID, connectionID, f.userID, f.connection.ID)
			}

			f.keptAccount = keepIDs

			return nil
		},
	}

	f.transactions = &fakeTransactions{
		upsert: func(_ context.Context, _ uuid.UUID, transactions []domain.Transaction) error {
			f.record("transactions.upsert")
			f.upsertedTx = transactions

			return nil
		},
		deleteByExternalIDs: func(_ context.Context, _ uuid.UUID, externalIDs []string) error {
			f.record("transactions.deleteByExternalIDs")
			f.deletedTx = externalIDs

			return nil
		},
	}

	f.provider = &fakeProvider{
		createLinkToken: func(context.Context, domain.Credential, string, string) (string, error) {
			f.record("provider.createLinkToken")

			return "link-token", nil
		},
		createUpdateLinkToken: func(_ context.Context, _ domain.Credential, _, accessToken, _ string) (string, error) {
			f.record("provider.createUpdateLinkToken")
			f.tokens = append(f.tokens, accessToken)

			return "update-link-token", nil
		},
		createAccountSelectionLinkToken: func(_ context.Context, _ domain.Credential, _, accessToken, _ string) (string, error) {
			f.record("provider.createAccountSelectionLinkToken")
			f.tokens = append(f.tokens, accessToken)

			return "account-selection-link-token", nil
		},
		accounts: func(_ context.Context, _ domain.Credential, accessToken string) ([]provider.Account, error) {
			f.record("provider.accounts")
			f.tokens = append(f.tokens, accessToken)

			return []provider.Account{{ExternalID: "acct-1", Name: "Checking", Type: domain.AccountTypeDepository}}, nil
		},
		syncTransactions: func(_ context.Context, _ domain.Credential, accessToken, _ string) (provider.SyncResult, error) {
			f.record("provider.syncTransactions")
			f.tokens = append(f.tokens, accessToken)

			return f.syncResult, nil
		},
		remove: func(_ context.Context, _ domain.Credential, accessToken string) error {
			f.record("provider.remove")
			f.tokens = append(f.tokens, accessToken)

			return nil
		},
	}

	f.service = NewConnectionService(ConnectionRepositories{
		Credentials:  f.credentials,
		Connections:  f.connections,
		Accounts:     f.accounts,
		Transactions: f.transactions,
	}, f.provider, f.cipher, f.recorder, testRedirectURI)
	f.service.now = fixedClock

	return f
}

func (f *connectionFixture) record(call string) {
	f.calls = append(f.calls, call)
}

func (f *connectionFixture) owns(userID, id uuid.UUID) bool {
	return userID == f.userID && id == f.connection.ID
}

func (f *connectionFixture) called(call string) bool {
	return slices.Contains(f.calls, call)
}

func (f *connectionFixture) providerCalls() []string {
	var calls []string

	for _, call := range f.calls {
		if strings.HasPrefix(call, "provider.") {
			calls = append(calls, call)
		}
	}

	return calls
}

func (f *connectionFixture) assertSynced(t *testing.T, want ...syncedCall) {
	t.Helper()

	if !slices.Equal(f.recorder.calls, want) {
		t.Errorf("ConnectionSynced calls = %+v, want %+v", f.recorder.calls, want)
	}
}

func TestConnectionServiceMissingCredentialsIsNotAllowed(t *testing.T) {
	tests := []struct {
		name string
		run  func(f *connectionFixture, ctx context.Context) error
	}{
		{
			name: "create link token",
			run: func(f *connectionFixture, ctx context.Context) error {
				_, err := f.service.CreateLinkToken(ctx, f.userID)

				return err
			},
		},
		{
			name: "link",
			run: func(f *connectionFixture, ctx context.Context) error {
				_, err := f.service.Link(ctx, f.userID, "public-token")

				return err
			},
		},
		{
			name: "sync",
			run: func(f *connectionFixture, ctx context.Context) error {
				return f.service.Sync(ctx, f.userID, f.connection.ID, metrics.SyncTriggerManual)
			},
		},
		{
			name: "create reauth link token",
			run: func(f *connectionFixture, ctx context.Context) error {
				f.connection.Status = domain.ConnectionStatusNeedsReauth

				_, err := f.service.CreateReauthLinkToken(ctx, f.userID, f.connection.ID)

				return err
			},
		},
		{
			name: "remove",
			run: func(f *connectionFixture, ctx context.Context) error {
				return f.service.Remove(ctx, f.userID, f.connection.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)

			f.credentials.find = func(context.Context, uuid.UUID, domain.Provider) (domain.Credential, error) {
				return domain.Credential{}, domain.ErrNotFound
			}

			err := tt.run(f, t.Context())
			if !errors.Is(err, domain.ErrNotAllowed) {
				t.Fatalf("error = %v, want %v", err, domain.ErrNotAllowed)
			}

			if calls := f.providerCalls(); len(calls) != 0 {
				t.Errorf("provider calls = %v, want none", calls)
			}

			if f.called("connections.delete") {
				t.Error("connection deleted without credentials")
			}
		})
	}
}

func TestCreateLinkTokenUsesCredentialUserAndRedirect(t *testing.T) {
	f := newConnectionFixture(t)

	var gotCredential domain.Credential
	var gotUserID, gotRedirect string

	f.provider.createLinkToken = func(_ context.Context, creds domain.Credential, userID, redirectURI string) (string, error) {
		gotCredential, gotUserID, gotRedirect = creds, userID, redirectURI

		return "link-token", nil
	}

	token, err := f.service.CreateLinkToken(t.Context(), f.userID)
	if err != nil {
		t.Fatalf("CreateLinkToken: %v", err)
	}

	if token != "link-token" {
		t.Errorf("token = %q, want link-token", token)
	}

	if gotCredential != f.credential || gotUserID != f.userID.String() || gotRedirect != testRedirectURI {
		t.Errorf("provider got (%+v, %q, %q), want (%+v, %q, %q)", gotCredential, gotUserID, gotRedirect, f.credential, f.userID.String(), testRedirectURI)
	}
}

func TestLinkStoresCiphertextThatDecryptsToProviderToken(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.AccessTokenCiphertext = ""

	var created domain.Connection
	var gotPublicToken string

	f.provider.exchange = func(_ context.Context, _ domain.Credential, publicToken string) (provider.LinkedItem, error) {
		gotPublicToken = publicToken

		return provider.LinkedItem{
			ExternalID:      "item-9",
			AccessToken:     testProviderToken,
			InstitutionID:   new("ins_1"),
			InstitutionName: new("First Bank"),
		}, nil
	}

	var gotInstitutionID string

	f.provider.institution = func(_ context.Context, _ domain.Credential, institutionID string) (provider.InstitutionInfo, error) {
		gotInstitutionID = institutionID

		return provider.InstitutionInfo{URL: new("https://firstbank.example.com")}, nil
	}

	f.connections.create = func(_ context.Context, connection domain.Connection) (domain.Connection, error) {
		f.record("connections.create")
		created = connection
		f.connection.AccessTokenCiphertext = connection.AccessTokenCiphertext

		return f.connection, nil
	}

	linked, err := f.service.Link(t.Context(), f.userID, "public-token")
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	if gotPublicToken != "public-token" {
		t.Errorf("Exchange public token = %q, want public-token", gotPublicToken)
	}

	if created.AccessTokenCiphertext == testProviderToken {
		t.Fatal("stored access token equals the plaintext provider token")
	}

	if strings.Contains(created.AccessTokenCiphertext, testProviderToken) {
		t.Error("stored ciphertext contains the plaintext provider token")
	}

	plaintext, err := f.cipher.Decrypt(created.AccessTokenCiphertext)
	if err != nil {
		t.Fatalf("Decrypt stored ciphertext: %v", err)
	}

	if plaintext != testProviderToken {
		t.Errorf("decrypted token = %q, want %q", plaintext, testProviderToken)
	}

	if created.UserID != f.userID || created.Provider != domain.ProviderPlaid || created.ExternalID != "item-9" || created.Status != domain.ConnectionStatusActive {
		t.Errorf("created connection = %+v, want user, plaid provider, item-9, active", created)
	}

	if created.InstitutionName == nil || *created.InstitutionName != "First Bank" {
		t.Errorf("institution name = %v, want First Bank", created.InstitutionName)
	}

	if gotInstitutionID != "ins_1" {
		t.Errorf("institution lookup id = %q, want ins_1", gotInstitutionID)
	}

	if created.InstitutionURL == nil || *created.InstitutionURL != "https://firstbank.example.com" {
		t.Errorf("institution url = %v, want https://firstbank.example.com", created.InstitutionURL)
	}

	if linked.ID != f.connection.ID {
		t.Errorf("linked connection = %v, want %v", linked.ID, f.connection.ID)
	}

	if !slices.Equal(f.tokens, []string{testProviderToken, testProviderToken}) {
		t.Errorf("provider tokens after link = %q, want decrypted token for accounts and transactions", f.tokens)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerLink, outcome: metrics.SyncOutcomeSuccess})
}

func TestProviderAlwaysReceivesDecryptedToken(t *testing.T) {
	tests := []struct {
		name      string
		status    domain.ConnectionStatus
		run       func(f *connectionFixture, ctx context.Context) error
		wantCalls []string
	}{
		{
			name:   "sync",
			status: domain.ConnectionStatusActive,
			run: func(f *connectionFixture, ctx context.Context) error {
				return f.service.Sync(ctx, f.userID, f.connection.ID, metrics.SyncTriggerManual)
			},
			wantCalls: []string{"provider.accounts", "provider.syncTransactions"},
		},
		{
			name:   "create reauth link token",
			status: domain.ConnectionStatusNeedsReauth,
			run: func(f *connectionFixture, ctx context.Context) error {
				_, err := f.service.CreateReauthLinkToken(ctx, f.userID, f.connection.ID)

				return err
			},
			wantCalls: []string{"provider.createUpdateLinkToken"},
		},
		{
			name:   "create account selection link token",
			status: domain.ConnectionStatusActive,
			run: func(f *connectionFixture, ctx context.Context) error {
				_, err := f.service.CreateAccountSelectionLinkToken(ctx, f.userID, f.connection.ID)

				return err
			},
			wantCalls: []string{"provider.createAccountSelectionLinkToken"},
		},
		{
			name:   "remove",
			status: domain.ConnectionStatusActive,
			run: func(f *connectionFixture, ctx context.Context) error {
				return f.service.Remove(ctx, f.userID, f.connection.ID)
			},
			wantCalls: []string{"provider.remove"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)
			f.connection.Status = tt.status

			if err := tt.run(f, t.Context()); err != nil {
				t.Fatalf("run: %v", err)
			}

			if calls := f.providerCalls(); !slices.Equal(calls, tt.wantCalls) {
				t.Fatalf("provider calls = %v, want %v", calls, tt.wantCalls)
			}

			for _, token := range f.tokens {
				if token != testProviderToken {
					t.Errorf("provider received token %q, want decrypted %q", token, testProviderToken)
				}
			}

			if len(f.tokens) != len(tt.wantCalls) {
				t.Errorf("provider received %d tokens, want %d", len(f.tokens), len(tt.wantCalls))
			}
		})
	}
}

func TestSyncWritesInOrderAndSavesCursorLast(t *testing.T) {
	f := newConnectionFixture(t)

	var gotCursor string

	f.provider.syncTransactions = func(_ context.Context, _ domain.Credential, _, cursor string) (provider.SyncResult, error) {
		f.record("provider.syncTransactions")
		gotCursor = cursor

		return provider.SyncResult{
			Added: []provider.Transaction{
				{ExternalID: "tx-added", AccountExternalID: "acct-1", Amount: "12.50", Name: "Coffee"},
			},
			Modified: []provider.Transaction{
				{ExternalID: "tx-modified", AccountExternalID: "acct-1", Amount: "-3.00", Name: "Refund"},
				{ExternalID: "tx-orphan", AccountExternalID: "acct-unknown", Amount: "1.00"},
			},
			Removed:    []string{"tx-removed-1", "tx-removed-2"},
			NextCursor: "cursor-2",
		}, nil
	}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	wantCalls := []string{
		"connections.get",
		"credentials.find",
		"provider.accounts",
		"accounts.upsert",
		"provider.syncTransactions",
		"transactions.upsert",
		"transactions.deleteByExternalIDs",
		"connections.updateSyncState",
	}

	if !slices.Equal(f.calls, wantCalls) {
		t.Errorf("call order = %v, want %v", f.calls, wantCalls)
	}

	if gotCursor != "cursor-1" {
		t.Errorf("provider cursor = %q, want stored cursor-1", gotCursor)
	}

	var upsertedIDs []string

	for _, transaction := range f.upsertedTx {
		upsertedIDs = append(upsertedIDs, transaction.ExternalID)

		if transaction.AccountID != f.accountID {
			t.Errorf("transaction %s account = %v, want %v", transaction.ExternalID, transaction.AccountID, f.accountID)
		}
	}

	if !slices.Equal(upsertedIDs, []string{"tx-added", "tx-modified"}) {
		t.Errorf("upserted transactions = %v, want added and modified for known accounts", upsertedIDs)
	}

	if !slices.Equal(f.deletedTx, []string{"tx-removed-1", "tx-removed-2"}) {
		t.Errorf("deleted transactions = %v, want removed IDs", f.deletedTx)
	}

	if f.savedCursor != "cursor-2" || !f.savedAt.Equal(fixedNow) {
		t.Errorf("saved sync state = (%q, %v), want (cursor-2, %v)", f.savedCursor, f.savedAt, fixedNow)
	}

	if len(f.statuses) != 0 || len(f.lastErrors) != 0 {
		t.Errorf("unexpected status updates %+v or last errors %v", f.statuses, f.lastErrors)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeSuccess})
}

func TestSyncFailureRecordsLastErrorWithoutCursorOrStatusChange(t *testing.T) {
	tests := []struct {
		name     string
		fail     func(f *connectionFixture)
		lastCall string
	}{
		{
			name: "provider accounts",
			fail: func(f *connectionFixture) {
				f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
					f.record("provider.accounts")

					return nil, errProviderDown
				}
			},
			lastCall: "provider.accounts",
		},
		{
			name: "account upsert",
			fail: func(f *connectionFixture) {
				f.accounts.upsert = func(context.Context, uuid.UUID, uuid.UUID, []domain.Account) (map[string]uuid.UUID, error) {
					f.record("accounts.upsert")

					return nil, errProviderDown
				}
			},
			lastCall: "accounts.upsert",
		},
		{
			name: "provider transactions",
			fail: func(f *connectionFixture) {
				f.provider.syncTransactions = func(context.Context, domain.Credential, string, string) (provider.SyncResult, error) {
					f.record("provider.syncTransactions")

					return provider.SyncResult{}, errProviderDown
				}
			},
			lastCall: "provider.syncTransactions",
		},
		{
			name: "transaction upsert",
			fail: func(f *connectionFixture) {
				f.transactions.upsert = func(context.Context, uuid.UUID, []domain.Transaction) error {
					f.record("transactions.upsert")

					return errProviderDown
				}
			},
			lastCall: "transactions.upsert",
		},
		{
			name: "removed transaction delete",
			fail: func(f *connectionFixture) {
				f.transactions.deleteByExternalIDs = func(context.Context, uuid.UUID, []string) error {
					f.record("transactions.deleteByExternalIDs")

					return errProviderDown
				}
			},
			lastCall: "transactions.deleteByExternalIDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)

			tt.fail(f)

			err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerScheduled)
			if !errors.Is(err, errProviderDown) {
				t.Fatalf("Sync error = %v, want %v", err, errProviderDown)
			}

			if f.called("connections.updateSyncState") {
				t.Error("cursor saved after a failed sync")
			}

			if f.called("connections.updateStatus") {
				t.Errorf("status changed after a non-reauth failure: %+v", f.statuses)
			}

			if len(f.lastErrors) != 1 || !strings.Contains(f.lastErrors[0], errProviderDown.Error()) {
				t.Errorf("last errors = %q, want one containing %q", f.lastErrors, errProviderDown)
			}

			failedAt := slices.Index(f.calls, tt.lastCall)
			if failedAt < 0 || !slices.Equal(f.calls[failedAt+1:], []string{"connections.updateLastError"}) {
				t.Errorf("calls = %v, want %s followed only by connections.updateLastError", f.calls, tt.lastCall)
			}

			f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerScheduled, outcome: metrics.SyncOutcomeError})
		})
	}
}

func TestSyncReauthRequiredMarksNeedsReauthAndKeepsCursor(t *testing.T) {
	tests := []struct {
		name string
		fail func(f *connectionFixture)
	}{
		{
			name: "accounts require reauth",
			fail: func(f *connectionFixture) {
				f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
					f.record("provider.accounts")

					return nil, fmt.Errorf("plaid ITEM_LOGIN_REQUIRED: %w", domain.ErrReauthRequired)
				}
			},
		},
		{
			name: "transactions require reauth",
			fail: func(f *connectionFixture) {
				f.provider.syncTransactions = func(context.Context, domain.Credential, string, string) (provider.SyncResult, error) {
					f.record("provider.syncTransactions")

					return provider.SyncResult{}, fmt.Errorf("plaid ITEM_LOGIN_REQUIRED: %w", domain.ErrReauthRequired)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)

			tt.fail(f)

			err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual)
			if !errors.Is(err, domain.ErrReauthRequired) {
				t.Fatalf("Sync error = %v, want %v", err, domain.ErrReauthRequired)
			}

			if len(f.statuses) != 1 || f.statuses[0].status != domain.ConnectionStatusNeedsReauth {
				t.Fatalf("status updates = %+v, want one needs_reauth", f.statuses)
			}

			if f.statuses[0].lastError == nil || !strings.Contains(*f.statuses[0].lastError, "ITEM_LOGIN_REQUIRED") {
				t.Errorf("needs_reauth last error = %v, want provider message", f.statuses[0].lastError)
			}

			if f.called("connections.updateSyncState") {
				t.Error("cursor saved after reauth required")
			}

			if f.called("connections.updateLastError") {
				t.Error("UpdateLastError called for reauth required")
			}

			f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeReauthRequired})
		})
	}
}

func TestEverySyncOutcomeIsRecordedWithItsTrigger(t *testing.T) {
	succeed := func(*connectionFixture) {}

	requireReauth := func(f *connectionFixture) {
		f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
			return nil, domain.ErrReauthRequired
		}
	}

	failProvider := func(f *connectionFixture) {
		f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
			return nil, errProviderDown
		}
	}

	runs := []struct {
		name    string
		trigger metrics.SyncTrigger
		run     func(f *connectionFixture, ctx context.Context)
	}{
		{
			name:    "link",
			trigger: metrics.SyncTriggerLink,
			run: func(f *connectionFixture, ctx context.Context) {
				f.provider.exchange = func(context.Context, domain.Credential, string) (provider.LinkedItem, error) {
					return provider.LinkedItem{ExternalID: "item-1", AccessToken: testProviderToken}, nil
				}
				f.connections.create = func(_ context.Context, connection domain.Connection) (domain.Connection, error) {
					f.connection.AccessTokenCiphertext = connection.AccessTokenCiphertext

					return f.connection, nil
				}

				_, _ = f.service.Link(ctx, f.userID, "public-token")
			},
		},
		{
			name:    "sync one",
			trigger: metrics.SyncTriggerManual,
			run: func(f *connectionFixture, ctx context.Context) {
				_, _ = f.service.SyncOne(ctx, f.userID, f.connection.ID)
			},
		},
		{
			name:    "sync with scheduled trigger",
			trigger: metrics.SyncTriggerScheduled,
			run: func(f *connectionFixture, ctx context.Context) {
				_ = f.service.Sync(ctx, f.userID, f.connection.ID, metrics.SyncTriggerScheduled)
			},
		},
		{
			name:    "complete reauth",
			trigger: metrics.SyncTriggerReauth,
			run: func(f *connectionFixture, ctx context.Context) {
				f.connection.Status = domain.ConnectionStatusNeedsReauth

				_, _ = f.service.CompleteReauth(ctx, f.userID, f.connection.ID)
			},
		},
		{
			name:    "complete account selection",
			trigger: metrics.SyncTriggerAccountSelection,
			run: func(f *connectionFixture, ctx context.Context) {
				_, _ = f.service.CompleteAccountSelection(ctx, f.userID, f.connection.ID)
			},
		},
		{
			name:    "sync all",
			trigger: metrics.SyncTriggerManual,
			run: func(f *connectionFixture, ctx context.Context) {
				f.connections.listByUserAndStatus = func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
					return []domain.Connection{f.connection}, 1, nil
				}

				_, _ = f.service.SyncAll(ctx, f.userID, metrics.SyncTriggerManual)
			},
		},
	}

	outcomes := []struct {
		name    string
		setup   func(f *connectionFixture)
		outcome metrics.SyncOutcome
	}{
		{name: "success", setup: succeed, outcome: metrics.SyncOutcomeSuccess},
		{name: "reauth required", setup: requireReauth, outcome: metrics.SyncOutcomeReauthRequired},
		{name: "error", setup: failProvider, outcome: metrics.SyncOutcomeError},
	}

	for _, run := range runs {
		for _, outcome := range outcomes {
			t.Run(run.name+" "+outcome.name, func(t *testing.T) {
				f := newConnectionFixture(t)

				outcome.setup(f)
				run.run(f, t.Context())

				f.assertSynced(t, syncedCall{trigger: run.trigger, outcome: outcome.outcome})
			})
		}
	}
}

func TestSyncOfMissingConnectionRecordsErrorOutcome(t *testing.T) {
	f := newConnectionFixture(t)

	err := f.service.Sync(t.Context(), f.userID, uuid.New(), metrics.SyncTriggerManual)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Sync error = %v, want %v", err, domain.ErrNotFound)
	}

	if f.called("connections.updateLastError") || f.called("connections.updateStatus") {
		t.Errorf("missing connection was updated: calls %v", f.calls)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeError})
}

func TestSyncAllCoversActiveConnectionsAndCountsOutcomes(t *testing.T) {
	f := newConnectionFixture(t)

	type listCall struct {
		userID uuid.UUID
		status domain.ConnectionStatus
		limit  int
		offset int
	}

	var listCalls []listCall

	behaviors := map[string]error{
		"token-ok-1":   nil,
		"token-reauth": fmt.Errorf("item login required: %w", domain.ErrReauthRequired),
		"token-broken": errProviderDown,
		"token-ok-2":   nil,
	}

	connections := make(map[uuid.UUID]domain.Connection)

	var listed []domain.Connection

	for _, token := range []string{"token-ok-1", "token-reauth", "token-broken", "token-ok-2"} {
		connection := domain.Connection{
			ID:                    uuid.New(),
			UserID:                f.userID,
			AccessTokenCiphertext: f.cipher.Encrypt(token),
			Status:                domain.ConnectionStatusActive,
		}

		connections[connection.ID] = connection
		listed = append(listed, connection)
	}

	var synced []uuid.UUID

	f.connections.listByUserAndStatus = func(_ context.Context, userID uuid.UUID, status domain.ConnectionStatus, limit, offset int) ([]domain.Connection, int, error) {
		listCalls = append(listCalls, listCall{userID: userID, status: status, limit: limit, offset: offset})

		return listed, len(listed), nil
	}

	f.connections.get = func(_ context.Context, userID, id uuid.UUID) (domain.Connection, error) {
		connection, ok := connections[id]
		if !ok || userID != f.userID {
			return domain.Connection{}, domain.ErrNotFound
		}

		synced = append(synced, id)

		return connection, nil
	}

	f.connections.updateSyncState = func(context.Context, uuid.UUID, uuid.UUID, string, time.Time) error {
		return nil
	}

	f.connections.updateStatus = func(context.Context, uuid.UUID, uuid.UUID, domain.ConnectionStatus, *string) error {
		return nil
	}

	f.connections.updateLastError = func(context.Context, uuid.UUID, uuid.UUID, string) error {
		return nil
	}

	f.accounts.upsert = func(context.Context, uuid.UUID, uuid.UUID, []domain.Account) (map[string]uuid.UUID, error) {
		return map[string]uuid.UUID{}, nil
	}

	f.provider.accounts = func(_ context.Context, _ domain.Credential, accessToken string) ([]provider.Account, error) {
		err, ok := behaviors[accessToken]
		if !ok {
			t.Errorf("provider received unexpected token %q", accessToken)
		}

		return nil, err
	}

	summary, err := f.service.SyncAll(t.Context(), f.userID, metrics.SyncTriggerManual)
	if err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	wantList := []listCall{{userID: f.userID, status: domain.ConnectionStatusActive, limit: syncAllLimit, offset: 0}}

	if !slices.Equal(listCalls, wantList) {
		t.Errorf("ListByUserAndStatus calls = %+v, want %+v", listCalls, wantList)
	}

	wantSynced := []uuid.UUID{listed[0].ID, listed[1].ID, listed[2].ID, listed[3].ID}

	if !slices.Equal(synced, wantSynced) {
		t.Errorf("synced connections = %v, want %v", synced, wantSynced)
	}

	if want := (SyncSummary{Synced: 2, ReauthRequired: 1, Failed: 1}); summary != want {
		t.Errorf("summary = %+v, want %+v", summary, want)
	}

	f.assertSynced(t,
		syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeSuccess},
		syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeReauthRequired},
		syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeError},
		syncedCall{trigger: metrics.SyncTriggerManual, outcome: metrics.SyncOutcomeSuccess},
	)
}

func TestSyncAllListFailureReturnsError(t *testing.T) {
	f := newConnectionFixture(t)

	f.connections.listByUserAndStatus = func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
		return nil, 0, errProviderDown
	}

	summary, err := f.service.SyncAll(t.Context(), f.userID, metrics.SyncTriggerManual)
	if !errors.Is(err, errProviderDown) {
		t.Fatalf("SyncAll error = %v, want %v", err, errProviderDown)
	}

	if summary != (SyncSummary{}) {
		t.Errorf("summary = %+v, want zero", summary)
	}

	f.assertSynced(t)
}

func TestAnotherUsersConnectionIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		run  func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error
	}{
		{
			name: "sync",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				return f.service.Sync(ctx, userID, f.connection.ID, metrics.SyncTriggerManual)
			},
		},
		{
			name: "sync one",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				_, err := f.service.SyncOne(ctx, userID, f.connection.ID)

				return err
			},
		},
		{
			name: "create reauth link token",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				_, err := f.service.CreateReauthLinkToken(ctx, userID, f.connection.ID)

				return err
			},
		},
		{
			name: "complete reauth",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				_, err := f.service.CompleteReauth(ctx, userID, f.connection.ID)

				return err
			},
		},
		{
			name: "create account selection link token",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				_, err := f.service.CreateAccountSelectionLinkToken(ctx, userID, f.connection.ID)

				return err
			},
		},
		{
			name: "complete account selection",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				_, err := f.service.CompleteAccountSelection(ctx, userID, f.connection.ID)

				return err
			},
		},
		{
			name: "remove",
			run: func(f *connectionFixture, ctx context.Context, userID uuid.UUID) error {
				return f.service.Remove(ctx, userID, f.connection.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)
			f.connection.Status = domain.ConnectionStatusNeedsReauth

			err := tt.run(f, t.Context(), uuid.New())
			if !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("error = %v, want %v", err, domain.ErrNotFound)
			}

			if calls := f.providerCalls(); len(calls) != 0 {
				t.Errorf("provider calls = %v, want none", calls)
			}

			if len(f.statuses) != 0 || len(f.lastErrors) != 0 || f.savedCursor != "" {
				t.Errorf("owner's connection was modified: statuses %+v, last errors %v, cursor %q", f.statuses, f.lastErrors, f.savedCursor)
			}
		})
	}
}

func TestCreateReauthLinkTokenRequiresNeedsReauth(t *testing.T) {
	tests := []struct {
		name      string
		status    domain.ConnectionStatus
		wantErr   error
		wantToken string
	}{
		{name: "active connection conflicts", status: domain.ConnectionStatusActive, wantErr: domain.ErrConflict},
		{name: "needs reauth connection gets update token", status: domain.ConnectionStatusNeedsReauth, wantToken: "update-link-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)
			f.connection.Status = tt.status

			var gotUserID, gotRedirect string

			f.provider.createUpdateLinkToken = func(_ context.Context, _ domain.Credential, userID, accessToken, redirectURI string) (string, error) {
				f.record("provider.createUpdateLinkToken")
				f.tokens = append(f.tokens, accessToken)
				gotUserID, gotRedirect = userID, redirectURI

				return "update-link-token", nil
			}

			token, err := f.service.CreateReauthLinkToken(t.Context(), f.userID, f.connection.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if token != tt.wantToken {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}

			if tt.wantErr != nil {
				if calls := f.providerCalls(); len(calls) != 0 {
					t.Errorf("provider calls = %v, want none", calls)
				}

				return
			}

			if gotUserID != f.userID.String() || gotRedirect != testRedirectURI {
				t.Errorf("provider got (%q, %q), want (%q, %q)", gotUserID, gotRedirect, f.userID.String(), testRedirectURI)
			}

			if !slices.Equal(f.tokens, []string{testProviderToken}) {
				t.Errorf("provider tokens = %q, want decrypted token", f.tokens)
			}
		})
	}
}

func TestCreateAccountSelectionLinkTokenRequiresActive(t *testing.T) {
	tests := []struct {
		name      string
		status    domain.ConnectionStatus
		wantErr   error
		wantToken string
	}{
		{name: "needs reauth connection conflicts", status: domain.ConnectionStatusNeedsReauth, wantErr: domain.ErrConflict},
		{name: "active connection gets account selection token", status: domain.ConnectionStatusActive, wantToken: "account-selection-link-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConnectionFixture(t)
			f.connection.Status = tt.status

			var gotUserID, gotRedirect string

			f.provider.createAccountSelectionLinkToken = func(_ context.Context, _ domain.Credential, userID, accessToken, redirectURI string) (string, error) {
				f.record("provider.createAccountSelectionLinkToken")
				f.tokens = append(f.tokens, accessToken)
				gotUserID, gotRedirect = userID, redirectURI

				return "account-selection-link-token", nil
			}

			token, err := f.service.CreateAccountSelectionLinkToken(t.Context(), f.userID, f.connection.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if token != tt.wantToken {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}

			if tt.wantErr != nil {
				if calls := f.providerCalls(); len(calls) != 0 {
					t.Errorf("provider calls = %v, want none", calls)
				}

				return
			}

			if gotUserID != f.userID.String() || gotRedirect != testRedirectURI {
				t.Errorf("provider got (%q, %q), want (%q, %q)", gotUserID, gotRedirect, f.userID.String(), testRedirectURI)
			}
		})
	}
}

func TestCompleteAccountSelectionKeepsOnlySelectedAccounts(t *testing.T) {
	f := newConnectionFixture(t)

	selected := map[string]uuid.UUID{"acct-1": uuid.New(), "acct-2": uuid.New()}

	f.provider.accounts = func(context.Context, domain.Credential, string) ([]provider.Account, error) {
		f.record("provider.accounts")

		return []provider.Account{
			{ExternalID: "acct-1", Name: "Checking", Type: domain.AccountTypeDepository},
			{ExternalID: "acct-2", Name: "Savings", Type: domain.AccountTypeDepository},
		}, nil
	}

	f.accounts.upsert = func(context.Context, uuid.UUID, uuid.UUID, []domain.Account) (map[string]uuid.UUID, error) {
		f.record("accounts.upsert")

		return selected, nil
	}

	connection, err := f.service.CompleteAccountSelection(t.Context(), f.userID, f.connection.ID)
	if err != nil {
		t.Fatalf("CompleteAccountSelection: %v", err)
	}

	wantCalls := []string{
		"connections.get",
		"credentials.find",
		"provider.accounts",
		"accounts.upsert",
		"accounts.deleteExcept",
		"provider.syncTransactions",
		"transactions.upsert",
		"transactions.deleteByExternalIDs",
		"connections.updateSyncState",
		"connections.get",
	}

	if !slices.Equal(f.calls, wantCalls) {
		t.Errorf("call order = %v, want %v", f.calls, wantCalls)
	}

	if len(f.keptAccount) != len(selected) {
		t.Fatalf("kept accounts = %v, want the %d selected accounts", f.keptAccount, len(selected))
	}

	for externalID, id := range selected {
		if !slices.Contains(f.keptAccount, id) {
			t.Errorf("selected account %s (%v) not kept: %v", externalID, id, f.keptAccount)
		}
	}

	if connection.ID != f.connection.ID {
		t.Errorf("returned connection = %v, want %v", connection.ID, f.connection.ID)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerAccountSelection, outcome: metrics.SyncOutcomeSuccess})
}

func TestCompleteAccountSelectionDeleteFailureKeepsCursor(t *testing.T) {
	f := newConnectionFixture(t)

	f.accounts.deleteExcept = func(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error {
		f.record("accounts.deleteExcept")

		return errProviderDown
	}

	if _, err := f.service.CompleteAccountSelection(t.Context(), f.userID, f.connection.ID); err != nil {
		t.Fatalf("CompleteAccountSelection: %v", err)
	}

	if f.called("provider.syncTransactions") || f.savedCursor != "" {
		t.Errorf("transactions synced after account delete failed: calls %v, cursor %q", f.calls, f.savedCursor)
	}

	if len(f.lastErrors) != 1 || !strings.Contains(f.lastErrors[0], errProviderDown.Error()) {
		t.Errorf("last errors = %v, want the delete failure", f.lastErrors)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerAccountSelection, outcome: metrics.SyncOutcomeError})
}

func TestCompleteReauthActivatesThenSyncs(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.Status = domain.ConnectionStatusNeedsReauth

	connection, err := f.service.CompleteReauth(t.Context(), f.userID, f.connection.ID)
	if err != nil {
		t.Fatalf("CompleteReauth: %v", err)
	}

	wantStatuses := []statusUpdate{{status: domain.ConnectionStatusActive, lastError: nil}}

	if !slices.Equal(f.statuses, wantStatuses) {
		t.Errorf("status updates = %+v, want %+v", f.statuses, wantStatuses)
	}

	activatedAt := slices.Index(f.calls, "connections.updateStatus")
	syncedAt := slices.Index(f.calls, "provider.accounts")

	if activatedAt != 0 || syncedAt <= activatedAt {
		t.Errorf("calls = %v, want status update first and sync after", f.calls)
	}

	if !f.called("connections.updateSyncState") {
		t.Error("cursor not saved after reauth sync")
	}

	if connection.Status != domain.ConnectionStatusActive || connection.ID != f.connection.ID {
		t.Errorf("returned connection = %+v, want reloaded active connection", connection)
	}

	f.assertSynced(t, syncedCall{trigger: metrics.SyncTriggerReauth, outcome: metrics.SyncOutcomeSuccess})
}

func TestCompleteReauthStatusFailureSkipsSync(t *testing.T) {
	f := newConnectionFixture(t)

	f.connections.updateStatus = func(context.Context, uuid.UUID, uuid.UUID, domain.ConnectionStatus, *string) error {
		f.record("connections.updateStatus")

		return errProviderDown
	}

	if _, err := f.service.CompleteReauth(t.Context(), f.userID, f.connection.ID); !errors.Is(err, errProviderDown) {
		t.Fatalf("CompleteReauth error = %v, want %v", err, errProviderDown)
	}

	if !slices.Equal(f.calls, []string{"connections.updateStatus"}) {
		t.Errorf("calls = %v, want only the status update", f.calls)
	}

	f.assertSynced(t)
}

func TestRemoveCallsProviderBeforeDeleting(t *testing.T) {
	f := newConnectionFixture(t)

	if err := f.service.Remove(t.Context(), f.userID, f.connection.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	removedAt := slices.Index(f.calls, "provider.remove")
	deletedAt := slices.Index(f.calls, "connections.delete")

	if removedAt < 0 || deletedAt < 0 || removedAt > deletedAt {
		t.Errorf("calls = %v, want provider.remove before connections.delete", f.calls)
	}
}

func TestRemoveProviderFailureKeepsConnection(t *testing.T) {
	f := newConnectionFixture(t)

	f.provider.remove = func(context.Context, domain.Credential, string) error {
		f.record("provider.remove")

		return errProviderDown
	}

	if err := f.service.Remove(t.Context(), f.userID, f.connection.ID); !errors.Is(err, errProviderDown) {
		t.Fatalf("Remove error = %v, want %v", err, errProviderDown)
	}

	if f.called("connections.delete") {
		t.Error("connection deleted after provider removal failed")
	}
}

func TestLinkStoresCiphertextThatDecryptsToProviderToken_InstitutionLookupFailureDoesNotFailLink(t *testing.T) {
	f := newConnectionFixture(t)

	f.provider.exchange = func(context.Context, domain.Credential, string) (provider.LinkedItem, error) {
		return provider.LinkedItem{
			ExternalID:    "item-9",
			AccessToken:   testProviderToken,
			InstitutionID: new("ins_1"),
		}, nil
	}

	f.provider.institution = func(context.Context, domain.Credential, string) (provider.InstitutionInfo, error) {
		return provider.InstitutionInfo{}, errProviderDown
	}

	var created domain.Connection

	f.connections.create = func(_ context.Context, connection domain.Connection) (domain.Connection, error) {
		created = connection
		f.connection.AccessTokenCiphertext = connection.AccessTokenCiphertext

		return f.connection, nil
	}

	if _, err := f.service.Link(t.Context(), f.userID, "public-token"); err != nil {
		t.Fatalf("Link: %v", err)
	}

	if created.InstitutionURL != nil {
		t.Errorf("institution url = %v, want nil after lookup failure", created.InstitutionURL)
	}
}

func TestSyncBackfillsInstitutionURLWhenMissing(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.InstitutionID = new("ins_1")
	f.connection.InstitutionURL = nil

	var gotInstitutionID string
	var savedUserID, savedConnectionID uuid.UUID
	var savedURL string

	f.provider.institution = func(_ context.Context, _ domain.Credential, institutionID string) (provider.InstitutionInfo, error) {
		f.record("provider.institution")
		gotInstitutionID = institutionID

		return provider.InstitutionInfo{URL: new("https://firstbank.example.com")}, nil
	}

	f.connections.updateInstitutionURL = func(_ context.Context, userID, id uuid.UUID, url string) error {
		f.record("connections.updateInstitutionURL")
		savedUserID = userID
		savedConnectionID = id
		savedURL = url

		return nil
	}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if gotInstitutionID != "ins_1" {
		t.Errorf("institution lookup id = %q, want ins_1", gotInstitutionID)
	}

	if savedUserID != f.userID || savedConnectionID != f.connection.ID || savedURL != "https://firstbank.example.com" {
		t.Errorf("saved institution url = (%v, %v, %q), want (%v, %v, https://firstbank.example.com)", savedUserID, savedConnectionID, savedURL, f.userID, f.connection.ID)
	}
}

func TestSyncSkipsInstitutionURLLookupWhenAlreadyPresent(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.InstitutionID = new("ins_1")
	f.connection.InstitutionURL = new("https://firstbank.example.com")

	f.provider.institution = func(context.Context, domain.Credential, string) (provider.InstitutionInfo, error) {
		f.record("provider.institution")

		return provider.InstitutionInfo{}, nil
	}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if f.called("provider.institution") {
		t.Error("institution lookup performed even though institution_url was already set")
	}
}

func TestSyncSkipsInstitutionURLLookupWhenInstitutionIDMissing(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.InstitutionID = nil
	f.connection.InstitutionURL = nil

	f.provider.institution = func(context.Context, domain.Credential, string) (provider.InstitutionInfo, error) {
		f.record("provider.institution")

		return provider.InstitutionInfo{}, nil
	}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if f.called("provider.institution") {
		t.Error("institution lookup performed even though institution_id was missing")
	}
}

func TestSyncBackfillInstitutionURLSaveFailureDoesNotFailSync(t *testing.T) {
	f := newConnectionFixture(t)
	f.connection.InstitutionID = new("ins_1")
	f.connection.InstitutionURL = nil

	f.provider.institution = func(context.Context, domain.Credential, string) (provider.InstitutionInfo, error) {
		return provider.InstitutionInfo{URL: new("https://firstbank.example.com")}, nil
	}

	f.connections.updateInstitutionURL = func(context.Context, uuid.UUID, uuid.UUID, string) error {
		return errProviderDown
	}

	if err := f.service.Sync(t.Context(), f.userID, f.connection.ID, metrics.SyncTriggerManual); err != nil {
		t.Fatalf("Sync: %v, want nil since backfill is best-effort", err)
	}

	if len(f.statuses) != 0 || len(f.lastErrors) != 0 {
		t.Errorf("unexpected status updates %+v or last errors %v after backfill save failure", f.statuses, f.lastErrors)
	}
}
