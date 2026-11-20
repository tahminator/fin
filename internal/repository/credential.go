package repository

import (
	"context"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type CredentialRepository interface {
	Find(ctx context.Context, userID uuid.UUID, provider domain.Provider) (domain.Credential, error)
	Upsert(ctx context.Context, credential domain.Credential) error
}
