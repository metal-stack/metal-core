package event

import "git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"

type eventHandler struct {
	*domain.AppContext
}

func NewHandler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{ctx}
}
