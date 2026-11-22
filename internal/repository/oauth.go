package repository

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type OAuthClientRepository interface {
	Find(ctx context.Context, id string) (domain.OAuthClient, error)
	Create(ctx context.Context, client domain.OAuthClient) error
	Upsert(ctx context.Context, client domain.OAuthClient) error
	DeleteUnusedRegistered(ctx context.Context, createdBefore time.Time) error
}

type NewAuthorization struct {
	RequestTokenHash []byte
	ClientID         string
	RedirectURI      string
	Scope            string
	Resource         string
	State            *string
	CodeChallenge    string
	ExpiresAt        time.Time
}

type AuthorizationApproval struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Scope         string
	GrantID       uuid.UUID
	CodeHash      []byte
	CodeExpiresAt time.Time
}

type OAuthAuthorizationRepository interface {
	Create(ctx context.Context, authorization NewAuthorization) error
	FindPending(ctx context.Context, requestTokenHash []byte) (domain.OAuthAuthorization, error)
	Approve(ctx context.Context, approval AuthorizationApproval) error
	DeletePending(ctx context.Context, id uuid.UUID) error
	ConsumeCode(ctx context.Context, codeHash []byte) (domain.OAuthAuthorization, error)
	FindConsumedGrantID(ctx context.Context, codeHash []byte) (uuid.UUID, error)
	DeleteExpired(ctx context.Context) error
}

type OAuthGrantRepository interface {
	Upsert(ctx context.Context, userID uuid.UUID, clientID, scope, resource string, expiresAt time.Time) (uuid.UUID, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.OAuthGrant, int, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
	TouchLastUsed(ctx context.Context, id uuid.UUID) error
}

type NewToken struct {
	GrantID   uuid.UUID
	UserID    uuid.UUID
	Kind      domain.TokenKind
	TokenHash []byte
	Scope     string
	ExpiresAt time.Time
}

type AccessTokenRecord struct {
	User       domain.User
	GrantID    uuid.UUID
	ClientID   string
	TokenScope string
	GrantScope string
	Resource   string
	ExpiresAt  time.Time
}

type RefreshTokenRecord struct {
	GrantID    uuid.UUID
	UserID     uuid.UUID
	ClientID   string
	TokenScope string
	GrantScope string
	Resource   string
}

type TokenOwner struct {
	GrantID  uuid.UUID
	ClientID string
	Kind     domain.TokenKind
	Used     bool
}

type OAuthTokenRepository interface {
	Create(ctx context.Context, tokens ...NewToken) error
	FindAccess(ctx context.Context, tokenHash []byte) (AccessTokenRecord, error)
	ConsumeRefresh(ctx context.Context, tokenHash []byte) (RefreshTokenRecord, error)
	FindOwner(ctx context.Context, tokenHash []byte) (TokenOwner, error)
	DeleteByHash(ctx context.Context, tokenHash []byte) error
	DeleteExpired(ctx context.Context) error
}
