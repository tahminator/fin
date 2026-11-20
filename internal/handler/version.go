package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type Version struct {
	Version *string `json:"version" doc:"Deployed image tag; null when VERSION is unset"`
}

type VersionOutput struct {
	Body Version
}

type VersionHandler struct {
	version *string
}

func NewVersionHandler(version string) *VersionHandler {
	if version == "" {
		return &VersionHandler{}
	}

	return &VersionHandler{version: &version}
}

func (h *VersionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-version",
		Method:      http.MethodGet,
		Path:        "/version",
		Summary:     "Deployed application version",
		Tags:        []string{"version"},
	}, h.get)
}

func (h *VersionHandler) get(_ context.Context, _ *struct{}) (*VersionOutput, error) {
	return &VersionOutput{Body: Version{Version: h.version}}, nil
}
