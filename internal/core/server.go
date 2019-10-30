package core

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
)

type coreServer struct {
	*domain.AppContext
}

func NewServer(ctx *domain.AppContext) domain.Server {
	return &coreServer{ctx}
}
