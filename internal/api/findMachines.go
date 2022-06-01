package api

import (
	"net/http"

	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) FindMachines(mac string) (int, []*models.V1MachineResponse) {
	findMachines := machine.NewFindMachinesParams()
	req := &models.V1MachineFindRequest{
		NicsMacAddresses: []string{mac},
	}
	findMachines.SetBody(req)

	ok, err := c.machineClient.FindMachines(findMachines, c.auth)
	if err != nil {
		c.log.Error("machine(s) not found",
			zap.String("MAC", mac),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
