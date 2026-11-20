package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
	"github.com/tahminator/fin/internal/service"
)

type ConnectionHandler struct {
	connections repository.ConnectionRepository
	service     *service.ConnectionService
}

func NewConnectionHandler(connections repository.ConnectionRepository, connectionService *service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{connections: connections, service: connectionService}
}

func (h *ConnectionHandler) Register(api huma.API) {
	tags := []string{"connections"}

	huma.Register(api, huma.Operation{
		OperationID: "list-connections",
		Method:      http.MethodGet,
		Path:        "/connections",
		Summary:     "List bank connections",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "create-link-token",
		Method:      http.MethodPost,
		Path:        "/connections/link-token",
		Summary:     "Create a Plaid Link token for a new connection",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.linkToken)

	huma.Register(api, huma.Operation{
		OperationID:   "create-connection",
		Method:        http.MethodPost,
		Path:          "/connections",
		Summary:       "Link a bank using a Plaid public token and run the initial sync",
		Tags:          tags,
		Security:      cookieAuth,
		DefaultStatus: http.StatusCreated,
	}, h.create)

	huma.Register(api, huma.Operation{
		OperationID: "sync-all-connections",
		Method:      http.MethodPost,
		Path:        "/connections/sync",
		Summary:     "Sync every active connection",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.syncAll)

	huma.Register(api, huma.Operation{
		OperationID: "sync-connection",
		Method:      http.MethodPost,
		Path:        "/connections/{connectionId}/sync",
		Summary:     "Sync one connection",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.sync)

	huma.Register(api, huma.Operation{
		OperationID: "create-reauth-link-token",
		Method:      http.MethodPost,
		Path:        "/connections/{connectionId}/reauth/link-token",
		Summary:     "Create an update-mode Plaid Link token for a connection that needs re-authentication",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.reauthLinkToken)

	huma.Register(api, huma.Operation{
		OperationID: "complete-reauth",
		Method:      http.MethodPost,
		Path:        "/connections/{connectionId}/reauth/complete",
		Summary:     "Mark a re-authenticated connection active and sync it",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.completeReauth)

	huma.Register(api, huma.Operation{
		OperationID: "create-account-selection-link-token",
		Method:      http.MethodPost,
		Path:        "/connections/{connectionId}/account-selection/link-token",
		Summary:     "Create an update-mode Plaid Link token for adding or removing a connection's accounts",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.accountSelectionLinkToken)

	huma.Register(api, huma.Operation{
		OperationID: "complete-account-selection",
		Method:      http.MethodPost,
		Path:        "/connections/{connectionId}/account-selection/complete",
		Summary:     "Sync a connection after its account selection changed and delete de-selected accounts",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.completeAccountSelection)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-connection",
		Method:        http.MethodDelete,
		Path:          "/connections/{connectionId}",
		Summary:       "Remove a connection and all of its data",
		Tags:          tags,
		Security:      cookieAuth,
		DefaultStatus: http.StatusNoContent,
	}, h.remove)
}

func (h *ConnectionHandler) list(ctx context.Context, input *ListConnectionsInput) (*PageOutput[Connection], error) {
	connections, total, err := h.connections.ListByUser(ctx, UserFrom(ctx).ID, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[Connection]{Body: NewPage(mapItems(connections, newConnection), input.PageParams, total)}, nil
}

func (h *ConnectionHandler) linkToken(ctx context.Context, _ *struct{}) (*LinkTokenOutput, error) {
	token, err := h.service.CreateLinkToken(ctx, UserFrom(ctx).ID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &LinkTokenOutput{Body: LinkToken{LinkToken: token}}, nil
}

func (h *ConnectionHandler) create(ctx context.Context, input *CreateConnectionInput) (*ConnectionOutput, error) {
	connection, err := h.service.Link(ctx, UserFrom(ctx).ID, input.Body.PublicToken)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConnectionOutput{Body: newConnection(connection)}, nil
}

func (h *ConnectionHandler) syncAll(ctx context.Context, _ *struct{}) (*SyncSummaryOutput, error) {
	summary, err := h.service.SyncAll(ctx, UserFrom(ctx).ID, metrics.SyncTriggerManual)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &SyncSummaryOutput{Body: newSyncSummary(summary)}, nil
}

func (h *ConnectionHandler) sync(ctx context.Context, input *ConnectionPathInput) (*ConnectionOutput, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	connection, err := h.service.SyncOne(ctx, UserFrom(ctx).ID, id)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConnectionOutput{Body: newConnection(connection)}, nil
}

func (h *ConnectionHandler) reauthLinkToken(ctx context.Context, input *ConnectionPathInput) (*LinkTokenOutput, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	token, err := h.service.CreateReauthLinkToken(ctx, UserFrom(ctx).ID, id)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &LinkTokenOutput{Body: LinkToken{LinkToken: token}}, nil
}

func (h *ConnectionHandler) completeReauth(ctx context.Context, input *ConnectionPathInput) (*ConnectionOutput, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	connection, err := h.service.CompleteReauth(ctx, UserFrom(ctx).ID, id)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConnectionOutput{Body: newConnection(connection)}, nil
}

func (h *ConnectionHandler) accountSelectionLinkToken(ctx context.Context, input *ConnectionPathInput) (*LinkTokenOutput, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	token, err := h.service.CreateAccountSelectionLinkToken(ctx, UserFrom(ctx).ID, id)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &LinkTokenOutput{Body: LinkToken{LinkToken: token}}, nil
}

func (h *ConnectionHandler) completeAccountSelection(ctx context.Context, input *ConnectionPathInput) (*ConnectionOutput, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	connection, err := h.service.CompleteAccountSelection(ctx, UserFrom(ctx).ID, id)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConnectionOutput{Body: newConnection(connection)}, nil
}

func (h *ConnectionHandler) remove(ctx context.Context, input *ConnectionPathInput) (*struct{}, error) {
	id, err := parseID(input.ConnectionID)
	if err != nil {
		return nil, err
	}

	if err := h.service.Remove(ctx, UserFrom(ctx).ID, id); err != nil {
		return nil, problem(ctx, err)
	}

	return nil, nil
}
