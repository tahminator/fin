package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/oauth"
	"github.com/tahminator/fin/internal/repository"
)

const (
	authorizationRequestTTL = 10 * time.Minute
	authorizationCodeTTL    = 60 * time.Second
	unusedClientRetention   = 7 * 24 * time.Hour
	maxClientNameLength     = 100
	maxRedirectURIs         = 10
	maxStateLength          = 512
	accessTokenPrefix       = "fin_at_"
	refreshTokenPrefix      = "fin_rt_"
	registeredClientPrefix  = "fin_client_"
	clientSecretPrefix      = "fin_cs_"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type OAuthConfig struct {
	Issuer          string
	Resource        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	GrantMaxAge     time.Duration
}

type OAuthRepositories struct {
	Clients        repository.OAuthClientRepository
	Authorizations repository.OAuthAuthorizationRepository
	Grants         repository.OAuthGrantRepository
	Tokens         repository.OAuthTokenRepository
}

type clientMetadataFetcher interface {
	Fetch(ctx context.Context, clientID string) (oauth.FetchedClient, error)
}

type OAuthService struct {
	cfg            OAuthConfig
	clients        repository.OAuthClientRepository
	authorizations repository.OAuthAuthorizationRepository
	grants         repository.OAuthGrantRepository
	tokens         repository.OAuthTokenRepository
	fetcher        clientMetadataFetcher
	recorder       metrics.Recorder
	now            func() time.Time
}

func NewOAuthService(cfg OAuthConfig, repos OAuthRepositories, fetcher clientMetadataFetcher, recorder metrics.Recorder) *OAuthService {
	return &OAuthService{
		cfg:            cfg,
		clients:        repos.Clients,
		authorizations: repos.Authorizations,
		grants:         repos.Grants,
		tokens:         repos.Tokens,
		fetcher:        fetcher,
		recorder:       recorder,
		now:            time.Now,
	}
}

func (s *OAuthService) Config() OAuthConfig {
	return s.cfg
}

func (s *OAuthService) ResolveClient(ctx context.Context, clientID string) (domain.OAuthClient, error) {
	if clientID == "" {
		return domain.OAuthClient{}, oauth.NewError(oauth.ErrInvalidClient, "client_id is required")
	}

	if oauth.IsMetadataDocumentClientID(clientID) {
		return s.resolveMetadataDocumentClient(ctx, clientID)
	}

	client, err := s.clients.Find(ctx, clientID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.OAuthClient{}, oauth.NewError(oauth.ErrInvalidClient, "unknown client_id")
	}

	if err != nil {
		return domain.OAuthClient{}, err
	}

	return client, nil
}

func (s *OAuthService) resolveMetadataDocumentClient(ctx context.Context, clientID string) (domain.OAuthClient, error) {
	cached, err := s.clients.Find(ctx, clientID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.OAuthClient{}, err
	}

	if err == nil && cached.MetadataExpiresAt != nil && cached.MetadataExpiresAt.After(s.now()) {
		return cached, nil
	}

	fetched, err := s.fetcher.Fetch(ctx, clientID)
	if err != nil {
		s.recorder.OAuthClientRegistered(metrics.ClientSourceMetadataDocument, metrics.OAuthOutcomeInvalidClient)

		return domain.OAuthClient{}, err
	}

	client := fetched.Client
	expiresAt := s.now().Add(fetched.MaxAge)
	client.MetadataExpiresAt = &expiresAt

	if err := s.clients.Upsert(ctx, client); err != nil {
		return domain.OAuthClient{}, err
	}

	s.recorder.OAuthClientRegistered(metrics.ClientSourceMetadataDocument, metrics.OAuthOutcomeSuccess)

	return client, nil
}

type RegistrationRequest struct {
	ClientName              string
	ClientURI               string
	LogoURI                 string
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
}

type RegisteredClient struct {
	Client       domain.OAuthClient
	ClientSecret string
}

func (s *OAuthService) Register(ctx context.Context, request RegistrationRequest) (RegisteredClient, error) {
	registered, err := s.register(ctx, request)

	outcome := metrics.OAuthOutcomeSuccess
	if err != nil {
		outcome = oauthOutcomeOf(err)
	}

	s.recorder.OAuthClientRegistered(metrics.ClientSourceRegistered, outcome)

	return registered, err
}

func (s *OAuthService) register(ctx context.Context, request RegistrationRequest) (RegisteredClient, error) {
	if len(request.RedirectURIs) == 0 || len(request.RedirectURIs) > maxRedirectURIs {
		return RegisteredClient{}, oauth.NewError(oauth.ErrInvalidRedirectURI, fmt.Sprintf("between 1 and %d redirect_uris are required", maxRedirectURIs))
	}

	for _, redirectURI := range request.RedirectURIs {
		if err := oauth.ValidateRedirectURI(redirectURI); err != nil {
			return RegisteredClient{}, oauth.NewError(oauth.ErrInvalidRedirectURI, err.Error())
		}
	}

	for _, grantType := range request.GrantTypes {
		if grantType != "authorization_code" && grantType != "refresh_token" {
			return RegisteredClient{}, oauth.NewError(oauth.ErrInvalidClientMetadata, fmt.Sprintf("unsupported grant_type %q", grantType))
		}
	}

	for _, responseType := range request.ResponseTypes {
		if responseType != "code" {
			return RegisteredClient{}, oauth.NewError(oauth.ErrInvalidClientMetadata, fmt.Sprintf("unsupported response_type %q", responseType))
		}
	}

	method := domain.TokenAuthMethod(request.TokenEndpointAuthMethod)
	if method == "" {
		method = domain.TokenAuthNone
	}

	if !method.Valid() {
		return RegisteredClient{}, oauth.NewError(oauth.ErrInvalidClientMetadata, fmt.Sprintf("unsupported token_endpoint_auth_method %q", method))
	}

	name := strings.TrimSpace(request.ClientName)
	if name == "" {
		name = "Unnamed app"
	}

	if len([]rune(name)) > maxClientNameLength {
		name = string([]rune(name)[:maxClientNameLength])
	}

	client := domain.OAuthClient{
		ID:                      registeredClientPrefix + randomToken(),
		Source:                  domain.ClientSourceRegistered,
		ClientName:              name,
		ClientURI:               httpsOrNil(request.ClientURI),
		RedirectURIs:            request.RedirectURIs,
		TokenEndpointAuthMethod: method,
		CreatedAt:               s.now(),
	}

	var secret string

	if method != domain.TokenAuthNone {
		secret = clientSecretPrefix + randomToken()
		client.ClientSecretHash = HashToken(secret)
	}

	if err := s.clients.DeleteUnusedRegistered(ctx, s.now().Add(-unusedClientRetention)); err != nil {
		slog.WarnContext(ctx, "delete unused oauth clients", "err", err)
	}

	if err := s.clients.Create(ctx, client); err != nil {
		return RegisteredClient{}, err
	}

	return RegisteredClient{Client: client, ClientSecret: secret}, nil
}

type AuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
	Scope               string
	Resource            string
}

type AuthorizeResult struct {
	RequestToken string
	RedirectURI  string
}

func (s *OAuthService) Authorize(ctx context.Context, request AuthorizeRequest) (AuthorizeResult, error) {
	client, err := s.ResolveClient(ctx, request.ClientID)
	if err != nil {
		return AuthorizeResult{}, err
	}

	redirectURI := request.RedirectURI
	if redirectURI == "" && len(client.RedirectURIs) == 1 {
		redirectURI = client.RedirectURIs[0]
	}

	if redirectURI == "" || !oauth.MatchRedirectURI(client.RedirectURIs, redirectURI) {
		return AuthorizeResult{}, oauth.NewError(oauth.ErrInvalidRequest, "redirect_uri is not registered for this client")
	}

	fail := func(code oauth.ErrorCode, description string) (AuthorizeResult, error) {
		return AuthorizeResult{RedirectURI: redirectURI}, oauth.NewRedirectableError(code, description)
	}

	if request.ResponseType != "code" {
		return fail(oauth.ErrUnsupportedResponseType, "response_type must be code")
	}

	if request.CodeChallengeMethod != "S256" || !oauth.ValidChallenge(request.CodeChallenge) {
		return fail(oauth.ErrInvalidRequest, "a PKCE code_challenge with code_challenge_method S256 is required")
	}

	if len(request.State) > maxStateLength {
		return fail(oauth.ErrInvalidRequest, "state is too long")
	}

	if request.Resource != "" && normalizeResource(request.Resource) != s.cfg.Resource {
		return fail(oauth.ErrInvalidTarget, "resource must be "+s.cfg.Resource)
	}

	scopes, err := oauth.ParseScopes(request.Scope)
	if err != nil {
		return fail(oauth.ErrInvalidScope, err.Error())
	}

	if len(scopes) == 0 {
		scopes = domain.AllScopes()
	}

	if !slices.Contains(scopes, domain.ScopeRead) {
		scopes = append([]domain.Scope{domain.ScopeRead}, scopes...)
	}

	requestToken := randomToken()

	var state *string
	if request.State != "" {
		state = &request.State
	}

	err = s.authorizations.Create(ctx, repository.NewAuthorization{
		RequestTokenHash: HashToken(requestToken),
		ClientID:         client.ID,
		RedirectURI:      redirectURI,
		Scope:            oauth.FormatScopes(scopes),
		Resource:         s.cfg.Resource,
		State:            state,
		CodeChallenge:    request.CodeChallenge,
		ExpiresAt:        s.now().Add(authorizationRequestTTL),
	})
	if err != nil {
		return AuthorizeResult{RedirectURI: redirectURI}, err
	}

	return AuthorizeResult{RequestToken: requestToken, RedirectURI: redirectURI}, nil
}

type ConsentRequest struct {
	Client      domain.OAuthClient
	RedirectURI string
	Scopes      []domain.Scope
	ExpiresAt   time.Time
}

func (s *OAuthService) PendingRequest(ctx context.Context, requestToken string) (ConsentRequest, error) {
	authorization, err := s.authorizations.FindPending(ctx, HashToken(requestToken))
	if err != nil {
		return ConsentRequest{}, err
	}

	client, err := s.clients.Find(ctx, authorization.ClientID)
	if err != nil {
		return ConsentRequest{}, err
	}

	scopes, err := oauth.ParseScopes(authorization.Scope)
	if err != nil {
		return ConsentRequest{}, err
	}

	return ConsentRequest{
		Client:      client,
		RedirectURI: authorization.RedirectURI,
		Scopes:      scopes,
		ExpiresAt:   authorization.ExpiresAt,
	}, nil
}

func (s *OAuthService) Approve(ctx context.Context, userID uuid.UUID, requestToken string, approved []domain.Scope) (string, error) {
	authorization, err := s.authorizations.FindPending(ctx, HashToken(requestToken))
	if err != nil {
		return "", err
	}

	requested, err := oauth.ParseScopes(authorization.Scope)
	if err != nil {
		return "", err
	}

	approved, err = oauth.ParseScopes(oauth.FormatScopes(approved))
	if err != nil {
		return "", err
	}

	if !slices.Contains(approved, domain.ScopeRead) || !oauth.Subset(approved, requested) {
		return "", fmt.Errorf("%w: approve fin:read and only scopes the app asked for", domain.ErrConflict)
	}

	scope := oauth.FormatScopes(approved)

	grantID, err := s.grants.Upsert(ctx, userID, authorization.ClientID, scope, authorization.Resource, s.now().Add(s.cfg.GrantMaxAge))
	if err != nil {
		return "", err
	}

	code := randomToken()

	err = s.authorizations.Approve(ctx, repository.AuthorizationApproval{
		ID:            authorization.ID,
		UserID:        userID,
		Scope:         scope,
		GrantID:       grantID,
		CodeHash:      HashToken(code),
		CodeExpiresAt: s.now().Add(authorizationCodeTTL),
	})
	if err != nil {
		return "", err
	}

	s.recorder.OAuthConsentDecided(metrics.ConsentApproved)

	return s.redirectWith(authorization, url.Values{"code": {code}}), nil
}

func (s *OAuthService) Deny(ctx context.Context, requestToken string) (string, error) {
	authorization, err := s.authorizations.FindPending(ctx, HashToken(requestToken))
	if err != nil {
		return "", err
	}

	if err := s.authorizations.DeletePending(ctx, authorization.ID); err != nil {
		return "", err
	}

	s.recorder.OAuthConsentDecided(metrics.ConsentDenied)

	return s.redirectWith(authorization, url.Values{
		"error":             {string(oauth.ErrAccessDenied)},
		"error_description": {"the user denied access"},
	}), nil
}

func (s *OAuthService) RedirectError(redirectURI, state string, err *oauth.Error) string {
	values := url.Values{"error": {string(err.Code)}, "error_description": {err.Description}}

	if state != "" {
		values.Set("state", state)
	}

	values.Set("iss", s.cfg.Issuer)

	return appendQuery(redirectURI, values)
}

func (s *OAuthService) redirectWith(authorization domain.OAuthAuthorization, values url.Values) string {
	if authorization.State != nil {
		values.Set("state", *authorization.State)
	}

	values.Set("iss", s.cfg.Issuer)

	return appendQuery(authorization.RedirectURI, values)
}

func (s *OAuthService) ListGrants(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.OAuthGrant, int, error) {
	return s.grants.ListByUser(ctx, userID, limit, offset)
}

func (s *OAuthService) RevokeGrant(ctx context.Context, userID, grantID uuid.UUID) error {
	return s.grants.Delete(ctx, userID, grantID)
}

func appendQuery(rawURL string, values url.Values) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsed.Query()

	for key, list := range values {
		for _, value := range list {
			query.Add(key, value)
		}
	}

	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func normalizeResource(resource string) string {
	return strings.TrimSuffix(resource, "/")
}

func httpsOrNil(raw string) *string {
	parsed, err := url.Parse(raw)
	if raw == "" || err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil
	}

	return &raw
}

func oauthOutcomeOf(err error) metrics.OAuthOutcome {
	oauthErr, ok := oauth.AsError(err)
	if !ok {
		return metrics.OAuthOutcomeError
	}

	if outcome, ok := oauthOutcomes[oauthErr.Code]; ok {
		return outcome
	}

	return metrics.OAuthOutcomeInvalidRequest
}

var oauthOutcomes = map[oauth.ErrorCode]metrics.OAuthOutcome{
	oauth.ErrInvalidClient: metrics.OAuthOutcomeInvalidClient,
	oauth.ErrInvalidGrant:  metrics.OAuthOutcomeInvalidGrant,
	oauth.ErrServerError:   metrics.OAuthOutcomeError,
}
