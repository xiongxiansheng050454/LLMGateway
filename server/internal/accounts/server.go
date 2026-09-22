package accounts

// Server owns account rules and orchestration over an injected Port and
// TxManager. Persistence primitives stay behind the port.
type Server struct {
	store Port
	tx    TxManager
}

func New(st Port, tx TxManager) *Server { return &Server{store: st, tx: tx} }
