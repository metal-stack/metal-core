package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) FindMachines(mac string) (int, []*models.V1MachineResponse) {
	findMachines := machine.NewFindMachinesParams()
	req := &models.V1MachineFindRequest{
		NicsMacAddresses: []string{mac},
	}
	findMachines.SetBody(req)

	ok, err := c.MachineClient.FindMachines(findMachines, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Machine(s) not found",
			zap.String("MAC", mac),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
