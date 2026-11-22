package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type OAuthClientRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthClientRepository(pool *pgxpool.Pool) *OAuthClientRepository {
	return &OAuthClientRepository{pool: pool}
}

const findOAuthClient = `
SELECT
    c.id,
    c.source,
    c.client_name,
    c.client_uri,
    c.logo_uri,
    c.redirect_uris,
    c.token_endpoint_auth_method,
    c.client_secret_hash,
    c.metadata_expires_at,
    c.created_at
FROM oauth_clients AS c
WHERE c.id = $1;
`

func (r *OAuthClientRepository) Find(ctx context.Context, id string) (domain.OAuthClient, error) {
	rows, err := r.pool.Query(ctx, findOAuthClient, id)
	if err != nil {
		return domain.OAuthClient{}, fmt.Errorf("query oauth client: %w", err)
	}

	client, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.OAuthClient])
	if err != nil {
		return domain.OAuthClient{}, notFound(err)
	}

	return client, nil
}

const createOAuthClient = `
INSERT INTO oauth_clients (
    id,
    source,
    client_name,
    client_uri,
    logo_uri,
    redirect_uris,
    token_endpoint_auth_method,
    client_secret_hash,
    metadata_expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
`

func (r *OAuthClientRepository) Create(ctx context.Context, client domain.OAuthClient) error {
	_, err := r.pool.Exec(ctx, createOAuthClient, clientArgs(client)...)
	if err != nil {
		return fmt.Errorf("create oauth client: %w", err)
	}

	return nil
}

const upsertOAuthClient = `
INSERT INTO oauth_clients (
    id,
    source,
    client_name,
    client_uri,
    logo_uri,
    redirect_uris,
    token_endpoint_auth_method,
    client_secret_hash,
    metadata_expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE
SET client_name                = EXCLUDED.client_name,
    client_uri                 = EXCLUDED.client_uri,
    logo_uri                   = EXCLUDED.logo_uri,
    redirect_uris              = EXCLUDED.redirect_uris,
    token_endpoint_auth_method = EXCLUDED.token_endpoint_auth_method,
    metadata_expires_at        = EXCLUDED.metadata_expires_at,
    updated_at                 = now()
WHERE oauth_clients.source = EXCLUDED.source;
`

func (r *OAuthClientRepository) Upsert(ctx context.Context, client domain.OAuthClient) error {
	_, err := r.pool.Exec(ctx, upsertOAuthClient, clientArgs(client)...)
	if err != nil {
		return fmt.Errorf("upsert oauth client: %w", err)
	}

	return nil
}

const deleteUnusedRegisteredClients = `
DELETE FROM oauth_clients AS c
WHERE c.source = 'registered'
  AND c.created_at < $1
  AND NOT EXISTS (
      SELECT 1
      FROM oauth_grants AS g
      WHERE g.client_id = c.id
  );
`

func (r *OAuthClientRepository) DeleteUnusedRegistered(ctx context.Context, createdBefore time.Time) error {
	_, err := r.pool.Exec(ctx, deleteUnusedRegisteredClients, createdBefore)
	if err != nil {
		return fmt.Errorf("delete unused oauth clients: %w", err)
	}

	return nil
}

func clientArgs(client domain.OAuthClient) []any {
	return []any{
		client.ID,
		string(client.Source),
		client.ClientName,
		client.ClientURI,
		client.LogoURI,
		client.RedirectURIs,
		string(client.TokenEndpointAuthMethod),
		client.ClientSecretHash,
		client.MetadataExpiresAt,
	}
}
