package api

import (
	"net/http"

	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) RegisterMachine(machineID string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.V1MachineResponse) {
	partitionId := c.Config.PartitionID
	rackId := c.Config.RackID
	params := machine.NewRegisterMachineParams()
	params.Body = &models.V1MachineRegisterRequest{
		UUID:        &machineID,
		Partitionid: &partitionId,
		Rackid:      &rackId,
		Hardware: &models.V1MachineHardwareExtended{
			Memory:   request.Memory,
			CPUCores: request.CPUCores,
			Nics:     request.Nics,
			Disks:    request.Disks,
		},
		IPMI: request.IPMI,
		Bios: request.BIOS,
	}

	ok, created, err := c.MachineClient.RegisterMachine(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to register machine at Metal-API",
			zap.String("machineID", machineID),
			zap.String("partitionID", partitionId),
			zap.String("rackID", rackId),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	if ok != nil {
		return http.StatusOK, ok.Payload
	}
	return http.StatusOK, created.Payload
}
