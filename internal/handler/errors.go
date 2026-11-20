package handler

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
)

func problem(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return huma.Error404NotFound("resource not found")
	case errors.Is(err, domain.ErrNotAllowed):
		return huma.Error403Forbidden(detailOf(err, domain.ErrNotAllowed))
	case errors.Is(err, domain.ErrConflict):
		return huma.Error409Conflict(detailOf(err, domain.ErrConflict))
	}

	slog.ErrorContext(ctx, "request failed", "err", err)

	return huma.Error500InternalServerError("internal error")
}

func detailOf(err, sentinel error) string {
	return strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, huma.Error404NotFound("resource not found")
	}

	return id, nil
}
