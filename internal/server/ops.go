package server

import (
	"net/http"

	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
)

func WithOps(app http.Handler, health *handler.HealthHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/", app)

	return mux
}
