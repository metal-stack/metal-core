package api

import "git.f-i-ts.de/cloud-native/maas/metal-core/domain"

type client struct {
	*domain.AppContext
}

func Handler(ctx *domain.AppContext) domain.APIClient {
	return client{ctx}
}
