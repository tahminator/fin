package domain

import (
	"time"
	"uuid"
)

type User struct {
	ID        uuid.UUID `db:"id"`
	Email     string    `db:"email"`
	GoogleSub *string   `db:"google_sub"`
	Name      *string   `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}
