package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	Upsert(ctx context.Context, email string) (domain.User, error)
	SetGoogleSub(ctx context.Context, id uuid.UUID, googleSub string, name *string) error
	Count(ctx context.Context) (int, error)
}
