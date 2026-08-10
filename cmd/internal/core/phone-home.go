//go:build client

package core

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
)

type phoneHomeMessage struct {
	machineID string
	payload   string
	time      time.Time
}

const (
	provisioningEventPhonedHome = "Phoned Home"
)

func (c *Core) ConstantlyPhoneHome(ctx context.Context, interval time.Duration) {
	// FIXME this list of interfaces is only read on startup
	// if additional interfaces are configured, no new lldpd client is started and therefore no
	// phoned home events are sent for these interfaces.
	// Solution:
	// - either ensure metal-core is restarted on interfaces added/removed
	// - dynamically detect changes and stop/start goroutines for the lldpd client per interface
	ifs, err := c.nos.GetSwitchPorts(ctx)
	if err != nil {
		c.log.Error("unable to find interfaces", "error", err)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult)
	discoveryResultChanWG := sync.WaitGroup{}

	var phoneHomeMessages sync.Map
	for _, iface := range ifs {
		lldpcli := lldp.NewClient(ctx, *iface)
		c.log.Info("start lldp client", "interface", iface.Name)

		ifaceName := iface.Name
		discoveryResultChanWG.Go(func() {
			err = lldpcli.Start(c.log, discoveryResultChan)
			if err != nil {
				c.log.Error("unable to start lldp discovery for interface", "interface", ifaceName, "error", err)
			}
		})
	}

	go func() {
		discoveryResultChanWG.Wait()
		close(discoveryResultChan)
	}()

	go func() {
		for phoneHome := range discoveryResultChan {
			msg := toPhoneHomeMessage(phoneHome)
			if msg == nil {
				continue
			}

			phoneHomeMessages.Store(msg.machineID, *msg)
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msgs := []phoneHomeMessage{}
			phoneHomeMessages.Range(func(key, value any) bool {
				msg, ok := value.(phoneHomeMessage)
				if !ok {
					return true
				}
				phoneHomeMessages.Delete(key)
				msgs = append(msgs, msg)
				return true
			})
			c.phoneHome(ctx, msgs)
		case <-ctx.Done():
			discoveryResultChanWG.Wait()
			return
		}
	}
}

func (c *Core) send(ctx context.Context, event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	s, err := c.eventServiceClient.Send(ctx, event)
	if err != nil {
		return nil, err
	}
	if s != nil {
		c.log.Info("event", "send", s.GetEvents(), "failed", s.GetFailed())
	}
	return s, err
}

func (c *Core) phoneHome(ctx context.Context, msgs []phoneHomeMessage) {
	c.log.Info("phonehome", "machines", len(msgs))

	events := make(map[string]*v1.MachineProvisioningEvent)
	phonedHomeEvent := string(provisioningEventPhonedHome)
	for i := range msgs {
		msg := msgs[i]
		event := &v1.MachineProvisioningEvent{
			Event:   phonedHomeEvent,
			Message: msg.payload,
			Time:    timestamppb.New(msg.time),
		}
		events[msg.machineID] = event
	}

	s, err := c.send(ctx, &v1.EventServiceSendRequest{Events: events})
	if err != nil {
		c.log.Error("unable to send provisioning event back to API", "error", err)
		c.metrics.CountError("send-provisioning")
	}
	if s != nil {
		c.log.Info("phonehome sent", "machines", s.GetEvents())
	}
}

func toPhoneHomeMessage(discoveryResult lldp.DiscoveryResult) *phoneHomeMessage {
	if strings.Contains(discoveryResult.SysDescription, "provisioned") {
		return &phoneHomeMessage{
			machineID: discoveryResult.SysName,
			payload:   discoveryResult.SysDescription,
			time:      time.Now(),
		}
	}
	return nil
}
