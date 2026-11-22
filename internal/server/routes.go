package server

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/mcpserver"
)

type Handlers struct {
	Auth           *handler.AuthHandler
	Summary        *handler.SummaryHandler
	Connections    *handler.ConnectionHandler
	Accounts       *handler.AccountHandler
	CardArt        *handler.CardArtHandler
	Transactions   *handler.TransactionHandler
	Notifications  *handler.NotificationHandler
	Version        *handler.VersionHandler
	OAuth          *handler.OAuthHandler
	OAuthConsent   *handler.OAuthConsentHandler
	MCP            http.Handler
	RequireSession func(ctx huma.Context, next func(huma.Context))
	SPA            http.Handler
}

func RegisterRoutes(mux *http.ServeMux, api huma.API, h Handlers) {
	mux.HandleFunc("GET /auth/google/login", h.Auth.Login)
	mux.HandleFunc("GET /auth/google/callback", h.Auth.Callback)

	h.Version.Register(api)

	authed := huma.NewGroup(api, "/api")
	authed.UseMiddleware(h.RequireSession)
	h.Auth.Register(authed)
	h.Summary.Register(authed)
	h.Connections.Register(authed)
	h.Accounts.Register(authed)
	h.CardArt.Register(authed)
	h.Transactions.Register(authed)
	h.Notifications.Register(authed)
	h.OAuthConsent.Register(authed)

	handler.DocumentOAuth(api)
	mcpserver.Document(api)

	if h.OAuth != nil {
		registerOAuthRoutes(mux, h.OAuth)
	}

	if h.MCP != nil {
		mux.Handle("POST /mcp", h.MCP)
		mux.Handle("OPTIONS /mcp", http.HandlerFunc(h.OAuth.Preflight))
		mux.Handle("/mcp", methodNotAllowed("POST"))
	}

	mux.Handle("/api/", http.NotFoundHandler())
	mux.Handle("/oauth/", http.NotFoundHandler())
	mux.Handle("/.well-known/", http.NotFoundHandler())

	if h.SPA != nil {
		mux.Handle("/", h.SPA)
	}
}

func registerOAuthRoutes(mux *http.ServeMux, oauth *handler.OAuthHandler) {
	mux.HandleFunc("GET "+handler.ResourceMetadataPath, oauth.ResourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", oauth.ResourceMetadata)
	mux.HandleFunc("GET "+handler.AuthorizationServerPath, oauth.ServerMetadata)
	mux.HandleFunc("GET /.well-known/openid-configuration", oauth.ServerMetadata)

	mux.HandleFunc("GET /oauth/authorize", oauth.Authorize)
	mux.HandleFunc("POST /oauth/token", oauth.Token)
	mux.HandleFunc("POST /oauth/register", oauth.Register)
	mux.HandleFunc("POST /oauth/revoke", oauth.Revoke)

	for _, path := range []string{"/oauth/token", "/oauth/register", "/oauth/revoke", handler.ResourceMetadataPath, handler.AuthorizationServerPath} {
		mux.HandleFunc("OPTIONS "+path, oauth.Preflight)
	}
}

func methodNotAllowed(allow string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
}
