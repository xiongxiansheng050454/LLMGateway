package usage

import (
	"LLMGateway/server/internal/store"
)

type Server struct{ store store.Store }

func New(st store.Store) *Server { return &Server{store: st} }
