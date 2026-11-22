package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type OAuthAuthorizationRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthAuthorizationRepository(pool *pgxpool.Pool) *OAuthAuthorizationRepository {
	return &OAuthAuthorizationRepository{pool: pool}
}

const createOAuthAuthorization = `
INSERT INTO oauth_authorizations (
    request_token_hash,
    client_id,
    redirect_uri,
    scope,
    resource,
    state,
    code_challenge,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
`

func (r *OAuthAuthorizationRepository) Create(ctx context.Context, authorization repository.NewAuthorization) error {
	_, err := r.pool.Exec(
		ctx,
		createOAuthAuthorization,
		authorization.RequestTokenHash,
		authorization.ClientID,
		authorization.RedirectURI,
		authorization.Scope,
		authorization.Resource,
		authorization.State,
		authorization.CodeChallenge,
		authorization.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create oauth authorization: %w", err)
	}

	return nil
}

const findPendingOAuthAuthorization = `
SELECT
    a.id,
    a.client_id,
    a.user_id,
    a.redirect_uri,
    a.scope,
    a.resource,
    a.state,
    a.code_challenge,
    a.grant_id,
    a.expires_at
FROM oauth_authorizations AS a
WHERE a.request_token_hash = $1
  AND a.user_id IS NULL
  AND a.expires_at > now();
`

func (r *OAuthAuthorizationRepository) FindPending(ctx context.Context, requestTokenHash []byte) (domain.OAuthAuthorization, error) {
	rows, err := r.pool.Query(ctx, findPendingOAuthAuthorization, requestTokenHash)
	if err != nil {
		return domain.OAuthAuthorization{}, fmt.Errorf("query oauth authorization: %w", err)
	}

	authorization, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.OAuthAuthorization])
	if err != nil {
		return domain.OAuthAuthorization{}, notFound(err)
	}

	return authorization, nil
}

const approveOAuthAuthorization = `
UPDATE oauth_authorizations
SET user_id         = $2,
    scope           = $3,
    grant_id        = $4,
    code_hash       = $5,
    code_expires_at = $6
WHERE id = $1
  AND user_id IS NULL
  AND expires_at > now();
`

func (r *OAuthAuthorizationRepository) Approve(ctx context.Context, approval repository.AuthorizationApproval) error {
	tag, err := r.pool.Exec(
		ctx,
		approveOAuthAuthorization,
		approval.ID,
		approval.UserID,
		approval.Scope,
		approval.GrantID,
		approval.CodeHash,
		approval.CodeExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("approve oauth authorization: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

const deletePendingOAuthAuthorization = `
DELETE FROM oauth_authorizations
WHERE id = $1
  AND user_id IS NULL;
`

func (r *OAuthAuthorizationRepository) DeletePending(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, deletePendingOAuthAuthorization, id)
	if err != nil {
		return fmt.Errorf("delete oauth authorization: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

const consumeOAuthCode = `
UPDATE oauth_authorizations
SET consumed_at = now()
WHERE code_hash = $1
  AND consumed_at IS NULL
  AND code_expires_at > now()
RETURNING
    id,
    client_id,
    user_id,
    redirect_uri,
    scope,
    resource,
    state,
    code_challenge,
    grant_id,
    expires_at;
`

func (r *OAuthAuthorizationRepository) ConsumeCode(ctx context.Context, codeHash []byte) (domain.OAuthAuthorization, error) {
	rows, err := r.pool.Query(ctx, consumeOAuthCode, codeHash)
	if err != nil {
		return domain.OAuthAuthorization{}, fmt.Errorf("consume oauth code: %w", err)
	}

	authorization, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.OAuthAuthorization])
	if err != nil {
		return domain.OAuthAuthorization{}, notFound(err)
	}

	return authorization, nil
}

const findConsumedOAuthCodeGrant = `
SELECT a.grant_id
FROM oauth_authorizations AS a
WHERE a.code_hash = $1
  AND a.consumed_at IS NOT NULL
  AND a.grant_id IS NOT NULL;
`

func (r *OAuthAuthorizationRepository) FindConsumedGrantID(ctx context.Context, codeHash []byte) (uuid.UUID, error) {
	var grantID uuid.UUID

	err := r.pool.QueryRow(ctx, findConsumedOAuthCodeGrant, codeHash).Scan(&grantID)
	if err != nil {
		return uuid.UUID{}, notFound(err)
	}

	return grantID, nil
}

const deleteExpiredOAuthAuthorizations = `
DELETE FROM oauth_authorizations
WHERE expires_at < now() - interval '1 day';
`

func (r *OAuthAuthorizationRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, deleteExpiredOAuthAuthorizations)
	if err != nil {
		return fmt.Errorf("delete expired oauth authorizations: %w", err)
	}

	return nil
}
