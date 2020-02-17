package event

import "github.com/metal-stack/metal-core/pkg/domain"

type eventHandler struct {
	*domain.AppContext
}

func NewHandler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{ctx}
}
