package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/service"
)

type sessionResolver interface {
	ResolveSession(ctx context.Context, token string) (domain.User, error)
}

func RequireSession(api huma.API, sessions sessionResolver) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		cookie, err := huma.ReadCookie(ctx, handler.SessionCookie)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "sign in required")

			return
		}

		user, err := sessions.ResolveSession(ctx.Context(), cookie.Value)
		if errors.Is(err, service.ErrUnauthenticated) {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "sign in required")

			return
		}

		if err != nil {
			slog.ErrorContext(ctx.Context(), "resolve session", "err", err)
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "internal error")

			return
		}

		next(handler.WithSession(ctx, user, cookie.Value))
	}
}

func Observe(next http.Handler, recorder metrics.Recorder, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(writer, r)

		elapsed := time.Since(started)

		recorder.HTTPRequestServed(routeGroupOf(r.URL.Path), metrics.MethodOf(r.Method), writer.status, elapsed)
		logger.InfoContext(
			r.Context(),
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", writer.status,
			"duration_ms", elapsed.Milliseconds(),
		)
	})
}

func routeGroupOf(path string) metrics.RouteGroup {
	switch {
	case path == "/api" || strings.HasPrefix(path, "/api/") || path == "/version":
		return metrics.RouteGroupAPI
	case strings.HasPrefix(path, "/auth/"):
		return metrics.RouteGroupAuth
	case strings.HasPrefix(path, "/openapi") || path == "/swagger" || strings.HasPrefix(path, "/swagger/"):
		return metrics.RouteGroupDocs
	}

	return metrics.RouteGroupStatic
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	w.wroteHeader = true

	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
