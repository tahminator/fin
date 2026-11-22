package oauth

import (
	"slices"
	"testing"

	"github.com/tahminator/fin/internal/domain"
)

func TestParseScopesNormalizesOrderAndDuplicates(t *testing.T) {
	scopes, err := ParseScopes("fin:sync  fin:read fin:sync")
	if err != nil {
		t.Fatalf("ParseScopes: %v", err)
	}

	if !slices.Equal(scopes, []domain.Scope{domain.ScopeRead, domain.ScopeSync}) {
		t.Errorf("scopes = %v", scopes)
	}

	if FormatScopes(scopes) != "fin:read fin:sync" {
		t.Errorf("FormatScopes = %q", FormatScopes(scopes))
	}
}

func TestParseScopesRejectsUnknown(t *testing.T) {
	if _, err := ParseScopes("fin:read admin"); err == nil {
		t.Fatal("unknown scope accepted")
	}
}

func TestSubset(t *testing.T) {
	read := []domain.Scope{domain.ScopeRead}
	both := []domain.Scope{domain.ScopeRead, domain.ScopeSync}

	if !Subset(read, both) || Subset(both, read) {
		t.Error("Subset gave the wrong answer")
	}
}
