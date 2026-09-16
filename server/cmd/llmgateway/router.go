package main

import (
	"net/http"

	"LLMGateway/server/internal/handler"
	"LLMGateway/server/internal/store"
)

// newRouter is the single place where HTTP paths are mapped to handlers.
func newRouter(dashboardDir string, st store.Store, opts ...handler.Option) http.Handler {
	server := handler.NewServer(dashboardDir, st, opts...)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.Healthz)
	mux.HandleFunc("/admin/", server.Admin)
	mux.HandleFunc("/admin", server.Admin)
	mux.HandleFunc("/v1/", server.OpenAI)
	mux.HandleFunc("/v1", server.OpenAI)
	mux.HandleFunc("/dashboard/index.html", server.DashboardIndex)
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", http.HandlerFunc(server.Dashboard)))
	mux.Handle("/", http.HandlerFunc(server.Dashboard))
	return mux
}
