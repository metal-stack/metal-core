package event

import (
	"github.com/go-openapi/runtime"
	"github.com/metal-stack/metal-core/pkg/domain"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"go.uber.org/zap"
)

type eventHandler struct {
	apiClient    domain.APIClient
	auth         runtime.ClientAuthInfoWriter
	config       *config
	log          *zap.Logger
	switchClient sw.ClientService
}

func NewHandler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{
		apiClient:    ctx.APIClient(),
		auth:         ctx.Auth,
		config:       newConfig(ctx.Config, ctx.DevMode),
		log:          ctx.Log,
		switchClient: ctx.SwitchClient,
	}
}
