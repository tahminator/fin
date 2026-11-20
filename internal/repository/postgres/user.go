package postgres

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const findUserByEmail = `
SELECT
    u.id,
    u.email,
    u.google_sub,
    u.name,
    u.created_at
FROM users AS u
WHERE lower(u.email) = lower($1);
`

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.findOne(ctx, findUserByEmail, email)
}

const findUserByID = `
SELECT
    u.id,
    u.email,
    u.google_sub,
    u.name,
    u.created_at
FROM users AS u
WHERE u.id = $1;
`

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return r.findOne(ctx, findUserByID, id)
}

const upsertUser = `
INSERT INTO users (email)
VALUES ($1)
ON CONFLICT ((lower(email))) DO UPDATE
SET email = EXCLUDED.email
RETURNING
    id,
    email,
    google_sub,
    name,
    created_at;
`

func (r *UserRepository) Upsert(ctx context.Context, email string) (domain.User, error) {
	return r.findOne(ctx, upsertUser, email)
}

const setUserGoogleSub = `
UPDATE users
SET google_sub = $2,
    name       = COALESCE($3, name)
WHERE id = $1;
`

func (r *UserRepository) SetGoogleSub(ctx context.Context, id uuid.UUID, googleSub string, name *string) error {
	tag, err := r.pool.Exec(ctx, setUserGoogleSub, id, googleSub, name)
	if err != nil {
		return fmt.Errorf("set google sub: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

const countUsers = `
SELECT COUNT(*) AS count
FROM users;
`

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int

	err := r.pool.QueryRow(ctx, countUsers).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}

func (r *UserRepository) findOne(ctx context.Context, query string, args ...any) (domain.User, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return domain.User{}, fmt.Errorf("query user: %w", err)
	}

	user, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		return domain.User{}, notFound(err)
	}

	return user, nil
}
