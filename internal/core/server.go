package core

import (
	"fmt"
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

type server struct {
	endpointHandler  domain.EndpointHandler
	log              *zap.Logger
	metricServerAddr string
	serverAddr       string
}

func NewServer(ctx *domain.AppContext) domain.Server {
	serverAddr := fmt.Sprintf("%v:%d", ctx.Config.BindAddress, ctx.Config.Port)
	metricServerAddr := fmt.Sprintf("%v:%d", ctx.Config.MetricsServerBindAddress, ctx.Config.MetricsServerPort)

	return &server{
		endpointHandler:  ctx.EndpointHandler(),
		log:              ctx.Log,
		metricServerAddr: metricServerAddr,
		serverAddr:       serverAddr,
	}
}
