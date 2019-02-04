package core

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
)

type coreServer struct {
	*domain.AppContext
}

func Server(ctx *domain.AppContext) domain.Server {
	return &coreServer{ctx}
}
