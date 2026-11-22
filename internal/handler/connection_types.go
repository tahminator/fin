package handler

import (
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/service"
)

type Connection struct {
	ID                 uuid.UUID               `json:"id"`
	InstitutionName    *string                 `json:"institutionName"`
	InstitutionLogoURL *string                 `json:"institutionLogoUrl"`
	Status             domain.ConnectionStatus `json:"status" enum:"active,needs_reauth"`
	StatusChangedAt    time.Time               `json:"statusChangedAt"`
	LastSyncedAt       *time.Time              `json:"lastSyncedAt"`
	LastError          *string                 `json:"lastError"`
	CreatedAt          time.Time               `json:"createdAt"`
}

func newConnection(connection domain.Connection) Connection {
	return Connection{
		ID:                 connection.ID,
		InstitutionName:    connection.InstitutionName,
		InstitutionLogoURL: institutionLogoURL(connection.InstitutionURL),
		Status:             connection.Status,
		StatusChangedAt:    connection.StatusChangedAt,
		LastSyncedAt:       connection.LastSyncedAt,
		LastError:          connection.LastError,
		CreatedAt:          connection.CreatedAt,
	}
}

type LinkToken struct {
	LinkToken string `json:"linkToken" doc:"Plaid Link token"`
}

type CreateConnectionBody struct {
	PublicToken string `json:"publicToken" minLength:"1" doc:"Public token returned by Plaid Link"`
}

type SyncSummary struct {
	Synced         int `json:"synced"`
	ReauthRequired int `json:"reauthRequired"`
	Failed         int `json:"failed"`
	Throttled      int `json:"throttled" doc:"Connections skipped because they were synced within the sync cooldown"`
}

func newSyncSummary(summary service.SyncSummary) SyncSummary {
	return SyncSummary{
		Synced:         summary.Synced,
		ReauthRequired: summary.ReauthRequired,
		Failed:         summary.Failed,
		Throttled:      summary.Throttled,
	}
}

type ListConnectionsInput struct {
	PageParams
}

type ConnectionPathInput struct {
	ConnectionID string `path:"connectionId" format:"uuid"`
}

type CreateConnectionInput struct {
	Body CreateConnectionBody
}

type ConnectionOutput struct {
	Body Connection
}

type LinkTokenOutput struct {
	Body LinkToken
}

type SyncSummaryOutput struct {
	Body SyncSummary
}
