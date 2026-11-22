package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"slices"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/oauth"
	"github.com/tahminator/fin/internal/repository"
)

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Resource     string
	Scope        string
}

type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scope        string
}

func (s *OAuthService) Exchange(ctx context.Context, request TokenRequest) (TokenResponse, error) {
	grantType := metrics.OAuthGrantAuthorizationCode
	if request.GrantType == "refresh_token" {
		grantType = metrics.OAuthGrantRefreshToken
	}

	response, err := s.exchange(ctx, request)

	switch {
	case err == nil:
		s.recorder.OAuthTokenRequested(grantType, metrics.OAuthOutcomeSuccess)
	case errors.Is(err, errTokenReuse):
		s.recorder.OAuthTokenRequested(grantType, metrics.OAuthOutcomeReuseDetected)
		err = oauth.NewError(oauth.ErrInvalidGrant, "this grant was already used; every token for it has been revoked")
	default:
		s.recorder.OAuthTokenRequested(grantType, oauthOutcomeOf(err))
	}

	s.deleteExpired(ctx)

	return response, err
}

var errTokenReuse = errors.New("oauth token reuse")

func (s *OAuthService) exchange(ctx context.Context, request TokenRequest) (TokenResponse, error) {
	client, err := s.authenticateClient(ctx, request.ClientID, request.ClientSecret)
	if err != nil {
		return TokenResponse{}, err
	}

	switch request.GrantType {
	case "authorization_code":
		return s.exchangeCode(ctx, client, request)
	case "refresh_token":
		return s.exchangeRefresh(ctx, client, request)
	}

	return TokenResponse{}, oauth.NewError(oauth.ErrUnsupportedGrantType, "grant_type must be authorization_code or refresh_token")
}

func (s *OAuthService) exchangeCode(ctx context.Context, client domain.OAuthClient, request TokenRequest) (TokenResponse, error) {
	if request.Code == "" || request.CodeVerifier == "" {
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidRequest, "code and code_verifier are required")
	}

	codeHash := HashToken(request.Code)

	authorization, err := s.authorizations.ConsumeCode(ctx, codeHash)
	if errors.Is(err, domain.ErrNotFound) {
		return TokenResponse{}, s.rejectReusedCode(ctx, codeHash)
	}

	if err != nil {
		return TokenResponse{}, err
	}

	invalid := oauth.NewError(oauth.ErrInvalidGrant, "the authorization code is invalid for this request")

	switch {
	case authorization.ClientID != client.ID:
		return TokenResponse{}, invalid
	case request.RedirectURI != "" && request.RedirectURI != authorization.RedirectURI:
		return TokenResponse{}, invalid
	case request.Resource != "" && normalizeResource(request.Resource) != authorization.Resource:
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidTarget, "resource must be "+authorization.Resource)
	case !oauth.VerifyS256(request.CodeVerifier, authorization.CodeChallenge):
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidGrant, "code_verifier does not match the code_challenge")
	case authorization.UserID == nil || authorization.GrantID == nil:
		return TokenResponse{}, invalid
	}

	return s.issueTokens(ctx, *authorization.GrantID, *authorization.UserID, authorization.Scope)
}

func (s *OAuthService) rejectReusedCode(ctx context.Context, codeHash []byte) error {
	grantID, err := s.authorizations.FindConsumedGrantID(ctx, codeHash)
	if errors.Is(err, domain.ErrNotFound) {
		return oauth.NewError(oauth.ErrInvalidGrant, "the authorization code is invalid or expired")
	}

	if err != nil {
		return err
	}

	if err := s.grants.DeleteByID(ctx, grantID); err != nil {
		return err
	}

	return errTokenReuse
}

func (s *OAuthService) exchangeRefresh(ctx context.Context, client domain.OAuthClient, request TokenRequest) (TokenResponse, error) {
	if request.RefreshToken == "" {
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidRequest, "refresh_token is required")
	}

	tokenHash := HashToken(request.RefreshToken)

	record, err := s.tokens.ConsumeRefresh(ctx, tokenHash)
	if errors.Is(err, domain.ErrNotFound) {
		return TokenResponse{}, s.rejectReusedRefresh(ctx, tokenHash)
	}

	if err != nil {
		return TokenResponse{}, err
	}

	if record.ClientID != client.ID {
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidGrant, "the refresh token was issued to another client")
	}

	if request.Resource != "" && normalizeResource(request.Resource) != record.Resource {
		return TokenResponse{}, oauth.NewError(oauth.ErrInvalidTarget, "resource must be "+record.Resource)
	}

	allowed := intersectScopes(record.TokenScope, record.GrantScope)

	scopes := allowed
	if request.Scope != "" {
		requested, err := oauth.ParseScopes(request.Scope)
		if err != nil || !oauth.Subset(requested, allowed) {
			return TokenResponse{}, oauth.NewError(oauth.ErrInvalidScope, "a refresh may narrow the scope but never widen it")
		}

		scopes = requested
	}

	return s.issueTokens(ctx, record.GrantID, record.UserID, oauth.FormatScopes(scopes))
}

func (s *OAuthService) rejectReusedRefresh(ctx context.Context, tokenHash []byte) error {
	owner, err := s.tokens.FindOwner(ctx, tokenHash)
	if errors.Is(err, domain.ErrNotFound) {
		return oauth.NewError(oauth.ErrInvalidGrant, "the refresh token is invalid or expired")
	}

	if err != nil {
		return err
	}

	if owner.Kind != domain.TokenKindRefresh || !owner.Used {
		return oauth.NewError(oauth.ErrInvalidGrant, "the refresh token is invalid or expired")
	}

	if err := s.grants.DeleteByID(ctx, owner.GrantID); err != nil {
		return err
	}

	return errTokenReuse
}

func (s *OAuthService) issueTokens(ctx context.Context, grantID, userID uuid.UUID, scope string) (TokenResponse, error) {
	accessToken := accessTokenPrefix + randomToken()
	refreshToken := refreshTokenPrefix + randomToken()
	now := s.now()

	err := s.tokens.Create(
		ctx,
		repository.NewToken{
			GrantID:   grantID,
			UserID:    userID,
			Kind:      domain.TokenKindAccess,
			TokenHash: HashToken(accessToken),
			Scope:     scope,
			ExpiresAt: now.Add(s.cfg.AccessTokenTTL),
		},
		repository.NewToken{
			GrantID:   grantID,
			UserID:    userID,
			Kind:      domain.TokenKindRefresh,
			TokenHash: HashToken(refreshToken),
			Scope:     scope,
			ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
		},
	)
	if err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
		Scope:        scope,
	}, nil
}

func (s *OAuthService) authenticateClient(ctx context.Context, clientID, clientSecret string) (domain.OAuthClient, error) {
	client, err := s.ResolveClient(ctx, clientID)
	if err != nil {
		return domain.OAuthClient{}, err
	}

	if client.TokenEndpointAuthMethod == domain.TokenAuthNone {
		return client, nil
	}

	if clientSecret == "" || subtle.ConstantTimeCompare(HashToken(clientSecret), client.ClientSecretHash) != 1 {
		return domain.OAuthClient{}, oauth.NewError(oauth.ErrInvalidClient, "client authentication failed")
	}

	return client, nil
}

func (s *OAuthService) Revoke(ctx context.Context, token, clientID, clientSecret string) error {
	client, err := s.authenticateClient(ctx, clientID, clientSecret)
	if err != nil {
		return err
	}

	tokenHash := HashToken(token)

	owner, err := s.tokens.FindOwner(ctx, tokenHash)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	if owner.ClientID != client.ID {
		return nil
	}

	if owner.Kind == domain.TokenKindRefresh {
		return s.grants.DeleteByID(ctx, owner.GrantID)
	}

	return s.tokens.DeleteByHash(ctx, tokenHash)
}

func (s *OAuthService) VerifyAccessToken(ctx context.Context, token string) (domain.AccessGrant, error) {
	record, err := s.tokens.FindAccess(ctx, HashToken(token))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.AccessGrant{}, ErrInvalidAccessToken
	}

	if err != nil {
		return domain.AccessGrant{}, err
	}

	if record.Resource != s.cfg.Resource {
		return domain.AccessGrant{}, ErrInvalidAccessToken
	}

	if err := s.grants.TouchLastUsed(ctx, record.GrantID); err != nil {
		slog.WarnContext(ctx, "touch oauth grant", "grant_id", record.GrantID, "err", err)
	}

	return domain.AccessGrant{
		User:      record.User,
		GrantID:   record.GrantID,
		ClientID:  record.ClientID,
		Scopes:    intersectScopes(record.TokenScope, record.GrantScope),
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (s *OAuthService) deleteExpired(ctx context.Context) {
	if err := s.tokens.DeleteExpired(ctx); err != nil {
		slog.WarnContext(ctx, "delete expired oauth tokens", "err", err)
	}

	if err := s.authorizations.DeleteExpired(ctx); err != nil {
		slog.WarnContext(ctx, "delete expired oauth authorizations", "err", err)
	}
}

func intersectScopes(a, b string) []domain.Scope {
	left, _ := oauth.ParseScopes(a)
	right, _ := oauth.ParseScopes(b)

	return slices.DeleteFunc(left, func(scope domain.Scope) bool {
		return !slices.Contains(right, scope)
	})
}
