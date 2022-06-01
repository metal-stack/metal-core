package event

import (
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

type eventHandler struct {
	apiClient domain.APIClient
	config    *config
	log       *zap.Logger
}

func NewHandler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{
		apiClient: ctx.APIClient(),
		config:    newConfig(ctx.Config, ctx.DevMode),
		log:       ctx.Log,
	}
}
