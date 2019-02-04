package event

import "git.f-i-ts.de/cloud-native/metal/metal-core/domain"

type eventHandler struct {
	*domain.AppContext
}

func Handler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{ctx}
}
