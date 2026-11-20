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

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

const createSession = `
INSERT INTO sessions (
    user_id,
    token_hash,
    expires_at
)
VALUES ($1, $2, $3);
`

func (r *SessionRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, createSession, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

const findValidSessionByTokenHash = `
SELECT
    s.id,
    s.user_id,
    s.token_hash,
    s.created_at,
    s.expires_at
FROM sessions AS s
WHERE s.token_hash = $1
  AND s.expires_at > now();
`

func (r *SessionRepository) FindValidByTokenHash(ctx context.Context, tokenHash []byte) (domain.Session, error) {
	rows, err := r.pool.Query(ctx, findValidSessionByTokenHash, tokenHash)
	if err != nil {
		return domain.Session{}, fmt.Errorf("query session: %w", err)
	}

	session, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Session])
	if err != nil {
		return domain.Session{}, notFound(err)
	}

	return session, nil
}

const deleteSession = `
DELETE FROM sessions
WHERE token_hash = $1;
`

func (r *SessionRepository) Delete(ctx context.Context, tokenHash []byte) error {
	_, err := r.pool.Exec(ctx, deleteSession, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

const deleteExpiredSessions = `
DELETE FROM sessions
WHERE expires_at <= now();
`

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, deleteExpiredSessions)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}
