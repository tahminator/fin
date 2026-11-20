package domain

import (
	"time"
	"uuid"
)

type NotificationKind string

const NotificationKindReauthRequired NotificationKind = "reauth_required"

type Notification struct {
	Kind            NotificationKind
	ConnectionID    uuid.UUID
	InstitutionName string
	Since           time.Time
}
