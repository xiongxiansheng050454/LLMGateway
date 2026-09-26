package main

import (
	"net/http"

	"LLMGateway/server/internal/httpapi"
)

// newRouter is the single place where HTTP paths are mapped to handlers.
func newRouter(st httpapi.Port, opts ...httpapi.Option) http.Handler {
	_, handler := newRouterServer(st, opts...)
	return handler
}

// newRouterServer also returns the assembled server so process workers can use
// its module delegates (for example the channel health bucket reaper).
func newRouterServer(st httpapi.Port, opts ...httpapi.Option) (*httpapi.Server, http.Handler) {
	server := httpapi.NewServer(st, opts...)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.Healthz)
	mux.HandleFunc("/admin/", server.Admin)
	mux.HandleFunc("/admin", server.Admin)
	mux.HandleFunc("/v1/", server.OpenAI)
	mux.HandleFunc("/v1", server.OpenAI)
	return server, mux
}
