package server

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/tahminator/fin/internal/handler"
)

const (
	apiTitle   = "fin"
	apiVersion = "1.0.0"
)

func NewAPI(mux *http.ServeMux) huma.API {
	config := huma.DefaultConfig(apiTitle, apiVersion)
	config.OpenAPIPath = "/openapi"
	config.DocsPath = "/swagger"
	config.DocsRenderer = huma.DocsRendererSwaggerUI
	config.CreateHooks = nil
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"cookieAuth": {
			Type: "apiKey",
			In:   "cookie",
			Name: handler.SessionCookie,
		},
		handler.MCPOAuthScheme: handler.MCPOAuthSecurityScheme(),
	}

	return humago.New(mux, config)
}
