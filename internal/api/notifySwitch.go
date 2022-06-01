package api

import (
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) NotifySwitch(switchID string, request *models.V1SwitchNotifyRequest) (*models.V1SwitchResponse, error) {
	params := sw.NewNotifySwitchParams()
	params.ID = switchID
	params.Body = request
	ok, err := c.switchClient.NotifySwitch(params, c.auth)
	if err != nil {
		c.log.Error("failed to notify switch", zap.Error(err))
		return nil, err
	}
	return ok.Payload, nil
}
