package catalog

import (
	"net/http"

	"LLMGateway/server/internal/store"
)

type Server struct {
	store  store.Store
	client *http.Client
}

func New(st store.Store, client *http.Client) *Server { return &Server{store: st, client: client} }
