package core

import (
	"github.com/metal-stack/metal-core/pkg/domain"
)

type coreServer struct {
	*domain.AppContext
}

func NewServer(ctx *domain.AppContext) domain.Server {
	return &coreServer{ctx}
}
