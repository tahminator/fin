package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/tahminator/fin/internal/domain"
)

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	return err
}
