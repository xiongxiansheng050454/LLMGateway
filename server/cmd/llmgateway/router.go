package main

import (
	"net/http"

	"LLMGateway/server/internal/httpapi"
)

// newRouter is the single place where HTTP paths are mapped to handlers.
func newRouter(st httpapi.Port, opts ...httpapi.Option) http.Handler {
	server := httpapi.NewServer(st, opts...)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.Healthy)
	mux.HandleFunc("/admin/", server.Admin)
	mux.HandleFunc("/admin", server.Admin)
	mux.HandleFunc("/v1/", server.OpenAI)
	mux.HandleFunc("/v1", server.OpenAI)
	return mux
}
