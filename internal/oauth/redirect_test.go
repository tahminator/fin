package oauth

import "testing"

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		uri   string
		valid bool
	}{
		{uri: "https://claude.ai/api/mcp/auth_callback", valid: true},
		{uri: "http://127.0.0.1:53682/callback", valid: true},
		{uri: "http://localhost/callback", valid: true},
		{uri: "http://[::1]:8080/cb", valid: true},
		{uri: "cursor://anysphere.cursor-mcp/oauth", valid: true},
		{uri: "http://example.com/callback", valid: false},
		{uri: "javascript:alert(1)", valid: false},
		{uri: "data:text/html,hi", valid: false},
		{uri: "file:///etc/passwd", valid: false},
		{uri: "https://example.com/cb#fragment", valid: false},
		{uri: "https://user@example.com/cb", valid: false},
		{uri: "/relative", valid: false},
		{uri: "https:///no-host", valid: false},
	}

	for _, tt := range tests {
		err := ValidateRedirectURI(tt.uri)
		if (err == nil) != tt.valid {
			t.Errorf("%s: err = %v, want valid %v", tt.uri, err, tt.valid)
		}
	}
}

func TestMatchRedirectURI(t *testing.T) {
	registered := []string{"https://app.example.com/cb", "http://127.0.0.1:9999/callback"}

	tests := []struct {
		uri   string
		match bool
	}{
		{uri: "https://app.example.com/cb", match: true},
		{uri: "https://app.example.com/cb/", match: false},
		{uri: "https://app.example.com/other", match: false},
		{uri: "http://127.0.0.1:4242/callback", match: true},
		{uri: "http://127.0.0.1:4242/elsewhere", match: false},
		{uri: "http://localhost:9999/callback", match: false},
		{uri: "https://evil.example.com/cb", match: false},
	}

	for _, tt := range tests {
		if got := MatchRedirectURI(registered, tt.uri); got != tt.match {
			t.Errorf("%s: match = %v, want %v", tt.uri, got, tt.match)
		}
	}
}
