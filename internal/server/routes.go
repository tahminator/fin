package server

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/handler"
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

	mux.Handle("/api/", http.NotFoundHandler())

	if h.SPA != nil {
		mux.Handle("/", h.SPA)
	}
}
