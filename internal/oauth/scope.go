package oauth

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tahminator/fin/internal/domain"
)

func ParseScopes(raw string) ([]domain.Scope, error) {
	var scopes []domain.Scope

	for field := range strings.FieldsSeq(raw) {
		scope := domain.Scope(field)

		if !scope.Valid() {
			return nil, fmt.Errorf("unknown scope %q", field)
		}

		if !slices.Contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}

	return sortScopes(scopes), nil
}

func FormatScopes(scopes []domain.Scope) string {
	names := make([]string, 0, len(scopes))

	for _, scope := range sortScopes(scopes) {
		names = append(names, string(scope))
	}

	return strings.Join(names, " ")
}

func Subset(scopes, of []domain.Scope) bool {
	for _, scope := range scopes {
		if !slices.Contains(of, scope) {
			return false
		}
	}

	return true
}

func sortScopes(scopes []domain.Scope) []domain.Scope {
	sorted := make([]domain.Scope, 0, len(scopes))

	for _, scope := range domain.AllScopes() {
		if slices.Contains(scopes, scope) {
			sorted = append(sorted, scope)
		}
	}

	return sorted
}
