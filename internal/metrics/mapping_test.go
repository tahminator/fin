package metrics

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/tahminator/fin/internal/domain"
)

func TestStatusClassOf(t *testing.T) {
	tests := []struct {
		status int
		want   StatusClass
	}{
		{status: 100, want: StatusClass1xx},
		{status: 199, want: StatusClass1xx},
		{status: 200, want: StatusClass2xx},
		{status: 299, want: StatusClass2xx},
		{status: 300, want: StatusClass3xx},
		{status: 399, want: StatusClass3xx},
		{status: 400, want: StatusClass4xx},
		{status: 499, want: StatusClass4xx},
		{status: 500, want: StatusClass5xx},
		{status: 599, want: StatusClass5xx},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			if got := statusClassOf(tt.status); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMethodOf(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   Method
	}{
		{name: "GET", method: http.MethodGet, want: MethodGet},
		{name: "POST", method: http.MethodPost, want: MethodPost},
		{name: "PUT", method: http.MethodPut, want: MethodPut},
		{name: "PATCH", method: http.MethodPatch, want: MethodPatch},
		{name: "DELETE", method: http.MethodDelete, want: MethodDelete},
		{name: "HEAD is other", method: http.MethodHead, want: MethodOther},
		{name: "OPTIONS is other", method: http.MethodOptions, want: MethodOther},
		{name: "lowercase get is other", method: "get", want: MethodOther},
		{name: "empty is other", method: "", want: MethodOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MethodOf(tt.method); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestConnectionStatusOf(t *testing.T) {
	tests := []struct {
		name   string
		status domain.ConnectionStatus
		want   ConnectionStatus
	}{
		{name: "active", status: domain.ConnectionStatusActive, want: ConnectionStatusActive},
		{name: "needs reauth", status: domain.ConnectionStatusNeedsReauth, want: ConnectionStatusNeedsReauth},
		{name: "unknown status is other", status: domain.ConnectionStatus("weird"), want: ConnectionStatusOther},
		{name: "empty status is other", status: domain.ConnectionStatus(""), want: ConnectionStatusOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectionStatusOf(tt.status); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}
