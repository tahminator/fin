package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/service"
)

const accessToken = "access-sandbox-123"

func ownedConnection(t *testing.T, owner domain.User, status domain.ConnectionStatus) domain.Connection {
	t.Helper()

	return domain.Connection{
		ID:                    uuid.New(),
		UserID:                owner.ID,
		Provider:              domain.ProviderPlaid,
		AccessTokenCiphertext: newCipher(t).Encrypt(accessToken),
		InstitutionName:       new("First Platypus Bank"),
		Status:                status,
	}
}

func failingProvider(t *testing.T) *fakeProvider {
	t.Helper()

	return &fakeProvider{
		createLinkToken: func(context.Context, domain.Credential, string, string) (string, error) {
			t.Error("provider CreateLinkToken called")

			return "", nil
		},
		createUpdateLinkToken: func(context.Context, domain.Credential, string, string, string) (string, error) {
			t.Error("provider CreateUpdateLinkToken called")

			return "", nil
		},
		createAccountSelectionLinkToken: func(context.Context, domain.Credential, string, string, string) (string, error) {
			t.Error("provider CreateAccountSelectionLinkToken called")

			return "", nil
		},
		remove: func(context.Context, domain.Credential, string) error {
			t.Error("provider Remove called")

			return nil
		},
	}
}

func TestConnectionTenancy(t *testing.T) {
	owner := newUser()
	intruder := newUser()
	connection := ownedConnection(t, owner, domain.ConnectionStatusNeedsReauth)
	connectionPath := "/api/connections/" + connection.ID.String()

	tests := []struct {
		name    string
		request func(api humatest.TestAPI) *httptest.ResponseRecorder
	}{
		{name: "sync", request: func(api humatest.TestAPI) *httptest.ResponseRecorder { return api.Post(connectionPath + "/sync") }},
		{name: "reauth link token", request: func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Post(connectionPath + "/reauth/link-token")
		}},
		{name: "account selection link token", request: func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Post(connectionPath + "/account-selection/link-token")
		}},
		{name: "complete account selection", request: func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Post(connectionPath + "/account-selection/complete")
		}},
		{name: "delete", request: func(api humatest.TestAPI) *httptest.ResponseRecorder { return api.Delete(connectionPath) }},
	}

	for _, tc := range tests {
		t.Run(tc.name+" of another user's connection is not found", func(t *testing.T) {
			connections := &fakeConnections{
				get: ownedConnections(connection),
				delete: func(context.Context, uuid.UUID, uuid.UUID) error {
					t.Error("connection deleted")

					return nil
				},
			}
			api := newAPI(t, intruder, newConnectionHandler(newCipher(t), connections, plaidCredentials(), failingProvider(t)))

			rec := tc.request(api)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}

func TestConnectionSentinelErrors(t *testing.T) {
	user := newUser()
	active := ownedConnection(t, user, domain.ConnectionStatusActive)
	needsReauth := ownedConnection(t, user, domain.ConnectionStatusNeedsReauth)

	tests := []struct {
		name        string
		connections *fakeConnections
		credentials *fakeCredentials
		method      string
		path        string
		status      int
	}{
		{
			name:        "missing credentials is forbidden",
			connections: &fakeConnections{},
			credentials: &fakeCredentials{
				find: func(context.Context, uuid.UUID, domain.Provider) (domain.Credential, error) {
					return domain.Credential{}, domain.ErrNotFound
				},
			},
			method: http.MethodPost,
			path:   "/api/connections/link-token",
			status: http.StatusForbidden,
		},
		{
			name:        "reauth link token for an active connection conflicts",
			connections: &fakeConnections{get: ownedConnections(active)},
			credentials: plaidCredentials(),
			method:      http.MethodPost,
			path:        "/api/connections/" + active.ID.String() + "/reauth/link-token",
			status:      http.StatusConflict,
		},
		{
			name:        "account selection link token for a connection needing reauth conflicts",
			connections: &fakeConnections{get: ownedConnections(needsReauth)},
			credentials: plaidCredentials(),
			method:      http.MethodPost,
			path:        "/api/connections/" + needsReauth.ID.String() + "/account-selection/link-token",
			status:      http.StatusConflict,
		},
		{
			name:        "unknown connection is not found",
			connections: &fakeConnections{get: ownedConnections(active)},
			credentials: plaidCredentials(),
			method:      http.MethodPost,
			path:        "/api/connections/" + uuid.New().String() + "/sync",
			status:      http.StatusNotFound,
		},
		{
			name: "unexpected repository error is internal",
			connections: &fakeConnections{
				listByUser: func(context.Context, uuid.UUID, int, int) ([]domain.Connection, int, error) {
					return nil, 0, errors.New("database exploded")
				},
			},
			credentials: plaidCredentials(),
			method:      http.MethodGet,
			path:        "/api/connections",
			status:      http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := newAPI(t, user, newConnectionHandler(newCipher(t), tc.connections, tc.credentials, failingProvider(t)))

			rec := api.Do(tc.method, tc.path)

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body.String())
			}

			if strings.Contains(rec.Body.String(), "database exploded") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}
		})
	}
}

func TestConnectionReauthLinkTokenForOwner(t *testing.T) {
	user := newUser()
	connection := ownedConnection(t, user, domain.ConnectionStatusNeedsReauth)

	var gotAccessToken, gotUserID string

	accountProvider := failingProvider(t)
	accountProvider.createUpdateLinkToken = func(_ context.Context, _ domain.Credential, userID, token, _ string) (string, error) {
		gotUserID = userID
		gotAccessToken = token

		return "link-update-1", nil
	}

	connections := &fakeConnections{get: ownedConnections(connection)}
	api := newAPI(t, user, newConnectionHandler(newCipher(t), connections, plaidCredentials(), accountProvider))

	rec := api.Post("/api/connections/" + connection.ID.String() + "/reauth/link-token")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := decode[handler.LinkToken](t, rec).LinkToken; got != "link-update-1" {
		t.Errorf("linkToken = %q, want %q", got, "link-update-1")
	}

	if gotAccessToken != accessToken {
		t.Errorf("provider access token = %q, want %q", gotAccessToken, accessToken)
	}

	if gotUserID != user.ID.String() {
		t.Errorf("provider user id = %q, want %q", gotUserID, user.ID.String())
	}
}

func TestConnectionDeleteByOwner(t *testing.T) {
	user := newUser()
	connection := ownedConnection(t, user, domain.ConnectionStatusActive)

	var removedToken string
	var deletedUser, deletedID uuid.UUID

	accountProvider := failingProvider(t)
	accountProvider.remove = func(_ context.Context, _ domain.Credential, token string) error {
		removedToken = token

		return nil
	}

	connections := &fakeConnections{
		get: ownedConnections(connection),
		delete: func(_ context.Context, userID, id uuid.UUID) error {
			deletedUser = userID
			deletedID = id

			return nil
		},
	}
	api := newAPI(t, user, newConnectionHandler(newCipher(t), connections, plaidCredentials(), accountProvider))

	rec := api.Delete("/api/connections/" + connection.ID.String())

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}

	if removedToken != accessToken {
		t.Errorf("provider removed token %q, want %q", removedToken, accessToken)
	}

	if deletedUser != user.ID || deletedID != connection.ID {
		t.Errorf("deleted (%s, %s), want (%s, %s)", deletedUser, deletedID, user.ID, connection.ID)
	}
}

type refusingLimiter struct{}

func (refusingLimiter) Allow(context.Context, domain.RateLimit, string) (domain.RateLimitDecision, error) {
	return domain.RateLimitDecision{Allowed: false, RetryAfter: 90 * time.Second}, nil
}

func (refusingLimiter) Stamp(context.Context, domain.RateLimit, string) error {
	return nil
}

func (refusingLimiter) Release(context.Context, domain.RateLimit, string) error {
	return nil
}

func TestConnectionSyncInsideCooldownIsTooManyRequests(t *testing.T) {
	user := newUser()
	connection := ownedConnection(t, user, domain.ConnectionStatusActive)
	connections := &fakeConnections{get: ownedConnections(connection)}

	connectionService := service.NewConnectionService(
		service.ConnectionRepositories{Credentials: plaidCredentials(), Connections: connections},
		failingProvider(t),
		nil,
		newCipher(t),
		metrics.Noop{},
		service.SyncCooldown{Limiter: refusingLimiter{}, Window: 10 * time.Minute},
		"https://fin.example.com/connections",
	)

	api := newAPI(t, user, handler.NewConnectionHandler(connections, connectionService))

	rec := api.Post("/api/connections/" + connection.ID.String() + "/sync")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}

	if got := rec.Header().Get("Retry-After"); got != "90" {
		t.Errorf("Retry-After = %q, want %q", got, "90")
	}
}
