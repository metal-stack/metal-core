package api

import (
	"net/http"

	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"

	"github.com/metal-stack/metal-core/pkg/domain"

	"go.uber.org/zap"
)

func (c *apiClient) AbortReinstall(machineID string, request *domain.MetalHammerAbortReinstallRequest) (int, *models.V1BootInfo) {
	params := machine.NewAbortReinstallMachineParams()
	params.Body = &models.V1MachineAbortReinstallRequest{
		PrimaryDiskWiped: &request.PrimaryDiskWiped,
	}

	ok, err := c.MachineClient.AbortReinstallMachine(params, c.Auth)
	if err != nil {
		c.Log.Error("Failed to abort reinstall",
			zap.String("machineID", machineID),
			zap.Bool("primary disk already wiped", request.PrimaryDiskWiped),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	if ok != nil {
		return http.StatusOK, ok.Payload
	}
	return http.StatusOK, nil
}
