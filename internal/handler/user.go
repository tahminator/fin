package handler

import (
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email" format:"email"`
	Name  *string   `json:"name"`
}

func newUser(user domain.User) User {
	return User{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}
}
