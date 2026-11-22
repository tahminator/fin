package mcpserver

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
)

const (
	securityScheme = "mcpOAuth"
	mcpTag         = "mcp"
	toolsExtension = "x-mcp-tools"
)

type ToolDocument struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Scope        string `json:"scope"`
	ReadOnly     bool   `json:"readOnly"`
	InputSchema  any    `json:"inputSchema"`
	OutputSchema any    `json:"outputSchema"`
}

func ToolDocuments() []ToolDocument {
	definitions := Definitions()
	documents := make([]ToolDocument, 0, len(definitions))

	for _, name := range metrics.AllMCPTools() {
		definition := definitions[name]

		documents = append(documents, ToolDocument{
			Name:         definition.Tool.Name,
			Title:        definition.Tool.Title,
			Description:  definition.Tool.Description,
			Scope:        string(definition.Scope),
			ReadOnly:     definition.Tool.Annotations.ReadOnlyHint,
			InputSchema:  definition.Tool.InputSchema,
			OutputSchema: definition.Tool.OutputSchema,
		})
	}

	return documents
}

func Document(api huma.API) {
	jsonRPC := &huma.Schema{
		Type:        huma.TypeObject,
		Description: "A JSON-RPC 2.0 message as defined by the Model Context Protocol (initialize, tools/list, tools/call, ...).",
		Properties: map[string]*huma.Schema{
			"jsonrpc": {Type: huma.TypeString, Enum: []any{"2.0"}},
			"id":      {Description: "Request ID; omitted for notifications"},
			"method":  {Type: huma.TypeString},
			"params":  {Type: huma.TypeObject},
			"result":  {Type: huma.TypeObject},
			"error":   {Type: huma.TypeObject},
		},
		Required: []string{"jsonrpc"},
	}

	challenge := map[string]*huma.Param{
		"WWW-Authenticate": {
			Description: "Bearer challenge with resource_metadata (RFC 9728) and the scopes to request",
			Schema:      &huma.Schema{Type: huma.TypeString},
		},
	}

	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "mcp",
		Method:      http.MethodPost,
		Path:        "/mcp",
		Summary:     "Model Context Protocol endpoint (Streamable HTTP, stateless, JSON responses)",
		Description: "Connect any MCP client to this URL. Tools are listed in the x-mcp-tools extension; refresh_data needs the fin:sync scope and is hidden from tools/list without it. No tool can link, reconnect or remove a bank.",
		Tags:        []string{mcpTag},
		Security:    []map[string][]string{{securityScheme: {string(domain.ScopeRead)}}},
		RequestBody: &huma.RequestBody{
			Required: true,
			Content:  map[string]*huma.MediaType{"application/json": {Schema: jsonRPC}},
		},
		Responses: map[string]*huma.Response{
			"200": {Description: "JSON-RPC response", Content: map[string]*huma.MediaType{"application/json": {Schema: jsonRPC}}},
			"202": {Description: "Notification accepted"},
			"401": {Description: "Missing, invalid or expired access token", Headers: challenge},
			"403": {Description: "Insufficient scope or a disallowed Origin", Headers: challenge},
			"429": {Description: "Too many calls for this app; see Retry-After"},
		},
		Extensions: map[string]any{toolsExtension: ToolDocuments()},
	})

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		api.OpenAPI().AddOperation(&huma.Operation{
			OperationID: "mcp-" + map[string]string{http.MethodGet: "get", http.MethodDelete: "delete"}[method],
			Method:      method,
			Path:        "/mcp",
			Summary:     "Not supported: the server is stateless and sends no server-initiated messages",
			Tags:        []string{mcpTag},
			Responses:   map[string]*huma.Response{"405": {Description: "Method not allowed"}},
		})
	}
}
