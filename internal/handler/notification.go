package handler

import (
	"context"
	"net/http"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/service"
)

type Notification struct {
	Kind            domain.NotificationKind `json:"kind" enum:"reauth_required"`
	ConnectionID    uuid.UUID               `json:"connectionId"`
	InstitutionName string                  `json:"institutionName"`
	Since           time.Time               `json:"since"`
}

func newNotification(notification domain.Notification) Notification {
	return Notification{
		Kind:            notification.Kind,
		ConnectionID:    notification.ConnectionID,
		InstitutionName: notification.InstitutionName,
		Since:           notification.Since,
	}
}

type ListNotificationsInput struct {
	PageParams
}

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func (h *NotificationHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-notifications",
		Method:      http.MethodGet,
		Path:        "/notifications",
		Summary:     "Connections that need attention, newest first",
		Tags:        []string{"notifications"},
		Security:    cookieAuth,
	}, h.list)
}

func (h *NotificationHandler) list(ctx context.Context, input *ListNotificationsInput) (*PageOutput[Notification], error) {
	notifications, total, err := h.notifications.List(ctx, UserFrom(ctx).ID, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[Notification]{Body: NewPage(mapItems(notifications, newNotification), input.PageParams, total)}, nil
}
