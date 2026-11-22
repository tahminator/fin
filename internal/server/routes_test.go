package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/identity"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/server"
	"github.com/tahminator/fin/internal/service"
)

type fakeResolver struct {
	resolve func(ctx context.Context, token string) (domain.User, error)
}

func (f fakeResolver) ResolveSession(ctx context.Context, token string) (domain.User, error) {
	return f.resolve(ctx, token)
}

type fakeUsers struct {
	repository.UserRepository
}

type fakeSessions struct {
	repository.SessionRepository
}

type fakeIdentity struct {
	identity.IdentityProvider
}

func (fakeIdentity) AuthCodeURL(state, _, _ string) string {
	return "https://accounts.example.com/auth?state=" + state
}

type fakeSummary struct {
	repository.SummaryRepository
}

type fakeConnections struct {
	repository.ConnectionRepository
}

type fakeAccounts struct {
	repository.AccountRepository
}

type fakeTransactions struct {
	repository.TransactionRepository
}

func newRouter(t *testing.T, resolver fakeResolver) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	api := server.NewAPI(mux)

	auth := service.NewAuthService(fakeUsers{}, fakeSessions{}, fakeIdentity{}, metrics.Noop{}, time.Hour)
	connections := service.NewConnectionService(service.ConnectionRepositories{}, nil, nil, nil, metrics.Noop{}, service.SyncCooldown{}, "")

	server.RegisterRoutes(mux, api, server.Handlers{
		Auth:           handler.NewAuthHandler(auth, handler.CookieConfig{}),
		Summary:        handler.NewSummaryHandler(fakeSummary{}),
		Connections:    handler.NewConnectionHandler(fakeConnections{}, connections),
		Accounts:       handler.NewAccountHandler(fakeAccounts{}),
		CardArt:        handler.NewCardArtHandler(service.NewCardArtService(fakeAccounts{}, nil)),
		Transactions:   handler.NewTransactionHandler(fakeTransactions{}),
		Notifications:  handler.NewNotificationHandler(service.NewNotificationService(fakeConnections{})),
		Version:        handler.NewVersionHandler("1.2.3"),
		RequireSession: server.RequireSession(api, resolver),
		SPA:            server.NewSPA(newDist(t)),
	})

	return mux
}

func serve(t *testing.T, router http.Handler, method, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	return rec
}

func TestRoutesSessionRequired(t *testing.T) {
	user := domain.User{ID: uuid.New(), Email: "ada@example.com"}
	sessionCookie := &http.Cookie{
		Name:     handler.SessionCookie,
		Value:    "cookie-value",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	tests := []struct {
		name    string
		cookies []*http.Cookie
		resolve func(ctx context.Context, token string) (domain.User, error)
		status  int
	}{
		{
			name:    "missing cookie is unauthorized",
			cookies: nil,
			resolve: func(context.Context, string) (domain.User, error) {
				return domain.User{}, errors.New("resolver must not be called")
			},
			status: http.StatusUnauthorized,
		},
		{
			name:    "resolved session is allowed",
			cookies: []*http.Cookie{sessionCookie},
			resolve: func(_ context.Context, token string) (domain.User, error) {
				if token != sessionCookie.Value {
					return domain.User{}, fmt.Errorf("token %q: %w", token, service.ErrUnauthenticated)
				}

				return user, nil
			},
			status: http.StatusOK,
		},
		{
			name:    "unauthenticated session is unauthorized",
			cookies: []*http.Cookie{sessionCookie},
			resolve: func(context.Context, string) (domain.User, error) {
				return domain.User{}, service.ErrUnauthenticated
			},
			status: http.StatusUnauthorized,
		},
		{
			name:    "wrapped unauthenticated session is unauthorized",
			cookies: []*http.Cookie{sessionCookie},
			resolve: func(context.Context, string) (domain.User, error) {
				return domain.User{}, fmt.Errorf("expired: %w", service.ErrUnauthenticated)
			},
			status: http.StatusUnauthorized,
		},
		{
			name:    "resolver failure is internal error",
			cookies: []*http.Cookie{sessionCookie},
			resolve: func(context.Context, string) (domain.User, error) {
				return domain.User{}, errors.New("database exploded")
			},
			status: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouter(t, fakeResolver{resolve: tc.resolve})

			rec := serve(t, router, http.MethodGet, "/api/me", tc.cookies...)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body.String())
			}

			if strings.Contains(rec.Body.String(), "database exploded") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}

			if tc.status != http.StatusOK {
				return
			}

			var me handler.User

			if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
				t.Fatalf("decode %q: %v", rec.Body.String(), err)
			}

			if me.ID != user.ID || me.Email != user.Email {
				t.Errorf("me = %+v, want %+v", me, user)
			}
		})
	}
}

func TestRoutesNonAPIPaths(t *testing.T) {
	router := newRouter(t, fakeResolver{resolve: func(context.Context, string) (domain.User, error) {
		return domain.User{}, service.ErrUnauthenticated
	}})

	tests := []struct {
		name      string
		method    string
		path      string
		status    int
		wantIndex bool
	}{
		{name: "unknown api path is not found instead of index", method: http.MethodGet, path: "/api/unknown", status: http.StatusNotFound, wantIndex: false},
		{name: "unknown api method is not found instead of index", method: http.MethodPost, path: "/api/unknown", status: http.StatusNotFound, wantIndex: false},
		{name: "openapi document is served", method: http.MethodGet, path: "/openapi.json", status: http.StatusOK, wantIndex: false},
		{name: "client route serves index", method: http.MethodGet, path: "/transactions", status: http.StatusOK, wantIndex: true},
		{name: "root serves index", method: http.MethodGet, path: "/", status: http.StatusOK, wantIndex: true},
		{name: "google login redirects to identity provider", method: http.MethodGet, path: "/auth/google/login", status: http.StatusFound, wantIndex: false},
		{name: "version is served without a session", method: http.MethodGet, path: "/version", status: http.StatusOK, wantIndex: false},
		{name: "non-get client route is not allowed", method: http.MethodPost, path: "/transactions", status: http.StatusMethodNotAllowed, wantIndex: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(t, router, tc.method, tc.path)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body.String())
			}

			if got := rec.Body.String() == indexBody; got != tc.wantIndex {
				t.Errorf("served index = %v, want %v; body %s", got, tc.wantIndex, rec.Body.String())
			}
		})
	}
}

func TestRoutesOpenAPIDocumentListsAPIPaths(t *testing.T) {
	router := newRouter(t, fakeResolver{})

	rec := serve(t, router, http.MethodGet, "/openapi.json")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var document struct {
		Paths map[string]any `json:"paths"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}

	for _, path := range []string{"/api/me", "/api/connections", "/api/accounts", "/api/transactions", "/api/notifications", "/api/summary", "/version"} {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("openapi paths missing %s", path)
		}
	}
}

func TestRoutesLoginSetsPendingCookie(t *testing.T) {
	router := newRouter(t, fakeResolver{})

	rec := serve(t, router, http.MethodGet, "/auth/google/login")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, "https://accounts.example.com/auth?state=") {
		t.Errorf("Location = %q, want identity provider URL", location)
	}

	cookie, err := http.ParseSetCookie(rec.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("parse Set-Cookie %q: %v", rec.Header().Get("Set-Cookie"), err)
	}

	if cookie.Name != "fin_oauth" || len(strings.Split(cookie.Value, ".")) != 3 {
		t.Errorf("cookie = %+v, want fin_oauth with state.nonce.verifier", cookie)
	}
}
