package domain

import (
	"time"
	"uuid"
)

type Scope string

const (
	ScopeRead Scope = "fin:read"
	ScopeSync Scope = "fin:sync"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopeRead, ScopeSync:
		return true
	}

	return false
}

func AllScopes() []Scope {
	return []Scope{ScopeRead, ScopeSync}
}

type ClientSource string

const (
	ClientSourceMetadataDocument ClientSource = "metadata_document"
	ClientSourceRegistered       ClientSource = "registered"
)

func (s ClientSource) Valid() bool {
	switch s {
	case ClientSourceMetadataDocument, ClientSourceRegistered:
		return true
	}

	return false
}

type TokenKind string

const (
	TokenKindAccess  TokenKind = "access"
	TokenKindRefresh TokenKind = "refresh"
)

type TokenAuthMethod string

const (
	TokenAuthNone  TokenAuthMethod = "none"
	TokenAuthBasic TokenAuthMethod = "client_secret_basic"
	TokenAuthPost  TokenAuthMethod = "client_secret_post"
)

func (m TokenAuthMethod) Valid() bool {
	switch m {
	case TokenAuthNone, TokenAuthBasic, TokenAuthPost:
		return true
	}

	return false
}

type OAuthClient struct {
	ID                      string          `db:"id"`
	Source                  ClientSource    `db:"source"`
	ClientName              string          `db:"client_name"`
	ClientURI               *string         `db:"client_uri"`
	LogoURI                 *string         `db:"logo_uri"`
	RedirectURIs            []string        `db:"redirect_uris"`
	TokenEndpointAuthMethod TokenAuthMethod `db:"token_endpoint_auth_method"`
	ClientSecretHash        []byte          `db:"client_secret_hash"`
	MetadataExpiresAt       *time.Time      `db:"metadata_expires_at"`
	CreatedAt               time.Time       `db:"created_at"`
}

type OAuthAuthorization struct {
	ID            uuid.UUID  `db:"id"`
	ClientID      string     `db:"client_id"`
	UserID        *uuid.UUID `db:"user_id"`
	RedirectURI   string     `db:"redirect_uri"`
	Scope         string     `db:"scope"`
	Resource      string     `db:"resource"`
	State         *string    `db:"state"`
	CodeChallenge string     `db:"code_challenge"`
	GrantID       *uuid.UUID `db:"grant_id"`
	ExpiresAt     time.Time  `db:"expires_at"`
}

type OAuthGrant struct {
	ID           uuid.UUID    `db:"id"`
	UserID       uuid.UUID    `db:"user_id"`
	ClientID     string       `db:"client_id"`
	ClientName   string       `db:"client_name"`
	ClientSource ClientSource `db:"client_source"`
	Scope        string       `db:"scope"`
	Resource     string       `db:"resource"`
	CreatedAt    time.Time    `db:"created_at"`
	ExpiresAt    time.Time    `db:"expires_at"`
	LastUsedAt   *time.Time   `db:"last_used_at"`
}

type OAuthToken struct {
	ID        uuid.UUID `db:"id"`
	GrantID   uuid.UUID `db:"grant_id"`
	UserID    uuid.UUID `db:"user_id"`
	Kind      TokenKind `db:"kind"`
	Scope     string    `db:"scope"`
	ExpiresAt time.Time `db:"expires_at"`
}

type AccessGrant struct {
	User      User
	GrantID   uuid.UUID
	ClientID  string
	Scopes    []Scope
	ExpiresAt time.Time
}

func (g AccessGrant) Has(scope Scope) bool {
	for _, s := range g.Scopes {
		if s == scope {
			return true
		}
	}

	return false
}
