package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type CredentialRepository struct {
	pool *pgxpool.Pool
}

func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

const findCredential = `
SELECT
    c.user_id,
    c.provider,
    c.environment,
    c.client_id,
    c.secret,
    c.created_at
FROM user_provider_credentials AS c
WHERE c.user_id = $1
  AND c.provider = $2;
`

func (r *CredentialRepository) Find(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error) {
	rows, err := r.pool.Query(ctx, findCredential, userID, provider)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("query credential: %w", err)
	}

	credential, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Credential])
	if err != nil {
		return domain.Credential{}, notFound(err)
	}

	return credential, nil
}

const upsertCredential = `
INSERT INTO user_provider_credentials (
    user_id,
    provider,
    environment,
    client_id,
    secret
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, provider) DO UPDATE
SET environment = EXCLUDED.environment,
    client_id   = EXCLUDED.client_id,
    secret      = EXCLUDED.secret;
`

func (r *CredentialRepository) Upsert(ctx context.Context, credential domain.Credential) error {
	_, err := r.pool.Exec(
		ctx,
		upsertCredential,
		credential.UserID,
		credential.Provider,
		credential.Environment,
		credential.ClientID,
		credential.Secret,
	)
	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}

	return nil
}
