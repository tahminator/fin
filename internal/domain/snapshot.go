package domain

import "time"

type MetricsSnapshot struct {
	Users                int
	UsersWithCredentials map[Provider]int
	PerUser              []UserSnapshot
}

type UserSnapshot struct {
	Email                 string                   `db:"email"`
	ActiveSessions        int                      `db:"active_sessions"`
	ConnectionsByStatus   map[ConnectionStatus]int `db:"-"`
	ConnectionsWithErrors int                      `db:"connections_with_errors"`
	Accounts              int                      `db:"accounts"`
	Transactions          int                      `db:"transactions"`
	OldestSyncAt          *time.Time               `db:"oldest_sync_at"`
}
