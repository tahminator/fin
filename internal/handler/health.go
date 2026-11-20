package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/tahminator/fin/internal/repository"
)

const readinessTimeout = 2 * time.Second

type HealthHandler struct {
	pinger repository.Pinger
}

func NewHealthHandler(pinger repository.Pinger) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.pinger.Ping(ctx); err != nil {
		slog.WarnContext(ctx, "readiness check failed", "err", err)
		writeStatus(w, http.StatusServiceUnavailable, "unavailable")

		return
	}

	writeStatus(w, http.StatusOK, "ok")
}

func writeStatus(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)

	_, _ = w.Write([]byte(body + "\n"))
}
