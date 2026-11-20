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

type ConnectionRepository struct {
	pool *pgxpool.Pool
}

func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

const createConnection = `
INSERT INTO connections AS c (
    user_id,
    provider,
    external_id,
    access_token_ciphertext,
    institution_id,
    institution_name,
    institution_url,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at;
`

func (r *ConnectionRepository) Create(ctx context.Context, connection domain.Connection) (domain.Connection, error) {
	rows, err := r.pool.Query(
		ctx,
		createConnection,
		connection.UserID,
		connection.Provider,
		connection.ExternalID,
		connection.AccessTokenCiphertext,
		connection.InstitutionID,
		connection.InstitutionName,
		connection.InstitutionURL,
		connection.Status,
	)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("create connection: %w", err)
	}

	created, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Connection])
	if err != nil {
		return domain.Connection{}, fmt.Errorf("create connection: %w", err)
	}

	return created, nil
}

const getConnection = `
SELECT
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at
FROM connections AS c
WHERE c.user_id = $1
  AND c.id = $2;
`

func (r *ConnectionRepository) Get(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error) {
	return r.findOne(ctx, getConnection, userID, id)
}

const findConnectionByID = `
SELECT
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at
FROM connections AS c
WHERE c.id = $1;
`

func (r *ConnectionRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Connection, error) {
	return r.findOne(ctx, findConnectionByID, id)
}

const listConnectionsByUser = `
SELECT
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at
FROM connections AS c
WHERE c.user_id = $1
ORDER BY c.created_at DESC, c.id DESC
LIMIT $2
OFFSET $3;
`

const countConnectionsByUser = `
SELECT COUNT(*) AS count
FROM connections AS c
WHERE c.user_id = $1;
`

func (r *ConnectionRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Connection, int, error) {
	connections, err := r.list(ctx, listConnectionsByUser, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := count(ctx, r.pool, countConnectionsByUser, userID)
	if err != nil {
		return nil, 0, err
	}

	return connections, total, nil
}

const listConnectionsByStatus = `
SELECT
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at
FROM connections AS c
WHERE c.status = $1
ORDER BY c.id;
`

func (r *ConnectionRepository) ListByStatus(ctx context.Context, status domain.ConnectionStatus) ([]domain.Connection, error) {
	return r.list(ctx, listConnectionsByStatus, status)
}

const listConnectionsByUserAndStatus = `
SELECT
    c.id,
    c.user_id,
    c.provider,
    c.external_id,
    c.access_token_ciphertext,
    c.institution_id,
    c.institution_name,
    c.institution_url,
    c.status,
    c.status_changed_at,
    c.sync_cursor,
    c.last_synced_at,
    c.last_error,
    c.created_at,
    c.updated_at
FROM connections AS c
WHERE c.user_id = $1
  AND c.status = $2
ORDER BY c.status_changed_at DESC, c.id DESC
LIMIT $3
OFFSET $4;
`

const countConnectionsByUserAndStatus = `
SELECT COUNT(*) AS count
FROM connections AS c
WHERE c.user_id = $1
  AND c.status = $2;
`

func (r *ConnectionRepository) ListByUserAndStatus(
	ctx context.Context,
	userID uuid.UUID,
	status domain.ConnectionStatus,
	limit, offset int,
) ([]domain.Connection, int, error) {
	connections, err := r.list(ctx, listConnectionsByUserAndStatus, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := count(ctx, r.pool, countConnectionsByUserAndStatus, userID, status)
	if err != nil {
		return nil, 0, err
	}

	return connections, total, nil
}

const updateConnectionSyncState = `
UPDATE connections
SET sync_cursor    = $3,
    last_synced_at = $4,
    last_error     = NULL,
    updated_at     = now()
WHERE user_id = $1
  AND id = $2;
`

func (r *ConnectionRepository) UpdateSyncState(ctx context.Context, userID, id uuid.UUID, cursor string, syncedAt time.Time) error {
	return r.update(ctx, updateConnectionSyncState, userID, id, cursor, syncedAt)
}

const updateConnectionLastError = `
UPDATE connections
SET last_error = $3,
    updated_at = now()
WHERE user_id = $1
  AND id = $2;
`

func (r *ConnectionRepository) UpdateLastError(ctx context.Context, userID, id uuid.UUID, lastError string) error {
	return r.update(ctx, updateConnectionLastError, userID, id, lastError)
}

const updateConnectionStatus = `
UPDATE connections
SET status            = $3,
    status_changed_at = CASE WHEN status = $3 THEN status_changed_at ELSE now() END,
    last_error        = $4,
    updated_at        = now()
WHERE user_id = $1
  AND id = $2;
`

func (r *ConnectionRepository) UpdateStatus(
	ctx context.Context,
	userID, id uuid.UUID,
	status domain.ConnectionStatus,
	lastError *string,
) error {
	return r.update(ctx, updateConnectionStatus, userID, id, status, lastError)
}

const updateConnectionInstitutionURL = `
UPDATE connections
SET institution_url = $3,
    updated_at      = now()
WHERE user_id = $1
  AND id = $2;
`

func (r *ConnectionRepository) UpdateInstitutionURL(ctx context.Context, userID, id uuid.UUID, url string) error {
	return r.update(ctx, updateConnectionInstitutionURL, userID, id, url)
}

const deleteConnection = `
DELETE FROM connections
WHERE user_id = $1
  AND id = $2;
`

func (r *ConnectionRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return r.update(ctx, deleteConnection, userID, id)
}

func (r *ConnectionRepository) findOne(ctx context.Context, query string, args ...any) (domain.Connection, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("query connection: %w", err)
	}

	connection, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Connection])
	if err != nil {
		return domain.Connection{}, notFound(err)
	}

	return connection, nil
}

func (r *ConnectionRepository) list(ctx context.Context, query string, args ...any) ([]domain.Connection, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}

	connections, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Connection])
	if err != nil {
		return nil, fmt.Errorf("collect connections: %w", err)
	}

	return connections, nil
}

func (r *ConnectionRepository) update(ctx context.Context, query string, args ...any) error {
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update connection: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
