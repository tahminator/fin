package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type OAuthTokenRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthTokenRepository(pool *pgxpool.Pool) *OAuthTokenRepository {
	return &OAuthTokenRepository{pool: pool}
}

const createOAuthToken = `
INSERT INTO oauth_tokens (
    grant_id,
    user_id,
    kind,
    token_hash,
    scope,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6);
`

func (r *OAuthTokenRepository) Create(ctx context.Context, tokens ...repository.NewToken) error {
	batch := &pgx.Batch{}

	for _, token := range tokens {
		batch.Queue(createOAuthToken, token.GrantID, token.UserID, string(token.Kind), token.TokenHash, token.Scope, token.ExpiresAt)
	}

	if err := r.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("create oauth tokens: %w", err)
	}

	return nil
}

const findOAuthAccessToken = `
SELECT
    u.id,
    u.email,
    u.google_sub,
    u.name,
    u.created_at,
    g.id                              AS grant_id,
    g.client_id,
    t.scope                           AS token_scope,
    g.scope                           AS grant_scope,
    g.resource,
    least(t.expires_at, g.expires_at) AS expires_at
FROM oauth_tokens AS t
JOIN oauth_grants AS g ON g.id = t.grant_id
JOIN users AS u ON u.id = t.user_id
WHERE t.token_hash = $1
  AND t.kind = 'access'
  AND t.expires_at > now()
  AND g.expires_at > now();
`

func (r *OAuthTokenRepository) FindAccess(ctx context.Context, tokenHash []byte) (repository.AccessTokenRecord, error) {
	var record repository.AccessTokenRecord

	err := r.pool.QueryRow(ctx, findOAuthAccessToken, tokenHash).Scan(
		&record.User.ID,
		&record.User.Email,
		&record.User.GoogleSub,
		&record.User.Name,
		&record.User.CreatedAt,
		&record.GrantID,
		&record.ClientID,
		&record.TokenScope,
		&record.GrantScope,
		&record.Resource,
		&record.ExpiresAt,
	)
	if err != nil {
		return repository.AccessTokenRecord{}, notFound(err)
	}

	return record, nil
}

const consumeOAuthRefreshToken = `
UPDATE oauth_tokens AS t
SET used_at = now()
FROM oauth_grants AS g
WHERE t.token_hash = $1
  AND t.kind = 'refresh'
  AND t.used_at IS NULL
  AND t.expires_at > now()
  AND g.id = t.grant_id
  AND g.expires_at > now()
RETURNING
    t.grant_id,
    t.user_id,
    g.client_id,
    t.scope    AS token_scope,
    g.scope    AS grant_scope,
    g.resource;
`

func (r *OAuthTokenRepository) ConsumeRefresh(ctx context.Context, tokenHash []byte) (repository.RefreshTokenRecord, error) {
	var record repository.RefreshTokenRecord

	err := r.pool.QueryRow(ctx, consumeOAuthRefreshToken, tokenHash).Scan(
		&record.GrantID,
		&record.UserID,
		&record.ClientID,
		&record.TokenScope,
		&record.GrantScope,
		&record.Resource,
	)
	if err != nil {
		return repository.RefreshTokenRecord{}, notFound(err)
	}

	return record, nil
}

const findOAuthTokenOwner = `
SELECT
    t.grant_id,
    g.client_id,
    t.kind,
    t.used_at IS NOT NULL AS used
FROM oauth_tokens AS t
JOIN oauth_grants AS g ON g.id = t.grant_id
WHERE t.token_hash = $1;
`

func (r *OAuthTokenRepository) FindOwner(ctx context.Context, tokenHash []byte) (repository.TokenOwner, error) {
	var (
		owner repository.TokenOwner
		kind  string
	)

	err := r.pool.QueryRow(ctx, findOAuthTokenOwner, tokenHash).Scan(&owner.GrantID, &owner.ClientID, &kind, &owner.Used)
	if err != nil {
		return repository.TokenOwner{}, notFound(err)
	}

	owner.Kind = domain.TokenKind(kind)

	return owner, nil
}

const deleteOAuthTokenByHash = `
DELETE FROM oauth_tokens
WHERE token_hash = $1;
`

func (r *OAuthTokenRepository) DeleteByHash(ctx context.Context, tokenHash []byte) error {
	_, err := r.pool.Exec(ctx, deleteOAuthTokenByHash, tokenHash)
	if err != nil {
		return fmt.Errorf("delete oauth token: %w", err)
	}

	return nil
}

const deleteExpiredOAuthTokens = `
DELETE FROM oauth_tokens
WHERE expires_at < now() - interval '1 day';
`

func (r *OAuthTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, deleteExpiredOAuthTokens)
	if err != nil {
		return fmt.Errorf("delete expired oauth tokens: %w", err)
	}

	return nil
}
