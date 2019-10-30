package metalcore

import "git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"

type Server struct {
	*domain.AppContext
}

func (s *Server) Run() {
	s.Server().Run()
}
