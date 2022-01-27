package api

import (
	"github.com/metal-stack/metal-go/api/client/partition"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) FindPartition(id string) (*models.V1PartitionResponse, error) {
	params := partition.NewFindPartitionParams()
	params.SetID(id)

	ok, err := c.PartitionClient.FindPartition(params, c.Auth)
	if err != nil {
		c.Log.Error("partition not found",
			zap.String("id", id),
			zap.Error(err),
		)
		return nil, err
	}
	return ok.Payload, nil
}
