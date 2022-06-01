package endpoint

import (
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

type endpointHandler struct {
	apiClient       domain.APIClient
	bootConfig      *bootConfig
	changeBootOrder bool
	grpcConfig      *grpcConfig
	log             *zap.Logger
	partitionID     string
}

func NewHandler(ctx *domain.AppContext) domain.EndpointHandler {
	grpcConfig := &grpcConfig{
		address:        ctx.Config.GrpcAddress,
		caCertFile:     ctx.Config.GrpcCACertFile,
		clientCertFile: ctx.Config.GrpcClientCertFile,
		clientKeyFile:  ctx.Config.GrpcClientKeyFile,
	}
	return &endpointHandler{
		apiClient:       ctx.APIClient(),
		bootConfig:      newBootConfig(ctx.BootConfig, ctx.Config),
		changeBootOrder: ctx.Config.ChangeBootOrder,
		grpcConfig:      grpcConfig,
		log:             ctx.Log,
	}
}
