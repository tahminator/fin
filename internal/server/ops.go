package server

import (
	"net/http"

	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/metrics"
)

func NewOpsMux(health *handler.HealthHandler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready)
	mux.Handle("GET /metrics", metrics.Handler())

	return mux
}
