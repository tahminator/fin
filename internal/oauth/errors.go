package oauth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ErrorCode string

const (
	ErrInvalidRequest          ErrorCode = "invalid_request"
	ErrInvalidClient           ErrorCode = "invalid_client"
	ErrInvalidGrant            ErrorCode = "invalid_grant"
	ErrUnauthorizedClient      ErrorCode = "unauthorized_client"
	ErrUnsupportedGrantType    ErrorCode = "unsupported_grant_type"
	ErrUnsupportedResponseType ErrorCode = "unsupported_response_type"
	ErrInvalidScope            ErrorCode = "invalid_scope"
	ErrInvalidTarget           ErrorCode = "invalid_target"
	ErrAccessDenied            ErrorCode = "access_denied"
	ErrInvalidRedirectURI      ErrorCode = "invalid_redirect_uri"
	ErrInvalidClientMetadata   ErrorCode = "invalid_client_metadata"
	ErrServerError             ErrorCode = "server_error"
	ErrSlowDown                ErrorCode = "slow_down"
)

type Error struct {
	Code        ErrorCode
	Description string
	Redirect    bool
}

func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Description
}

func NewError(code ErrorCode, description string) *Error {
	return &Error{Code: code, Description: description}
}

func NewRedirectableError(code ErrorCode, description string) *Error {
	return &Error{Code: code, Description: description, Redirect: true}
}

func AsError(err error) (*Error, bool) {
	var oauthErr *Error

	ok := errors.As(err, &oauthErr)

	return oauthErr, ok
}

var errorStatuses = map[ErrorCode]int{
	ErrInvalidClient: http.StatusUnauthorized,
	ErrServerError:   http.StatusInternalServerError,
	ErrSlowDown:      http.StatusTooManyRequests,
}

func (e *Error) Status() int {
	if status, ok := errorStatuses[e.Code]; ok {
		return status
	}

	return http.StatusBadRequest
}

type errorBody struct {
	Error       ErrorCode `json:"error"`
	Description string    `json:"error_description,omitempty"`
}

func WriteError(w http.ResponseWriter, err *Error) {
	WriteJSON(w, err.Status(), errorBody{Error: err.Code, Description: err.Description})
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(body)
}
