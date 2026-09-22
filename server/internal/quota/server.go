package quota

import "time"

// Server owns quota rules and reservation orchestration over injected
// primitives.
type Server struct {
	store Port
	tx    TxManager
	now   func() time.Time
}

func New(st Port, tx TxManager, now func() time.Time) *Server {
	if now == nil {
		now = time.Now
	}
	return &Server{store: st, tx: tx, now: now}
}
