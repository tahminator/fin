package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/oauth"
)

const (
	testIssuer   = "https://fin.example.test"
	testResource = "https://fin.example.test/mcp"
	testRedirect = "https://app.example.test/callback"
	testVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
)

type oauthFixture struct {
	store    *memoryOAuth
	recorder *oauthRecorder
	service  *OAuthService
	user     domain.User
	clientID string
}

func newOAuthFixture(t *testing.T) *oauthFixture {
	t.Helper()

	store := newMemoryOAuth()
	recorder := &oauthRecorder{}
	svc := NewOAuthService(OAuthConfig{
		Issuer:          testIssuer,
		Resource:        testResource,
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		GrantMaxAge:     90 * 24 * time.Hour,
	}, store.repos(), unusedFetcher{}, recorder)
	svc.now = fixedClock

	user := domain.User{ID: uuid.New(), Email: "ada@example.com"}
	store.users[user.ID] = user

	registered, err := svc.Register(t.Context(), RegistrationRequest{ClientName: "Test app", RedirectURIs: []string{testRedirect}})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	return &oauthFixture{store: store, recorder: recorder, service: svc, user: user, clientID: registered.Client.ID}
}

func challengeOf(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *oauthFixture) authorize(t *testing.T, scope string) string {
	t.Helper()

	result, err := f.service.Authorize(t.Context(), AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            f.clientID,
		RedirectURI:         testRedirect,
		CodeChallenge:       challengeOf(testVerifier),
		CodeChallengeMethod: "S256",
		State:               "xyz",
		Scope:               scope,
		Resource:            testResource,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	return result.RequestToken
}

func (f *oauthFixture) approve(t *testing.T, request string, scopes ...domain.Scope) string {
	t.Helper()

	redirect, err := f.service.Approve(t.Context(), f.user.ID, request, scopes)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}

	return parsed.Query().Get("code")
}

func (f *oauthFixture) exchangeCode(t *testing.T, code string) TokenResponse {
	t.Helper()

	response, err := f.service.Exchange(t.Context(), TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		CodeVerifier: testVerifier,
		RedirectURI:  testRedirect,
		ClientID:     f.clientID,
	})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}

	return response
}

func oauthCode(err error) oauth.ErrorCode {
	oauthErr, ok := oauth.AsError(err)
	if !ok {
		return ""
	}

	return oauthErr.Code
}

func TestAuthorizeRejectsBeforeRedirectingForUnknownClientOrRedirect(t *testing.T) {
	f := newOAuthFixture(t)

	tests := map[string]AuthorizeRequest{
		"unknown client":   {ClientID: "fin_client_missing", RedirectURI: testRedirect},
		"unregistered uri": {ClientID: f.clientID, RedirectURI: "https://evil.example.test/cb"},
	}

	for name, request := range tests {
		result, err := f.service.Authorize(t.Context(), request)

		oauthErr, ok := oauth.AsError(err)
		if !ok || oauthErr.Redirect || result.RedirectURI != "" {
			t.Errorf("%s: err = %v, redirect %q; want a non-redirectable error", name, err, result.RedirectURI)
		}
	}
}

func TestAuthorizeRedirectableErrors(t *testing.T) {
	f := newOAuthFixture(t)
	valid := AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            f.clientID,
		RedirectURI:         testRedirect,
		CodeChallenge:       challengeOf(testVerifier),
		CodeChallengeMethod: "S256",
	}

	tests := []struct {
		name   string
		mutate func(r *AuthorizeRequest)
		code   oauth.ErrorCode
	}{
		{name: "token response type", mutate: func(r *AuthorizeRequest) { r.ResponseType = "token" }, code: oauth.ErrUnsupportedResponseType},
		{name: "plain pkce", mutate: func(r *AuthorizeRequest) { r.CodeChallengeMethod = "plain" }, code: oauth.ErrInvalidRequest},
		{name: "missing pkce", mutate: func(r *AuthorizeRequest) { r.CodeChallenge = "" }, code: oauth.ErrInvalidRequest},
		{name: "other resource", mutate: func(r *AuthorizeRequest) { r.Resource = "https://other.example.test/mcp" }, code: oauth.ErrInvalidTarget},
		{name: "unknown scope", mutate: func(r *AuthorizeRequest) { r.Scope = "admin" }, code: oauth.ErrInvalidScope},
	}

	for _, tt := range tests {
		request := valid
		tt.mutate(&request)

		result, err := f.service.Authorize(t.Context(), request)

		oauthErr, ok := oauth.AsError(err)
		if !ok || oauthErr.Code != tt.code || !oauthErr.Redirect || result.RedirectURI != testRedirect {
			t.Errorf("%s: err = %v, redirect %q", tt.name, err, result.RedirectURI)
		}
	}
}

func TestAuthorizeDefaultsToAllScopesAndOnlyStoresHashes(t *testing.T) {
	f := newOAuthFixture(t)
	request := f.authorize(t, "")

	pending, err := f.service.PendingRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("PendingRequest: %v", err)
	}

	if !slices.Equal(pending.Scopes, domain.AllScopes()) {
		t.Errorf("scopes = %v, want all", pending.Scopes)
	}

	if bytes.Equal(f.store.authorizations[0].requestHash, []byte(request)) || !bytes.Equal(f.store.authorizations[0].requestHash, HashToken(request)) {
		t.Error("request token stored in plaintext")
	}
}

func TestApproveRedirectsWithCodeStateAndIssuer(t *testing.T) {
	f := newOAuthFixture(t)

	redirect, err := f.service.Approve(t.Context(), f.user.ID, f.authorize(t, ""), []domain.Scope{domain.ScopeRead})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	parsed, _ := url.Parse(redirect)
	query := parsed.Query()

	if !strings.HasPrefix(redirect, testRedirect+"?") || query.Get("code") == "" || query.Get("state") != "xyz" || query.Get("iss") != testIssuer {
		t.Errorf("redirect = %s", redirect)
	}

	if !slices.Equal(f.recorder.consents, []metrics.ConsentDecision{metrics.ConsentApproved}) {
		t.Errorf("consents = %v", f.recorder.consents)
	}
}

func TestApproveRequiresReadAndOnlyRequestedScopes(t *testing.T) {
	f := newOAuthFixture(t)

	for name, scopes := range map[string][]domain.Scope{
		"without read":       {domain.ScopeSync},
		"wider than request": {domain.ScopeRead, domain.ScopeSync},
	} {
		_, err := f.service.Approve(t.Context(), f.user.ID, f.authorize(t, "fin:read"), scopes)
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("%s: err = %v, want ErrConflict", name, err)
		}
	}
}

func TestDenyRedirectsWithAccessDenied(t *testing.T) {
	f := newOAuthFixture(t)
	request := f.authorize(t, "")

	redirect, err := f.service.Deny(t.Context(), request)
	if err != nil {
		t.Fatalf("Deny: %v", err)
	}

	parsed, _ := url.Parse(redirect)
	if parsed.Query().Get("error") != "access_denied" || parsed.Query().Get("state") != "xyz" {
		t.Errorf("redirect = %s", redirect)
	}

	if _, err := f.service.PendingRequest(t.Context(), request); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("denied request still pending: %v", err)
	}
}

func TestCodeExchangeIssuesHashedTokensWithApprovedScope(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead))

	if !strings.HasPrefix(tokens.AccessToken, accessTokenPrefix) || !strings.HasPrefix(tokens.RefreshToken, refreshTokenPrefix) {
		t.Errorf("tokens = %+v", tokens)
	}

	if tokens.Scope != "fin:read" || tokens.ExpiresIn != 3600 {
		t.Errorf("scope %q expires_in %d", tokens.Scope, tokens.ExpiresIn)
	}

	for _, stored := range f.store.tokens {
		if bytes.Contains(stored.TokenHash, []byte("fin_")) {
			t.Error("token stored in plaintext")
		}
	}

	grant, err := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}

	if grant.User.ID != f.user.ID || !grant.Has(domain.ScopeRead) || grant.Has(domain.ScopeSync) {
		t.Errorf("grant = %+v", grant)
	}
}

func TestCodeExchangeRejections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *TokenRequest)
		code   oauth.ErrorCode
	}{
		{name: "wrong verifier", mutate: func(r *TokenRequest) { r.CodeVerifier = strings.Repeat("a", 43) }, code: oauth.ErrInvalidGrant},
		{name: "wrong redirect", mutate: func(r *TokenRequest) { r.RedirectURI = "https://app.example.test/other" }, code: oauth.ErrInvalidGrant},
		{name: "wrong resource", mutate: func(r *TokenRequest) { r.Resource = "https://other.example.test/mcp" }, code: oauth.ErrInvalidTarget},
		{name: "unknown client", mutate: func(r *TokenRequest) { r.ClientID = "fin_client_other" }, code: oauth.ErrInvalidClient},
		{name: "unknown grant type", mutate: func(r *TokenRequest) { r.GrantType = "password" }, code: oauth.ErrUnsupportedGrantType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			request := TokenRequest{
				GrantType:    "authorization_code",
				Code:         f.approve(t, f.authorize(t, ""), domain.ScopeRead),
				CodeVerifier: testVerifier,
				RedirectURI:  testRedirect,
				ClientID:     f.clientID,
			}
			tt.mutate(&request)

			if _, err := f.service.Exchange(t.Context(), request); oauthCode(err) != tt.code {
				t.Errorf("err = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestCodeReuseRevokesTheGrant(t *testing.T) {
	f := newOAuthFixture(t)
	code := f.approve(t, f.authorize(t, ""), domain.ScopeRead)
	tokens := f.exchangeCode(t, code)

	_, err := f.service.Exchange(t.Context(), TokenRequest{GrantType: "authorization_code", Code: code, CodeVerifier: testVerifier, ClientID: f.clientID})
	if oauthCode(err) != oauth.ErrInvalidGrant {
		t.Fatalf("reuse err = %v, want invalid_grant", err)
	}

	if _, err := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken); !errors.Is(err, ErrInvalidAccessToken) {
		t.Errorf("access token still valid after code reuse: %v", err)
	}

	if !slices.Contains(f.recorder.tokens, metrics.OAuthOutcomeReuseDetected) {
		t.Errorf("token outcomes = %v, want reuse_detected", f.recorder.tokens)
	}
}

func TestRefreshRotatesAndReplayRevokesTheGrant(t *testing.T) {
	f := newOAuthFixture(t)
	first := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead, domain.ScopeSync))
	refresh := TokenRequest{GrantType: "refresh_token", RefreshToken: first.RefreshToken, ClientID: f.clientID}

	second, err := f.service.Exchange(t.Context(), refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if second.RefreshToken == first.RefreshToken || second.AccessToken == first.AccessToken {
		t.Fatal("refresh did not rotate the tokens")
	}

	narrowed, err := f.service.Exchange(t.Context(), TokenRequest{GrantType: "refresh_token", RefreshToken: second.RefreshToken, ClientID: f.clientID, Scope: "fin:read"})
	if err != nil || narrowed.Scope != "fin:read" {
		t.Fatalf("narrowing refresh = %+v, %v", narrowed, err)
	}

	if _, err := f.service.Exchange(t.Context(), refresh); oauthCode(err) != oauth.ErrInvalidGrant {
		t.Fatalf("replay err = %v, want invalid_grant", err)
	}

	if _, err := f.service.VerifyAccessToken(t.Context(), narrowed.AccessToken); !errors.Is(err, ErrInvalidAccessToken) {
		t.Errorf("tokens survived a refresh replay: %v", err)
	}
}

func TestRefreshCannotWidenScope(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead))

	_, err := f.service.Exchange(t.Context(), TokenRequest{GrantType: "refresh_token", RefreshToken: tokens.RefreshToken, ClientID: f.clientID, Scope: "fin:read fin:sync"})
	if oauthCode(err) != oauth.ErrInvalidScope {
		t.Errorf("err = %v, want invalid_scope", err)
	}
}

func TestReapprovalWithFewerScopesNarrowsExistingTokens(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead, domain.ScopeSync))

	f.approve(t, f.authorize(t, ""), domain.ScopeRead)

	grant, err := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}

	if grant.Has(domain.ScopeSync) {
		t.Error("old token kept fin:sync after the user removed it")
	}
}

func TestRevokeGrantInvalidatesTokensAndIsTenantScoped(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead))

	grant, _ := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken)

	if err := f.service.RevokeGrant(t.Context(), uuid.New(), grant.GrantID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("another user revoked the grant: %v", err)
	}

	if err := f.service.RevokeGrant(t.Context(), f.user.ID, grant.GrantID); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}

	if _, err := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken); !errors.Is(err, ErrInvalidAccessToken) {
		t.Errorf("token valid after revocation: %v", err)
	}
}

func TestRevokeEndpointWithRefreshTokenDisconnectsTheApp(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.exchangeCode(t, f.approve(t, f.authorize(t, ""), domain.ScopeRead))

	if err := f.service.Revoke(t.Context(), tokens.RefreshToken, f.clientID, ""); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, err := f.service.VerifyAccessToken(t.Context(), tokens.AccessToken); !errors.Is(err, ErrInvalidAccessToken) {
		t.Errorf("access token valid after refresh revocation: %v", err)
	}

	if err := f.service.Revoke(t.Context(), "fin_rt_unknown", f.clientID, ""); err != nil {
		t.Errorf("unknown token revocation should succeed: %v", err)
	}
}

func TestConfidentialClientNeedsItsSecret(t *testing.T) {
	f := newOAuthFixture(t)

	registered, err := f.service.Register(t.Context(), RegistrationRequest{RedirectURIs: []string{testRedirect}, TokenEndpointAuthMethod: "client_secret_post"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if registered.ClientSecret == "" || bytes.Equal(registered.Client.ClientSecretHash, []byte(registered.ClientSecret)) {
		t.Fatal("secret missing or stored in plaintext")
	}

	if _, err := f.service.Exchange(t.Context(), TokenRequest{GrantType: "refresh_token", RefreshToken: "x", ClientID: registered.Client.ID, ClientSecret: "wrong"}); oauthCode(err) != oauth.ErrInvalidClient {
		t.Errorf("wrong secret err = %v, want invalid_client", err)
	}
}

func TestRegisterValidatesMetadata(t *testing.T) {
	f := newOAuthFixture(t)

	tests := map[string]RegistrationRequest{
		"no redirect":     {},
		"http redirect":   {RedirectURIs: []string{"http://example.test/cb"}},
		"implicit grant":  {RedirectURIs: []string{testRedirect}, GrantTypes: []string{"implicit"}},
		"token response":  {RedirectURIs: []string{testRedirect}, ResponseTypes: []string{"token"}},
		"jwt auth method": {RedirectURIs: []string{testRedirect}, TokenEndpointAuthMethod: "private_key_jwt"},
	}

	for name, request := range tests {
		if _, err := f.service.Register(t.Context(), request); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
