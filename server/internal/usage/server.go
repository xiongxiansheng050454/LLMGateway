package usage

type Server struct{ store Port }

func New(st Port) *Server { return &Server{store: st} }
