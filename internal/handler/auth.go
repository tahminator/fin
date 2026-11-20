package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/service"
)

const (
	oauthCookie        = "fin_oauth"
	oauthCookiePath    = "/auth/google/callback"
	oauthCookieMaxAge  = 600
	oauthCookieDivider = "."
)

type CookieConfig struct {
	Domain string
	Secure bool
}

type AuthHandler struct {
	auth    *service.AuthService
	cookies CookieConfig
}

func NewAuthHandler(auth *service.AuthService, cookies CookieConfig) *AuthHandler {
	return &AuthHandler{auth: auth, cookies: cookies}
}

func (h *AuthHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        "/me",
		Summary:     "Current user",
		Tags:        []string{"auth"},
		Security:    cookieAuth,
	}, h.me)

	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		Summary:       "End the current session",
		Tags:          []string{"auth"},
		Security:      cookieAuth,
		DefaultStatus: http.StatusNoContent,
	}, h.logout)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	start := h.auth.BeginLogin()

	pending := strings.Join([]string{start.Pending.State, start.Pending.Nonce, start.Pending.Verifier}, oauthCookieDivider)

	http.SetCookie(w, h.oauthCookie(pending, oauthCookieMaxAge))
	http.Redirect(w, r, start.URL, http.StatusFound)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.oauthCookie("", -1))

	query := r.URL.Query()

	token, err := h.auth.CompleteLogin(r.Context(), service.LoginCallback{
		Code:    query.Get("code"),
		State:   query.Get("state"),
		Pending: pendingLoginFrom(r),
	})
	if err != nil {
		h.redirectLoginError(w, r, err)

		return
	}

	http.SetCookie(w, h.sessionCookie(token, int(h.auth.SessionTTL()/time.Second)))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type MeOutput struct {
	Body User
}

func (h *AuthHandler) me(ctx context.Context, _ *struct{}) (*MeOutput, error) {
	return &MeOutput{Body: newUser(UserFrom(ctx))}, nil
}

type LogoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func (h *AuthHandler) logout(ctx context.Context, _ *struct{}) (*LogoutOutput, error) {
	if err := h.auth.Logout(ctx, sessionTokenFrom(ctx)); err != nil {
		return nil, problem(ctx, err)
	}

	return &LogoutOutput{SetCookie: *h.sessionCookie("", -1)}, nil
}

func (h *AuthHandler) redirectLoginError(w http.ResponseWriter, r *http.Request, err error) {
	reason := "failed"

	if errors.Is(err, domain.ErrNotAllowed) {
		reason = "not_allowed"
	}

	slog.WarnContext(r.Context(), "login failed", "reason", reason, "err", err)

	http.Redirect(w, r, "/login?"+url.Values{"error": {reason}}.Encode(), http.StatusSeeOther)
}

func (h *AuthHandler) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		Domain:   h.cookies.Domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteStrictMode,
	}
}

func (h *AuthHandler) oauthCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     oauthCookie,
		Value:    value,
		Path:     oauthCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func pendingLoginFrom(r *http.Request) service.PendingLogin {
	cookie, err := r.Cookie(oauthCookie)
	if err != nil {
		return service.PendingLogin{}
	}

	parts := strings.Split(cookie.Value, oauthCookieDivider)

	if len(parts) != 3 {
		return service.PendingLogin{}
	}

	return service.PendingLogin{State: parts[0], Nonce: parts[1], Verifier: parts[2]}
}
