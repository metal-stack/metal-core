package endpoint

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
)

type endpoint struct {
	*domain.AppContext
}

func Handler(ctx *domain.AppContext) domain.Endpoint {
	return endpoint{ctx}
}
