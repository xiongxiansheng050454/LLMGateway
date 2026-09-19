package catalog

import (
	"net/http"
	"time"
)

type Server struct {
	store interface {
		Port
		HealthPort
	}
	client      *http.Client
	testTimeout time.Duration
}

func New(st interface {
	Port
	HealthPort
}, client *http.Client) *Server {
	return &Server{store: st, client: client, testTimeout: channelTestTimeout}
}
