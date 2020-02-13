package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
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
