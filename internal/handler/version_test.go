package handler_test

import (
	"net/http"
	"testing"

	"github.com/tahminator/fin/internal/handler"
)

func TestVersionRendersConfiguredValueOrNull(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "configured version", version: "1.2.3", want: `{"version":"1.2.3"}`},
		{name: "unset version is null", version: "", want: `{"version":null}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := newAPI(t, newUser(), handler.NewVersionHandler(tc.version))

			rec := api.Get("/api/version")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			if got := rec.Body.String(); got != tc.want+"\n" {
				t.Errorf("body = %q, want %q", got, tc.want+"\n")
			}
		})
	}
}
