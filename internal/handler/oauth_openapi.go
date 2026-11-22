package handler

import (
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
)

const (
	MCPOAuthScheme = "mcpOAuth"
	oauthTag       = "oauth"
	jsonMediaType  = "application/json"
	formMediaType  = "application/x-www-form-urlencoded"
)

func MCPOAuthSecurityScheme() *huma.SecurityScheme {
	scopes := map[string]string{
		string(domain.ScopeRead): "Read connections, accounts, balances and transactions",
		string(domain.ScopeSync): "Refresh already-linked connections (subject to the sync cooldown)",
	}

	return &huma.SecurityScheme{
		Type:        "oauth2",
		Description: "OAuth 2.1 authorization code flow with PKCE (S256). Clients identify themselves with a client metadata document URL or register at /oauth/register.",
		Flows: &huma.OAuthFlows{
			AuthorizationCode: &huma.OAuthFlow{
				AuthorizationURL: authorizeEndpointPath,
				TokenURL:         tokenEndpointPath,
				RefreshURL:       tokenEndpointPath,
				Scopes:           scopes,
			},
		},
	}
}

func DocumentOAuth(api huma.API) {
	oauthError := schemaFor[OAuthErrorResponse](api)
	errorResponses := map[string]*huma.Response{
		"400": jsonResponse("RFC 6749 error", oauthError),
		"401": jsonResponse("Client authentication failed", oauthError),
		"429": jsonResponse("Rate limited; see Retry-After", oauthError),
	}

	add := func(op *huma.Operation, responses map[string]*huma.Response) {
		for status, response := range errorResponses {
			if _, ok := responses[status]; !ok {
				responses[status] = response
			}
		}

		op.Tags = []string{oauthTag}
		op.Responses = responses
		api.OpenAPI().AddOperation(op)
	}

	add(&huma.Operation{
		OperationID: "get-oauth-protected-resource-metadata",
		Method:      http.MethodGet,
		Path:        ResourceMetadataPath,
		Summary:     "OAuth protected resource metadata for /mcp (RFC 9728)",
	}, map[string]*huma.Response{"200": jsonResponse("Protected resource metadata", schemaFor[ProtectedResourceMetadata](api))})

	add(&huma.Operation{
		OperationID: "get-oauth-authorization-server-metadata",
		Method:      http.MethodGet,
		Path:        AuthorizationServerPath,
		Summary:     "OAuth authorization server metadata (RFC 8414); also served at /.well-known/openid-configuration",
	}, map[string]*huma.Response{"200": jsonResponse("Authorization server metadata", schemaFor[AuthorizationServerMetadata](api))})

	add(&huma.Operation{
		OperationID: "oauth-authorize",
		Method:      http.MethodGet,
		Path:        authorizeEndpointPath,
		Summary:     "Start an authorization request; redirects the browser to the consent page",
		Parameters: []*huma.Param{
			queryParam("response_type", "Must be code", true),
			queryParam("client_id", "Client metadata document URL or registered client ID", true),
			queryParam("redirect_uri", "One of the client's registered redirect URIs", false),
			queryParam("code_challenge", "PKCE S256 challenge", true),
			queryParam("code_challenge_method", "Must be S256", true),
			queryParam("state", "Opaque value returned to the client", false),
			queryParam("scope", "Space-separated: fin:read, fin:sync", false),
			queryParam("resource", "Must be PUBLIC_URL/mcp (RFC 8707)", false),
		},
	}, map[string]*huma.Response{
		"303": {Description: "Redirect to /consent?request=…"},
		"302": {Description: "Redirect back to the client with an error"},
	})

	add(&huma.Operation{
		OperationID: "oauth-token",
		Method:      http.MethodPost,
		Path:        tokenEndpointPath,
		Summary:     "Exchange an authorization code or rotate a refresh token",
		RequestBody: formBody(schemaFor[TokenRequestForm](api)),
	}, map[string]*huma.Response{"200": jsonResponse("Tokens", schemaFor[TokenResponse](api))})

	add(&huma.Operation{
		OperationID: "oauth-register",
		Method:      http.MethodPost,
		Path:        registrationEndpointPath,
		Summary:     "Dynamic client registration (RFC 7591), for clients without a client metadata document",
		RequestBody: &huma.RequestBody{
			Required: true,
			Content:  map[string]*huma.MediaType{jsonMediaType: {Schema: schemaFor[ClientRegistrationRequest](api)}},
		},
	}, map[string]*huma.Response{"201": jsonResponse("Registered client", schemaFor[ClientRegistrationResponse](api))})

	add(&huma.Operation{
		OperationID: "oauth-revoke",
		Method:      http.MethodPost,
		Path:        revocationEndpointPath,
		Summary:     "Revoke a token (RFC 7009); revoking a refresh token disconnects the app",
		RequestBody: formBody(schemaFor[RevokeRequestForm](api)),
	}, map[string]*huma.Response{"200": {Description: "Revoked, or the token was unknown"}})
}

func schemaFor[T any](api huma.API) *huma.Schema {
	return api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[T](), true, "")
}

func jsonResponse(description string, schema *huma.Schema) *huma.Response {
	return &huma.Response{
		Description: description,
		Content:     map[string]*huma.MediaType{jsonMediaType: {Schema: schema}},
	}
}

func formBody(schema *huma.Schema) *huma.RequestBody {
	return &huma.RequestBody{
		Required: true,
		Content:  map[string]*huma.MediaType{formMediaType: {Schema: schema}},
	}
}

func queryParam(name, description string, required bool) *huma.Param {
	return &huma.Param{
		Name:        name,
		In:          "query",
		Description: description,
		Required:    required,
		Schema:      &huma.Schema{Type: huma.TypeString},
	}
}
