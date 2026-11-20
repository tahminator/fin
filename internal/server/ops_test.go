package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tahminator/fin/internal/handler"
)

type fakePinger struct{}

func (fakePinger) Ping(context.Context) error {
	return nil
}

func serveOps(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))

	return rec
}

func TestWithOpsServesOpsRoutesBeforeApp(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		reachesApp bool
	}{
		{name: "liveness", method: http.MethodGet, path: "/healthz", reachesApp: false},
		{name: "readiness", method: http.MethodGet, path: "/readyz", reachesApp: false},
		{name: "metrics", method: http.MethodGet, path: "/metrics", reachesApp: false},
		{name: "client route", method: http.MethodGet, path: "/transactions", reachesApp: true},
		{name: "version route", method: http.MethodGet, path: "/version", reachesApp: true},
		{name: "non-get ops path", method: http.MethodPost, path: "/healthz", reachesApp: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusTeapot)
			})
			router := WithOps(app, handler.NewHealthHandler(fakePinger{}))

			rec := serveOps(t, router, tc.method, tc.path)

			if reached != tc.reachesApp {
				t.Fatalf("reached app = %v, want %v; status %d", reached, tc.reachesApp, rec.Code)
			}

			if !tc.reachesApp && rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestWithOpsKeepsProbesOutOfRequestObservation(t *testing.T) {
	recorder := &requestRecorder{}
	app := Observe(http.NotFoundHandler(), recorder, slog.New(slog.DiscardHandler))
	router := WithOps(app, handler.NewHealthHandler(fakePinger{}))

	serveOps(t, router, http.MethodGet, "/healthz")
	serveOps(t, router, http.MethodGet, "/metrics")
	serveOps(t, router, http.MethodGet, "/transactions")

	if len(recorder.served) != 1 {
		t.Fatalf("observed %d requests, want 1: %+v", len(recorder.served), recorder.served)
	}
}
