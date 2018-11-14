package server

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
)

type server struct {
	*domain.AppContext
}

func Handler(ctx *domain.AppContext) domain.Server {
	return server{ctx}
}
