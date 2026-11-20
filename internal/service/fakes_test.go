package service

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/identity"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
	"github.com/tahminator/fin/internal/repository"
)

var fixedNow = time.Date(2026, time.March, 14, 9, 30, 0, 0, time.UTC)

func fixedClock() time.Time {
	return fixedNow
}

type fakeUsers struct {
	repository.UserRepository
	findByEmail  func(ctx context.Context, email string) (domain.User, error)
	findByID     func(ctx context.Context, id uuid.UUID) (domain.User, error)
	setGoogleSub func(ctx context.Context, id uuid.UUID, googleSub string, name *string) error
}

func (f *fakeUsers) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return f.findByEmail(ctx, email)
}

func (f *fakeUsers) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return f.findByID(ctx, id)
}

func (f *fakeUsers) SetGoogleSub(ctx context.Context, id uuid.UUID, googleSub string, name *string) error {
	return f.setGoogleSub(ctx, id, googleSub, name)
}

type fakeSessions struct {
	repository.SessionRepository
	create               func(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error
	findValidByTokenHash func(ctx context.Context, tokenHash []byte) (domain.Session, error)
	delete               func(ctx context.Context, tokenHash []byte) error
	deleteExpired        func(ctx context.Context) error
}

func (f *fakeSessions) Create(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	return f.create(ctx, userID, tokenHash, expiresAt)
}

func (f *fakeSessions) FindValidByTokenHash(ctx context.Context, tokenHash []byte) (domain.Session, error) {
	return f.findValidByTokenHash(ctx, tokenHash)
}

func (f *fakeSessions) Delete(ctx context.Context, tokenHash []byte) error {
	return f.delete(ctx, tokenHash)
}

func (f *fakeSessions) DeleteExpired(ctx context.Context) error {
	return f.deleteExpired(ctx)
}

type fakeIdentity struct {
	identity.IdentityProvider
	authCodeURL func(state, nonce, verifier string) string
	exchange    func(ctx context.Context, code, verifier, nonce string) (identity.Claims, error)
}

func (f *fakeIdentity) AuthCodeURL(state, nonce, verifier string) string {
	return f.authCodeURL(state, nonce, verifier)
}

func (f *fakeIdentity) Exchange(ctx context.Context, code, verifier, nonce string) (identity.Claims, error) {
	return f.exchange(ctx, code, verifier, nonce)
}

type fakeCredentials struct {
	repository.CredentialRepository
	find func(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error)
}

func (f *fakeCredentials) Find(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error) {
	return f.find(ctx, userID, provider)
}

type fakeConnections struct {
	repository.ConnectionRepository
	create              func(ctx context.Context, connection domain.Connection) (domain.Connection, error)
	get                 func(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error)
	listByStatus        func(ctx context.Context, status domain.ConnectionStatus) ([]domain.Connection, error)
	listByUserAndStatus func(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus, limit, offset int) ([]domain.Connection, int, error)
	updateSyncState     func(ctx context.Context, userID, id uuid.UUID, cursor string, syncedAt time.Time) error
	updateLastError     func(ctx context.Context, userID, id uuid.UUID, lastError string) error
	updateStatus        func(ctx context.Context, userID, id uuid.UUID, status domain.ConnectionStatus, lastError *string) error
	delete              func(ctx context.Context, userID, id uuid.UUID) error
}

func (f *fakeConnections) Create(ctx context.Context, connection domain.Connection) (domain.Connection, error) {
	return f.create(ctx, connection)
}

func (f *fakeConnections) Get(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error) {
	return f.get(ctx, userID, id)
}

func (f *fakeConnections) ListByStatus(ctx context.Context, status domain.ConnectionStatus) ([]domain.Connection, error) {
	return f.listByStatus(ctx, status)
}

func (f *fakeConnections) ListByUserAndStatus(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus, limit, offset int) ([]domain.Connection, int, error) {
	return f.listByUserAndStatus(ctx, userID, status, limit, offset)
}

func (f *fakeConnections) UpdateSyncState(ctx context.Context, userID, id uuid.UUID, cursor string, syncedAt time.Time) error {
	return f.updateSyncState(ctx, userID, id, cursor, syncedAt)
}

func (f *fakeConnections) UpdateLastError(ctx context.Context, userID, id uuid.UUID, lastError string) error {
	return f.updateLastError(ctx, userID, id, lastError)
}

func (f *fakeConnections) UpdateStatus(ctx context.Context, userID, id uuid.UUID, status domain.ConnectionStatus, lastError *string) error {
	return f.updateStatus(ctx, userID, id, status, lastError)
}

func (f *fakeConnections) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return f.delete(ctx, userID, id)
}

type fakeAccounts struct {
	repository.AccountRepository
	upsert func(ctx context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error)
}

func (f *fakeAccounts) Upsert(ctx context.Context, userID, connectionID uuid.UUID, accounts []domain.Account) (map[string]uuid.UUID, error) {
	return f.upsert(ctx, userID, connectionID, accounts)
}

type fakeTransactions struct {
	repository.TransactionRepository
	upsert              func(ctx context.Context, userID uuid.UUID, transactions []domain.Transaction) error
	deleteByExternalIDs func(ctx context.Context, userID uuid.UUID, externalIDs []string) error
}

func (f *fakeTransactions) Upsert(ctx context.Context, userID uuid.UUID, transactions []domain.Transaction) error {
	return f.upsert(ctx, userID, transactions)
}

func (f *fakeTransactions) DeleteByExternalIDs(ctx context.Context, userID uuid.UUID, externalIDs []string) error {
	return f.deleteByExternalIDs(ctx, userID, externalIDs)
}

type fakeProvider struct {
	provider.Provider
	createLinkToken       func(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error)
	createUpdateLinkToken func(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	exchange              func(ctx context.Context, creds domain.Credential, publicToken string) (provider.LinkedItem, error)
	accounts              func(ctx context.Context, creds domain.Credential, accessToken string) ([]provider.Account, error)
	syncTransactions      func(ctx context.Context, creds domain.Credential, accessToken, cursor string) (provider.SyncResult, error)
	remove                func(ctx context.Context, creds domain.Credential, accessToken string) error
}

func (f *fakeProvider) CreateLinkToken(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error) {
	return f.createLinkToken(ctx, creds, userID, redirectURI)
}

func (f *fakeProvider) CreateUpdateLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error) {
	return f.createUpdateLinkToken(ctx, creds, userID, accessToken, redirectURI)
}

func (f *fakeProvider) Exchange(ctx context.Context, creds domain.Credential, publicToken string) (provider.LinkedItem, error) {
	return f.exchange(ctx, creds, publicToken)
}

func (f *fakeProvider) Accounts(ctx context.Context, creds domain.Credential, accessToken string) ([]provider.Account, error) {
	return f.accounts(ctx, creds, accessToken)
}

func (f *fakeProvider) SyncTransactions(ctx context.Context, creds domain.Credential, accessToken, cursor string) (provider.SyncResult, error) {
	return f.syncTransactions(ctx, creds, accessToken, cursor)
}

func (f *fakeProvider) Remove(ctx context.Context, creds domain.Credential, accessToken string) error {
	return f.remove(ctx, creds, accessToken)
}

type loginRecorder struct {
	metrics.Noop
	outcomes []metrics.LoginOutcome
}

func (r *loginRecorder) LoginAttempted(outcome metrics.LoginOutcome) {
	r.outcomes = append(r.outcomes, outcome)
}

type syncedCall struct {
	trigger metrics.SyncTrigger
	outcome metrics.SyncOutcome
}

type syncRecorder struct {
	metrics.Noop
	calls []syncedCall
}

func (r *syncRecorder) ConnectionSynced(trigger metrics.SyncTrigger, outcome metrics.SyncOutcome, _ time.Duration) {
	r.calls = append(r.calls, syncedCall{trigger: trigger, outcome: outcome})
}

type scheduledRunRecorder struct {
	metrics.Noop
	runs []metrics.ScheduledRun
}

func (r *scheduledRunRecorder) ScheduledRunFinished(run metrics.ScheduledRun) {
	r.runs = append(r.runs, run)
}
