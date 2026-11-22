package handler

import (
	"context"
	"encoding/json"
	"html"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/oauth"
	"github.com/tahminator/fin/internal/service"
)

const (
	oauthBodyLimit           = 64 << 10
	ResourceMetadataPath     = "/.well-known/oauth-protected-resource/mcp"
	AuthorizationServerPath  = "/.well-known/oauth-authorization-server"
	authorizeEndpointPath    = "/oauth/authorize"
	tokenEndpointPath        = "/oauth/token"
	registrationEndpointPath = "/oauth/register"
	revocationEndpointPath   = "/oauth/revoke"
)

type RateLimiter interface {
	Allow(ctx context.Context, limit domain.RateLimit, key string) (domain.RateLimitDecision, error)
}

type OAuthLimits struct {
	Register  domain.RateLimit
	Token     domain.RateLimit
	Authorize domain.RateLimit
}

func DefaultOAuthLimits() OAuthLimits {
	return OAuthLimits{
		Register:  domain.RateLimit{Scope: domain.RateLimitScopeOAuthRegister, Max: 10, Window: time.Hour},
		Token:     domain.RateLimit{Scope: domain.RateLimitScopeOAuthToken, Max: 60, Window: time.Minute},
		Authorize: domain.RateLimit{Scope: domain.RateLimitScopeOAuthAuthorize, Max: 60, Window: time.Minute},
	}
}

type OAuthHandler struct {
	service *service.OAuthService
	limiter RateLimiter
	limits  OAuthLimits
}

func NewOAuthHandler(oauthService *service.OAuthService, limiter RateLimiter, limits OAuthLimits) *OAuthHandler {
	return &OAuthHandler{service: oauthService, limiter: limiter, limits: limits}
}

func (h *OAuthHandler) ResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	cfg := h.service.Config()

	allowAnyOrigin(w)
	writePublicJSON(w, ProtectedResourceMetadata{
		Resource:               cfg.Resource,
		AuthorizationServers:   []string{cfg.Issuer},
		ScopesSupported:        scopeNames(domain.AllScopes()),
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "fin",
	})
}

func (h *OAuthHandler) ServerMetadata(w http.ResponseWriter, _ *http.Request) {
	cfg := h.service.Config()

	allowAnyOrigin(w)
	writePublicJSON(w, AuthorizationServerMetadata{
		Issuer:                                     cfg.Issuer,
		AuthorizationEndpoint:                      cfg.Issuer + authorizeEndpointPath,
		TokenEndpoint:                              cfg.Issuer + tokenEndpointPath,
		RegistrationEndpoint:                       cfg.Issuer + registrationEndpointPath,
		RevocationEndpoint:                         cfg.Issuer + revocationEndpointPath,
		ResponseTypesSupported:                     []string{"code"},
		ResponseModesSupported:                     []string{"query"},
		GrantTypesSupported:                        []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:              []string{"S256"},
		TokenEndpointAuthMethodsSupported:          []string{"none", "client_secret_basic", "client_secret_post"},
		RevocationEndpointAuthMethodsSupported:     []string{"none", "client_secret_basic", "client_secret_post"},
		ScopesSupported:                            scopeNames(domain.AllScopes()),
		ClientIDMetadataDocumentSupported:          true,
		AuthorizationResponseIssParameterSupported: true,
	})
}

func (h *OAuthHandler) Preflight(w http.ResponseWriter, _ *http.Request) {
	allowAnyOrigin(w)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusNoContent)
}

func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r, h.limits.Authorize) {
		return
	}

	query := r.URL.Query()

	result, err := h.service.Authorize(r.Context(), service.AuthorizeRequest{
		ResponseType:        query.Get("response_type"),
		ClientID:            query.Get("client_id"),
		RedirectURI:         query.Get("redirect_uri"),
		CodeChallenge:       query.Get("code_challenge"),
		CodeChallengeMethod: query.Get("code_challenge_method"),
		State:               query.Get("state"),
		Scope:               query.Get("scope"),
		Resource:            query.Get("resource"),
	})
	if err == nil {
		http.Redirect(w, r, "/consent?"+url.Values{"request": {result.RequestToken}}.Encode(), http.StatusSeeOther)

		return
	}

	oauthErr, ok := oauth.AsError(err)
	if !ok {
		slog.ErrorContext(r.Context(), "oauth authorize failed", "err", err)

		oauthErr = oauth.NewRedirectableError(oauth.ErrServerError, "the authorization server failed, try again")
	}

	if oauthErr.Redirect && result.RedirectURI != "" {
		http.Redirect(w, r, h.service.RedirectError(result.RedirectURI, query.Get("state"), oauthErr), http.StatusFound)

		return
	}

	writeAuthorizeErrorPage(w, oauthErr)
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	allowAnyOrigin(w)

	if !h.allow(w, r, h.limits.Token) {
		return
	}

	form, ok := parseForm(w, r)
	if !ok {
		return
	}

	clientID, clientSecret := clientCredentials(r, form)

	response, err := h.service.Exchange(r.Context(), service.TokenRequest{
		GrantType:    form.Get("grant_type"),
		Code:         form.Get("code"),
		RedirectURI:  form.Get("redirect_uri"),
		CodeVerifier: form.Get("code_verifier"),
		RefreshToken: form.Get("refresh_token"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Resource:     form.Get("resource"),
		Scope:        form.Get("scope"),
	})
	if err != nil {
		writeOAuthError(r.Context(), w, err)

		return
	}

	oauth.WriteJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  response.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    response.ExpiresIn,
		RefreshToken: response.RefreshToken,
		Scope:        response.Scope,
	})
}

func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	allowAnyOrigin(w)

	if !h.allow(w, r, h.limits.Register) {
		return
	}

	var request ClientRegistrationRequest

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, oauthBodyLimit))
	if err := decoder.Decode(&request); err != nil {
		oauth.WriteError(w, oauth.NewError(oauth.ErrInvalidClientMetadata, "the request body must be a JSON client registration"))

		return
	}

	registered, err := h.service.Register(r.Context(), service.RegistrationRequest{
		ClientName:              request.ClientName,
		ClientURI:               request.ClientURI,
		LogoURI:                 request.LogoURI,
		RedirectURIs:            request.RedirectURIs,
		GrantTypes:              request.GrantTypes,
		ResponseTypes:           request.ResponseTypes,
		TokenEndpointAuthMethod: request.TokenEndpointAuthMethod,
	})
	if err != nil {
		writeOAuthError(r.Context(), w, err)

		return
	}

	client := registered.Client

	oauth.WriteJSON(w, http.StatusCreated, ClientRegistrationResponse{
		ClientID:                client.ID,
		ClientSecret:            registered.ClientSecret,
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		ClientName:              client.ClientName,
		RedirectURIs:            client.RedirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: string(client.TokenEndpointAuthMethod),
	})
}

func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	allowAnyOrigin(w)

	if !h.allow(w, r, h.limits.Token) {
		return
	}

	form, ok := parseForm(w, r)
	if !ok {
		return
	}

	clientID, clientSecret := clientCredentials(r, form)

	if err := h.service.Revoke(r.Context(), form.Get("token"), clientID, clientSecret); err != nil {
		writeOAuthError(r.Context(), w, err)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *OAuthHandler) allow(w http.ResponseWriter, r *http.Request, limit domain.RateLimit) bool {
	decision, err := h.limiter.Allow(r.Context(), limit, ClientIP(r))
	if err != nil {
		slog.ErrorContext(r.Context(), "oauth rate limit", "err", err)
		oauth.WriteError(w, oauth.NewError(oauth.ErrServerError, "the authorization server failed, try again"))

		return false
	}

	if !decision.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(max(int(math.Ceil(decision.RetryAfter.Seconds())), 1)))
		oauth.WriteError(w, oauth.NewError(oauth.ErrSlowDown, "too many requests, try again later"))

		return false
	}

	return true
}

func parseForm(w http.ResponseWriter, r *http.Request) (url.Values, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		oauth.WriteError(w, oauth.NewError(oauth.ErrInvalidRequest, "the body must be application/x-www-form-urlencoded"))

		return nil, false
	}

	if err := r.ParseForm(); err != nil {
		oauth.WriteError(w, oauth.NewError(oauth.ErrInvalidRequest, "the form body could not be parsed"))

		return nil, false
	}

	return r.PostForm, true
}

func clientCredentials(r *http.Request, form url.Values) (string, string) {
	if id, secret, ok := r.BasicAuth(); ok {
		decodedID, idErr := url.QueryUnescape(id)
		decodedSecret, secretErr := url.QueryUnescape(secret)

		if idErr == nil && secretErr == nil {
			return decodedID, decodedSecret
		}
	}

	return form.Get("client_id"), form.Get("client_secret")
}

func writeOAuthError(ctx context.Context, w http.ResponseWriter, err error) {
	oauthErr, ok := oauth.AsError(err)
	if ok {
		oauth.WriteError(w, oauthErr)

		return
	}

	slog.ErrorContext(ctx, "oauth request failed", "err", err)
	oauth.WriteError(w, oauth.NewError(oauth.ErrServerError, "the authorization server failed, try again"))
}

func writeAuthorizeErrorPage(w http.ResponseWriter, err *oauth.Error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusBadRequest)

	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>fin: can't connect this app</title>` +
		`<body style="font-family:system-ui,sans-serif;max-width:32rem;margin:4rem auto;padding:0 1rem;line-height:1.5">` +
		`<h1 style="font-size:1.25rem">This app can't connect to fin</h1><p>` + html.EscapeString(err.Description) +
		`</p><p style="color:#666">Error: <code>` + html.EscapeString(string(err.Code)) + `</code>. Start again from the app.</p></body>`))
}

func allowAnyOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func writePublicJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	_ = json.NewEncoder(w).Encode(body)
}

func scopeNames(scopes []domain.Scope) []string {
	names := make([]string, 0, len(scopes))

	for _, scope := range scopes {
		names = append(names, string(scope))
	}

	return names
}
