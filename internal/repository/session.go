package repository

import (
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error
	FindValidByTokenHash(ctx context.Context, tokenHash []byte) (domain.Session, error)
	Delete(ctx context.Context, tokenHash []byte) error
	DeleteExpired(ctx context.Context) error
}
