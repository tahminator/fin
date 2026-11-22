package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/tahminator/fin/internal/domain"
)

type grantKey struct{}

func grantFrom(ctx context.Context) (domain.AccessGrant, bool) {
	grant, ok := ctx.Value(grantKey{}).(domain.AccessGrant)

	return grant, ok
}

func requireBearer(deps Deps, next http.Handler) http.Handler {
	challenge := fmt.Sprintf(`Bearer resource_metadata=%q, scope=%q`, deps.Config.ResourceMetadataURL, string(domain.ScopeRead)+" "+string(domain.ScopeSync))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && !slices.Contains(deps.Config.AllowedOrigins, origin) {
			http.Error(w, "cross-origin requests are not allowed", http.StatusForbidden)

			return
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
			w.Header().Add("Vary", "Origin")
		}

		if r.URL.Query().Has("access_token") {
			w.Header().Set("WWW-Authenticate", challenge+`, error="invalid_request"`)
			http.Error(w, "access tokens are only accepted in the Authorization header", http.StatusBadRequest)

			return
		}

		token, ok := bearerToken(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", challenge)
			http.Error(w, "authorization required", http.StatusUnauthorized)

			return
		}

		grant, err := deps.Verifier.VerifyAccessToken(r.Context(), token)
		if err != nil {
			unauthorized(w, r, challenge, err)

			return
		}

		if !grant.Has(domain.ScopeRead) {
			w.Header().Set("WWW-Authenticate", challenge+`, error="insufficient_scope"`)
			http.Error(w, "insufficient scope", http.StatusForbidden)

			return
		}

		if !allowCall(w, r, deps, grant) {
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), grantKey{}, grant)))
	})
}

func unauthorized(w http.ResponseWriter, r *http.Request, challenge string, err error) {
	if !errors.Is(err, domain.ErrInvalidToken) {
		slog.ErrorContext(r.Context(), "verify mcp access token", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("WWW-Authenticate", challenge+`, error="invalid_token"`)
	http.Error(w, "invalid or expired access token", http.StatusUnauthorized)
}

func allowCall(w http.ResponseWriter, r *http.Request, deps Deps, grant domain.AccessGrant) bool {
	if deps.Limiter == nil {
		return true
	}

	decision, err := deps.Limiter.Allow(r.Context(), deps.Config.CallLimit, grant.GrantID.String())
	if err != nil {
		slog.ErrorContext(r.Context(), "mcp rate limit", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return false
	}

	if !decision.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(max(int(math.Ceil(decision.RetryAfter.Seconds())), 1)))
		http.Error(w, "too many requests", http.StatusTooManyRequests)

		return false
	}

	return true
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}

	return strings.TrimSpace(token), true
}
