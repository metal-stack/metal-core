package api

import (
	"net/http"

	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) RegisterMachine(machineID string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.V1MachineResponse) {
	partitionID := c.partitionID
	rackID := c.rackID
	params := machine.NewRegisterMachineParams()
	params.Body = &models.V1MachineRegisterRequest{
		UUID:        &machineID,
		Partitionid: &partitionID,
		Rackid:      &rackID,
		Hardware: &models.V1MachineHardwareExtended{
			Memory:   request.Memory,
			CPUCores: request.CPUCores,
			Nics:     request.Nics,
			Disks:    request.Disks,
		},
		Ipmi: request.IPMI,
		Bios: request.BIOS,
	}

	ok, created, err := c.machineClient.RegisterMachine(params, c.auth)
	if err != nil {
		c.log.Error("failed to register machine at metal-api",
			zap.String("machineID", machineID),
			zap.String("partitionID", partitionID),
			zap.String("rackID", rackID),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	if ok != nil {
		return http.StatusOK, ok.Payload
	}
	return http.StatusOK, created.Payload
}
