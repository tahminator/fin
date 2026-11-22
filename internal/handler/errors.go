package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
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
	case errors.Is(err, domain.ErrThrottled):
		return throttled(err)
	case errors.Is(err, domain.ErrUnavailable):
		slog.WarnContext(ctx, "upstream unavailable", "err", err)

		return huma.Error503ServiceUnavailable("an upstream service is unavailable, try again later")
	}

	slog.ErrorContext(ctx, "request failed", "err", err)

	return huma.Error500InternalServerError("internal error")
}

func throttled(err error) error {
	var throttledErr *domain.ThrottledError

	if !errors.As(err, &throttledErr) {
		return huma.Error429TooManyRequests("too many requests, try again later")
	}

	seconds := int(math.Ceil(throttledErr.RetryAfter.Seconds()))
	headers := http.Header{}
	headers.Set("Retry-After", strconv.Itoa(max(seconds, 1)))

	return huma.ErrorWithHeaders(
		huma.Error429TooManyRequests(fmt.Sprintf("This bank was synced recently. Try again in %s.", waitText(max(seconds, 1)))),
		headers,
	)
}

func waitText(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := (seconds + 59) / 60
	if minutes == 1 {
		return "1 minute"
	}

	return fmt.Sprintf("%d minutes", minutes)
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
