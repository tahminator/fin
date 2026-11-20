package metrics

import (
	"net/http"

	"github.com/tahminator/fin/internal/domain"
)

func MethodOf(method string) Method {
	switch method {
	case http.MethodGet:
		return MethodGet
	case http.MethodPost:
		return MethodPost
	case http.MethodPut:
		return MethodPut
	case http.MethodPatch:
		return MethodPatch
	case http.MethodDelete:
		return MethodDelete
	}

	return MethodOther
}

func statusClassOf(status int) StatusClass {
	switch {
	case status < 200:
		return StatusClass1xx
	case status < 300:
		return StatusClass2xx
	case status < 400:
		return StatusClass3xx
	case status < 500:
		return StatusClass4xx
	}

	return StatusClass5xx
}

func connectionStatusOf(status domain.ConnectionStatus) ConnectionStatus {
	switch status {
	case domain.ConnectionStatusActive:
		return ConnectionStatusActive
	case domain.ConnectionStatusNeedsReauth:
		return ConnectionStatusNeedsReauth
	}

	return ConnectionStatusOther
}
