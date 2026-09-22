package ratelimit

import "time"

// Server owns rate-limit rules and reservation orchestration over injected
// primitives.
type Server struct {
	store Port
	now   func() time.Time
}

func New(st Port, now func() time.Time) *Server {
	if now == nil {
		now = time.Now
	}
	return &Server{store: st, now: now}
}
