package api

import (
	"github.com/go-openapi/runtime"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/client/partition"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"go.uber.org/zap"
)

type apiClient struct {
	additionalBridgePorts []string
	auth                  runtime.ClientAuthInfoWriter
	machineClient         machine.ClientService
	log                   *zap.Logger
	partitionClient       partition.ClientService
	partitionID           string
	rackID                string
	switchClient          sw.ClientService
}

func NewClient(ctx *domain.AppContext) domain.APIClient {
	return &apiClient{
		additionalBridgePorts: ctx.Config.AdditionalBridgePorts,
		auth:                  ctx.Auth,
		machineClient:         ctx.MachineClient,
		log:                   ctx.Log,
		partitionClient:       ctx.PartitionClient,
		partitionID:           ctx.Config.PartitionID,
		rackID:                ctx.Config.RackID,
		switchClient:          ctx.SwitchClient,
	}
}
