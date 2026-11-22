package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
)

type ToolDefinition struct {
	Tool  *mcp.Tool
	Scope domain.Scope
}

var errInvalidParams = errors.New("invalid params")

func invalidParams(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalidParams, fmt.Sprintf(format, args...))
}

func define[In, Out any](name metrics.MCPTool, title, description string, scope domain.Scope, enums map[string][]any) ToolDefinition {
	input, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("input schema for %s: %v", name, err))
	}

	output, err := jsonschema.For[Out](nil)
	if err != nil {
		panic(fmt.Sprintf("output schema for %s: %v", name, err))
	}

	for field, values := range enums {
		setEnum(input, field, values)
	}

	readOnly := scope == domain.ScopeRead
	idempotent := true
	destructive := false
	openWorld := !readOnly

	return ToolDefinition{
		Tool: &mcp.Tool{
			Name:         name.String(),
			Title:        title,
			Description:  description,
			InputSchema:  input,
			OutputSchema: output,
			Annotations: &mcp.ToolAnnotations{
				Title:           title,
				ReadOnlyHint:    readOnly,
				DestructiveHint: &destructive,
				IdempotentHint:  idempotent,
				OpenWorldHint:   &openWorld,
			},
		},
		Scope: scope,
	}
}

func setEnum(schema *jsonschema.Schema, field string, values []any) {
	property, ok := schema.Properties[field]
	if !ok {
		panic(fmt.Sprintf("enum for unknown field %q", field))
	}

	if property.Items != nil {
		property.Items.Enum = values

		return
	}

	property.Enum = values
}

func strings2any(values ...string) []any {
	result := make([]any, len(values))

	for i, value := range values {
		result[i] = value
	}

	return result
}

func addTool[In, Out any](server *mcp.Server, t *toolset, name metrics.MCPTool, handler func(ctx context.Context, input In) (Out, error)) {
	definition := Definitions()[name]

	mcp.AddTool(server, definition.Tool, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		started := t.deps.Now()

		output, err := handler(ctx, input)

		t.deps.Recorder.MCPToolCalled(name, toolOutcomeOf(err), t.deps.Now().Sub(started))

		if err != nil {
			var zero Out

			return nil, zero, publicError(ctx, name, t.grant, err)
		}

		return nil, output, nil
	})
}

func toolOutcomeOf(err error) metrics.ToolOutcome {
	switch {
	case err == nil:
		return metrics.ToolOutcomeSuccess
	case errors.Is(err, errInvalidParams):
		return metrics.ToolOutcomeInvalidParams
	case errors.Is(err, domain.ErrNotFound):
		return metrics.ToolOutcomeNotFound
	}

	return metrics.ToolOutcomeError
}

func publicError(ctx context.Context, name metrics.MCPTool, grant domain.AccessGrant, err error) error {
	switch {
	case errors.Is(err, errInvalidParams):
		return errors.New(strings.TrimPrefix(err.Error(), errInvalidParams.Error()+": "))
	case errors.Is(err, domain.ErrNotFound):
		return errors.New("not found")
	case errors.Is(err, context.DeadlineExceeded):
		return errors.New("the query took too long; narrow the date range or filters")
	}

	slog.ErrorContext(ctx, "mcp tool failed", "tool", name.String(), "user_id", grant.User.ID, "grant_id", grant.GrantID, "err", err)

	return errors.New("internal error")
}
