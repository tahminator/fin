package service

import (
	"context"
	"fmt"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const unknownInstitution = "Unknown institution"

type NotificationService struct {
	connections repository.ConnectionRepository
}

func NewNotificationService(connections repository.ConnectionRepository) *NotificationService {
	return &NotificationService{connections: connections}
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Notification, int, error) {
	connections, total, err := s.connections.ListByUserAndStatus(ctx, userID, domain.ConnectionStatusNeedsReauth, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list connections needing reauth: %w", err)
	}

	notifications := make([]domain.Notification, 0, len(connections))

	for _, connection := range connections {
		notifications = append(notifications, reauthNotification(connection))
	}

	return notifications, total, nil
}

func reauthNotification(connection domain.Connection) domain.Notification {
	institutionName := unknownInstitution

	if connection.InstitutionName != nil {
		institutionName = *connection.InstitutionName
	}

	return domain.Notification{
		Kind:            domain.NotificationKindReauthRequired,
		ConnectionID:    connection.ID,
		InstitutionName: institutionName,
		Since:           connection.StatusChangedAt,
	}
}
