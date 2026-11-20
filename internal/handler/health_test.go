package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tahminator/fin/internal/handler"
)

func TestHealthReady(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "database reachable is ready", err: nil, status: http.StatusOK, body: "ok\n"},
		{name: "database unreachable is unavailable", err: errors.New("connection refused"), status: http.StatusServiceUnavailable, body: "unavailable\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			health := handler.NewHealthHandler(fakePinger{err: tc.err})
			rec := httptest.NewRecorder()

			health.Ready(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d", rec.Code, tc.status)
			}

			if rec.Body.String() != tc.body {
				t.Errorf("body = %q, want %q", rec.Body.String(), tc.body)
			}
		})
	}
}

func TestHealthLiveIgnoresDatabase(t *testing.T) {
	health := handler.NewHealthHandler(fakePinger{err: errors.New("connection refused")})
	rec := httptest.NewRecorder()

	health.Live(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "ok\n" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok\n")
	}
}
