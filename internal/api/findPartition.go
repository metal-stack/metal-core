package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/partition"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) FindPartition(id string) (*models.V1PartitionResponse, error) {
	params := partition.NewFindPartitionParams()
	params.SetID(id)

	ok, err := c.PartitionClient.FindPartition(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Partition not found",
			zap.String("id", id),
			zap.Error(err),
		)
		return nil, err
	}
	return ok.Payload, nil
}
