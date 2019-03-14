package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"net/http"
)

func (c *apiClient) FindMachines(mac string) (int, []*models.MetalMachine) {
	params := machine.NewSearchMachineParams()
	params.Mac = &mac

	ok, err := c.MachineClient.SearchMachine(params)
	if err != nil {
		zapup.MustRootLogger().Error("Machine(s) not found",
			zap.String("MAC", mac),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
