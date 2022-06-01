package api

import (
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) FindSwitch(id string) (*models.V1SwitchResponse, error) {
	params := sw.NewFindSwitchParams()
	params.ID = id
	ok, err := c.switchClient.FindSwitch(params, c.auth)
	if err != nil {
		c.log.Error("switch not found",
			zap.String("ID", id),
			zap.Error(err),
		)
		return nil, err
	}
	return ok.Payload, nil
}
