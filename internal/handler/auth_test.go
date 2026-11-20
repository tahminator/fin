package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

func newAuthHandler(sessions repository.SessionRepository) *handler.AuthHandler {
	auth := service.NewAuthService(nil, sessions, nil, metrics.Noop{}, time.Hour)

	return handler.NewAuthHandler(auth, handler.CookieConfig{Domain: "fin.example.com", Secure: true})
}

func TestMeReturnsSessionUser(t *testing.T) {
	user := newUser()
	user.Name = new("Ada Lovelace")
	user.GoogleSub = new("google-sub-1")
	api := newAPI(t, user, newAuthHandler(&fakeSessions{}))

	rec := api.Get("/api/me")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := decode[map[string]any](t, rec)

	want := map[string]any{"id": user.ID.String(), "email": "ada@example.com", "name": "Ada Lovelace"}

	for key, value := range want {
		if body[key] != value {
			t.Errorf("%s = %#v, want %#v", key, body[key], value)
		}
	}

	if _, ok := body["googleSub"]; ok {
		t.Error("response exposes googleSub")
	}
}

func TestLogoutDeletesSessionAndClearsCookie(t *testing.T) {
	var deleted []byte

	sessions := &fakeSessions{
		delete: func(_ context.Context, tokenHash []byte) error {
			deleted = tokenHash

			return nil
		},
	}
	api := newAPI(t, newUser(), newAuthHandler(sessions))

	rec := api.Post("/api/auth/logout")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	if !bytes.Equal(deleted, service.HashToken(sessionToken)) {
		t.Errorf("deleted hash = %x, want %x", deleted, service.HashToken(sessionToken))
	}

	cookie, err := http.ParseSetCookie(rec.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("parse Set-Cookie %q: %v", rec.Header().Get("Set-Cookie"), err)
	}

	if cookie.Name != handler.SessionCookie || cookie.Value != "" || cookie.MaxAge >= 0 || !cookie.HttpOnly || !cookie.Secure {
		t.Errorf("cookie = %+v, want cleared HttpOnly Secure %s", cookie, handler.SessionCookie)
	}

	if cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "fin.example.com" {
		t.Errorf("cookie = %+v, want SameSite=Strict Path=/ Domain=fin.example.com", cookie)
	}
}

func TestSessionUserMissingFromContext(t *testing.T) {
	if got := handler.UserFrom(context.Background()); got != (domain.User{}) {
		t.Errorf("UserFrom(empty) = %+v, want zero user", got)
	}
}
