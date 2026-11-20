package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tahminator/fin/internal/server"
)

func printOpenAPI(out io.Writer) error {
	mux := http.NewServeMux()
	api := server.NewAPI(mux)

	server.RegisterRoutes(mux, api, server.Handlers{})

	spec, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal openapi: %w", err)
	}

	_, err = fmt.Fprintln(out, string(spec))

	return err
}
