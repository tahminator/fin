package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrNotAllowed     = errors.New("not allowed")
	ErrConflict       = errors.New("conflict")
	ErrReauthRequired = errors.New("re-authentication required")
	ErrUnavailable    = errors.New("unavailable")
	ErrInvalidToken   = errors.New("invalid access token")
	ErrUnsupported    = errors.New("not supported")
)
