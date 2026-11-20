package cardart

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/tahminator/fin/internal/domain"
)

func newTestOpenCard(t *testing.T, handler http.HandlerFunc) *OpenCard {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL + "/v1/")
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	return &OpenCard{httpClient: server.Client(), baseURL: baseURL}
}

func TestOpenCardSearchKeepsOnlyCardsWithArtwork(t *testing.T) {
	var gotQuery, gotClient string

	catalog := newTestOpenCard(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotClient = r.Header.Get("X-Client-Name")

		_, _ = w.Write([]byte(`{"data":[
			{"id":"absolute","name":"Absolute","issuer":"Chase","image":{"url":"https://cards.example/a.png"}},
			{"id":"relative","name":"Relative","issuer":null,"image":{"url":"/v1/assets/relative.png"}},
			{"id":"placeholder","name":"Placeholder","issuer":null,"image":{"url":"/v1/assets/default-card.webp"}},
			{"id":"insecure","name":"Insecure","issuer":null,"image":{"url":"http://cards.example/i.png"}},
			{"id":"no-image","name":"No image","issuer":null,"image":null},
			{"id":"malformed","name":42}
		]}`))
	})

	cards, err := catalog.Search(t.Context(), "freedom & flex")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if gotQuery != "freedom & flex" || gotClient != clientName {
		t.Errorf("request q=%q client=%q, want escaped query and client name", gotQuery, gotClient)
	}

	want := []string{"https://cards.example/a.png", catalog.baseURL.Scheme + "://" + catalog.baseURL.Host + "/v1/assets/relative.png"}

	if len(cards) != len(want) {
		t.Fatalf("cards = %+v, want %d cards with artwork", cards, len(want))
	}

	for i, card := range cards {
		if card.ImageURL != want[i] {
			t.Errorf("cards[%d].ImageURL = %q, want %q", i, card.ImageURL, want[i])
		}
	}
}

func TestOpenCardErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{name: "unknown card", status: http.StatusNotFound, body: `{"error":"not_found"}`, wantErr: domain.ErrNotFound},
		{name: "card without artwork", status: http.StatusOK, body: `{"id":"x","name":"X","image":null}`, wantErr: domain.ErrNotFound},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{}`, wantErr: domain.ErrUnavailable},
		{name: "invalid body", status: http.StatusOK, body: `not json`, wantErr: domain.ErrUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newTestOpenCard(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawPath != "/v1/cards/x%2Fy" {
					t.Errorf("path = %q, want escaped card id", r.URL.RawPath)
				}

				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			_, err := catalog.Card(t.Context(), "x/y")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Card error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
