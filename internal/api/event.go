package api

import (
	"context"
	"fmt"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-core/pkg/domain"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.uber.org/zap"
)

func (c *apiClient) Send(event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	s, err := c.EventServiceClient.Send(context.Background(), event)
	if err != nil {
		return nil, err
	}
	if s != nil {
		c.Log.Sugar().Infow("event", "send", s.Events, "failed", s.Failed)
	}
	return s, err
}

func (c *apiClient) PhoneHome(msgs []phoneHomeMessage) {
	c.Log.Debug("phonehome",
		zap.String("machines", fmt.Sprintf("%v", msgs)),
	)
	c.Log.Info("phonehome",
		zap.Int("machines", len(msgs)),
	)
	events := make(map[string]*v1.MachineProvisioningEvent)
	phonedHomeEvent := string(domain.ProvisioningEventPhonedHome)
	for i := range msgs {
		msg := msgs[i]
		event := &v1.MachineProvisioningEvent{
			Event:   phonedHomeEvent,
			Message: msg.payload,
			Time:    timestamppb.New(msg.time),
		}
		events[msg.machineID] = event
	}

	s, err := c.Send(&v1.EventServiceSendRequest{Events: events})
	if err != nil {
		c.Log.Error("unable to send provisioning event back to API",
			zap.Error(err),
		)
	}
	if s != nil {
		c.Log.Info("phonehome sent",
			zap.Uint64("machines", s.Events),
		)
	}
}
