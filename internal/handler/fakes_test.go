package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/encryption"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

const sessionToken = "token"

type registrar interface {
	Register(api huma.API)
}

func newAPI(t *testing.T, user domain.User, handlers ...registrar) humatest.TestAPI {
	t.Helper()

	_, api := humatest.New(t)

	grp := huma.NewGroup(api, "/api")
	grp.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		next(handler.WithSession(ctx, user, sessionToken))
	})

	for _, h := range handlers {
		h.Register(grp)
	}

	return api
}

func newUser() domain.User {
	return domain.User{ID: uuid.New(), Email: "ada@example.com"}
}

func newCipher(t *testing.T) *encryption.Cipher {
	t.Helper()

	cipher, err := encryption.New([]byte("0123456789abcdefghijklmnopqrstuv"))
	if err != nil {
		t.Fatalf("encryption.New: %v", err)
	}

	return cipher
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T

	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}

	return out
}

type fakeConnections struct {
	repository.ConnectionRepository
	get        func(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error)
	listByUser func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Connection, int, error)
	delete     func(ctx context.Context, userID, id uuid.UUID) error
}

func (f *fakeConnections) Get(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error) {
	return f.get(ctx, userID, id)
}

func (f *fakeConnections) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Connection, int, error) {
	return f.listByUser(ctx, userID, limit, offset)
}

func (f *fakeConnections) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return f.delete(ctx, userID, id)
}

func ownedConnections(connections ...domain.Connection) func(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error) {
	return func(_ context.Context, userID, id uuid.UUID) (domain.Connection, error) {
		for _, connection := range connections {
			if connection.ID == id && connection.UserID == userID {
				return connection, nil
			}
		}

		return domain.Connection{}, domain.ErrNotFound
	}
}

type fakeCredentials struct {
	repository.CredentialRepository
	find func(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error)
}

func (f *fakeCredentials) Find(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error) {
	return f.find(ctx, userID, provider)
}

func plaidCredentials() *fakeCredentials {
	return &fakeCredentials{
		find: func(_ context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error) {
			return domain.Credential{UserID: userID, Provider: provider, ClientID: "client"}, nil
		},
	}
}

type fakeProvider struct {
	provider.Provider
	createLinkToken                 func(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error)
	createUpdateLinkToken           func(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string, consentInvestments bool) (string, error)
	createAccountSelectionLinkToken func(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	remove                          func(ctx context.Context, creds domain.Credential, accessToken string) error
}

func (f *fakeProvider) CreateLinkToken(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error) {
	return f.createLinkToken(ctx, creds, userID, redirectURI)
}

func (f *fakeProvider) CreateUpdateLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string, consentInvestments bool) (string, error) {
	return f.createUpdateLinkToken(ctx, creds, userID, accessToken, redirectURI, consentInvestments)
}

func (f *fakeProvider) CreateAccountSelectionLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error) {
	return f.createAccountSelectionLinkToken(ctx, creds, userID, accessToken, redirectURI)
}

func (f *fakeProvider) Remove(ctx context.Context, creds domain.Credential, accessToken string) error {
	return f.remove(ctx, creds, accessToken)
}

func newConnectionHandler(
	cipher *encryption.Cipher,
	connections repository.ConnectionRepository,
	credentials repository.CredentialRepository,
	accountProvider provider.Provider,
) *handler.ConnectionHandler {
	accounts := &fakeAccounts{
		listByUser: func(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
			return nil, 0, nil
		},
	}

	connectionService := service.NewConnectionService(
		service.ConnectionRepositories{Credentials: credentials, Connections: connections, Accounts: accounts},
		accountProvider,
		nil,
		cipher,
		metrics.Noop{},
		service.SyncCooldown{},
		"https://fin.example.com/connections",
	)

	return handler.NewConnectionHandler(connections, connectionService)
}

type fakeAccounts struct {
	repository.AccountRepository
	listByUser    func(ctx context.Context, userID uuid.UUID, filter repository.AccountFilter, limit, offset int) ([]domain.Account, int, error)
	selectCardArt func(ctx context.Context, userID, id uuid.UUID, art domain.CardArt) (domain.Account, error)
}

func (f *fakeAccounts) ListByUser(ctx context.Context, userID uuid.UUID, filter repository.AccountFilter, limit, offset int) ([]domain.Account, int, error) {
	return f.listByUser(ctx, userID, filter, limit, offset)
}

func (f *fakeAccounts) SelectCardArt(ctx context.Context, userID, id uuid.UUID, art domain.CardArt) (domain.Account, error) {
	return f.selectCardArt(ctx, userID, id, art)
}

type fakeCatalog struct {
	search func(ctx context.Context, query string) ([]domain.CatalogCard, error)
	card   func(ctx context.Context, id string) (domain.CatalogCard, error)
}

func (f *fakeCatalog) Search(ctx context.Context, query string) ([]domain.CatalogCard, error) {
	return f.search(ctx, query)
}

func (f *fakeCatalog) Card(ctx context.Context, id string) (domain.CatalogCard, error) {
	return f.card(ctx, id)
}

type fakeTransactions struct {
	repository.TransactionRepository
	listByUser func(ctx context.Context, userID uuid.UUID, filter repository.TransactionFilter, limit, offset int) ([]domain.Transaction, int, error)
}

func (f *fakeTransactions) ListByUser(ctx context.Context, userID uuid.UUID, filter repository.TransactionFilter, limit, offset int) ([]domain.Transaction, int, error) {
	return f.listByUser(ctx, userID, filter, limit, offset)
}

type fakeSessions struct {
	repository.SessionRepository
	delete func(ctx context.Context, tokenHash []byte) error
}

func (f *fakeSessions) Delete(ctx context.Context, tokenHash []byte) error {
	return f.delete(ctx, tokenHash)
}

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

type fakeSummary struct {
	repository.SummaryRepository
	balances func(ctx context.Context, userID uuid.UUID, excludeAccountIDs []uuid.UUID) ([]domain.CurrencyBalance, error)
	counts   func(ctx context.Context, userID uuid.UUID) (domain.SummaryCounts, error)
	cashFlow func(ctx context.Context, userID uuid.UUID, days int, excludeAccountIDs []uuid.UUID) ([]domain.DailyCashFlow, error)
}

func (f *fakeSummary) Balances(ctx context.Context, userID uuid.UUID, excludeAccountIDs []uuid.UUID) ([]domain.CurrencyBalance, error) {
	return f.balances(ctx, userID, excludeAccountIDs)
}

func (f *fakeSummary) Counts(ctx context.Context, userID uuid.UUID) (domain.SummaryCounts, error) {
	return f.counts(ctx, userID)
}

func (f *fakeSummary) CashFlow(ctx context.Context, userID uuid.UUID, days int, excludeAccountIDs []uuid.UUID) ([]domain.DailyCashFlow, error) {
	return f.cashFlow(ctx, userID, days, excludeAccountIDs)
}
