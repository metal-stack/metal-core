package metalcore

import "github.com/metal-stack/metal-core/pkg/domain"

type Server struct {
	*domain.AppContext
}

func (s *Server) Run() {
	s.Server().Run()
}
