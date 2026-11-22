package domain

import (
	"time"
	"uuid"
)

type ConnectionStatus string

const (
	ConnectionStatusActive      ConnectionStatus = "active"
	ConnectionStatusNeedsReauth ConnectionStatus = "needs_reauth"
)

func (s ConnectionStatus) Valid() bool {
	return s == ConnectionStatusActive || s == ConnectionStatusNeedsReauth
}

type Connection struct {
	ID                    uuid.UUID          `db:"id"`
	UserID                uuid.UUID          `db:"user_id"`
	Provider              Provider           `db:"provider"`
	ExternalID            string             `db:"external_id"`
	AccessTokenCiphertext string             `db:"access_token_ciphertext"`
	InstitutionID         *string            `db:"institution_id"`
	InstitutionName       *string            `db:"institution_name"`
	InstitutionURL        *string            `db:"institution_url"`
	Status                ConnectionStatus   `db:"status"`
	StatusChangedAt       time.Time          `db:"status_changed_at"`
	SyncCursor            string             `db:"sync_cursor"`
	LastSyncedAt          *time.Time         `db:"last_synced_at"`
	LastError             *string            `db:"last_error"`
	InvestmentsStatus     *InvestmentsStatus `db:"investments_status"`
	CreatedAt             time.Time          `db:"created_at"`
	UpdatedAt             time.Time          `db:"updated_at"`
}
