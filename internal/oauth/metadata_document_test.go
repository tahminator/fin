package oauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

func TestPublicAddressesOnly(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "::1", "fc00::1", "fe80::1", "0.0.0.0"}
	allowed := []string{"1.1.1.1", "160.79.104.10", "2606:4700::1111"}

	for _, raw := range blocked {
		if PublicAddressesOnly(netip.MustParseAddr(raw)) {
			t.Errorf("%s allowed, want blocked", raw)
		}
	}

	for _, raw := range allowed {
		if !PublicAddressesOnly(netip.MustParseAddr(raw)) {
			t.Errorf("%s blocked, want allowed", raw)
		}
	}
}

func TestIsMetadataDocumentClientID(t *testing.T) {
	tests := map[string]bool{
		"https://claude.ai/oauth/mcp-client.json": true,
		"https://claude.ai":                       false,
		"https://claude.ai/":                      false,
		"http://claude.ai/client.json":            false,
		"fin_client_abc":                          false,
	}

	for id, want := range tests {
		if got := IsMetadataDocumentClientID(id); got != want {
			t.Errorf("%s: %v, want %v", id, got, want)
		}
	}
}

func TestFetchRefusesLoopbackServers(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	fetcher := NewMetadataDocumentFetcher(PublicAddressesOnly)

	_, err := fetcher.Fetch(t.Context(), server.URL+"/client.json")

	var oauthErr *Error
	if !errors.As(err, &oauthErr) || oauthErr.Code != ErrInvalidClient {
		t.Fatalf("Fetch error = %v, want invalid_client", err)
	}
}

func TestClientFromMetadataDocument(t *testing.T) {
	id := "https://app.example.com/client.json"
	valid := ClientMetadata{ClientID: id, ClientName: "App", RedirectURIs: []string{"https://app.example.com/cb"}}

	client, err := ClientFromMetadataDocument(id, valid)
	if err != nil {
		t.Fatalf("valid document rejected: %v", err)
	}

	if client.Source != domain.ClientSourceMetadataDocument || client.TokenEndpointAuthMethod != domain.TokenAuthNone {
		t.Errorf("client = %+v", client)
	}

	invalid := map[string]ClientMetadata{
		"mismatched id":     {ClientID: "https://other.example.com/client.json", ClientName: "App", RedirectURIs: valid.RedirectURIs},
		"no name":           {ClientID: id, RedirectURIs: valid.RedirectURIs},
		"no redirects":      {ClientID: id, ClientName: "App"},
		"bad redirect":      {ClientID: id, ClientName: "App", RedirectURIs: []string{"http://example.com/cb"}},
		"confidential auth": {ClientID: id, ClientName: "App", RedirectURIs: valid.RedirectURIs, TokenEndpointAuthMethod: "client_secret_basic"},
	}

	for name, metadata := range invalid {
		if _, err := ClientFromMetadataDocument(id, metadata); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestMaxAgeOf(t *testing.T) {
	tests := map[string]time.Duration{
		"":                       metadataDocumentMaxAge,
		"max-age=3600":           time.Hour,
		"public, max-age=10":     metadataDocumentMinAge,
		"max-age=999999":         metadataDocumentMaxAge,
		"no-store, max-age=oops": metadataDocumentMaxAge,
	}

	for header, want := range tests {
		if got := maxAgeOf(header); got != want {
			t.Errorf("%q: %s, want %s", header, got, want)
		}
	}
}
