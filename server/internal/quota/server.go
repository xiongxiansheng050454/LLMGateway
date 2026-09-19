package quota

type Server struct{ store Port }

func New(st Port) *Server { return &Server{store: st} }
