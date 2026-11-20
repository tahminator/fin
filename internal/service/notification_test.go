package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

func TestNotificationListRequestsOnlyNeedsReauthWithPaging(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "first page", limit: 20, offset: 0},
		{name: "later page", limit: 5, offset: 15},
		{name: "maximum page size", limit: 100, offset: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()

			var (
				gotUserID uuid.UUID
				gotStatus domain.ConnectionStatus
				gotLimit  int
				gotOffset int
				calls     int
			)

			connections := &fakeConnections{
				listByUserAndStatus: func(_ context.Context, listedUserID uuid.UUID, status domain.ConnectionStatus, limit, offset int) ([]domain.Connection, int, error) {
					calls++
					gotUserID, gotStatus, gotLimit, gotOffset = listedUserID, status, limit, offset

					return nil, 0, nil
				},
			}

			if _, _, err := NewNotificationService(connections).List(t.Context(), userID, tt.limit, tt.offset); err != nil {
				t.Fatalf("List: %v", err)
			}

			if calls != 1 {
				t.Fatalf("ListByUserAndStatus called %d times, want 1", calls)
			}

			if gotUserID != userID || gotStatus != domain.ConnectionStatusNeedsReauth {
				t.Errorf("listed (%v, %q), want (%v, %q)", gotUserID, gotStatus, userID, domain.ConnectionStatusNeedsReauth)
			}

			if gotLimit != tt.limit || gotOffset != tt.offset {
				t.Errorf("paging = (%d, %d), want (%d, %d)", gotLimit, gotOffset, tt.limit, tt.offset)
			}
		})
	}
}

func TestNotificationListMapsConnectionsInRepositoryOrder(t *testing.T) {
	newest := domain.Connection{
		ID:              uuid.New(),
		InstitutionName: new("First Bank"),
		Status:          domain.ConnectionStatusNeedsReauth,
		StatusChangedAt: fixedNow.Add(-time.Hour),
	}

	unnamed := domain.Connection{
		ID:              uuid.New(),
		Status:          domain.ConnectionStatusNeedsReauth,
		StatusChangedAt: fixedNow.Add(-48 * time.Hour),
	}

	oldest := domain.Connection{
		ID:              uuid.New(),
		InstitutionName: new("Credit Union"),
		Status:          domain.ConnectionStatusNeedsReauth,
		StatusChangedAt: fixedNow.Add(-72 * time.Hour),
	}

	connections := &fakeConnections{
		listByUserAndStatus: func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
			return []domain.Connection{newest, unnamed, oldest}, 7, nil
		},
	}

	notifications, total, err := NewNotificationService(connections).List(t.Context(), uuid.New(), 3, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if total != 7 {
		t.Errorf("total = %d, want 7", total)
	}

	want := []domain.Notification{
		{Kind: domain.NotificationKindReauthRequired, ConnectionID: newest.ID, InstitutionName: "First Bank", Since: newest.StatusChangedAt},
		{Kind: domain.NotificationKindReauthRequired, ConnectionID: unnamed.ID, InstitutionName: "Unknown institution", Since: unnamed.StatusChangedAt},
		{Kind: domain.NotificationKindReauthRequired, ConnectionID: oldest.ID, InstitutionName: "Credit Union", Since: oldest.StatusChangedAt},
	}

	if !slices.Equal(notifications, want) {
		t.Errorf("notifications = %+v, want %+v", notifications, want)
	}
}

func TestNotificationListWithoutConnectionsReturnsEmptySlice(t *testing.T) {
	connections := &fakeConnections{
		listByUserAndStatus: func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
			return nil, 0, nil
		},
	}

	notifications, total, err := NewNotificationService(connections).List(t.Context(), uuid.New(), 20, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if notifications == nil || len(notifications) != 0 || total != 0 {
		t.Errorf("List = (%v, %d), want non-nil empty slice and 0", notifications, total)
	}
}

func TestNotificationListPropagatesRepositoryError(t *testing.T) {
	errDatabase := errors.New("database unavailable")

	connections := &fakeConnections{
		listByUserAndStatus: func(context.Context, uuid.UUID, domain.ConnectionStatus, int, int) ([]domain.Connection, int, error) {
			return nil, 0, errDatabase
		},
	}

	notifications, _, err := NewNotificationService(connections).List(t.Context(), uuid.New(), 20, 0)
	if !errors.Is(err, errDatabase) {
		t.Fatalf("List error = %v, want %v", err, errDatabase)
	}

	if notifications != nil {
		t.Errorf("notifications = %v, want nil", notifications)
	}
}
