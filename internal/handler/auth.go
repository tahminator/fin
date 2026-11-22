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

	consentRequestLength = 43
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
	consent := consentRequestFrom(r.URL.Query().Get("consent"))

	parts := []string{start.Pending.State, start.Pending.Nonce, start.Pending.Verifier}

	if consent != "" {
		parts = append(parts, consent)
	}

	pending := strings.Join(parts, oauthCookieDivider)

	http.SetCookie(w, h.oauthCookie(pending, oauthCookieMaxAge))
	http.Redirect(w, r, start.URL, http.StatusFound)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.oauthCookie("", -1))

	query := r.URL.Query()
	pending, consent := pendingLoginFrom(r)

	token, err := h.auth.CompleteLogin(r.Context(), service.LoginCallback{
		Code:    query.Get("code"),
		State:   query.Get("state"),
		Pending: pending,
	})
	if err != nil {
		h.redirectLoginError(w, r, err, consent)

		return
	}

	http.SetCookie(w, h.sessionCookie(token, int(h.auth.SessionTTL()/time.Second)))
	http.Redirect(w, r, afterLogin(consent, ""), http.StatusSeeOther)
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

func (h *AuthHandler) redirectLoginError(w http.ResponseWriter, r *http.Request, err error, consent string) {
	reason := "failed"

	if errors.Is(err, domain.ErrNotAllowed) {
		reason = "not_allowed"
	}

	slog.WarnContext(r.Context(), "login failed", "reason", reason, "err", err)

	if consent != "" {
		http.Redirect(w, r, afterLogin(consent, reason), http.StatusSeeOther)

		return
	}

	http.Redirect(w, r, "/login?"+url.Values{"error": {reason}}.Encode(), http.StatusSeeOther)
}

func afterLogin(consent, loginError string) string {
	if consent == "" {
		return "/"
	}

	values := url.Values{"request": {consent}}

	if loginError != "" {
		values.Set("error", loginError)
	}

	return "/consent?" + values.Encode()
}

func consentRequestFrom(raw string) string {
	if len(raw) != consentRequestLength {
		return ""
	}

	for _, r := range raw {
		valid := r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_'
		if !valid {
			return ""
		}
	}

	return raw
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

func pendingLoginFrom(r *http.Request) (service.PendingLogin, string) {
	cookie, err := r.Cookie(oauthCookie)
	if err != nil {
		return service.PendingLogin{}, ""
	}

	parts := strings.Split(cookie.Value, oauthCookieDivider)

	if len(parts) != 3 && len(parts) != 4 {
		return service.PendingLogin{}, ""
	}

	pending := service.PendingLogin{State: parts[0], Nonce: parts[1], Verifier: parts[2]}

	if len(parts) == 3 {
		return pending, ""
	}

	return pending, consentRequestFrom(parts[3])
}
