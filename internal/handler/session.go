package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
)

const SessionCookie = "fin_session"

var cookieAuth = []map[string][]string{{"cookieAuth": {}}}

type sessionKey struct{}

type session struct {
	user  domain.User
	token string
}

func WithSession(ctx huma.Context, user domain.User, token string) huma.Context {
	return huma.WithValue(ctx, sessionKey{}, session{user: user, token: token})
}

func UserFrom(ctx context.Context) domain.User {
	current, _ := ctx.Value(sessionKey{}).(session)

	return current.user
}

func sessionTokenFrom(ctx context.Context) string {
	current, _ := ctx.Value(sessionKey{}).(session)

	return current.token
}
