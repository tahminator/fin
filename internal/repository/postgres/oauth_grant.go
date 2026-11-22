package postgres

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type OAuthGrantRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthGrantRepository(pool *pgxpool.Pool) *OAuthGrantRepository {
	return &OAuthGrantRepository{pool: pool}
}

const upsertOAuthGrant = `
INSERT INTO oauth_grants (
    user_id,
    client_id,
    scope,
    resource,
    expires_at
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, client_id) DO UPDATE
SET scope      = EXCLUDED.scope,
    resource   = EXCLUDED.resource,
    expires_at = EXCLUDED.expires_at
RETURNING id;
`

func (r *OAuthGrantRepository) Upsert(ctx context.Context, userID uuid.UUID, clientID, scope, resource string, expiresAt time.Time) (uuid.UUID, error) {
	var id uuid.UUID

	err := r.pool.QueryRow(ctx, upsertOAuthGrant, userID, clientID, scope, resource, expiresAt).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("upsert oauth grant: %w", err)
	}

	return id, nil
}

const listOAuthGrantsByUser = `
SELECT
    g.id,
    g.user_id,
    g.client_id,
    c.client_name,
    c.source      AS client_source,
    g.scope,
    g.resource,
    g.created_at,
    g.expires_at,
    g.last_used_at
FROM oauth_grants AS g
JOIN oauth_clients AS c ON c.id = g.client_id
WHERE g.user_id = $1
  AND g.expires_at > now()
ORDER BY g.created_at DESC, g.id DESC
LIMIT $2
OFFSET $3;
`

const countOAuthGrantsByUser = `
SELECT COUNT(*)
FROM oauth_grants AS g
WHERE g.user_id = $1
  AND g.expires_at > now();
`

func (r *OAuthGrantRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.OAuthGrant, int, error) {
	rows, err := r.pool.Query(ctx, listOAuthGrantsByUser, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query oauth grants: %w", err)
	}

	grants, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.OAuthGrant])
	if err != nil {
		return nil, 0, fmt.Errorf("scan oauth grants: %w", err)
	}

	total, err := count(ctx, r.pool, countOAuthGrantsByUser, userID)
	if err != nil {
		return nil, 0, err
	}

	return grants, total, nil
}

const deleteOAuthGrant = `
DELETE FROM oauth_grants
WHERE user_id = $1
  AND id = $2;
`

func (r *OAuthGrantRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, deleteOAuthGrant, userID, id)
	if err != nil {
		return fmt.Errorf("delete oauth grant: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

const deleteOAuthGrantByID = `
DELETE FROM oauth_grants
WHERE id = $1;
`

func (r *OAuthGrantRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, deleteOAuthGrantByID, id)
	if err != nil {
		return fmt.Errorf("delete oauth grant: %w", err)
	}

	return nil
}

const touchOAuthGrant = `
UPDATE oauth_grants
SET last_used_at = now()
WHERE id = $1
  AND (last_used_at IS NULL OR last_used_at < now() - interval '5 minutes');
`

func (r *OAuthGrantRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, touchOAuthGrant, id)
	if err != nil {
		return fmt.Errorf("touch oauth grant: %w", err)
	}

	return nil
}
