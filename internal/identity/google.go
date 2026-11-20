package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

type Claims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          *string
}

type IdentityProvider interface {
	AuthCodeURL(state, nonce, verifier string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (Claims, error)
}

type Google struct {
	oauth      *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	httpClient *http.Client
}

func NewGoogle(ctx context.Context, httpClient *http.Client, clientID, clientSecret, redirectURL string) (*Google, error) {
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("discover google oidc: %w", err)
	}

	return &Google{
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier:   provider.Verifier(&oidc.Config{ClientID: clientID}),
		httpClient: httpClient,
	}, nil
}

func (g *Google) AuthCodeURL(state, nonce, verifier string) string {
	return g.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func (g *Google) Exchange(ctx context.Context, code, verifier, nonce string) (Claims, error) {
	ctx = oidc.ClientContext(ctx, g.httpClient)

	token, err := g.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Claims{}, fmt.Errorf("exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return Claims{}, errors.New("token response has no id_token")
	}

	idToken, err := g.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Claims{}, fmt.Errorf("verify id token: %w", err)
	}

	if idToken.Nonce != nonce {
		return Claims{}, errors.New("id token nonce mismatch")
	}

	var parsed googleClaims

	if err := idToken.Claims(&parsed); err != nil {
		return Claims{}, fmt.Errorf("parse id token claims: %w", err)
	}

	claims := Claims{
		Subject:       idToken.Subject,
		Email:         parsed.Email,
		EmailVerified: parsed.EmailVerified,
	}

	if parsed.Name != "" {
		claims.Name = &parsed.Name
	}

	return claims, nil
}
