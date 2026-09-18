package catalog

import (
	"net/http"
	"time"

	"LLMGateway/server/internal/store"
)

type Server struct {
	store       store.Store
	client      *http.Client
	testTimeout time.Duration
}

func New(st store.Store, client *http.Client) *Server {
	return &Server{store: st, client: client, testTimeout: channelTestTimeout}
}
