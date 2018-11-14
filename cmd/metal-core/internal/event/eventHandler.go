package event

import "git.f-i-ts.de/cloud-native/maas/metal-core/domain"

type listener struct {
	*domain.AppContext
}

func Handler(ctx *domain.AppContext) domain.EventReaction {
	return listener{ctx}
}
