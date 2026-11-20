package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/tahminator/fin/internal/server"
)

const (
	indexBody = "<!doctype html><title>fin</title>"
	assetBody = "console.log('fin')"
	swBody    = "self.addEventListener('fetch', () => {})"
	immutable = "public, max-age=31536000, immutable"
)

func newDist(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	files := map[string]string{
		"index.html":        indexBody,
		"assets/app-123.js": assetBody,
		"sw.js":             swBody,
	}

	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return dir
}

func TestSPA(t *testing.T) {
	spa := server.NewSPA(newDist(t))

	tests := []struct {
		name         string
		path         string
		body         string
		cacheControl string
	}{
		{name: "hashed asset is immutable", path: "/assets/app-123.js", body: assetBody, cacheControl: immutable},
		{name: "root serves index without caching", path: "/", body: indexBody, cacheControl: "no-cache"},
		{name: "client route falls back to index", path: "/transactions", body: indexBody, cacheControl: "no-cache"},
		{name: "nested client route falls back to index", path: "/connections/abc/settings", body: indexBody, cacheControl: "no-cache"},
		{name: "missing asset falls back to index", path: "/assets/missing.js", body: indexBody, cacheControl: "no-cache"},
		{name: "directory falls back to index", path: "/assets", body: indexBody, cacheControl: "no-cache"},
		{name: "service worker is never cached", path: "/sw.js", body: swBody, cacheControl: "no-cache"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			spa.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			if rec.Body.String() != tc.body {
				t.Errorf("body = %q, want %q", rec.Body.String(), tc.body)
			}

			if got := rec.Header().Get("Cache-Control"); got != tc.cacheControl {
				t.Errorf("Cache-Control = %q, want %q", got, tc.cacheControl)
			}
		})
	}
}

func TestSPARejectsNonReadMethods(t *testing.T) {
	spa := server.NewSPA(newDist(t))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()

			spa.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/transactions", nil))

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
				t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
			}

			if rec.Body.String() == indexBody {
				t.Error("served index for a non-read method")
			}
		})
	}
}

func TestSPAHeadServesHeadersWithoutBody(t *testing.T) {
	spa := server.NewSPA(newDist(t))
	rec := httptest.NewRecorder()

	spa.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/assets/app-123.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}

	if got := rec.Header().Get("Cache-Control"); got != immutable {
		t.Errorf("Cache-Control = %q, want %q", got, immutable)
	}
}
