package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tahminator/fin/internal/metrics"
)

func TestRouteGroupOf(t *testing.T) {
	tests := []struct {
		path  string
		group metrics.RouteGroup
	}{
		{path: "/api", group: metrics.RouteGroupAPI},
		{path: "/api/x", group: metrics.RouteGroupAPI},
		{path: "/api/connections/123/sync", group: metrics.RouteGroupAPI},
		{path: "/apiary", group: metrics.RouteGroupStatic},
		{path: "/auth/google/login", group: metrics.RouteGroupAuth},
		{path: "/auth/google/callback", group: metrics.RouteGroupAuth},
		{path: "/openapi.json", group: metrics.RouteGroupDocs},
		{path: "/openapi.yaml", group: metrics.RouteGroupDocs},
		{path: "/swagger", group: metrics.RouteGroupDocs},
		{path: "/swagger/index.html", group: metrics.RouteGroupDocs},
		{path: "/swaggerish", group: metrics.RouteGroupStatic},
		{path: "/", group: metrics.RouteGroupStatic},
		{path: "/assets/x", group: metrics.RouteGroupStatic},
		{path: "/transactions", group: metrics.RouteGroupStatic},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := routeGroupOf(tc.path); got != tc.group {
				t.Errorf("routeGroupOf(%q) = %s, want %s", tc.path, got, tc.group)
			}
		})
	}
}

type servedRequest struct {
	group  metrics.RouteGroup
	method metrics.Method
	status int
}

type requestRecorder struct {
	metrics.Noop
	served []servedRequest
}

func (r *requestRecorder) HTTPRequestServed(group metrics.RouteGroup, method metrics.Method, status int, _ time.Duration) {
	r.served = append(r.served, servedRequest{group: group, method: method, status: status})
}

func TestObserveRecordsStatus(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
		want    servedRequest
	}{
		{
			name:   "explicit not found",
			method: http.MethodGet,
			path:   "/api/unknown",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			want: servedRequest{group: metrics.RouteGroupAPI, method: metrics.MethodGet, status: http.StatusNotFound},
		},
		{
			name:   "body without header is ok",
			method: http.MethodGet,
			path:   "/assets/app.js",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
			},
			want: servedRequest{group: metrics.RouteGroupStatic, method: metrics.MethodGet, status: http.StatusOK},
		},
		{
			name:   "first header wins",
			method: http.MethodPost,
			path:   "/auth/logout",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusSeeOther)
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: servedRequest{group: metrics.RouteGroupAuth, method: metrics.MethodPost, status: http.StatusSeeOther},
		},
		{
			name:   "header after body is ignored",
			method: http.MethodDelete,
			path:   "/api/connections/1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("done"))
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: servedRequest{group: metrics.RouteGroupAPI, method: metrics.MethodDelete, status: http.StatusOK},
		},
		{
			name:   "unusual method is other",
			method: http.MethodOptions,
			path:   "/openapi.json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			want: servedRequest{group: metrics.RouteGroupDocs, method: metrics.MethodOther, status: http.StatusNoContent},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &requestRecorder{}
			observed := Observe(tc.handler, recorder, slog.New(slog.DiscardHandler))
			rec := httptest.NewRecorder()

			observed.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))

			if len(recorder.served) != 1 {
				t.Fatalf("served = %+v, want exactly one record", recorder.served)
			}

			if recorder.served[0] != tc.want {
				t.Errorf("served = %+v, want %+v", recorder.served[0], tc.want)
			}
		})
	}
}
